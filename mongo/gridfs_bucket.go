// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/internal/csot"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

// TODO: add sessions options

// DefaultGridFSChunkSize is the default size of each file chunk.
const DefaultGridFSChunkSize int32 = 255 * 1024 // 255 KiB

// ErrFileNotFound occurs if a user asks to download a file with a file ID that isn't found in the files collection.
var ErrFileNotFound = errors.New("file with given parameters not found")

// ErrMissingGridFSChunkSize occurs when downloading a file if the files
// collection document is missing the "chunkSize" field.
var ErrMissingGridFSChunkSize = errors.New("files collection document does not contain a 'chunkSize' field")

// GridFSBucket represents a GridFS bucket.
type GridFSBucket struct {
	db         *Database
	chunksColl *Collection // collection to store file chunks
	filesColl  *Collection // collection to store file metadata

	name      string
	chunkSize int32
	wc        *writeconcern.WriteConcern
	rc        *readconcern.ReadConcern
	rp        *readpref.ReadPref

	firstWriteDone bool
	readBuf        []byte
	writeBuf       []byte
}

// Upload contains options to upload a file to a bucket.
type Upload struct {
	chunkSize int32
	metadata  bson.D
}

// OpenUploadStream creates a file ID new upload stream for a file given the
// filename.
//
// The context provided to this method controls the entire lifetime of an
// upload stream io.Writer.
func (b *GridFSBucket) OpenUploadStream(
	ctx context.Context,
	filename string,
	opts ...*options.UploadOptions,
) (*GridFSUploadStream, error) {
	return b.OpenUploadStreamWithID(ctx, primitive.NewObjectID(), filename, opts...)
}

// OpenUploadStreamWithID creates a new upload stream for a file given the file
// ID and filename.
//
// The context provided to this method controls the entire lifetime of an
// upload stream io.Writer.
func (b *GridFSBucket) OpenUploadStreamWithID(
	ctx context.Context,
	fileID interface{},
	filename string,
	opts ...*options.UploadOptions,
) (*GridFSUploadStream, error) {
	if err := b.checkFirstWrite(ctx); err != nil {
		return nil, err
	}

	upload, err := b.parseUploadOptions(opts...)
	if err != nil {
		return nil, err
	}

	return newUploadStream(ctx, upload, fileID, filename, b.chunksColl, b.filesColl), nil
}

// UploadFromStream creates a fileID and uploads a file given a source stream.
//
// If this upload requires a custom write deadline to be set on the bucket, it
// cannot be done concurrently with other write operations operations on this
// bucket that also require a custom deadline.
//
// The context provided to this method controls the entire lifetime of an
// upload stream io.Writer.
func (b *GridFSBucket) UploadFromStream(
	ctx context.Context,
	filename string,
	source io.Reader,
	opts ...*options.UploadOptions,
) (primitive.ObjectID, error) {
	fileID := primitive.NewObjectID()
	err := b.UploadFromStreamWithID(ctx, fileID, filename, source, opts...)
	return fileID, err
}

// UploadFromStreamWithID uploads a file given a source stream.
//
// If this upload requires a custom write deadline to be set on the bucket, it
// cannot be done concurrently with other write operations operations on this
// bucket that also require a custom deadline.
//
// The context provided to this method controls the entire lifetime of an
// upload stream io.Writer.
func (b *GridFSBucket) UploadFromStreamWithID(
	ctx context.Context,
	fileID interface{},
	filename string,
	source io.Reader,
	opts ...*options.UploadOptions,
) error {
	us, err := b.OpenUploadStreamWithID(ctx, fileID, filename, opts...)
	if err != nil {
		return err
	}

	for {
		n, err := source.Read(b.readBuf)
		if err != nil && err != io.EOF {
			_ = us.Abort() // upload considered aborted if source stream returns an error
			return err
		}

		if n > 0 {
			_, err := us.Write(b.readBuf[:n])
			if err != nil {
				return err
			}
		}

		if n == 0 || err == io.EOF {
			break
		}
	}

	return us.Close()
}

