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

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
	"go.mongodb.org/mongo-driver/x/mongo/driver/uuid"
	"go.mongodb.org/mongo-driver/x/network/command"
	"go.mongodb.org/mongo-driver/x/network/description"
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
	skipIfBelow32(t) // maxTimeMS doesn't exist in OP_GET_MORE

	startedChan, succeededChan, failedChan, monitor := initMonitor()

	dbName := fmt.Sprintf("mongo-go-driver-%d-find", os.Getpid())
	colName := testutil.ColName(t)

	topo := testutil.MonitoredTopology(t, dbName, monitor)
	server, err := topo.SelectServer(context.Background(), description.WriteSelector())
	noerr(t, err)

	// create capped collection
	createCmd := bsonx.Doc{
		{"create", bsonx.String(colName)},
		{"capped", bsonx.Boolean(true)},
		{"size", bsonx.Int32(1000)}}
	_, err = testutil.RunCommand(t, server.Server, dbName, createCmd)
	noerr(t, err)

	// insert some documents
	insertCmd := bsonx.Doc{
		{"insert", bsonx.String(colName)},
		{"documents", bsonx.Array(bsonx.Arr{
			bsonx.Document(bsonx.Doc{{"_id", bsonx.Int32(1)}}),
			bsonx.Document(bsonx.Doc{{"_id", bsonx.Int32(2)}}),
			bsonx.Document(bsonx.Doc{{"_id", bsonx.Int32(3)}}),
			bsonx.Document(bsonx.Doc{{"_id", bsonx.Int32(4)}}),
			bsonx.Document(bsonx.Doc{{"_id", bsonx.Int32(5)}})})}}
	_, err = testutil.RunCommand(t, server.Server, dbName, insertCmd)

	// find those documents, setting cursor type to TAILABLEAWAIT
	clientID, err := uuid.New()
	noerr(t, err)
	cursor, err := driver.Find(
		context.Background(),
		command.Find{
			NS:     command.Namespace{DB: dbName, Collection: colName},
			Filter: bsonx.Doc{{"_id", bsonx.Document(bsonx.Doc{{"$gte", bsonx.Int32(1)}})}},
		},
		topo,
		description.WriteSelector(),
		clientID,
		&session.Pool{},
		bson.DefaultRegistry,
		options.Find().SetBatchSize(3).SetCursorType(options.TailableAwait).SetMaxAwaitTime(250*time.Millisecond),
	)

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
			assert.Equal(t, started.Command.Lookup("maxAwaitTimeMS"), bson.RawValue{},
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

			actual, err := succeeded.Reply.Lookup("cursor", "firstBatch").Array().Values()
			assert.NoError(t, err)

			for _, v := range actual {
				assert.Equal(t, id, int(v.Document().Lookup("_id").Int32()))
				id++
			}
		case "getMore":
			assert.Equal(t, "getMore", succeeded.CommandName)
			assert.Equal(t, 1, int(succeeded.Reply.Lookup("ok").Double()))

			actual, err := succeeded.Reply.Lookup("cursor", "nextBatch").Array().Values()
			assert.NoError(t, err)

			for _, v := range actual {
				assert.Equal(t, id, int(v.Document().Lookup("_id").Int32()))
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
