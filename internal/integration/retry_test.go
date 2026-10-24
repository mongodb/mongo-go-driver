// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"errors"
	"slices"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/internal/errutil"
	"go.mongodb.org/mongo-driver/v2/internal/failpoint"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// TestRetryErrorHistory asserts that an operation that is retried returns the
// error from every attempt, in chronological order, in addition to the single
// error that retry error selection picks out. See GODRIVER-3600.
func TestRetryErrorHistory(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().CreateClient(false).
		MinServerVersion("6.0").
		Topologies(mtest.ReplicaSet))

	hasErrorCode := func(err error, code int) bool {
		return slices.Contains(mongo.ErrorCodes(err), code)
	}

	// failAndEscalate fails "insert" once with firstCode, then fails it
	// indefinitely with secondCode once the first failure has been observed.
	// The labels are applied to both errors. It returns a client option that
	// installs the monitor driving the escalation.
	failAndEscalate := func(mt *mtest.T, labels []string, firstCode, secondCode int32) *options.ClientOptions {
		mt.SetFailPoint(failpoint.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: failpoint.Mode{
				Times: 1,
			},
			Data: failpoint.Data{
				FailCommands: []string{"insert"},
				ErrorLabels:  &labels,
				ErrorCode:    firstCode,
			},
		})

		monitor := &event.CommandMonitor{
			Failed: func(_ context.Context, evt *event.CommandFailedEvent) {
				if evt.CommandName != "insert" {
					return
				}
				if hasErrorCode(evt.Failure, int(firstCode)) {
					mt.SetFailPoint(failpoint.FailPoint{
						ConfigureFailPoint: "failCommand",
						Mode:               failpoint.ModeAlwaysOn,
						Data: failpoint.Data{
							FailCommands: []string{"insert"},
							ErrorLabels:  &labels,
							ErrorCode:    secondCode,
						},
					})
				}
			},
		}

		return options.Client().SetRetryWrites(true).SetMonitor(monitor)
	}

	// Without the NoWritesPerformed label the returned error is the one from
	// the final attempt, so the history simply ends with it.
	mt.Run("records the error from every attempt", func(mt *mtest.T) {
		labels := []string{"RetryableError", "SystemOverloadedError"}
		mt.ResetClient(failAndEscalate(mt, labels, 91, 10107))

		_, err := mt.Coll.InsertOne(context.Background(), bson.D{})
		require.True(mt, hasErrorCode(err, 10107), "expected the returned error to be the 10107 error")

		var retryErr errutil.RetryError
		require.True(mt, errors.As(err, &retryErr), "expected err to wrap an errutil.RetryError")

		attempts := retryErr.Attempts()
		require.Greater(mt, len(attempts), 1, "expected more than one attempt to be recorded")
		require.True(mt, hasErrorCode(attempts[0], 91),
			"expected the first recorded attempt to be the 91 error, got %v", attempts[0])
		require.True(mt, hasErrorCode(attempts[len(attempts)-1], 10107),
			"expected the last recorded attempt to be the 10107 error, got %v",
			attempts[len(attempts)-1])
	})

	// With the NoWritesPerformed label on every error, the returned error is
	// the first attempt's error rather than the final attempt's. Substituting
	// the returned error must not rewrite the history: the final attempt's own
	// error must still be recorded.
	mt.Run("records the final attempt when the returned error is substituted", func(mt *mtest.T) {
		labels := []string{"RetryableError", "SystemOverloadedError", "NoWritesPerformed"}
		mt.ResetClient(failAndEscalate(mt, labels, 91, 10107))

		_, err := mt.Coll.InsertOne(context.Background(), bson.D{})
		require.True(mt, hasErrorCode(err, 91), "expected the returned error to be the 91 error")

		var retryErr errutil.RetryError
		require.True(mt, errors.As(err, &retryErr), "expected err to wrap an errutil.RetryError")

		attempts := retryErr.Attempts()
		require.Greater(mt, len(attempts), 1, "expected more than one attempt to be recorded")
		require.True(mt, hasErrorCode(attempts[0], 91),
			"expected the first recorded attempt to be the 91 error, got %v", attempts[0])
		require.True(mt, hasErrorCode(attempts[len(attempts)-1], 10107),
			"expected the last recorded attempt to be the 10107 error, got %v",
			attempts[len(attempts)-1])
	})
}
