// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package testutil

import (
	"context"
	"strings"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/internal/testutil/helpers"
	"github.com/mongodb/mongo-go-driver/mongo/private/cluster"
	"github.com/mongodb/mongo-go-driver/mongo/private/conn"
	"github.com/mongodb/mongo-go-driver/mongo/private/msg"
	"github.com/mongodb/mongo-go-driver/mongo/readpref"
	"github.com/stretchr/testify/require"
)

// AutoCreateIndex creates an index in the test cluster.
func AutoCreateIndex(t *testing.T, keys []string) {
	indexes := bson.NewDocument()
	for _, k := range keys {
		indexes.Append(bson.C.Int32(k, 1))
	}
	name := strings.Join(keys, "_")
	indexes = bson.NewDocument(
		bson.C.SubDocument("key", indexes),
		bson.C.String("name", name))

	createIndexCommand := bson.NewDocument(
		bson.C.String("createIndexes", ColName(t)),
		bson.C.ArrayFromElements("indexes", bson.AC.Document(indexes)))

	request := msg.NewCommand(
		msg.NextRequestID(),
		DBName(t),
		false,
		createIndexCommand,
	)

	s, err := Cluster(t).SelectServer(context.Background(), cluster.WriteSelector(), readpref.Primary())
	require.NoError(t, err)
	c, err := s.Connection(context.Background())
	require.NoError(t, err)
	defer testhelpers.RequireNoErrorOnClose(t, c)

	_, err = conn.ExecuteCommand(context.Background(), c, request)
	require.NoError(t, err)
}

// AutoDropCollection drops the collection in the test cluster.
func AutoDropCollection(t *testing.T) {
	DropCollection(t, DBName(t), ColName(t))
}

// DropCollection drops the collection in the test cluster.
func DropCollection(t *testing.T, dbname, colname string) {
	s, err := Cluster(t).SelectServer(context.Background(), cluster.WriteSelector(), readpref.Primary())
	require.NoError(t, err)
	c, err := s.Connection(context.Background())
	require.NoError(t, err)
	defer testhelpers.RequireNoErrorOnClose(t, c)

	_, err = conn.ExecuteCommand(
		context.Background(),
		c,
		msg.NewCommand(
			msg.NextRequestID(),
			dbname,
			false,
			bson.NewDocument(bson.C.String("drop", colname)),
		),
	)
	if err != nil && !strings.HasSuffix(err.Error(), "ns not found") {
		t.Fatal(err)
	}
}

func autoDropDB(t *testing.T, clstr *cluster.Cluster) {
	s, err := clstr.SelectServer(context.Background(), cluster.WriteSelector(), readpref.Primary())
	require.NoError(t, err)

	c, err := s.Connection(context.Background())
	require.NoError(t, err)
	defer testhelpers.RequireNoErrorOnClose(t, c)

	_, err = conn.ExecuteCommand(
		context.Background(),
		c,
		msg.NewCommand(
			msg.NextRequestID(),
			DBName(t),
			false,
			bson.NewDocument(bson.C.Int32("dropDatabase", 1)),
		),
	)
	require.NoError(t, err)
}

// AutoInsertDocs inserts the docs into the test cluster.
func AutoInsertDocs(t *testing.T, docs ...*bson.Document) {
	InsertDocs(t, DBName(t), ColName(t), docs...)
}

// InsertDocs inserts the docs into the test cluster.
func InsertDocs(t *testing.T, dbname, colname string, docs ...*bson.Document) {
	arrDocs := make([]*bson.Value, 0, len(docs))
	for _, doc := range docs {
		arrDocs = append(arrDocs, bson.AC.Document(doc))
	}
	insertCommand := bson.NewDocument(
		bson.C.String("insert", colname),
		bson.C.ArrayFromElements("documents", arrDocs...))

	request := msg.NewCommand(
		msg.NextRequestID(),
		dbname,
		false,
		insertCommand,
	)

	s, err := Cluster(t).SelectServer(context.Background(), cluster.WriteSelector(), readpref.Primary())
	require.NoError(t, err)

	c, err := s.Connection(context.Background())
	require.NoError(t, err)
	defer testhelpers.RequireNoErrorOnClose(t, c)

	_, err = conn.ExecuteCommand(context.Background(), c, request)
	require.NoError(t, err)
}

// EnableMaxTimeFailPoint turns on the max time fail point in the test cluster.
func EnableMaxTimeFailPoint(t *testing.T, s cluster.Server) error {
	c, err := s.Connection(context.Background())
	require.NoError(t, err)
	defer testhelpers.RequireNoErrorOnClose(t, c)

	_, err = conn.ExecuteCommand(
		context.Background(),
		c,
		msg.NewCommand(
			msg.NextRequestID(),
			"admin",
			false,
			bson.NewDocument(
				bson.C.String("configureFailPoint", "maxTimeAlwaysTimeOut"),
				bson.C.String("mode", "alwaysOn")),
		),
	)
	return err
}

// DisableMaxTimeFailPoint turns off the max time fail point in the test cluster.
func DisableMaxTimeFailPoint(t *testing.T, s cluster.Server) {
	c, err := s.Connection(context.Background())
	require.NoError(t, err)
	defer testhelpers.RequireNoErrorOnClose(t, c)

	_, err = conn.ExecuteCommand(
		context.Background(),
		c,
		msg.NewCommand(msg.NextRequestID(),
			"admin",
			false,
			bson.NewDocument(
				bson.C.String("configureFailPoint", "maxTimeAlwaysTimeOut"),
				bson.C.String("mode", "off")),
		),
	)
	require.NoError(t, err)
}
