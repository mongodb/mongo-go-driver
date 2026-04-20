// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/failpoint"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/randutil"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestBackpressureProse(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().MinServerVersion("4.4").ClientType(mtest.Pinned).
		CreateClient(false).AllowFailPointsOnSharded())
	mt.Run("1. Operation Retry Uses Exponential Backoff", func(mt *mtest.T) {
		mt.SetFailPoint(failpoint.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode:               failpoint.ModeAlwaysOn,
			Data: failpoint.Data{
				FailCommands: []string{"insert"},
				ErrorCode:    2,
				ErrorLabels:  &[]string{"SystemOverloadedError", "RetryableError"},
			},
		})

		mt.ResetClient(options.Client())

		transWithJitter := func(t *mtest.T, ratio float64) time.Duration {
			defer randutil.SetJitterForTesting(ratio)()

			startTime := time.Now()
			_, err := t.Coll.InsertOne(context.Background(), bson.D{{"a", 1}})
			assert.IsTypef(t, mongo.CommandError{}, err, "expected a CommandError, got: %T", err)
			return time.Since(startTime)
		}
		noBackoffTime := transWithJitter(mt, 0)
		withBackoffTime := transWithJitter(mt, 1)
		assert.GreaterOrEqualf(
			mt,
			withBackoffTime, noBackoffTime+2_100*time.Millisecond,
			"with backoff time: %v, no backoff time: %v", withBackoffTime, noBackoffTime,
		)
	})
	mt.Run("3. Overload Errors are Retried a Maximum of MAX_RETRIES times", func(mt *mtest.T) {
		mt.SetFailPoint(failpoint.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode:               failpoint.ModeAlwaysOn,
			Data: failpoint.Data{
				FailCommands: []string{"find"},
				ErrorCode:    462,
				ErrorLabels:  &[]string{"SystemOverloadedError", "RetryableError"},
			},
		})

		var opsCnt int
		monitor := &event.CommandMonitor{
			Started: func(_ context.Context, e *event.CommandStartedEvent) {
				if e.CommandName == "find" {
					opsCnt++
				}
			},
		}
		mt.ResetClient(options.Client().SetMonitor(monitor))

		_, err := mt.Coll.Find(context.Background(), bson.D{})
		var cmdErr mongo.CommandError
		require.Truef(mt, errors.As(err, &cmdErr), "expected a CommandError, got %T: %v", err, err)
		assert.True(mt, cmdErr.HasErrorLabel("RetryableError"), `expected error has "RetryableError" label`)
		assert.True(mt, cmdErr.HasErrorLabel("SystemOverloadedError"), `expected error has "SystemOverloadedError" label`)
		assert.Equalf(mt, 6, opsCnt, "expected 6 attempts (1 original + 5 retries), got %d", opsCnt)
	})
}
