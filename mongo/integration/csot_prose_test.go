// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/internal/testutil/helpers"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestCSOTProse(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().CreateClient(false))
	defer mt.Close()

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
		testutil.AddTestServerAPIVersion(cliOptions)
		cli, err := mongo.Connect(context.Background(), cliOptions)
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
	mt.Run("8. server selection", func(mt *mtest.T) {
		cliOpts := options.Client().ApplyURI("mongodb://invalid/?serverSelectionTimeoutMS=100")
		mtOpts := mtest.NewOptions().ClientOptions(cliOpts).CreateCollection(false)
		mt.RunOpts("serverSelectionTimeoutMS honored if timeoutMS is not set", mtOpts, func(mt *mtest.T) {
			mt.Parallel()

			callback := func(ctx context.Context) {
				err := mt.Client.Ping(ctx, nil)
				assert.NotNil(mt, err, "expected Ping error, got nil")
			}

			// Assert that Ping fails within 150ms due to server selection timeout.
			helpers.AssertSoon(mt, callback, 150*time.Millisecond)
		})

		cliOpts = options.Client().ApplyURI("mongodb://invalid/?timeoutMS=100&serverSelectionTimeoutMS=200")
		mtOpts = mtest.NewOptions().ClientOptions(cliOpts).CreateCollection(false)
		mt.RunOpts("timeoutMS honored for server selection if it's lower than serverSelectionTimeoutMS", mtOpts, func(mt *mtest.T) {
			mt.Parallel()

			callback := func(ctx context.Context) {
				err := mt.Client.Ping(ctx, nil)
				assert.NotNil(mt, err, "expected Ping error, got nil")
			}

			// Assert that Ping fails within 150ms due to timeout.
			helpers.AssertSoon(mt, callback, 150*time.Millisecond)
		})

		cliOpts = options.Client().ApplyURI("mongodb://invalid/?timeoutMS=200&serverSelectionTimeoutMS=100")
		mtOpts = mtest.NewOptions().ClientOptions(cliOpts).CreateCollection(false)
		mt.RunOpts("serverSelectionTimeoutMS honored for server selection if it's lower than timeoutMS", mtOpts, func(mt *mtest.T) {
			mt.Parallel()

			callback := func(ctx context.Context) {
				err := mt.Client.Ping(ctx, nil)
				assert.NotNil(mt, err, "expected Ping error, got nil")
			}

			// Assert that Ping fails within 150ms due to server selection timeout.
			helpers.AssertSoon(mt, callback, 150*time.Millisecond)
		})

		cliOpts = options.Client().ApplyURI("mongodb://invalid/?timeoutMS=0&serverSelectionTimeoutMS=100")
		mtOpts = mtest.NewOptions().ClientOptions(cliOpts).CreateCollection(false)
		mt.RunOpts("serverSelectionTimeoutMS honored for server selection if timeoutMS=0", mtOpts, func(mt *mtest.T) {
			mt.Parallel()

			callback := func(ctx context.Context) {
				err := mt.Client.Ping(ctx, nil)
				assert.NotNil(mt, err, "expected Ping error, got nil")
			}

			// Assert that Ping fails within 150ms due to server selection timeout.
			helpers.AssertSoon(mt, callback, 150*time.Millisecond)
		})
	})
}
