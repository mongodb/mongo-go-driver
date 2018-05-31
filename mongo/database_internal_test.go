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
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/internal/testutil"
	"github.com/stretchr/testify/require"
	"os"
)

func createTestDatabase(t *testing.T, name *string) *Database {
	if name == nil {
		db := testutil.DBName(t)
		name = &db
	}

	client := createTestClient(t)
	return client.Database(*name)
}

func TestDatabase_initialize(t *testing.T) {
	t.Parallel()

	name := "foo"

	db := createTestDatabase(t, &name)
	require.Equal(t, db.name, name)
	require.NotNil(t, db.client)
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

func TestDatabase_ListCollections(t *testing.T) {
	t.Parallel()
	dbName := "db_list_collection"
	db := createTestDatabase(t, &dbName)
	collName := "list_collections_name"
	coll := db.Collection(collName)
	require.Equal(t, coll.Name(), collName)
	require.NotNil(t, coll)
	cursor, err := db.ListCollections(context.Background(), nil)
	require.NoError(t, err)

	next := bson.NewDocument()

	for cursor.Next(context.Background()) {
		err = cursor.Decode(next)
		require.NoError(t, err)

		elem, err := next.LookupErr("name")
		require.NoError(t, err)
		if elem.Type() != bson.TypeString {
			t.Errorf("Incorrect type for 'name'. got %v; want %v", elem.Type(), bson.TypeString)
			t.FailNow()
		}
		if elem.StringValue() != collName {
			t.Errorf("Incorrect collection name. got %s: want %s", elem.StringValue(), collName)
			t.FailNow()
		}
		//Because we run it without nameOnly parameter we should check if another parameter is exist
		docType, err := next.LookupErr("type")
		require.NoError(t, err)
		if docType.StringValue() != "collections" {
			t.Errorf("Incorrect cursor type. got %s: want %s", docType.StringValue(), "collections")
			t.FailNow()
		}
	}
	defer func() {
		err := db.Drop(context.Background())
		require.NoError(t, err)
	}()
}

func TestDatabase_ListCollectionsOptions(t *testing.T) {
	t.Parallel()
	dbName := "db_list_collection_options"
	db := createTestDatabase(t, &dbName)
	collName := "list_collections_options"
	coll := db.Collection(collName)
	require.Equal(t, coll.Name(), collName)
	require.NotNil(t, coll)
	cursor, err := db.ListCollections(context.Background(), nil, option.OptNameOnly(true))
	require.NoError(t, err)

	next := bson.NewDocument()

	for cursor.Next(context.Background()) {
		err = cursor.Decode(next)
		require.NoError(t, err)

		elem, err := next.LookupErr("name")
		require.NoError(t, err)

		if elem.StringValue() != collName {
			t.Errorf("Incorrect collection name. got %s: want %s", elem.StringValue(), collName)
			t.FailNow()
		}

		// Because we run it with name only parameter we should check that there are no other parameters
		_, err = next.LookupErr("type")
		require.Error(t, err)
	}
	defer func() {
		err := db.Drop(context.Background())
		require.NoError(t, err)
	}()
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
	next := bson.NewDocument()
	uncappedFound, cappedFound := false, false

	for cursor.Next(context.Background()) {
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

	cursor, err := db.ListCollections(context.Background(), filter)
	if err != nil {
		return err
	}

	return verifyListCollections(cursor, uncappedName, cappedName, cappedOnly)
}

func TestDatabase_ListCollections_Standalone_NoFilter(t *testing.T) {
	t.Parallel()
	dbName := "db_list_collections_nofilter"
	db := createTestDatabase(t, &dbName)

	defer func() {
		err := db.Drop(context.Background())
		require.NoError(t, err)
	}()

	err := listCollectionsTest(db, false)
	require.NoError(t, err)
}

func TestDatabase_ListCollections_Standalone_Filter(t *testing.T) {
	t.Parallel()
	dbName := "db_list_collections_filter"
	db := createTestDatabase(t, &dbName)

	defer func() {
		err := db.Drop(context.Background())
		require.NoError(t, err)
	}()

	err := listCollectionsTest(db, true)
	require.NoError(t, err)
}

func TestDatabase_ListCollections_Primary_NoFilter(t *testing.T) {
	// TODO(GODRIVER-272): become primary using read preference
	if os.Getenv("TOPOLOGY") != "replica_set" {
		t.Skip()
	}

	t.Parallel()
	dbName := "db_list_collections_primary_nofilter"
	db := createTestDatabase(t, &dbName)

	defer func() {
		err := db.Drop(context.Background())
		require.NoError(t, err)
	}()

	err := listCollectionsTest(db, false)
	require.NoError(t, err)
}

func TestDatabase_ListCollections_Primary_Filter(t *testing.T) {
	if os.Getenv("TOPOLOGY") != "replica_set" {
		t.Skip()
	}

	t.Parallel()
	dbName := "db_list_collections_primary_filter"
	db := createTestDatabase(t, &dbName)

	defer func() {
		err := db.Drop(context.Background())
		require.NoError(t, err)
	}()

	err := listCollectionsTest(db, true)
	require.NoError(t, err)
}

func TestDatabase_ListCollections_Secondary_NoFilter(t *testing.T) {
	if os.Getenv("TOPOLOGY") != "replica_set" {
		t.Skip()
	}

	t.Parallel()
	dbName := "db_list_collections_secondary_nofilter"
	db := createTestDatabase(t, &dbName)

	defer func() {
		err := db.Drop(context.Background())
		require.NoError(t, err)
	}()

	err := listCollectionsTest(db, false)
	require.NoError(t, err)
}

func TestDatabase_ListCollections_Secondary_Filter(t *testing.T) {
	if os.Getenv("TOPOLOGY") != "replica_set" {
		t.Skip()
	}

	t.Parallel()
	dbName := "db_list_collections_secondary_filter"
	db := createTestDatabase(t, &dbName)

	defer func() {
		err := db.Drop(context.Background())
		require.NoError(t, err)
	}()

	err := listCollectionsTest(db, true)
	require.NoError(t, err)
}

func TestDatabase_ListCollections_Sharded_Nofilter(t *testing.T) {
	if os.Getenv("TOPOLOGY") != "sharded_cluster" {
		t.Skip()
	}

	t.Parallel()
	dbName := "db_list_collections_sharded_nofilter"
	db := createTestDatabase(t, &dbName)

	defer func() {
		err := db.Drop(context.Background())
		require.NoError(t, err)
	}()

	err := listCollectionsTest(db, true)
	require.NoError(t, err)
}

func TestDatabase_ListCollections_Sharded_Filter(t *testing.T) {
	if os.Getenv("TOPOLOGY") != "sharded_cluster" {
		t.Skip()
	}

	t.Parallel()
	dbName := "db_list_collections_sharded_filter"
	db := createTestDatabase(t, &dbName)

	defer func() {
		err := db.Drop(context.Background())
		require.NoError(t, err)
	}()

	err := listCollectionsTest(db, true)
	require.NoError(t, err)
}
