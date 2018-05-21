// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/mongodb/mongo-go-driver/internal/testutil"
)

func TestCommandListCollections(t *testing.T) {
	noerr := func(t *testing.T, err error) {
		// t.Helper()
		if err != nil {
			t.Errorf("Unepexted error: %v", err)
			t.FailNow()
		}
	}
	t.Run("InvalidDatabaseName", func(t *testing.T) {
		server, err := testutil.Topology(t).SelectServer(context.Background(), description.WriteSelector())
		noerr(t, err)
		conn, err := server.Connection(context.Background())
		noerr(t, err)

		_, err = (&command.ListCollections{}).RoundTrip(context.Background(), server.SelectedDescription(), server, conn)
		switch errt := err.(type) {
		case command.Error:
			if errt.Code != 73 {
				t.Errorf("Incorrect error code returned from server. got %d; want %d", errt.Code, 73)
			}
		case command.QueryFailureError:
			rdr := errt.Response
			v, err := rdr.Lookup("code")
			noerr(t, err)
			code, ok := v.Value().Int32OK()
			if !ok {
				t.Errorf("Incorrect value for code. It is a BSON %v", v.Value().Type())
			}
			if code != 73 {
				t.Errorf("Incorrect error code returned from server. got %d; want %d", code, 73)
			}
		default:
			t.Errorf("Incorrect type of command returned. got %T; want %T or %T", err, command.Error{}, command.QueryFailureError{})
		}
	})
	t.Run("SingleBatch", func(t *testing.T) {
		server, err := testutil.Topology(t).SelectServer(context.Background(), description.WriteSelector())
		noerr(t, err)
		conn, err := server.Connection(context.Background())
		noerr(t, err)

		wc := writeconcern.New(writeconcern.WMajority())
		collOne := testutil.ColName(t)
		collTwo := testutil.ColName(t) + "2"
		collThree := testutil.ColName(t) + "3"
		testutil.DropCollection(t, testutil.DBName(t), collOne)
		testutil.DropCollection(t, testutil.DBName(t), collTwo)
		testutil.DropCollection(t, testutil.DBName(t), collThree)
		testutil.InsertDocs(t, testutil.DBName(t), collOne, wc, bson.NewDocument(bson.EC.Int32("_id", 1)))
		testutil.InsertDocs(t, testutil.DBName(t), collTwo, wc, bson.NewDocument(bson.EC.Int32("_id", 2)))
		testutil.InsertDocs(t, testutil.DBName(t), collThree, wc, bson.NewDocument(bson.EC.Int32("_id", 3)))

		cursor, err := (&command.ListCollections{DB: dbName}).RoundTrip(context.Background(), server.SelectedDescription(), server, conn)
		noerr(t, err)

		names := map[string]bool{}
		next := bson.NewDocument()

		for cursor.Next(context.Background()) {
			err = cursor.Decode(next)
			noerr(t, err)

			val, err := next.LookupErr("name")
			noerr(t, err)
			if val.Type() != bson.TypeString {
				t.Errorf("Incorrect type for 'name'. got %v; want %v", val.Type(), bson.TypeString)
				t.FailNow()
			}
			names[val.StringValue()] = true
		}

		for _, required := range []string{collOne, collTwo, collThree} {
			_, ok := names[required]
			if !ok {
				t.Errorf("listCollections command did not return all collections. Missing %s", required)
			}
		}

	})
	t.Run("MultipleBatches", func(t *testing.T) {})
}
