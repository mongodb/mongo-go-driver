// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/internal/testutil"
	. "github.com/mongodb/mongo-go-driver/mongo/private/ops"
	"github.com/stretchr/testify/require"
)

func TestListCollectionsWithInvalidDatabaseName(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)

	s := getServer(t)
	_, err := ListCollections(context.Background(), s, "", ListCollectionsOptions{})
	require.Error(t, err)
}

func TestListCollections(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)
	dbname := testutil.DBName(t)
	collectionNameOne := testutil.ColName(t)
	collectionNameTwo := collectionNameOne + "2"
	collectionNameThree := collectionNameOne + "3"
	testutil.DropCollection(t, dbname, collectionNameOne)
	testutil.DropCollection(t, dbname, collectionNameTwo)
	testutil.DropCollection(t, dbname, collectionNameThree)
	testutil.InsertDocs(t, dbname, collectionNameOne, nil, bson.NewDocument(bson.EC.Int32("_id", 1)))
	testutil.InsertDocs(t, dbname, collectionNameTwo, nil, bson.NewDocument(bson.EC.Int32("_id", 1)))
	testutil.InsertDocs(t, dbname, collectionNameThree, nil, bson.NewDocument(bson.EC.Int32("_id", 1)))

	s := getServer(t)
	cursor, err := ListCollections(context.Background(), s, dbname, ListCollectionsOptions{})
	require.NoError(t, err)
	names := []string{}
	var next = make(bson.Reader, 1024)
	for cursor.Next(context.Background()) {
		err = cursor.Decode(next)
		require.NoError(t, err)

		name, err := next.Lookup("name")
		require.NoError(t, err)
		if name.Value().Type() != bson.TypeString {
			t.Errorf("Expected String but got %s", name.Value().Type())
			t.FailNow()
		}
		names = append(names, name.Value().StringValue())
	}

	require.Contains(t, names, collectionNameOne)
	require.Contains(t, names, collectionNameTwo)
	require.Contains(t, names, collectionNameThree)
}

func TestListCollectionsMultipleBatches(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)
	dbname := testutil.DBName(t)
	collectionNameOne := testutil.ColName(t)
	collectionNameTwo := collectionNameOne + "2"
	collectionNameThree := collectionNameOne + "3"
	testutil.DropCollection(t, dbname, collectionNameOne)
	testutil.DropCollection(t, dbname, collectionNameTwo)
	testutil.DropCollection(t, dbname, collectionNameThree)
	testutil.InsertDocs(t, dbname, collectionNameOne, nil, bson.NewDocument(bson.EC.Int32("_id", 1)))
	testutil.InsertDocs(t, dbname, collectionNameTwo, nil, bson.NewDocument(bson.EC.Int32("_id", 1)))
	testutil.InsertDocs(t, dbname, collectionNameThree, nil, bson.NewDocument(bson.EC.Int32("_id", 1)))

	s := getServer(t)
	cursor, err := ListCollections(context.Background(), s, dbname, ListCollectionsOptions{
		Filter:    bson.NewDocument(bson.EC.Regex("name", fmt.Sprintf("^%s.*", collectionNameOne), "")),
		BatchSize: 2})
	require.NoError(t, err)

	names := []string{}
	var next = make(bson.Reader, 1024)

	for cursor.Next(context.Background()) {
		err = cursor.Decode(next)
		require.NoError(t, err)

		name, err := next.Lookup("name")
		require.NoError(t, err)
		if name.Value().Type() != bson.TypeString {
			t.Errorf("Expected String but got %s", name.Value().Type())
			t.FailNow()
		}
		names = append(names, name.Value().StringValue())
	}
	err = cursor.Err()
	require.NoError(t, err)

	require.Equal(t, 3, len(names))
	require.Contains(t, names, collectionNameOne)
	require.Contains(t, names, collectionNameTwo)
	require.Contains(t, names, collectionNameThree)
}

func TestListCollectionsWithMaxTimeMS(t *testing.T) {
	t.Skip("max time is flaky on the server")
	t.Parallel()
	testutil.Integration(t)

	s := getServer(t)

	if testutil.EnableMaxTimeFailPoint(t, s) != nil {
		t.Skip("skipping maxTimeMS test when max time failpoint is disabled")
	}
	defer testutil.DisableMaxTimeFailPoint(t, s)

	_, err := ListCollections(context.Background(), s, testutil.DBName(t), ListCollectionsOptions{MaxTime: time.Millisecond})
	require.Error(t, err)

	// Hacky check for the error message.  Should we be returning a more structured error?
	require.Contains(t, err.Error(), "operation exceeded time limit")
}