// OpenDownloadStream creates a stream from which the contents of the file can
// be read.
//
// The context provided to this method controls the entire lifetime of a
// download stream io.Reader.
func (b *GridFSBucket) OpenDownloadStream(ctx context.Context, fileID interface{}) (*GridFSDownloadStream, error) {
	return b.openDownloadStream(ctx, bson.D{{"_id", fileID}})
}

// DownloadToStream downloads the file with the specified fileID and writes it
// to the provided io.Writer. Returns the number of bytes written to the stream
// and an error, or nil if there was no error.
//
// If this download requires a custom read deadline to be set on the bucket, it
// cannot be done concurrently with other read operations operations on this
// bucket that also require a custom deadline.
//
// The context provided to this method controls the entire lifetime of a
// download stream io.Reader.
func (b *GridFSBucket) DownloadToStream(ctx context.Context, fileID interface{}, stream io.Writer) (int64, error) {
	ds, err := b.OpenDownloadStream(ctx, fileID)
	if err != nil {
		return 0, err
	}

	return b.downloadToStream(ds, stream)
}

// OpenDownloadStreamByName opens a download stream for the file with the given
// filename.
//
// The context provided to this method controls the entire lifetime of a
// download stream io.Reader.
func (b *GridFSBucket) OpenDownloadStreamByName(
	ctx context.Context,
	filename string,
	opts ...*options.NameOptions,
) (*GridFSDownloadStream, error) {
	var numSkip int32 = -1
	var sortOrder int32 = 1

	nameOpts := options.GridFSName()
	nameOpts.Revision = &options.DefaultRevision

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if opt.Revision != nil {
			nameOpts.Revision = opt.Revision
		}
	}
	if nameOpts.Revision != nil {
		numSkip = *nameOpts.Revision
	}

	if numSkip < 0 {
		sortOrder = -1
		numSkip = (-1 * numSkip) - 1
	}

	findOpts := options.FindOne().SetSkip(int64(numSkip)).SetSort(bson.D{{"uploadDate", sortOrder}})

	return b.openDownloadStream(ctx, bson.D{{"filename", filename}}, findOpts)
}

// DownloadToStreamByName downloads the file with the given name to the given
// io.Writer.
//
// If this download requires a custom read deadline to be set on the bucket, it
// cannot be done concurrently with other read operations operations on this
// bucket that also require a custom deadline.
//
// The context provided to this method controls the entire lifetime of a
// download stream io.Reader.
func (b *GridFSBucket) DownloadToStreamByName(
	ctx context.Context,
	filename string,
	stream io.Writer,
	opts ...*options.NameOptions,
) (int64, error) {
	ds, err := b.OpenDownloadStreamByName(ctx, filename, opts...)
	if err != nil {
		return 0, err
	}

	return b.downloadToStream(ds, stream)
}

// Delete deletes all chunks and metadata associated with the file with the given file ID and runs the underlying
// delete operations with the provided context.
//
// Use the context parameter to time-out or cancel the delete operation. The deadline set by SetWriteDeadline is ignored.
func (b *GridFSBucket) Delete(ctx context.Context, fileID interface{}) error {
	// If no deadline is set on the passed-in context, Timeout is set on the Client, and context is
	// not already a Timeout context, honor Timeout in new Timeout context for operation execution to
	// be shared by both delete operations.
	if _, deadlineSet := ctx.Deadline(); !deadlineSet && b.db.Client().Timeout() != nil && !csot.IsTimeoutContext(ctx) {
		newCtx, cancelFunc := csot.MakeTimeoutContext(ctx, *b.db.Client().Timeout())
		// Redefine ctx to be the new timeout-derived context.
		ctx = newCtx
		// Cancel the timeout-derived context at the end of Execute to avoid a context leak.
		defer cancelFunc()
	}

	// Delete document in files collection and then chunks to minimize race conditions.
	res, err := b.filesColl.DeleteOne(ctx, bson.D{{"_id", fileID}})
	if err == nil && res.DeletedCount == 0 {
		err = ErrFileNotFound
	}
	if err != nil {
		_ = b.deleteChunks(ctx, fileID) // Can attempt to delete chunks even if no docs in files collection matched.
		return err
	}

	return b.deleteChunks(ctx, fileID)
}

