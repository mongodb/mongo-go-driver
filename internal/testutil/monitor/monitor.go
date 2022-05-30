// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package monitor provides test types that are used to monitor client state and actions via the
// various monitor types supported by the driver.
package monitor

import (
	"sync"

	"go.mongodb.org/mongo-driver/event"
)

// TestPoolMonitor exposes an *event.TestPoolMonitor and collects all events logged to that
// *event.TestPoolMonitor. It is safe to use from multiple concurrent goroutines.
type TestPoolMonitor struct {
	*event.PoolMonitor

	events []*event.PoolEvent
	mu     sync.RWMutex
}

func NewTestPoolMonitor() *TestPoolMonitor {
	tpm := &TestPoolMonitor{
		events: make([]*event.PoolEvent, 0),
	}
	tpm.PoolMonitor = &event.PoolMonitor{
		Event: func(evt *event.PoolEvent) {
			tpm.mu.Lock()
			defer tpm.mu.Unlock()
			tpm.events = append(tpm.events, evt)
		},
	}
	return tpm
}

// Events returns a copy of the events collected by the testPoolMonitor. Filters can optionally be
// applied to the returned events set and are applied using AND logic (i.e. all filters must return
// true to include the event in the result).
func (tpm *TestPoolMonitor) Events(filters ...func(*event.PoolEvent) bool) []*event.PoolEvent {
	tpm.mu.RLock()
	defer tpm.mu.RUnlock()

	filtered := make([]*event.PoolEvent, 0, len(tpm.events))
	for _, evt := range tpm.events {
		keep := true
		for _, filter := range filters {
			if !filter(evt) {
				keep = false
				break
			}
		}
		if keep {
			filtered = append(filtered, evt)
		}
	}

	return filtered
}

// ClearEvents will reset the events collected by the testPoolMonitor.
func (tpm *TestPoolMonitor) ClearEvents() {
	tpm.mu.Lock()
	defer tpm.mu.Unlock()
	tpm.events = tpm.events[:0]
}

// IsPoolCleared returns true if there are any events of type "event.PoolCleared" in the events
// recorded by the testPoolMonitor.
func (tpm *TestPoolMonitor) IsPoolCleared() bool {
	poolClearedEvents := tpm.Events(func(evt *event.PoolEvent) bool {
		return evt.Type == event.PoolCleared
	})
	return len(poolClearedEvents) > 0
}
