// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/mongodb/mongo-go-driver/internal/testutil"
	"github.com/mongodb/mongo-go-driver/mongo/aggregateopt"
	"github.com/mongodb/mongo-go-driver/mongo/collectionopt"
	"github.com/mongodb/mongo-go-driver/mongo/countopt"
	"github.com/mongodb/mongo-go-driver/mongo/deleteopt"
	"github.com/mongodb/mongo-go-driver/mongo/distinctopt"
	"github.com/mongodb/mongo-go-driver/mongo/findopt"
	"github.com/mongodb/mongo-go-driver/mongo/insertopt"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
	"github.com/mongodb/mongo-go-driver/mongo/replaceopt"
	"github.com/mongodb/mongo-go-driver/mongo/updateopt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestCollection(t *testing.T, dbName *string, collName *string, opts ...collectionopt.Option) *Collection {
	if collName == nil {
		coll := testutil.ColName(t)
		collName = &coll
	}

	db := createTestDatabase(t, dbName)

	return db.Collection(*collName, opts...)
}

func initCollection(t *testing.T, coll *Collection) {
	doc1 := bson.NewDocument(bson.EC.Int32("x", 1))
	doc2 := bson.NewDocument(bson.EC.Int32("x", 2))
	doc3 := bson.NewDocument(bson.EC.Int32("x", 3))
	doc4 := bson.NewDocument(bson.EC.Int32("x", 4))
	doc5 := bson.NewDocument(bson.EC.Int32("x", 5))

	var err error

	_, err = coll.InsertOne(context.Background(), doc1)
	require.Nil(t, err)

	_, err = coll.InsertOne(context.Background(), doc2)
	require.Nil(t, err)

	_, err = coll.InsertOne(context.Background(), doc3)
	require.Nil(t, err)

	_, err = coll.InsertOne(context.Background(), doc4)
	require.Nil(t, err)

	_, err = coll.InsertOne(context.Background(), doc5)
	require.Nil(t, err)
}

func TestCollection_initialize(t *testing.T) {
	t.Parallel()

	dbName := "foo"
	collName := "bar"

	coll := createTestCollection(t, &dbName, &collName)
	require.Equal(t, coll.name, collName)
	require.NotNil(t, coll.db)
}

func compareColls(t *testing.T, expected *Collection, got *Collection) {
	switch {
	case expected.readPreference != got.readPreference:
		t.Errorf("expected read preference %#v. got %#v", expected.readPreference, got.readPreference)
	case expected.readConcern != got.readConcern:
		t.Errorf("expected read concern %#v. got %#v", expected.readConcern, got.readConcern)
	case expected.writeConcern != got.writeConcern:
		t.Errorf("expected write concern %#v. got %#v", expected.writeConcern, got.writeConcern)
	}
}

func TestCollection_Options(t *testing.T) {
	name := "testDb_options"
	rpPrimary := readpref.Primary()
	rpSecondary := readpref.Secondary()
	wc1 := writeconcern.New(writeconcern.W(5))
	wc2 := writeconcern.New(writeconcern.W(10))
	rcLocal := readconcern.Local()
	rcMajority := readconcern.Majority()

	opts := []collectionopt.Option{collectionopt.ReadPreference(rpPrimary), collectionopt.ReadConcern(rcLocal), collectionopt.WriteConcern(wc1),
		collectionopt.ReadPreference(rpSecondary), collectionopt.ReadConcern(rcMajority), collectionopt.WriteConcern(wc2)}

	dbName := "collection_internal_test_db1"

	expectedColl := &Collection{
		readConcern:    rcMajority,
		readPreference: rpSecondary,
		writeConcern:   wc2,
	}

	t.Run("IndividualOptions", func(t *testing.T) {
		// if options specified multiple times, last instance should take precedence
		coll := createTestCollection(t, &dbName, &name, opts...)
		compareColls(t, expectedColl, coll)
	})

	t.Run("Bundle", func(t *testing.T) {
		coll := createTestCollection(t, &dbName, &name, collectionopt.BundleCollection(opts...))
		compareColls(t, expectedColl, coll)
	})
}

func TestCollection_InheritOptions(t *testing.T) {
	name := "testDb_options_inherit"
	client := createTestClient(t)

	rpPrimary := readpref.Primary()
	rcLocal := readconcern.Local()
	wc1 := writeconcern.New(writeconcern.W(10))

	db := client.Database("collection_internal_test_db2")
	db.readPreference = rpPrimary
	db.readConcern = rcLocal
	coll := db.Collection(name, collectionopt.WriteConcern(wc1))

	// coll should inherit read preference and read concern from client
	switch {
	case coll.readPreference != rpPrimary:
		t.Errorf("expected read preference primary. got %#v", coll.readPreference)
	case coll.readConcern != rcLocal:
		t.Errorf("expected read concern local. got %#v", coll.readConcern)
	case coll.writeConcern != wc1:
		t.Errorf("expected write concern %#v. got %#v", wc1, coll.writeConcern)
	}
}

func TestCollection_namespace(t *testing.T) {
	t.Parallel()

	dbName := "foo"
	collName := "bar"

	coll := createTestCollection(t, &dbName, &collName)
	namespace := coll.namespace()
	require.Equal(t, namespace.FullName(), fmt.Sprintf("%s.%s", dbName, collName))
}

