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
	"strings"
	"testing"
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/internal/testutil"
	"github.com/mongodb/mongo-go-driver/internal/testutil/israce"
	"github.com/mongodb/mongo-go-driver/mongo/writeconcern"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
	"github.com/mongodb/mongo-go-driver/x/bsonx/bsoncore"
	"github.com/mongodb/mongo-go-driver/x/mongo/driver"
	"github.com/mongodb/mongo-go-driver/x/mongo/driver/topology"
	"github.com/mongodb/mongo-go-driver/x/network/address"
	"github.com/mongodb/mongo-go-driver/x/network/command"
	"github.com/mongodb/mongo-go-driver/x/network/description"
	"github.com/stretchr/testify/assert"
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
		// TODO(GODRIVER-617): Restore these tests in the driver package.
		// server, err := testutil.Topology(t).SelectServer(context.Background(), description.WriteSelector())
		// noerr(t, err)
		// conn, err := server.Connection(context.Background())
		// noerr(t, err)
		// ds := []bsonx.Doc{
		// 	{{"_id", bsonx.Int32(1)}},
		// 	{{"_id", bsonx.Int32(2)}},
		// 	{{"_id", bsonx.Int32(3)}},
		// 	{{"_id", bsonx.Int32(4)}},
		// 	{{"_id", bsonx.Int32(5)}},
		// }
		// wc := writeconcern.New(writeconcern.WMajority())
		// testutil.AutoInsertDocs(t, wc, ds...)
		//
		// readers := make([]bson.Raw, 0, len(ds))
		// for _, doc := range ds {
		// 	r, err := doc.MarshalBSON()
		// 	noerr(t, err)
		// 	readers = append(readers, r)
		// }
		// cursor, err := (&command.Aggregate{
		// 	NS: command.Namespace{DB: dbName, Collection: testutil.ColName(t)},
		// 	Pipeline: bsonx.Arr{
		// 		bsonx.Document(bsonx.Doc{
		// 			{"$match", bsonx.Document(bsonx.Doc{
		// 				{"_id", bsonx.Document(bsonx.Doc{{"$gt", bsonx.Int32(2)}})},
		// 			})}},
		// 		),
		// 		bsonx.Document(bsonx.Doc{{"$sort", bsonx.Document(bsonx.Doc{{"_id", bsonx.Int32(-1)}})}}),
		// 	},
		// 	Opts: []bsonx.Elem{{"batchSize", bsonx.Int32(2)}},
		// }).RoundTrip(context.Background(), server.SelectedDescription(), server, conn)
		// noerr(t, err)
		//
		// var next bson.Raw
		//
		// for i := 4; i > 1; i-- {
		// 	if !cursor.Next(context.Background()) {
		// 		t.Error("Cursor should have results, but does not have a next result")
		// 	}
		// 	err = cursor.Decode(&next)
		// 	noerr(t, err)
		// 	if !bytes.Equal(next[:len(readers[i])], readers[i]) {
		// 		t.Errorf("Did not get expected document. got %v; want %v", bson.Raw(next[:len(readers[i])]), readers[i])
		// 	}
		// }
		//
		// if cursor.Next(context.Background()) {
		// 	t.Error("Cursor should be exhausted but has more results")
		// }
	})
	t.Run("AllowDiskUse", func(t *testing.T) {
		server, err := testutil.Topology(t).SelectServer(context.Background(), description.WriteSelector())
		noerr(t, err)
		conn, err := server.Connection(context.Background())
		noerr(t, err)
		ds := []bsonx.Doc{
			{{"_id", bsonx.Int32(1)}},
			{{"_id", bsonx.Int32(2)}},
		}
		wc := writeconcern.New(writeconcern.WMajority())
		testutil.AutoInsertDocs(t, wc, ds...)

		readers := make([]bson.Raw, 0, len(ds))
		for _, doc := range ds {
			r, err := doc.MarshalBSON()
			noerr(t, err)
			readers = append(readers, r)
		}
		_, err = (&command.Aggregate{
			NS:       command.Namespace{DB: dbName, Collection: testutil.ColName(t)},
			Pipeline: bsonx.Arr{},
			Opts:     []bsonx.Elem{{"allowDiskUse", bsonx.Boolean(true)}},
		}).RoundTrip(context.Background(), server.SelectedDescription(), conn)
		if err != nil {
			t.Errorf("Expected no error from allowing disk use, but got %v", err)
		}
	})
	t.Run("MaxTime", func(t *testing.T) {
		t.Skip("max time is flaky on the server")

		server, err := topology.ConnectServer(context.Background(), address.Address(*host))
		noerr(t, err)
		conn, err := server.Connection(context.Background())
		noerr(t, err)

		_, err = (&command.Write{
			DB: "admin",
			Command: bsonx.Doc{
				{"configureFailPoint", bsonx.String("maxTimeAlwaysTimeOut")},
				{"mode", bsonx.String("alwaysOn")},
			},
		}).RoundTrip(context.Background(), server.SelectedDescription(), conn)
		noerr(t, err)

		_, err = (&command.Aggregate{
			NS:       command.Namespace{DB: dbName, Collection: testutil.ColName(t)},
			Pipeline: bsonx.Arr{},
			Opts:     []bsonx.Elem{{"maxTimeMS", bsonx.Int64(1)}},
		}).RoundTrip(context.Background(), server.SelectedDescription(), conn)
		if !strings.Contains(err.Error(), "operation exceeded time limit") {
			t.Errorf("Expected time limit exceeded error, but got %v", err)
		}

		_, err = (&command.Write{
			DB: "admin",
			Command: bsonx.Doc{
				{"configureFailPoint", bsonx.String("maxTimeAlwaysTimeOut")},
				{"mode", bsonx.String("off")},
			},
		}).RoundTrip(context.Background(), server.SelectedDescription(), conn)
		noerr(t, err)
	})
}

