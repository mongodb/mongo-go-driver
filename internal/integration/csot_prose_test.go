// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/failpoint"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/integtest"
	"go.mongodb.org/mongo-driver/v2/internal/mongoutil"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestCSOTProse(t *testing.T) {
	// Skip CSOT tests when SKIP_CSOT_TESTS=true. In Evergreen, we typically set
	// that environment variable on Windows and macOS because the CSOT spec
	// tests are unreliable on those hosts.
	if os.Getenv("SKIP_CSOT_TESTS") == "true" {
		t.Skip("Skipping CSOT test because SKIP_CSOT_TESTS=true")
	}

	mt := mtest.New(t, mtest.NewOptions().CreateClient(false))

	mt.RunOpts("1. multi-batch writes", mtest.NewOptions().MinServerVersion("4.4").
		Topologies(mtest.Single), func(mt *mtest.T) {
		// Test that multi-batch writes do not refresh the Timeout between batches.

		err := mt.Client.Database("db").Collection("coll").Drop(context.Background())
		assert.Nil(mt, err, "Drop error: %v", err)

		// Configure a fail point to block both inserts of the multi-write for 1010ms (2020ms total).
		mt.SetFailPoint(failpoint.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: failpoint.Mode{
				Times: 2,
			},
			Data: failpoint.Data{
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
		var docs []any
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
			assert.Eventually(mt,
				callback,
				150*time.Millisecond,
				time.Millisecond,
				"expected ping to fail within 150ms")
		})

		cliOpts = options.Client().ApplyURI("mongodb://invalid/?timeoutMS=100&serverSelectionTimeoutMS=200")
		mtOpts = mtest.NewOptions().ClientOptions(cliOpts).CreateCollection(false)
		mt.RunOpts("timeoutMS honored for server selection if it's lower than serverSelectionTimeoutMS", mtOpts, func(mt *mtest.T) {
			// TODO(GODRIVER-3266): Why do parallel tests fail on windows builds?
			// mt.Parallel()

			callback := func() bool {
				err := mt.Client.Ping(context.Background(), nil)
				assert.Error(mt, err, "expected Ping error, got nil")
				return true
			}

			// Assert that Ping fails within 150ms due to timeout.
			assert.Eventually(mt,
				callback,
				150*time.Millisecond,
				time.Millisecond,
				"expected ping to fail within 150ms")
		})

		cliOpts = options.Client().ApplyURI("mongodb://invalid/?timeoutMS=200&serverSelectionTimeoutMS=100")
		mtOpts = mtest.NewOptions().ClientOptions(cliOpts).CreateCollection(false)
		mt.RunOpts("serverSelectionTimeoutMS honored for server selection if it's lower than timeoutMS", mtOpts, func(mt *mtest.T) {
			// TODO(GODRIVER-3266): Why do parallel tests fail on windows builds?
			// mt.Parallel()

			callback := func() bool {
				err := mt.Client.Ping(context.Background(), nil)
				assert.Error(mt, err, "expected Ping error, got nil")
				return true
			}

			// Assert that Ping fails within 150ms due to server selection timeout.
			assert.Eventually(mt,
				callback,
				150*time.Millisecond,
				time.Millisecond,
				"expected ping to fail within 150ms")
		})

		cliOpts = options.Client().ApplyURI("mongodb://invalid/?timeoutMS=0&serverSelectionTimeoutMS=100")
		mtOpts = mtest.NewOptions().ClientOptions(cliOpts).CreateCollection(false)
		mt.RunOpts("serverSelectionTimeoutMS honored for server selection if timeoutMS=0", mtOpts, func(mt *mtest.T) {
			// TODO(GODRIVER-3266): Why do parallel tests fail on windows builds?
			// mt.Parallel()

			callback := func() bool {
				err := mt.Client.Ping(context.Background(), nil)
				assert.Error(mt, err, "expected Ping error, got nil")
				return true
			}

			// Assert that Ping fails within 150ms due to server selection timeout.
			assert.Eventually(mt,
				callback,
				150*time.Millisecond,
				time.Millisecond,
				"expected ping to fail within 150ms")
		})
	})

	mt.RunOpts("11. multi-batch bulkWrites", mtest.NewOptions().MinServerVersion("8.0").
		Topologies(mtest.Single), func(mt *mtest.T) {
		coll := mt.CreateCollection(mtest.Collection{DB: "db", Name: "coll"}, false)
		err := coll.Drop(context.Background())
		require.NoError(mt, err, "Drop error: %v", err)

		mt.SetFailPoint(failpoint.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: failpoint.Mode{
				Times: 2,
			},
			Data: failpoint.Data{
				FailCommands:    []string{"bulkWrite"},
				BlockConnection: true,
				BlockTimeMS:     1010,
			},
		})

		var hello struct {
			MaxBsonObjectSize   int
			MaxMessageSizeBytes int
		}
		err = mt.DB.RunCommand(context.Background(), bson.D{{"hello", 1}}).Decode(&hello)
		require.NoError(mt, err, "Hello error: %v", err)

		var writes []mongo.ClientBulkWrite
		n := hello.MaxMessageSizeBytes/hello.MaxBsonObjectSize + 1
		for i := 0; i < n; i++ {
			writes = append(writes, mongo.ClientBulkWrite{
				Database:   "db",
				Collection: "coll",
				Model: &mongo.ClientInsertOneModel{
					Document: bson.D{{"a", strings.Repeat("b", hello.MaxBsonObjectSize-500)}},
				},
			})
		}

		var cnt int
		cm := &event.CommandMonitor{
			Started: func(_ context.Context, evt *event.CommandStartedEvent) {
				if evt.CommandName == "bulkWrite" {
					cnt++
				}
			},
		}
		cliOptions := options.Client().
			SetTimeout(2 * time.Second).
			SetMonitor(cm).
			ApplyURI(mtest.ClusterURI())
		mt.ResetClient(cliOptions)
		_, err = mt.Client.BulkWrite(context.Background(), writes)
		assert.ErrorIs(mt, err, context.DeadlineExceeded, "expected a timeout error, got: %v", err)
		assert.Equal(mt, 2, cnt, "expected bulkWrite calls: %d, got: %d", 2, cnt)
	})
}

