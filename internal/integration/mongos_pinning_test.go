// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/mongoutil"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestMongosPinning(t *testing.T) {
	clientOpts := options.Client().SetLocalThreshold(1 * time.Second).SetWriteConcern(mtest.MajorityWc)
	mtOpts := mtest.NewOptions().Topologies(mtest.Sharded).MinServerVersion("4.1").CreateClient(false).
		ClientOptions(clientOpts)
	mt := mtest.New(t, mtOpts)

	hosts, err := mongoutil.HostsFromURI(mtest.ClusterURI())
	require.NoError(t, err)

	if len(hosts) < 2 {
		mt.Skip("skipping because at least 2 mongoses are required")
	}

	mt.Run("unpin for next transaction", func(mt *mtest.T) {
		addresses := map[string]struct{}{}
		_ = mt.Client.UseSession(context.Background(), func(sctx context.Context) error {
			sess := mongo.SessionFromContext(sctx)
			// Insert a document in a transaction to pin session to a mongos
			err := sess.StartTransaction()
			assert.Nil(mt, err, "StartTransaction error: %v", err)
			_, err = mt.Coll.InsertOne(sctx, bson.D{{"x", 1}})
			assert.Nil(mt, err, "InsertOne error: %v", err)
			err = sess.CommitTransaction(sctx)
			assert.Nil(mt, err, "CommitTransaction error: %v", err)

			for i := 0; i < 50; i++ {
				// Call Find in a new transaction to unpin from the old mongos and select a new one
				err = sess.StartTransaction()
				assert.Nil(mt, err, iterationErrmsg("StartTransaction", i, err))

				cursor, err := mt.Coll.Find(sctx, bson.D{})
				assert.Nil(mt, err, iterationErrmsg("Find", i, err))
				assert.True(mt, cursor.Next(context.Background()), "Next returned false on iteration %v", i)

				descConn, err := mongo.BatchCursorFromCursor(cursor).Server().Connection(context.Background())
				assert.Nil(mt, err, iterationErrmsg("Connection", i, err))
				addresses[descConn.Description().Addr.String()] = struct{}{}
				err = descConn.Close()
				assert.Nil(mt, err, iterationErrmsg("connection Close", i, err))

				err = sess.CommitTransaction(sctx)
				assert.Nil(mt, err, iterationErrmsg("CommitTransaction", i, err))
			}
			return nil
		})
		assert.True(mt, len(addresses) > 1, "expected more than 1 address, got %v", addresses)
	})
	mt.Run("unpin for non transaction operation", func(mt *mtest.T) {
		addresses := map[string]struct{}{}
		_ = mt.Client.UseSession(context.Background(), func(sctx context.Context) error {
			sess := mongo.SessionFromContext(sctx)

			// Insert a document in a transaction to pin session to a mongos
			err := sess.StartTransaction()
			assert.Nil(mt, err, "StartTransaction error: %v", err)
			_, err = mt.Coll.InsertOne(sctx, bson.D{{"x", 1}})
			assert.Nil(mt, err, "InsertOne error: %v", err)
			err = sess.CommitTransaction(sctx)
			assert.Nil(mt, err, "CommitTransaction error: %v", err)

			for i := 0; i < 50; i++ {
				// Call Find with the session but outside of a transaction
				cursor, err := mt.Coll.Find(sctx, bson.D{})
				assert.Nil(mt, err, iterationErrmsg("Find", i, err))
				assert.True(mt, cursor.Next(context.Background()), "Next returned false on iteration %v", i)

				descConn, err := mongo.BatchCursorFromCursor(cursor).Server().Connection(context.Background())
				assert.Nil(mt, err, iterationErrmsg("Connection", i, err))
				addresses[descConn.Description().Addr.String()] = struct{}{}
				err = descConn.Close()
				assert.Nil(mt, err, iterationErrmsg("connection Close", i, err))
			}
			return nil
		})
		assert.True(mt, len(addresses) > 1, "expected more than 1 address, got %v", addresses)
	})
}

func iterationErrmsg(op string, i int, wrapped error) string {
	return fmt.Sprintf("%v error on iteration %v: %v", op, i, wrapped)
}
