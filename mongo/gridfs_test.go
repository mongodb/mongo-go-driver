// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/integtest"
	"go.mongodb.org/mongo-driver/v2/internal/mongoutil"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

var (
	gridfsConnsCheckedOut int
)

func TestGridFS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cs := integtest.ConnString(t)
	poolMonitor := &event.PoolMonitor{
		Event: func(evt *event.PoolEvent) {
			switch evt.Type {
			case event.ConnectionCheckedOut:
				gridfsConnsCheckedOut++
			case event.ConnectionCheckedIn:
				gridfsConnsCheckedOut--
			}
		},
	}
	clientOpts := options.Client().
		ApplyURI(cs.Original).
		SetReadPreference(readpref.Primary()).
		SetWriteConcern(writeconcern.Majority()).
		SetPoolMonitor(poolMonitor).
		// Connect to a single host. For sharded clusters, this will pin to a single mongos, which avoids
		// non-deterministic versioning errors in the server. This has no effect for replica sets because the driver
		// will discover the other hosts during SDAM checks.
		SetHosts(cs.Hosts[:1])

	integtest.AddTestServerAPIVersion(clientOpts)

	client, err := Connect(clientOpts)
	assert.Nil(t, err, "Connect error: %v", err)
	db := client.Database("gridfs")
	defer func() {
		sessions := client.NumberSessionsInProgress()
		conns := gridfsConnsCheckedOut

		_ = db.Drop(context.Background())
		_ = client.Disconnect(context.Background())
		assert.Equal(t, 0, sessions, "%v sessions checked out", sessions)
		assert.Equal(t, 0, conns, "%v connections checked out", conns)
	}()

	// Unit tests showing the chunk size is set correctly on the bucket and upload stream objects.
	t.Run("ChunkSize", func(t *testing.T) {
		chunkSizeTests := []struct {
			testName   string
			bucketOpts *options.BucketOptionsBuilder
			uploadOpts *options.GridFSUploadOptionsBuilder
		}{
			{"Default values", nil, nil},
			{"Options provided without chunk size", options.GridFSBucket(), options.GridFSUpload()},
			{"Bucket chunk size set", options.GridFSBucket().SetChunkSizeBytes(27), nil},
			{"Upload stream chunk size set", nil, options.GridFSUpload().SetChunkSizeBytes(27)},
			{"Bucket and upload set to different values", options.GridFSBucket().SetChunkSizeBytes(27), options.GridFSUpload().SetChunkSizeBytes(31)},
		}

		for _, tt := range chunkSizeTests {
			t.Run(tt.testName, func(t *testing.T) {
				bucket := db.GridFSBucket(tt.bucketOpts)

				us, err := bucket.OpenUploadStream(context.Background(), "filename", tt.uploadOpts)
				assert.Nil(t, err, "OpenUploadStream error: %v", err)

				bucketArgs, err := mongoutil.NewOptions[options.BucketOptions](tt.bucketOpts)
				require.NoError(t, err, "failed to construct options from builder")

				expectedBucketChunkSize := DefaultGridFSChunkSize
				if tt.bucketOpts != nil && bucketArgs.ChunkSizeBytes != nil {
					expectedBucketChunkSize = *bucketArgs.ChunkSizeBytes
				}
				assert.Equal(t, expectedBucketChunkSize, bucket.chunkSize,
					"expected chunk size %v, got %v", expectedBucketChunkSize, bucket.chunkSize)

				uploadArgs, err := mongoutil.NewOptions[options.GridFSUploadOptions](tt.uploadOpts)
				require.NoError(t, err, "failed to construct options from builder")

				expectedUploadChunkSize := expectedBucketChunkSize
				if tt.uploadOpts != nil && uploadArgs.ChunkSizeBytes != nil {
					expectedUploadChunkSize = *uploadArgs.ChunkSizeBytes
				}
				assert.Equal(t, expectedUploadChunkSize, us.chunkSize,
					"expected chunk size %v, got %v", expectedUploadChunkSize, us.chunkSize)
			})
		}
	})
}

func TestGridFSFile_UnmarshalBSON(t *testing.T) {
	cs := integtest.ConnString(t)

	client, err := Connect(options.Client().ApplyURI(cs.Original))
	require.NoError(t, err)

	defer func() {
		err := client.Disconnect(context.Background())
		require.NoError(t, err)
	}()

	// Get the database and create a GridFS bucket
	db := client.Database("gridfs_test_db")

	// Drop the collection
	err = db.Collection("myfiles.files").Drop(context.Background())
	require.NoError(t, err)

	err = db.Collection("myfiles.chunks").Drop(context.Background())
	require.NoError(t, err)

	bucket := db.GridFSBucket(options.GridFSBucket().SetName("myfiles"))

	// Data to upload
	fileName := "example-file.txt"
	fileContent := []byte("Hello GridFS! This is a test file.")

	// Upload data into GridFS
	uploadStream, err := bucket.OpenUploadStream(context.Background(), fileName)
	require.NoError(t, err)

	_, err = uploadStream.Write(fileContent)
	require.NoError(t, err)

	uploadStream.Close()

	// Verify the file metadata
	fileCursor, err := bucket.Find(context.Background(), bson.D{})
	require.NoError(t, err)

	for fileCursor.Next(context.Background()) {
		var file GridFSFile
		err := fileCursor.Decode(&file)
		require.NoError(t, err)

		assert.NotNil(t, file.ID)
		assert.Equal(t, int64(34), file.Length)
		assert.Equal(t, int32(261120), file.ChunkSize)
		assert.NotNil(t, file.UploadDate)
		assert.Equal(t, fileName, file.Name)
	}
}