func TestCollection_name_accessor(t *testing.T) {
	t.Parallel()

	dbName := "foo"
	collName := "bar"

	coll := createTestCollection(t, &dbName, &collName)
	namespace := coll.namespace()
	require.Equal(t, coll.Name(), collName)
	require.Equal(t, coll.Name(), namespace.Collection)

}

func TestCollection_InsertOne(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	id := objectid.New()
	want := bson.EC.ObjectID("_id", id)
	doc := bson.NewDocument(want, bson.EC.Int32("x", 1))
	coll := createTestCollection(t, nil, nil)

	result, err := coll.InsertOne(context.Background(), doc)
	require.Nil(t, err)
	require.Equal(t, result.InsertedID, want)
}

func TestCollection_InsertOne_WriteError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	want := WriteError{Code: 11000}
	doc := bson.NewDocument(bson.EC.ObjectID("_id", objectid.New()))
	coll := createTestCollection(t, nil, nil)

	_, err := coll.InsertOne(context.Background(), doc)
	require.NoError(t, err)
	_, err = coll.InsertOne(context.Background(), doc)
	got, ok := err.(WriteErrors)
	if !ok {
		t.Errorf("Did not receive correct type of error. got %T; want %T", err, WriteErrors{})
	}
	if len(got) != 1 {
		t.Errorf("Incorrect number of errors receieved. got %d; want %d", len(got), 1)
		t.FailNow()
	}
	if got[0].Code != want.Code {
		t.Errorf("Did not receive the correct error code. got %d; want %d", got[0].Code, want.Code)
	}

}

func TestCollection_InsertOne_WriteConcernError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	if os.Getenv("TOPOLOGY") != "replica_set" {
		t.Skip()
	}

	want := WriteConcernError{Code: 100, Message: "Not enough data-bearing nodes"}
	doc := bson.NewDocument(bson.EC.ObjectID("_id", objectid.New()))
	coll := createTestCollection(t, nil, nil, collectionopt.WriteConcern(writeconcern.New(writeconcern.W(25))))

	_, err := coll.InsertOne(context.Background(), doc)
	got, ok := err.(WriteConcernError)
	if !ok {
		t.Errorf("Did not receive correct type of error. got %T; want %T", err, WriteConcernError{})
	}
	if got.Code != want.Code {
		t.Errorf("Did not receive the correct error code. got %d; want %d", got.Code, want.Code)
	}
	if got.Message != want.Message {
		t.Errorf("Did not receive the correct error message. got %s; want %s", got.Message, want.Message)
	}
}

func TestCollection_InsertMany(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	want1 := bson.EC.Int32("_id", 11)
	want2 := bson.EC.Int32("_id", 12)
	docs := []interface{}{
		bson.NewDocument(want1),
		bson.NewDocument(bson.EC.Int32("x", 6)),
		bson.NewDocument(want2),
	}
	coll := createTestCollection(t, nil, nil)

	result, err := coll.InsertMany(context.Background(), docs)
	require.Nil(t, err)

	require.Len(t, result.InsertedIDs, 3)
	require.Equal(t, result.InsertedIDs[0], want1)
	require.NotNil(t, result.InsertedIDs[1])
	require.Equal(t, result.InsertedIDs[2], want2)
}

func TestCollection_InsertMany_Batches(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// TODO(GODRIVER-425): remove this as part a larger project to
	// refactor integration and other longrunning tasks.
	if os.Getenv("EVR_TASK_ID") == "" {
		t.Skip("skipping long running integration test outside of evergreen")
	}

	t.Parallel()

	const (
		megabyte = 10 * 10 * 10 * 10 * 10 * 10
		numDocs  = 700000
	)

	docs := []interface{}{}
	total := uint32(0)
	expectedDocSize := uint32(26)
	for i := 0; i < numDocs; i++ {
		d := bson.NewDocument(
			bson.EC.Int32("a", int32(i)),
			bson.EC.Int32("b", int32(i*2)),
			bson.EC.Int32("c", int32(i*3)),
		)
		len, err := d.Validate()
		require.NoError(t, err)
		require.Equal(t, expectedDocSize, len, "len=%d expected=%d", len, expectedDocSize)
		docs = append(docs, d)
		total += len
	}
	assert.True(t, total > 16*megabyte)
	coll := createTestCollection(t, nil, nil)

	result, err := coll.InsertMany(context.Background(), docs)
	require.Nil(t, err)
	require.Len(t, result.InsertedIDs, numDocs)
}