// Find returns the files collection documents that match the given filter and runs the underlying
// find query with the provided context.
//
// Use the context parameter to time-out or cancel the find operation. The deadline set by SetReadDeadline
// is ignored.
func (b *GridFSBucket) Find(
	ctx context.Context,
	filter interface{},
	opts ...*options.GridFSFindOptions,
) (*Cursor, error) {
	gfsOpts := options.GridFSFind()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if opt.AllowDiskUse != nil {
			gfsOpts.AllowDiskUse = opt.AllowDiskUse
		}
		if opt.BatchSize != nil {
			gfsOpts.BatchSize = opt.BatchSize
		}
		if opt.Limit != nil {
			gfsOpts.Limit = opt.Limit
		}
		if opt.MaxTime != nil {
			gfsOpts.MaxTime = opt.MaxTime
		}
		if opt.NoCursorTimeout != nil {
			gfsOpts.NoCursorTimeout = opt.NoCursorTimeout
		}
		if opt.Skip != nil {
			gfsOpts.Skip = opt.Skip
		}
		if opt.Sort != nil {
			gfsOpts.Sort = opt.Sort
		}
	}
	find := options.Find()
	if gfsOpts.AllowDiskUse != nil {
		find.SetAllowDiskUse(*gfsOpts.AllowDiskUse)
	}
	if gfsOpts.BatchSize != nil {
		find.SetBatchSize(*gfsOpts.BatchSize)
	}
	if gfsOpts.Limit != nil {
		find.SetLimit(int64(*gfsOpts.Limit))
	}
	if gfsOpts.MaxTime != nil {
		find.SetMaxTime(*gfsOpts.MaxTime)
	}
	if gfsOpts.NoCursorTimeout != nil {
		find.SetNoCursorTimeout(*gfsOpts.NoCursorTimeout)
	}
	if gfsOpts.Skip != nil {
		find.SetSkip(int64(*gfsOpts.Skip))
	}
	if gfsOpts.Sort != nil {
		find.SetSort(gfsOpts.Sort)
	}

	return b.filesColl.Find(ctx, filter, find)
}

// Rename renames the stored file with the specified file ID.
//
// If this operation requires a custom write deadline to be set on the bucket, it cannot be done concurrently with other
// write operations operations on this bucket that also require a custom deadline
//
// Use SetWriteDeadline to set a deadline for the rename operation.
func (b *GridFSBucket) Rename(ctx context.Context, fileID interface{}, newFilename string) error {
	res, err := b.filesColl.UpdateOne(ctx,
		bson.D{{"_id", fileID}},
		bson.D{{"$set", bson.D{{"filename", newFilename}}}},
	)
	if err != nil {
		return err
	}

	if res.MatchedCount == 0 {
		return ErrFileNotFound
	}

	return nil
}

// Drop drops the files and chunks collections associated with this bucket and runs the drop operations with
// the provided context.
//
// Use the context parameter to time-out or cancel the drop operation. The deadline set by SetWriteDeadline is ignored.
func (b *GridFSBucket) Drop(ctx context.Context) error {
	// If no deadline is set on the passed-in context, Timeout is set on the Client, and context is
	// not already a Timeout context, honor Timeout in new Timeout context for operation execution to
	// be shared by both drop operations.
	if _, deadlineSet := ctx.Deadline(); !deadlineSet && b.db.Client().Timeout() != nil && !csot.IsTimeoutContext(ctx) {
		newCtx, cancelFunc := csot.MakeTimeoutContext(ctx, *b.db.Client().Timeout())
		// Redefine ctx to be the new timeout-derived context.
		ctx = newCtx
		// Cancel the timeout-derived context at the end of Execute to avoid a context leak.
		defer cancelFunc()
	}

	err := b.filesColl.Drop(ctx)
	if err != nil {
		return err
	}

	return b.chunksColl.Drop(ctx)
}

