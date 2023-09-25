// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/internal/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestSearchIndexProse(t *testing.T) {
	t.Parallel()

	const timeout = 5 * time.Minute

	uri := os.Getenv("TEST_INDEX_URI")
	if uri == "" {
		t.Skip("skipping")
	}

	opts := options.Client().ApplyURI(uri).SetTimeout(timeout)
	mt := mtest.New(t, mtest.NewOptions().ClientOptions(opts).MinServerVersion("7.0").Topologies(mtest.ReplicaSet))

	mt.Run("case 1: Driver can successfully create and list search indexes", func(mt *mtest.T) {
		ctx := context.Background()

		_, err := mt.Coll.InsertOne(ctx, bson.D{})
		require.NoError(mt, err, "failed to insert")

		view := mt.Coll.SearchIndexes()

		definition := bson.D{{"mappings", bson.D{{"dynamic", false}}}}
		searchName := "test-search-index"
		opts := options.SearchIndexes().SetName(searchName)
		index, err := view.CreateOne(ctx, mongo.SearchIndexModel{
			Definition: definition,
			Options:    opts,
		})
		require.NoError(mt, err, "failed to create index")
		require.Equal(mt, searchName, index, "unmatched name")

		var doc bson.Raw
		for doc == nil {
			cursor, err := view.List(ctx, opts)
			require.NoError(mt, err, "failed to list")

			if !cursor.Next(ctx) {
				break
			}
			if cursor.Current.Lookup("queryable").Boolean() {
				doc = cursor.Current
			} else {
				t.Logf("cursor: %s, sleep 5 seconds...", cursor.Current.String())
				time.Sleep(5 * time.Second)
			}
		}
		require.NotNil(mt, doc, "got empty document")
		assert.Equal(mt, searchName, doc.Lookup("name").StringValue(), "unmatched name")
		expected, err := bson.Marshal(definition)
		require.NoError(mt, err, "failed to marshal definition")
		actual := doc.Lookup("latestDefinition").Value
		assert.Equal(mt, expected, actual, "unmatched definition")
	})

	mt.Run("case 2: Driver can successfully create multiple indexes in batch", func(mt *mtest.T) {
		ctx := context.Background()

		_, err := mt.Coll.InsertOne(ctx, bson.D{})
		require.NoError(mt, err, "failed to insert")

		view := mt.Coll.SearchIndexes()

		definition := bson.D{{"mappings", bson.D{{"dynamic", false}}}}
		models := []mongo.SearchIndexModel{
			{
				Definition: definition,
				Options:    options.SearchIndexes().SetName("test-search-index-1"),
			},
			{
				Definition: definition,
				Options:    options.SearchIndexes().SetName("test-search-index-2"),
			},
		}
		indexes, err := view.CreateMany(ctx, models)
		require.NoError(mt, err, "failed to create index")
		require.Equal(mt, len(indexes), 2, "expected 2 indexes")
		for _, model := range models {
			require.Contains(mt, indexes, *model.Options.Name)
		}

		getDocument := func(opts *options.SearchIndexesOptions) bson.Raw {
			for {
				cursor, err := view.List(ctx, opts)
				require.NoError(mt, err, "failed to list")

				if !cursor.Next(ctx) {
					return nil
				}
				if cursor.Current.Lookup("queryable").Boolean() {
					return cursor.Current
				}
				t.Logf("cursor: %s, sleep 5 seconds...", cursor.Current.String())
				time.Sleep(5 * time.Second)
			}
		}

		var wg sync.WaitGroup
		wg.Add(len(models))
		for i := range models {
			go func(opts *options.SearchIndexesOptions) {
				defer wg.Done()

				doc := getDocument(opts)
				require.NotNil(mt, doc, "got empty document")
				assert.Equal(mt, *opts.Name, doc.Lookup("name").StringValue(), "unmatched name")
				expected, err := bson.Marshal(definition)
				require.NoError(mt, err, "failed to marshal definition")
				actual := doc.Lookup("latestDefinition").Value
				assert.Equal(mt, expected, actual, "unmatched definition")
			}(models[i].Options)
		}
		wg.Wait()
	})

	mt.Run("case 3: Driver can successfully drop search indexes", func(mt *mtest.T) {
		ctx := context.Background()

		_, err := mt.Coll.InsertOne(ctx, bson.D{})
		require.NoError(mt, err, "failed to insert")

		view := mt.Coll.SearchIndexes()

		definition := bson.D{{"mappings", bson.D{{"dynamic", false}}}}
		searchName := "test-search-index"
		opts := options.SearchIndexes().SetName(searchName)
		index, err := view.CreateOne(ctx, mongo.SearchIndexModel{
			Definition: definition,
			Options:    opts,
		})
		require.NoError(mt, err, "failed to create index")
		require.Equal(mt, searchName, index, "unmatched name")

		var doc bson.Raw
		for doc == nil {
			cursor, err := view.List(ctx, opts)
			require.NoError(mt, err, "failed to list")

			if !cursor.Next(ctx) {
				break
			}
			if cursor.Current.Lookup("queryable").Boolean() {
				doc = cursor.Current
			} else {
				t.Logf("cursor: %s, sleep 5 seconds...", cursor.Current.String())
				time.Sleep(5 * time.Second)
			}
		}
		require.NotNil(mt, doc, "got empty document")
		require.Equal(mt, searchName, doc.Lookup("name").StringValue(), "unmatched name")

		err = view.DropOne(ctx, searchName)
		require.NoError(mt, err, "failed to drop index")
		for {
			cursor, err := view.List(ctx, opts)
			require.NoError(mt, err, "failed to list")

			if !cursor.Next(ctx) {
				break
			}
			t.Logf("cursor: %s, sleep 5 seconds...", cursor.Current.String())
			time.Sleep(5 * time.Second)
		}
	})

	mt.Run("case 4: Driver can update a search index", func(mt *mtest.T) {
		ctx := context.Background()

		_, err := mt.Coll.InsertOne(ctx, bson.D{})
		require.NoError(mt, err, "failed to insert")

		view := mt.Coll.SearchIndexes()

		definition := bson.D{{"mappings", bson.D{{"dynamic", false}}}}
		searchName := "test-search-index"
		opts := options.SearchIndexes().SetName(searchName)
		index, err := view.CreateOne(ctx, mongo.SearchIndexModel{
			Definition: definition,
			Options:    opts,
		})
		require.NoError(mt, err, "failed to create index")
		require.Equal(mt, searchName, index, "unmatched name")

		getDocument := func() bson.Raw {
			for {
				cursor, err := view.List(ctx, opts)
				require.NoError(mt, err, "failed to list")

				if !cursor.Next(ctx) {
					return nil
				}
				if cursor.Current.Lookup("queryable").Boolean() {
					return cursor.Current
				}
				t.Logf("cursor: %s, sleep 5 seconds...", cursor.Current.String())
				time.Sleep(5 * time.Second)
			}
		}

		doc := getDocument()
		require.NotNil(mt, doc, "got empty document")
		require.Equal(mt, searchName, doc.Lookup("name").StringValue(), "unmatched name")

		definition = bson.D{{"mappings", bson.D{{"dynamic", true}}}}
		err = view.UpdateOne(ctx, searchName, definition)
		require.NoError(mt, err, "failed to drop index")
		doc = getDocument()
		require.NotNil(mt, doc, "got empty document")
		assert.Equal(mt, searchName, doc.Lookup("name").StringValue(), "unmatched name")
		assert.Equal(mt, "READY", doc.Lookup("status").StringValue(), "unexpected status")
		expected, err := bson.Marshal(definition)
		require.NoError(mt, err, "failed to marshal definition")
		actual := doc.Lookup("latestDefinition").Value
		assert.Equal(mt, expected, actual, "unmatched definition")
	})

	mt.Run("case 5: dropSearchIndex suppresses namespace not found errors", func(mt *mtest.T) {
		ctx := context.Background()

		id, err := uuid.New()
		require.NoError(mt, err)

		collection := mt.CreateCollection(mtest.Collection{
			Name: id.String(),
		}, false)

		err = collection.SearchIndexes().DropOne(ctx, "foo")
		require.NoError(mt, err)
	})
}