func TestCSOTProse_GridFS(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().CreateClient(false))

	mt.RunOpts("6. gridfs - upload", mtest.NewOptions().MinServerVersion("4.4"), func(mt *mtest.T) {
		// TODO(GODRIVER-3328): FailPoints are not currently reliable on sharded
		// topologies. Allow running on sharded topologies once that is fixed.
		noShardedOpts := mtest.NewOptions().Topologies(mtest.Single, mtest.ReplicaSet, mtest.LoadBalanced)
		mt.RunOpts("uploads via openUploadStream can be timed out", noShardedOpts, func(mt *mtest.T) {
			// Drop and re-create the db.fs.files and db.fs.chunks collections.
			err := mt.Client.Database("db").Collection("fs.files").Drop(context.Background())
			assert.NoError(mt, err, "failed to drop files")

			err = mt.Client.Database("db").Collection("fs.chunks").Drop(context.Background())
			assert.NoError(mt, err, "failed to drop chunks")

			hosts, err := mongoutil.HostsFromURI(mtest.ClusterURI())
			require.NoError(mt, err)

			failpointHost := hosts[0]

			mt.ResetClient(options.Client().
				SetHosts([]string{failpointHost}))

			// Set a blocking "insert" fail point.
			mt.SetFailPoint(failpoint.FailPoint{
				ConfigureFailPoint: "failCommand",
				Mode: failpoint.Mode{
					Times: 1,
				},
				Data: failpoint.Data{
					FailCommands:    []string{"insert"},
					BlockConnection: true,
					BlockTimeMS:     1250,
				},
			})

			// The automatic failpoint clearing may not clear failpoints set on
			// specific hosts, so manually clear the failpoint we set on the specific
			// mongos when the test is done.
			defer func() {
				mt.ResetClient(options.Client().
					SetHosts([]string{failpointHost}))
				mt.ClearFailPoints()
			}()

			// Create a new MongoClient with timeoutMS=1000.
			cliOptions := options.Client().SetTimeout(1000 * time.Millisecond).ApplyURI(mtest.ClusterURI()).
				SetHosts([]string{failpointHost})

			integtest.AddTestServerAPIVersion(cliOptions)

			client, err := mongo.Connect(cliOptions)
			assert.NoError(mt, err, "failed to connect to server")

			// Create a GridFS bucket that wraps the db database.
			bucket := client.Database("db").GridFSBucket()

			uploadStream, err := bucket.OpenUploadStream(context.Background(), "filename")
			require.NoError(mt, err, "failed to open upload stream")

			_, err = uploadStream.Write([]byte{0x12})
			require.NoError(mt, err, "failed to write to upload stream")

			err = uploadStream.Close()
			assert.Error(t, err, context.DeadlineExceeded)
		})

		// TODO(GODRIVER-3328): FailPoints are not currently reliable on sharded
		// topologies. Allow running on sharded topologies once that is fixed.
		mt.RunOpts("Aborting an upload stream can be timed out", noShardedOpts, func(mt *mtest.T) {
			// Drop and re-create the db.fs.files and db.fs.chunks collections.
			err := mt.Client.Database("db").Collection("fs.files").Drop(context.Background())
			assert.NoError(mt, err, "failed to drop files")

			err = mt.Client.Database("db").Collection("fs.chunks").Drop(context.Background())
			assert.NoError(mt, err, "failed to drop chunks")

			hosts, err := mongoutil.HostsFromURI(mtest.ClusterURI())
			require.NoError(mt, err)

			failpointHost := hosts[0]

			mt.ResetClient(options.Client().
				SetHosts([]string{failpointHost}))

			// Set a blocking "delete" fail point.
			mt.SetFailPoint(failpoint.FailPoint{
				ConfigureFailPoint: "failCommand",
				Mode: failpoint.Mode{
					Times: 1,
				},
				Data: failpoint.Data{
					FailCommands:    []string{"delete"},
					BlockConnection: true,
					BlockTimeMS:     1250,
				},
			})

			// The automatic failpoint clearing may not clear failpoints set on
			// specific hosts, so manually clear the failpoint we set on the specific
			// mongos when the test is done.
			defer func() {
				mt.ResetClient(options.Client().
					SetHosts([]string{failpointHost}))
				mt.ClearFailPoints()
			}()

			// Create a new MongoClient with timeoutMS=1000.
			cliOptions := options.Client().SetTimeout(1000 * time.Millisecond).ApplyURI(mtest.ClusterURI()).
				SetHosts([]string{failpointHost})
			integtest.AddTestServerAPIVersion(cliOptions)

			client, err := mongo.Connect(cliOptions)
			assert.NoError(mt, err, "failed to connect to server")

			// Create a GridFS bucket that wraps the db database.
			bucket := client.Database("db").GridFSBucket(options.GridFSBucket().SetChunkSizeBytes(2))

			// Call bucket.open_upload_stream() with the filename filename to create
			// an upload stream (referred to as uploadStream).
			uploadStream, err := bucket.OpenUploadStream(context.Background(), "filename")
			require.NoError(mt, err)

			// Using uploadStream, upload the bytes [0x01, 0x02, 0x03, 0x04].
			_, err = uploadStream.Write([]byte{0x01, 0x02, 0x03, 0x04})
			require.NoError(mt, err)

			err = uploadStream.Abort()
			assert.Error(mt, err, context.DeadlineExceeded)
		})
	})

	const test61 = "6.1 gridfs - upload and download with non-expiring client-level timeout"
	mt.RunOpts(test61, mtest.NewOptions().MinServerVersion("4.4"), func(mt *mtest.T) {
		// Drop and re-create the db.fs.files and db.fs.chunks collections.
		err := mt.Client.Database("db").Collection("fs.files").Drop(context.Background())
		assert.NoError(mt, err, "failed to drop files")

		err = mt.Client.Database("db").Collection("fs.chunks").Drop(context.Background())
		assert.NoError(mt, err, "failed to drop chunks")

		// Create a new MongoClient with timeoutMS=500.
		cliOptions := options.Client().SetTimeout(500 * time.Millisecond).ApplyURI(mtest.ClusterURI())
		integtest.AddTestServerAPIVersion(cliOptions)

		client, err := mongo.Connect(cliOptions)
		assert.NoError(mt, err, "failed to connect to server")

		// Create a GridFS bucket that wraps the db database.
		bucket := client.Database("db").GridFSBucket()

		mt.Run("UploadFromStream", func(mt *mtest.T) {
			// Upload file and ensure it uploaded correctly.
			fileID, err := bucket.UploadFromStream(context.Background(), "filename", bytes.NewReader([]byte{0x12}))
			assert.NoError(mt, err, "failed to upload stream")

			buf := bytes.Buffer{}

			_, err = bucket.DownloadToStream(context.Background(), fileID, &buf)
			assert.NoError(mt, err, "failed to download stream")
			assert.Equal(mt, buf.Len(), 1)
			assert.Equal(mt, buf.Bytes(), []byte{0x12})
		})

		mt.Run("OpenUploadStream", func(mt *mtest.T) {
			// Upload file and ensure it uploaded correctly.
			uploadStream, err := bucket.OpenUploadStream(context.Background(), "filename2")
			require.NoError(mt, err, "failed to open upload stream")

			_, err = uploadStream.Write([]byte{0x13})
			require.NoError(mt, err, "failed to write data to upload stream")

			err = uploadStream.Close()
			require.NoError(mt, err, "failed to close upload stream")

			buf := bytes.Buffer{}

			_, err = bucket.DownloadToStream(context.Background(), uploadStream.FileID, &buf)
			assert.NoError(mt, err, "failed to download stream")
			assert.Equal(mt, buf.Len(), 1)
			assert.Equal(mt, buf.Bytes(), []byte{0x13})
		})
	})

	const test62 = "6.2 gridfs - upload with operation-level timeout"
	mtOpts := mtest.NewOptions().
		MinServerVersion("4.4").
		// TODO(GODRIVER-3328): FailPoints are not currently reliable on sharded
		// topologies. Allow running on sharded topologies once that is fixed.
		Topologies(mtest.Single, mtest.ReplicaSet, mtest.LoadBalanced)
	mt.RunOpts(test62, mtOpts, func(mt *mtest.T) {
		// Drop and re-create the db.fs.files and db.fs.chunks collections.
		err := mt.Client.Database("db").Collection("fs.files").Drop(context.Background())
		assert.NoError(mt, err, "failed to drop files")

		err = mt.Client.Database("db").Collection("fs.chunks").Drop(context.Background())
		assert.NoError(mt, err, "failed to drop chunks")

		hosts, err := mongoutil.HostsFromURI(mtest.ClusterURI())
		require.NoError(mt, err)

		failpointHost := hosts[0]

		mt.ResetClient(options.Client().
			SetHosts([]string{failpointHost}))

		// Set a blocking "insert" fail point.
		mt.SetFailPoint(failpoint.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: failpoint.Mode{
				Times: 1,
			},
			Data: failpoint.Data{
				FailCommands:    []string{"insert"},
				BlockConnection: true,
				BlockTimeMS:     200,
			},
		})

		// The automatic failpoint clearing may not clear failpoints set on
		// specific hosts, so manually clear the failpoint we set on the specific
		// mongos when the test is done.
		defer func() {
			mt.ResetClient(options.Client().
				SetHosts([]string{failpointHost}))
			mt.ClearFailPoints()
		}()

		cliOptions := options.Client().SetTimeout(100 * time.Millisecond).ApplyURI(mtest.ClusterURI())
		integtest.AddTestServerAPIVersion(cliOptions)

		client, err := mongo.Connect(cliOptions)
		assert.NoError(mt, err, "failed to connect to server")

		// Create a GridFS bucket that wraps the db database.
		bucket := client.Database("db").GridFSBucket()

		mt.Run("UploadFromStream", func(mt *mtest.T) {
			// If the operation-level context is not respected, then the client-level
			// timeout will exceed deadline.
			ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
			defer cancel()

			// Upload file and ensure it uploaded correctly.
			fileID, err := bucket.UploadFromStream(ctx, "filename", bytes.NewReader([]byte{0x12}))
			require.NoError(mt, err, "failed to upload stream")

			buf := bytes.Buffer{}

			_, err = bucket.DownloadToStream(context.Background(), fileID, &buf)
			assert.NoError(mt, err, "failed to download stream")
			assert.Equal(mt, buf.Len(), 1)
			assert.Equal(mt, buf.Bytes(), []byte{0x12})
		})

		mt.Run("OpenUploadStream", func(mt *mtest.T) {
			// If the operation-level context is not respected, then the client-level
			// timeout will exceed deadline.
			ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
			defer cancel()

			// Upload file and ensure it uploaded correctly.
			uploadStream, err := bucket.OpenUploadStream(ctx, "filename2")
			require.NoError(mt, err, "failed to open upload stream")

			_, err = uploadStream.Write([]byte{0x13})
			require.NoError(mt, err, "failed to write data to upload stream")

			err = uploadStream.Close()
			require.NoError(mt, err, "failed to close upload stream")

			buf := bytes.Buffer{}

			_, err = bucket.DownloadToStream(context.Background(), uploadStream.FileID, &buf)
			assert.NoError(mt, err, "failed to download stream")
			assert.Equal(mt, buf.Len(), 1)
			assert.Equal(mt, buf.Bytes(), []byte{0x13})
		})
	})

	const test63 = "6.3 gridfs - cancel context mid-stream"
	mt.RunOpts(test63, mtest.NewOptions().MinServerVersion("4.4"), func(mt *mtest.T) {
		// Drop and re-create the db.fs.files and db.fs.chunks collections.
		err := mt.Client.Database("db").Collection("fs.files").Drop(context.Background())
		assert.NoError(mt, err, "failed to drop files")

		err = mt.Client.Database("db").Collection("fs.chunks").Drop(context.Background())
		assert.NoError(mt, err, "failed to drop chunks")

		cliOptions := options.Client().ApplyURI(mtest.ClusterURI())
		integtest.AddTestServerAPIVersion(cliOptions)

		client, err := mongo.Connect(cliOptions)
		assert.NoError(mt, err, "failed to connect to server")

		// Create a GridFS bucket that wraps the db database.
		bucket := client.Database("db").GridFSBucket()

		mt.Run("Upload#Close", func(mt *mtest.T) {
			uploadStream, err := bucket.OpenUploadStream(context.Background(), "filename")
			require.NoError(mt, err)

			_ = uploadStream.Close()

			_, err = uploadStream.Write([]byte{0x13})
			assert.Error(mt, err, context.Canceled)
		})

		mt.Run("Upload#Abort", func(mt *mtest.T) {
			uploadStream, err := bucket.OpenUploadStream(context.Background(), "filename2")
			require.NoError(mt, err)

			_ = uploadStream.Abort()

			_, err = uploadStream.Write([]byte{0x13})
			assert.Error(mt, err, context.Canceled)
		})

		mt.Run("Download#Close", func(mt *mtest.T) {
			fileID, err := bucket.UploadFromStream(context.Background(), "filename3", bytes.NewReader([]byte{0x12}))
			require.NoError(mt, err, "failed to upload stream")

			downloadStream, err := bucket.OpenDownloadStream(context.Background(), fileID)
			assert.NoError(mt, err)

			_ = downloadStream.Close()

			_, err = downloadStream.Read([]byte{})
			assert.Error(mt, err, context.Canceled)
		})
	})
}
