// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package documentation_examples_test

import (
	"context"
	"flag"
	"log"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/examples/documentation_examples"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestMain(m *testing.M) {
	// All tests that use mtest.Setup() are expected to be integration tests, so skip them when the
	// -short flag is included in the "go test" command. Also, we have to parse flags here to use
	// testing.Short() because flags aren't parsed before TestMain() is called.
	flag.Parse()
	if testing.Short() {
		log.Print("skipping mtest integration test in short mode")
		return
	}

	if err := mtest.Setup(); err != nil {
		log.Fatal(err)
	}
	defer os.Exit(m.Run())
	if err := mtest.Teardown(); err != nil {
		log.Fatal(err)
	}
}

func TestDocumentationExamples(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mtest.ClusterURI()))
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	db := client.Database("documentation_examples")

	t.Run("InsertExamples", func(t *testing.T) {
		documentation_examples.InsertExamples(t, db)
	})
	t.Run("QueryArraysExamples", func(t *testing.T) {
		documentation_examples.QueryArraysExamples(t, db)
	})
	t.Run("ProjectionExamples", func(t *testing.T) {
		documentation_examples.ProjectionExamples(t, db)
	})
	t.Run("UpdateExamples", func(t *testing.T) {
		documentation_examples.UpdateExamples(t, db)
	})
	t.Run("DeleteExamples", func(t *testing.T) {
		documentation_examples.DeleteExamples(t, db)
	})
	t.Run("RunCommandExamples", func(t *testing.T) {
		documentation_examples.RunCommandExamples(t, db)
	})
	t.Run("IndexExamples", func(t *testing.T) {
		documentation_examples.IndexExamples(t, db)
	})
	t.Run("StableAPExamples", func(t *testing.T) {
		documentation_examples.StableAPIExamples()
	})
	t.Run("QueryToplevelFieldsExamples", func(t *testing.T) {
		documentation_examples.QueryToplevelFieldsExamples(t, db)
	})
	t.Run("QueryEmbeddedDocumentsExamples", func(t *testing.T) {
		documentation_examples.QueryEmbeddedDocumentsExamples(t, db)
	})
	t.Run("QueryArrayEmbeddedDocumentsExamples", func(t *testing.T) {
		documentation_examples.QueryArrayEmbeddedDocumentsExamples(t, db)
	})
	t.Run("QueryNullMissingFieldsExamples", func(t *testing.T) {
		documentation_examples.QueryNullMissingFieldsExamples(t, db)
	})

	mt := mtest.New(t)
	defer mt.Close()

	// Stable API is supported in 5.0+
	mtOpts := mtest.NewOptions().MinServerVersion("5.0")
	mt.RunOpts("StableAPIStrictCountExample", mtOpts, func(mt *mtest.T) {
		// TODO(GODRIVER-2482): Unskip when this test is rewritten to work with Stable API v1.
		mt.Skip(`skipping because "count" is now part of Stable API v1; see GODRIVER-2482`)

		documentation_examples.StableAPIStrictCountExample(mt.T)
	})

	// Snapshot queries can only run on 5.0+ non-sharded and sharded replicasets.
	mtOpts = mtest.NewOptions().MinServerVersion("5.0").Topologies(mtest.ReplicaSet, mtest.Sharded)
	mt.RunOpts("SnapshotQueryExamples", mtOpts, func(mt *mtest.T) {
		documentation_examples.SnapshotQueryExamples(mt)
	})

	// Only 3.6+ supports $lookup in aggregations as is used in aggregation examples. No
	// collection needs to be created, and collection creation can sometimes cause failures
	// on sharded clusters.
	mtOpts = mtest.NewOptions().MinServerVersion("3.6").CreateCollection(false)
	mt.RunOpts("AggregationExamples", mtOpts, func(mt *mtest.T) {
		documentation_examples.AggregationExamples(mt.T, db)
	})

	// Transaction examples can only run on replica sets on 4.0+.
	mtOpts = mtest.NewOptions().MinServerVersion("4.0").Topologies(mtest.ReplicaSet)
	mt.RunOpts("TransactionsExamples", mtOpts, func(mt *mtest.T) {
		mt.ResetClient(options.Client().ApplyURI(mtest.ClusterURI()))
		documentation_examples.TransactionsExamples(ctx, mt.Client)
	})
	mt.RunOpts("WithTransactionExample", mtOpts, func(mt *mtest.T) {
		mt.ResetClient(options.Client().ApplyURI(mtest.ClusterURI()))
		documentation_examples.WithTransactionExample(ctx)
	})

	// Change stream examples can only run on replica sets on 3.6+.
	mtOpts = mtest.NewOptions().MinServerVersion("3.6").Topologies(mtest.ReplicaSet)
	mt.RunOpts("ChangeStreamExamples", mtOpts, func(mt *mtest.T) {
		mt.ResetClient(options.Client().ApplyURI(mtest.ClusterURI()))
		csdb := client.Database("changestream_examples")
		documentation_examples.ChangeStreamExamples(mt.T, csdb)
	})

	// Causal consistency examples cannot run on 4.0.
	// TODO(GODRIVER-2238): Remove version filtering once failures on 4.0 sharded clusters are fixed.
	mtOpts = mtest.NewOptions().MinServerVersion("4.2").Topologies(mtest.ReplicaSet)
	mt.RunOpts("CausalConsistencyExamples/post4.2", mtOpts, func(mt *mtest.T) {
		documentation_examples.CausalConsistencyExamples(mt.Client)
	})
	mtOpts = mtest.NewOptions().MaxServerVersion("4.0").Topologies(mtest.ReplicaSet)
	mt.RunOpts("CausalConsistencyExamples/pre4.0", mtOpts, func(mt *mtest.T) {
		documentation_examples.CausalConsistencyExamples(mt.Client)
	})
}
