// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"net"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/failpoint"
	"go.mongodb.org/mongo-driver/v2/internal/handshake"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/mongoutil"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo/address"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/topology"
)

func TestSDAMProse(t *testing.T) {
	mt := mtest.New(t)

	// Server limits non-streaming heartbeats and explicit server transition checks to at most one
	// per 500ms. Set the test interval to 500ms to minimize the difference between the behavior of
	// streaming and non-streaming heartbeat intervals.
	heartbeatInterval := 500 * time.Millisecond
	heartbeatIntervalClientOpts := options.Client().
		SetHeartbeatInterval(heartbeatInterval)
	heartbeatIntervalMtOpts := mtest.NewOptions().
		ClientOptions(heartbeatIntervalClientOpts).
		CreateCollection(false).
		ClientType(mtest.Proxy).
		MinServerVersion("4.4") // RTT Monitor / Streaming protocol is not supported for versions < 4.4.
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
		//   X = (N * (2 handshakes + D/I heartbeats + D/I RTTs))
		//
		// Assert that a Client processes the expected number of operations for heartbeats sent at
		// an interval between I and 2*I to account for different actual heartbeat intervals under
		// different runtime conditions.

		// Measure the actual amount of time between the start of the test and when we inspect the
		// sent messages. The sleep duration will be at least the specified duration but
		// possibly longer, which could lead to extra heartbeat messages, so account for that in
		// the assertions.
		if len(os.Getenv("DOCKER_RUNNING")) > 0 {
			mt.Skip("skipping test in docker environment")
		}
		start := time.Now()
		time.Sleep(2 * time.Second)
		messages := mt.GetProxiedMessages()
		duration := time.Since(start)

		hosts, err := mongoutil.HostsFromURI(mtest.ClusterURI())
		require.NoError(mt, err)

		numNodes := len(hosts)
		maxExpected := numNodes * (2 + 2*int(duration/heartbeatInterval))
		minExpected := numNodes * (2 + 2*int(duration/(heartbeatInterval*2)))

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
			ClientOptions(clientOpts).
			// TODO(GODRIVER-3328): FailPoints are not currently reliable on sharded
			// clusters. Remove this exclusion once we fix that.
			Topologies(mtest.Single, mtest.ReplicaSet, mtest.LoadBalanced)
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
			mt.SetFailPoint(failpoint.FailPoint{
				ConfigureFailPoint: "failCommand",
				Mode: failpoint.Mode{
					Times: 1000,
				},
				Data: failpoint.Data{
					FailCommands:    []string{handshake.LegacyHello, "hello"},
					BlockConnection: true,
					BlockTimeMS:     500,
					AppName:         "streamingRttTest",
				},
			})
			callback := func() bool {
				// We don't know which server received the failpoint command, so we wait until any of the server
				// RTTs cross the threshold.
				for _, serverDesc := range testTopology.Description().Servers {
					if serverDesc.AverageRTT > 250*time.Millisecond {
						return true
					}
				}

				// The next update will be in ~500ms.
				return false
			}
			assert.Eventually(mt,
				callback,
				defaultCallbackTimeout,
				500*time.Millisecond,
				"expected average rtt heartbeats at least within every 500 ms period")
		})
	})

	mt.RunOpts("client waits between failed Hellos", mtest.NewOptions().MinServerVersion("4.9").Topologies(mtest.Single), func(mt *mtest.T) {
		// Force hello requests to fail 5 times.
		mt.SetFailPoint(failpoint.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: failpoint.Mode{
				Times: 5,
			},
			Data: failpoint.Data{
				FailCommands: []string{handshake.LegacyHello, "hello"},
				ErrorCode:    1234,
				AppName:      "SDAMMinHeartbeatFrequencyTest",
			},
		})

		// Reset client options to use direct connection, app name, and 5s SS timeout.
		clientOpts := options.Client().SetDirect(true).
			SetAppName("SDAMMinHeartbeatFrequencyTest").
			SetServerSelectionTimeout(5 * time.Second)
		mt.ResetClient(clientOpts)

		// Assert that Ping completes successfully within 2 to 3.5 seconds.
		start := time.Now()
		err := mt.Client.Ping(context.Background(), nil)
		assert.Nil(mt, err, "Ping error: %v", err)
		pingTime := time.Since(start)
		assert.True(mt, pingTime > 2000*time.Millisecond && pingTime < 3500*time.Millisecond,
			"expected Ping to take between 2 and 3.5 seconds, took %v seconds", pingTime.Seconds())

	})
}

