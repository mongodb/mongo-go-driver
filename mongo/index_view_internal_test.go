// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/options"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/stretchr/testify/require"
)

var seed = time.Now().UnixNano()

type index struct {
	Key  map[string]int
	NS   string
	Name string
}

func getIndexableCollection(t *testing.T) (string, *Collection) {
	atomic.AddInt64(&seed, 1)
	rand.Seed(atomic.LoadInt64(&seed))

	client := createTestClient(t)
	db := client.Database("IndexView")

	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	require.NoError(t, err)

	dbName := hex.EncodeToString(randomBytes)

	_, err = db.RunCommand(
		context.Background(),
		bson.NewDocument(
			bson.EC.String("create", dbName),
		),
	)
	require.NoError(t, err)

	return dbName, db.Collection(dbName)
}

func TestIndexView_List(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip()
	}

	dbName, coll := getIndexableCollection(t)
	expectedNS := fmt.Sprintf("IndexView.%s", dbName)
	indexView := coll.Indexes()

	cursor, err := indexView.List(context.Background())
	require.NoError(t, err)

	found := false
	var index = index{}

	for cursor.Next(context.Background()) {
		err := cursor.Decode(&index)
		require.NoError(t, err)

		require.Equal(t, expectedNS, index.NS)

		if index.Name == "_id_" {
			require.Len(t, index.Key, 1)
			require.Equal(t, 1, index.Key["_id"])
			found = true
		}
	}
	require.NoError(t, cursor.Err())
	require.True(t, found)
}

func TestIndexView_CreateOne(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip()
	}

	dbName, coll := getIndexableCollection(t)
	expectedNS := fmt.Sprintf("IndexView.%s", dbName)
	indexView := coll.Indexes()

	indexName, err := indexView.CreateOne(
		context.Background(),
		IndexModel{
			Keys: bson.NewDocument(
				bson.EC.Int32("foo", -1),
			),
		},
	)
	require.NoError(t, err)

	cursor, err := indexView.List(context.Background())
	require.NoError(t, err)

	found := false
	var index = index{}

	for cursor.Next(context.Background()) {
		err := cursor.Decode(&index)
		require.NoError(t, err)

		require.Equal(t, expectedNS, index.NS)

		if index.Name == indexName {
			require.Len(t, index.Key, 1)
			require.Equal(t, -1, index.Key["foo"])
			found = true
		}
	}
	require.NoError(t, cursor.Err())
	require.True(t, found)
}

func TestIndexView_CreateMany(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip()
	}

	dbName, coll := getIndexableCollection(t)
	expectedNS := fmt.Sprintf("IndexView.%s", dbName)
	indexView := coll.Indexes()

	indexNames, err := indexView.CreateMany(
		context.Background(),
		[]options.CreateIndexesOptioner{},
		IndexModel{
			Keys: bson.NewDocument(
				bson.EC.Int32("foo", -1),
			),
		},
		IndexModel{
			Keys: bson.NewDocument(
				bson.EC.Int32("bar", 1),
				bson.EC.Int32("baz", -1),
			),
		},
	)
	require.NoError(t, err)

	require.Len(t, indexNames, 2)

	fooName := indexNames[0]
	barBazName := indexNames[1]

	cursor, err := indexView.List(context.Background())
	require.NoError(t, err)

	fooFound := false
	barBazFound := false
	var index = index{}

	for cursor.Next(context.Background()) {
		err := cursor.Decode(&index)
		require.NoError(t, err)

		require.Equal(t, expectedNS, index.NS)

		if index.Name == fooName {
			require.Len(t, index.Key, 1)
			require.Equal(t, -1, index.Key["foo"])
			fooFound = true
		}

		if index.Name == barBazName {
			require.Len(t, index.Key, 2)
			require.Equal(t, 1, index.Key["bar"])
			require.Equal(t, -1, index.Key["baz"])
			barBazFound = true
		}
	}
	require.NoError(t, cursor.Err())
	require.True(t, fooFound)
	require.True(t, barBazFound)
}

func TestIndexView_DropOne(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip()
	}

	dbName, coll := getIndexableCollection(t)
	expectedNS := fmt.Sprintf("IndexView.%s", dbName)
	indexView := coll.Indexes()

	indexNames, err := indexView.CreateMany(
		context.Background(),
		[]options.CreateIndexesOptioner{},
		IndexModel{
			Keys: bson.NewDocument(
				bson.EC.Int32("foo", -1),
			),
		},
		IndexModel{
			Keys: bson.NewDocument(
				bson.EC.Int32("bar", 1),
				bson.EC.Int32("baz", -1),
			),
		},
	)
	require.NoError(t, err)

	require.Len(t, indexNames, 2)

	_, err = indexView.DropOne(
		context.Background(),
		indexNames[1],
	)
	require.NoError(t, err)

	cursor, err := indexView.List(context.Background())
	require.NoError(t, err)

	var index = index{}

	for cursor.Next(context.Background()) {
		err := cursor.Decode(&index)
		require.NoError(t, err)

		require.Equal(t, expectedNS, index.NS)
		require.NotEqual(t, indexNames[1], index.Name)
	}
	require.NoError(t, cursor.Err())
}