func TestCollection_InsertMany_ErrorCases(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	want := WriteError{Code: 11000}
	docs := []interface{}{
		bson.NewDocument(bson.EC.ObjectID("_id", objectid.New())),
		bson.NewDocument(bson.EC.ObjectID("_id", objectid.New())),
		bson.NewDocument(bson.EC.ObjectID("_id", objectid.New())),
	}
	coll := createTestCollection(t, nil, nil)

	t.Run("insert_batch_unordered", func(t *testing.T) {
		_, err := coll.InsertMany(context.Background(), docs)
		require.NoError(t, err)

		// without option ordered
		_, err = coll.InsertMany(context.Background(), docs, insertopt.Ordered(false))
		got, ok := err.(BulkWriteError)
		if !ok {
			t.Errorf("Did not receive correct type of error. got %T; want %T", err, WriteErrors{})
			t.FailNow()
		}
		if len(got.WriteErrors) != 3 {
			t.Errorf("Incorrect number of errors receieved. got %d; want %d", len(got.WriteErrors), 3)
			t.FailNow()
		}
		if got.WriteErrors[0].Code != want.Code {
			t.Errorf("Did not receive the correct error code. got %d; want %d", got.WriteErrors[0].Code, want.Code)
		}
	})
	t.Run("insert_batch_ordered_write_error", func(t *testing.T) {
		// run the insert again to ensure that the documents
		// are there in cases when this case is run
		// independently of the previous test
		_, _ = coll.InsertMany(context.Background(), docs)

		// with the ordered option (default, we should only get one write error)
		_, err := coll.InsertMany(context.Background(), docs)
		got, ok := err.(BulkWriteError)
		if !ok {
			t.Errorf("Did not receive correct type of error. got %T; want %T", err, WriteErrors{})
			t.FailNow()
		}
		if len(got.WriteErrors) != 1 {
			t.Errorf("Incorrect number of errors receieved. got %d; want %d", len(got.WriteErrors), 1)
			t.FailNow()
		}
		if got.WriteErrors[0].Code != want.Code {
			t.Errorf("Did not receive the correct error code. got %d; want %d", got.WriteErrors[0].Code, want.Code)
		}

	})
	t.Run("insert_batch_write_concern_error", func(t *testing.T) {
		if os.Getenv("TOPOLOGY") != "replica_set" {
			t.Skip()
		}

		docs = []interface{}{
			bson.NewDocument(bson.EC.ObjectID("_id", objectid.New())),
			bson.NewDocument(bson.EC.ObjectID("_id", objectid.New())),
			bson.NewDocument(bson.EC.ObjectID("_id", objectid.New())),
		}

		copyColl, err := coll.Clone(collectionopt.WriteConcern(writeconcern.New(writeconcern.W(42))))
		if err != nil {
			t.Errorf("err copying collection: %s", err)
		}

		_, err = copyColl.InsertMany(context.Background(), docs)
		if err == nil {
			t.Errorf("write concern error not propagated from command: %+v", err)
		}
		bulkErr, ok := err.(BulkWriteError)
		if !ok {
			t.Errorf("incorrect error type returned: %T", err)
		}
		if bulkErr.WriteConcernError == nil {
			t.Errorf("write concern error is nil: %+v", bulkErr)
		}
	})
}

func TestCollection_InsertMany_WriteConcernError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	if os.Getenv("TOPOLOGY") != "replica_set" {
		t.Skip()
	}

	want := WriteConcernError{Code: 100, Message: "Not enough data-bearing nodes"}
	docs := []interface{}{
		bson.NewDocument(bson.EC.ObjectID("_id", objectid.New())),
		bson.NewDocument(bson.EC.ObjectID("_id", objectid.New())),
		bson.NewDocument(bson.EC.ObjectID("_id", objectid.New())),
	}
	coll := createTestCollection(t, nil, nil, collectionopt.WriteConcern(writeconcern.New(writeconcern.W(25))))

	_, err := coll.InsertMany(context.Background(), docs)
	got, ok := err.(BulkWriteError)
	if !ok {
		t.Errorf("Did not receive correct type of error. got %T; want %T\nError message: %s", err, BulkWriteError{}, err)
	}
	if got.WriteConcernError.Code != want.Code {
		t.Errorf("Did not receive the correct error code. got %d; want %d", got.WriteConcernError.Code, want.Code)
	}
	if got.WriteConcernError.Message != want.Message {
		t.Errorf("Did not receive the correct error message. got %s; want %s", got.WriteConcernError.Message, want.Message)
	}
}

func TestCollection_DeleteOne_found(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(bson.EC.Int32("x", 1))
	result, err := coll.DeleteOne(context.Background(), filter)
	require.Nil(t, err)
	require.NotNil(t, result)
	require.Equal(t, result.DeletedCount, int64(1))
}

func TestCollection_DeleteOne_notFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(bson.EC.Int32("x", 0))
	result, err := coll.DeleteOne(context.Background(), filter)
	require.Nil(t, err)
	require.Equal(t, result.DeletedCount, int64(0))
}

func TestCollection_DeleteOne_notFound_withOption(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(bson.EC.Int32("x", 0))

	collationOpt := &mongoopt.Collation{
		Locale: "en_US",
	}
	result, err := coll.DeleteOne(context.Background(), filter, deleteopt.Collation(collationOpt))
	require.Nil(t, err)
	require.Equal(t, result.DeletedCount, int64(0))
}

func TestCollection_DeleteOne_WriteError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	want := WriteError{Code: 20}
	filter := bson.NewDocument(bson.EC.Int32("x", 1))
	db := createTestDatabase(t, nil)
	_, err := db.RunCommand(
		context.Background(),
		bson.NewDocument(
			bson.EC.String("create", testutil.ColName(t)),
			bson.EC.Boolean("capped", true),
			bson.EC.Int32("size", 64*1024),
		),
	)
	require.NoError(t, err)
	coll := db.Collection(testutil.ColName(t))

	_, err = coll.DeleteOne(context.Background(), filter)
	got, ok := err.(WriteErrors)
	if !ok {
		t.Errorf("Did not receive correct type of error. got %T; want %T", err, WriteErrors{})
	}
	if len(got) != 1 {
		t.Errorf("Incorrect number of errors receieved. got %d; want %d", len(got), 1)
		t.FailNow()
	}
	if got[0].Code != want.Code {
		t.Errorf("Did not receive the correct error code. got %d; want %d", got[0].Code, want.Code)
	}
}