// GetFilesCollection returns a handle to the collection that stores the file documents for this bucket.
func (b *GridFSBucket) GetFilesCollection() *Collection {
	return b.filesColl
}

// GetChunksCollection returns a handle to the collection that stores the file chunks for this bucket.
func (b *GridFSBucket) GetChunksCollection() *Collection {
	return b.chunksColl
}

func (b *GridFSBucket) openDownloadStream(
	ctx context.Context,
	filter interface{},
	opts ...*options.FindOneOptions,
) (*GridFSDownloadStream, error) {
	result := b.filesColl.FindOne(ctx, filter, opts...)

	// Unmarshal the data into a File instance, which can be passed to newGridFSDownloadStream. The _id value has to be
	// parsed out separately because "_id" will not match the File.ID field and we want to avoid exposing BSON tags
	// in the File type. After parsing it, use RawValue.Unmarshal to ensure File.ID is set to the appropriate value.
	var resp findFileResponse
	if err := result.Decode(&resp); err != nil {
		if errors.Is(err, ErrNoDocuments) {
			return nil, ErrFileNotFound
		}

		return nil, fmt.Errorf("error decoding files collection document: %w", err)
	}

	foundFile := newFileFromResponse(resp)

	if foundFile.Length == 0 {
		return newGridFSDownloadStream(ctx, nil, foundFile.ChunkSize, foundFile), nil
	}

	// For a file with non-zero length, chunkSize must exist so we know what size to expect when downloading chunks.
	if foundFile.ChunkSize == 0 {
		return nil, ErrMissingGridFSChunkSize
	}

	chunksCursor, err := b.findChunks(ctx, foundFile.ID)
	if err != nil {
		return nil, err
	}
	// The chunk size can be overridden for individual files, so the expected chunk size should be the "chunkSize"
	// field from the files collection document, not the bucket's chunk size.
	return newGridFSDownloadStream(ctx, chunksCursor, foundFile.ChunkSize, foundFile), nil
}

func (b *GridFSBucket) downloadToStream(ds *GridFSDownloadStream, stream io.Writer) (int64, error) {
	copied, err := io.Copy(stream, ds)
	if err != nil {
		_ = ds.Close()
		return 0, err
	}

	return copied, ds.Close()
}

func (b *GridFSBucket) deleteChunks(ctx context.Context, fileID interface{}) error {
	_, err := b.chunksColl.DeleteMany(ctx, bson.D{{"files_id", fileID}})
	return err
}

func (b *GridFSBucket) findChunks(ctx context.Context, fileID interface{}) (*Cursor, error) {
	chunksCursor, err := b.chunksColl.Find(ctx,
		bson.D{{"files_id", fileID}},
		options.Find().SetSort(bson.D{{"n", 1}})) // sort by chunk index
	if err != nil {
		return nil, err
	}

	return chunksCursor, nil
}

// returns true if the 2 index documents are equal
func numericalIndexDocsEqual(expected, actual bsoncore.Document) (bool, error) {
	if bytes.Equal(expected, actual) {
		return true, nil
	}

	actualElems, err := actual.Elements()
	if err != nil {
		return false, err
	}
	expectedElems, err := expected.Elements()
	if err != nil {
		return false, err
	}

	if len(actualElems) != len(expectedElems) {
		return false, nil
	}

	for idx, expectedElem := range expectedElems {
		actualElem := actualElems[idx]
		if actualElem.Key() != expectedElem.Key() {
			return false, nil
		}

		actualVal := actualElem.Value()
		expectedVal := expectedElem.Value()
		actualInt, actualOK := actualVal.AsInt64OK()
		expectedInt, expectedOK := expectedVal.AsInt64OK()

		//GridFS indexes always have numeric values
		if !actualOK || !expectedOK {
			return false, nil
		}

		if actualInt != expectedInt {
			return false, nil
		}
	}
	return true, nil
}

