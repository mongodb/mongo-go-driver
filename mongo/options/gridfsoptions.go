// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

// DefaultName is the default name for a GridFS bucket.
var DefaultName = "fs"

// DefaultChunkSize is the default size of each file chunk in bytes (255 KiB).
var DefaultChunkSize int32 = 255 * 1024

// DefaultRevision is the default revision number for a download by name operation.
var DefaultRevision int32 = -1

// BucketOptions represents arguments that can be used to configure GridFS
// bucket.
//
// See corresponding setter methods for documentation.
type BucketOptions struct {
	Name           *string
	ChunkSizeBytes *int32
	WriteConcern   *writeconcern.WriteConcern
	ReadConcern    *readconcern.ReadConcern
	ReadPreference *readpref.ReadPref
}

// BucketOptionsBuilder contains options to configure a gridfs bucket. Each
// option can be set through setter functions. See documentation for each setter
// function for an explanation of the option.
type BucketOptionsBuilder struct {
	Opts []func(*BucketOptions) error
}

// GridFSBucket creates a new BucketOptions instance.
func GridFSBucket() *BucketOptionsBuilder {
	bo := &BucketOptionsBuilder{}
	bo.SetName(DefaultName).SetChunkSizeBytes(DefaultChunkSize)

	return bo
}

// List returns a list of CountOptions setter functions.
func (b *BucketOptionsBuilder) List() []func(*BucketOptions) error {
	return b.Opts
}

// SetName sets the value for the Name field. Specifies the name of the bucket.
// The default value is "fs".
func (b *BucketOptionsBuilder) SetName(name string) *BucketOptionsBuilder {
	b.Opts = append(b.Opts, func(opts *BucketOptions) error {
		opts.Name = &name

		return nil
	})

	return b
}

// SetChunkSizeBytes sets the value for the ChunkSize field. Specifies the number
// of bytes in each chunk in the bucket. The default value is 255 KiB.
func (b *BucketOptionsBuilder) SetChunkSizeBytes(i int32) *BucketOptionsBuilder {
	b.Opts = append(b.Opts, func(opts *BucketOptions) error {
		opts.ChunkSizeBytes = &i

		return nil
	})

	return b
}

// SetWriteConcern sets the value for the WriteConcern field. Specifies the write
// concern for the bucket. The default value is the write concern of the database
// from which the bucket is created.
func (b *BucketOptionsBuilder) SetWriteConcern(wc *writeconcern.WriteConcern) *BucketOptionsBuilder {
	b.Opts = append(b.Opts, func(opts *BucketOptions) error {
		opts.WriteConcern = wc

		return nil
	})

	return b
}

// SetReadConcern sets the value for the ReadConcern field. Specifies the read
// concern for the bucket. The default value is the read concern of the database
// from which the bucket is created.
func (b *BucketOptionsBuilder) SetReadConcern(rc *readconcern.ReadConcern) *BucketOptionsBuilder {
	b.Opts = append(b.Opts, func(opts *BucketOptions) error {
		opts.ReadConcern = rc

		return nil
	})

	return b
}

// SetReadPreference sets the value for the ReadPreference field. Specifies the
// read preference for the bucket. The default value is the read preference of
// the database from which the bucket is created.
func (b *BucketOptionsBuilder) SetReadPreference(rp *readpref.ReadPref) *BucketOptionsBuilder {
	b.Opts = append(b.Opts, func(opts *BucketOptions) error {
		opts.ReadPreference = rp

		return nil
	})

	return b
}

// GridFSUploadOptions represents arguments that can be used to configure a GridFS
// upload operation.
//
// See corresponding setter methods for documentation.
type GridFSUploadOptions struct {
	ChunkSizeBytes *int32
	Metadata       any
	Registry       *bson.Registry
}

// GridFSUploadOptionsBuilder contains options to configure a GridFS Upload.
// Each option can be set through setter functions. See documentation for each
// setter function for an explanation of the option.
type GridFSUploadOptionsBuilder struct {
	Opts []func(*GridFSUploadOptions) error
}

// GridFSUpload creates a new GridFSUploadOptions instance.
func GridFSUpload() *GridFSUploadOptionsBuilder {
	opts := &GridFSUploadOptionsBuilder{}
	opts.SetRegistry(defaultRegistry)

	return opts
}

// List returns a list of GridFSUploadOptions setter functions.
func (u *GridFSUploadOptionsBuilder) List() []func(*GridFSUploadOptions) error {
	return u.Opts
}

// SetChunkSizeBytes sets the value for the ChunkSize field. Specifies the number of
// bytes in each chunk in the bucket. The default value is DefaultChunkSize (255 KiB).
func (u *GridFSUploadOptionsBuilder) SetChunkSizeBytes(i int32) *GridFSUploadOptionsBuilder {
	u.Opts = append(u.Opts, func(opts *GridFSUploadOptions) error {
		opts.ChunkSizeBytes = &i

		return nil
	})

	return u
}

// SetMetadata sets the value for the Metadata field. Specifies additional application data
// that will be stored in the "metadata" field of the document in the files collection.
// The default value is nil, which means that the document in the files collection will
// not contain a "metadata" field.
func (u *GridFSUploadOptionsBuilder) SetMetadata(doc any) *GridFSUploadOptionsBuilder {
	u.Opts = append(u.Opts, func(opts *GridFSUploadOptions) error {
		opts.Metadata = doc

		return nil
	})

	return u
}

