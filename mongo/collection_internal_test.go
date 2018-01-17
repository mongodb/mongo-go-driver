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

	oldbson "github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/mongo/internal/testutil"
	"github.com/10gen/mongo-go-driver/mongo/options"
	"github.com/skriptble/wilson/bson"
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
	doc1 := oldbson.D{oldbson.NewDocElem("x", 1)}
	doc2 := oldbson.D{oldbson.NewDocElem("x", 2)}
	doc3 := oldbson.D{oldbson.NewDocElem("x", 3)}
	doc4 := oldbson.D{oldbson.NewDocElem("x", 4)}
	doc5 := oldbson.D{oldbson.NewDocElem("x", 5)}

	var err error

	_, err = coll.InsertOne(doc1)
	require.Nil(t, err)

	_, err = coll.InsertOne(doc2)
	require.Nil(t, err)

	_, err = coll.InsertOne(doc3)
	require.Nil(t, err)

	_, err = coll.InsertOne(doc4)
	require.Nil(t, err)

	_, err = coll.InsertOne(doc5)
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

	id := bson.NewObjectID()
	want := bson.C.ObjectID("_id", id)
	doc := bson.NewDocument(2).Append(want, bson.C.Int32("x", 1))
	coll := createTestCollection(t, nil, nil)

	result, err := coll.InsertOne(doc)
	require.Nil(t, err)
	require.Equal(t, result.InsertedID, want)
}

func TestCollection_InsertMany(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	want1 := bson.C.Int32("_id", 11)
	want2 := bson.C.Int32("_id", 12)
	docs := []interface{}{
		bson.NewDocument(1).Append(want1),
		bson.NewDocument(1).Append(bson.C.Int32("x", 6)),
		bson.NewDocument(1).Append(want2),
	}
	coll := createTestCollection(t, nil, nil)

	result, err := coll.InsertMany(docs)
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

	filter := oldbson.D{{Name: "x", Value: 1}}
	result, err := coll.DeleteOne(filter)
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

	filter := oldbson.D{{Name: "x", Value: 0}}
	result, err := coll.DeleteOne(filter)
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

	filter := oldbson.D{{Name: "x", Value: 0}}
	result, err := coll.DeleteOne(filter, Collation(&options.CollationOptions{Locale: "en_US"}))
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

	filter := oldbson.D{
		oldbson.NewDocElem("x", oldbson.D{
			oldbson.NewDocElem("$gte", 3),
		}),
	}
	result, err := coll.DeleteMany(filter)
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

	filter := oldbson.D{
		oldbson.NewDocElem("x", oldbson.D{
			oldbson.NewDocElem("$lt", 1),
		}),
	}
	result, err := coll.DeleteMany(filter)
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

	filter := oldbson.D{
		oldbson.NewDocElem("x", oldbson.D{
			oldbson.NewDocElem("$lt", 1),
		}),
	}
	result, err := coll.DeleteMany(filter, Collation(&options.CollationOptions{Locale: "en_US"}))
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

	filter := oldbson.D{{Name: "x", Value: 1}}
	update := oldbson.M{
		"$inc": oldbson.M{
			"x": 1,
		},
	}

	result, err := coll.UpdateOne(filter, update)
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

	filter := oldbson.D{{Name: "x", Value: 0}}
	update := oldbson.M{
		"$inc": oldbson.M{
			"x": 1,
		},
	}

	result, err := coll.UpdateOne(filter, update)
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

	filter := oldbson.D{{Name: "x", Value: 0}}
	update := oldbson.M{
		"$inc": oldbson.M{
			"x": 1,
		},
	}

	result, err := coll.UpdateOne(filter, update, Upsert(true))
	require.Nil(t, err)
	require.Equal(t, result.MatchedCount, int64(1))
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

	filter := oldbson.D{
		oldbson.NewDocElem("x", oldbson.D{
			oldbson.NewDocElem("$gte", 3),
		}),
	}
	update := oldbson.M{
		"$inc": oldbson.M{
			"x": 1,
		},
	}

	result, err := coll.UpdateMany(filter, update)
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

	filter := oldbson.D{
		oldbson.NewDocElem("x", oldbson.D{
			oldbson.NewDocElem("$lt", 1),
		}),
	}
	update := oldbson.M{
		"$inc": oldbson.M{
			"x": 1,
		},
	}

	result, err := coll.UpdateMany(filter, update)
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

	filter := oldbson.D{
		oldbson.NewDocElem("x", oldbson.D{
			oldbson.NewDocElem("$lt", 1),
		}),
	}
	update := oldbson.M{
		"$inc": oldbson.M{
			"x": 1,
		},
	}

	result, err := coll.UpdateMany(filter, update, Upsert(true))
	require.Nil(t, err)
	require.Equal(t, result.MatchedCount, int64(1))
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

	filter := oldbson.D{{Name: "x", Value: 1}}
	replacement := oldbson.D{{Name: "y", Value: 1}}

	result, err := coll.ReplaceOne(filter, replacement)
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

	filter := oldbson.D{{Name: "x", Value: 0}}
	replacement := oldbson.D{{Name: "y", Value: 1}}

	result, err := coll.ReplaceOne(filter, replacement)
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

	filter := oldbson.D{{Name: "x", Value: 0}}
	replacement := oldbson.D{{Name: "y", Value: 1}}

	result, err := coll.ReplaceOne(filter, replacement, Upsert(true))
	require.Nil(t, err)
	require.Equal(t, result.MatchedCount, int64(1))
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

	pipeline := []oldbson.D{
		{
			oldbson.NewDocElem("$match", oldbson.D{
				oldbson.NewDocElem("x", oldbson.D{
					oldbson.NewDocElem("$gte", 2),
				}),
			}),
		},
		{
			oldbson.NewDocElem("$project", oldbson.D{
				oldbson.NewDocElem("_id", 0),
				oldbson.NewDocElem("x", 1),
			}),
		},
		{
			oldbson.NewDocElem("$sort", oldbson.D{
				oldbson.NewDocElem("x", 1),
			}),
		},
	}
	cursor, err := coll.Aggregate(pipeline)
	require.Nil(t, err)

	for i := 2; i < 5; i++ {
		var doc oldbson.M
		cursor.Next(context.Background(), &doc)
		require.Len(t, doc, 1)
		require.Equal(t, doc["x"], i)
	}
}

