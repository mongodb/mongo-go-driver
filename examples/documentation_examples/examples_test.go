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

	"github.com/mongodb/mongo-go-driver/examples/documentation_examples"
	"github.com/mongodb/mongo-go-driver/internal/testutil"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
	"github.com/stretchr/testify/require"
)

func TestDocumentationExamples(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cs := testutil.ConnString(t)
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cs.String()))
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	db := client.Database("documentation_examples")

	documentation_examples.InsertExamples(t, db)
	documentation_examples.QueryToplevelFieldsExamples(t, db)
	documentation_examples.QueryEmbeddedDocumentsExamples(t, db)
	documentation_examples.QueryArraysExamples(t, db)
	documentation_examples.QueryArrayEmbeddedDocumentsExamples(t, db)
	documentation_examples.QueryNullMissingFieldsExamples(t, db)
	documentation_examples.ProjectionExamples(t, db)
	documentation_examples.UpdateExamples(t, db)
	documentation_examples.DeleteExamples(t, db)
}

func TestTransactionExamples(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cs := testutil.ConnString(t)
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cs.String()))
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	ver, err := getServerVersion(ctx, client)
	if err != nil || testutil.CompareVersions(t, ver, "4.0") < 0 || os.Getenv("TOPOLOGY") != "replica_set" {
		t.Skip("server does not support transactions")
	}
	err = documentation_examples.TransactionsExamples(ctx, client)
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
