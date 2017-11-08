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

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/mongo/internal/testutil"
	. "github.com/10gen/mongo-go-driver/mongo/private/ops"
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
	testutil.InsertDocs(t, dbname, collectionNameOne, bson.D{bson.NewDocElem("_id", 1)})
	testutil.InsertDocs(t, dbname, collectionNameTwo, bson.D{bson.NewDocElem("_id", 1)})
	testutil.InsertDocs(t, dbname, collectionNameThree, bson.D{bson.NewDocElem("_id", 1)})

	s := getServer(t)
	cursor, err := ListCollections(context.Background(), s, dbname, ListCollectionsOptions{})
	require.NoError(t, err)
	names := []string{}
	var next bson.M
	for cursor.Next(context.Background(), &next) {
		names = append(names, next["name"].(string))
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
	testutil.InsertDocs(t, dbname, collectionNameOne, bson.D{bson.NewDocElem("_id", 1)})
	testutil.InsertDocs(t, dbname, collectionNameTwo, bson.D{bson.NewDocElem("_id", 1)})
	testutil.InsertDocs(t, dbname, collectionNameThree, bson.D{bson.NewDocElem("_id", 1)})

	s := getServer(t)
	cursor, err := ListCollections(context.Background(), s, dbname, ListCollectionsOptions{
		Filter:    bson.D{bson.NewDocElem("name", bson.RegEx{Pattern: fmt.Sprintf("^%s.*", collectionNameOne)})},
		BatchSize: 2})
	require.NoError(t, err)

	names := []string{}
	var next bson.M

	for cursor.Next(context.Background(), &next) {
		names = append(names, next["name"].(string))
	}

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
