// Copyright (C) MongoDB, Inc. 2017-present.
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
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
)

func TestCMAPSpec(t *testing.T) {
	mtOpts := mtest.NewOptions().
		CreateClient(false).
		MinServerVersion("4.4.0")
	mt := mtest.New(t, mtOpts)
	defer mt.Close()

	// Test that maxConnecting is enforced.
	// Based on CMAP spec test
	// "connection-monitoring-and-pooling/pool-checkout-maxConnecting-is-enforced.json"
	mt.Run("pool-checkout-maxConnecting-is-enforced.json", func(mt *mtest.T) {
		mt.SetFailPoint(mtest.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: mtest.FailPointMode{
				Times: 50,
			},
			Data: mtest.FailPointData{
				FailCommands:    []string{"hello", internal.LegacyHello},
				CloseConnection: false,
				BlockConnection: true,
				BlockTimeMS:     750,
			},
		})

		tpm := newTestPoolMonitor()
		mt.ResetClient(options.Client().
			SetPoolMonitor(tpm.PoolMonitor).
			SetMaxPoolSize(10).
			SetMaxConnecting(2))

		events := make(chan *event.PoolEvent, 20)
		tpm.Subscribe(events)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()

			_, err := mt.Coll.InsertOne(context.Background(), bson.D{{"key", "value"}})
			assert.NoErrorf(mt, err, "InsertOne error")
		}()

		count := 1
		for evt := range events {
			if evt.Type == "ConnectionCreated" {
				count--
			}
			if count <= 0 {
				break
			}
		}

		time.Sleep(100 * time.Millisecond)

		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				_, err := mt.Coll.InsertOne(context.Background(), bson.D{{"key", "value"}})
				assert.NoErrorf(mt, err, "InsertOne error")
			}()
		}

		count = 3
		for evt := range events {
			if evt.Type == "ConnectionReady" {
				count--
			}
			if count <= 0 {
				break
			}
		}

		wg.Wait()

		want := []*event.PoolEvent{
			{
				Type:         "ConnectionCreated",
				ConnectionID: 1,
			},
			{Type: "ConnectionCreated"},
			{
				Type:         "ConnectionReady",
				ConnectionID: 1,
			},
			{Type: "ConnectionCreated"},
			{Type: "ConnectionReady"},
			{Type: "ConnectionReady"},
		}
		ignore := []string{
			"ConnectionCheckOutStarted",
			"ConnectionCheckedIn",
			"ConnectionCheckedOut",
			"ConnectionClosed",
			"ConnectionPoolCreated",
			"ConnectionPoolReady",
		}
		assertEvents(mt, want, ignore, tpm.Events())
	})

	// Test that waiting on maxConnecting is limited by WaitQueueTimeoutMS.
	// Based on CMAP spec test
	// "connection-monitoring-and-pooling/pool-checkout-maxConnecting-timeout.json"
	mt.Run("pool-checkout-maxConnecting-timeout.json", func(mt *mtest.T) {
		mt.SetFailPoint(mtest.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: mtest.FailPointMode{
				Times: 50,
			},
			Data: mtest.FailPointData{
				FailCommands:    []string{"hello", internal.LegacyHello},
				CloseConnection: false,
				BlockConnection: true,
				BlockTimeMS:     750,
			},
		})

		tpm := newTestPoolMonitor()
		mt.ResetClient(options.Client().
			SetPoolMonitor(tpm.PoolMonitor).
			SetMaxPoolSize(10).
			SetMaxConnecting(2))

		events := make(chan *event.PoolEvent, 20)
		tpm.Subscribe(events)

		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				_, err := mt.Coll.InsertOne(context.Background(), bson.D{{"key", "value"}})
				assert.NoErrorf(mt, err, "InsertOne error")
			}()
		}

		count := 2
		for evt := range events {
			if evt.Type == "ConnectionCreated" {
				count--
			}
			if count <= 0 {
				break
			}
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			_, err := mt.Coll.InsertOne(ctx, bson.D{{"key", "value"}})
			assert.IsTypef(
				mt,
				topology.WaitQueueTimeoutError{},
				err,
				"expected InsertOne error to be a WaitQueueTimeoutError")
		}()

		count = 1
		for evt := range events {
			if evt.Type == "ConnectionCheckOutFailed" {
				count--
			}
			if count <= 0 {
				break
			}
		}

		wg.Wait()

		want := []*event.PoolEvent{
			{Type: "ConnectionCheckOutStarted"},
			{Type: "ConnectionCheckOutStarted"},
			{Type: "ConnectionCheckOutStarted"},
			{
				Type:   "ConnectionCheckOutFailed",
				Reason: "timeout",
			},
		}
		ignore := []string{
			"ConnectionCreated",
			"ConnectionCheckedIn",
			"ConnectionCheckedOut",
			"ConnectionClosed",
			"ConnectionPoolCreated",
			"ConnectionPoolReady",
		}
		assertEvents(mt, want, ignore, tpm.Events())
	})

	// Test that threads blocked by maxConnecting check out returned connections.
	// Based on CMAP spec test
	// "connection-monitoring-and-pooling/pool-checkout-returned-connection-maxConnecting.json"
	// Note that the sequence of operations is modified from the original CMAP spec test to be able
	// to use the exported APIs.
	mt.Run("pool-checkout-returned-connection-maxConnecting.json", func(mt *mtest.T) {
		mt.SetFailPoint(mtest.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: mtest.FailPointMode{
				Times: 50,
			},
			Data: mtest.FailPointData{
				FailCommands:    []string{"hello", internal.LegacyHello, "insert"},
				CloseConnection: false,
				BlockConnection: true,
				BlockTimeMS:     750,
			},
		})

		tpm := newTestPoolMonitor()
		mt.ResetClient(options.Client().
			SetPoolMonitor(tpm.PoolMonitor).
			SetMaxPoolSize(10).
			SetMinPoolSize(3).
			SetMaxConnecting(2))

		events := make(chan *event.PoolEvent, 20)
		tpm.Subscribe(events)

		// Wait until 3 connections are ready in the pool. We need all 3 inserts to start at about
		// the same time, so we don't want maxConnecting to impact when connections are available
		// for these first 3 inserts.
		count := 3
		for evt := range events {
			if evt.Type == "ConnectionReady" {
				count--
			}
			if count <= 0 {
				break
			}
		}

		// Start 3 inserts that should take about 750ms to complete. Each will check their
		// connection back into the pool when the operation completes.
		var wg sync.WaitGroup
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				_, err := mt.Coll.InsertOne(context.Background(), bson.D{{"key", "value"}})
				assert.NoErrorf(mt, err, "InsertOne error")
			}()
		}

		// Wait for all 3 connection check-outs to complete so we know the insert operation just
		// started.
		count = 3
		for evt := range events {
			if evt.Type == "ConnectionCheckedOut" {
				count--
			}
			if count <= 0 {
				break
			}
		}

		// Wait for about half of the 750ms operation delay, then start 2 new insert operations.
		// That should allow 2 new connections to start establishing. Expect that those 2 new
		// connections won't complete before the 3 previously started inserts check their
		// connections back into the pool.
		time.Sleep(300 * time.Millisecond)
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				_, err := mt.Coll.InsertOne(context.Background(), bson.D{{"key", "value"}})
				assert.NoErrorf(mt, err, "InsertOne error")
			}()
		}

		// Wait for 2 connection created events so we know that the next operation should block on
		// maxConnecting for about 750ms before it can start creating a new connection.
		count = 2
		for evt := range events {
			if evt.Type == "ConnectionCreated" {
				count--
			}
			if count <= 0 {
				break
			}
		}

		// Start one more insert operation that should try to check out a connection, find an empty
		// pool, and block on maxConnecting waiting to create a new connection. One of the first 3
		// insert operations should complete before the number of connections being created is less
		// than maxConnecting. This last insert should take one of those connections instead of
		// creating a new one.
		wg.Add(1)
		go func() {
			defer wg.Done()

			_, err := mt.Coll.InsertOne(context.Background(), bson.D{{"key", "value"}})
			assert.NoErrorf(mt, err, "InsertOne error")
		}()

		wg.Wait()

		// Assert that there are 6 connections checked out and 5 created.
		want := []*event.PoolEvent{
			{Type: "ConnectionCreated"},
			{Type: "ConnectionCreated"},
			{Type: "ConnectionCreated"},
			{Type: "ConnectionCheckedOut"},
			{Type: "ConnectionCheckedOut"},
			{Type: "ConnectionCheckedOut"},
			{Type: "ConnectionCreated"},
			{Type: "ConnectionCreated"},
			{Type: "ConnectionCheckedOut"},
			{Type: "ConnectionCheckedOut"},
			{Type: "ConnectionCheckedOut"},
		}
		ignore := []string{
			"ConnectionPoolReady",
			"ConnectionClosed",
			"ConnectionReady",
			"ConnectionPoolCreated",
			"ConnectionCheckOutStarted",
			"ConnectionCheckedIn",
		}
		assertEvents(mt, want, ignore, tpm.Events())
	})
}

