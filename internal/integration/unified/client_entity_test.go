// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo/address"
)

// Helper functions to condense event recording in tests
func recordPoolEvent(c *clientEntity) {
	c.pooled = append(c.pooled, &event.PoolEvent{})
	c.eventSequencer.recordEvent(poolAnyEvent)
}

func recordServerDescChanged(c *clientEntity) {
	c.serverDescriptionChanged = append(c.serverDescriptionChanged, &event.ServerDescriptionChangedEvent{})
	c.eventSequencer.recordEvent(serverDescriptionChangedEvent)
}

func recordTopologyOpening(c *clientEntity) {
	c.topologyOpening = append(c.topologyOpening, &event.TopologyOpeningEvent{})
	c.eventSequencer.recordEvent(topologyOpeningEvent)
}

func Test_eventSequencer(t *testing.T) {
	tests := []struct {
		name           string
		setupEvents    func(*clientEntity)
		expectedPooled int
		expectedSDAM   map[monitoringEventType]int
	}{
		{
			name: "no cutoff filters nothing",
			setupEvents: func(c *clientEntity) {
				recordPoolEvent(c)
				recordPoolEvent(c)
				recordPoolEvent(c)
				recordServerDescChanged(c)
				recordServerDescChanged(c)
			},
			expectedPooled: 3,
			expectedSDAM: map[monitoringEventType]int{
				serverDescriptionChangedEvent: 2,
			},
		},
		{
			name: "cutoff after 2 pool events filters first 2",
			setupEvents: func(c *clientEntity) {
				recordPoolEvent(c)
				recordPoolEvent(c)
				c.eventSequencer.setCutoff()
				recordPoolEvent(c)
				recordPoolEvent(c)
				recordPoolEvent(c)
			},
			expectedPooled: 3, // Events 3, 4, 5
			expectedSDAM:   map[monitoringEventType]int{},
		},
		{
			name: "cutoff filters mixed pool and SDAM events",
			setupEvents: func(c *clientEntity) {
				recordPoolEvent(c)
				recordServerDescChanged(c)
				recordPoolEvent(c)
				recordTopologyOpening(c)
				c.eventSequencer.setCutoff()
				recordPoolEvent(c)
				recordServerDescChanged(c)
				recordTopologyOpening(c)
			},
			expectedPooled: 1,
			expectedSDAM: map[monitoringEventType]int{
				serverDescriptionChangedEvent: 1,
				topologyOpeningEvent:          1,
			},
		},
		{
			name: "cutoff after all events filters everything",
			setupEvents: func(c *clientEntity) {
				recordPoolEvent(c)
				recordPoolEvent(c)
				recordServerDescChanged(c)
				c.eventSequencer.setCutoff()
			},
			expectedPooled: 0,
			expectedSDAM: map[monitoringEventType]int{
				serverDescriptionChangedEvent: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal clientEntity
			client := &clientEntity{
				eventSequencer: eventSequencer{
					seqByEventType: make(map[monitoringEventType][]int64),
				},
			}

			// Setup events
			tt.setupEvents(client)

			// Test pool event filtering
			filteredPool := filterEventsBySeq(client, client.pooled, poolAnyEvent)
			assert.Equal(t, tt.expectedPooled, len(filteredPool), "pool events count mismatch")

			// Test SDAM event filtering
			for eventType, expectedCount := range tt.expectedSDAM {
				var actualCount int

				switch eventType {
				case serverDescriptionChangedEvent:
					actualCount = len(filterEventsBySeq(client, client.serverDescriptionChanged, serverDescriptionChangedEvent))
				case serverHeartbeatSucceededEvent:
					actualCount = len(filterEventsBySeq(client, client.serverHeartbeatSucceeded, serverHeartbeatSucceededEvent))
				case topologyOpeningEvent:
					actualCount = len(filterEventsBySeq(client, client.topologyOpening, topologyOpeningEvent))
				}

				assert.Equal(t, expectedCount, actualCount, "%s count mismatch", eventType)
			}
		})
	}
}

func Test_eventSequencer_setCutoff(t *testing.T) {
	client := &clientEntity{
		eventSequencer: eventSequencer{
			seqByEventType: make(map[monitoringEventType][]int64),
		},
	}

	// Record some events
	recordPoolEvent(client)
	recordPoolEvent(client)

	// Verify counter is at 2
	assert.Equal(t, int64(2), client.eventSequencer.counter.Load(), "counter should be 2")

	// Set cutoff
	client.eventSequencer.setCutoff()

	// Verify cutoff matches counter
	assert.Equal(t, int64(2), client.eventSequencer.cutoff.Load(), "cutoff should be 2")

	// Record more events
	recordPoolEvent(client)

	// Verify counter incremented but cutoff didn't
	assert.Equal(t, int64(3), client.eventSequencer.counter.Load(), "counter should be 3")
	assert.Equal(t, int64(2), client.eventSequencer.cutoff.Load(), "cutoff should still be 2")
}

