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
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/examples/documentation_examples"
	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
)

func TestDocumentationExamples(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cs := testutil.ConnString(t)
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cs.String()))
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	db := client.Database("documentation_examples")

	stw := newSubtestWrapper(t, db)
	stw.run("InsertExamples", documentation_examples.InsertExamples)
	stw.run("QueryToplevelFieldsExamples", documentation_examples.QueryToplevelFieldsExamples)
	stw.run("QueryEmbeddedDocumentsExamples", documentation_examples.QueryEmbeddedDocumentsExamples)
	stw.run("QueryArraysExamples", documentation_examples.QueryArraysExamples)
	stw.run("QueryArrayEmbeddedDocumentsExamples", documentation_examples.QueryArrayEmbeddedDocumentsExamples)
	stw.run("QueryNullMissingFieldssExamples", documentation_examples.QueryNullMissingFieldsExamples)
	stw.run("ProjectionExamples", documentation_examples.ProjectionExamples)
	stw.run("UpdateExamples", documentation_examples.UpdateExamples)
	stw.run("DeleteExamples", documentation_examples.DeleteExamples)
	stw.run("RunCommandExamples", documentation_examples.RunCommandExamples)
	stw.run("IndexExamples", documentation_examples.IndexExamples)
	stw.runEmpty("StableAPExamples", documentation_examples.StableAPIExamples)

	// Because it uses RunCommand with an apiVersion, the strict count example can only be
	// run on 5.0+ without auth. It also cannot be run on 6.0+ since the count command was
	// added to API version 1 and no longer results in an error when strict is enabled.
	ver, err := getServerVersion(ctx, client)
	require.NoError(t, err, "getServerVersion error: %v", err)
	auth := os.Getenv("AUTH") == "auth"
	if testutil.CompareVersions(t, ver, "5.0") >= 0 && testutil.CompareVersions(t, ver, "6.0") < 0 && !auth {
		documentation_examples.StableAPIStrictCountExample(t)
	} else {
		t.Log("skipping stable API strict count example")
	}
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
		bsonx.Doc{{"serverStatus", bsonx.Int32(1)}},
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

// subtestWrapper maintains testing state for subtest operations
type subtestWrapper struct {
	t  *testing.T
	db *mongo.Database
}

func newSubtestWrapper(t *testing.T, db *mongo.Database) subtestWrapper {
	stw := new(subtestWrapper)
	stw.t = t
	stw.db = db
	return *stw
}

type subtest func(*testing.T, *mongo.Database)
type subtestEmpty func()

// run wraps a subtest using an `stw` object
func (stw subtestWrapper) run(name string, st subtest) {
	stw.t.Run(name, func(t *testing.T) { st(stw.t, stw.db) })
}

// runEmpty wraps a subtest without functional arguments from an `stw` object.
func (stw subtestWrapper) runEmpty(name string, st subtestEmpty) {
	stw.t.Run(name, func(t *testing.T) { st() })
}
