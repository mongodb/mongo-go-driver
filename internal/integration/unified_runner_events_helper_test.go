// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/serverselector"
	"go.mongodb.org/mongo-driver/v2/mongo/address"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/topology"
)

// Helper functions for the operations in the unified spec test runner that require assertions about SDAM and connection
// pool events.

var (
	poolEventTypesMap = map[string]string{
		"PoolClearedEvent": event.ConnectionPoolCleared,
	}
	defaultCallbackTimeout = 10 * time.Second
)

// unifiedRunnerEventMonitor monitors connection pool-related events.
type unifiedRunnerEventMonitor struct {
	poolEventCount               map[string]int
	poolEventCountLock           sync.Mutex
	sdamMonitor                  *event.ServerMonitor
	serverMarkedUnknownCount     int
	serverMarkedUnknownCountLock sync.Mutex
}

func newUnifiedRunnerEventMonitor() *unifiedRunnerEventMonitor {
	urem := unifiedRunnerEventMonitor{
		poolEventCount: make(map[string]int),
	}
	urem.sdamMonitor = &event.ServerMonitor{
		ServerDescriptionChanged: (func(e *event.ServerDescriptionChangedEvent) {
			urem.serverMarkedUnknownCountLock.Lock()
			defer urem.serverMarkedUnknownCountLock.Unlock()

			// Spec tests only ever handle ServerMarkedUnknown ServerDescriptionChangedEvents
			// for the time being.
			if e.NewDescription.Kind == description.UnknownStr {
				urem.serverMarkedUnknownCount++
			}
		}),
	}
	return &urem
}

// handlePoolEvent can be used as the event handler for a connection pool monitor.
func (u *unifiedRunnerEventMonitor) handlePoolEvent(evt *event.PoolEvent) {
	u.poolEventCountLock.Lock()
	defer u.poolEventCountLock.Unlock()

	u.poolEventCount[evt.Type]++
}

// getPoolEventCount returns the number of pool events of the given type, or 0 if no events were recorded.
func (u *unifiedRunnerEventMonitor) getPoolEventCount(eventType string) int {
	u.poolEventCountLock.Lock()
	defer u.poolEventCountLock.Unlock()

	mappedType := poolEventTypesMap[eventType]
	return u.poolEventCount[mappedType]
}

// getServerMarkedUnknownCount returns the number of ServerMarkedUnknownEvents, or 0 if none were recorded.
func (u *unifiedRunnerEventMonitor) getServerMarkedUnknownCount() int {
	u.serverMarkedUnknownCountLock.Lock()
	defer u.serverMarkedUnknownCountLock.Unlock()

	return u.serverMarkedUnknownCount
}

func waitForEvent(mt *mtest.T, test *testCase, op *operation) {
	eventType := op.Arguments.Lookup("event").StringValue()
	expectedCount := int(op.Arguments.Lookup("count").Int32())

	callback := func() bool {
		var count int
		// Spec tests only ever wait for ServerMarkedUnknown SDAM events for the time being.
		if eventType == "ServerMarkedUnknownEvent" {
			count = test.monitor.getServerMarkedUnknownCount()
		} else {
			count = test.monitor.getPoolEventCount(eventType)
		}

		return count >= expectedCount
	}

	assert.Eventually(mt,
		callback,
		defaultCallbackTimeout,
		100*time.Millisecond,
		"expected spec tests to only wait for Server Marked Unknown SDAM events")
}

func assertEventCount(mt *mtest.T, testCase *testCase, op *operation) {
	eventType := op.Arguments.Lookup("event").StringValue()
	expectedCount := int(op.Arguments.Lookup("count").Int32())

	var gotCount int
	// Spec tests only ever assert ServerMarkedUnknown SDAM events for the time being.
	if eventType == "ServerMarkedUnknownEvent" {
		gotCount = testCase.monitor.getServerMarkedUnknownCount()
	} else {
		gotCount = testCase.monitor.getPoolEventCount(eventType)
	}
	assert.Equal(mt, expectedCount, gotCount, "expected count %d for event %s, got %d", expectedCount, eventType,
		gotCount)
}

func recordPrimary(mt *mtest.T, testCase *testCase) {
	testCase.recordedPrimary = getPrimaryAddress(mt, testCase.testTopology, true)
}

func waitForPrimaryChange(mt *mtest.T, testCase *testCase, op *operation) {
	callback := func() bool {
		return getPrimaryAddress(mt, testCase.testTopology, false) != testCase.recordedPrimary
	}

	timeout := convertValueToMilliseconds(mt, op.Arguments.Lookup("timeoutMS"))
	assert.Eventually(mt,
		callback,
		timeout,
		100*time.Millisecond,
		"expected primary address to be different within the timeout period")
}

// getPrimaryAddress returns the address of the current primary. If failFast is true, the server selection fast path
// is used and the function will fail if the fast path doesn't return a server.
func getPrimaryAddress(mt *mtest.T, topo *topology.Topology, failFast bool) address.Address {
	mt.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if failFast {
		cancel()
	}

	primary, err := topo.SelectServer(ctx, &serverselector.ReadPref{ReadPref: readpref.Primary()})
	assert.Nil(mt, err, "SelectServer error: %v", err)
	return primary.(*topology.SelectedServer).Description().Addr
}
