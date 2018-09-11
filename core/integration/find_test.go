// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/event"
	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/internal/testutil"
	"github.com/stretchr/testify/assert"
)

var startedChan = make(chan *event.CommandStartedEvent, 10)
var succeededChan = make(chan *event.CommandSucceededEvent, 10)
var failedChan = make(chan *event.CommandFailedEvent, 10)
var cursorID int64

var monitor = &event.CommandMonitor{
	Started: func(ctx context.Context, cse *event.CommandStartedEvent) {
		startedChan <- cse
	},
	Succeeded: func(ctx context.Context, cse *event.CommandSucceededEvent) {
		succeededChan <- cse
	},
	Failed: func(ctx context.Context, cfe *event.CommandFailedEvent) {
		failedChan <- cfe
	},
}

func drainChannels() {
	for len(startedChan) > 0 {
		<-startedChan
	}

	for len(succeededChan) > 0 {
		<-succeededChan
	}

	for len(failedChan) > 0 {
		<-failedChan
	}
}

func TestFindPassesMaxAwaitTimeMSThroughToGetMore(t *testing.T) {
	server, err := testutil.MonitoredTopology(t, monitor).SelectServer(context.Background(), description.WriteSelector())
	noerr(t, err)

	colName := testutil.ColName(t)

	// create capped collection
	createCmd := bson.NewDocument(
		bson.EC.String("create", colName),
		bson.EC.Boolean("capped", true),
		bson.EC.Int32("size", 1000))
	_, err = testutil.RunCommand(t, server.Server, dbName, createCmd)
	noerr(t, err)

	// insert some documents
	insertCmd := bson.NewDocument(
		bson.EC.String("insert", colName),
		bson.EC.ArrayFromElements("documents",
			bson.VC.Document(bson.NewDocument(bson.EC.Int32("_id", 1))),
			bson.VC.Document(bson.NewDocument(bson.EC.Int32("_id", 2))),
			bson.VC.Document(bson.NewDocument(bson.EC.Int32("_id", 3))),
			bson.VC.Document(bson.NewDocument(bson.EC.Int32("_id", 4))),
			bson.VC.Document(bson.NewDocument(bson.EC.Int32("_id", 5)))))
	_, err = testutil.RunCommand(t, server.Server, dbName, insertCmd)

	conn, err := server.Connection(context.Background())
	noerr(t, err)

	// ignore all previous commands
	drainChannels()

	// find those documents, setting cursor type to TAILABLEAWAIT
	cursor, err := (&command.Find{
		NS:     command.Namespace{DB: dbName, Collection: colName},
		Filter: bson.NewDocument(bson.EC.SubDocument("_id", bson.NewDocument(bson.EC.Int32("$gte", 1)))),
		Opts: []option.FindOptioner{
			option.OptBatchSize(3),
			option.OptMaxAwaitTime(time.Millisecond * 50),
			option.OptCursorType(option.TailableAwait)},
	}).RoundTrip(context.Background(), server.SelectedDescription(), server, conn)
	noerr(t, err)

	// exhaust the cursor, triggering a getMore command
	for cursor.Next(context.Background()) {
	}

	// check that the Find command was started and sent the correct options (not maxAwaitTimeMS)
	started := <-startedChan
	assert.Equal(t, "find", started.CommandName)
	assert.Equal(t, 3, int(started.Command.Lookup("batchSize").Int32()))
	assert.True(t, started.Command.Lookup("tailable").Boolean())
	assert.True(t, started.Command.Lookup("awaitData").Boolean())
	assert.Nil(t, started.Command.Lookup("maxAwaitTimeMS"),
		"Should not have sent maxAwaitTimeMS in find command")

	// check that the Find command succeeded and returned the correct first batch
	succeeded := <-succeededChan
	assert.Equal(t, "find", succeeded.CommandName)
	assert.Equal(t, 1, int(succeeded.Reply.Lookup("ok").Double()))

	actual := succeeded.Reply.Lookup("cursor", "firstBatch").MutableArray()

	if actual.Len() != 3 {
		t.Errorf("incorrect firstBatch returned for find")
	}

	for i := 0; i < 3; i++ {
		v, _ := actual.Lookup(uint(i))
		assert.Equal(t, i+1, int(v.MutableDocument().Lookup("_id").Int32()))
	}

	// check that the getMore command started and sent the correct options (maxTimeMS)
	started = <-startedChan
	assert.Equal(t, "getMore", started.CommandName)
	assert.Equal(t, 3, int(started.Command.Lookup("batchSize").Int32()))
	assert.Equal(t, 50, int(started.Command.Lookup("maxTimeMS").Int64()),
		"Should have sent maxTimeMS in getMore command")

	// check that the getMore command succeeded and returned the correct batch
	succeeded = <-succeededChan
	assert.Equal(t, "getMore", succeeded.CommandName)
	assert.Equal(t, 1, int(succeeded.Reply.Lookup("ok").Double()))

	actual = succeeded.Reply.Lookup("cursor", "nextBatch").MutableArray()

	if actual.Len() != 2 {
		t.Errorf("incorrect nextBatch returned for getMore")
	}

	for i := 0; i < 2; i++ {
		v, _ := actual.Lookup(uint(i))
		assert.Equal(t, i+4, int(v.MutableDocument().Lookup("_id").Int32()))
	}
}