func TestCollection_DeleteMany_WriteConcernError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	if os.Getenv("TOPOLOGY") != "replica_set" {
		t.Skip()
	}

	want := WriteConcernError{Code: 100, Message: "Not enough data-bearing nodes"}
	filter := bson.NewDocument(bson.EC.Int32("x", 1))
	coll := createTestCollection(t, nil, nil, collectionopt.WriteConcern(writeconcern.New(writeconcern.W(25))))

	_, err := coll.DeleteOne(context.Background(), filter)
	got, ok := err.(WriteConcernError)
	if !ok {
		t.Errorf("Did not receive correct type of error. got %T; want %T", err, WriteConcernError{})
	}
	if got.Code != want.Code {
		t.Errorf("Did not receive the correct error code. got %d; want %d", got.Code, want.Code)
	}
	if got.Message != want.Message {
		t.Errorf("Did not receive the correct error message. got %s; want %s", got.Message, want.Message)
	}
}

func TestCollection_DeleteMany_found(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(
		bson.EC.SubDocumentFromElements("x", bson.EC.Int32("$gte", 3)))

	result, err := coll.DeleteMany(context.Background(), filter)
	require.Nil(t, err)
	require.Equal(t, result.DeletedCount, int64(3))
}

func TestCollection_DeleteMany_notFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(
		bson.EC.SubDocumentFromElements("x", bson.EC.Int32("$lt", 1)))

	result, err := coll.DeleteMany(context.Background(), filter)
	require.Nil(t, err)
	require.Equal(t, result.DeletedCount, int64(0))
}

func TestCollection_DeleteMany_notFound_withOption(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(
		bson.EC.SubDocumentFromElements("x", bson.EC.Int32("$lt", 1)))

	result, err := coll.DeleteMany(context.Background(), filter, deleteopt.Collation(&mongoopt.Collation{Locale: "en_US"}))
	require.Nil(t, err)
	require.Equal(t, result.DeletedCount, int64(0))
}

func TestCollection_DeleteMany_WriteError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	want := WriteError{Code: 20}
	filter := bson.NewDocument(bson.EC.Int32("x", 1))
	db := createTestDatabase(t, nil)
	_, err := db.RunCommand(
		context.Background(),
		bson.NewDocument(
			bson.EC.String("create", testutil.ColName(t)),
			bson.EC.Boolean("capped", true),
			bson.EC.Int32("size", 64*1024),
		),
	)
	require.NoError(t, err)
	coll := db.Collection(testutil.ColName(t))

	_, err = coll.DeleteMany(context.Background(), filter)
	got, ok := err.(WriteErrors)
	if !ok {
		t.Errorf("Did not receive correct type of error. got %T; want %T", err, WriteErrors{})
	}
	if len(got) != 1 {
		t.Errorf("Incorrect number of errors receieved. got %d; want %d", len(got), 1)
		t.FailNow()
	}
	if got[0].Code != want.Code {
		t.Errorf("Did not receive the correct error code. got %d; want %d", got[0].Code, want.Code)
	}
}

func TestCollection_DeleteOne_WriteConcernError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	if os.Getenv("TOPOLOGY") != "replica_set" {
		t.Skip()
	}

	want := WriteConcernError{Code: 100, Message: "Not enough data-bearing nodes"}
	filter := bson.NewDocument(bson.EC.Int32("x", 1))
	coll := createTestCollection(t, nil, nil, collectionopt.WriteConcern(writeconcern.New(writeconcern.W(25))))

	_, err := coll.DeleteMany(context.Background(), filter)
	got, ok := err.(WriteConcernError)
	if !ok {
		t.Errorf("Did not receive correct type of error. got %T; want %T", err, WriteConcernError{})
	}
	if got.Code != want.Code {
		t.Errorf("Did not receive the correct error code. got %d; want %d", got.Code, want.Code)
	}
	if got.Message != want.Message {
		t.Errorf("Did not receive the correct error message. got %s; want %s", got.Message, want.Message)
	}
}

func TestCollection_UpdateOne_found(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(bson.EC.Int32("x", 1))
	update := bson.NewDocument(
		bson.EC.SubDocumentFromElements("$inc", bson.EC.Int32("x", 1)))

	result, err := coll.UpdateOne(context.Background(), filter, update)
	require.Nil(t, err)
	require.NotNil(t, result)
	require.Equal(t, result.MatchedCount, int64(1))
	require.Equal(t, result.ModifiedCount, int64(1))
	require.Nil(t, result.UpsertedID)
}

func TestCollection_UpdateOne_notFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(bson.EC.Int32("x", 0))
	update := bson.NewDocument(
		bson.EC.SubDocumentFromElements("$inc", bson.EC.Int32("x", 1)))

	result, err := coll.UpdateOne(context.Background(), filter, update)
	require.Nil(t, err)
	require.Equal(t, result.MatchedCount, int64(0))
	require.Equal(t, result.ModifiedCount, int64(0))
	require.Nil(t, result.UpsertedID)
}

func TestCollection_UpdateOne_upsert(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(bson.EC.Int32("x", 0))
	update := bson.NewDocument(
		bson.EC.SubDocumentFromElements("$inc", bson.EC.Int32("x", 1)))

	result, err := coll.UpdateOne(context.Background(), filter, update, updateopt.Upsert(true))
	require.Nil(t, err)
	require.Equal(t, result.MatchedCount, int64(0))
	require.Equal(t, result.ModifiedCount, int64(0))
	require.NotNil(t, result.UpsertedID)
}

