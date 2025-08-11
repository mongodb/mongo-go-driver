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

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/csot"
	"go.mongodb.org/mongo-driver/v2/internal/mongoutil"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
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

// upload contains options to upload a file to a bucket.
type upload struct {
	chunkSize int32
	metadata  bson.D
}

// OpenUploadStream creates a file ID new upload stream for a file given the
// filename.
//
// The context provided to this method controls the entire lifetime of an
// upload stream io.Writer. If the context does set a deadline, then the
// client-level timeout will be used to cap the lifetime of the stream.
func (b *GridFSBucket) OpenUploadStream(
	ctx context.Context,
	filename string,
	opts ...options.Lister[options.GridFSUploadOptions],
) (*GridFSUploadStream, error) {
	return b.OpenUploadStreamWithID(ctx, bson.NewObjectID(), filename, opts...)
}

// OpenUploadStreamWithID creates a new upload stream for a file given the file
// ID and filename.
//
// The context provided to this method controls the entire lifetime of an
// upload stream io.Writer. If the context does set a deadline, then the
// client-level timeout will be used to cap the lifetime of the stream.
func (b *GridFSBucket) OpenUploadStreamWithID(
	ctx context.Context,
	fileID any,
	filename string,
	opts ...options.Lister[options.GridFSUploadOptions],
) (*GridFSUploadStream, error) {
	ctx, cancel := csot.WithTimeout(ctx, b.db.client.timeout)

	if err := b.checkFirstWrite(ctx); err != nil {
		return nil, err
	}

	upload, err := b.parseGridFSUploadOptions(opts...)
	if err != nil {
		return nil, err
	}

	return newUploadStream(ctx, cancel, upload, fileID, filename, b.chunksColl, b.filesColl), nil
}

