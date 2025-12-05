// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"errors"
	"io"
	"math"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ErrMissingChunk indicates that the number of chunks read from the server is
// less than expected. This error is specific to GridFS operations.
var ErrMissingChunk = errors.New("EOF missing one or more chunks")

// ErrWrongSize is used when the chunk retrieved from the server does not have
// the expected size. This error is specific to GridFS operations.
var ErrWrongSize = errors.New("chunk size does not match expected size")

var errNoMoreChunks = errors.New("no more chunks remaining")

// GridFSDownloadStream is a io.Reader that can be used to download a file from a GridFS bucket.
type GridFSDownloadStream struct {
	numChunks     int32
	chunkSize     int32
	cursor        *Cursor
	done          bool
	closed        bool
	buffer        []byte // store up to 1 chunk if the user provided buffer isn't big enough
	bufferStart   int
	bufferEnd     int
	expectedChunk int32 // index of next expected chunk
	fileLen       int64
	ctx           context.Context
	cancel        context.CancelFunc

	// The pointer returned by GetFile. This should not be used in the actual GridFSDownloadStream code outside of the
	// newGridFSDownloadStream constructor because the values can be mutated by the user after calling GetFile. Instead,
	// any values needed in the code should be stored separately and copied over in the constructor.
	file *GridFSFile
}

// GridFSFile represents a file stored in GridFS. This type can be used to
// access file information when downloading using the
// GridFSDownloadStream.GetFile method.
type GridFSFile struct {
	// ID is the file's ID. This will match the file ID specified when uploading the file. If an upload helper that
	// does not require a file ID was used, this field will be a bson.ObjectID.
	ID any

	// Length is the length of this file in bytes.
	Length int64

	// ChunkSize is the maximum number of bytes for each chunk in this file.
	ChunkSize int32

	// UploadDate is the time this file was added to GridFS in UTC. This field is set by the driver and is not configurable.
	// The Metadata field can be used to store a custom date.
	UploadDate time.Time

	// Name is the name of this file.
	Name string

	// Metadata is additional data that was specified when creating this file. This field can be unmarshalled into a
	// custom type using the bson.Unmarshal family of functions.
	Metadata bson.Raw
}

var _ bson.Unmarshaler = &GridFSFile{}

// findFileResponse is a temporary type used to unmarshal documents from the
// files collection and can be transformed into a File instance. This type
// exists to avoid adding BSON struct tags to the exported File type.
type findFileResponse struct {
	ID         any       `bson:"_id"`
	Length     int64     `bson:"length"`
	ChunkSize  int32     `bson:"chunkSize"`
	UploadDate time.Time `bson:"uploadDate"`
	Name       string    `bson:"filename"`
	Metadata   bson.Raw  `bson:"metadata"`
}

func newFileFromResponse(resp findFileResponse) *GridFSFile {
	return &GridFSFile{
		ID:         resp.ID,
		Length:     resp.Length,
		ChunkSize:  resp.ChunkSize,
		UploadDate: resp.UploadDate,
		Name:       resp.Name,
		Metadata:   resp.Metadata,
	}
}

// UnmarshalBSON implements the bson.Unmarshaler interface.
func (f *GridFSFile) UnmarshalBSON(data []byte) error {
	var temp findFileResponse
	if err := bson.Unmarshal(data, &temp); err != nil {
		return err
	}

	f.ID = temp.ID
	f.Length = temp.Length
	f.ChunkSize = temp.ChunkSize
	f.UploadDate = temp.UploadDate
	f.Name = temp.Name
	f.Metadata = temp.Metadata

	return nil
}

