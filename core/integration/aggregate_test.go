// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/address"
	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/topology"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/mongodb/mongo-go-driver/internal/testutil"
)

func TestCommandAggregate(t *testing.T) {
	noerr := func(t *testing.T, err error) {
		// t.Helper()
		if err != nil {
			t.Errorf("Unepexted error: %v", err)
			t.FailNow()
		}
	}
	t.Run("Invalid Namespace", func(t *testing.T) {
		_, err := (&command.Aggregate{NS: command.Namespace{}}).Encode(description.SelectedServer{})
		if err.Error() != "database name cannot be empty" {
			t.Errorf("Expected namespace error, but got %v", err)
		}
	})
	t.Run("Multiple Batches", func(t *testing.T) {
		server, err := testutil.Topology(t).SelectServer(context.Background(), description.WriteSelector())
		noerr(t, err)
		conn, err := server.Connection(context.Background())
		noerr(t, err)
		ds := []*bson.Document{
			bson.NewDocument(bson.EC.Int32("_id", 1)),
			bson.NewDocument(bson.EC.Int32("_id", 2)),
			bson.NewDocument(bson.EC.Int32("_id", 3)),
			bson.NewDocument(bson.EC.Int32("_id", 4)),
			bson.NewDocument(bson.EC.Int32("_id", 5)),
		}
		wc := writeconcern.New(writeconcern.WMajority())
		testutil.AutoInsertDocs(t, wc, ds...)

		readers := make([]bson.Reader, 0, len(ds))
		for _, doc := range ds {
			r, err := doc.MarshalBSON()
			noerr(t, err)
			readers = append(readers, r)
		}
		cursor, err := (&command.Aggregate{
			NS: command.Namespace{DB: dbName, Collection: testutil.ColName(t)},
			Pipeline: bson.NewArray(
				bson.VC.Document(bson.NewDocument(
					bson.EC.SubDocument("$match", bson.NewDocument(
						bson.EC.SubDocument("_id", bson.NewDocument(bson.EC.Int32("$gt", 2))),
					))),
				),
				bson.VC.Document(bson.NewDocument(bson.EC.SubDocument("$sort", bson.NewDocument(bson.EC.Int32("_id", -1))))),
			),
			Opts: []option.AggregateOptioner{option.OptBatchSize(2)},
		}).RoundTrip(context.Background(), server.SelectedDescription(), server, conn)
		noerr(t, err)

		var next = make(bson.Reader, 1024)

		for i := 4; i > 1; i-- {
			if !cursor.Next(context.Background()) {
				t.Error("Cursor should have results, but does not have a next result")
			}
			err = cursor.Decode(next)
			noerr(t, err)
			if !bytes.Equal(next[:len(readers[i])], readers[i]) {
				t.Errorf("Did not get expected document. got %v; want %v", bson.Reader(next[:len(readers[i])]), readers[i])
			}
		}

		if cursor.Next(context.Background()) {
			t.Error("Cursor should be exhausted but has more results")
		}
	})
	t.Run("AllowDiskUse", func(t *testing.T) {
		server, err := testutil.Topology(t).SelectServer(context.Background(), description.WriteSelector())
		noerr(t, err)
		conn, err := server.Connection(context.Background())
		noerr(t, err)
		ds := []*bson.Document{
			bson.NewDocument(bson.EC.Int32("_id", 1)),
			bson.NewDocument(bson.EC.Int32("_id", 2)),
		}
		wc := writeconcern.New(writeconcern.WMajority())
		testutil.AutoInsertDocs(t, wc, ds...)

		readers := make([]bson.Reader, 0, len(ds))
		for _, doc := range ds {
			r, err := doc.MarshalBSON()
			noerr(t, err)
			readers = append(readers, r)
		}
		_, err = (&command.Aggregate{
			NS:       command.Namespace{DB: dbName, Collection: testutil.ColName(t)},
			Pipeline: bson.NewArray(),
			Opts:     []option.AggregateOptioner{option.OptAllowDiskUse(true)},
		}).RoundTrip(context.Background(), server.SelectedDescription(), server, conn)
		if err != nil {
			t.Errorf("Expected no error from allowing disk use, but got %v", err)
		}
	})
	t.Run("MaxTimeMS", func(t *testing.T) {
		t.Skip("max time is flaky on the server")

		server, err := topology.ConnectServer(context.Background(), address.Address(*host))
		noerr(t, err)
		conn, err := server.Connection(context.Background())
		noerr(t, err)

		_, err = (&command.Write{
			DB: "admin",
			Command: bson.NewDocument(
				bson.EC.String("configureFailPoint", "maxTimeAlwaysTimeOut"),
				bson.EC.String("mode", "alwaysOn"),
			),
		}).RoundTrip(context.Background(), server.SelectedDescription(), conn)
		noerr(t, err)

		_, err = (&command.Aggregate{
			NS:       command.Namespace{DB: dbName, Collection: testutil.ColName(t)},
			Pipeline: bson.NewArray(),
			Opts:     []option.AggregateOptioner{option.OptMaxTime(time.Millisecond)},
		}).RoundTrip(context.Background(), server.SelectedDescription(), server, conn)
		if !strings.Contains(err.Error(), "operation exceeded time limit") {
			t.Errorf("Expected time limit exceeded error, but got %v", err)
		}

		_, err = (&command.Write{
			DB: "admin",
			Command: bson.NewDocument(
				bson.EC.String("configureFailPoint", "maxTimeAlwaysTimeOut"),
				bson.EC.String("mode", "off"),
			),
		}).RoundTrip(context.Background(), server.SelectedDescription(), conn)
		noerr(t, err)
	})
}

