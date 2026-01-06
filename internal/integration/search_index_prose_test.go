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

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/mongoutil"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/internal/uuid"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

func TestSearchIndexProse(t *testing.T) {
	t.Parallel()

	const timeout = 5 * time.Minute

	uri := os.Getenv("SEARCH_INDEX_URI")
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

		awaitCtx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		var doc bson.Raw
		for doc == nil {
			cursor, err := view.List(awaitCtx, opts)
			require.NoError(mt, err, "failed to list")

			if !cursor.Next(awaitCtx) {
				break
			}
			name := cursor.Current.Lookup("name").StringValue()
			queryable := cursor.Current.Lookup("queryable").Boolean()
			if name == searchName && queryable {
				doc = cursor.Current
			} else {
				mt.Logf("cursor: %s, sleep 5 seconds...", cursor.Current.String())
				time.Sleep(5 * time.Second)
			}
		}
		require.NotNil(mt, doc, "got empty document")
		actual := doc.Lookup("latestDefinition", "mappings", "dynamic").Boolean()
		assert.False(mt, actual, "expected latestDefinition.mappings.dynamic to be false")
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
			args, err := mongoutil.NewOptions[options.SearchIndexesOptions](model.Options)
			require.NoError(mt, err, "failed to construct options from builder")

			require.Contains(mt, indexes, *args.Name)
		}

		getDocument := func(ctx context.Context, opts *options.SearchIndexesOptionsBuilder) bson.Raw {
			for {
				cursor, err := view.List(ctx, opts)
				require.NoError(mt, err, "failed to list")

				if !cursor.Next(ctx) {
					return nil
				}
				name := cursor.Current.Lookup("name").StringValue()
				queryable := cursor.Current.Lookup("queryable").Boolean()

				args, err := mongoutil.NewOptions[options.SearchIndexesOptions](opts)
				require.NoError(mt, err, "failed to construct options from builder")

				if name == *args.Name && queryable {
					return cursor.Current
				}
				mt.Logf("cursor: %s, sleep 5 seconds...", cursor.Current.String())
				time.Sleep(5 * time.Second)
			}
		}

		var wg sync.WaitGroup
		wg.Add(len(models))
		for i := range models {
			go func(opts *options.SearchIndexesOptionsBuilder) {
				defer wg.Done()

				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
				defer cancel()

				doc := getDocument(ctx, opts)
				require.NotNil(mt, doc, "got empty document")

				args, err := mongoutil.NewOptions[options.SearchIndexesOptions](opts)
				require.NoError(mt, err, "failed to construct options from builder")

				assert.Equal(mt, *args.Name, doc.Lookup("name").StringValue(), "unmatched name")

				actual := doc.Lookup("latestDefinition", "mappings", "dynamic").Boolean()
				assert.False(mt, actual, "expected latestDefinition.mappings.dynamic to be false")
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

		createOneCtx, createOneCancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer createOneCancel()
		var doc bson.Raw
		for doc == nil {
			cursor, err := view.List(createOneCtx, opts)
			require.NoError(mt, err, "failed to list")

			if !cursor.Next(createOneCtx) {
				break
			}
			name := cursor.Current.Lookup("name").StringValue()
			queryable := cursor.Current.Lookup("queryable").Boolean()
			if name == searchName && queryable {
				doc = cursor.Current
			} else {
				mt.Logf("cursor: %s, sleep 5 seconds...", cursor.Current.String())
				time.Sleep(5 * time.Second)
			}
		}
		require.NotNil(mt, doc, "got empty document")

		err = view.DropOne(ctx, searchName)
		require.NoError(mt, err, "failed to drop index")
		dropOneCtx, dropOneCancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer dropOneCancel()
		for {
			cursor, err := view.List(dropOneCtx, opts)
			require.NoError(mt, err, "failed to list")

			if !cursor.Next(dropOneCtx) {
				break
			}
			mt.Logf("cursor: %s, sleep 5 seconds...", cursor.Current.String())
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

		createOneCtx, createOneCancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer createOneCancel()
		var doc bson.Raw
		for doc == nil {
			cursor, err := view.List(createOneCtx, opts)
			require.NoError(mt, err, "failed to list")

			if !cursor.Next(createOneCtx) {
				break
			}
			name := cursor.Current.Lookup("name").StringValue()
			queryable := cursor.Current.Lookup("queryable").Boolean()
			if name == searchName && queryable {
				doc = cursor.Current
			} else {
				mt.Logf("cursor: %s, sleep 5 seconds...", cursor.Current.String())
				time.Sleep(5 * time.Second)
			}
		}
		require.NotNil(mt, doc, "got empty document")

		definition = bson.D{{"mappings", bson.D{{"dynamic", true}}}}
		err = view.UpdateOne(ctx, searchName, definition)
		require.NoError(mt, err, "failed to update index")
		updateOneCtx, updateOneCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer updateOneCancel()
		doc = nil
		for doc == nil {
			cursor, err := view.List(updateOneCtx, opts)
			require.NoError(mt, err, "failed to list")

			if !cursor.Next(updateOneCtx) {
				break
			}
			name := cursor.Current.Lookup("name").StringValue()
			queryable := cursor.Current.Lookup("queryable").Boolean()
			status := cursor.Current.Lookup("status").StringValue()
			latestDefinition, ok := cursor.Current.Lookup("latestDefinition", "mappings", "dynamic").BooleanOK()
			if name == searchName && queryable && status == "READY" && ok && latestDefinition {
				doc = cursor.Current
			} else {
				mt.Logf("cursor: %s, sleep 5 seconds...", cursor.Current.String())
				time.Sleep(5 * time.Second)
			}
		}
		require.NotNil(mt, doc, "got empty document")
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

	mt.RunOpts("case 6: Driver can successfully create and list search indexes with non-default readConcern and writeConcern",
		mtest.NewOptions().CollectionOptions(options.Collection().SetWriteConcern(writeconcern.W1()).SetReadConcern(readconcern.Majority())),
		func(mt *mtest.T) {
			ctx := context.Background()

			_, err := mt.Coll.InsertOne(ctx, bson.D{})
			require.NoError(mt, err, "failed to insert")

			view := mt.Coll.SearchIndexes()

			definition := bson.D{{"mappings", bson.D{{"dynamic", false}}}}
			const searchName = "test-search-index-case6"
			opts := options.SearchIndexes().SetName(searchName)
			index, err := view.CreateOne(ctx, mongo.SearchIndexModel{
				Definition: definition,
				Options:    opts,
			})
			require.NoError(mt, err, "failed to create index")
			require.Equal(mt, searchName, index, "unmatched name")
			awaitCtx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancel()
			var doc bson.Raw
			for doc == nil {
				cursor, err := view.List(awaitCtx, opts)
				require.NoError(mt, err, "failed to list")

				if !cursor.Next(awaitCtx) {
					break
				}
				name := cursor.Current.Lookup("name").StringValue()
				queryable := cursor.Current.Lookup("queryable").Boolean()
				if name == searchName && queryable {
					doc = cursor.Current
				} else {
					mt.Logf("cursor: %s, sleep 5 seconds...", cursor.Current.String())
					time.Sleep(5 * time.Second)
				}
			}
			require.NotNil(mt, doc, "got empty document")
			actual := doc.Lookup("latestDefinition", "mappings", "dynamic").Boolean()
			assert.False(mt, actual, "expected latestDefinition.mappings.dynamic to be false")
		})

	case7CollName, err := uuid.New()
	assert.NoError(mt, err, "failed to create random collection name for case #7")

	mt.RunOpts("case 7: Driver can successfully handle search index types when creating indexes",
		mtest.NewOptions().CollectionName(case7CollName.String()),
		func(mt *mtest.T) {
			ctx := context.Background()

			_, err := mt.Coll.InsertOne(ctx, bson.D{})
			require.NoError(mt, err, "failed to insert")

			view := mt.Coll.SearchIndexes()

			definition := bson.D{{"mappings", bson.D{{"dynamic", false}}}}
			indexName := "test-search-index-case7-implicit"
			opts := options.SearchIndexes().SetName(indexName)
			index, err := view.CreateOne(ctx, mongo.SearchIndexModel{
				Definition: definition,
				Options:    opts,
			})
			require.NoError(mt, err, "failed to create index")
			require.Equal(mt, indexName, index, "unmatched name")
			implicitCtx, implicitCancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer implicitCancel()
			var doc bson.Raw
			for doc == nil {
				cursor, err := view.List(implicitCtx, opts)
				require.NoError(mt, err, "failed to list")

				if !cursor.Next(implicitCtx) {
					break
				}
				name := cursor.Current.Lookup("name").StringValue()
				queryable := cursor.Current.Lookup("queryable").Boolean()
				indexType := cursor.Current.Lookup("type").StringValue()
				if name == indexName && queryable {
					doc = cursor.Current
					assert.Equal(mt, indexType, "search")
				} else {
					mt.Logf("cursor: %s, sleep 5 seconds...", cursor.Current.String())
					time.Sleep(5 * time.Second)
				}
			}

			indexName = "test-search-index-case7-explicit"
			opts = options.SearchIndexes().SetName(indexName).SetType("search")
			index, err = view.CreateOne(ctx, mongo.SearchIndexModel{
				Definition: definition,
				Options:    opts,
			})
			require.NoError(mt, err, "failed to create index")
			require.Equal(mt, indexName, index, "unmatched name")
			explicitCtx, explicitCancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer explicitCancel()
			doc = nil
			for doc == nil {
				cursor, err := view.List(explicitCtx, opts)
				require.NoError(mt, err, "failed to list")

				if !cursor.Next(explicitCtx) {
					break
				}
				name := cursor.Current.Lookup("name").StringValue()
				queryable := cursor.Current.Lookup("queryable").Boolean()
				indexType := cursor.Current.Lookup("type").StringValue()
				if name == indexName && queryable {
					doc = cursor.Current
					assert.Equal(mt, indexType, "search")
				} else {
					mt.Logf("cursor: %s, sleep 5 seconds...", cursor.Current.String())
					time.Sleep(5 * time.Second)
				}
			}

			indexName = "test-search-index-case7-vector"
			type vectorDefinitionField struct {
				Type          string `bson:"type"`
				Path          string `bson:"path"`
				NumDimensions int    `bson:"numDimensions"`
				Similarity    string `bson:"similarity"`
			}

			type vectorDefinition struct {
				Fields []vectorDefinitionField `bson:"fields"`
			}

			opts = options.SearchIndexes().SetName(indexName).SetType("vectorSearch")
			index, err = view.CreateOne(ctx, mongo.SearchIndexModel{
				Definition: vectorDefinition{
					Fields: []vectorDefinitionField{{"vector", "path", 1536, "euclidean"}},
				},
				Options: opts,
			})
			require.NoError(mt, err, "failed to create index")
			require.Equal(mt, indexName, index, "unmatched name")
			vectorCtx, vectorCancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer vectorCancel()
			doc = nil
			for doc == nil {
				cursor, err := view.List(vectorCtx, opts)
				require.NoError(mt, err, "failed to list")

				if !cursor.Next(vectorCtx) {
					break
				}
				name := cursor.Current.Lookup("name").StringValue()
				queryable := cursor.Current.Lookup("queryable").Boolean()
				indexType := cursor.Current.Lookup("type").StringValue()
				if name == indexName && queryable {
					doc = cursor.Current
					assert.Equal(mt, indexType, "vectorSearch")
				} else {
					mt.Logf("cursor: %s, sleep 5 seconds...", cursor.Current.String())
					time.Sleep(5 * time.Second)
				}
			}
		})

	case8CollName, err := uuid.New()
	assert.NoError(mt, err, "failed to create random collection name for case #8")

	mt.RunOpts("case 8: Driver requires explicit type to create a vector search index",
		mtest.NewOptions().CollectionName(case8CollName.String()),
		func(mt *mtest.T) {
			ctx := context.Background()

			_, err := mt.Coll.InsertOne(ctx, bson.D{})
			require.NoError(mt, err, "failed to insert")

			view := mt.Coll.SearchIndexes()

			type vectorDefinitionField struct {
				Type          string `bson:"type"`
				Path          string `bson:"path"`
				NumDimensions int    `bson:"numDimensions"`
				Similarity    string `bson:"similarity"`
			}

			type vectorDefinition struct {
				Fields []vectorDefinitionField `bson:"fields"`
			}

			const indexName = "test-search-index-case7-vector"
			opts := options.SearchIndexes().SetName(indexName)
			_, err = view.CreateOne(ctx, mongo.SearchIndexModel{
				Definition: vectorDefinition{
					Fields: []vectorDefinitionField{{"vector", "plot_embedding", 1536, "euclidean"}},
				},
				Options: opts,
			})
			assert.ErrorContains(mt, err, "Attribute mappings missing")
		})

	mt.Run("case 9: Drivers use server default for unspecified name (`default`) and type (`search`)", func(mt *mtest.T) {
		cases := []struct {
			name string
			opts *options.SearchIndexesOptionsBuilder
		}{
			{name: "empty options", opts: options.SearchIndexes()},
			{name: "nil options", opts: nil},
		}

		for _, tc := range cases {
			mt.Run(tc.name, func(mt *mtest.T) {
				view := mt.Coll.SearchIndexes()
				definition := bson.D{
					{"mappings", bson.D{
						{"dynamic", true},
					}},
				}

				indexName, err := view.CreateOne(context.Background(), mongo.SearchIndexModel{
					Definition: definition,
					Options:    tc.opts,
				})
				require.NoError(mt, err, "failed to create index")
				require.Equal(mt, "default", indexName)

				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
				defer cancel()

				var doc bson.Raw
				for doc == nil {
					cursor, err := view.List(ctx, tc.opts)
					require.NoError(mt, err, "failed to list")

					if !cursor.Next(ctx) {
						break
					}
					name := cursor.Current.Lookup("name").StringValue()
					queryable := cursor.Current.Lookup("queryable").Boolean()
					indexType := cursor.Current.Lookup("type").StringValue()
					if name == indexName && queryable {
						doc = cursor.Current
						require.Equal(mt, indexType, "search")
					} else {
						mt.Logf("cursor: %s, sleep 5 seconds...", cursor.Current.String())
						time.Sleep(5 * time.Second)
					}
				}
			})
		}
	})
}
