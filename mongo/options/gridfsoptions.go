// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

// DefaultName is the default name for a GridFS bucket.
var DefaultName = "fs"

// DefaultChunkSize is the default size of each file chunk in bytes (255 KiB).
var DefaultChunkSize int32 = 255 * 1024

// DefaultRevision is the default revision number for a download by name operation.
var DefaultRevision int32 = -1

// BucketArgs represents arguments that can be used to configure GridFS bucket.
type BucketArgs struct {
	// The name of the bucket. The default value is "fs".
	Name *string

	// The number of bytes in each chunk in the bucket. The default value is 255 KiB.
	ChunkSizeBytes *int32

	// The write concern for the bucket. The default value is the write concern of the database from which the bucket
	// is created.
	WriteConcern *writeconcern.WriteConcern

	// The read concern for the bucket. The default value is the read concern of the database from which the bucket
	// is created.
	ReadConcern *readconcern.ReadConcern

	// The read preference for the bucket. The default value is the read preference of the database from which the
	// bucket is created.
	ReadPreference *readpref.ReadPref
}

// BucketOptions contains options to configure a gridfs bucket. Each option can
// be set through setter functions. See documentation for each setter function
// for an explanation of the option.
type BucketOptions struct {
	Opts []func(*BucketArgs) error
}

// GridFSBucket creates a new BucketOptions instance.
func GridFSBucket() *BucketOptions {
	bo := &BucketOptions{}
	bo.SetName(DefaultName).SetChunkSizeBytes(DefaultChunkSize)

	return bo
}

// ArgsSetters returns a list of CountArgs setter functions.
func (b *BucketOptions) ArgsSetters() []func(*BucketArgs) error {
	return b.Opts
}

// SetName sets the value for the Name field.
func (b *BucketOptions) SetName(name string) *BucketOptions {
	b.Opts = append(b.Opts, func(args *BucketArgs) error {
		args.Name = &name

		return nil
	})

	return b
}

// SetChunkSizeBytes sets the value for the ChunkSize field.
func (b *BucketOptions) SetChunkSizeBytes(i int32) *BucketOptions {
	b.Opts = append(b.Opts, func(args *BucketArgs) error {
		args.ChunkSizeBytes = &i

		return nil
	})

	return b
}

// SetWriteConcern sets the value for the WriteConcern field.
func (b *BucketOptions) SetWriteConcern(wc *writeconcern.WriteConcern) *BucketOptions {
	b.Opts = append(b.Opts, func(args *BucketArgs) error {
		args.WriteConcern = wc

		return nil
	})

	return b
}

// SetReadConcern sets the value for the ReadConcern field.
func (b *BucketOptions) SetReadConcern(rc *readconcern.ReadConcern) *BucketOptions {
	b.Opts = append(b.Opts, func(args *BucketArgs) error {
		args.ReadConcern = rc

		return nil
	})

	return b
}

// SetReadPreference sets the value for the ReadPreference field.
func (b *BucketOptions) SetReadPreference(rp *readpref.ReadPref) *BucketOptions {
	b.Opts = append(b.Opts, func(args *BucketArgs) error {
		args.ReadPreference = rp

		return nil
	})

	return b
}

// GridFSUploadArgs represents arguments that can be used to configure a GridFS
// upload operation.
type GridFSUploadArgs struct {
	// The number of bytes in each chunk in the bucket. The default value is DefaultChunkSize (255 KiB).
	ChunkSizeBytes *int32

	// Additional application data that will be stored in the "metadata" field of the document in the files collection.
	// The default value is nil, which means that the document in the files collection will not contain a "metadata"
	// field.
	Metadata interface{}

	// The BSON registry to use for converting filters to BSON documents. The default value is bson.DefaultRegistry.
	Registry *bsoncodec.Registry
}

// GridFSUploadOptions contains options to configure a GridFS Upload. Each
// option can be set through setter functions. See documentation for each setter
// function for an explanation of the option.
type GridFSUploadOptions struct {
	Opts []func(*GridFSUploadArgs) error
}

// GridFSUpload creates a new GridFSUploadOptions instance.
func GridFSUpload() *GridFSUploadOptions {
	opts := &GridFSUploadOptions{}
	opts.SetRegistry(bson.DefaultRegistry)

	return opts
}

// ArgsSetters returns a list of GridFSUploadArgs setter functions.
func (u *GridFSUploadOptions) ArgsSetters() []func(*GridFSUploadArgs) error {
	return u.Opts
}

// SetChunkSizeBytes sets the value for the ChunkSize field.
func (u *GridFSUploadOptions) SetChunkSizeBytes(i int32) *GridFSUploadOptions {
	u.Opts = append(u.Opts, func(args *GridFSUploadArgs) error {
		args.ChunkSizeBytes = &i

		return nil
	})

	return u
}

// SetMetadata sets the value for the Metadata field.
func (u *GridFSUploadOptions) SetMetadata(doc interface{}) *GridFSUploadOptions {
	u.Opts = append(u.Opts, func(args *GridFSUploadArgs) error {
		args.Metadata = doc

		return nil
	})

	return u
}

// SetRegistry sets the bson codec registry for the Registry field.
func (u *GridFSUploadOptions) SetRegistry(registry *bsoncodec.Registry) *GridFSUploadOptions {
	u.Opts = append(u.Opts, func(args *GridFSUploadArgs) error {
		args.Registry = registry

		return nil
	})

	return u
}