// assertEvents asserts that the events slice contains events equivalent to the events in
// wantEvents, ignoring any event types in ignoreEventTypes.
func assertEvents(mt *mtest.T, wantEvents []*event.PoolEvent, ignoreEventTypes []string, events []*event.PoolEvent) {
	ignore := make(map[string]bool, len(ignoreEventTypes))
	for _, typ := range ignoreEventTypes {
		ignore[typ] = true
	}

	offset := 0
	for i, evt := range events {
		if ignore[evt.Type] {
			continue
		}
		if offset >= len(wantEvents) {
			break
		}

		wantEvent := wantEvents[offset]
		assert.Equalf(mt, wantEvent.Type, evt.Type, "event %d: expected event types to match", i)
		assert.NotEmpty(mt, evt.Address, "event %d: expected event address to not be empty", i)
		if len(wantEvent.Reason) > 0 {
			assert.Equalf(mt, wantEvent.Reason, evt.Reason, "event %d: expected event reasons to match", i)
		}
		if wantEvent.ConnectionID > 0 {
			assert.Equalf(mt, wantEvent.ConnectionID, evt.ConnectionID, "event %d: expected event connection IDs to match", i)
		}
		offset++
	}

	if offset != len(wantEvents) {
		mt.Errorf("missing expected events: %v", wantEvents[offset:])
	}
}
