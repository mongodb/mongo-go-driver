// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"sync"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	errorNotPrimary            int32 = 10107
	errorShutdownInProgress    int32 = 91
	errorInterruptedAtShutdown int32 = 11600
)

// testPoolMonitor exposes an *event.PoolMonitor and collects all events logged to that
// *event.PoolMonitor. It is safe to use from multiple concurrent goroutines.
type testPoolMonitor struct {
	*event.PoolMonitor

	events []*event.PoolEvent
	mu     sync.RWMutex
}

func newTestPoolMonitor() *testPoolMonitor {
	tpm := &testPoolMonitor{
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
func (tpm *testPoolMonitor) Events(filters ...func(*event.PoolEvent) bool) []*event.PoolEvent {
	filtered := make([]*event.PoolEvent, 0, len(tpm.events))
	tpm.mu.RLock()
	defer tpm.mu.RUnlock()

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

// IsPoolCleared returns true if there are any events of type "event.PoolCleared" in the events
// recorded by the testPoolMonitor.
func (tpm *testPoolMonitor) IsPoolCleared() bool {
	poolClearedEvents := tpm.Events(func(evt *event.PoolEvent) bool {
		return evt.Type == event.PoolCleared
	})
	return len(poolClearedEvents) > 0
}

var poolChan = make(chan *event.PoolEvent, 100)

// TODO(GODRIVER-2068): Replace all uses of poolMonitor with individual instances of testPoolMonitor.
var poolMonitor = &event.PoolMonitor{
	Event: func(event *event.PoolEvent) {
		poolChan <- event
	},
}

func isPoolCleared() bool {
	for len(poolChan) > 0 {
		curr := <-poolChan
		if curr.Type == event.PoolCleared {
			return true
		}
	}
	return false
}

func clearPoolChan() {
	for len(poolChan) > 0 {
		<-poolChan
	}
}

func TestConnectionsSurvivePrimaryStepDown(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().Topologies(mtest.ReplicaSet).CreateClient(false))
	defer mt.Close()

	clientOpts := options.Client().
		ApplyURI(mtest.ClusterURI()).
		SetRetryWrites(false).
		SetPoolMonitor(poolMonitor)

	getMoreOpts := mtest.NewOptions().MinServerVersion("4.2").ClientOptions(clientOpts)
	mt.RunOpts("getMore iteration", getMoreOpts, func(mt *mtest.T) {
		clearPoolChan()

		initCollection(mt, mt.Coll)
		cur, err := mt.Coll.Find(mtest.Background, bson.D{}, options.Find().SetBatchSize(2))
		assert.Nil(mt, err, "Find error: %v", err)
		defer cur.Close(mtest.Background)
		assert.True(mt, cur.Next(mtest.Background), "expected Next true, got false")

		// replSetStepDown can fail with transient errors, so we use executeAdminCommandWithRetry to handle them and
		// retry until a timeout is hit.
		stepDownCmd := bson.D{
			{"replSetStepDown", 5},
			{"force", true},
		}
		stepDownOpts := options.RunCmd().SetReadPreference(mtest.PrimaryRp)
		executeAdminCommandWithRetry(mt, mt.Client, stepDownCmd, stepDownOpts)

		assert.True(mt, cur.Next(mtest.Background), "expected Next true, got false")
		assert.False(mt, isPoolCleared(), "expected pool to not be cleared but was")
	})
	mt.RunOpts("server errors", noClientOpts, func(mt *mtest.T) {
		// Use a low heartbeat frequency so the Client will quickly recover when using failpoints that cause SDAM state
		// changes.
		clientOpts.SetHeartbeatInterval(defaultHeartbeatInterval)

		testCases := []struct {
			name                   string
			minVersion, maxVersion string
			errCode                int32
			poolCleared            bool
		}{
			{"notPrimary keep pool", "4.2", "", errorNotPrimary, false},
			{"notPrimary reset pool", "4.0", "4.0", errorNotPrimary, true},
			{"shutdown in progress reset pool", "4.0", "", errorShutdownInProgress, true},
			{"interrupted at shutdown reset pool", "4.0", "", errorInterruptedAtShutdown, true},
		}
		for _, tc := range testCases {
			tcOpts := mtest.NewOptions().ClientOptions(clientOpts)
			if tc.minVersion != "" {
				tcOpts.MinServerVersion(tc.minVersion)
			}
			if tc.maxVersion != "" {
				tcOpts.MaxServerVersion(tc.maxVersion)
			}
			mt.RunOpts(tc.name, tcOpts, func(mt *mtest.T) {
				clearPoolChan()

				mt.SetFailPoint(mtest.FailPoint{
					ConfigureFailPoint: "failCommand",
					Mode: mtest.FailPointMode{
						Times: 1,
					},
					Data: mtest.FailPointData{
						FailCommands: []string{"insert"},
						ErrorCode:    tc.errCode,
					},
				})

				_, err := mt.Coll.InsertOne(mtest.Background, bson.D{{"test", 1}})
				assert.NotNil(mt, err, "expected InsertOne error, got nil")
				cerr, ok := err.(mongo.CommandError)
				assert.True(mt, ok, "expected error type %v, got %v", mongo.CommandError{}, err)
				assert.Equal(mt, tc.errCode, cerr.Code, "expected error code %v, got %v", tc.errCode, cerr.Code)

				if tc.poolCleared {
					assert.True(mt, isPoolCleared(), "expected pool to be cleared but was not")
					return
				}

				// if pool shouldn't be cleared, another operation should succeed
				_, err = mt.Coll.InsertOne(mtest.Background, bson.D{{"test", 1}})
				assert.Nil(mt, err, "InsertOne error: %v", err)
				assert.False(mt, isPoolCleared(), "expected pool to not be cleared but was")
			})
		}
	})
}
