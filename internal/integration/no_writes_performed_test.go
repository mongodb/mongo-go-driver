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

	const errorCode int32 = 262 // ExceededTimeLimit

	testCases := []struct {
		name        string
		failCommand string
		setup       func(mt *mtest.T) // nil when the test needs no seed data
		execute     func(mt *mtest.T) *mongo.SingleResult
	}{
		{
			name:        "RunCommand returns the server error on the first attempt",
			failCommand: "insert",
			execute: func(mt *mtest.T) *mongo.SingleResult {
				return mt.DB.RunCommand(context.Background(), bson.D{
					{Key: "insert", Value: mt.Coll.Name()},
					{Key: "documents", Value: bson.A{bson.D{{Key: "x", Value: 1}}}},
				})
			},
		},
		{
			// GODRIVER-4098: findAndModify reports its result in the "value" field. A discarded
			// error leaves the SingleResult reading that field from the server's error response,
			// where it does not exist, so a matching document is reported as no document at all.
			name:        "FindOneAndUpdate returns the server error on the first attempt",
			failCommand: "findAndModify",
			setup: func(mt *mtest.T) {
				_, err := mt.Coll.InsertOne(context.Background(), bson.D{{Key: "x", Value: 1}})
				require.NoError(mt, err, "InsertOne error: %v", err)
			},
			execute: func(mt *mtest.T) *mongo.SingleResult {
				return mt.Coll.FindOneAndUpdate(context.Background(),
					bson.D{{Key: "x", Value: 1}},
					bson.D{{Key: "$set", Value: bson.D{{Key: "y", Value: 2}}}})
			},
		},
	}

	for _, tc := range testCases {
		mt.RunOpts(tc.name, mtOpts, func(mt *mtest.T) {
			if tc.setup != nil {
				tc.setup(mt)
			}

			mt.SetFailPoint(failpoint.FailPoint{
				ConfigureFailPoint: "failCommand",
				Mode: failpoint.Mode{
					Times: 1,
				},
				Data: failpoint.Data{
					FailCommands: []string{tc.failCommand},
					ErrorCode:    errorCode,
					ErrorLabels:  &[]string{"SystemOverloadedError", "NoWritesPerformed"},
				},
			})

			res := tc.execute(mt)

			err := res.Err()
			require.Error(mt, err, "expected an error from %s, got nil", tc.failCommand)
			require.False(mt, errors.Is(err, mongo.ErrNoDocuments),
				"expected the server error, got ErrNoDocuments")

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
}
