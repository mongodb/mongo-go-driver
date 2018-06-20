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
	"github.com/mongodb/mongo-go-driver/mongo/indexopt"
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

	var found bool
	var idx index

	for cursor.Next(context.Background()) {
		err := cursor.Decode(&idx)
		require.NoError(t, err)

		require.Equal(t, expectedNS, idx.NS)

		if idx.Name == "_id_" {
			require.Len(t, idx.Key, 1)
			require.Equal(t, 1, idx.Key["_id"])
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

	var found bool
	var idx index

	for cursor.Next(context.Background()) {
		err := cursor.Decode(&idx)
		require.NoError(t, err)

		require.Equal(t, expectedNS, idx.NS)

		if idx.Name == indexName {
			require.Len(t, idx.Key, 1)
			require.Equal(t, -1, idx.Key["foo"])
			found = true
		}
	}
	require.NoError(t, cursor.Err())
	require.True(t, found)
}

func TestIndexView_CreateOneWithNameOption(t *testing.T) {
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
			Options: NewIndexOptionsBuilder().Name("testname").Build(),
		},
	)
	require.NoError(t, err)
	require.Equal(t, "testname", indexName)

	cursor, err := indexView.List(context.Background())
	require.NoError(t, err)

	var found bool
	var idx index

	for cursor.Next(context.Background()) {
		err := cursor.Decode(&idx)
		require.NoError(t, err)

		require.Equal(t, expectedNS, idx.NS)

		if idx.Name == indexName {
			require.Len(t, idx.Key, 1)
			require.Equal(t, -1, idx.Key["foo"])
			found = true
		}
	}
	require.NoError(t, cursor.Err())
	require.True(t, found)
}

// Omits collation option because it's incompatible with version option
func TestIndexView_CreateOneWithAllOptions(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip()
	}

	_, coll := getIndexableCollection(t)
	indexView := coll.Indexes()

	_, err := indexView.CreateOne(
		context.Background(),
		IndexModel{
			Keys: bson.NewDocument(
				bson.EC.String("foo", "text"),
			),
			Options: NewIndexOptionsBuilder().
				Background(false).
				ExpireAfterSeconds(10).
				Name("a").
				Sparse(false).
				Unique(false).
				Version(1).
				DefaultLanguage("english").
				LanguageOverride("english").
				TextVersion(1).
				Weights(bson.NewDocument()).
				SphereVersion(1).
				Bits(32).
				Max(10).
				Min(1).
				BucketSize(1).
				PartialFilterExpression(bson.NewDocument()).
				StorageEngine(bson.NewDocument(
					bson.EC.SubDocument("wiredTiger", bson.NewDocument(
						bson.EC.String("configString", "block_compressor=zlib"),
					)),
				)).
				Build(),
		},
	)
	require.NoError(t, err)
}

func TestIndexView_CreateOneWithCollationOption(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip()
	}

	_, coll := getIndexableCollection(t)
	indexView := coll.Indexes()

	_, err := indexView.CreateOne(
		context.Background(),
		IndexModel{
			Keys: bson.NewDocument(
				bson.EC.String("bar", "text"),
			),
			Options: NewIndexOptionsBuilder().
				Collation(bson.NewDocument(
					bson.EC.String("locale", "simple"),
				)).
				Build(),
		},
	)
	require.NoError(t, err)
}

func TestIndexView_CreateOneWithNilKeys(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip()
	}

	_, coll := getIndexableCollection(t)
	indexView := coll.Indexes()

	_, err := indexView.CreateOne(
		context.Background(),
		IndexModel{
			Keys: nil,
		},
	)
	require.Error(t, err)
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
		[]IndexModel{
			{
				Keys: bson.NewDocument(
					bson.EC.Int32("foo", -1),
				),
			},
			{
				Keys: bson.NewDocument(
					bson.EC.Int32("bar", 1),
					bson.EC.Int32("baz", -1),
				),
			},
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
	var idx index

	for cursor.Next(context.Background()) {
		err := cursor.Decode(&idx)
		require.NoError(t, err)

		require.Equal(t, expectedNS, idx.NS)

		if idx.Name == fooName {
			require.Len(t, idx.Key, 1)
			require.Equal(t, -1, idx.Key["foo"])
			fooFound = true
		}

		if idx.Name == barBazName {
			require.Len(t, idx.Key, 2)
			require.Equal(t, 1, idx.Key["bar"])
			require.Equal(t, -1, idx.Key["baz"])
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
		[]IndexModel{
			{
				Keys: bson.NewDocument(
					bson.EC.Int32("foo", -1),
				),
			},
			{
				Keys: bson.NewDocument(
					bson.EC.Int32("bar", 1),
					bson.EC.Int32("baz", -1),
				),
			},
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

	var idx index

	for cursor.Next(context.Background()) {
		err := cursor.Decode(&idx)
		require.NoError(t, err)

		require.Equal(t, expectedNS, idx.NS)
		require.NotEqual(t, indexNames[1], idx.Name)
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
		[]IndexModel{
			{
				Keys: bson.NewDocument(
					bson.EC.Int32("foo", -1),
				),
			},
			{
				Keys: bson.NewDocument(
					bson.EC.Int32("bar", 1),
					bson.EC.Int32("baz", -1),
				),
			},
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

	var idx index

	for cursor.Next(context.Background()) {
		err := cursor.Decode(&idx)
		require.NoError(t, err)

		require.Equal(t, expectedNS, idx.NS)
		require.NotEqual(t, indexNames[0], idx.Name)
		require.NotEqual(t, indexNames[1], idx.Name)
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

	var opts []indexopt.Create
	optMax := indexopt.MaxTime(1000)
	opts = append(opts, optMax)

	indexNames, err := indexView.CreateMany(
		context.Background(),
		[]IndexModel{
			{
				Keys: bson.NewDocument(
					bson.EC.Int32("foo", -1),
				),
			},
			{
				Keys: bson.NewDocument(
					bson.EC.Int32("bar", 1),
					bson.EC.Int32("baz", -1),
				),
			},
		},
		opts...,
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
	var idx index

	for cursor.Next(context.Background()) {
		err := cursor.Decode(&idx)
		require.NoError(t, err)

		require.Equal(t, expectedNS, idx.NS)

		if idx.Name == fooName {
			require.Len(t, idx.Key, 1)
			require.Equal(t, -1, idx.Key["foo"])
			fooFound = true
		}

		if idx.Name == barBazName {
			require.Len(t, idx.Key, 2)
			require.Equal(t, 1, idx.Key["bar"])
			require.Equal(t, -1, idx.Key["baz"])
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

	var opts []indexopt.Drop
	optMax := indexopt.MaxTime(1000)
	opts = append(opts, optMax)

	indexNames, err := indexView.CreateMany(
		context.Background(),
		[]IndexModel{
			{
				Keys: bson.NewDocument(
					bson.EC.Int32("foo", -1),
				),
			},
			{
				Keys: bson.NewDocument(
					bson.EC.Int32("bar", 1),
					bson.EC.Int32("baz", -1),
				),
			},
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

	var idx index

	for cursor.Next(context.Background()) {
		err := cursor.Decode(&idx)
		require.NoError(t, err)
		require.Equal(t, expectedNS, idx.NS)
		require.NotEqual(t, indexNames[0], idx.Name)
		require.NotEqual(t, indexNames[1], idx.Name)
	}
	require.NoError(t, cursor.Err())
}
