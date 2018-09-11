// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"fmt"
	"os"
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

func initMonitor() (chan *event.CommandStartedEvent, chan *event.CommandSucceededEvent, chan *event.CommandFailedEvent, *event.CommandMonitor) {
	startedChan := make(chan *event.CommandStartedEvent, 100)
	succeededChan := make(chan *event.CommandSucceededEvent, 100)
	failedChan := make(chan *event.CommandFailedEvent, 100)
	monitor := &event.CommandMonitor{
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

	return startedChan, succeededChan, failedChan, monitor
}

func TestFindPassesMaxAwaitTimeMSThroughToGetMore(t *testing.T) {
	startedChan, succeededChan, failedChan, monitor := initMonitor()

	dbName := fmt.Sprintf("mongo-go-driver-%d-find", os.Getpid())
	colName := testutil.ColName(t)

	server, err := testutil.MonitoredTopology(t, dbName, monitor).SelectServer(context.Background(), description.WriteSelector())
	noerr(t, err)

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

	// find those documents, setting cursor type to TAILABLEAWAIT
	cursor, err := (&command.Find{
		NS:     command.Namespace{DB: dbName, Collection: colName},
		Filter: bson.NewDocument(bson.EC.SubDocument("_id", bson.NewDocument(bson.EC.Int32("$gte", 1)))),
		Opts: []option.FindOptioner{
			option.OptBatchSize(3),
			option.OptMaxAwaitTime(time.Millisecond * 250),
			option.OptCursorType(option.TailableAwait)},
	}).RoundTrip(context.Background(), server.SelectedDescription(), server, conn)
	noerr(t, err)

	// exhaust the cursor, triggering getMore commands
	for i := 0; i < 4; i++ {
		cursor.Next(context.Background())
	}

	// allow for iteration over range chan
	close(startedChan)
	close(succeededChan)
	close(failedChan)

	// no commands should have failed
	if len(failedChan) != 0 {
		t.Errorf("%d command(s) failed", len(failedChan))
	}

	// check that the expected commands were started
	for started := range startedChan {
		switch started.CommandName {
		case "find":
			assert.Equal(t, 3, int(started.Command.Lookup("batchSize").Int32()))
			assert.True(t, started.Command.Lookup("tailable").Boolean())
			assert.True(t, started.Command.Lookup("awaitData").Boolean())
			assert.Nil(t, started.Command.Lookup("maxAwaitTimeMS"),
				"Should not have sent maxAwaitTimeMS in find command")
		case "getMore":
			assert.Equal(t, 3, int(started.Command.Lookup("batchSize").Int32()))
			assert.Equal(t, 250, int(started.Command.Lookup("maxTimeMS").Int64()),
				"Should have sent maxTimeMS in getMore command")
		default:
			continue
		}
	}

	// to keep track of seen documents
	id := 1

	// check expected commands succeeded
	for succeeded := range succeededChan {
		switch succeeded.CommandName {
		case "find":
			assert.Equal(t, 1, int(succeeded.Reply.Lookup("ok").Double()))

			actual := succeeded.Reply.Lookup("cursor", "firstBatch").MutableArray()

			for i := 0; i < actual.Len(); i++ {
				v, _ := actual.Lookup(uint(i))
				assert.Equal(t, id, int(v.MutableDocument().Lookup("_id").Int32()))
				id++
			}
		case "getMore":
			assert.Equal(t, "getMore", succeeded.CommandName)
			assert.Equal(t, 1, int(succeeded.Reply.Lookup("ok").Double()))

			actual := succeeded.Reply.Lookup("cursor", "nextBatch").MutableArray()

			for i := 0; i < actual.Len(); i++ {
				v, _ := actual.Lookup(uint(i))
				assert.Equal(t, id, int(v.MutableDocument().Lookup("_id").Int32()))
				id++
			}
		default:
			continue
		}
	}

	if id <= 5 {
		t.Errorf("not all documents returned; last seen id = %d", id-1)
	}
}
