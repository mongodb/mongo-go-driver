// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"math/rand"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestCMAPProse_PendingResponse(t *testing.T) {
	const timeout = 200 * time.Millisecond

	// Skip on compressor due to proxy complexity.
	if os.Getenv("MONGO_GO_DRIVER_COMPRESSOR") != "" {
		t.Skip("Skipping test because MONGO_GO_DRIVER_COMPRESSOR is set, which is not supported by the proxy")
	}

	// Skip on auth due to proxy complexity.
	if os.Getenv("AUTH") == "auth" {
		t.Skip("Skipping test because AUTH is set")
	}

	// Skip on SSL due to proxy complexity.
	if os.Getenv("SSL") == "ssl" {
		t.Skip("Skipping test because SSL is set")
	}

	// Skip unless the proxy URI is set.
	if os.Getenv("MONGO_PROXY_URI") == "" {
		t.Skip("Skipping test because MONGO_PROXY_URI is not set")
	}

	// Create a direct connection to the proxy server.
	proxyURI := os.Getenv("MONGO_PROXY_URI")

	clientOpts := options.Client().ApplyURI(proxyURI).SetMaxPoolSize(1).SetDirect(true)
	mt := mtest.New(t, mtest.NewOptions().ClientOptions(clientOpts).ClientType(mtest.MongoProxy))

	opts := mtest.NewOptions().CreateCollection(false)

	// Ensure that if part of the header is received before a socket timeout,
	// it does not cause a problem when the pool attempts to drain the connection.
	// for the next operation.
	//
	// Path where the size has been determined while draining the pending
	// response.
	mt.RunOpts("recover partial header response", opts, func(mt *mtest.T) {
		// Chose a random number between 1 and 3 to maximize coverage.
		bytesToSend := rand.Intn(3) + 1

		// The proxy should deliver 3 bytes from the header and then pause to cause
		// a socket timeout.
		proxyTest := bson.D{
			{Key: "actions", Value: bson.A{
				bson.D{{Key: "sendBytes", Value: bytesToSend}},
				// Causes the timeout in the initial try.
				bson.D{{Key: "delayMs", Value: 400}},
				// Send the rest of the response for discarding on retry.
				bson.D{{Key: "sendAll", Value: true}},
			}},
		}

		type myStruct struct {
			Name string `bson:"name"`
			Age  int    `bson:"age"`
		}

		cmd := bson.D{
			{Key: "insert", Value: "mycoll"},
			{Key: "documents", Value: bson.A{myStruct{Name: "Alice", Age: 30}}},
			{Key: "proxyTest", Value: proxyTest},
		}

		db := mt.Client.Database("testdb")
		coll := db.Collection("mycoll")

		// Run the command against the proxy with deadline.
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		err := db.RunCommand(ctx, cmd).Err()

		// Expect the command to fail due to the timeout.
		require.Error(mt, err, "expected command to fail due to timeout")
		assert.ErrorIs(mt, err, context.DeadlineExceeded)

		// Run an insertOne without a timeout.
		_, err = coll.InsertOne(context.Background(), myStruct{Name: "Bob", Age: 25})
		require.NoError(mt, err)

		// There should be 1 ConnectionPendingResponseStarted event.
		assert.Equal(mt, 1, mt.NumberConnectionsPendingReadStarted())

		// There should be 0 ConnectionPendingResponseFailed event.
		assert.Equal(mt, 0, mt.NumberConnectionsPendingReadFailed())

		// There should be 1 ConnectionPendingResponseSucceeded event.
		assert.Equal(mt, 1, mt.NumberConnectionsPendingReadSucceeded())

		// The connection should not have been closed.
		assert.Equal(mt, 0, mt.NumberConnectionsClosed())
	})

	// Unified spec tests block the entire response, so we need to ensure that if
	// we send part of the response before a socket timeout, the connection
	// will successfully drain the connection for a subsequent operation.
	//
	// Path where the size has been determined during the round trip.
	mt.RunOpts("recover partial response", opts, func(mt *mtest.T) {
		// The proxy should deliver 3 bytes from the header and then pause to cause
		// a socket timeout.
		proxyTest := bson.D{
			{Key: "actions", Value: bson.A{
				bson.D{{Key: "sendBytes", Value: 10}},
				// Causes the timeout in the initial try.
				bson.D{{Key: "delayMs", Value: 400}},
				// Send the rest of the response for discarding on retry.
				bson.D{{Key: "sendAll", Value: true}},
			}},
		}

		type myStruct struct {
			Name string `bson:"name"`
			Age  int    `bson:"age"`
		}

		cmd := bson.D{
			{Key: "insert", Value: "mycoll"},
			{Key: "documents", Value: bson.A{myStruct{Name: "Alice", Age: 30}}},
			{Key: "proxyTest", Value: proxyTest},
		}

		db := mt.Client.Database("testdb")
		coll := db.Collection("mycoll")

		// Run the command against the proxy with deadline.
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		err := db.RunCommand(ctx, cmd).Err()

		// Expect the command to fail due to the timeout.
		require.Error(mt, err, "expected command to fail due to timeout")
		assert.ErrorIs(mt, err, context.DeadlineExceeded)

		// Run an insertOne without a timeout.
		_, err = coll.InsertOne(context.Background(), myStruct{Name: "Bob", Age: 25})
		require.NoError(mt, err)

		// There should be 1 ConnectionPendingResponseStarted event.
		assert.Equal(mt, 1, mt.NumberConnectionsPendingReadStarted())

		// There should be 0 ConnectionPendingResponseFailed event.
		assert.Equal(mt, 0, mt.NumberConnectionsPendingReadFailed())

		// There should be 1 ConnectionPendingResponseSucceeded event.
		assert.Equal(mt, 1, mt.NumberConnectionsPendingReadSucceeded())

		// The connection should not have been closed.
		assert.Equal(mt, 0, mt.NumberConnectionsClosed())
	})

	// Ensure that if part of the header is received before a socket timeout, and
	// the connection idles in the pool for longer than 3 seconds, the required
	// aliveness check for draining the connection does not attempt to discard
	// bytes from the TCP stream.
	//
	// Path where an aliveness check is performed and does not pull data from the
	// TCP stream.
	mt.RunOpts("non-destructive aliveness check", opts, func(mt *mtest.T) {
		rand.Seed(time.Now().UnixNano())

		// Chose a random number between 1 and 3 to maximize coverage.
		bytesToSend := rand.Intn(3) + 1

		// The proxy should deliver 3 bytes from the header and then pause to cause
		// a socket timeout.
		proxyTest := bson.D{
			{Key: "actions", Value: bson.A{
				bson.D{{Key: "sendBytes", Value: bytesToSend}},
				// Causes the timeout in the initial try.
				bson.D{{Key: "delayMs", Value: 400}},
				// Send the rest of the response for discarding on retry.
				bson.D{{Key: "sendAll", Value: true}},
			}},
		}

		type myStruct struct {
			Name string `bson:"name"`
			Age  int    `bson:"age"`
		}

		cmd := bson.D{
			{Key: "insert", Value: "mycoll"},
			{Key: "documents", Value: bson.A{myStruct{Name: "Alice", Age: 30}}},
			{Key: "proxyTest", Value: proxyTest},
		}

		db := mt.Client.Database("testdb")
		coll := db.Collection("mycoll")

		// Run the command against the proxy with deadline.
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		err := db.RunCommand(ctx, cmd).Err()

		// Expect the command to fail due to the timeout.
		require.Error(mt, err, "expected command to fail due to timeout")
		assert.ErrorIs(mt, err, context.DeadlineExceeded)

		// Wait for 3 seconds to ensure the connection idles longer than the
		// window for draining the connection.
		time.Sleep(3 * time.Second)

		// Run an insertOne without a timeout.
		_, err = coll.InsertOne(context.Background(), myStruct{Name: "Bob", Age: 25})
		require.NoError(mt, err)

		// There should be 1 ConnectionPendingResponseStarted event.
		//   - One for the aliveness check
		//   - One for the subsequent retry
		assert.Equal(mt, 2, mt.NumberConnectionsPendingReadStarted())

		// There should be 1 ConnectionPendingResponseFailed event from the
		// aliveness check which should propagate a retryable error.
		assert.Equal(mt, 1, mt.NumberConnectionsPendingReadFailed())

		// There should be 1 ConnectionPendingResponseSucceeded event.
		assert.Equal(mt, 1, mt.NumberConnectionsPendingReadSucceeded())

		// The connection should not have been closed.
		assert.Equal(mt, 0, mt.NumberConnectionsClosed())
	})
}
