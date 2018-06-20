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
	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/dispatch"
	"github.com/mongodb/mongo-go-driver/core/topology"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
	"github.com/stretchr/testify/require"
)

// AutoCreateIndexes creates an index in the test cluster.
func AutoCreateIndexes(t *testing.T, keys []string) {
	indexes := bson.NewDocument()
	for _, k := range keys {
		indexes.Append(bson.EC.Int32(k, 1))
	}
	name := strings.Join(keys, "_")
	indexes = bson.NewDocument(
		bson.EC.SubDocument("key", indexes),
		bson.EC.String("name", name),
	)
	cmd := command.CreateIndexes{
		NS:      command.NewNamespace(DBName(t), ColName(t)),
		Indexes: bson.NewArray(bson.VC.Document(indexes)),
	}
	_, err := dispatch.CreateIndexes(context.Background(), cmd, Topology(t), description.WriteSelector())
	require.NoError(t, err)
}

// AutoDropCollection drops the collection in the test cluster.
func AutoDropCollection(t *testing.T) {
	DropCollection(t, DBName(t), ColName(t))
}

// DropCollection drops the collection in the test cluster.
func DropCollection(t *testing.T, dbname, colname string) {
	cmd := command.Write{DB: dbname, Command: bson.NewDocument(bson.EC.String("drop", colname))}
	_, err := dispatch.Write(context.Background(), cmd, Topology(t), description.WriteSelector())
	if err != nil && !command.IsNotFound(err) {
		require.NoError(t, err)
	}
}

func autoDropDB(t *testing.T, topo *topology.Topology) {
	cmd := command.Write{DB: DBName(t), Command: bson.NewDocument(bson.EC.Int32("dropDatabase", 1))}
	_, err := dispatch.Write(context.Background(), cmd, topo, description.WriteSelector())
	require.NoError(t, err)
}

// AutoInsertDocs inserts the docs into the test cluster.
func AutoInsertDocs(t *testing.T, writeConcern *writeconcern.WriteConcern, docs ...*bson.Document) {
	InsertDocs(t, DBName(t), ColName(t), writeConcern, docs...)
}

// InsertDocs inserts the docs into the test cluster.
func InsertDocs(t *testing.T, dbname, colname string, writeConcern *writeconcern.WriteConcern, docs ...*bson.Document) {
	cmd := command.Insert{NS: command.NewNamespace(dbname, colname), Docs: docs}

	topo := Topology(t)
	_, err := dispatch.Insert(context.Background(), cmd, topo, description.WriteSelector())
	require.NoError(t, err)
}

// EnableMaxTimeFailPoint turns on the max time fail point in the test cluster.
func EnableMaxTimeFailPoint(t *testing.T, s *topology.Server) error {
	cmd := command.Write{
		DB: "admin",
		Command: bson.NewDocument(
			bson.EC.String("configureFailPoint", "maxTimeAlwaysTimeOut"),
			bson.EC.String("mode", "alwaysOn"),
		),
	}
	conn, err := s.Connection(context.Background())
	require.NoError(t, err)
	defer testhelpers.RequireNoErrorOnClose(t, conn)
	_, err = cmd.RoundTrip(context.Background(), s.SelectedDescription(), conn)
	return err
}

// DisableMaxTimeFailPoint turns off the max time fail point in the test cluster.
func DisableMaxTimeFailPoint(t *testing.T, s *topology.Server) {
	cmd := command.Write{
		DB: "admin",
		Command: bson.NewDocument(
			bson.EC.String("configureFailPoint", "maxTimeAlwaysTimeOut"),
			bson.EC.String("mode", "off"),
		),
	}
	conn, err := s.Connection(context.Background())
	require.NoError(t, err)
	defer testhelpers.RequireNoErrorOnClose(t, conn)
	_, err = cmd.RoundTrip(context.Background(), s.SelectedDescription(), conn)
	require.NoError(t, err)
}