func TestCollection_Aggregate_withOptions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	pipeline := []oldbson.D{
		{
			oldbson.NewDocElem("$match", oldbson.D{
				oldbson.NewDocElem("x", oldbson.D{
					oldbson.NewDocElem("$gte", 2),
				}),
			}),
		},
		{
			oldbson.NewDocElem("$project", oldbson.D{
				oldbson.NewDocElem("_id", 0),
				oldbson.NewDocElem("x", 1),
			}),
		},
		{
			oldbson.NewDocElem("$sort", oldbson.D{
				oldbson.NewDocElem("x", 1),
			}),
		},
	}
	cursor, err := coll.Aggregate(pipeline, AllowDiskUse(true))
	require.Nil(t, err)

	for i := 2; i < 5; i++ {
		var doc oldbson.M
		cursor.Next(context.Background(), &doc)
		require.Len(t, doc, 1)
		require.Equal(t, doc["x"], i)
	}
}

func TestCollection_Count(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	count, err := coll.Count(nil)
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

	filter := oldbson.D{
		oldbson.NewDocElem("x", oldbson.D{
			oldbson.NewDocElem("$gt", 2),
		}),
	}
	count, err := coll.Count(filter)
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

	count, err := coll.Count(nil, Limit(3))
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

	results, err := coll.Distinct("x", nil)
	require.Nil(t, err)
	require.Equal(t, results, []interface{}{1, 2, 3, 4, 5})
}

func TestCollection_Distinct_withFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := oldbson.D{
		oldbson.NewDocElem("x", oldbson.D{
			oldbson.NewDocElem("$gt", 2),
		}),
	}
	results, err := coll.Distinct("x", filter)
	require.Nil(t, err)
	require.Equal(t, results, []interface{}{3, 4, 5})
}

func TestCollection_Distinct_withOption(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	results, err := coll.Distinct("x", nil, Collation(&options.CollationOptions{Locale: "en_US"}))
	require.Nil(t, err)
	require.Equal(t, results, []interface{}{1, 2, 3, 4, 5})
}

func TestCollection_Find_found(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	cursor, err := coll.Find(
		nil,
		Sort(oldbson.D{oldbson.NewDocElem("x", 1)}),
	)
	require.Nil(t, err)

	results := make([]int, 0, 5)
	var doc oldbson.M
	for cursor.Next(context.Background(), &doc) {
		require.Nil(t, err)
		require.Len(t, doc, 2)
		_, ok := doc["_id"]
		require.True(t, ok)

		results = append(results, doc["x"].(int))
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

	cursor, err := coll.Find(oldbson.D{oldbson.NewDocElem("x", 6)})
	require.Nil(t, err)

	var doc oldbson.M
	require.False(t, cursor.Next(context.Background(), &doc))
}

func TestCollection_FindOne_found(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := oldbson.D{oldbson.NewDocElem("x", 1)}
	var result oldbson.M
	found, err := coll.FindOne(
		filter,
		&result,
	)
	require.Nil(t, err)
	require.True(t, found)
	require.Len(t, result, 2)

	_, ok := result["_id"]
	require.True(t, ok)
	require.Equal(t, result["x"], 1)
}

func TestCollection_FindOne_found_withOption(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := oldbson.D{oldbson.NewDocElem("x", 1)}
	var result oldbson.M
	found, err := coll.FindOne(
		filter,
		&result,
		Comment("here's a query for ya"),
	)
	require.Nil(t, err)
	require.True(t, found)
	require.Len(t, result, 2)

	_, ok := result["_id"]
	require.True(t, ok)
	require.Equal(t, result["x"], 1)
}

func TestCollection_FindOne_notFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := oldbson.D{oldbson.NewDocElem("x", 6)}
	var result oldbson.M
	found, err := coll.FindOne(filter, &result)
	require.Nil(t, err)
	require.False(t, found)
}

