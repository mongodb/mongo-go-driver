// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestSDAMErrorHandling(t *testing.T) {
	mt := mtest.New(t, noClientOpts)
	clientOpts := options.Client().
		ApplyURI(mt.ConnString()).
		SetRetryWrites(false).
		SetPoolMonitor(poolMonitor).
		SetWriteConcern(mtest.MajorityWc)
	mtOpts := mtest.NewOptions().
		Topologies(mtest.ReplicaSet). // Don't run on sharded clusters to avoid complexity of sharded failpoints.
		MinServerVersion("4.0").      // 4.0+ is required to use failpoints on replica sets.
		ClientOptions(clientOpts)

	mt.RunOpts("network errors", mtOpts, func(mt *mtest.T) {
		mt.Run("pool cleared on non-timeout network error", func(mt *mtest.T) {
			clearPoolChan()
			mt.SetFailPoint(mtest.FailPoint{
				ConfigureFailPoint: "failCommand",
				Mode: mtest.FailPointMode{
					Times: 1,
				},
				Data: mtest.FailPointData{
					FailCommands:    []string{"insert"},
					CloseConnection: true,
				},
			})

			_, err := mt.Coll.InsertOne(mtest.Background, bson.D{{"test", 1}})
			assert.NotNil(mt, err, "expected InsertOne error, got nil")
			assert.True(mt, isPoolCleared(), "expected pool to be cleared but was not")
		})
		mt.Run("pool not cleared on timeout network error", func(mt *mtest.T) {
			clearPoolChan()

			_, err := mt.Coll.InsertOne(mtest.Background, bson.D{{"x", 1}})
			assert.Nil(mt, err, "InsertOne error: %v", err)

			filter := bson.M{
				"$where": "function() { sleep(1000); return false; }",
			}
			timeoutCtx, cancel := context.WithTimeout(mtest.Background, 100*time.Millisecond)
			defer cancel()
			_, err = mt.Coll.Find(timeoutCtx, filter)
			assert.NotNil(mt, err, "expected Find error, got %v", err)

			assert.False(mt, isPoolCleared(), "expected pool to not be cleared but was")
		})
	})
}