func TestCollection_UpdateOne_WriteError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	want := WriteError{Code: 66}
	filter := bson.NewDocument(bson.EC.String("_id", "foo"))
	update := bson.NewDocument(
		bson.EC.SubDocumentFromElements(
			"$set",
			bson.EC.Double("_id", 3.14159),
		),
	)
	coll := createTestCollection(t, nil, nil)

	_, err := coll.InsertOne(context.Background(), bson.NewDocument(bson.EC.String("_id", "foo")))
	require.NoError(t, err)

	_, err = coll.UpdateOne(context.Background(), filter, update)
	got, ok := err.(WriteErrors)
	if !ok {
		t.Errorf("Did not receive correct type of error. got %T; want %T", err, WriteErrors{})
	}
	if len(got) != 1 {
		t.Errorf("Incorrect number of errors receieved. got %d; want %d", len(got), 1)
		t.FailNow()
	}
	if got[0].Code != want.Code {
		t.Errorf("Did not receive the correct error code. got %d; want %d", got[0].Code, want.Code)
	}
}

func TestCollection_UpdateOne_WriteConcernError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	if os.Getenv("TOPOLOGY") != "replica_set" {
		t.Skip()
	}

	want := WriteConcernError{Code: 100, Message: "Not enough data-bearing nodes"}
	filter := bson.NewDocument(bson.EC.String("_id", "foo"))
	update := bson.NewDocument(
		bson.EC.SubDocumentFromElements(
			"$set",
			bson.EC.Double("pi", 3.14159),
		),
	)
	coll := createTestCollection(t, nil, nil, collectionopt.WriteConcern(writeconcern.New(writeconcern.W(25))))

	_, err := coll.UpdateOne(context.Background(), filter, update)
	got, ok := err.(WriteConcernError)
	if !ok {
		t.Errorf("Did not receive correct type of error. got %T; want %T", err, WriteConcernError{})
	}
	if got.Code != want.Code {
		t.Errorf("Did not receive the correct error code. got %d; want %d", got.Code, want.Code)
	}
	if got.Message != want.Message {
		t.Errorf("Did not receive the correct error message. got %s; want %s", got.Message, want.Message)
	}
}

func TestCollection_UpdateMany_found(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(
		bson.EC.SubDocumentFromElements("x", bson.EC.Int32("$gte", 3)))

	update := bson.NewDocument(
		bson.EC.SubDocumentFromElements("$inc", bson.EC.Int32("x", 1)))

	result, err := coll.UpdateMany(context.Background(), filter, update)
	require.Nil(t, err)
	require.Equal(t, result.MatchedCount, int64(3))
	require.Equal(t, result.ModifiedCount, int64(3))
	require.Nil(t, result.UpsertedID)
}

func TestCollection_UpdateMany_notFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(
		bson.EC.SubDocumentFromElements("x", bson.EC.Int32("$lt", 1)))

	update := bson.NewDocument(
		bson.EC.SubDocumentFromElements("$inc", bson.EC.Int32("x", 1)))

	result, err := coll.UpdateMany(context.Background(), filter, update)
	require.Nil(t, err)
	require.Equal(t, result.MatchedCount, int64(0))
	require.Equal(t, result.ModifiedCount, int64(0))
	require.Nil(t, result.UpsertedID)
}

func TestCollection_UpdateMany_upsert(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(
		bson.EC.SubDocumentFromElements("x", bson.EC.Int32("$lt", 1)))

	update := bson.NewDocument(
		bson.EC.SubDocumentFromElements("$inc", bson.EC.Int32("x", 1)))

	result, err := coll.UpdateMany(context.Background(), filter, update, updateopt.Upsert(true))
	require.Nil(t, err)
	require.Equal(t, result.MatchedCount, int64(0))
	require.Equal(t, result.ModifiedCount, int64(0))
	require.NotNil(t, result.UpsertedID)
}

func TestCollection_UpdateMany_WriteError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	want := WriteError{Code: 66}
	filter := bson.NewDocument(bson.EC.String("_id", "foo"))
	update := bson.NewDocument(
		bson.EC.SubDocumentFromElements(
			"$set",
			bson.EC.Double("_id", 3.14159),
		),
	)
	coll := createTestCollection(t, nil, nil)

	_, err := coll.InsertOne(context.Background(), bson.NewDocument(bson.EC.String("_id", "foo")))
	require.NoError(t, err)

	_, err = coll.UpdateMany(context.Background(), filter, update)
	got, ok := err.(WriteErrors)
	if !ok {
		t.Errorf("Did not receive correct type of error. got %T; want %T", err, WriteErrors{})
	}
	if len(got) != 1 {
		t.Errorf("Incorrect number of errors receieved. got %d; want %d", len(got), 1)
		t.FailNow()
	}
	if got[0].Code != want.Code {
		t.Errorf("Did not receive the correct error code. got %d; want %d", got[0].Code, want.Code)
	}

}

func TestCollection_UpdateMany_WriteConcernError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	if os.Getenv("TOPOLOGY") != "replica_set" {
		t.Skip()
	}

	want := WriteConcernError{Code: 100, Message: "Not enough data-bearing nodes"}
	filter := bson.NewDocument(bson.EC.String("_id", "foo"))
	update := bson.NewDocument(
		bson.EC.SubDocumentFromElements(
			"$set",
			bson.EC.Double("pi", 3.14159),
		),
	)
	coll := createTestCollection(t, nil, nil, collectionopt.WriteConcern(writeconcern.New(writeconcern.W(25))))

	_, err := coll.UpdateMany(context.Background(), filter, update)
	got, ok := err.(WriteConcernError)
	if !ok {
		t.Errorf("Did not receive correct type of error. got %T; want %T", err, WriteConcernError{})
	}
	if got.Code != want.Code {
		t.Errorf("Did not receive the correct error code. got %d; want %d", got.Code, want.Code)
	}
	if got.Message != want.Message {
		t.Errorf("Did not receive the correct error message. got %s; want %s", got.Message, want.Message)
	}
}

