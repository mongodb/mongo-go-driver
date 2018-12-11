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
	"github.com/mongodb/mongo-go-driver/internal/testutil"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
	"github.com/mongodb/mongo-go-driver/x/mongo/driver"
	"github.com/mongodb/mongo-go-driver/x/mongo/driver/session"
	"github.com/mongodb/mongo-go-driver/x/mongo/driver/uuid"
	"github.com/mongodb/mongo-go-driver/x/network/command"
	"github.com/mongodb/mongo-go-driver/x/network/description"
)

func runCommand(t *testing.T, cmd command.ListIndexes, opts ...*options.ListIndexesOptions) (command.Cursor, error) {
	clientID, err := uuid.New()
	noerr(t, err)

	return driver.ListIndexes(
		context.Background(),
		cmd,
		testutil.Topology(t),
		description.WriteSelector(),
		clientID,
		&session.Pool{},
		opts...,
	)
}

func TestCommandListIndexes(t *testing.T) {
	noerr := func(t *testing.T, err error) {
		// t.Helper()
		if err != nil {
			t.Errorf("Unepexted error: %v", err)
			t.FailNow()
		}
	}

	t.Run("InvalidDatabaseName", func(t *testing.T) {
		ns := command.Namespace{DB: "ex", Collection: "space"}
		cursor, err := runCommand(t, command.ListIndexes{NS: ns})
		noerr(t, err)

		indexes := []string{}
		var next bsonx.Doc

		for cursor.Next(context.Background()) {
			err = cursor.Decode(&next)
			noerr(t, err)

			val, err := next.LookupErr("name")
			noerr(t, err)
			if val.Type() != bson.TypeString {
				t.Errorf("Incorrect type for 'name'. got %v; want %v", val.Type(), bson.TypeString)
				t.FailNow()
			}
			indexes = append(indexes, val.StringValue())
		}

		if len(indexes) != 0 {
			t.Errorf("Expected no indexes from invalid database. got %d; want %d", len(indexes), 0)
		}
	})
	t.Run("InvalidCollectionName", func(t *testing.T) {
		ns := command.Namespace{DB: "ex", Collection: testutil.ColName(t)}
		cursor, err := runCommand(t, command.ListIndexes{NS: ns})
		noerr(t, err)

		indexes := []string{}
		var next bsonx.Doc

		for cursor.Next(context.Background()) {
			err = cursor.Decode(&next)
			noerr(t, err)

			val, err := next.LookupErr("name")
			noerr(t, err)
			if val.Type() != bson.TypeString {
				t.Errorf("Incorrect type for 'name'. got %v; want %v", val.Type(), bson.TypeString)
				t.FailNow()
			}
			indexes = append(indexes, val.StringValue())
		}

		if len(indexes) != 0 {
			t.Errorf("Expected no indexes from invalid database. got %d; want %d", len(indexes), 0)
		}
	})
	t.Run("SingleBatch", func(t *testing.T) {
		testutil.AutoDropCollection(t)
		testutil.AutoCreateIndexes(t, []string{"a"})
		testutil.AutoCreateIndexes(t, []string{"b"})
		testutil.AutoCreateIndexes(t, []string{"c"})
		testutil.AutoCreateIndexes(t, []string{"d", "e"})

		ns := command.NewNamespace(dbName, testutil.ColName(t))
		cursor, err := runCommand(t, command.ListIndexes{NS: ns})
		noerr(t, err)

		indexes := []string{}
		var next bsonx.Doc

		for cursor.Next(context.Background()) {
			next = next[:0]
			err = cursor.Decode(&next)
			noerr(t, err)

			val, err := next.LookupErr("name")
			noerr(t, err)
			if val.Type() != bson.TypeString {
				t.Errorf("Incorrect type for 'name'. got %v; want %v", val.Type(), bson.TypeString)
				t.FailNow()
			}
			indexes = append(indexes, val.StringValue())
		}

		if len(indexes) != 5 {
			t.Errorf("Incorrect number of indexes. got %d; want %d", len(indexes), 5)
		}
		for i, want := range []string{"_id_", "a", "b", "c", "d_e"} {
			got := indexes[i]
			if got != want {
				t.Errorf("Mismatched index %d. got %s; want %s", i, got, want)
			}
		}
	})
	t.Run("MultipleBatch", func(t *testing.T) {
		testutil.AutoDropCollection(t)
		testutil.AutoCreateIndexes(t, []string{"a"})
		testutil.AutoCreateIndexes(t, []string{"b"})
		testutil.AutoCreateIndexes(t, []string{"c"})

		ns := command.NewNamespace(dbName, testutil.ColName(t))
		cursor, err := runCommand(t, command.ListIndexes{NS: ns}, options.ListIndexes().SetBatchSize(1))
		noerr(t, err)

		indexes := []string{}
		var next bsonx.Doc

		for cursor.Next(context.Background()) {
			next = next[:0]
			err = cursor.Decode(&next)
			noerr(t, err)

			val, err := next.LookupErr("name")
			noerr(t, err)
			if val.Type() != bson.TypeString {
				t.Errorf("Incorrect type for 'name'. got %v; want %v", val.Type(), bson.TypeString)
				t.FailNow()
			}
			indexes = append(indexes, val.StringValue())
		}

		if len(indexes) != 4 {
			t.Errorf("Incorrect number of indexes. got %d; want %d", len(indexes), 5)
		}
		for i, want := range []string{"_id_", "a", "b", "c"} {
			got := indexes[i]
			if got != want {
				t.Errorf("Mismatched index %d. got %s; want %s", i, got, want)
			}
		}
	})
}
