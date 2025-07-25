// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"errors"

	"context"
	"time"

	"math"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// uploadBufferSize is the size in bytes of one stream batch. Chunks will be written to the db after the sum of chunk
// lengths is equal to the batch size.
const uploadBufferSize = 16 * 1024 * 1024 // 16 MiB

// ErrStreamClosed is an error returned if an operation is attempted on a closed/aborted stream.
var ErrStreamClosed = errors.New("stream is closed or aborted")

// GridFSUploadStream is used to upload a file in chunks. This type implements the io.Writer interface and a file can be
// uploaded using the Write method. After an upload is complete, the Close method must be called to write file
// metadata.
type GridFSUploadStream struct {
	*upload // chunk size and metadata
	FileID  any

	chunkIndex  int
	chunksColl  *Collection // collection to store file chunks
	filename    string
	filesColl   *Collection // collection to store file metadata
	closed      bool
	buffer      []byte
	bufferIndex int
	fileLen     int64
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewUploadStream creates a new upload stream.
func newUploadStream(
	ctx context.Context,
	cancel context.CancelFunc,
	up *upload,
	fileID any,
	filename string,
	chunks, files *Collection,
) *GridFSUploadStream {
	return &GridFSUploadStream{
		upload: up,
		FileID: fileID,

		chunksColl: chunks,
		filename:   filename,
		filesColl:  files,
		buffer:     make([]byte, uploadBufferSize),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Close writes file metadata to the files collection and cleans up any resources associated with the UploadStream.
func (us *GridFSUploadStream) Close() error {
	defer func() {
		if us.cancel != nil {
			us.cancel()
		}
	}()

	if us.closed {
		return ErrStreamClosed
	}

	if us.bufferIndex != 0 {
		if err := us.uploadChunks(us.ctx, true); err != nil {
			return err
		}
	}

	if err := us.createFilesCollDoc(us.ctx); err != nil {
		return err
	}

	us.closed = true
	return nil
}

// Write transfers the contents of a byte slice into this upload stream. If the stream's underlying buffer fills up,
// the buffer will be uploaded as chunks to the server. Implements the io.Writer interface.
func (us *GridFSUploadStream) Write(p []byte) (int, error) {
	if us.closed {
		return 0, ErrStreamClosed
	}

	origLen := len(p)
	for {
		if len(p) == 0 {
			break
		}

		n := copy(us.buffer[us.bufferIndex:], p) // copy as much as possible
		p = p[n:]
		us.bufferIndex += n

		if us.bufferIndex == uploadBufferSize {
			err := us.uploadChunks(us.ctx, false)
			if err != nil {
				return 0, err
			}
		}
	}
	return origLen, nil
}

// Abort closes the stream and deletes all file chunks that have already been written.
func (us *GridFSUploadStream) Abort() error {
	defer func() {
		if us.cancel != nil {
			us.cancel()
		}
	}()

	if us.closed {
		return ErrStreamClosed
	}

	_, err := us.chunksColl.DeleteMany(us.ctx, bson.D{{"files_id", us.FileID}})
	if err != nil {
		return err
	}

	us.closed = true
	return nil
}

// uploadChunks uploads the current buffer as a series of chunks to the bucket
// if uploadPartial is true, any data at the end of the buffer that is smaller than a chunk will be uploaded as a partial
// chunk. if it is false, the data will be moved to the front of the buffer.
// uploadChunks sets us.bufferIndex to the next available index in the buffer after uploading
func (us *GridFSUploadStream) uploadChunks(ctx context.Context, uploadPartial bool) error {
	chunks := float64(us.bufferIndex) / float64(us.chunkSize)
	numChunks := int(math.Ceil(chunks))
	if !uploadPartial {
		numChunks = int(math.Floor(chunks))
	}

	docs := make([]any, numChunks)

	begChunkIndex := us.chunkIndex
	for i := 0; i < us.bufferIndex; i += int(us.chunkSize) {
		endIndex := i + int(us.chunkSize)
		if us.bufferIndex-i < int(us.chunkSize) {
			// partial chunk
			if !uploadPartial {
				break
			}
			endIndex = us.bufferIndex
		}
		chunkData := us.buffer[i:endIndex]
		docs[us.chunkIndex-begChunkIndex] = bson.D{
			{"_id", bson.NewObjectID()},
			{"files_id", us.FileID},
			{"n", int32(us.chunkIndex)},
			{"data", bson.Binary{Subtype: 0x00, Data: chunkData}},
		}
		us.chunkIndex++
		us.fileLen += int64(len(chunkData))
	}

	_, err := us.chunksColl.InsertMany(ctx, docs)
	if err != nil {
		return err
	}

	// copy any remaining bytes to beginning of buffer and set buffer index
	bytesUploaded := numChunks * int(us.chunkSize)
	if bytesUploaded != uploadBufferSize && !uploadPartial {
		copy(us.buffer[0:], us.buffer[bytesUploaded:us.bufferIndex])
	}
	us.bufferIndex = uploadBufferSize - bytesUploaded
	return nil
}

func (us *GridFSUploadStream) createFilesCollDoc(ctx context.Context) error {
	doc := bson.D{
		{"_id", us.FileID},
		{"length", us.fileLen},
		{"chunkSize", us.chunkSize},
		{"uploadDate", bson.DateTime(time.Now().UnixNano() / int64(time.Millisecond))},
		{"filename", us.filename},
	}

	if us.metadata != nil {
		doc = append(doc, bson.E{"metadata", us.metadata})
	}

	_, err := us.filesColl.InsertOne(ctx, doc)
	if err != nil {
		return err
	}

	return nil
}