// GridFSNameArgs represents arguments that can be used to configure a GridFS
// DownloadByName operation.
type GridFSNameArgs struct {
	// Specifies the revision of the file to retrieve. Revision numbers are defined as follows:
	//
	// * 0 = the original stored file
	// * 1 = the first revision
	// * 2 = the second revision
	// * etc..
	// * -2 = the second most recent revision
	// * -1 = the most recent revision.
	//
	// The default value is -1
	Revision *int32
}

// GridFSNameOptions contains options to configure a GridFS name. Each option
// can be set through setter functions. See documentation for each setter
// function for an explanation of the option.
type GridFSNameOptions struct {
	Opts []func(*GridFSNameArgs) error
}

// GridFSName creates a new GridFSNameOptions instance.
func GridFSName() *GridFSNameOptions {
	return &GridFSNameOptions{}
}

// ArgsSetters returns a list of GridFSNameArgs setter functions.
func (n *GridFSNameOptions) ArgsSetters() []func(*GridFSNameArgs) error {
	return n.Opts
}

// SetRevision sets the value for the Revision field.
func (n *GridFSNameOptions) SetRevision(r int32) *GridFSNameOptions {
	n.Opts = append(n.Opts, func(args *GridFSNameArgs) error {
		args.Revision = &r

		return nil
	})

	return n
}

// GridFSFindArgs represents options that can be used to configure a GridFS Find
// operation.
type GridFSFindArgs struct {
	// If true, the server can write temporary data to disk while executing the find operation. The default value
	// is false. This option is only valid for MongoDB versions >= 4.4. For previous server versions, the server will
	// return an error if this option is used.
	AllowDiskUse *bool

	// The maximum number of documents to be included in each batch returned by the server.
	BatchSize *int32

	// The maximum number of documents to return. The default value is 0, which means that all documents matching the
	// filter will be returned. A negative limit specifies that the resulting documents should be returned in a single
	// batch. The default value is 0.
	Limit *int32

	// The maximum amount of time that the query can run on the server. The default value is nil, meaning that there
	// is no time limit for query execution.
	//
	// NOTE(benjirewis): MaxTime will be deprecated in a future release. The more general Timeout option may be used
	// in its place to control the amount of time that a single operation can run before returning an error. MaxTime
	// is ignored if Timeout is set on the client.
	MaxTime *time.Duration

	// If true, the cursor created by the operation will not timeout after a period of inactivity. The default value
	// is false.
	NoCursorTimeout *bool

	// The number of documents to skip before adding documents to the result. The default value is 0.
	Skip *int32

	// A document specifying the order in which documents should be returned.  The driver will return an error if the
	// sort parameter is a multi-key map.
	Sort interface{}
}

// GridFSFindOptions contains options to configure find operations. Each option
// can be set through setter functions. See documentation for each setter
// function for an explanation of the option.
type GridFSFindOptions struct {
	Opts []func(*GridFSFindArgs) error
}

// GridFSFind creates a new GridFSFindOptions instance.
func GridFSFind() *GridFSFindOptions {
	return &GridFSFindOptions{}
}

// ArgsSetters returns a list of GridFSFindArgs setter functions.
func (f *GridFSFindOptions) ArgsSetters() []func(*GridFSFindArgs) error {
	return f.Opts
}

// SetAllowDiskUse sets the value for the AllowDiskUse field.
func (f *GridFSFindOptions) SetAllowDiskUse(b bool) *GridFSFindOptions {
	f.Opts = append(f.Opts, func(args *GridFSFindArgs) error {
		args.AllowDiskUse = &b

		return nil
	})

	return f
}

// SetBatchSize sets the value for the BatchSize field.
func (f *GridFSFindOptions) SetBatchSize(i int32) *GridFSFindOptions {
	f.Opts = append(f.Opts, func(args *GridFSFindArgs) error {
		args.BatchSize = &i

		return nil
	})

	return f
}

// SetLimit sets the value for the Limit field.
func (f *GridFSFindOptions) SetLimit(i int32) *GridFSFindOptions {
	f.Opts = append(f.Opts, func(args *GridFSFindArgs) error {
		args.Limit = &i

		return nil
	})

	return f
}

// SetMaxTime sets the value for the MaxTime field.
//
// NOTE(benjirewis): MaxTime will be deprecated in a future release. The more general Timeout
// option may be used in its place to control the amount of time that a single operation can
// run before returning an error. MaxTime is ignored if Timeout is set on the client.
func (f *GridFSFindOptions) SetMaxTime(d time.Duration) *GridFSFindOptions {
	f.Opts = append(f.Opts, func(args *GridFSFindArgs) error {
		args.MaxTime = &d

		return nil
	})

	return f
}

// SetNoCursorTimeout sets the value for the NoCursorTimeout field.
func (f *GridFSFindOptions) SetNoCursorTimeout(b bool) *GridFSFindOptions {
	f.Opts = append(f.Opts, func(args *GridFSFindArgs) error {
		args.NoCursorTimeout = &b

		return nil
	})

	return f
}

// SetSkip sets the value for the Skip field.
func (f *GridFSFindOptions) SetSkip(i int32) *GridFSFindOptions {
	f.Opts = append(f.Opts, func(args *GridFSFindArgs) error {
		args.Skip = &i

		return nil
	})

	return f
}

// SetSort sets the value for the Sort field.
func (f *GridFSFindOptions) SetSort(sort interface{}) *GridFSFindOptions {
	f.Opts = append(f.Opts, func(args *GridFSFindArgs) error {
		args.Sort = sort

		return nil
	})

	return f
}
