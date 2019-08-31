// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
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

func createTestCollection(t *testing.T, dbName *string, collName *string, opts ...*options.CollectionOptions) *Collection {
	if collName == nil {
		coll := testutil.ColName(t)
		collName = &coll
	}

	db := createTestDatabase(t, dbName)
	db.RunCommand(
		context.Background(),
		bsonx.Doc{{"create", bsonx.String(*collName)}},
	)

	collOpts := []*options.CollectionOptions{options.Collection().SetWriteConcern(writeconcern.New(writeconcern.WMajority()))}
	collOpts = append(collOpts, opts...)
	return db.Collection(*collName, collOpts...)
}

func skipIfBelow34(t *testing.T, db *Database) {
	versionStr, err := getServerVersion(db)
	if err != nil {
		t.Fatalf("error getting server version: %s", err)
	}
	if compareVersions(t, versionStr, "3.4") < 0 {
		t.Skip("skipping collation test for server version < 3.4")
	}
}

func initCollection(t *testing.T, coll *Collection) {
	docs := []interface{}{}
	var i int32
	for i = 1; i <= 5; i++ {
		docs = append(docs, bsonx.Doc{{"x", bsonx.Int32(i)}})
	}

	_, err := coll.InsertMany(ctx, docs)
	require.Nil(t, err)
}