// SetRegistry sets the bson codec registry for the Registry field. Specifies the BSON
// registry to use for converting filters to BSON documents. The default value is
// bson.NewRegistry().
func (u *GridFSUploadOptionsBuilder) SetRegistry(registry *bson.Registry) *GridFSUploadOptionsBuilder {
	u.Opts = append(u.Opts, func(opts *GridFSUploadOptions) error {
		opts.Registry = registry

		return nil
	})

	return u
}

// GridFSNameOptions represents arguments that can be used to configure a GridFS
// DownloadByName operation.
//
// See corresponding setter methods for documentation.
type GridFSNameOptions struct {
	Revision *int32
}

// GridFSNameOptionsBuilder contains options to configure a GridFS name. Each
// option can be set through setter functions. See documentation for each setter
// function for an explanation of the option.
type GridFSNameOptionsBuilder struct {
	Opts []func(*GridFSNameOptions) error
}

// GridFSName creates a new GridFSNameOptions instance.
func GridFSName() *GridFSNameOptionsBuilder {
	return &GridFSNameOptionsBuilder{}
}

// List returns a list of GridFSNameOptions setter functions.
func (n *GridFSNameOptionsBuilder) List() []func(*GridFSNameOptions) error {
	return n.Opts
}

// SetRevision sets the value for the Revision field. Specifies the revision
// of the file to retrieve. Revision numbers are defined as follows:
//
// * 0 = the original stored file
// * 1 = the first revision
// * 2 = the second revision
// * etc..
// * -2 = the second most recent revision
// * -1 = the most recent revision.
//
// The default value is -1
func (n *GridFSNameOptionsBuilder) SetRevision(r int32) *GridFSNameOptionsBuilder {
	n.Opts = append(n.Opts, func(opts *GridFSNameOptions) error {
		opts.Revision = &r

		return nil
	})

	return n
}

// GridFSFindOptions represents arguments that can be used to configure a GridFS
// Find operation.
//
// See corresponding setter methods for documentation.
type GridFSFindOptions struct {
	AllowDiskUse    *bool
	BatchSize       *int32
	Limit           *int32
	NoCursorTimeout *bool
	Skip            *int32
	Sort            any
}

// GridFSFindOptionsBuilder contains options to configure find operations. Each
// option can be set through setter functions. See documentation for each setter
// function for an explanation of the option.
type GridFSFindOptionsBuilder struct {
	Opts []func(*GridFSFindOptions) error
}

// GridFSFind creates a new GridFSFindOptions instance.
func GridFSFind() *GridFSFindOptionsBuilder {
	return &GridFSFindOptionsBuilder{}
}

// List returns a list of GridFSFindOptions setter functions.
func (f *GridFSFindOptionsBuilder) List() []func(*GridFSFindOptions) error {
	return f.Opts
}

// SetAllowDiskUse sets the value for the AllowDiskUse field. If true, the server can
// write temporary data to disk while executing the find operation. The default value
// is false. This option is only valid for MongoDB versions >= 4.4. For previous server
// versions, the server will return an error if this option is used.
func (f *GridFSFindOptionsBuilder) SetAllowDiskUse(b bool) *GridFSFindOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *GridFSFindOptions) error {
		opts.AllowDiskUse = &b

		return nil
	})

	return f
}

// SetBatchSize sets the value for the BatchSize field. Specifies the maximum number
// of documents to be included in each batch returned by the server.
func (f *GridFSFindOptionsBuilder) SetBatchSize(i int32) *GridFSFindOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *GridFSFindOptions) error {
		opts.BatchSize = &i

		return nil
	})

	return f
}

// SetLimit sets the value for the Limit field. Specifies the maximum number of
// documents to return. The default value is 0, which means that all documents
// matching the filter will be returned. A negative limit specifies that the
// resulting documents should be returned in a single batch. The default value is 0.
func (f *GridFSFindOptionsBuilder) SetLimit(i int32) *GridFSFindOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *GridFSFindOptions) error {
		opts.Limit = &i

		return nil
	})

	return f
}

// SetNoCursorTimeout sets the value for the NoCursorTimeout field. If true, the
// cursor created by the operation will not timeout after a period of inactivity.
// The default value is false.
func (f *GridFSFindOptionsBuilder) SetNoCursorTimeout(b bool) *GridFSFindOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *GridFSFindOptions) error {
		opts.NoCursorTimeout = &b

		return nil
	})

	return f
}

// SetSkip sets the value for the Skip field. Specifies the number of documents
// to skip before adding documents to the result. The default value is 0.
func (f *GridFSFindOptionsBuilder) SetSkip(i int32) *GridFSFindOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *GridFSFindOptions) error {
		opts.Skip = &i

		return nil
	})

	return f
}

// SetSort sets the value for the Sort field. Sets a document specifying the order
// in which documents should be returned. The sort parameter is evaluated sequentially,
// so the driver will return an error if it is a multi-key map (which is unordeded).
// The default value is nil.
func (f *GridFSFindOptionsBuilder) SetSort(sort any) *GridFSFindOptionsBuilder {
	f.Opts = append(f.Opts, func(opts *GridFSFindOptions) error {
		opts.Sort = sort

		return nil
	})

	return f
}
