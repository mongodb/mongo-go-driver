// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"runtime"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/internal"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestSDAMProse(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()

	// Server limits non-streaming heartbeats and explicit server transition checks to at most one
	// per 500ms. Set the test interval to 500ms to minimize the difference between the behavior of
	// streaming and non-streaming heartbeat intervals.
	heartbeatInterval := 500 * time.Millisecond
	heartbeatIntervalClientOpts := options.Client().
		SetHeartbeatInterval(heartbeatInterval)
	heartbeatIntervalMtOpts := mtest.NewOptions().
		ClientOptions(heartbeatIntervalClientOpts).
		CreateCollection(false).
		ClientType(mtest.Proxy)
	mt.RunOpts("heartbeats processed more frequently", heartbeatIntervalMtOpts, func(mt *mtest.T) {
		// Test that setting heartbeat interval to 500ms causes the client to process heartbeats
		// approximately every 500ms instead of the default 10s. Note that a Client doesn't
		// guarantee that it will process heartbeats exactly every 500ms, just that it will wait at
		// least 500ms between heartbeats (and should process heartbeats more frequently for shorter
		// interval settings).
		//
		// For number of nodes N, interval I, and duration D, a Client should process at most X
		// operations:
		//
		//   X = (N * (1 handshake + D/I heartbeats + D/I RTTs))
		//
		// Assert that a Client processes the expected number of operations for heartbeats sent at
		// an interval between I and 2*I to account for different actual heartbeat intervals under
		// different runtime conditions.

		duration := 2 * time.Second

		numNodes := len(options.Client().ApplyURI(mtest.ClusterURI()).Hosts)
		maxExpected := numNodes * (1 + 2*int(duration/heartbeatInterval))
		minExpected := numNodes * (1 + 2*int(duration/(heartbeatInterval*2)))

		time.Sleep(duration)
		messages := mt.GetProxiedMessages()
		assert.True(
			mt,
			len(messages) >= minExpected && len(messages) <= maxExpected,
			"expected number of messages to be in range [%d, %d], got %d"+
				" (num nodes = %d, duration = %v, interval = %v)",
			minExpected,
			maxExpected,
			len(messages),
			numNodes,
			duration,
			heartbeatInterval)
	})

	mt.RunOpts("rtt tests", noClientOpts, func(mt *mtest.T) {
		clientOpts := options.Client().
			SetHeartbeatInterval(500 * time.Millisecond).
			SetAppName("streamingRttTest")
		mtOpts := mtest.NewOptions().
			MinServerVersion("4.4").
			ClientOptions(clientOpts)
		mt.RunOpts("rtt is continuously updated", mtOpts, func(mt *mtest.T) {
			// Test that the RTT monitor updates the RTT for server descriptions.

			// The server has been discovered by the create command issued by mtest. Sleep for two seconds to allow
			// multiple heartbeats to finish.
			testTopology := getTopologyFromClient(mt.Client)
			time.Sleep(2 * time.Second)
			for _, serverDesc := range testTopology.Description().Servers {
				assert.NotEqual(mt, description.Unknown, serverDesc.Kind, "server %v is Unknown", serverDesc)
				assert.True(mt, serverDesc.AverageRTTSet, "AverageRTTSet for server description %v is false", serverDesc)

				if runtime.GOOS != "windows" {
					// Windows has a lower time resolution than other platforms, which causes the reported RTT to be
					// 0 if it's below some threshold. The assertion above already confirms that the RTT is set to
					// a value, so we can skip this assertion on Windows.
					assert.True(mt, serverDesc.AverageRTT > 0, "server description %v has 0 RTT", serverDesc)
				}
			}

			// Force hello requests to block for 500ms and wait until a server's average RTT goes over 250ms.
			mt.SetFailPoint(mtest.FailPoint{
				ConfigureFailPoint: "failCommand",
				Mode: mtest.FailPointMode{
					Times: 1000,
				},
				Data: mtest.FailPointData{
					FailCommands:    []string{internal.LegacyHello, "hello"},
					BlockConnection: true,
					BlockTimeMS:     500,
					AppName:         "streamingRttTest",
				},
			})
			callback := func() {
				for {
					// We don't know which server received the failpoint command, so we wait until any of the server
					// RTTs cross the threshold.
					for _, serverDesc := range testTopology.Description().Servers {
						if serverDesc.AverageRTT > 250*time.Millisecond {
							return
						}
					}

					// The next update will be in ~500ms.
					time.Sleep(500 * time.Millisecond)
				}
			}
			assert.Soon(t, callback, defaultCallbackTimeout)
		})
	})
}
