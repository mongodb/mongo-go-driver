// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestLoadBalancerSupport(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().Topologies(mtest.LoadBalanced).CreateClient(false))

	mt.Run("RunCommandCursor pins to a connection", func(mt *mtest.T) {
		// The LB spec tests cover the behavior for cursors created by CRUD operations, but RunCommandCursor is
		// Go-specific so there is no spec test coverage for it.

		initCollection(mt, mt.Coll)
		findCmd := bson.D{
			{"find", mt.Coll.Name()},
			{"filter", bson.D{}},
			{"batchSize", 2},
		}
		cursor, err := mt.DB.RunCommandCursor(context.Background(), findCmd)
		assert.Nil(mt, err, "RunCommandCursor error: %v", err)
		defer func() {
			_ = cursor.Close(context.Background())
		}()

		assert.True(mt, cursor.ID() > 0, "expected cursor ID to be non-zero")
		assert.Equal(mt, 1, mt.NumberConnectionsCheckedOut(),
			"expected one connection to be checked out, got %d", mt.NumberConnectionsCheckedOut())
	})

	mt.RunOpts("wait queue timeout errors include extra information", noClientOpts, func(mt *mtest.T) {
		// There are spec tests to assert this behavior, but they rely on the waitQueueTimeoutMS Client option, which is
		// not supported in Go, so we have to skip them. These prose tests make the same assertions, but use context
		// deadlines to force wait queue timeout errors.

		assertErrorHasInfo := func(mt *mtest.T, err error, numCursorConns, numTxnConns, numOtherConns int) {
			mt.Helper()

			assert.NotNil(mt, err, "expected wait queue timeout error, got nil")
			expectedMsg := fmt.Sprintf("maxPoolSize: 1, "+
				"connections in use by cursors: %d, "+
				"connections in use by transactions: %d, "+
				"connections in use by other operations: %d",
				numCursorConns, numTxnConns, numOtherConns,
			)
			assert.True(mt, strings.Contains(err.Error(), expectedMsg),
				"expected error %q to contain substring %q", err, expectedMsg)
		}
		maxPoolSizeMtOpts := mtest.NewOptions().
			ClientOptions(options.Client().SetMaxPoolSize(1))

		mt.RunOpts("cursors", maxPoolSizeMtOpts, func(mt *mtest.T) {
			initCollection(mt, mt.Coll)
			findOpts := options.Find().SetBatchSize(2)
			cursor, err := mt.Coll.Find(context.Background(), bson.M{}, findOpts)
			assert.Nil(mt, err, "Find error: %v", err)
			defer func() {
				_ = cursor.Close(context.Background())
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
			defer cancel()
			_, err = mt.Coll.InsertOne(ctx, bson.M{"x": 1})
			assertErrorHasInfo(mt, err, 1, 0, 0)
		})
		mt.RunOpts("transactions", maxPoolSizeMtOpts, func(mt *mtest.T) {
			{
				sess, err := mt.Client.StartSession()
				assert.Nil(mt, err, "StartSession error: %v", err)
				defer sess.EndSession(context.Background())
				ctx := mongo.NewSessionContext(context.Background(), sess)

				// Start a transaction and perform one transactional operation to pin a connection.
				err = sess.StartTransaction()
				assert.Nil(mt, err, "StartTransaction error: %v", err)
				_, err = mt.Coll.InsertOne(ctx, bson.M{"x": 1})
				assert.Nil(mt, err, "InsertOne error: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
			defer cancel()
			_, err := mt.Coll.InsertOne(ctx, bson.M{"x": 1})
			assertErrorHasInfo(mt, err, 0, 1, 0)
		})

		// GODRIVER-2867: Test that connections are unpinned from transactions
		// when the transaction session is ended. Create a Client with
		// maxPoolSize=1 and expect that it can start and commit 5 transactions
		// with that 1 connection.
		mt.RunOpts("transaction connections are unpinned", maxPoolSizeMtOpts, func(mt *mtest.T) {
			{
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
				defer cancel()

				for i := 0; i < 5; i++ {
					sess, err := mt.Client.StartSession()
					require.NoError(mt, err, "StartSession error")

					err = sess.StartTransaction()
					require.NoError(mt, err, "StartTransaction error")

					ctx := mongo.NewSessionContext(ctx, sess)
					_, err = mt.Coll.InsertOne(ctx, bson.M{"x": 1})
					assert.NoError(mt, err, "InsertOne error")

					err = sess.CommitTransaction(ctx)
					assert.NoError(mt, err, "CommitTransaction error")

					sess.EndSession(ctx)
				}
			}
		})
	})
}
