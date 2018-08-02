// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"testing"

	"fmt"
	"os"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/mongodb/mongo-go-driver/internal/testutil"
	"github.com/mongodb/mongo-go-driver/mongo/dbopt"
	"github.com/stretchr/testify/require"
)

func createTestDatabase(t *testing.T, name *string, opts ...dbopt.Option) *Database {
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

	opts := []dbopt.Option{dbopt.ReadPreference(rpPrimary), dbopt.ReadConcern(rcLocal), dbopt.WriteConcern(wc1),
		dbopt.ReadPreference(rpSecondary), dbopt.ReadConcern(rcMajority), dbopt.WriteConcern(wc2)}

	expectedDb := &Database{
		readConcern:    rcMajority,
		readPreference: rpSecondary,
		writeConcern:   wc2,
	}

	t.Run("IndividualOptions", func(t *testing.T) {
		// if options specified multiple times, last instance should take precedence
		db := createTestDatabase(t, &name, opts...)
		compareDbs(t, expectedDb, db)
	})

	t.Run("Bundle", func(t *testing.T) {
		db := createTestDatabase(t, &name, dbopt.BundleDatabase(opts...))
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
	db := client.Database(name, dbopt.WriteConcern(wc1))

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

func TestDatabase_RunCommand(t *testing.T) {
	t.Parallel()

	db := createTestDatabase(t, nil)

	result, err := db.RunCommand(context.Background(), bson.NewDocument(bson.EC.Int32("ismaster", 1)))
	require.NoError(t, err)

	isMaster, err := result.Lookup("ismaster")
	require.NoError(t, err)
	require.Equal(t, isMaster.Value().Type(), bson.TypeBoolean)
	require.Equal(t, isMaster.Value().Boolean(), true)

	ok, err := result.Lookup("ok")
	require.NoError(t, err)
	require.Equal(t, ok.Value().Type(), bson.TypeDouble)
	require.Equal(t, ok.Value().Double(), 1.0)
}

func TestDatabase_Drop(t *testing.T) {
	t.Parallel()

	name := "TestDatabase_Drop"

	db := createTestDatabase(t, &name)

	client := createTestClient(t)
	err := db.Drop(context.Background())
	require.NoError(t, err)
	list, err := client.ListDatabaseNames(context.Background(), nil)

	require.NoError(t, err)
	require.NotContains(t, list, name)

}

// creates 1 normal collection and 1 capped collection of size 64*1024
func setupListCollectionsDb(db *Database) (uncappedName string, cappedName string, err error) {
	uncappedName, cappedName = "listcoll_uncapped", "listcoll_capped"
	uncappedColl := db.Collection(uncappedName)

	_, err = db.RunCommand(
		context.Background(),
		bson.NewDocument(
			bson.EC.String("create", cappedName),
			bson.EC.Boolean("capped", true),
			bson.EC.Int32("size", 64*1024),
		),
	)
	if err != nil {
		return "", "", err
	}
	cappedColl := db.Collection(cappedName)

	id := objectid.New()
	want := bson.EC.ObjectID("_id", id)
	doc := bson.NewDocument(want, bson.EC.Int32("x", 1))

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
func verifyListCollections(cursor Cursor, uncappedName string, cappedName string, cappedOnly bool) (err error) {
	var uncappedFound bool
	var cappedFound bool

	for cursor.Next(context.Background()) {
		next := bson.NewDocument()
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

		if elemName != uncappedName && elemName != cappedName {
			return fmt.Errorf("incorrect collection name. got: %s. wanted: %s or %s", elemName, uncappedName,
				cappedName)
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

func listCollectionsTest(db *Database, cappedOnly bool) error {
	uncappedName, cappedName, err := setupListCollectionsDb(db)
	if err != nil {
		return err
	}

	var filter *bson.Document
	if cappedOnly {
		filter = bson.NewDocument(
			bson.EC.Boolean("options.capped", true),
		)
	}

	for i := 0; i < 10; i++ {
		cursor, err := db.ListCollections(context.Background(), filter)
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

func TestDatabase_ListCollections(t *testing.T) {
	rpPrimary := readpref.Primary()
	rpSecondary := readpref.Secondary()

	var listCollectionsTable = []struct {
		name             string
		expectedTopology string
		cappedOnly       bool
		rp               *readpref.ReadPref
	}{
		{"standalone_nofilter", "server", false, rpPrimary},
		{"standalone_filter", "server", true, rpPrimary},
		{"replicaset_nofilter", "replica_set", false, rpPrimary},
		{"replicaset_filter", "replica_set", true, rpPrimary},
		{"replicaset_secondary_nofilter", "replica_set", false, rpSecondary},
		{"replicaset_secondary_filter", "replica_set", true, rpSecondary},
		{"sharded_nofilter", "sharded_cluster", false, rpPrimary},
		{"sharded_filter", "sharded_cluster", true, rpPrimary},
	}

	for _, tt := range listCollectionsTable {
		t.Run(tt.name, func(t *testing.T) {
			if os.Getenv("topology") != tt.expectedTopology {
				t.Skip()
			}
			dbName := tt.name
			db := createTestDatabase(t, &dbName, dbopt.ReadPreference(tt.rp))

			defer func() {
				err := db.Drop(context.Background())
				require.NoError(t, err)
			}()

			err := listCollectionsTest(db, tt.cappedOnly)
			require.NoError(t, err)
		})
	}
}