func TestCollection_ReplaceOne_found(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(bson.EC.Int32("x", 1))
	replacement := bson.NewDocument(bson.EC.Int32("y", 1))

	result, err := coll.ReplaceOne(context.Background(), filter, replacement)
	require.Nil(t, err)
	require.NotNil(t, result)
	require.Equal(t, result.MatchedCount, int64(1))
	require.Equal(t, result.ModifiedCount, int64(1))
	require.Nil(t, result.UpsertedID)
}

func TestCollection_ReplaceOne_notFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(bson.EC.Int32("x", 0))
	replacement := bson.NewDocument(bson.EC.Int32("y", 1))

	result, err := coll.ReplaceOne(context.Background(), filter, replacement)
	require.Nil(t, err)
	require.Equal(t, result.MatchedCount, int64(0))
	require.Equal(t, result.ModifiedCount, int64(0))
	require.Nil(t, result.UpsertedID)
}

func TestCollection_ReplaceOne_upsert(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(bson.EC.Int32("x", 0))
	replacement := bson.NewDocument(bson.EC.Int32("y", 1))

	result, err := coll.ReplaceOne(context.Background(), filter, replacement, replaceopt.Upsert(true))
	require.Nil(t, err)
	require.Equal(t, result.MatchedCount, int64(0))
	require.Equal(t, result.ModifiedCount, int64(0))
	require.NotNil(t, result.UpsertedID)
}

func TestCollection_ReplaceOne_WriteError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	filter := bson.NewDocument(bson.EC.String("_id", "foo"))
	replacement := bson.NewDocument(bson.EC.Double("_id", 3.14159))
	coll := createTestCollection(t, nil, nil)

	_, err := coll.InsertOne(context.Background(), bson.NewDocument(bson.EC.String("_id", "foo")))
	require.NoError(t, err)

	_, err = coll.ReplaceOne(context.Background(), filter, replacement)
	got, ok := err.(WriteErrors)
	if !ok {
		t.Errorf("Did not receive correct type of error. got %T; want %T", err, WriteErrors{})
	}
	if len(got) != 1 {
		t.Errorf("Incorrect number of errors receieved. got %d; want %d", len(got), 1)
		t.FailNow()
	}
	switch got[0].Code {
	case 66: // mongod v3.6
	case 16837: //mongod v3.4, mongod v3.2
	default:
		t.Errorf("Did not receive the correct error code. got %d; want (one of) %d", got[0].Code, []int{66, 16837})
		fmt.Printf("%#v\n", got)
	}
}

func TestCollection_ReplaceOne_WriteConcernError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	if os.Getenv("TOPOLOGY") != "replica_set" {
		t.Skip()
	}

	want := WriteConcernError{Code: 100, Message: "Not enough data-bearing nodes"}
	filter := bson.NewDocument(bson.EC.String("_id", "foo"))
	update := bson.NewDocument(bson.EC.Double("pi", 3.14159))
	coll := createTestCollection(t, nil, nil, collectionopt.WriteConcern(writeconcern.New(writeconcern.W(25))))

	_, err := coll.ReplaceOne(context.Background(), filter, update)
	got, ok := err.(WriteConcernError)
	if !ok {
		t.Errorf("Did not receive correct type of error. got %T; want %T", err, WriteConcernError{})
	}
	if got.Code != want.Code {
		t.Errorf("Did not receive the correct error code. got %d; want %d", got.Code, want.Code)
	}
	if got.Message != want.Message {
		t.Errorf("Did not receive the correct error message. got %s; want %s", got.Message, want.Message)
	}
}

func TestCollection_Aggregate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	pipeline := bson.NewArray(
		bson.VC.DocumentFromElements(
			bson.EC.SubDocumentFromElements(
				"$match",
				bson.EC.SubDocumentFromElements(
					"x",
					bson.EC.Int32("$gte", 2),
				),
			),
		),
		bson.VC.DocumentFromElements(
			bson.EC.SubDocumentFromElements(
				"$project",
				bson.EC.Int32("_id", 0),
				bson.EC.Int32("x", 1),
			),
		),
		bson.VC.DocumentFromElements(
			bson.EC.SubDocumentFromElements(
				"$sort",
				bson.EC.Int32("x", 1),
			),
		))

	cursor, err := coll.Aggregate(context.Background(), pipeline, aggregateopt.BundleAggregate())
	require.Nil(t, err)

	for i := 2; i < 5; i++ {
		var doc = bson.NewDocument()
		cursor.Next(context.Background())
		err = cursor.Decode(doc)
		require.NoError(t, err)

		require.Equal(t, doc.Len(), 1)
		num, err := doc.LookupErr("x")
		require.NoError(t, err)
		if num.Type() != bson.TypeInt32 {
			t.Errorf("Incorrect type for x. Got %s, but wanted Int32", num.Type())
			t.FailNow()
		}
		require.Equal(t, int(num.Int32()), i)
	}
}

