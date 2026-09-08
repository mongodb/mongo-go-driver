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
			defer randutil.SetJitterForTesting(func() float64 { return ratio })()

			startTime := time.Now()
			_, err := t.Coll.InsertOne(context.Background(), bson.D{{"a", 1}})
			assert.IsTypef(t, mongo.CommandError{}, err, "expected a CommandError, got: %T", err)
			return time.Since(startTime)
		}
		noBackoffTime := transWithJitter(mt, 0)
		withBackoffTime := transWithJitter(mt, 1)
		assert.InDelta(
			mt,
			withBackoffTime, noBackoffTime+600*time.Millisecond, float64(600*time.Millisecond),
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
		assert.Equalf(mt, 3, opsCnt, "expected 3 attempts (1 original + 2 retries), got %d", opsCnt)
	})
	mt.Run("4. Overload Errors are Retried a Maximum of maxAdaptiveRetries times when configured", func(mt *mtest.T) {
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
		mt.ResetClient(options.Client().SetMonitor(monitor).SetMaxAdaptiveRetries(1))

		_, err := mt.Coll.Find(context.Background(), bson.D{})
		var cmdErr mongo.CommandError
		require.Truef(mt, errors.As(err, &cmdErr), "expected a CommandError, got %T: %v", err, err)
		assert.True(mt, cmdErr.HasErrorLabel("RetryableError"), `expected error has "RetryableError" label`)
		assert.True(mt, cmdErr.HasErrorLabel("SystemOverloadedError"), `expected error has "SystemOverloadedError" label`)
		assert.Equalf(mt, 2, opsCnt, "expected 2 attempts (1 original + 1 retry), got %d", opsCnt)
	})
	mt.RunOpts("5. Overload Errors with baseBackoffMS override base backoff", mtest.NewOptions().MinServerVersion("9.0"), func(mt *mtest.T) {
		mt.SetFailPoint(failpoint.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode:               failpoint.ModeAlwaysOn,
			Data: failpoint.Data{
				FailCommands: []string{"insert"},
				ErrorCode:    462,
				ErrorLabels:  &[]string{"SystemOverloadedError", "RetryableError"},
			},
		})

		mt.ResetClient(options.Client())

		setExternalClientBaseBackoffMS := func(ms int) {
			err := mt.Client.Database("admin").RunCommand(context.Background(), bson.D{
				{"setParameter", 1},
				{"externalClientBaseBackoffMS", ms},
			}).Err()
			require.NoError(mt, err, "setParameter externalClientBaseBackoffMS=%d error: %v", ms, err)
		}

		insertWithJitter := func() (time.Duration, mongo.CommandError) {
			defer randutil.SetJitterForTesting(func() float64 { return 1 })()

			startTime := time.Now()
			_, err := mt.Coll.InsertOne(context.Background(), bson.D{{"a", 1}})
			duration := time.Since(startTime)

			var cmdErr mongo.CommandError
			require.Truef(mt, errors.As(err, &cmdErr), "expected a CommandError, got %T: %v", err, err)
			return duration, cmdErr
		}

		exponentialTime, exponentialErr := insertWithJitter()
		_, err := exponentialErr.Raw.LookupErr("baseBackoffMS")
		require.Error(mt, err, "expected no baseBackoffMS on the error before setting externalClientBaseBackoffMS")

		setExternalClientBaseBackoffMS(50)
		defer setExternalClientBaseBackoffMS(0)

		baseBackoffTime, baseBackoffErr := insertWithJitter()
		baseBackoffMS, err := baseBackoffErr.Raw.LookupErr("baseBackoffMS")
		require.NoError(mt, err, "expected the server to attach baseBackoffMS to the error")
		baseBackoffMSVal, ok := baseBackoffMS.AsInt64OK()
		require.True(mt, ok, "expected baseBackoffMS to be numeric, got %v", baseBackoffMS.Type)
		require.Equal(mt, int64(50), baseBackoffMSVal, "expected baseBackoffMS to be 50")

		assert.GreaterOrEqual(mt, exponentialTime, 600*time.Millisecond,
			"expected the default backoff run to take at least 600ms, took %v", exponentialTime)
		assert.GreaterOrEqual(mt, baseBackoffTime, 300*time.Millisecond,
			"expected the baseBackoffMS run to take at least 300ms, took %v", baseBackoffTime)
		assert.Less(mt, baseBackoffTime, 600*time.Millisecond,
			"expected the baseBackoffMS run to take less than 600ms, took %v", baseBackoffTime)
	})
}