func TestCollection_FindOneAndDelete_found(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := oldbson.D{oldbson.NewDocElem("x", 3)}

	var result oldbson.M
	found, err := coll.FindOneAndDelete(filter, &result)
	require.NoError(t, err)
	require.True(t, found)

	x, ok := result["x"]
	require.True(t, ok)
	require.Equal(t, x, 3)
}

func TestCollection_FindOneAndDelete_found_ignoreResult(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := oldbson.D{oldbson.NewDocElem("x", 3)}

	found, err := coll.FindOneAndDelete(filter, nil)
	require.NoError(t, err)
	require.True(t, found)
}

func TestCollection_FindOneAndDelete_notFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := oldbson.D{oldbson.NewDocElem("x", 6)}

	var result oldbson.M
	found, err := coll.FindOneAndDelete(filter, &result)
	require.NoError(t, err)
	require.False(t, found)
}

func TestCollection_FindOneAndDelete_notFound_ignoreResult(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := oldbson.D{oldbson.NewDocElem("x", 6)}

	found, err := coll.FindOneAndDelete(filter, nil)
	require.NoError(t, err)
	require.False(t, found)
}

func TestCollection_FindOneAndReplace_found(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := oldbson.D{oldbson.NewDocElem("x", 3)}
	replacement := oldbson.D{oldbson.NewDocElem("y", 3)}

	var result oldbson.M
	found, err := coll.FindOneAndReplace(filter, replacement, &result)
	require.NoError(t, err)
	require.True(t, found)

	x, ok := result["x"]
	require.True(t, ok)
	require.Equal(t, x, 3)
}

func TestCollection_FindOneAndReplace_found_ignoreResult(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := oldbson.D{oldbson.NewDocElem("x", 3)}
	replacement := oldbson.D{oldbson.NewDocElem("y", 3)}

	found, err := coll.FindOneAndReplace(filter, replacement, nil)
	require.NoError(t, err)
	require.True(t, found)
}

func TestCollection_FindOneAndReplace_notFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := oldbson.D{oldbson.NewDocElem("x", 6)}
	replacement := oldbson.D{oldbson.NewDocElem("y", 6)}

	var result oldbson.M
	found, err := coll.FindOneAndReplace(filter, replacement, &result)
	require.NoError(t, err)
	require.False(t, found)
}

func TestCollection_FindOneAndReplace_notFound_ignoreResult(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := oldbson.D{oldbson.NewDocElem("x", 6)}
	replacement := oldbson.D{oldbson.NewDocElem("y", 6)}

	found, err := coll.FindOneAndReplace(filter, replacement, nil)
	require.NoError(t, err)
	require.False(t, found)
}

func TestCollection_FindOneAndUpdate_found(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := oldbson.D{oldbson.NewDocElem("x", 3)}
	update := oldbson.D{
		oldbson.NewDocElem("$set", oldbson.D{
			oldbson.NewDocElem("x", 6),
		}),
	}
	var result oldbson.M
	found, err := coll.FindOneAndUpdate(filter, update, &result)
	require.NoError(t, err)
	require.True(t, found)

	x, ok := result["x"]
	require.True(t, ok)
	require.Equal(t, x, 3)
}

func TestCollection_FindOneAndUpdate_found_ignoreResult(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := oldbson.D{oldbson.NewDocElem("x", 3)}
	update := oldbson.D{
		oldbson.NewDocElem("$set", oldbson.D{
			oldbson.NewDocElem("x", 6),
		}),
	}

	found, err := coll.FindOneAndUpdate(filter, update, nil)
	require.NoError(t, err)
	require.True(t, found)
}

func TestCollection_FindOneAndUpdate_notFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := oldbson.D{oldbson.NewDocElem("x", 6)}
	update := oldbson.D{
		oldbson.NewDocElem("$set", oldbson.D{
			oldbson.NewDocElem("x", 6),
		}),
	}

	var result oldbson.M
	found, err := coll.FindOneAndUpdate(filter, update, &result)
	require.NoError(t, err)
	require.False(t, found)
}

func TestCollection_FindOneAndUpdate_notFound_ignoreResult(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	coll := createTestCollection(t, nil, nil)
	initCollection(t, coll)

	filter := oldbson.D{oldbson.NewDocElem("x", 6)}
	update := oldbson.D{
		oldbson.NewDocElem("$set", oldbson.D{
			oldbson.NewDocElem("x", 6),
		}),
	}

	found, err := coll.FindOneAndUpdate(filter, update, nil)
	require.NoError(t, err)
	require.False(t, found)
}