func Test_checkAllPoolsReady(t *testing.T) {
	tests := []struct {
		name           string
		minPoolSize    uint64
		poolEvents     []*event.PoolEvent
		topologyEvents []*event.TopologyDescriptionChangedEvent
		want           bool
	}{
		{
			name:        "single pool reaches minPoolSize",
			minPoolSize: 2,
			poolEvents: []*event.PoolEvent{
				{Type: event.ConnectionReady, Address: "localhost:27017"},
				{Type: event.ConnectionReady, Address: "localhost:27017"},
			},
			topologyEvents: []*event.TopologyDescriptionChangedEvent{
				{
					NewDescription: event.TopologyDescription{
						Servers: []event.ServerDescription{
							{Addr: address.Address("localhost:27017"), Kind: "RSPrimary"},
						},
					},
				},
			},
			want: true,
		},
		{
			name:        "multiple pools all reach minPoolSize",
			minPoolSize: 2,
			poolEvents: []*event.PoolEvent{
				{Type: event.ConnectionReady, Address: "localhost:27017"},
				{Type: event.ConnectionReady, Address: "localhost:27017"},
				{Type: event.ConnectionReady, Address: "localhost:27018"},
				{Type: event.ConnectionReady, Address: "localhost:27018"},
				{Type: event.ConnectionReady, Address: "localhost:27019"},
				{Type: event.ConnectionReady, Address: "localhost:27019"},
			},
			topologyEvents: []*event.TopologyDescriptionChangedEvent{
				{
					NewDescription: event.TopologyDescription{
						Servers: []event.ServerDescription{
							{Addr: address.Address("localhost:27017"), Kind: "RSPrimary"},
							{Addr: address.Address("localhost:27018"), Kind: "RSSecondary"},
							{Addr: address.Address("localhost:27019"), Kind: "RSSecondary"},
						},
					},
				},
			},
			want: true,
		},
		{
			name:        "total connections sufficient but not per-pool",
			minPoolSize: 3,
			poolEvents: []*event.PoolEvent{
				// Total of 6 connections, but unevenly distributed
				{Type: event.ConnectionReady, Address: "localhost:27017"},
				{Type: event.ConnectionReady, Address: "localhost:27017"},
				{Type: event.ConnectionReady, Address: "localhost:27017"},
				{Type: event.ConnectionReady, Address: "localhost:27017"},
				{Type: event.ConnectionReady, Address: "localhost:27018"},
				{Type: event.ConnectionReady, Address: "localhost:27018"},
			},
			topologyEvents: []*event.TopologyDescriptionChangedEvent{
				{
					NewDescription: event.TopologyDescription{
						Servers: []event.ServerDescription{
							{Addr: address.Address("localhost:27017"), Kind: "RSPrimary"},
							{Addr: address.Address("localhost:27018"), Kind: "RSSecondary"},
						},
					},
				},
			},
			want: false,
		},
		{
			name:        "ignores non-ConnectionReady events",
			minPoolSize: 2,
			poolEvents: []*event.PoolEvent{
				{Type: event.ConnectionCreated, Address: "localhost:27017"},
				{Type: event.ConnectionReady, Address: "localhost:27017"},
				{Type: event.ConnectionReady, Address: "localhost:27017"},
				{Type: event.ConnectionCheckedOut, Address: "localhost:27017"},
			},
			topologyEvents: []*event.TopologyDescriptionChangedEvent{
				{
					NewDescription: event.TopologyDescription{
						Servers: []event.ServerDescription{
							{Addr: address.Address("localhost:27017"), Kind: "RSPrimary"},
						},
					},
				},
			},
			want: true,
		},
		{
			name:           "no pools returns false",
			minPoolSize:    1,
			poolEvents:     []*event.PoolEvent{},
			topologyEvents: []*event.TopologyDescriptionChangedEvent{},
			want:           false,
		},
		{
			name:        "no ConnectionReady events returns false",
			minPoolSize: 1,
			poolEvents: []*event.PoolEvent{
				{Type: event.ConnectionCreated, Address: "localhost:27017"},
				{Type: event.ConnectionCheckedOut, Address: "localhost:27017"},
			},
			topologyEvents: []*event.TopologyDescriptionChangedEvent{
				{
					NewDescription: event.TopologyDescription{
						Servers: []event.ServerDescription{
							{Addr: address.Address("localhost:27017"), Kind: "RSPrimary"},
						},
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkAllPoolsReady(tt.poolEvents, tt.topologyEvents, tt.minPoolSize)
			require.Equal(t, tt.want, got)
		})
	}
}
