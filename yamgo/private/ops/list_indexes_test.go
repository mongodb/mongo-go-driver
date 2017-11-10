// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops_test

import (
	"context"
	"testing"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/yamgo/internal/testutil"
	. "github.com/10gen/mongo-go-driver/yamgo/private/ops"
	"github.com/stretchr/testify/require"
)

func TestListIndexesWithInvalidDatabaseName(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)

	s := getServer(t)
	ns := Namespace{Collection: "space", DB: "ex"}
	cursor, err := ListIndexes(context.Background(), s, ns, ListIndexesOptions{})
	require.Nil(t, err)

	indexes := []string{}
	var next bson.M

	for cursor.Next(context.Background(), &next) {
		indexes = append(indexes, next["name"].(string))
	}

	require.Equal(t, 0, len(indexes))
}

func TestListIndexesWithInvalidCollectionName(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)

	s := getServer(t)
	ns := Namespace{Collection: testutil.DBName(t), DB: "ex"}
	cursor, err := ListIndexes(context.Background(), s, ns, ListIndexesOptions{})
	require.Nil(t, err)

	indexes := []string{}
	var next bson.M

	for cursor.Next(context.Background(), &next) {
		indexes = append(indexes, next["name"].(string))
	}

	require.Equal(t, 0, len(indexes))
}

func TestListIndexes(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)
	testutil.AutoDropCollection(t)
	testutil.AutoCreateIndex(t, []string{"a"})
	testutil.AutoCreateIndex(t, []string{"b"})
	testutil.AutoCreateIndex(t, []string{"c"})
	testutil.AutoCreateIndex(t, []string{"d", "e"})

	ns := NewNamespace(testutil.DBName(t), testutil.ColName(t))

	s := getServer(t)
	cursor, err := ListIndexes(context.Background(), s, ns, ListIndexesOptions{})
	require.Nil(t, err)

	indexes := []string{}
	var next bson.M

	for cursor.Next(context.Background(), &next) {
		indexes = append(indexes, next["name"].(string))
	}

	expected := []string{"_id_", "a", "b", "c", "d_e"}
	require.Equal(t, 5, len(indexes))
	require.Contains(t, indexes, expected[0])
	require.Contains(t, indexes, expected[1])
	require.Contains(t, indexes, expected[2])
	require.Contains(t, indexes, expected[3])
	require.Contains(t, indexes, expected[4])
}

func TestListIndexesMultipleBatches(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)
	testutil.AutoDropCollection(t)
	testutil.AutoCreateIndex(t, []string{"a"})
	testutil.AutoCreateIndex(t, []string{"b"})
	testutil.AutoCreateIndex(t, []string{"c"})

	s := getServer(t)
	ns := NewNamespace(testutil.DBName(t), testutil.ColName(t))
	options := ListIndexesOptions{BatchSize: 1}
	cursor, err := ListIndexes(context.Background(), s, ns, options)
	require.Nil(t, err)

	indexes := []string{}
	var next bson.M

	for cursor.Next(context.Background(), &next) {
		indexes = append(indexes, next["name"].(string))
	}

	expected := []string{"_id_", "a", "b", "c"}
	require.Equal(t, 4, len(indexes))
	require.Contains(t, indexes, expected[0])
	require.Contains(t, indexes, expected[1])
	require.Contains(t, indexes, expected[2])
	require.Contains(t, indexes, expected[3])
}
