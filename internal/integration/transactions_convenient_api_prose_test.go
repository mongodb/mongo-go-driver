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

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/failpoint"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/randutil"
	"go.mongodb.org/mongo-driver/v2/internal/require"
)

func TestTransactionConvenientApiProse(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().MinServerVersion("4.4").ClientType(mtest.Pinned).
		AllowFailPointsOnSharded().Topologies(mtest.ReplicaSet, mtest.Sharded))
	mt.Run("retry backoff is enforced", func(mt *mtest.T) {
		fp := failpoint.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: failpoint.Mode{
				Times: 13,
			},
			Data: failpoint.Data{
				FailCommands: []string{"commitTransaction"},
				ErrorCode:    251,
			},
		}

		transWithJitter := func(t *mtest.T, ratio float64) time.Duration {
			defer randutil.SetJitterForTesting(func(n int64) int64 {
				val := int64(ratio * float64(n))
				if val < 0 {
					return 0
				}
				if val > n {
					return n
				}
				return val
			})()

			mt.SetFailPoint(fp)

			session, err := mt.Client.StartSession()
			require.NoError(t, err, "StartSession error: %v", err)
			defer session.EndSession(context.Background())

			startTime := time.Now()
			_, err = session.WithTransaction(context.Background(), func(sesctx context.Context) (any, error) {
				if _, err := mt.Coll.InsertOne(sesctx, bson.D{}); err != nil {
					return nil, err
				}
				return nil, nil
			})
			require.NoError(t, err, "WithTransaction error: %v", err)
			return time.Since(startTime)
		}
		noBackoffTime := transWithJitter(mt, 0)
		withBackoffTime := transWithJitter(mt, 1)
		assert.InDelta(
			t,
			withBackoffTime, noBackoffTime+1_800*time.Millisecond, float64(500*time.Millisecond),
			"with backoff time: %v, no backoff time: %v", withBackoffTime, noBackoffTime,
		)
	})
}
