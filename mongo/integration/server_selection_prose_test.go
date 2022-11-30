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
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/internal/testutil/monitor"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type saturatedConnections map[uint64]bool

// saturatedHosts is used to maintain information about events with specific host+pool combinations.
type saturatedHosts map[string]saturatedConnections

func (set saturatedHosts) add(host string, connectionID uint64) {
	if set[host] == nil {
		set[host] = make(saturatedConnections)
	}
	set[host][connectionID] = true
}

// isSaturated returns true when each client on the cluster URI has a tolerable number of ready connections.
func (set saturatedHosts) isSaturated(tolerance uint64) bool {
	for _, host := range options.Client().ApplyURI(mtest.ClusterURI()).Hosts {
		if cxns := set[host]; cxns == nil || uint64(len(cxns)) < tolerance {
			return false
		}
	}
	return true
}

// awaitSaturation uses CMAP events to ensure that the client's connection pools for N-mongoses have been saturated.
// The qualification for a host to be "saturated" is for each host on the client to have a tolerable number of ready
// connections.
func awaitSaturation(ctx context.Context, mt *mtest.T, monitor *monitor.TestPoolMonitor, tolerance uint64) error {
	set := make(saturatedHosts)
	var err error
	for !set.isSaturated(tolerance) {
		if err = ctx.Err(); err != nil {
			break
		}
		if err = mt.Coll.FindOne(ctx, bson.D{}).Err(); err != nil {
			break
		}
		monitor.Events(func(evt *event.PoolEvent) bool {
			// Add host only when the connection is ready for use.
			if evt.Type == event.ConnectionReady {
				set.add(evt.Address, evt.ConnectionID)
			}
			return true
		})
	}
	return err
}

// runsServerSelection will run opCount-many `FindOne` operations within threadCount-many go routines.  The purpose of
// this is to test the reliability of the server selection algorithm, which can be verified with the `counts` map and
// `event.PoolEvent` slice.
func runsServerSelection(mt *mtest.T, monitor *monitor.TestPoolMonitor,
	threadCount, opCount int) (map[string]int, []*event.PoolEvent) {
	var wg sync.WaitGroup
	for i := 0; i < threadCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for i := 0; i < opCount; i++ {
				res := mt.Coll.FindOne(context.Background(), bson.D{})
				assert.NoError(mt.T, res.Err(), "FindOne() error for Collection '%s'", mt.Coll.Name())
			}
		}()
	}
	wg.Wait()

	// Get all checkOut events and calculate the number of times each server was selected. The prose test spec says to
	// use command monitoring events, but those don't include the server address, so use checkOut events instead.
	checkOutEvents := monitor.Events(func(evt *event.PoolEvent) bool {
		return evt.Type == event.GetStarted
	})
	counts := make(map[string]int)
	for _, evt := range checkOutEvents {
		counts[evt.Address]++
	}
	assert.Equal(mt, 2, len(counts), "expected exactly 2 server addresses")
	return counts, checkOutEvents
}

// TestServerSelectionProse implements the Server Selection prose tests:
// https://github.com/mongodb/specifications/blob/master/source/server-selection/server-selection-tests.rst
func TestServerSelectionProse(t *testing.T) {
	const maxPoolSize = 10
	const localThreshold = 30 * time.Second

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

		// Reset the client with exactly 2 mongos hosts. Use a ServerMonitor to wait for both mongos
		// host descriptions to move from kind "Unknown" to kind "Mongos".
		topologyEvents := make(chan *event.TopologyDescriptionChangedEvent, 10)
		tpm := monitor.NewTestPoolMonitor()
		mt.ResetClient(options.Client().
			SetLocalThreshold(localThreshold).
			SetMaxPoolSize(maxPoolSize).
			SetMinPoolSize(maxPoolSize).
			SetHosts(hosts[:2]).
			SetPoolMonitor(tpm.PoolMonitor).
			SetAppName("loadBalancingTest").
			SetServerMonitor(&event.ServerMonitor{
				TopologyDescriptionChanged: func(evt *event.TopologyDescriptionChangedEvent) {
					topologyEvents <- evt
				},
			}))
		for evt := range topologyEvents {
			servers := evt.NewDescription.Servers
			if len(servers) == 2 && servers[0].Kind == description.Mongos && servers[1].Kind == description.Mongos {
				break
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := awaitSaturation(ctx, mt, tpm, maxPoolSize); err != nil {
			mt.Fatalf("Error awaiting saturation: %v", err.Error())
		}

		counts, checkOutEvents := runsServerSelection(mt, tpm, 10, 10)
		// Calculate the frequency that the server with the failpoint was selected. Assert that it
		// was selected less than 25% of the time.
		frequency := float64(counts[failpointHost]) / float64(len(checkOutEvents))
		assert.Lessf(mt,
			frequency,
			0.25,
			"expected failpoint host %q to be selected less than 25%% of the time",
			failpointHost)
	})

	mtOpts = mtest.NewOptions().Topologies(mtest.Sharded)
	mt.RunOpts("operationCount-based selection within latency window, no failpoint", mtOpts, func(mt *mtest.T) {
		_, err := mt.Coll.InsertOne(context.Background(), bson.D{})
		require.NoError(mt, err, "InsertOne() error")

		hosts := options.Client().ApplyURI(mtest.ClusterURI()).Hosts
		require.GreaterOrEqualf(mt, len(hosts), 2, "test cluster must have at least 2 mongos hosts")

		// Reset the client with exactly 2 mongos hosts. Use a ServerMonitor to wait for both mongos
		// host descriptions to move from kind "Unknown" to kind "Mongos".
		topologyEvents := make(chan *event.TopologyDescriptionChangedEvent, 10)
		tpm := monitor.NewTestPoolMonitor()
		mt.ResetClient(options.Client().
			SetHosts(hosts[:2]).
			SetPoolMonitor(tpm.PoolMonitor).
			SetLocalThreshold(localThreshold).
			SetMaxPoolSize(maxPoolSize).
			SetMinPoolSize(maxPoolSize).
			SetServerMonitor(&event.ServerMonitor{
				TopologyDescriptionChanged: func(evt *event.TopologyDescriptionChangedEvent) {
					topologyEvents <- evt
				},
			}))
		for evt := range topologyEvents {
			servers := evt.NewDescription.Servers
			if len(servers) == 2 && servers[0].Kind == description.Mongos && servers[1].Kind == description.Mongos {
				break
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := awaitSaturation(ctx, mt, tpm, maxPoolSize); err != nil {
			mt.Fatalf("Error awaiting saturation: %v", err.Error())
		}

		counts, checkOutEvents := runsServerSelection(mt, tpm, 10, 100)
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
