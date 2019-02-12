// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"io"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
	"go.mongodb.org/mongo-driver/x/mongo/driver/uuid"
	"go.mongodb.org/mongo-driver/x/network/command"
	"go.mongodb.org/mongo-driver/x/network/description"
)

func skipIfBelow32(t *testing.T) {
	server, err := testutil.Topology(t).SelectServer(context.Background(), description.WriteSelector())
	if err != nil {
		t.Fatalf("error selecting server: %s", err)
	}
	if server.Description().WireVersion.Max < 4 {
		t.Skip("skipping for legacy servers")
	}
}

func skipIfBelow30(t *testing.T) {
	server, err := testutil.Topology(t).SelectServer(context.Background(), description.WriteSelector())
	if err != nil {
		t.Fatalf("error selecting server: %s", err)
	}
	if server.Description().WireVersion.Max < 3 {
		t.Skip("skipping for legacy servers")
	}
}

func invalidNsCode(code int32) bool {
	return code == 73 || code == 16256 || code == 15918
}

func TestCommandListCollections(t *testing.T) {
	noerr := func(t *testing.T, err error) {
		// t.Helper()
		if err != nil {
			t.Errorf("Unepexted error: %v", err)
			t.FailNow()
		}
	}

	t.Run("InvalidDatabaseName", func(t *testing.T) {
		// 2.6 server doesn't throw error for invalid database name
		skipIfBelow30(t)

		server, err := testutil.Topology(t).SelectServer(context.Background(), description.WriteSelector())
		noerr(t, err)
		conn, err := server.Connection(context.Background())
		noerr(t, err)

		_, err = (&command.ListCollections{}).RoundTrip(context.Background(), server.SelectedDescription(), conn)
		switch errt := err.(type) {
		case command.Error:
			if !invalidNsCode(errt.Code) {
				t.Errorf("Incorrect error code returned from server. got %d", errt.Code)
			}
		case command.QueryFailureError:
			rdr := errt.Response
			v, err := rdr.LookupErr("code")
			noerr(t, err)
			code, ok := v.Int32OK()
			if !ok {
				t.Errorf("Incorrect value for code. It is a BSON %v", v.Type)
			}
			if !invalidNsCode(code) {
				t.Errorf("Incorrect error code returned from server. got %d", code)
			}
		default:
			t.Errorf("Incorrect type of command returned. got %T; want %T or %T", err, command.Error{}, command.QueryFailureError{})
		}
	})
	t.Run("SingleBatch", func(t *testing.T) {
		wc := writeconcern.New(writeconcern.WMajority())
		collOne := testutil.ColName(t)
		collTwo := testutil.ColName(t) + "2"
		collThree := testutil.ColName(t) + "3"
		testutil.DropCollection(t, testutil.DBName(t), collOne)
		testutil.DropCollection(t, testutil.DBName(t), collTwo)
		testutil.DropCollection(t, testutil.DBName(t), collThree)
		testutil.InsertDocs(t, testutil.DBName(t), collOne, wc, bsonx.Doc{{"_id", bsonx.Int32(1)}})
		testutil.InsertDocs(t, testutil.DBName(t), collTwo, wc, bsonx.Doc{{"_id", bsonx.Int32(2)}})
		testutil.InsertDocs(t, testutil.DBName(t), collThree, wc, bsonx.Doc{{"_id", bsonx.Int32(3)}})

		clientID, err := uuid.New()
		noerr(t, err)
		cursor, err := driver.ListCollections(
			context.Background(),
			command.ListCollections{DB: dbName},
			testutil.Topology(t),
			description.WriteSelector(),
			clientID,
			&session.Pool{},
		)
		noerr(t, err)

		names := map[string]bool{}

		for cursor.Next(context.Background()) {
			docs := cursor.Batch()
			var next bsoncore.Document
			for {
				next, err = docs.Next()
				if err == io.EOF {
					break
				}
				noerr(t, err)

				val, err := next.LookupErr("name")
				noerr(t, err)
				if val.Type != bson.TypeString {
					t.Errorf("Incorrect type for 'name'. got %v; want %v", val.Type, bson.TypeString)
					t.FailNow()
				}
				names[val.StringValue()] = true
			}
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
