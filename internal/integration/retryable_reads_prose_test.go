// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/eventtest"
	"go.mongodb.org/mongo-driver/v2/internal/failpoint"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/mongoutil"
	"go.mongodb.org/mongo-driver/v2/internal/randutil"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

func TestRetryableReadsProse(t *testing.T) {
	tpm := eventtest.NewTestPoolMonitor()

	// Client options with MaxPoolSize of 1 and RetryReads used per the test description.
	// Lower HeartbeatInterval used to speed the test up for any server that uses streaming
	// heartbeats. Only connect to first host in list for sharded clusters.
	hosts := mtest.ClusterConnString().Hosts
	clientOpts := options.Client().SetMaxPoolSize(1).SetRetryReads(true).
		SetPoolMonitor(tpm.PoolMonitor).SetHeartbeatInterval(500 * time.Millisecond).
		SetHosts(hosts[:1])

	mt := mtest.New(t, mtest.NewOptions().ClientOptions(clientOpts))

	mtOpts := mtest.NewOptions().
		MinServerVersion("4.3").
		// Load-balanced topologies have a different behavior for clearing the
		// pool, so don't run the test on load-balanced topologies
		//
		// TODO(GODRIVER-3328): FailPoints are not currently reliable on sharded
		// topologies. Allow running on sharded topologies once that is fixed.
		Topologies(mtest.Single, mtest.ReplicaSet)
	mt.RunOpts("PoolClearedError retryability", mtOpts, func(mt *mtest.T) {
		// Insert a document to test collection.
		_, err := mt.Coll.InsertOne(context.Background(), bson.D{{"x", 1}})
		assert.Nil(mt, err, "InsertOne error: %v", err)

		// Force Find to block for 1 second once.
		mt.SetFailPoint(failpoint.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: failpoint.Mode{
				Times: 1,
			},
			Data: failpoint.Data{
				FailCommands:    []string{"find"},
				ErrorCode:       91,
				BlockConnection: true,
				BlockTimeMS:     1000,
			},
		})

		// Clear CMAP and command events.
		tpm.ClearEvents()
		mt.ClearEvents()

		// Perform a FindOne on two different threads and assert both operations are
		// successful.
		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				res := mt.Coll.FindOne(context.Background(), bson.D{})
				assert.Nil(mt, res.Err())
			}()
		}
		wg.Wait()

		// Gather ConnectionCheckedOut, ConnectionCheckOutFailed and PoolCleared pool events.
		events := tpm.Events(func(e *event.PoolEvent) bool {
			connectionCheckedOut := e.Type == event.ConnectionCheckedOut
			connectionCheckOutFailed := e.Type == event.ConnectionCheckOutFailed
			poolCleared := e.Type == event.ConnectionPoolCleared
			return connectionCheckedOut || connectionCheckOutFailed || poolCleared
		})

		// Assert that first check out succeeds, pool is cleared, and second check
		// out fails due to connection error.
		assert.True(mt, len(events) >= 3, "expected at least 3 events, got %v", len(events))
		assert.Equal(mt, event.ConnectionCheckedOut, events[0].Type,
			"expected ConnectionCheckedOut event, got %v", events[0].Type)
		assert.Equal(mt, event.ConnectionPoolCleared, events[1].Type,
			"expected ConnectionPoolCleared event, got %v", events[1].Type)
		assert.Equal(mt, event.ConnectionCheckOutFailed, events[2].Type,
			"expected ConnectionCheckedOutFailed event, got %v", events[2].Type)
		assert.Equal(mt, event.ReasonConnectionErrored, events[2].Reason,
			"expected check out failure due to connection error, failed due to %q", events[2].Reason)

		// Assert that three find CommandStartedEvents were observed.
		for i := 0; i < 3; i++ {
			cmdEvt := mt.GetStartedEvent()
			assert.NotNil(mt, cmdEvt, "expected a find event, got nil")
			assert.Equal(mt, cmdEvt.CommandName, "find",
				"expected a find event, got a(n) %v event", cmdEvt.CommandName)
		}
	})

	mtOpts = mtest.NewOptions().Topologies(mtest.Sharded).MinServerVersion("4.2").AllowFailPointsOnSharded()
	mt.RunOpts("retrying in sharded cluster", mtOpts, func(mt *mtest.T) {
		tests := []struct {
			name string

			// Note that setting this value greater than 2 will result in false
			// negatives. The current specification does not account for CSOT, which
			// might allow for an "infinite" number of retries over a period of time.
			// Because of this, we only track the "previous server".
			hostCount            int
			failpointErrorCode   int32
			expectedFailCount    int
			expectedSuccessCount int
		}{
			{
				name:                 "retry on different mongos",
				hostCount:            2,
				failpointErrorCode:   6, // HostUnreachable
				expectedFailCount:    2,
				expectedSuccessCount: 0,
			},
			{
				name:                 "retry on same mongos",
				hostCount:            1,
				failpointErrorCode:   6, // HostUnreachable
				expectedFailCount:    1,
				expectedSuccessCount: 1,
			},
		}

		for _, tc := range tests {
			mt.Run(tc.name, func(mt *mtest.T) {
				hosts, err := mongoutil.HostsFromURI(mtest.ClusterURI())

				require.NoError(mt, err)
				require.GreaterOrEqualf(mt, len(hosts), tc.hostCount,
					"test cluster must have at least %v mongos hosts", tc.hostCount)

				// Configure the failpoint options for each mongos.
				failPoint := failpoint.FailPoint{
					ConfigureFailPoint: "failCommand",
					Mode: failpoint.Mode{
						Times: 1,
					},
					Data: failpoint.Data{
						FailCommands:    []string{"find"},
						ErrorCode:       tc.failpointErrorCode,
						CloseConnection: false,
					},
				}

				// In order to ensure that each mongos in the hostCount-many mongos
				// hosts are tried at least once (i.e. failures are deprioritized), we
				// set a failpoint on all mongos hosts. The idea is that if we get
				// hostCount-many failures, then by the pigeonhole principal all mongos
				// hosts must have been tried.
				for i := 0; i < tc.hostCount; i++ {
					mt.ResetClient(options.Client().SetHosts([]string{hosts[i]}))
					mt.SetFailPoint(failPoint)

					// The automatic failpoint clearing may not clear failpoints set on
					// specific hosts, so manually clear the failpoint we set on the
					// specific mongos when the test is done.
					defer mt.ResetClient(options.Client().SetHosts([]string{hosts[i]}))
					defer mt.ClearFailPoints()
				}

				failCount := 0
				successCount := 0

				commandMonitor := &event.CommandMonitor{
					Failed: func(context.Context, *event.CommandFailedEvent) {
						failCount++
					},
					Succeeded: func(context.Context, *event.CommandSucceededEvent) {
						successCount++
					},
				}

				// Reset the client with exactly hostCount-many mongos hosts.
				mt.ResetClient(options.Client().
					SetHosts(hosts[:tc.hostCount]).
					SetRetryReads(true).
					SetMonitor(commandMonitor))

				mt.Coll.FindOne(context.Background(), bson.D{})

				assert.Equal(mt, tc.expectedFailCount, failCount)
				assert.Equal(mt, tc.expectedSuccessCount, successCount)
			})
		}
	})

	mtOpts = mtest.NewOptions().Topologies(mtest.ReplicaSet).MinServerVersion("4.4").CreateClient(false)
	mt.RunOpts("retrying reads in a replica set", mtOpts, func(mt *mtest.T) {
		tests := []struct {
			name                      string
			errLabels                 []string
			enableOverloadRetargeting bool
			isServerIdentical         bool
		}{
			{
				name:                      "overload errors retried on a different replicaset server",
				errLabels:                 []string{"RetryableError", "SystemOverloadedError"},
				enableOverloadRetargeting: true,
				isServerIdentical:         false,
			},
			{
				name:              "non-overload errors retried on the same replicaset server",
				errLabels:         []string{"RetryableError"},
				isServerIdentical: true,
			},
			{
				name:              "overload errors retried on the same replicaset server",
				errLabels:         []string{"RetryableError", "SystemOverloadedError"},
				isServerIdentical: true,
			},
		}

		for _, tc := range tests {
			clientOpts := options.Client().SetRetryReads(true).SetReadPreference(readpref.PrimaryPreferred())
			if tc.enableOverloadRetargeting {
				clientOpts = clientOpts.SetEnableOverloadRetargeting(true)
			}
			mt.RunOpts(tc.name, mtest.NewOptions().ClientOptions(clientOpts), func(mt *mtest.T) {
				failPoint := failpoint.FailPoint{
					ConfigureFailPoint: "failCommand",
					Mode: failpoint.Mode{
						Times: 1,
					},
					Data: failpoint.Data{
						FailCommands: []string{"find"},
						ErrorLabels:  &tc.errLabels,
						ErrorCode:    6,
					},
				}
				mt.SetFailPoint(failPoint)
				mt.ClearEvents()

				_ = mt.Coll.FindOne(context.Background(), bson.D{})
				succeeded := mt.GetAllSucceededEvents()
				require.Len(mt, succeeded, 1, "expected exactly one succeeded attempt")
				failed := mt.GetAllFailedEvents()
				require.Len(mt, failed, 1, "expected exactly one failed attempt")
				wanted := "different"
				if tc.isServerIdentical {
					wanted = "identical"
				}
				assert.Equal(mt, tc.isServerIdentical, failed[0].ConnectionID == succeeded[0].ConnectionID,
					"expected failed and succeeded events to have %s connection IDs, got %v and %v", wanted,
					failed[0].ConnectionID, succeeded[0].ConnectionID)
			})
		}
	})

	errorCodesContains := func(err error, code int) bool {
		for _, ec := range mongo.ErrorCodes(err) {
			if ec == code {
				return true
			}
		}
		return false
	}

	mtOpts = mtest.NewOptions().MinServerVersion("4.4").ClientType(mtest.Pinned).AllowFailPointsOnSharded()
	mt.RunOpts("set the maximum number of retries for all retryable read errors", mtOpts, func(mt *mtest.T) {
		mt.SetFailPoint(failpoint.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: failpoint.Mode{
				Times: 1,
			},
			Data: failpoint.Data{
				FailCommands: []string{"find"},
				ErrorLabels:  &[]string{"RetryableError", "SystemOverloadedError"},
				ErrorCode:    91,
			},
		})

		var opsCnt int
		monitor := &event.CommandMonitor{
			Started: func(_ context.Context, e *event.CommandStartedEvent) {
				if e.CommandName == "find" {
					opsCnt++
				}
			},
			Failed: func(_ context.Context, event *event.CommandFailedEvent) {
				if event.CommandName != "find" {
					return
				}
				if errorCodesContains(event.Failure, 91) {
					mt.SetFailPoint(failpoint.FailPoint{
						ConfigureFailPoint: "failCommand",
						Mode:               failpoint.ModeAlwaysOn,
						Data: failpoint.Data{
							FailCommands: []string{"find"},
							ErrorLabels:  &[]string{"RetryableError"},
							ErrorCode:    91,
						},
					})
				}
			},
		}
		mt.ResetClient(options.Client().SetRetryReads(true).SetMonitor(monitor))
		_, err := mt.Coll.Find(context.Background(), bson.D{})
		var cmdErr mongo.CommandError
		require.Truef(mt, errors.As(err, &cmdErr), "expected a CommandError, got %T: %v", err, err)
		assert.Equalf(mt, 3, opsCnt, "expected 3 attempts (1 original + 2 retries), got %d", opsCnt)
	})

	mt.RunOpts("do not apply backoff to non-overload errors", mtOpts, func(mt *mtest.T) {
		mt.SetFailPoint(failpoint.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: failpoint.Mode{
				Times: 1,
			},
			Data: failpoint.Data{
				FailCommands: []string{"find"},
				ErrorLabels:  &[]string{"RetryableError", "SystemOverloadedError"},
				ErrorCode:    91,
			},
		})

		var ops []bool
		monitor := &event.CommandMonitor{
			Started: func(_ context.Context, e *event.CommandStartedEvent) {
				if e.CommandName == "find" {
					ops = append(ops, false)
				}
			},
			Failed: func(_ context.Context, event *event.CommandFailedEvent) {
				if event.CommandName != "find" {
					return
				}
				if errorCodesContains(event.Failure, 91) {
					mt.SetFailPoint(failpoint.FailPoint{
						ConfigureFailPoint: "failCommand",
						Mode:               failpoint.ModeAlwaysOn,
						Data: failpoint.Data{
							FailCommands: []string{"find"},
							ErrorLabels:  &[]string{"RetryableError"},
							ErrorCode:    91,
						},
					})
				}
			},
		}

		defer randutil.SetJitterForTesting(func(int64) int64 {
			ops[len(ops)-1] = true
			return 0
		})()

		mt.ResetClient(options.Client().SetRetryReads(true).SetMonitor(monitor))
		_, err := mt.Coll.Find(context.Background(), bson.D{})
		var cmdErr mongo.CommandError
		require.Truef(mt, errors.As(err, &cmdErr), "expected a CommandError, got %T: %v", err, err)
		assert.Equal(t, []bool{true, false, false}, ops, "expected backoff to be applied on the first attempt only, got %v", ops)
	})
}
