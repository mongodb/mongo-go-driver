// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/internal/uuid"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func newTestCollection(t *testing.T, client *Client) *Collection {
	t.Helper()

	id, err := uuid.New()
	require.NoError(t, err)
	collName := fmt.Sprintf("col%s", id.String())
	return client.Database("test").Collection(collName)
}

func TestSearchIndexProse(t *testing.T) {
	t.Parallel()

	const timeout = 5 * time.Minute

	ctx := context.Background()

	uri := os.Getenv("TEST_INDEX_URI")
	if uri == "" {
		t.Skip("skipping")
	}
	clientOptions := options.Client().ApplyURI(uri).SetTimeout(timeout)

	client, err := Connect(ctx, clientOptions)
	assert.NoError(t, err, "failed to connect %s", uri)
	defer client.Disconnect(ctx)

	t.Run("case 1: Driver can successfully create and list search indexes", func(t *testing.T) {
		ctx := context.Background()

		collection := newTestCollection(t, client)
		defer func() {
			err = collection.Drop(ctx)
			assert.NoError(t, err, "failed to drop collection")
		}()

		_, err = collection.InsertOne(ctx, bson.D{})
		require.NoError(t, err, "failed to insert")

		view := collection.SearchIndexes()

		definition := bson.D{{"mappings", bson.D{{"dynamic", false}}}}
		searchName := "test-search-index"
		model := SearchIndexModel{
			Definition: definition,
			Name:       &searchName,
		}
		index, err := view.CreateOne(ctx, model)
		require.NoError(t, err, "failed to create index")
		require.Equal(t, searchName, index, "unmatched name")

		var doc bson.Raw
		for doc == nil {
			cursor, err := view.List(ctx, &index, nil)
			require.NoError(t, err, "failed to list")

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
		require.NotNil(t, doc, "got empty document")
		assert.Equal(t, searchName, doc.Lookup("name").StringValue(), "unmatched name")
		expected, err := bson.Marshal(definition)
		require.NoError(t, err, "failed to marshal definition")
		actual := doc.Lookup("latestDefinition").Value
		assert.Equal(t, expected, actual, "unmatched definition")
	})

	t.Run("case 2: Driver can successfully create multiple indexes in batch", func(t *testing.T) {
		ctx := context.Background()

		collection := newTestCollection(t, client)
		defer func() {
			err = collection.Drop(ctx)
			assert.NoError(t, err, "failed to drop collection")
		}()

		_, err = collection.InsertOne(ctx, bson.D{})
		require.NoError(t, err, "failed to insert")

		view := collection.SearchIndexes()

		definition := bson.D{{"mappings", bson.D{{"dynamic", false}}}}
		searchName0 := "test-search-index-1"
		searchName1 := "test-search-index-2"
		models := []SearchIndexModel{
			{
				Definition: definition,
				Name:       &searchName0,
			},
			{
				Definition: definition,
				Name:       &searchName1,
			},
		}
		indexes, err := view.CreateMany(ctx, models)
		require.NoError(t, err, "failed to create index")
		require.Equal(t, len(indexes), 2, "expected 2 indexes")
		require.Contains(t, indexes, searchName0)
		require.Contains(t, indexes, searchName1)

		getDocument := func(index string) bson.Raw {
			t.Helper()

			for {
				cursor, err := view.List(ctx, &index, nil)
				require.NoError(t, err, "failed to list")

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
		wg.Add(2)
		go func() {
			defer wg.Done()

			doc := getDocument(searchName0)
			require.NotNil(t, doc, "got empty document")
			assert.Equal(t, searchName0, doc.Lookup("name").StringValue(), "unmatched name")
			expected, err := bson.Marshal(definition)
			require.NoError(t, err, "failed to marshal definition")
			actual := doc.Lookup("latestDefinition").Value
			assert.Equal(t, expected, actual, "unmatched definition")
		}()
		go func() {
			defer wg.Done()

			doc := getDocument(searchName1)
			require.NotNil(t, doc, "got empty document")
			assert.Equal(t, searchName1, doc.Lookup("name").StringValue(), "unmatched name")
			expected, err := bson.Marshal(definition)
			require.NoError(t, err, "failed to marshal definition")
			actual := doc.Lookup("latestDefinition").Value
			assert.Equal(t, expected, actual, "unmatched definition")
		}()
		wg.Wait()
	})

	t.Run("case 3: Driver can successfully drop search indexes", func(t *testing.T) {
		ctx := context.Background()

		collection := newTestCollection(t, client)
		defer func() {
			err = collection.Drop(ctx)
			assert.NoError(t, err, "failed to drop collection")
		}()

		_, err = collection.InsertOne(ctx, bson.D{})
		require.NoError(t, err, "failed to insert")

		view := collection.SearchIndexes()

		definition := bson.D{{"mappings", bson.D{{"dynamic", false}}}}
		searchName := "test-search-index"
		model := SearchIndexModel{
			Definition: definition,
			Name:       &searchName,
		}
		index, err := view.CreateOne(ctx, model)
		require.NoError(t, err, "failed to create index")
		require.Equal(t, searchName, index, "unmatched name")

		var doc bson.Raw
		for doc == nil {
			cursor, err := view.List(ctx, &index, nil)
			require.NoError(t, err, "failed to list")

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
		require.NotNil(t, doc, "got empty document")
		require.Equal(t, searchName, doc.Lookup("name").StringValue(), "unmatched name")

		err = view.DropOne(ctx, searchName)
		require.NoError(t, err, "failed to drop index")
		for {
			cursor, err := view.List(ctx, &index, nil)
			require.NoError(t, err, "failed to list")

			if !cursor.Next(ctx) {
				break
			}
			t.Logf("cursor: %s, sleep 5 seconds...", cursor.Current.String())
			time.Sleep(5 * time.Second)
		}
	})

	t.Run("case 4: Driver can update a search index", func(t *testing.T) {
		ctx := context.Background()

		collection := newTestCollection(t, client)
		defer func() {
			err = collection.Drop(ctx)
			assert.NoError(t, err, "failed to drop collection")
		}()

		_, err = collection.InsertOne(ctx, bson.D{})
		require.NoError(t, err, "failed to insert")

		view := collection.SearchIndexes()

		definition := bson.D{{"mappings", bson.D{{"dynamic", false}}}}
		searchName := "test-search-index"
		model := SearchIndexModel{
			Definition: definition,
			Name:       &searchName,
		}
		index, err := view.CreateOne(ctx, model)
		require.NoError(t, err, "failed to create index")
		require.Equal(t, searchName, index, "unmatched name")

		getDocument := func(index string) bson.Raw {
			t.Helper()

			for {
				cursor, err := view.List(ctx, &index, nil)
				require.NoError(t, err, "failed to list")

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

		doc := getDocument(searchName)
		require.NotNil(t, doc, "got empty document")
		require.Equal(t, searchName, doc.Lookup("name").StringValue(), "unmatched name")

		definition = bson.D{{"mappings", bson.D{{"dynamic", true}}}}
		err = view.UpdateOne(ctx, searchName, definition)
		require.NoError(t, err, "failed to drop index")
		doc = getDocument(searchName)
		require.NotNil(t, doc, "got empty document")
		assert.Equal(t, searchName, doc.Lookup("name").StringValue(), "unmatched name")
		assert.Equal(t, "READY", doc.Lookup("status").StringValue(), "unexpected status")
		expected, err := bson.Marshal(definition)
		require.NoError(t, err, "failed to marshal definition")
		actual := doc.Lookup("latestDefinition").Value
		assert.Equal(t, expected, actual, "unmatched definition")
	})

	t.Run("case 5: dropSearchIndex suppresses namespace not found errors", func(t *testing.T) {
		ctx := context.Background()

		collection := newTestCollection(t, client)
		defer func() {
			err = collection.Drop(ctx)
			assert.NoError(t, err, "failed to drop collection")
		}()

		err = collection.SearchIndexes().DropOne(ctx, "foo")
		require.NoError(t, err)
	})
}
