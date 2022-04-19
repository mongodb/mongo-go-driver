// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// NOTE: Any time this file is modified, a WEBSITE ticket should be opened to sync the changes with
// the "What is MongoDB" webpage, which the example was originally added to as part of WEBSITE-5148.

package documentation_examples_test

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/examples/documentation_examples"
	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
)

func TestMain(m *testing.M) {
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

	cs := testutil.ConnString(t)
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cs.String()))
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

	// Because it uses RunCommand with an apiVersion, the strict count example can only be
	// run on 5.0+ without auth. It also cannot be run on 6.0+ since the count command was
	// added to API version 1 and no longer results in an error when strict is enabled.
	mtOpts := mtest.NewOptions().MinServerVersion("5.0").MaxServerVersion("5.3")
	mt.RunOpts("StableAPIStrictCountExample", mtOpts, func(t *mtest.T) {
		documentation_examples.StableAPIStrictCountExample(mt.T)
	})

	mtOpts = mtest.NewOptions().MinServerVersion("5.0").Topologies(mtest.ReplicaSet, mtest.Sharded)
	mt.RunOpts("SnapshotQueryExamples", mtOpts, func(t *mtest.T) {
		documentation_examples.SnapshotQueryExamples(t)
	})
}

func TestAggregationExamples(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cs := testutil.ConnString(t)
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cs.String()))
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	db := client.Database("documentation_examples")

	ver, err := getServerVersion(ctx, client)
	if err != nil || testutil.CompareVersions(t, ver, "3.6") < 0 {
		t.Skip("server does not support let in $lookup in aggregations")
	}
	documentation_examples.AggregationExamples(t, db)
}

func TestTransactionExamples(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	topo := createTopology(t)
	client, err := mongo.Connect(context.Background(), &options.ClientOptions{Deployment: topo})
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	ver, err := getServerVersion(ctx, client)
	if err != nil || testutil.CompareVersions(t, ver, "4.0") < 0 || topo.Kind() != description.ReplicaSet {
		t.Skip("server does not support transactions")
	}

	err = documentation_examples.TransactionsExamples(ctx, client)
	require.NoError(t, err)

	err = documentation_examples.WithTransactionExample(ctx)
	require.NoError(t, err)
}

func TestChangeStreamExamples(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	topo := createTopology(t)
	client, err := mongo.Connect(context.Background(), &options.ClientOptions{Deployment: topo})
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	db := client.Database("changestream_examples")
	ver, err := getServerVersion(ctx, client)
	if err != nil || testutil.CompareVersions(t, ver, "3.6") < 0 || topo.Kind() != description.ReplicaSet {
		t.Skip("server does not support changestreams")
	}
	documentation_examples.ChangeStreamExamples(t, db)
}

func TestCausalConsistencyExamples(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cs := testutil.ConnString(t)
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cs.String()))
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	// TODO(GODRIVER-2238): Remove skip once failures on MongoDB v4.0 sharded clusters are fixed.
	ver, err := getServerVersion(ctx, client)
	if err != nil || testutil.CompareVersions(t, ver, "4.0") == 0 {
		t.Skip("TODO(GODRIVER-2238): Skip until failures on MongoDB v4.0 sharded clusters are fixed")
	}

	err = documentation_examples.CausalConsistencyExamples(client)
	require.NoError(t, err)
}

func getServerVersion(ctx context.Context, client *mongo.Client) (string, error) {
	serverStatus, err := client.Database("admin").RunCommand(
		ctx,
		bson.D{{"serverStatus", 1}},
	).DecodeBytes()
	if err != nil {
		return "", err
	}

	version, err := serverStatus.LookupErr("version")
	if err != nil {
		return "", err
	}

	return version.StringValue(), nil
}

func createTopology(t *testing.T) *topology.Topology {
	topo, err := topology.New(topology.WithConnString(func(connstring.ConnString) connstring.ConnString {
		return testutil.ConnString(t)
	}))
	if err != nil {
		t.Fatalf("topology.New error: %v", err)
	}
	return topo
}