func newGridFSDownloadStream(
	ctx context.Context,
	cancel context.CancelFunc,
	cursor *Cursor,
	chunkSize int32,
	file *GridFSFile,
) *GridFSDownloadStream {
	numChunks := int32(math.Ceil(float64(file.Length) / float64(chunkSize)))

	return &GridFSDownloadStream{
		numChunks: numChunks,
		chunkSize: chunkSize,
		cursor:    cursor,
		buffer:    make([]byte, chunkSize),
		done:      cursor == nil,
		fileLen:   file.Length,
		file:      file,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Close closes this download stream.
func (ds *GridFSDownloadStream) Close() error {
	defer func() {
		if ds.cancel != nil {
			ds.cancel()
		}
	}()

	if ds.closed {
		return ErrStreamClosed
	}

	ds.closed = true
	if ds.cursor != nil {
		return ds.cursor.Close(context.Background())
	}
	return nil
}

// Read reads the file from the server and writes it to a destination byte slice.
func (ds *GridFSDownloadStream) Read(p []byte) (int, error) {
	if ds.closed {
		return 0, ErrStreamClosed
	}

	if ds.done {
		return 0, io.EOF
	}

	bytesCopied := 0
	var err error
	for bytesCopied < len(p) {
		if ds.bufferStart >= ds.bufferEnd {
			// Buffer is empty and can load in data from new chunk.
			err = ds.fillBuffer(ds.ctx)
			if err != nil {
				if errors.Is(err, errNoMoreChunks) {
					if bytesCopied == 0 {
						ds.done = true
						return 0, io.EOF
					}
					return bytesCopied, nil
				}
				return bytesCopied, err
			}
		}

		copied := copy(p[bytesCopied:], ds.buffer[ds.bufferStart:ds.bufferEnd])

		bytesCopied += copied
		ds.bufferStart += copied
	}

	return len(p), nil
}

// Skip skips a given number of bytes in the file.
func (ds *GridFSDownloadStream) Skip(skip int64) (int64, error) {
	if ds.closed {
		return 0, ErrStreamClosed
	}

	if ds.done {
		return 0, nil
	}

	var skipped int64
	var err error

	for skipped < skip {
		if ds.bufferStart >= ds.bufferEnd {
			// Buffer is empty and can load in data from new chunk.
			err = ds.fillBuffer(ds.ctx)
			if err != nil {
				if errors.Is(err, errNoMoreChunks) {
					return skipped, nil
				}
				return skipped, err
			}
		}

		toSkip := skip - skipped
		// Cap the amount to skip to the remaining bytes in the buffer to be consumed.
		bufferRemaining := ds.bufferEnd - ds.bufferStart
		if toSkip > int64(bufferRemaining) {
			toSkip = int64(bufferRemaining)
		}

		skipped += toSkip
		ds.bufferStart += int(toSkip)
	}

	return skip, nil
}

// GetFile returns a File object representing the file being downloaded.
func (ds *GridFSDownloadStream) GetFile() *GridFSFile {
	return ds.file
}

func (ds *GridFSDownloadStream) fillBuffer(ctx context.Context) error {
	if !ds.cursor.Next(ctx) {
		ds.done = true
		// Check for cursor error, otherwise there are no more chunks.
		if ds.cursor.Err() != nil {
			_ = ds.cursor.Close(ctx)
			return ds.cursor.Err()
		}
		// If there are no more chunks, but we didn't read the expected number of chunks, return an
		// ErrMissingChunk error to indicate that we're missing chunks at the end of the file.
		if ds.expectedChunk != ds.numChunks {
			return ErrMissingChunk
		}
		return errNoMoreChunks
	}

	chunkIndex, err := ds.cursor.Current.LookupErr("n")
	if err != nil {
		return err
	}

	var chunkIndexInt32 int32
	if chunkIndexInt64, ok := chunkIndex.Int64OK(); ok {
		chunkIndexInt32 = int32(chunkIndexInt64)
	} else {
		chunkIndexInt32 = chunkIndex.Int32()
	}

	if chunkIndexInt32 != ds.expectedChunk {
		return ErrMissingChunk
	}

	ds.expectedChunk++
	data, err := ds.cursor.Current.LookupErr("data")
	if err != nil {
		return err
	}

	_, dataBytes := data.Binary()
	copied := copy(ds.buffer, dataBytes)

	bytesLen := int32(len(dataBytes))
	if ds.expectedChunk == ds.numChunks {
		// final chunk can be fewer than ds.chunkSize bytes
		bytesDownloaded := int64(ds.chunkSize) * (int64(ds.expectedChunk) - int64(1))
		bytesRemaining := ds.fileLen - bytesDownloaded

		if int64(bytesLen) != bytesRemaining {
			return ErrWrongSize
		}
	} else if bytesLen != ds.chunkSize {
		// all intermediate chunks must have size ds.chunkSize
		return ErrWrongSize
	}

	ds.bufferStart = 0
	ds.bufferEnd = copied

	return nil
}