func TestAggregatePassesMaxAwaitTimeMSThroughToGetMore(t *testing.T) {
	if os.Getenv("TOPOLOGY") != "replica_set" {
		t.Skip()
	}

	startedChan, succeededChan, failedChan, monitor := initMonitor()

	dbName := fmt.Sprintf("mongo-go-driver-%d-agg", os.Getpid())
	colName := testutil.ColName(t)

	server, err := testutil.MonitoredTopology(t, dbName, monitor).SelectServer(context.Background(), description.WriteSelector())
	noerr(t, err)

	versionCmd := bsonx.Doc{{"serverStatus", bsonx.Int32(1)}}
	serverStatus, err := testutil.RunCommand(t, server.Server, dbName, versionCmd)
	version, err := serverStatus.LookupErr("version")

	if compareVersions(t, version.StringValue(), "3.6") < 0 {
		t.Skip()
	}

	// create capped collection
	createCmd := bsonx.Doc{
		{"create", bsonx.String(colName)},
		{"capped", bsonx.Boolean(true)},
		{"size", bsonx.Int32(1000)}}
	_, err = testutil.RunCommand(t, server.Server, dbName, createCmd)
	noerr(t, err)

	conn, err := server.Connection(context.Background())
	noerr(t, err)

	// create an aggregate command that results with a TAILABLEAWAIT cursor
	result, err := (&command.Aggregate{
		NS: command.Namespace{DB: dbName, Collection: testutil.ColName(t)},
		Pipeline: bsonx.Arr{
			bsonx.Document(bsonx.Doc{
				{"$changeStream", bsonx.Document(bsonx.Doc{})}}),
			bsonx.Document(bsonx.Doc{
				{"$match", bsonx.Document(bsonx.Doc{
					{"fullDocument._id", bsonx.Document(bsonx.Doc{{"$gte", bsonx.Int32(1)}})},
				})}})},
		Opts: []bsonx.Elem{{"batchSize", bsonx.Int32(2)}},
		CursorOpts: []bsonx.Elem{
			{"batchSize", bsonx.Int32(2)},
			{"maxTimeMS", bsonx.Int64(50)},
		},
	}).RoundTrip(context.Background(), server.SelectedDescription(), conn)
	noerr(t, err)

	cursor, err := driver.NewBatchCursor(
		bsoncore.Document(result), nil, nil, server.Server,
		bsonx.Elem{"batchSize", bsonx.Int32(2)}, bsonx.Elem{"maxTimeMS", bsonx.Int64(50)},
	)
	noerr(t, err)

	// insert some documents
	insertCmd := bsonx.Doc{
		{"insert", bsonx.String(colName)},
		{"documents", bsonx.Array(bsonx.Arr{
			bsonx.Document(bsonx.Doc{{"_id", bsonx.Int32(1)}}),
			bsonx.Document(bsonx.Doc{{"_id", bsonx.Int32(2)}}),
			bsonx.Document(bsonx.Doc{{"_id", bsonx.Int32(3)}})})}}
	_, err = testutil.RunCommand(t, server.Server, dbName, insertCmd)

	// wait a bit between insert and getMore commands
	time.Sleep(time.Millisecond * 100)
	if israce.Enabled {
		time.Sleep(time.Millisecond * 400) // wait a little longer when race detector is enabled.
	}

	ctx, cancel := context.WithCancel(context.Background())
	if israce.Enabled {
		time.AfterFunc(time.Millisecond*2000, cancel)
	} else {
		time.AfterFunc(time.Millisecond*900, cancel)
	}
	for cursor.Next(ctx) {
	}

	// allow for iteration over range chan
	close(startedChan)
	close(succeededChan)
	close(failedChan)

	// no commands should have failed
	if len(failedChan) != 0 {
		t.Errorf("%d command(s) failed", len(failedChan))
	}

	// check the expected commands were started
	for started := range startedChan {
		switch started.CommandName {
		case "aggregate":
			assert.Equal(t, 2, int(started.Command.Lookup("cursor", "batchSize").Int32()))
			assert.Equal(t, started.Command.Lookup("maxAwaitTimeMS"), bson.RawValue{},
				"Should not have sent maxAwaitTimeMS in find command")
		case "getMore":
			assert.Equal(t, 2, int(started.Command.Lookup("batchSize").Int32()))
			assert.Equal(t, 50, int(started.Command.Lookup("maxTimeMS").Int64()),
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
		case "aggregate":
			assert.Equal(t, 1, int(succeeded.Reply.Lookup("ok").Double()))

			actual, err := succeeded.Reply.Lookup("cursor", "firstBatch").Array().Values()
			assert.NoError(t, err)

			for _, v := range actual {
				assert.Equal(t, id, int(v.Document().Lookup("fullDocument", "_id").Int32()))
				id++
			}
		case "getMore":
			assert.Equal(t, "getMore", succeeded.CommandName)
			assert.Equal(t, 1, int(succeeded.Reply.Lookup("ok").Double()))

			actual, err := succeeded.Reply.Lookup("cursor", "nextBatch").Array().Values()
			assert.NoError(t, err)

			for _, v := range actual {
				assert.Equal(t, id, int(v.Document().Lookup("fullDocument", "_id").Int32()))
				id++
			}
		default:
			continue
		}
	}

	if id <= 3 {
		t.Errorf("not all documents returned; last seen id = %d", id-1)
	}
}