// Create an index if it doesn't already exist
func createNumericalIndexIfNotExists(ctx context.Context, iv IndexView, model IndexModel) error {
	c, err := iv.List(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = c.Close(ctx)
	}()

	modelKeysBytes, err := bson.Marshal(model.Keys)
	if err != nil {
		return err
	}
	modelKeysDoc := bsoncore.Document(modelKeysBytes)

	for c.Next(ctx) {
		keyElem, err := c.Current.LookupErr("key")
		if err != nil {
			return err
		}

		keyElemDoc := keyElem.Document()

		found, err := numericalIndexDocsEqual(modelKeysDoc, bsoncore.Document(keyElemDoc))
		if err != nil {
			return err
		}
		if found {
			return nil
		}
	}

	_, err = iv.CreateOne(ctx, model)
	return err
}

// create indexes on the files and chunks collection if needed
func (b *GridFSBucket) createIndexes(ctx context.Context) error {
	// must use primary read pref mode to check if files coll empty
	cloned, err := b.filesColl.Clone(options.Collection().SetReadPreference(readpref.Primary()))
	if err != nil {
		return err
	}

	docRes := cloned.FindOne(ctx, bson.D{}, options.FindOne().SetProjection(bson.D{{"_id", 1}}))

	_, err = docRes.Raw()
	if err != ErrNoDocuments {
		// nil, or error that occurred during the FindOne operation
		return err
	}

	filesIv := b.filesColl.Indexes()
	chunksIv := b.chunksColl.Indexes()

	filesModel := IndexModel{
		Keys: bson.D{
			{"filename", int32(1)},
			{"uploadDate", int32(1)},
		},
	}

	chunksModel := IndexModel{
		Keys: bson.D{
			{"files_id", int32(1)},
			{"n", int32(1)},
		},
		Options: options.Index().SetUnique(true),
	}

	if err = createNumericalIndexIfNotExists(ctx, filesIv, filesModel); err != nil {
		return err
	}
	return createNumericalIndexIfNotExists(ctx, chunksIv, chunksModel)
}

func (b *GridFSBucket) checkFirstWrite(ctx context.Context) error {
	if !b.firstWriteDone {
		// before the first write operation, must determine if files collection is empty
		// if so, create indexes if they do not already exist

		if err := b.createIndexes(ctx); err != nil {
			return err
		}
		b.firstWriteDone = true
	}

	return nil
}

func (b *GridFSBucket) parseUploadOptions(opts ...*options.UploadOptions) (*Upload, error) {
	upload := &Upload{
		chunkSize: b.chunkSize, // upload chunk size defaults to bucket's value
	}

	uo := options.GridFSUpload()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if opt.ChunkSizeBytes != nil {
			uo.ChunkSizeBytes = opt.ChunkSizeBytes
		}
		if opt.Metadata != nil {
			uo.Metadata = opt.Metadata
		}
		if opt.Registry != nil {
			uo.Registry = opt.Registry
		}
	}
	if uo.ChunkSizeBytes != nil {
		upload.chunkSize = *uo.ChunkSizeBytes
	}
	if uo.Registry == nil {
		uo.Registry = bson.DefaultRegistry
	}
	if uo.Metadata != nil {
		// TODO(GODRIVER-2726): Replace with marshal() and unmarshal() once the
		// TODO gridfs package is merged into the mongo package.
		raw, err := bson.MarshalWithRegistry(uo.Registry, uo.Metadata)
		if err != nil {
			return nil, err
		}
		var doc bson.D
		unMarErr := bson.UnmarshalWithRegistry(uo.Registry, raw, &doc)
		if unMarErr != nil {
			return nil, unMarErr
		}
		upload.metadata = doc
	}

	return upload, nil
}
