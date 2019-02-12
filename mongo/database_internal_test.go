// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"errors"
	"testing"

	"fmt"
	"os"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"go.mongodb.org/mongo-driver/x/network/connstring"
	"go.mongodb.org/mongo-driver/x/network/description"
)

func createTestDatabase(t *testing.T, name *string, opts ...*options.DatabaseOptions) *Database {
	if name == nil {
		db := testutil.DBName(t)
		name = &db
	}

	client := createTestClient(t)
	return client.Database(*name, opts...)
}

func TestDatabase_initialize(t *testing.T) {
	t.Parallel()

	name := "foo"

	db := createTestDatabase(t, &name)
	require.Equal(t, db.name, name)
	require.NotNil(t, db.client)
}

func compareDbs(t *testing.T, expected *Database, got *Database) {
	switch {
	case expected.readPreference != got.readPreference:
		t.Errorf("expected read preference %#v. got %#v", expected.readPreference, got.readPreference)
	case expected.readConcern != got.readConcern:
		t.Errorf("expected read concern %#v. got %#v", expected.readConcern, got.readConcern)
	case expected.writeConcern != got.writeConcern:
		t.Errorf("expected write concern %#v. got %#v", expected.writeConcern, got.writeConcern)
	}
}

func TestDatabase_Options(t *testing.T) {
	name := "testDb_options"
	rpPrimary := readpref.Primary()
	rpSecondary := readpref.Secondary()
	wc1 := writeconcern.New(writeconcern.W(5))
	wc2 := writeconcern.New(writeconcern.W(10))
	rcLocal := readconcern.Local()
	rcMajority := readconcern.Majority()

	opts := options.Database().SetReadPreference(rpPrimary).SetReadConcern(rcLocal).SetWriteConcern(wc1).
		SetReadPreference(rpSecondary).SetReadConcern(rcMajority).SetWriteConcern(wc2)

	expectedDb := &Database{
		readConcern:    rcMajority,
		readPreference: rpSecondary,
		writeConcern:   wc2,
	}

	t.Run("IndividualOptions", func(t *testing.T) {
		// if options specified multiple times, last instance should take precedence
		db := createTestDatabase(t, &name, opts)
		compareDbs(t, expectedDb, db)
	})
}

func TestDatabase_InheritOptions(t *testing.T) {
	name := "testDb_options_inherit"
	client := createTestClient(t)

	rpPrimary := readpref.Primary()
	rcLocal := readconcern.Local()
	client.readPreference = rpPrimary
	client.readConcern = rcLocal

	wc1 := writeconcern.New(writeconcern.W(10))
	db := client.Database(name, options.Database().SetWriteConcern(wc1))

	// db should inherit read preference and read concern from client
	switch {
	case db.readPreference != rpPrimary:
		t.Errorf("expected read preference primary. got %#v", db.readPreference)
	case db.readConcern != rcLocal:
		t.Errorf("expected read concern local. got %#v", db.readConcern)
	case db.writeConcern != wc1:
		t.Errorf("expected write concern %#v. got %#v", wc1, db.writeConcern)
	}
}

func TestDatabase_ReplaceTopologyError(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip()
	}

	cs := testutil.ConnString(t)
	c, err := NewClient(options.Client().ApplyURI(cs.String()))
	require.NoError(t, err)
	require.NotNil(t, c)

	db := c.Database("TestDatabase_ReplaceTopologyError")

	err = db.RunCommand(context.Background(), bsonx.Doc{{"ismaster", bsonx.Int32(1)}}).Err()
	require.Equal(t, err, ErrClientDisconnected)

	err = db.Drop(ctx)
	require.Equal(t, err, ErrClientDisconnected)

	_, err = db.ListCollections(ctx, bsonx.Doc{})
	require.Equal(t, err, ErrClientDisconnected)
}

func TestDatabase_RunCommand(t *testing.T) {
	t.Parallel()

	db := createTestDatabase(t, nil)

	var result bsonx.Doc
	err := db.RunCommand(context.Background(), bsonx.Doc{{"ismaster", bsonx.Int32(1)}}).Decode(&result)
	require.NoError(t, err)

	isMaster, err := result.LookupErr("ismaster")
	require.NoError(t, err)
	require.Equal(t, isMaster.Type(), bson.TypeBoolean)
	require.Equal(t, isMaster.Boolean(), true)

	ok, err := result.LookupErr("ok")
	require.NoError(t, err)
	require.Equal(t, ok.Type(), bson.TypeDouble)
	require.Equal(t, ok.Double(), 1.0)
}

func TestDatabase_RunCommand_DecodeStruct(t *testing.T) {
	t.Parallel()

	db := createTestDatabase(t, nil)

	result := struct {
		Ismaster bool    `bson:"ismaster"`
		Ok       float64 `bson:"ok"`
	}{}

	err := db.RunCommand(context.Background(), bsonx.Doc{{"ismaster", bsonx.Int32(1)}}).Decode(&result)
	require.NoError(t, err)
	require.Equal(t, result.Ismaster, true)
	require.Equal(t, result.Ok, 1.0)
}

func TestDatabase_NilDocumentError(t *testing.T) {
	t.Parallel()

	db := createTestDatabase(t, nil)

	err := db.RunCommand(context.Background(), nil).Err()
	require.Equal(t, err, ErrNilDocument)

	_, err = db.Watch(context.Background(), nil)
	require.Equal(t, err, errors.New("can only transform slices and arrays into aggregation pipelines, but got invalid"))

	_, err = db.ListCollections(context.Background(), nil)
	require.Equal(t, err, ErrNilDocument)
}

