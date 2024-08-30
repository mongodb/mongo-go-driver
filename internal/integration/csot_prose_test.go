// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/integtest"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestCSOTProse(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().CreateClient(false))

	mt.RunOpts("1. multi-batch writes", mtest.NewOptions().MinServerVersion("4.4").
		Topologies(mtest.Single), func(mt *mtest.T) {
		// Test that multi-batch writes do not refresh the Timeout between batches.

		err := mt.Client.Database("db").Collection("coll").Drop(context.Background())
		assert.Nil(mt, err, "Drop error: %v", err)

		// Configure a fail point to block both inserts of the multi-write for 1010ms (2020ms total).
		mt.SetFailPoint(mtest.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: mtest.FailPointMode{
				Times: 2,
			},
			Data: mtest.FailPointData{
				FailCommands:    []string{"insert"},
				BlockConnection: true,
				BlockTimeMS:     1010,
			},
		})

		// Use a separate client with 2s Timeout and a separate command monitor to run a multi-batch
		// insert against db.coll.
		var started []*event.CommandStartedEvent
		cm := &event.CommandMonitor{
			Started: func(_ context.Context, evt *event.CommandStartedEvent) {
				started = append(started, evt)
			},
		}
		cliOptions := options.Client().
			SetTimeout(2 * time.Second).
			SetMonitor(cm).
			ApplyURI(mtest.ClusterURI())
		integtest.AddTestServerAPIVersion(cliOptions)
		cli, err := mongo.Connect(cliOptions)
		assert.Nil(mt, err, "Connect error: %v", err)

		// Insert 50 1MB documents (OP_MSG payloads can only fit 48MB in one batch).
		var bigStringBuilder strings.Builder
		for i := 0; i < 1024*1024; i++ {
			bigStringBuilder.WriteByte('a')
		}
		bigString := bigStringBuilder.String()
		var docs []interface{}
		for i := 0; i < 50; i++ {
			docs = append(docs, bson.D{{"1mb", bigString}})
		}

		// Expect a timeout error from InsertMany (from the second batch).
		_, err = cli.Database("db").Collection("coll").InsertMany(context.Background(), docs)
		assert.NotNil(mt, err, "expected error from InsertMany, got nil")
		assert.True(mt, mongo.IsTimeout(err), "expected error to be a timeout, got %v", err)

		// Expect that two 'insert's were sent.
		assert.True(mt, len(started) == 2, "expected two started events, got %d", len(started))
		assert.Equal(mt, started[0].CommandName,
			"insert", "expected an insert event, got %v", started[0].CommandName)
		assert.Equal(mt, started[1].CommandName,
			"insert", "expected a second insert event, got %v", started[1].CommandName)
	})

	mt.RunOpts("6. gridfs - upload", mtest.NewOptions().MinServerVersion("4.4"), func(mt *mtest.T) {
		// Drop and re-create the db.fs.files and db.fs.chunks collections.
		err := mt.Client.Database("db").Collection("fs.files").Drop(context.Background())
		assert.NoError(t, err, "failed to drop files")

		err = mt.Client.Database("db").Collection("fs.chunks").Drop(context.Background())
		assert.NoError(t, err, "failed to drop chunks")

		// Set a blocking "insert" fail point.
		mt.SetFailPoint(mtest.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: mtest.FailPointMode{
				Times: 1,
			},
			Data: mtest.FailPointData{
				FailCommands:    []string{"insert"},
				BlockConnection: true,
				BlockTimeMS:     15,
			},
		})

		// Create a new MongoClient with timeoutMS=10.
		cliOptions := options.Client().SetTimeout(10).ApplyURI(mtest.ClusterURI())
		integtest.AddTestServerAPIVersion(cliOptions)

		client, err := mongo.Connect(cliOptions)
		assert.NoError(t, err, "failed to connect to server")

		// Create a GridFS bucket that wraps the db database.
		bucket := client.Database("db").GridFSBucket()

		// Note that UploadFromStream accounts for the following steps:
		// - Call bucket.open_upload_stream()
		// - Using uploadStream, upload a single 0x12 byte
		// - Call uploadStream.close() to flush the stream and insert chunks
		_, err = bucket.UploadFromStream(context.Background(), "filename", bytes.NewReader([]byte{0x12}))
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})

	const test61 = "6.1 gridfs - upload and download with non-expiring client-level timeout"
	mt.RunOpts(test61, mtest.NewOptions().MinServerVersion("4.4"), func(mt *mtest.T) {
		// Drop and re-create the db.fs.files and db.fs.chunks collections.
		err := mt.Client.Database("db").Collection("fs.files").Drop(context.Background())
		assert.NoError(t, err, "failed to drop files")

		err = mt.Client.Database("db").Collection("fs.chunks").Drop(context.Background())
		assert.NoError(t, err, "failed to drop chunks")

		// Create a new MongoClient with timeoutMS=10.
		cliOptions := options.Client().SetTimeout(500 * time.Millisecond).ApplyURI(mtest.ClusterURI())
		integtest.AddTestServerAPIVersion(cliOptions)

		client, err := mongo.Connect(cliOptions)
		assert.NoError(t, err, "failed to connect to server")

		// Create a GridFS bucket that wraps the db database.
		bucket := client.Database("db").GridFSBucket()

		// Upload file and ensure it uploaded correctly.
		fileID, err := bucket.UploadFromStream(context.Background(), "filename", bytes.NewReader([]byte{0x12}))
		assert.NoError(t, err, "failed to upload stream")

		buf := bytes.Buffer{}

		_, err = bucket.DownloadToStream(context.Background(), fileID, &buf)
		assert.NoError(t, err, "failed to download stream")
		assert.Equal(t, buf.Len(), 1)
		assert.Equal(t, buf.Bytes(), []byte{0x12})
	})

	const test62 = "6.2 gridfs - upload with operation-level timeout"
	mt.RunOpts(test62, mtest.NewOptions().MinServerVersion("4.4"), func(mt *mtest.T) {
		// Drop and re-create the db.fs.files and db.fs.chunks collections.
		err := mt.Client.Database("db").Collection("fs.files").Drop(context.Background())
		assert.NoError(t, err, "failed to drop files")

		err = mt.Client.Database("db").Collection("fs.chunks").Drop(context.Background())
		assert.NoError(t, err, "failed to drop chunks")

		// Set a blocking "insert" fail point.
		mt.SetFailPoint(mtest.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: mtest.FailPointMode{
				Times: 1,
			},
			Data: mtest.FailPointData{
				FailCommands:    []string{"insert"},
				BlockConnection: true,
				BlockTimeMS:     15,
			},
		})

		// Create a new MongoClient with timeoutMS=10.
		cliOptions := options.Client().SetTimeout(10 * time.Second).ApplyURI(mtest.ClusterURI())
		integtest.AddTestServerAPIVersion(cliOptions)

		client, err := mongo.Connect(cliOptions)
		assert.NoError(t, err, "failed to connect to server")

		// Create a GridFS bucket that wraps the db database.
		bucket := client.Database("db").GridFSBucket()

		// If the operation-level context is not respected, then the client-level
		// timeout will exceed deadline.
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		// Upload file and ensure it uploaded correctly.
		fileID, err := bucket.UploadFromStream(ctx, "filename", bytes.NewReader([]byte{0x12}))
		assert.NoError(t, err, "failed to upload stream")

		buf := bytes.Buffer{}

		_, err = bucket.DownloadToStream(ctx, fileID, &buf)
		assert.NoError(t, err, "failed to download stream")
		assert.Equal(t, buf.Len(), 1)
		assert.Equal(t, buf.Bytes(), []byte{0x12})
	})

	mt.Run("8. server selection", func(mt *mtest.T) {
		cliOpts := options.Client().ApplyURI("mongodb://invalid/?serverSelectionTimeoutMS=100")
		mtOpts := mtest.NewOptions().ClientOptions(cliOpts).CreateCollection(false)
		mt.RunOpts("serverSelectionTimeoutMS honored if timeoutMS is not set", mtOpts, func(mt *mtest.T) {
			// TODO(GODRIVER-3266): Why do parallel tests fail on windows builds?
			// mt.Parallel()

			callback := func() bool {
				err := mt.Client.Ping(context.Background(), nil)
				assert.Error(mt, err, "expected Ping error, got nil")
				return true
			}

			// Assert that Ping fails within 150ms due to server selection timeout.
			assert.Eventually(t,
				callback,
				150*time.Millisecond,
				time.Millisecond,
				"expected ping to fail within 150ms")
		})

		cliOpts = options.Client().ApplyURI("mongodb://invalid/?timeoutMS=100&serverSelectionTimeoutMS=200")
		mtOpts = mtest.NewOptions().ClientOptions(cliOpts).CreateCollection(false)
		mt.RunOpts("timeoutMS honored for server selection if it's lower than serverSelectionTimeoutMS", mtOpts, func(mt *mtest.T) {
			mt.Parallel()

			callback := func() bool {
				err := mt.Client.Ping(context.Background(), nil)
				assert.Error(mt, err, "expected Ping error, got nil")
				return true
			}

			// Assert that Ping fails within 150ms due to timeout.
			assert.Eventually(t,
				callback,
				150*time.Millisecond,
				time.Millisecond,
				"expected ping to fail within 150ms")
		})

		cliOpts = options.Client().ApplyURI("mongodb://invalid/?timeoutMS=200&serverSelectionTimeoutMS=100")
		mtOpts = mtest.NewOptions().ClientOptions(cliOpts).CreateCollection(false)
		mt.RunOpts("serverSelectionTimeoutMS honored for server selection if it's lower than timeoutMS", mtOpts, func(mt *mtest.T) {
			mt.Parallel()

			callback := func() bool {
				err := mt.Client.Ping(context.Background(), nil)
				assert.Error(mt, err, "expected Ping error, got nil")
				return true
			}

			// Assert that Ping fails within 150ms due to server selection timeout.
			assert.Eventually(t,
				callback,
				150*time.Millisecond,
				time.Millisecond,
				"expected ping to fail within 150ms")
		})

		cliOpts = options.Client().ApplyURI("mongodb://invalid/?timeoutMS=0&serverSelectionTimeoutMS=100")
		mtOpts = mtest.NewOptions().ClientOptions(cliOpts).CreateCollection(false)
		mt.RunOpts("serverSelectionTimeoutMS honored for server selection if timeoutMS=0", mtOpts, func(mt *mtest.T) {
			mt.Parallel()

			callback := func() bool {
				err := mt.Client.Ping(context.Background(), nil)
				assert.Error(mt, err, "expected Ping error, got nil")
				return true
			}

			// Assert that Ping fails within 150ms due to server selection timeout.
			assert.Eventually(t,
				callback,
				150*time.Millisecond,
				time.Millisecond,
				"expected ping to fail within 150ms")
		})
	})
}