func testAggregateWithOptions(t *testing.T, createIndex bool, opts aggregateopt.Aggregate) error {
	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	if createIndex {
		indexView := coll.Indexes()
		_, err := indexView.CreateOne(context.Background(), IndexModel{
			Keys: bson.NewDocument(bson.EC.Int32("x", 1)),
		})

		if err != nil {
			return err
		}
	}

	pipeline := bson.NewArray(
		bson.VC.DocumentFromElements(
			bson.EC.SubDocumentFromElements(
				"$match",
				bson.EC.SubDocumentFromElements(
					"x",
					bson.EC.Int32("$gte", 2),
				),
			),
		),
		bson.VC.DocumentFromElements(
			bson.EC.SubDocumentFromElements(
				"$project",
				bson.EC.Int32("_id", 0),
				bson.EC.Int32("x", 1),
			),
		),
		bson.VC.DocumentFromElements(
			bson.EC.SubDocumentFromElements(
				"$sort",
				bson.EC.Int32("x", 1),
			),
		))

	cursor, err := coll.Aggregate(context.Background(), pipeline, opts)
	if err != nil {
		return err
	}

	for i := 2; i < 5; i++ {
		var doc = bson.NewDocument()
		cursor.Next(context.Background())
		err = cursor.Decode(doc)
		if err != nil {
			return err
		}

		if doc.Len() != 1 {
			return fmt.Errorf("got doc len %d, expected 1", doc.Len())
		}

		num, err := doc.LookupErr("x")
		if err != nil {
			return err
		}

		if num.Type() != bson.TypeInt32 {
			return fmt.Errorf("incorrect type for x. got %s, wanted Int32", num.Type())
		}

		if int(num.Int32()) != i {
			return fmt.Errorf("unexpected value returned. got %d, expected %d", int(num.Int32()), i)
		}
	}

	return nil
}

func TestCollection_Aggregate_IndexHint(t *testing.T) {
	skipIfBelow36(t)

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	hint := aggregateopt.Hint(bson.NewDocument(bson.EC.Int32("x", 1)))

	err := testAggregateWithOptions(t, true, hint)
	require.NoError(t, err)
}

func TestCollection_Aggregate_withOptions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	err := testAggregateWithOptions(t, false, aggregateopt.AllowDiskUse(true))
	require.NoError(t, err)
}

func TestCollection_Count(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	count, err := coll.Count(context.Background(), nil)
	require.Nil(t, err)
	require.Equal(t, count, int64(5))
}

func TestCollection_Count_withFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(
		bson.EC.SubDocumentFromElements("x", bson.EC.Int32("$gt", 2)))

	count, err := coll.Count(context.Background(), filter)
	require.Nil(t, err)
	require.Equal(t, count, int64(3))
}

func TestCollection_Count_withOption(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	count, err := coll.Count(context.Background(), nil, countopt.Limit(3))
	require.Nil(t, err)
	require.Equal(t, count, int64(3))
}

func TestCollection_Distinct(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	results, err := coll.Distinct(context.Background(), "x", nil)
	require.Nil(t, err)
	require.Equal(t, results, []interface{}{int32(1), int32(2), int32(3), int32(4), int32(5)})
}

func TestCollection_Distinct_withFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(
		bson.EC.SubDocumentFromElements("x", bson.EC.Int32("$gt", 2)))

	results, err := coll.Distinct(context.Background(), "x", filter)
	require.Nil(t, err)
	require.Equal(t, results, []interface{}{int32(3), int32(4), int32(5)})
}

func TestCollection_Distinct_withOption(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	results, err := coll.Distinct(context.Background(), "x", nil, distinctopt.Collation(&mongoopt.Collation{Locale: "en_US"}))
	require.Nil(t, err)
	require.Equal(t, results, []interface{}{int32(1), int32(2), int32(3), int32(4), int32(5)})
}

func TestCollection_Find_found(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	cursor, err := coll.Find(context.Background(),
		nil,
		findopt.Sort(bson.NewDocument(bson.EC.Int32("x", 1))),
	)
	require.Nil(t, err)

	results := make([]int, 0, 5)
	var doc = make(bson.Reader, 1024)
	for cursor.Next(context.Background()) {
		err = cursor.Decode(doc)
		require.NoError(t, err)

		_, err = doc.Lookup("_id")
		require.NoError(t, err)

		i, err := doc.Lookup("x")
		require.NoError(t, err)
		if i.Value().Type() != bson.TypeInt32 {
			t.Errorf("Incorrect type for x. Got %s, but wanted Int32", i.Value().Type())
			t.FailNow()
		}
		results = append(results, int(i.Value().Int32()))
	}

	require.Len(t, results, 5)
	require.Equal(t, results, []int{1, 2, 3, 4, 5})
}

func TestCollection_Find_notFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	cursor, err := coll.Find(context.Background(), bson.NewDocument(bson.EC.Int32("x", 6)))
	require.Nil(t, err)

	require.False(t, cursor.Next(context.Background()))
}

func TestCollection_FindOne_found(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(bson.EC.Int32("x", 1))
	var result = bson.NewDocument()
	err := coll.FindOne(context.Background(),
		filter,
	).Decode(result)

	require.Nil(t, err)
	require.Equal(t, result.Len(), 2)

	_, err = result.LookupErr("_id")
	require.NoError(t, err)

	num, err := result.LookupErr("x")
	require.NoError(t, err)
	if num.Type() != bson.TypeInt32 {
		t.Errorf("Incorrect type for x. Got %s, but wanted Int32", num.Type())
		t.FailNow()
	}
	require.Equal(t, int(num.Int32()), 1)
}

