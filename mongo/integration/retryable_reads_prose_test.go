// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/eventtest"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
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

	mtOpts := mtest.NewOptions().ClientOptions(clientOpts).MinServerVersion("4.3")
	mt := mtest.New(t, mtOpts)

	mt.Run("PoolClearedError retryability", func(mt *mtest.T) {
		if mtest.ClusterTopologyKind() == mtest.LoadBalanced {
			mt.Skip("skipping as load balanced topology has different pool clearing behavior")
		}

		// Insert a document to test collection.
		_, err := mt.Coll.InsertOne(context.Background(), bson.D{{"x", 1}})
		assert.Nil(mt, err, "InsertOne error: %v", err)

		// Force Find to block for 1 second once.
		mt.SetFailPoint(mtest.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: mtest.FailPointMode{
				Times: 1,
			},
			Data: mtest.FailPointData{
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

		// Gather GetSucceeded, GetFailed and PoolCleared pool events.
		events := tpm.Events(func(e *event.PoolEvent) bool {
			getSucceeded := e.Type == event.GetSucceeded
			getFailed := e.Type == event.GetFailed
			poolCleared := e.Type == event.PoolCleared
			return getSucceeded || getFailed || poolCleared
		})

		// Assert that first check out succeeds, pool is cleared, and second check
		// out fails due to connection error.
		assert.True(mt, len(events) >= 3, "expected at least 3 events, got %v", len(events))
		assert.Equal(mt, event.GetSucceeded, events[0].Type,
			"expected ConnectionCheckedOut event, got %v", events[0].Type)
		assert.Equal(mt, event.PoolCleared, events[1].Type,
			"expected ConnectionPoolCleared event, got %v", events[1].Type)
		assert.Equal(mt, event.GetFailed, events[2].Type,
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

	mtOpts = mtest.NewOptions().Topologies(mtest.Sharded).MinServerVersion("4.2")
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
				hosts := options.Client().ApplyURI(mtest.ClusterURI()).Hosts
				require.GreaterOrEqualf(mt, len(hosts), tc.hostCount,
					"test cluster must have at least %v mongos hosts", tc.hostCount)

				// Configure the failpoint options for each mongos.
				failPoint := mtest.FailPoint{
					ConfigureFailPoint: "failCommand",
					Mode: mtest.FailPointMode{
						Times: 1,
					},
					Data: mtest.FailPointData{
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
}
