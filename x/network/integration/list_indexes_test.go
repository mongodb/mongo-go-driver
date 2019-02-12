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
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
	"go.mongodb.org/mongo-driver/x/mongo/driver/uuid"
	"go.mongodb.org/mongo-driver/x/network/command"
	"go.mongodb.org/mongo-driver/x/network/description"
)

func runCommand(t *testing.T, cmd command.ListIndexes, opts ...*options.ListIndexesOptions) (*driver.BatchCursor, error) {
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
		skipIfBelow30(t)
		ns := command.Namespace{DB: "ex", Collection: "space"}
		_, err := runCommand(t, command.ListIndexes{NS: ns})
		if err != command.ErrEmptyCursor {
			t.Errorf("Expected to receive empty cursor, but didn't. got %v; want %v", err, command.ErrEmptyCursor)
		}
	})
	t.Run("InvalidCollectionName", func(t *testing.T) {
		skipIfBelow30(t)
		ns := command.Namespace{DB: "ex", Collection: testutil.ColName(t)}
		_, err := runCommand(t, command.ListIndexes{NS: ns})
		if err != command.ErrEmptyCursor {
			t.Errorf("Expected to receive empty cursor, but didn't. got %v; want %v", err, command.ErrEmptyCursor)
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
				indexes = append(indexes, val.StringValue())
			}
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
				indexes = append(indexes, val.StringValue())
			}
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
