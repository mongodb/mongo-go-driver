// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"fmt"
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
	defer mt.Close()

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

	mtOpts = mtest.NewOptions().ClientOptions(clientOpts).MinServerVersion("4.2").
		Topologies(mtest.Sharded)

	mt.RunOpts("retry on different mongos", mtOpts, func(mt *mtest.T) {
		const hostCount = 3

		hosts := options.Client().ApplyURI(mtest.ClusterURI()).Hosts
		require.GreaterOrEqualf(mt, len(hosts), hostCount, "test cluster must have at least 2 mongos hosts")

		// Configure a failpoint for the first mongos host.
		failPoint := mtest.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: mtest.FailPointMode{
				Times: 1,
			},
			Data: mtest.FailPointData{
				FailCommands:    []string{"find"},
				ErrorCode:       11600,
				CloseConnection: false,
			},
		}

		// In order to ensure that each mongos in the hostCount-many mongos hosts
		// are tried at least once (i.e. failures are deprioritized), we set a
		// failpoint on all mongos hosts. The idea is that if we get hostCount-many
		// failures, then by the pigeonhole principal all mongos hosts must have
		// been tried.
		for i := 0; i < hostCount; i++ {
			mt.ResetClient(options.Client().SetHosts([]string{hosts[i]}))
			mt.SetFailPoint(failPoint)

			// The automatic failpoint clearing may not clear failpoints set on
			// specific hosts, so manually clear the failpoint we set on the specific
			// mongos when the test is done.
			defer mt.ResetClient(options.Client().SetHosts([]string{hosts[i]}))
			defer mt.ClearFailPoints()
		}

		findCommandFailedCount := 0

		commandMonitor := &event.CommandMonitor{
			Failed: func(_ context.Context, _ *event.CommandFailedEvent) {
				findCommandFailedCount++
			},
		}

		// Reset the client with exactly hostCount-many mongos hosts.
		mt.ResetClient(options.Client().
			SetHosts(hosts[:hostCount]).
			SetTimeout(10000 * time.Millisecond).
			SetRetryReads(true).
			SetMonitor(commandMonitor))

		// ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		// defer cancel()

		err := mt.Coll.FindOne(context.Background(), bson.D{}).Err()
		fmt.Println("err: ", err)

		assert.Equal(mt, hostCount, findCommandFailedCount)

		// Create a connection to a database for each mongos host
		// mongosOpts := options.Client().ApplyURI(hosts[0])

		// firstMongos, err := mongo.Connect(context.Background(), mongosOpts)
		// require.NoError(mt, err)

		// result := firstMongos.Database("admin").RunCommand(context.Background(), doc)
		// require.NoError(mt, result.Err())

		// secondMongos, err := mongo.Connect(context.Background(), mongosOpts)
		// require.NoError(mt, err)

		// result = secondMongos.Database("admin").RunCommand(context.Background(), doc)
		// require.NoError(mt, result.Err())
	})
}
