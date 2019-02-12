// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
	"go.mongodb.org/mongo-driver/x/mongo/driver/uuid"
	"go.mongodb.org/mongo-driver/x/network/command"
	"go.mongodb.org/mongo-driver/x/network/description"
	"testing"
	"time"
)

func setUpMonitor() (*event.CommandMonitor, chan *event.CommandStartedEvent, chan *event.CommandSucceededEvent, chan *event.CommandFailedEvent) {
	started := make(chan *event.CommandStartedEvent, 1)
	succeeded := make(chan *event.CommandSucceededEvent, 1)
	failed := make(chan *event.CommandFailedEvent, 1)

	return &event.CommandMonitor{
		Started: func(ctx context.Context, e *event.CommandStartedEvent) {
			started <- e
		},
		Succeeded: func(ctx context.Context, e *event.CommandSucceededEvent) {
			succeeded <- e
		},
		Failed: func(ctx context.Context, e *event.CommandFailedEvent) {
			failed <- e
		},
	}, started, succeeded, failed
}

func skipIfBelow32(ctx context.Context, t *testing.T, topo *topology.Topology) {
	server, err := topo.SelectServer(ctx, description.WriteSelector())
	noerr(t, err)

	versionCmd := bsonx.Doc{{"serverStatus", bsonx.Int32(1)}}
	serverStatus, err := testutil.RunCommand(t, server.Server, dbName, versionCmd)
	version, err := serverStatus.LookupErr("version")

	if testutil.CompareVersions(t, version.StringValue(), "3.2") < 0 {
		t.Skip()
	}
}

func TestAggregate(t *testing.T) {
	t.Run("TestMaxTimeMSInGetMore", func(t *testing.T) {
		ctx := context.Background()
		monitor, started, succeeded, failed := setUpMonitor()
		dbName := "TestAggMaxTimeDB"
		collName := "TestAggMaxTimeColl"
		top := testutil.MonitoredTopology(t, dbName, monitor)
		clearChannels(started, succeeded, failed)
		skipIfBelow32(ctx, t, top)

		clientID, err := uuid.New()
		noerr(t, err)

		ns := command.Namespace{
			DB:         dbName,
			Collection: collName,
		}
		pool := &session.Pool{}

		clearChannels(started, succeeded, failed)
		_, err = driver.Insert(
			ctx,
			command.Insert{
				NS: ns,
				Docs: []bsonx.Doc{
					{{"x", bsonx.Int32(1)}},
					{{"x", bsonx.Int32(1)}},
					{{"x", bsonx.Int32(1)}},
				},
			},
			top,
			description.WriteSelector(),
			clientID,
			pool,
			false,
		)
		noerr(t, err)

		clearChannels(started, succeeded, failed)
		cmd := command.Aggregate{
			NS:       ns,
			Pipeline: bsonx.Arr{},
		}
		batchCursor, err := driver.Aggregate(
			ctx,
			cmd,
			top,
			description.ReadPrefSelector(readpref.Primary()),
			description.WriteSelector(),
			clientID,
			pool,
			bson.DefaultRegistry,
			options.Aggregate().SetMaxAwaitTime(10*time.Millisecond).SetBatchSize(2),
		)
		noerr(t, err)

		var e *event.CommandStartedEvent
		select {
		case e = <-started:
		case <-time.After(200 * time.Millisecond):
			t.Fatal("timed out waiting for aggregate")
		}

		require.Equal(t, "aggregate", e.CommandName)

		clearChannels(started, succeeded, failed)
		// first Next() should automatically return true
		require.True(t, batchCursor.Next(ctx), "expected true from first Next, got false")
		clearChannels(started, succeeded, failed)
		batchCursor.Next(ctx) // should do getMore

		select {
		case e = <-started:
		case <-time.After(200 * time.Millisecond):
			t.Fatal("timed out waiting for getMore")
		}
		require.Equal(t, "getMore", e.CommandName)
		_, err = e.Command.LookupErr("maxTimeMS")
		noerr(t, err)
	})
}

func clearChannels(s chan *event.CommandStartedEvent, succ chan *event.CommandSucceededEvent, f chan *event.CommandFailedEvent) {
	for len(s) > 0 {
		<-s
	}
	for len(succ) > 0 {
		<-succ
	}
	for len(f) > 0 {
		<-f
	}
}
