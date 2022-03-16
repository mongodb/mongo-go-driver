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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestServerSelectionProse implements the Server Selection prose tests:
// https://github.com/mongodb/specifications/blob/master/source/server-selection/server-selection-tests.rst
func TestServerSelectionProse(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().CreateClient(false))
	defer mt.Close()

	mtOpts := mtest.NewOptions().Topologies(mtest.Sharded).MinServerVersion("4.9")
	mt.RunOpts("operationCount-based selection within latency window, with failpoint", mtOpts, func(mt *mtest.T) {
		_, err := mt.Coll.InsertOne(context.Background(), bson.D{})
		require.NoError(mt, err, "InsertOne() error")

		hosts := options.Client().ApplyURI(mtest.ClusterURI()).Hosts
		require.GreaterOrEqualf(mt, len(hosts), 2, "test cluster must have at least 2 mongos hosts")

		// Set a failpoint on a specific mongos host that delays all "find" commands for 500ms. We
		// need to know which mongos we set the failpoint on for our assertions later.
		failpointHost := hosts[0]
		mt.ResetClient(options.Client().
			SetHosts([]string{failpointHost}))
		mt.SetFailPoint(mtest.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: mtest.FailPointMode{
				Times: 10000,
			},
			Data: mtest.FailPointData{
				FailCommands:    []string{"find"},
				BlockConnection: true,
				BlockTimeMS:     500,
				AppName:         "loadBalancingTest",
			},
		})
		// The automatic failpoint clearing may not clear failpoints set on specific hosts, so
		// manually clear the failpoint we set on the specific mongos when the test is done.
		defer func() {
			mt.ResetClient(options.Client().
				SetHosts([]string{failpointHost}))
			mt.ClearFailPoints()
		}()

		// Reset the client with exactly 2 mongos hosts.
		tpm := newTestPoolMonitor()
		mt.ResetClient(options.Client().
			SetAppName("loadBalancingTest").
			SetHosts(hosts[:2]).
			SetPoolMonitor(tpm.PoolMonitor))

		// Start 25 goroutines that each run 10 findOne operations. Run 25 goroutines instead of the
		// 10 that the prose test specifies to reduce intermittent test failures caused by the
		// random selections not being perfectly even over small numbers of operations.
		var wg sync.WaitGroup
		for i := 0; i < 25; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for i := 0; i < 10; i++ {
					res := mt.Coll.FindOne(context.Background(), bson.D{})
					assert.NoError(t, res.Err(), "FindOne() error")
				}
			}()
		}
		wg.Wait()

		// Get all checkOut events and calculate the number of times each server was selected. The
		// prose test spec says to use command monitoring events, but those don't include the server
		// address, so use checkOut events instead.
		checkOutEvents := tpm.Events(func(evt *event.PoolEvent) bool {
			return evt.Type == event.GetStarted
		})
		counts := make(map[string]int)
		for _, evt := range checkOutEvents {
			counts[evt.Address]++
		}
		assert.Equal(mt, 2, len(counts), "expected exactly 2 server addresses")

		// Calculate the frequency that the server with the failpoint was selected. Assert that it
		// was selected less than 25% of the time.
		frequency := float64(counts[failpointHost]) / float64(len(checkOutEvents))
		assert.Lessf(mt, frequency, 0.25, "expected failpoint host %q to be selected less than 25%% of the time", failpointHost)
	})

	mtOpts = mtest.NewOptions().Topologies(mtest.Sharded)
	mt.RunOpts("operationCount-based selection within latency window, no failpoint", mtOpts, func(mt *mtest.T) {
		_, err := mt.Coll.InsertOne(context.Background(), bson.D{})
		require.NoError(mt, err, "InsertOne() error")

		hosts := options.Client().ApplyURI(mtest.ClusterURI()).Hosts
		require.GreaterOrEqualf(mt, len(hosts), 2, "test cluster must have at least 2 mongos hosts")

		// Reset the client with exactly 2 mongos hosts.
		tpm := newTestPoolMonitor()
		mt.ResetClient(options.Client().
			SetHosts(hosts[:2]).
			SetPoolMonitor(tpm.PoolMonitor))

		// Sleep for 100ms to allow all server state discovery to complete. We need both servers to
		// be selectable when we start running the test or the distribution of selected servers will
		// be skewed. Unfortunately there's not currently another signal we can block on.
		time.Sleep(100 * time.Millisecond)

		// Start 25 goroutines that each run 10 findOne operations. Run 25 goroutines instead of the
		// 10 that the prose test specifies to reduce intermittent test failures caused by the
		// random selections not being perfectly even over small numbers of operations.
		var wg sync.WaitGroup
		for i := 0; i < 25; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for i := 0; i < 10; i++ {
					res := mt.Coll.FindOne(context.Background(), bson.D{})
					assert.NoError(mt, res.Err(), "FindOne() error")
				}
			}()
		}
		wg.Wait()

		// Get all checkOut events and calculate the number of times each server was selected. The
		// prose test spec says to use command monitoring events, but those don't include the server
		// address, so use checkOut events instead.
		checkOutEvents := tpm.Events(func(evt *event.PoolEvent) bool {
			return evt.Type == event.GetStarted
		})
		counts := make(map[string]int)
		for _, evt := range checkOutEvents {
			counts[evt.Address]++
		}
		assert.Equal(mt, 2, len(counts), "expected exactly 2 server addresses")

		// Calculate the frequency that each server was selected. Assert that each server was
		// selected 50% (+/- 10%) of the time.
		for addr, count := range counts {
			frequency := float64(count) / float64(len(checkOutEvents))
			assert.InDeltaf(mt,
				0.5,
				frequency,
				0.1,
				"expected server %q to be selected 50%% (+/- 10%%) of the time, but was selected %v%% of the time",
				addr, frequency*100)
		}
	})
}
