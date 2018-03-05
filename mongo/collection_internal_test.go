// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"fmt"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
	"github.com/mongodb/mongo-go-driver/mongo/internal/testutil"
	"github.com/mongodb/mongo-go-driver/mongo/private/options"
	"github.com/stretchr/testify/require"
)

func createTestCollection(t *testing.T, dbName *string, collName *string) *Collection {
	if collName == nil {
		coll := testutil.ColName(t)
		collName = &coll
	}

	db := createTestDatabase(t, dbName)

	return db.Collection(*collName)
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

func TestCollection_namespace(t *testing.T) {
	t.Parallel()

	dbName := "foo"
	collName := "bar"

	coll := createTestCollection(t, &dbName, &collName)
	namespace := coll.namespace()
	require.Equal(t, namespace.FullName(), fmt.Sprintf("%s.%s", dbName, collName))
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
	result, err := coll.DeleteOne(context.Background(), filter, Opt.Collation(&options.Collation{Locale: "en_US"}))
	require.Nil(t, err)
	require.Equal(t, result.DeletedCount, int64(0))
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

	result, err := coll.DeleteMany(context.Background(), filter, Opt.Collation(&options.Collation{Locale: "en_US"}))
	require.Nil(t, err)
	require.Equal(t, result.DeletedCount, int64(0))
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

	result, err := coll.UpdateOne(context.Background(), filter, update, Opt.Upsert(true))
	require.Nil(t, err)
	require.Equal(t, result.MatchedCount, int64(0))
	require.Equal(t, result.ModifiedCount, int64(0))
	require.NotNil(t, result.UpsertedID)
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

	result, err := coll.UpdateMany(context.Background(), filter, update, Opt.Upsert(true))
	require.Nil(t, err)
	require.Equal(t, result.MatchedCount, int64(0))
	require.Equal(t, result.ModifiedCount, int64(0))
	require.NotNil(t, result.UpsertedID)
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

	result, err := coll.ReplaceOne(context.Background(), filter, replacement, Opt.Upsert(true))
	require.Nil(t, err)
	require.Equal(t, result.MatchedCount, int64(0))
	require.Equal(t, result.ModifiedCount, int64(0))
	require.NotNil(t, result.UpsertedID)
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

	cursor, err := coll.Aggregate(context.Background(), pipeline)
	require.Nil(t, err)

	for i := 2; i < 5; i++ {
		var doc = bson.NewDocument()
		cursor.Next(context.Background())
		err = cursor.Decode(doc)
		require.NoError(t, err)

		require.Equal(t, doc.Len(), 1)
		num, err := doc.Lookup("x")
		require.NoError(t, err)
		if num.Value().Type() != bson.TypeInt32 {
			t.Errorf("Incorrect type for x. Got %s, but wanted Int32", num.Value().Type())
			t.FailNow()
		}
		require.Equal(t, int(num.Value().Int32()), i)
	}
}

func TestCollection_Aggregate_withOptions(t *testing.T) {
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

	cursor, err := coll.Aggregate(context.Background(), pipeline, Opt.AllowDiskUse(true))
	require.Nil(t, err)

	for i := 2; i < 5; i++ {
		var doc = bson.NewDocument()
		cursor.Next(context.Background())
		err = cursor.Decode(doc)
		require.NoError(t, err)

		require.Equal(t, doc.Len(), 1)
		num, err := doc.Lookup("x")
		require.NoError(t, err)
		if num.Value().Type() != bson.TypeInt32 {
			t.Errorf("Incorrect type for x. Got %s, but wanted Int32", num.Value().Type())
			t.FailNow()
		}
		require.Equal(t, int(num.Value().Int32()), i)
	}
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

	count, err := coll.Count(context.Background(), nil, Opt.Limit(3))
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

	results, err := coll.Distinct(context.Background(), "x", nil, Opt.Collation(&options.Collation{Locale: "en_US"}))
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

	sort, err := Opt.Sort(bson.NewDocument(bson.EC.Int32("x", 1)))
	require.NoError(t, err)
	cursor, err := coll.Find(context.Background(),
		nil,
		sort,
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

	_, err = result.Lookup("_id")
	require.NoError(t, err)

	num, err := result.Lookup("x")
	require.NoError(t, err)
	if num.Value().Type() != bson.TypeInt32 {
		t.Errorf("Incorrect type for x. Got %s, but wanted Int32", num.Value().Type())
		t.FailNow()
	}
	require.Equal(t, int(num.Value().Int32()), 1)
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
		Opt.Comment("here's a query for ya"),
	).Decode(result)
	require.Nil(t, err)
	require.Equal(t, result.Len(), 2)

	_, err = result.Lookup("_id")
	require.NoError(t, err)

	num, err := result.Lookup("x")
	require.NoError(t, err)
	if num.Value().Type() != bson.TypeInt32 {
		t.Errorf("Incorrect type for x. Got %s, but wanted Int32", num.Value().Type())
		t.FailNow()
	}
	require.Equal(t, int(num.Value().Int32()), 1)
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

	elem, err := result.Lookup("x")
	require.NoError(t, err)
	require.Equal(t, elem.Value().Type(), bson.TypeInt32, "Incorrect BSON Element type")
	require.Equal(t, int(elem.Value().Int32()), 3)
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
	require.Equal(t, err, ErrNoDocuments)
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

	elem, err := result.Lookup("x")
	require.NoError(t, err)
	require.Equal(t, elem.Value().Type(), bson.TypeInt32, "Incorrect BSON Element type")
	require.Equal(t, int(elem.Value().Int32()), 3)
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

	elem, err := result.Lookup("x")
	require.NoError(t, err)
	require.Equal(t, elem.Value().Type(), bson.TypeInt32, "Incorrect BSON Element type")
	require.Equal(t, int(elem.Value().Int32()), 3)
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