// UploadFromStream creates a fileID and uploads a file given a source stream.
//
// If this upload requires a custom write deadline to be set on the bucket, it
// cannot be done concurrently with other write operations operations on this
// bucket that also require a custom deadline.
//
// The context provided to this method controls the entire lifetime of an
// upload stream io.Writer. If the context does set a deadline, then the
// client-level timeout will be used to cap the lifetime of the stream.
func (b *GridFSBucket) UploadFromStream(
	ctx context.Context,
	filename string,
	source io.Reader,
	opts ...options.Lister[options.GridFSUploadOptions],
) (bson.ObjectID, error) {
	fileID := bson.NewObjectID()
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
// upload stream io.Writer. If the context does set a deadline, then the
// client-level timeout will be used to cap the lifetime of the stream.
func (b *GridFSBucket) UploadFromStreamWithID(
	ctx context.Context,
	fileID any,
	filename string,
	source io.Reader,
	opts ...options.Lister[options.GridFSUploadOptions],
) error {
	ctx, cancel := csot.WithTimeout(ctx, b.db.client.timeout)
	defer cancel()

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
// The context provided to this method controls the entire lifetime of an
// upload stream io.Writer. If the context does set a deadline, then the
// client-level timeout will be used to cap the lifetime of the stream.
func (b *GridFSBucket) OpenDownloadStream(ctx context.Context, fileID any) (*GridFSDownloadStream, error) {
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
// The context provided to this method controls the entire lifetime of an
// upload stream io.Writer. If the context does set a deadline, then the
// client-level timeout will be used to cap the lifetime of the stream.
func (b *GridFSBucket) DownloadToStream(ctx context.Context, fileID any, stream io.Writer) (int64, error) {
	ds, err := b.OpenDownloadStream(ctx, fileID)
	if err != nil {
		return 0, err
	}

	return b.downloadToStream(ds, stream)
}

// OpenDownloadStreamByName opens a download stream for the file with the given
// filename.
//
// The context provided to this method controls the entire lifetime of an
// upload stream io.Writer. If the context does set a deadline, then the
// client-level timeout will be used to cap the lifetime of the stream.
func (b *GridFSBucket) OpenDownloadStreamByName(
	ctx context.Context,
	filename string,
	opts ...options.Lister[options.GridFSNameOptions],
) (*GridFSDownloadStream, error) {
	args, err := mongoutil.NewOptions[options.GridFSNameOptions](opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to construct options from builder: %w", err)
	}

	numSkip := options.DefaultRevision
	if args.Revision != nil {
		numSkip = *args.Revision
	}

	var sortOrder int32 = 1

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
// The context provided to this method controls the entire lifetime of an
// upload stream io.Writer. If the context does set a deadline, then the
// client-level timeout will be used to cap the lifetime of the stream.
func (b *GridFSBucket) DownloadToStreamByName(
	ctx context.Context,
	filename string,
	stream io.Writer,
	opts ...options.Lister[options.GridFSNameOptions],
) (int64, error) {
	ds, err := b.OpenDownloadStreamByName(ctx, filename, opts...)
	if err != nil {
		return 0, err
	}

	return b.downloadToStream(ds, stream)
}

// Delete deletes all chunks and metadata associated with the file with the
// given file ID and runs the underlying delete operations with the provided
// context.
func (b *GridFSBucket) Delete(ctx context.Context, fileID any) error {
	ctx, cancel := csot.WithTimeout(ctx, b.db.client.timeout)
	defer cancel()

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

// Find returns the files collection documents that match the given filter and
// runs the underlying find query with the provided context.
func (b *GridFSBucket) Find(
	ctx context.Context,
	filter any,
	opts ...options.Lister[options.GridFSFindOptions],
) (*Cursor, error) {
	args, err := mongoutil.NewOptions[options.GridFSFindOptions](opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to construct options from builder: %w", err)
	}

	find := options.Find()
	if args.AllowDiskUse != nil {
		find.SetAllowDiskUse(*args.AllowDiskUse)
	}
	if args.BatchSize != nil {
		find.SetBatchSize(*args.BatchSize)
	}
	if args.Limit != nil {
		find.SetLimit(int64(*args.Limit))
	}
	if args.NoCursorTimeout != nil {
		find.SetNoCursorTimeout(*args.NoCursorTimeout)
	}
	if args.Skip != nil {
		find.SetSkip(int64(*args.Skip))
	}
	if args.Sort != nil {
		find.SetSort(args.Sort)
	}

	return b.filesColl.Find(ctx, filter, find)
}

// Rename renames the stored file with the specified file ID.
func (b *GridFSBucket) Rename(ctx context.Context, fileID any, newFilename string) error {
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

// Drop drops the files and chunks collections associated with this bucket and
// runs the drop operations with the provided context.
func (b *GridFSBucket) Drop(ctx context.Context) error {
	ctx, cancel := csot.WithTimeout(ctx, b.db.client.timeout)
	defer cancel()

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
	filter any,
	opts ...options.Lister[options.FindOneOptions],
) (*GridFSDownloadStream, error) {
	ctx, cancel := csot.WithTimeout(ctx, b.db.client.timeout)

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
		return newGridFSDownloadStream(ctx, cancel, nil, foundFile.ChunkSize, foundFile), nil
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
	return newGridFSDownloadStream(ctx, cancel, chunksCursor, foundFile.ChunkSize, foundFile), nil
}

func (b *GridFSBucket) downloadToStream(ds *GridFSDownloadStream, stream io.Writer) (int64, error) {
	copied, err := io.Copy(stream, ds)
	if err != nil {
		_ = ds.Close()
		return 0, err
	}

	return copied, ds.Close()
}

func (b *GridFSBucket) deleteChunks(ctx context.Context, fileID any) error {
	_, err := b.chunksColl.DeleteMany(ctx, bson.D{{"files_id", fileID}})
	return err
}

func (b *GridFSBucket) findChunks(ctx context.Context, fileID any) (*Cursor, error) {
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

		// GridFS indexes always have numeric values
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
	cloned := b.filesColl.Clone(options.Collection().SetReadPreference(readpref.Primary()))

	docRes := cloned.FindOne(ctx, bson.D{}, options.FindOne().SetProjection(bson.D{{"_id", 1}}))

	_, err := docRes.Raw()
	if !errors.Is(err, ErrNoDocuments) {
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

func (b *GridFSBucket) parseGridFSUploadOptions(opts ...options.Lister[options.GridFSUploadOptions]) (*upload, error) {
	upload := &upload{
		chunkSize: b.chunkSize, // upload chunk size defaults to bucket's value
	}

	args, err := mongoutil.NewOptions[options.GridFSUploadOptions](opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to construct options from builder: %w", err)
	}

	if args.ChunkSizeBytes != nil {
		upload.chunkSize = *args.ChunkSizeBytes
	}
	if args.Registry == nil {
		args.Registry = defaultRegistry
	}
	if args.Metadata != nil {
		// TODO(GODRIVER-2726): Replace with marshal() and unmarshal() once the
		// TODO gridfs package is merged into the mongo package.
		buf := new(bytes.Buffer)
		vw := bson.NewDocumentWriter(buf)
		enc := bson.NewEncoder(vw)
		enc.SetRegistry(args.Registry)
		err := enc.Encode(args.Metadata)
		if err != nil {
			return nil, err
		}
		var doc bson.D
		dec := bson.NewDecoder(bson.NewDocumentReader(bytes.NewReader(buf.Bytes())))
		dec.SetRegistry(args.Registry)
		unMarErr := dec.Decode(&doc)
		if unMarErr != nil {
			return nil, unMarErr
		}
		upload.metadata = doc
	}

	return upload, nil
}
