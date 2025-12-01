// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

func TestSearchIndex(t *testing.T) {
	t.Parallel()

	const timeout = 5 * time.Minute

	uri := os.Getenv("SEARCH_INDEX_URI")
	if uri == "" {
		t.Skip("skipping")
	}

	opts := options.Client().ApplyURI(uri).SetTimeout(timeout)
	mt := mtest.New(t, mtest.NewOptions().ClientOptions(opts).MinServerVersion("7.0").Topologies(mtest.ReplicaSet))

	mt.Run("search indexes with empty option", func(mt *mtest.T) {
		ctx := context.Background()

		expected := bson.RawArray(bsoncore.NewArrayBuilder().AppendDocument(
			bsoncore.NewDocumentBuilder().
				AppendDocument("definition", bsoncore.NewDocumentBuilder().
					AppendDocument("mappings", bsoncore.NewDocumentBuilder().
						AppendBoolean("dynamic", true).
						Build()).
					Build()).
				Build()).
			Build())

		_, err := mt.Coll.SearchIndexes().CreateOne(ctx, mongo.SearchIndexModel{
			Definition: bson.D{
				{"mappings", bson.D{
					{"dynamic", true},
				}},
			},
		})
		require.NoError(mt, err, "failed to create index")
		evt := mt.GetStartedEvent()
		actual, ok := evt.Command.Lookup("indexes").ArrayOK()
		require.True(mt, ok, "expected command %v to contain an indexes array", evt.Command)
		require.Equal(mt, expected, actual, "expected indexes array %v, got %v", expected, actual)
	})
}
