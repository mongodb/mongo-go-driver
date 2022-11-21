// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/internal/testutil/monitor"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	errorNotPrimary            int32 = 10107
	errorShutdownInProgress    int32 = 91
	errorInterruptedAtShutdown int32 = 11600
)

func TestConnectionsSurvivePrimaryStepDown(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().Topologies(mtest.ReplicaSet).CreateClient(false))
	defer mt.Close()

	getMoreOpts := mtest.NewOptions().MinServerVersion("4.2")
	mt.RunOpts("getMore iteration", getMoreOpts, func(mt *mtest.T) {
		tpm := monitor.NewTestPoolMonitor()
		mt.ResetClient(options.Client().
			SetRetryWrites(false).
			SetPoolMonitor(tpm.PoolMonitor))

		initCollection(mt, mt.Coll)
		cur, err := mt.Coll.Find(context.Background(), bson.D{}, options.Find().SetBatchSize(2))
		assert.Nil(mt, err, "Find error: %v", err)
		defer cur.Close(context.Background())
		assert.True(mt, cur.Next(context.Background()), "expected Next true, got false")

		// replSetStepDown can fail with transient errors, so we use executeAdminCommandWithRetry to handle them and
		// retry until a timeout is hit.
		stepDownCmd := bson.D{
			{"replSetStepDown", 5},
			{"force", true},
		}
		stepDownOpts := options.RunCmd().SetReadPreference(mtest.PrimaryRp)
		executeAdminCommandWithRetry(mt, mt.Client, stepDownCmd, stepDownOpts)

		assert.True(mt, cur.Next(context.Background()), "expected Next true, got false")
		assert.False(mt, tpm.IsPoolCleared(), "expected pool to not be cleared but was")
	})
	mt.RunOpts("server errors", noClientOpts, func(mt *mtest.T) {
		testCases := []struct {
			name                   string
			minVersion, maxVersion string
			errCode                int32
			poolCleared            bool
		}{
			{"notPrimary keep pool", "4.2", "", errorNotPrimary, false},
			{"notPrimary reset pool", "4.0", "4.0", errorNotPrimary, true},
			{"shutdown in progress reset pool", "4.0", "", errorShutdownInProgress, true},
			{"interrupted at shutdown reset pool", "4.0", "", errorInterruptedAtShutdown, true},
		}
		for _, tc := range testCases {
			opts := mtest.NewOptions()
			if tc.minVersion != "" {
				opts.MinServerVersion(tc.minVersion)
			}
			if tc.maxVersion != "" {
				opts.MaxServerVersion(tc.maxVersion)
			}
			mt.RunOpts(tc.name, opts, func(mt *mtest.T) {
				tpm := monitor.NewTestPoolMonitor()
				mt.ResetClient(options.Client().
					SetRetryWrites(false).
					SetPoolMonitor(tpm.PoolMonitor).
					// Use a low heartbeat frequency so the Client will quickly recover when using
					// failpoints that cause SDAM state changes.
					SetHeartbeatInterval(defaultHeartbeatInterval))

				mt.SetFailPoint(mtest.FailPoint{
					ConfigureFailPoint: "failCommand",
					Mode: mtest.FailPointMode{
						Times: 1,
					},
					Data: mtest.FailPointData{
						FailCommands: []string{"insert"},
						ErrorCode:    tc.errCode,
					},
				})

				_, err := mt.Coll.InsertOne(context.Background(), bson.D{{"test", 1}})
				assert.NotNil(mt, err, "expected InsertOne error, got nil")
				cerr, ok := err.(mongo.CommandError)
				assert.True(mt, ok, "expected error type %v, got %v", mongo.CommandError{}, err)
				assert.Equal(mt, tc.errCode, cerr.Code, "expected error code %v, got %v", tc.errCode, cerr.Code)

				if tc.poolCleared {
					assert.True(mt, tpm.IsPoolCleared(), "expected pool to be cleared but was not")
					return
				}

				// if pool shouldn't be cleared, another operation should succeed
				_, err = mt.Coll.InsertOne(context.Background(), bson.D{{"test", 1}})
				assert.Nil(mt, err, "InsertOne error: %v", err)
				assert.False(mt, tpm.IsPoolCleared(), "expected pool to not be cleared but was")
			})
		}
	})
}
