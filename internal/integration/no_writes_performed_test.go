// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/failpoint"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// TestNoWritesPerformedLabel asserts that an operation which fails on its first
// and only attempt with the "NoWritesPerformed" label surfaces the server
// error. The label instructs the driver to return the error from the previous
// attempt, so when there is no previous attempt the current error must be
// returned rather than substituted with a nil error.
func TestNoWritesPerformedLabel(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().CreateClient(false))

	// Sharded topologies are excluded because a failpoint is applied to only a single mongoS.
	mtOpts := mtest.NewOptions().MinServerVersion("4.4").Topologies(mtest.ReplicaSet)

	mt.RunOpts("RunCommand returns the server error on the first attempt", mtOpts, func(mt *mtest.T) {
		const errorCode int32 = 262 // ExceededTimeLimit

		mt.SetFailPoint(failpoint.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: failpoint.Mode{
				Times: 1,
			},
			Data: failpoint.Data{
				FailCommands: []string{"insert"},
				ErrorCode:    errorCode,
				ErrorLabels:  &[]string{"SystemOverloadedError", "NoWritesPerformed"},
			},
		})

		res := mt.DB.RunCommand(context.Background(), bson.D{
			{Key: "insert", Value: mt.Coll.Name()},
			{Key: "documents", Value: bson.A{bson.D{{Key: "x", Value: 1}}}},
		})

		err := res.Err()
		require.Error(mt, err, "expected an error from RunCommand, got nil")

		var cerr mongo.CommandError
		require.True(mt, errors.As(err, &cerr), "expected a mongo.CommandError, got %v", err)
		require.Equal(mt, errorCode, cerr.Code, "expected error code 262")
		require.True(mt, cerr.HasErrorLabel("NoWritesPerformed"),
			"expected the error to have the NoWritesPerformed label")

		// Decode must report the error rather than handing back the server's
		// error-response document as a result.
		var raw bson.Raw
		require.Error(mt, res.Decode(&raw), "expected an error from Decode, got nil and document %v", raw)
	})
}
