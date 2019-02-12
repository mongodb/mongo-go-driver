// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
	"go.mongodb.org/mongo-driver/x/network/address"
	"go.mongodb.org/mongo-driver/x/network/command"
	"go.mongodb.org/mongo-driver/x/network/description"
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
