// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package gridfs

import (
	"bytes"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/internal/testutil"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/options"
)

func createBucket(t *testing.T, bucketOpts *options.BucketOptions) *Bucket {
	cs := testutil.ConnString(t)
	client, err := mongo.NewClient(cs.String())
	testhelpers.RequireNil(t, err, "error creating client: %s", err)

	err = client.Connect(ctx)
	testhelpers.RequireNil(t, err, "error connecting client: %s", err)

	db = client.Database("gridFSTestDB")
	bucket, err := NewBucket(db, bucketOpts)
	if err != nil {
		t.Fatalf("Failed to create bucket.")
	}
	return bucket
}

var chunkSizeTests = []struct {
	testName   string
	bucketOpts *options.BucketOptions
	uploadOpts *options.UploadOptions
}{
	{"Default values", nil, nil},
	{"Bucket set", options.GridFSBucket().SetChunkSizeBytes(27), nil},
	{"Upload stream set", nil, options.GridFSUpload().SetChunkSizeBytes(27)},
	{"Bucket and upload set to different values", options.GridFSBucket().SetChunkSizeBytes(27), options.GridFSUpload().SetChunkSizeBytes(31)},
}

// Unit tests showing the chunk size is set correctly on the bucket and upload stream objects.
func TestGridFS_ChunkSize(t *testing.T) {
	for _, tt := range chunkSizeTests {
		t.Run(tt.testName, func(t *testing.T) {
			bucket := createBucket(t, tt.bucketOpts)
			us, err := bucket.OpenUploadStream("filename", tt.uploadOpts)
			if err != nil {
				t.Fatalf("Failed to open upload stream.")
			}

			expectedBucketChunkSize := DefaultChunkSize
			if tt.bucketOpts != nil {
				expectedBucketChunkSize = *tt.bucketOpts.ChunkSizeBytes
			}
			if bucket.chunkSize != expectedBucketChunkSize {
				t.Errorf("Bucket had wrong chunkSize. Want %v, got %v.", expectedBucketChunkSize, bucket.chunkSize)
			}

			expectedUploadChunkSize := expectedBucketChunkSize
			if tt.uploadOpts != nil {
				expectedUploadChunkSize = *tt.uploadOpts.ChunkSizeBytes
			}
			if us.chunkSize != expectedUploadChunkSize {
				t.Errorf("Upload stream had wrong chunkSize. Want %v, got %v.", expectedUploadChunkSize, us.chunkSize)
			}

		})
	}
}

// Unit tests showing that UploadFromStream creates indexes on the chunks and files collections.
func TestGridFS_Index(t *testing.T) {
	bucket := createBucket(t, nil)

	bucket.filesColl.Drop(ctx)
	bucket.chunksColl.Drop(ctx)

	byteData := []byte("Hello, world!")
	r := bytes.NewReader(byteData)

	_, err := bucket.UploadFromStream("filename", r)
	if err != nil {
		t.Fatalf("Failed to open upload stream.")
	}

	// Check for correct index in files collection.
	cur, err := bucket.filesColl.Indexes().List(ctx)
	foundIndex := false
	for cur.Next(ctx) {
		elem := &bson.Raw{}
		if err := cur.Decode(elem); err != nil {
			t.Fatalf("%v", err)
		}
		if _, err := elem.LookupErr("key", "filename"); err != nil {
			foundIndex = true
		}
	}
	if !foundIndex {
		t.Errorf("Expected an index on filename, but did not find one.")
	}

	// Check for correct index in chunks collection.
	cur, err = bucket.chunksColl.Indexes().List(ctx)
	foundIndex = false
	for cur.Next(ctx) {
		elem := &bson.Raw{}
		if err := cur.Decode(elem); err != nil {
			t.Fatalf("%v", err)
		}
		if _, err := elem.LookupErr("key", "files_id"); err != nil {
			foundIndex = true
		}
	}
	if !foundIndex {
		t.Errorf("Expected an index on files_id, but did not find one.")
	}
}

func TestGridFS_Write(t *testing.T) {
	bucket := createBucket(t, nil)

	bucket.filesColl.Drop(ctx)
	bucket.chunksColl.Drop(ctx)

	// Test that Upload works when the buffer to write is longer than the upload stream's internal buffer.
	// This requires multiple calls to uploadChunks.
	size := UploadBufferSize + 1000000
	p := make([]byte, size)
	for i := 0; i < size; i++ {
		p[i] = byte((i % 57) + 65)
	}

	_, err := bucket.UploadFromStream("filename", bytes.NewReader(p))
	if err != nil {
		t.Fatalf("Upload failed.")
	}

	w := bytes.NewBuffer(make([]byte, 0))
	_, err = bucket.DownloadToStreamByName("filename", w)
	if !bytes.Equal(p, w.Bytes()) {
		t.Fatal("Downloaded file did not match p.")
	}
}