func TestServerHeartbeatStartedEvent(t *testing.T) {
	t.Run("emits the first HeartbeatStartedEvent before the monitoring socket was created", func(t *testing.T) {
		t.Parallel()

		const address = address.Address("localhost:9999")
		expectedEvents := []string{
			"serverHeartbeatStartedEvent",
			"client connected",
			"client hello received",
			"serverHeartbeatFailedEvent",
		}

		events := make(chan string)

		listener, err := net.Listen("tcp", address.String())
		assert.NoError(t, err)
		defer listener.Close()
		go func() {
			conn, err := listener.Accept()
			assert.NoError(t, err)
			defer conn.Close()

			events <- "client connected"
			_, _ = conn.Read(nil)
			events <- "client hello received"
		}()

		server := topology.NewServer(
			address,
			bson.NewObjectID(),
			1*time.Second,
			topology.WithServerMonitor(func(*event.ServerMonitor) *event.ServerMonitor {
				return &event.ServerMonitor{
					ServerHeartbeatStarted: func(*event.ServerHeartbeatStartedEvent) {
						events <- "serverHeartbeatStartedEvent"
					},
					ServerHeartbeatFailed: func(*event.ServerHeartbeatFailedEvent) {
						events <- "serverHeartbeatFailedEvent"
					},
				}
			}),
		)
		require.NoError(t, server.Connect(nil))

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		actualEvents := make([]string, 0, len(expectedEvents))
		for len(actualEvents) < len(expectedEvents) {
			select {
			case event := <-events:
				actualEvents = append(actualEvents, event)
			case <-ticker.C:
				assert.FailNow(t, "timed out for incoming event")
			}
		}
		assert.Equal(t, expectedEvents, actualEvents)
	})

	mt := mtest.New(t)

	mt.Run("polling must await frequency", func(mt *mtest.T) {
		var heartbeatStartedCount atomic.Int64

		servers := map[string]bool{}
		serversMu := sync.RWMutex{} // Guard the servers set

		serverMonitor := &event.ServerMonitor{
			ServerHeartbeatStarted: func(*event.ServerHeartbeatStartedEvent) {
				heartbeatStartedCount.Add(1)
			},
			TopologyDescriptionChanged: func(evt *event.TopologyDescriptionChangedEvent) {
				serversMu.Lock()
				defer serversMu.Unlock()

				for _, srv := range evt.NewDescription.Servers {
					servers[srv.Addr.String()] = true
				}
			},
		}

		// Create a client with  heartbeatFrequency=100ms,
		// serverMonitoringMode=poll. Use SDAM to record the number of times the
		// a heartbeat is started and the number of servers discovered.
		mt.ResetClient(options.Client().
			SetServerMonitor(serverMonitor).
			SetServerMonitoringMode(options.ServerMonitoringModePoll))

		// Per specifications, minHeartbeatFrequencyMS=500ms. So, within the first
		// 500ms the heartbeatStartedCount should be LEQ to the number of discovered
		// servers.
		time.Sleep(500 * time.Millisecond)

		serversMu.Lock()
		serverCount := int64(len(servers))
		serversMu.Unlock()

		assert.LessOrEqual(mt, heartbeatStartedCount.Load(), serverCount)
	})
}
