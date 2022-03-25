// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package testutil // import "go.mongodb.org/mongo-driver/internal/testutil"

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/operation"
)

// DropCollection drops the collection in the test cluster.
func DropCollection(t *testing.T, dbname, colname string) {
	err := operation.NewCommand(bsoncore.BuildDocument(nil, bsoncore.AppendStringElement(nil, "drop", colname))).
		Database(dbname).ServerSelector(description.WriteSelector()).Deployment(Topology(t)).
		Execute(context.Background())
	if de, ok := err.(driver.Error); err != nil && !(ok && de.NamespaceNotFound()) {
		require.NoError(t, err)
	}
}

// AutoInsertDocs inserts the docs into the test cluster.
func AutoInsertDocs(t *testing.T, writeConcern *writeconcern.WriteConcern, docs ...bsoncore.Document) {
	InsertDocs(t, DBName(t), ColName(t), writeConcern, docs...)
}

// InsertDocs inserts the docs into the test cluster.
func InsertDocs(t *testing.T, dbname, colname string, writeConcern *writeconcern.WriteConcern, docs ...bsoncore.Document) {
	err := operation.NewInsert(docs...).Collection(colname).Database(dbname).
		Deployment(Topology(t)).ServerSelector(description.WriteSelector()).Execute(context.Background())
	require.NoError(t, err)
}

// RunCommand runs an arbitrary command on a given database of target server
func RunCommand(t *testing.T, s driver.Server, db string, cmd bsoncore.Document) (bsoncore.Document, error) {
	op := operation.NewCommand(cmd).
		Database(db).Deployment(driver.SingleServerDeployment{Server: s})
	err := op.Execute(context.Background())
	res := op.Result()
	return res, err
}

// AddTestServerAPIVersion adds the latest server API version in a ServerAPIOptions to passed-in opts.
func AddTestServerAPIVersion(opts *options.ClientOptions) {
	if os.Getenv("REQUIRE_API_VERSION") == "true" {
		opts.SetServerAPIOptions(options.ServerAPI(driver.TestServerAPIVersion))
	}
}