func TestCollection_FindOne_found_withOption(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(bson.EC.Int32("x", 1))
	var result = bson.NewDocument()
	err := coll.FindOne(context.Background(),
		filter,
		findopt.Comment("here's a query for ya"),
	).Decode(result)
	require.Nil(t, err)
	require.Equal(t, result.Len(), 2)

	_, err = result.LookupErr("_id")
	require.NoError(t, err)

	num, err := result.LookupErr("x")
	require.NoError(t, err)
	if num.Type() != bson.TypeInt32 {
		t.Errorf("Incorrect type for x. Got %s, but wanted Int32", num.Type())
		t.FailNow()
	}
	require.Equal(t, int(num.Int32()), 1)
}

func TestCollection_FindOne_notFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(bson.EC.Int32("x", 6))
	err := coll.FindOne(context.Background(), filter).Decode(nil)
	require.Equal(t, err, ErrNoDocuments)
}

func TestCollection_FindOneAndDelete_found(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(bson.EC.Int32("x", 3))

	var result = bson.NewDocument()
	err := coll.FindOneAndDelete(context.Background(), filter).Decode(result)
	require.NoError(t, err)

	elem, err := result.LookupErr("x")
	require.NoError(t, err)
	require.Equal(t, elem.Type(), bson.TypeInt32, "Incorrect BSON Element type")
	require.Equal(t, int(elem.Int32()), 3)
}

func TestCollection_FindOneAndDelete_found_ignoreResult(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(bson.EC.Int32("x", 3))

	err := coll.FindOneAndDelete(context.Background(), filter).Decode(nil)
	require.NoError(t, err)
}

func TestCollection_FindOneAndDelete_notFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(bson.EC.Int32("x", 6))

	err := coll.FindOneAndDelete(context.Background(), filter).Decode(nil)
	require.Equal(t, err, ErrNoDocuments)
}

func TestCollection_FindOneAndDelete_notFound_ignoreResult(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(bson.EC.Int32("x", 6))

	err := coll.FindOneAndDelete(context.Background(), filter).Decode(nil)
	require.Equal(t, ErrNoDocuments, err)
}

func TestCollection_FindOneAndReplace_found(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(bson.EC.Int32("x", 3))
	replacement := bson.NewDocument(bson.EC.Int32("y", 3))

	var result = bson.NewDocument()
	err := coll.FindOneAndReplace(context.Background(), filter, replacement).Decode(result)
	require.NoError(t, err)

	elem, err := result.LookupErr("x")
	require.NoError(t, err)
	require.Equal(t, elem.Type(), bson.TypeInt32, "Incorrect BSON Element type")
	require.Equal(t, int(elem.Int32()), 3)
}

func TestCollection_FindOneAndReplace_found_ignoreResult(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(bson.EC.Int32("x", 3))
	replacement := bson.NewDocument(bson.EC.Int32("y", 3))

	err := coll.FindOneAndReplace(context.Background(), filter, replacement).Decode(nil)
	require.NoError(t, err)
}

func TestCollection_FindOneAndReplace_notFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(bson.EC.Int32("x", 6))
	replacement := bson.NewDocument(bson.EC.Int32("y", 6))

	err := coll.FindOneAndReplace(context.Background(), filter, replacement).Decode(nil)
	require.Equal(t, err, ErrNoDocuments)
}

func TestCollection_FindOneAndReplace_notFound_ignoreResult(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(bson.EC.Int32("x", 6))
	replacement := bson.NewDocument(bson.EC.Int32("y", 6))

	err := coll.FindOneAndReplace(context.Background(), filter, replacement).Decode(nil)
	require.Equal(t, err, ErrNoDocuments)
}

func TestCollection_FindOneAndUpdate_found(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(bson.EC.Int32("x", 3))
	update := bson.NewDocument(
		bson.EC.SubDocumentFromElements("$set", bson.EC.Int32("x", 6)))

	var result = bson.NewDocument()
	err := coll.FindOneAndUpdate(context.Background(), filter, update).Decode(result)
	require.NoError(t, err)

	elem, err := result.LookupErr("x")
	require.NoError(t, err)
	require.Equal(t, elem.Type(), bson.TypeInt32, "Incorrect BSON Element type")
	require.Equal(t, int(elem.Int32()), 3)
}

func TestCollection_FindOneAndUpdate_found_ignoreResult(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(bson.EC.Int32("x", 3))
	update := bson.NewDocument(
		bson.EC.SubDocumentFromElements("$set", bson.EC.Int32("x", 6)))

	err := coll.FindOneAndUpdate(context.Background(), filter, update).Decode(nil)
	require.NoError(t, err)
}

func TestCollection_FindOneAndUpdate_notFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(bson.EC.Int32("x", 6))
	update := bson.NewDocument(
		bson.EC.SubDocumentFromElements("$set", bson.EC.Int32("x", 6)))

	err := coll.FindOneAndUpdate(context.Background(), filter, update).Decode(nil)
	require.Equal(t, err, ErrNoDocuments)
}

func TestCollection_FindOneAndUpdate_notFound_ignoreResult(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := bson.NewDocument(bson.EC.Int32("x", 6))
	update := bson.NewDocument(
		bson.EC.SubDocumentFromElements("$set", bson.EC.Int32("x", 6)))

	err := coll.FindOneAndUpdate(context.Background(), filter, update).Decode(nil)
	require.Equal(t, err, ErrNoDocuments)
}