func TestAggregatePassesMaxAwaitTimeMSThroughToGetMore(t *testing.T) {
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

	conn, err := server.Connection(context.Background())
	noerr(t, err)

	// ignore all previous commands
	drainChannels()

	// create an aggregate command that results with a TAILABLEAWAIT cursor
	cursor, err := (&command.Aggregate{
		NS: command.Namespace{DB: dbName, Collection: testutil.ColName(t)},
		Pipeline: bson.NewArray(
			bson.VC.Document(bson.NewDocument(
				bson.EC.SubDocument("$changeStream", bson.NewDocument()))),
			bson.VC.Document(bson.NewDocument(
				bson.EC.SubDocument("$match", bson.NewDocument(
					bson.EC.SubDocument("fullDocument._id", bson.NewDocument(bson.EC.Int32("$gte", 1))),
				))))),
		Opts: []option.AggregateOptioner{option.OptBatchSize(1), option.OptMaxAwaitTime(time.Millisecond * 50)},
	}).RoundTrip(context.Background(), server.SelectedDescription(), server, conn)
	noerr(t, err)

	// insert some documents
	insertCmd := bson.NewDocument(
		bson.EC.String("insert", colName),
		bson.EC.ArrayFromElements("documents",
			bson.VC.Document(bson.NewDocument(bson.EC.Int32("_id", 1))),
			bson.VC.Document(bson.NewDocument(bson.EC.Int32("_id", 2))),
			bson.VC.Document(bson.NewDocument(bson.EC.Int32("_id", 3)))))
	_, err = testutil.RunCommand(t, server.Server, dbName, insertCmd)

	// exhaust the cursor, triggering getMore commands
	for cursor.Next(context.Background()) {
	}

	// check that the Find command was started and sent the correct options (not maxAwaitTimeMS)
	started := <-startedChan
	assert.Equal(t, "aggregate", started.CommandName)
	assert.Equal(t, 1, int(started.Command.Lookup("cursor", "batchSize").Int32()))
	assert.Nil(t, started.Command.Lookup("maxAwaitTimeMS"),
		"Should not have sent maxAwaitTimeMS in find command")

	// check that the Find command succeeded and returned the correct first batch
	succeeded := <-succeededChan
	assert.Equal(t, "aggregate", succeeded.CommandName)
	assert.Equal(t, 1, int(succeeded.Reply.Lookup("ok").Double()))

	// check the Insert command
	started = <-startedChan
	assert.Equal(t, "insert", started.CommandName)

	succeeded = <-succeededChan
	assert.Equal(t, "insert", succeeded.CommandName)

	// check that the getMore commands
	for i := 0; i < 3; i++ {
		// check that the getMore command started and sent the correction options (maxTimeMS)
		started = <-startedChan
		assert.Equal(t, "getMore", started.CommandName)
		assert.Equal(t, 1, int(started.Command.Lookup("batchSize").Int32()))
		assert.Equal(t, 50, int(started.Command.Lookup("maxTimeMS").Int64()),
			"Should have sent maxTimeMS in getMore command")

		// check that the getMore command succeeded and returned the correct batch
		succeeded = <-succeededChan
		assert.Equal(t, "getMore", succeeded.CommandName)
		assert.Equal(t, 1, int(succeeded.Reply.Lookup("ok").Double()))

		actual := succeeded.Reply.Lookup("cursor", "nextBatch").MutableArray()

		if actual.Len() != 1 {
			t.Errorf("incorrect nextBatch returned for getMore")
		}

		v, _ := actual.Lookup(0)
		assert.Equal(t, i + 1, int(v.MutableDocument().Lookup("fullDocument", "_id").Int32()))
	}
}
