// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"sync/atomic"
	"testing"

	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
)

// Helper functions to condense event recording in tests
func recordPoolEvent(c *clientEntity) {
	c.pooled = append(c.pooled, &event.PoolEvent{})
	c.eventSequencer.recordPooledEvent()
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
		cutoffAfter    int // Set cutoff after this many events (0 = no cutoff)
		expectedPooled int
		expectedSDAM   map[monitoringEventType]int
	}{
		{
			name:        "no cutoff filters nothing",
			cutoffAfter: 0,
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
			name:        "cutoff after 2 pool events filters first 2",
			cutoffAfter: 2,
			setupEvents: func(c *clientEntity) {
				recordPoolEvent(c)
				recordPoolEvent(c)
				// Cutoff will be set here (after event 2)
				recordPoolEvent(c)
				recordPoolEvent(c)
				recordPoolEvent(c)
			},
			expectedPooled: 3, // Events 3, 4, 5
			expectedSDAM:   map[monitoringEventType]int{},
		},
		{
			name:        "cutoff filters mixed pool and SDAM events",
			cutoffAfter: 4,
			setupEvents: func(c *clientEntity) {
				recordPoolEvent(c)
				recordServerDescChanged(c)
				recordPoolEvent(c)
				recordTopologyOpening(c)
				// Cutoff will be set here (after event 4)
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
			name:        "cutoff after all events filters everything",
			cutoffAfter: 3,
			setupEvents: func(c *clientEntity) {
				recordPoolEvent(c)
				recordPoolEvent(c)
				recordServerDescChanged(c)
				// Cutoff will be set here (after all 3 events)
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

			// Set cutoff if specified
			if tt.cutoffAfter > 0 {
				// Manually set cutoff to the specified event sequence
				client.eventSequencer.cutoff.Store(int64(tt.cutoffAfter))
			}

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

func Test_eventSequencer_shouldFilter(t *testing.T) {
	es := &eventSequencer{
		seqByEventType: map[monitoringEventType][]int64{
			serverDescriptionChangedEvent: {1, 2, 3, 4, 5},
		},
	}
	es.counter = atomic.Int64{}
	es.counter.Store(5)

	tests := []struct {
		name      string
		cutoff    int64
		eventType monitoringEventType
		index     int
		expected  bool
	}{
		{
			name:      "no cutoff",
			cutoff:    0,
			eventType: serverDescriptionChangedEvent,
			index:     0,
			expected:  false,
		},
		{
			name:      "before cutoff",
			cutoff:    3,
			eventType: serverDescriptionChangedEvent,
			index:     0,
			expected:  true, // seq=1 <= 3
		},
		{
			name:      "at cutoff",
			cutoff:    3,
			eventType: serverDescriptionChangedEvent,
			index:     2,
			expected:  true, // seq=3 <= 3
		},
		{
			name:      "after cutoff",
			cutoff:    3,
			eventType: serverDescriptionChangedEvent,
			index:     3,
			expected:  false, // seq=4 > 3
		},
		{
			name:      "last event",
			cutoff:    3,
			eventType: serverDescriptionChangedEvent,
			index:     4,
			expected:  false, // seq=5 > 3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			es.cutoff.Store(tt.cutoff)
			result := es.shouldFilter(tt.eventType, tt.index)
			assert.Equal(t, tt.expected, result, "shouldFilter result mismatch")
		})
	}
}
