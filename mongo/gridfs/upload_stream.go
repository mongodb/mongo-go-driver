// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package gridfs

import (
	"errors"

	"math"

	"context"
	"time"

	"github.com/mongodb/mongo-go-driver/bson/primitive"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
)

// UploadBufferSize is the size in bytes of one stream batch. Chunks will be written to the db after the sum of chunk
// lengths is equal to the batch size.
const UploadBufferSize = 16 * 1000000 // 16 MB

// ErrStreamClosed is an error returned if an operation is attempted on a closed/aborted stream.
var ErrStreamClosed = errors.New("stream is closed or aborted")

// UploadStream is used to upload files in chunks.
type UploadStream struct {
	*Upload // chunk size and metadata
	FileID  primitive.ObjectID

	chunkIndex    int
	chunksColl    *mongo.Collection // collection to store file chunks
	filename      string
	filesColl     *mongo.Collection // collection to store file metadata
	closed        bool
	buffer        []byte
	bufferIndex   int
	fileLen       int64
	writeDeadline time.Time
}

// NewUploadStream creates a new upload stream.
func newUploadStream(upload *Upload, fileID primitive.ObjectID, filename string, chunks *mongo.Collection, files *mongo.Collection) *UploadStream {
	return &UploadStream{
		Upload: upload,
		FileID: fileID,

		chunksColl: chunks,
		filename:   filename,
		filesColl:  files,
		buffer:     make([]byte, UploadBufferSize),
	}
}

// Close closes this upload stream.
func (us *UploadStream) Close() error {
	if us.closed {
		return ErrStreamClosed
	}

	ctx, cancel := deadlineContext(us.writeDeadline)
	if cancel != nil {
		defer cancel()
	}

	if us.bufferIndex != 0 {
		if err := us.uploadChunks(ctx); err != nil {
			return err
		}
	}

	if err := us.createFilesCollDoc(ctx); err != nil {
		return err
	}

	us.closed = true
	return nil
}

// SetWriteDeadline sets the write deadline for this stream.
func (us *UploadStream) SetWriteDeadline(t time.Time) error {
	if us.closed {
		return ErrStreamClosed
	}

	us.writeDeadline = t
	return nil
}

// Write transfers the contents of a byte slice into this upload stream. If the stream's underlying buffer fills up,
// the buffer will be uploaded as chunks to the server. Implements the io.Writer interface.
func (us *UploadStream) Write(p []byte) (int, error) {
	if us.closed {
		return 0, ErrStreamClosed
	}

	var ctx context.Context

	ctx, cancel := deadlineContext(us.writeDeadline)
	if cancel != nil {
		defer cancel()
	}

	origLen := len(p)
	for {
		if len(p) == 0 {
			break
		}

		n := copy(us.buffer[us.bufferIndex:], p) // copy as much as possible
		p = p[n:]
		us.bufferIndex += n

		if us.bufferIndex == UploadBufferSize {
			err := us.uploadChunks(ctx)
			if err != nil {
				return 0, err
			}
			us.bufferIndex = 0
		}
	}
	return origLen, nil
}

// Abort closes the stream and deletes all file chunks that have already been written.
func (us *UploadStream) Abort() error {
	if us.closed {
		return ErrStreamClosed
	}

	ctx, cancel := deadlineContext(us.writeDeadline)
	if cancel != nil {
		defer cancel()
	}

	_, err := us.chunksColl.DeleteMany(ctx, bsonx.Doc{{"files_id", bsonx.ObjectID(us.FileID)}})
	if err != nil {
		return err
	}

	us.closed = true
	return nil
}

func (us *UploadStream) uploadChunks(ctx context.Context) error {
	numChunks := math.Ceil(float64(us.bufferIndex) / float64(us.chunkSize))

	docs := make([]interface{}, int(numChunks))
	begChunkIndex := us.chunkIndex
	for i := 0; i < us.bufferIndex; i += int(us.chunkSize) {
		var chunkData []byte
		if us.bufferIndex-i < int(us.chunkSize) {
			chunkData = us.buffer[i:us.bufferIndex]
		} else {
			chunkData = us.buffer[i : i+int(us.chunkSize)]
		}
		docs[us.chunkIndex-begChunkIndex] = bsonx.Doc{
			{"_id", bsonx.ObjectID(primitive.NewObjectID())},
			{"files_id", bsonx.ObjectID(us.FileID)},
			{"n", bsonx.Int32(int32(us.chunkIndex))},
			{"data", bsonx.Binary(0x00, chunkData)},
		}

		us.chunkIndex++
		us.fileLen += int64(len(chunkData))
	}

	_, err := us.chunksColl.InsertMany(ctx, docs)
	if err != nil {
		return err
	}

	return nil
}

func (us *UploadStream) createFilesCollDoc(ctx context.Context) error {
	doc := bsonx.Doc{
		{"_id", bsonx.ObjectID(us.FileID)},
		{"length", bsonx.Int64(us.fileLen)},
		{"chunkSize", bsonx.Int32(us.chunkSize)},
		{"uploadDate", bsonx.DateTime(time.Now().UnixNano() / int64(time.Millisecond))},
		{"filename", bsonx.String(us.filename)},
	}

	if us.metadata != nil {
		doc = append(doc, bsonx.Elem{"metadata", bsonx.Document(us.metadata)})
	}

	_, err := us.filesColl.InsertOne(ctx, doc)
	if err != nil {
		return err
	}

	return nil
}
