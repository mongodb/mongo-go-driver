// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
	"go.mongodb.org/mongo-driver/x/mongo/driver/uuid"
)

// temporary file to hold test helpers as we are porting tests to the new integration testing framework.

var wcMajority = writeconcern.New(writeconcern.WMajority())

var doc1 = bsonx.Doc{
	{"x", bsonx.Int32(1)},
}

func createTestClient(t *testing.T) *Client {
	id, _ := uuid.New()
	return &Client{
		id:             id,
		deployment:     testutil.Topology(t),
		connString:     testutil.ConnString(t),
		readPreference: readpref.Primary(),
		clock:          &session.ClusterClock{},
		registry:       bson.DefaultRegistry,
		retryWrites:    true,
		sessionPool:    testutil.SessionPool(),
	}
}

func createTestClientWithConnstring(t *testing.T, cs connstring.ConnString) *Client {
	id, _ := uuid.New()
	return &Client{
		id:             id,
		deployment:     testutil.TopologyWithConnString(t, cs),
		connString:     cs,
		readPreference: readpref.Primary(),
		clock:          &session.ClusterClock{},
		registry:       bson.DefaultRegistry,
	}
}

func createTestDatabase(t *testing.T, name *string, opts ...*options.DatabaseOptions) *Database {
	if name == nil {
		db := testutil.DBName(t)
		name = &db
	}

	client := createTestClient(t)

	dbOpts := []*options.DatabaseOptions{options.Database().SetWriteConcern(writeconcern.New(writeconcern.WMajority()))}
	dbOpts = append(dbOpts, opts...)
	return client.Database(*name, dbOpts...)
}

func skipIfBelow36(t *testing.T) {
	serverVersion, err := getServerVersion(createTestDatabase(t, nil))
	require.NoError(t, err, "unable to get server version of database")

	if compareVersions(t, serverVersion, "3.6") < 0 {
		t.Skip()
	}
}

func skipIfBelow32(t *testing.T) {
	serverVersion, err := getServerVersion(createTestDatabase(t, nil))
	require.NoError(t, err)

	if compareVersions(t, serverVersion, "3.2") < 0 {
		t.Skip()
	}
}
