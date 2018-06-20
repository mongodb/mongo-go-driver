// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"bytes"
	"context"
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