func TestIndexView_DropAll(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip()
	}

	dbName, coll := getIndexableCollection(t)
	expectedNS := fmt.Sprintf("IndexView.%s", dbName)
	indexView := coll.Indexes()

	indexNames, err := indexView.CreateMany(
		context.Background(),
		[]options.CreateIndexesOptioner{},
		IndexModel{
			Keys: bson.NewDocument(
				bson.EC.Int32("foo", -1),
			),
		},
		IndexModel{
			Keys: bson.NewDocument(
				bson.EC.Int32("bar", 1),
				bson.EC.Int32("baz", -1),
			),
		},
	)
	require.NoError(t, err)

	require.Len(t, indexNames, 2)

	_, err = indexView.DropAll(
		context.Background(),
	)
	require.NoError(t, err)

	cursor, err := indexView.List(context.Background())
	require.NoError(t, err)

	var index = index{}

	for cursor.Next(context.Background()) {
		err := cursor.Decode(&index)
		require.NoError(t, err)

		require.Equal(t, expectedNS, index.NS)
		require.NotEqual(t, indexNames[0], index.Name)
		require.NotEqual(t, indexNames[1], index.Name)
	}
	require.NoError(t, cursor.Err())
}

func TestIndexView_CreateIndexesOptioner(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip()
	}

	dbName, coll := getIndexableCollection(t)
	expectedNS := fmt.Sprintf("IndexView.%s", dbName)
	indexView := coll.Indexes()
	var opts []options.CreateIndexesOptioner
	wc := writeconcern.New(writeconcern.W(1))
	elem, err := wc.MarshalBSONElement()
	require.NoError(t, err)
	optwc := options.OptWriteConcern{WriteConcern: elem, Acknowledged: wc.Acknowledged()}
	opts = append(opts, optwc)
	indexNames, err := indexView.CreateMany(
		context.Background(),
		opts,
		IndexModel{
			Keys: bson.NewDocument(
				bson.EC.Int32("foo", -1),
			),
		},
		IndexModel{
			Keys: bson.NewDocument(
				bson.EC.Int32("bar", 1),
				bson.EC.Int32("baz", -1),
			),
		},
	)
	require.NoError(t, err)
	require.NoError(t, err)

	require.Len(t, indexNames, 2)

	fooName := indexNames[0]
	barBazName := indexNames[1]

	cursor, err := indexView.List(context.Background())
	require.NoError(t, err)

	fooFound := false
	barBazFound := false
	var index = index{}

	for cursor.Next(context.Background()) {
		err := cursor.Decode(&index)
		require.NoError(t, err)

		require.Equal(t, expectedNS, index.NS)

		if index.Name == fooName {
			require.Len(t, index.Key, 1)
			require.Equal(t, -1, index.Key["foo"])
			fooFound = true
		}

		if index.Name == barBazName {
			require.Len(t, index.Key, 2)
			require.Equal(t, 1, index.Key["bar"])
			require.Equal(t, -1, index.Key["baz"])
			barBazFound = true
		}
	}
	require.NoError(t, cursor.Err())
	require.True(t, fooFound)
	require.True(t, barBazFound)
	defer func() {
		_, err := indexView.DropAll(context.Background())
		require.NoError(t, err)
	}()
}

func TestIndexView_DropIndexesOptioner(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip()
	}

	dbName, coll := getIndexableCollection(t)
	expectedNS := fmt.Sprintf("IndexView.%s", dbName)
	indexView := coll.Indexes()
	var opts []options.DropIndexesOptioner
	wc := writeconcern.New(writeconcern.W(1))
	elem, err := wc.MarshalBSONElement()
	require.NoError(t, err)
	optwc := options.OptWriteConcern{WriteConcern: elem, Acknowledged: wc.Acknowledged()}
	opts = append(opts, optwc)
	indexNames, err := indexView.CreateMany(
		context.Background(),
		[]options.CreateIndexesOptioner{},
		IndexModel{
			Keys: bson.NewDocument(
				bson.EC.Int32("foo", -1),
			),
		},
		IndexModel{
			Keys: bson.NewDocument(
				bson.EC.Int32("bar", 1),
				bson.EC.Int32("baz", -1),
			),
		},
	)
	require.NoError(t, err)

	require.Len(t, indexNames, 2)

	_, err = indexView.DropAll(
		context.Background(),
		opts...,
	)
	require.NoError(t, err)

	cursor, err := indexView.List(context.Background())
	require.NoError(t, err)

	var index = index{}

	for cursor.Next(context.Background()) {
		err := cursor.Decode(&index)
		require.NoError(t, err)
		require.Equal(t, expectedNS, index.NS)
		require.NotEqual(t, indexNames[0], index.Name)
		require.NotEqual(t, indexNames[1], index.Name)
	}
	require.NoError(t, cursor.Err())
}