func TestDatabase_Drop(t *testing.T) {
	t.Parallel()

	name := "TestDatabase_Drop"

	db := createTestDatabase(t, &name)

	client := createTestClient(t)
	err := db.Drop(context.Background())
	require.NoError(t, err)
	list, err := client.ListDatabaseNames(context.Background(), bsonx.Doc{})

	require.NoError(t, err)
	require.NotContains(t, list, name)

}

// creates 1 normal collection and 1 capped collection of size 64*1024
func setupListCollectionsDb(db *Database) (uncappedName string, cappedName string, err error) {
	uncappedName, cappedName = "listcoll_uncapped", "listcoll_capped"
	uncappedColl := db.Collection(uncappedName)

	err = db.RunCommand(
		context.Background(),
		bsonx.Doc{
			{"create", bsonx.String(cappedName)},
			{"capped", bsonx.Boolean(true)},
			{"size", bsonx.Int32(64 * 1024)},
		},
	).Err()
	if err != nil {
		return "", "", err
	}
	cappedColl := db.Collection(cappedName)

	id := primitive.NewObjectID()
	want := bsonx.Elem{"_id", bsonx.ObjectID(id)}
	doc := bsonx.Doc{want, {"x", bsonx.Int32(1)}}

	_, err = uncappedColl.InsertOne(context.Background(), doc)
	if err != nil {
		return "", "", err
	}

	_, err = cappedColl.InsertOne(context.Background(), doc)
	if err != nil {
		return "", "", err
	}

	return uncappedName, cappedName, nil
}

// verifies both collection names are found in cursor, cursor does not have extra collections, and cursor has no
// duplicates
func verifyListCollections(cursor *Cursor, uncappedName string, cappedName string, cappedOnly bool) (err error) {
	var uncappedFound bool
	var cappedFound bool

	for cursor.Next(context.Background()) {
		next := &bsonx.Doc{}
		err = cursor.Decode(next)
		if err != nil {
			return err
		}

		elem, err := next.LookupErr("name")
		if err != nil {
			return err
		}

		if elem.Type() != bson.TypeString {
			return fmt.Errorf("incorrect type for 'name'. got %v. want %v", elem.Type(), bson.TypeString)
		}

		elemName := elem.StringValue()

		// legacy servers can return an indexes collection that shouldn't be considered here
		if elemName != cappedName && elemName != uncappedName {
			continue
		}

		if elemName == uncappedName && !uncappedFound {
			if cappedOnly {
				return fmt.Errorf("found uncapped collection %s. expected only capped collections", uncappedName)
			}

			uncappedFound = true
			continue
		}

		if elemName == cappedName && !cappedFound {
			cappedFound = true
			continue
		}

		// duplicate found
		return fmt.Errorf("found duplicate collection %s", elemName)
	}

	if !cappedFound {
		return fmt.Errorf("did not find collection %s", cappedName)
	}

	if !cappedOnly && !uncappedFound {
		return fmt.Errorf("did not find collection %s", uncappedName)
	}

	return nil
}

func listCollectionsTest(db *Database, cappedOnly bool, cappedName, uncappedName string) error {
	var filter bsonx.Doc
	if cappedOnly {
		filter = bsonx.Doc{{"options.capped", bsonx.Boolean(true)}}
	}

	var cursor *Cursor
	var err error
	for i := 0; i < 10; i++ {
		cursor, err = db.ListCollections(context.Background(), filter)
		if err != nil {
			return err
		}

		err = verifyListCollections(cursor, uncappedName, cappedName, cappedOnly)
		if err == nil {
			return nil
		}
	}

	return err // all tests failed
}

// get the connection string for a direct connection to a secondary in a replica set
func getSecondaryConnString(t *testing.T) connstring.ConnString {
	topo := testutil.Topology(t)
	for _, server := range topo.Description().Servers {
		if server.Kind != description.RSSecondary {
			continue
		}

		fullAddr := "mongodb://" + server.Addr.String() + "/?connect=direct"
		cs, err := connstring.Parse(fullAddr)
		require.NoError(t, err)
		return cs
	}

	t.Fatalf("no secondary found for %s", t.Name())
	return connstring.ConnString{}
}

func TestDatabase_ListCollections(t *testing.T) {
	var listCollectionsTable = []struct {
		name             string
		expectedTopology string
		cappedOnly       bool
		direct           bool
	}{
		{"standalone_nofilter", "server", false, false},
		{"standalone_filter", "server", true, false},
		{"replicaset_nofilter", "replica_set", false, false},
		{"replicaset_filter", "replica_set", true, false},
		{"replicaset_secondary_nofilter", "replica_set", false, true},
		{"replicaset_secondary_filter", "replica_set", true, true},
		{"sharded_nofilter", "sharded_cluster", false, false},
		{"sharded_filter", "sharded_cluster", true, false},
	}

	for _, tt := range listCollectionsTable {
		t.Run(tt.name, func(t *testing.T) {
			if os.Getenv("TOPOLOGY") != tt.expectedTopology {
				t.Skip()
			}

			createDb := createTestDatabase(t, &tt.name, options.Database().SetWriteConcern(wcMajority))
			defer func() {
				err := createDb.Drop(context.Background())
				require.NoError(t, err)
			}()

			uncappedName, cappedName, err := setupListCollectionsDb(createDb)
			require.NoError(t, err)

			var cs connstring.ConnString
			if tt.direct {
				// TODO(GODRIVER-641) - correctly set read preference on direct connections for OP_MSG
				t.Skip()
				cs = getSecondaryConnString(t)
			} else {
				cs = testutil.ConnString(t)
			}

			client := createTestClientWithConnstring(t, cs)
			db := client.Database(tt.name)

			err = listCollectionsTest(db, tt.cappedOnly, cappedName, uncappedName)
			require.NoError(t, err)
		})
	}
}
