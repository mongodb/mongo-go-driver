// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package gridfs

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	"github.com/mongodb/mongo-go-driver/internal/testutil"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/options"

	"golang.org/x/net/context"
)

var chunkSizeTests = []struct {
	testName   string
	bucketOpts *options.BucketOptions
	uploadOpts *options.UploadOptions
}{
	{"Default values", nil, nil},
	{"Options provided without chunk size", options.GridFSBucket(), options.GridFSUpload()},
	{"Bucket chunk size set", options.GridFSBucket().SetChunkSizeBytes(27), nil},
	{"Upload stream chunk size set", nil, options.GridFSUpload().SetChunkSizeBytes(27)},
	{"Bucket and upload set to different values", options.GridFSBucket().SetChunkSizeBytes(27), options.GridFSUpload().SetChunkSizeBytes(31)},
}

func findIndex(t *testing.T, coll *mongo.Collection, ctx context.Context, keys ...string) {
	cur, err := coll.Indexes().List(ctx)
	if err != nil {
		t.Fatalf("Couldn't establish a cursor on the collection %v: %v", coll.Name(), err)
	}
	foundIndex := false
	for cur.Next(ctx) {
		elem, err := cur.DecodeBytes()
		if err != nil {
			t.Fatalf("%v", err)
		}
		if _, err := elem.LookupErr(keys...); err == nil {
			foundIndex = true
		}
	}
	if !foundIndex {
		t.Errorf("Expected index on %v, but did not find one.", keys)
	}
}

func TestGridFS(t *testing.T) {
	cs := testutil.ConnString(t)
	client, err := mongo.NewClient(cs.String())
	testhelpers.RequireNil(t, err, "error creating client: %s", err)

	ctx := context.Background()
	err = client.Connect(ctx)
	testhelpers.RequireNil(t, err, "error connecting client: %s", err)

	db := client.Database("gridFSTestDB")

	// Unit tests showing the chunk size is set correctly on the bucket and upload stream objects.
	t.Run("ChunkSize", func(t *testing.T) {
		for _, tt := range chunkSizeTests {
			t.Run(tt.testName, func(t *testing.T) {
				bucket, err := NewBucket(db, tt.bucketOpts)
				if err != nil {
					t.Fatalf("Failed to create bucket.")
				}

				us, err := bucket.OpenUploadStream("filename", tt.uploadOpts)
				if err != nil {
					t.Fatalf("Failed to open upload stream.")
				}

				expectedBucketChunkSize := DefaultChunkSize
				if tt.bucketOpts != nil && tt.bucketOpts.ChunkSizeBytes != nil {
					expectedBucketChunkSize = *tt.bucketOpts.ChunkSizeBytes
				}
				if bucket.chunkSize != expectedBucketChunkSize {
					t.Errorf("Bucket had wrong chunkSize. Want %v, got %v.", expectedBucketChunkSize, bucket.chunkSize)
				}

				expectedUploadChunkSize := expectedBucketChunkSize
				if tt.uploadOpts != nil && tt.uploadOpts.ChunkSizeBytes != nil {
					expectedUploadChunkSize = *tt.uploadOpts.ChunkSizeBytes
				}
				if us.chunkSize != expectedUploadChunkSize {
					t.Errorf("Upload stream had wrong chunkSize. Want %v, got %v.", expectedUploadChunkSize, us.chunkSize)
				}

			})
		}
	})

	// Unit tests showing that UploadFromStream creates indexes on the chunks and files collections.
	t.Run("IndexCreation", func(t *testing.T) {
		bucket, err := NewBucket(db, nil)
		if err != nil {
			t.Fatalf("Failed to create bucket.")
		}
		bucket.SetWriteDeadline(time.Now().Add(5 * time.Second))
		bucket.Drop()

		byteData := []byte("Hello, world!")
		r := bytes.NewReader(byteData)

		_, err = bucket.UploadFromStream("filename", r)
		if err != nil {
			t.Fatalf("Failed to open upload stream.")
		}

		findCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		findIndex(t, bucket.filesColl, findCtx, "key", "filename")
		findIndex(t, bucket.chunksColl, findCtx, "key", "files_id")
	})

	t.Run("RoundTrip", func(t *testing.T) {
		bucket, err := NewBucket(db, nil)
		if err != nil {
			t.Fatalf("Failed to create bucket.")
		}
		bucket.SetWriteDeadline(time.Now().Add(5 * time.Second))
		bucket.Drop()

		// Test that Upload works when the buffer to write is longer than the upload stream's internal buffer.
		// This requires multiple calls to uploadChunks.
		size := UploadBufferSize + 1000000
		p := make([]byte, size)
		for i := 0; i < size; i++ {
			p[i] = byte(rand.Intn(100))
		}

		_, err = bucket.UploadFromStream("filename", bytes.NewReader(p))
		if err != nil {
			t.Fatalf("Upload failed.")
		}

		w := bytes.NewBuffer(make([]byte, 0))
		_, err = bucket.DownloadToStreamByName("filename", w)
		if err != nil {
			t.Fatalf("Download failed.")
		}

		if !bytes.Equal(p, w.Bytes()) {
			t.Errorf("Downloaded file did not match p.")
		}
	})
}
