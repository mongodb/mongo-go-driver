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

	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestSDAMProse(t *testing.T) {
	mt := mtest.New(t)
	defer mt.Close()

	lowHeartbeatFrequency := 50 * time.Millisecond
	heartbeatFrequencyClientOpts := options.Client().
		SetHeartbeatInterval(lowHeartbeatFrequency)
	heartbeatFrequencyMtOpts := mtest.NewOptions().
		ClientOptions(heartbeatFrequencyClientOpts).
		CreateCollection(false).
		ClientType(mtest.Proxy)
	mt.RunOpts("heartbeats processed more frequently", heartbeatFrequencyMtOpts, func(mt *mtest.T) {
		// Test that lowering heartbeat frequency to 50ms causes the client to process heartbeats more frequently.
		//
		// In X ms, tests on 4.4+ should process at least numberOfNodes * (1 + X/frequency + X/frequency) isMaster
		// responses:
		// Each node should process 1 normal response (connection handshake) + X/frequency awaitable responses.
		// Each node should also process X/frequency RTT isMaster responses.
		//
		// Tests on < 4.4 should process at least numberOfNodes * X/frequency messages.

		numNodes := len(options.Client().ApplyURI(mtest.ClusterURI()).Hosts)
		timeDuration := 250 * time.Millisecond
		numExpectedResponses := numNodes * int(timeDuration/lowHeartbeatFrequency)
		if mtest.CompareServerVersions(mtest.ServerVersion(), "4.4") >= 0 {
			numExpectedResponses = numNodes * (2*int(timeDuration/lowHeartbeatFrequency) + 1)
		}

		time.Sleep(timeDuration + 50*time.Millisecond)
		messages := mt.GetProxiedMessages()
		assert.True(mt, len(messages) >= numExpectedResponses, "expected at least %d responses, got %d",
			numExpectedResponses, len(messages))
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

			// Force isMaster and hello requests to block for 500ms and wait until a server's average RTT goes over 250ms.
			mt.SetFailPoint(mtest.FailPoint{
				ConfigureFailPoint: "failCommand",
				Mode: mtest.FailPointMode{
					Times: 1000,
				},
				Data: mtest.FailPointData{
					FailCommands:    []string{"isMaster", "hello"},
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
