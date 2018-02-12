// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// NOTE: Any time this file is modified, a WEBSITE ticket should be opened to sync the changes with
// the "What is MongoDB" webpage, which the example was originally added to as part of WEBSITE-5148.

package documentation_examples

import (
	"context"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/stretchr/testify/require"
)

func requireCursorLength(t *testing.T, cursor mongo.Cursor, length int) {
	i := 0
	for cursor.Next(context.Background()) {
		i++
	}

	require.NoError(t, cursor.Err())
	require.Equal(t, i, length)
}

func stringSliceEquals(s1 []string, s2 []string) bool {
	if len(s1) != len(s2) {
		return false
	}

	for i := range s1 {
		if s1[i] != s2[i] {
			return false
		}
	}

	return true
}

func containsKey(keys bson.Keys, key string, prefix []string) bool {
	for _, k := range keys {
		if k.Name == key && stringSliceEquals(k.Prefix, prefix) {
			return true
		}
	}

	return false
}

func InsertExamples(t *testing.T, db *mongo.Database) {
	_, err := db.RunCommand(
		context.Background(),
		bson.NewDocument(bson.C.Int32("dropDatabase", 1)),
	)
	require.NoError(t, err)

	coll := db.Collection("inventory")

	{
		// Start Example 1

		result, err := coll.InsertOne(
			context.Background(),
			bson.NewDocument(
				bson.C.String("item", "canvas"),
				bson.C.Int32("qty", 100),
				bson.C.ArrayFromElements("tags",
					bson.AC.String("cotton"),
				),
				bson.C.SubDocumentFromElements("size",
					bson.C.Int32("h", 28),
					bson.C.Double("w", 35.5),
					bson.C.String("uom", "cm"),
				),
			))

		// End Example 1

		require.NoError(t, err)
		require.NotNil(t, result.InsertedID)
	}

	{
		// Start Example 2

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(bson.C.String("item", "canvas")),
		)

		// End Example 2

		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)

	}

	{
		// Start Example 3

		result, err := coll.InsertMany(
			context.Background(),
			[]interface{}{
				bson.NewDocument(
					bson.C.String("item", "journal"),
					bson.C.Int32("qty", 25),
					bson.C.ArrayFromElements("tags",
						bson.AC.String("blank"),
						bson.AC.String("red"),
					),
					bson.C.SubDocumentFromElements("size",
						bson.C.Int32("h", 14),
						bson.C.Int32("w", 21),
						bson.C.String("uom", "cm"),
					),
				),
				bson.NewDocument(
					bson.C.String("item", "mat"),
					bson.C.Int32("qty", 25),
					bson.C.ArrayFromElements("tags",
						bson.AC.String("gray"),
					),
					bson.C.SubDocumentFromElements("size",
						bson.C.Double("h", 27.9),
						bson.C.Double("w", 35.5),
						bson.C.String("uom", "cm"),
					),
				),
				bson.NewDocument(
					bson.C.String("item", "mousepad"),
					bson.C.Int32("qty", 25),
					bson.C.ArrayFromElements("tags",
						bson.AC.String("gel"),
						bson.AC.String("blue"),
					),
					bson.C.SubDocumentFromElements("size",
						bson.C.Int32("h", 19),
						bson.C.Double("w", 22.85),
						bson.C.String("uom", "cm"),
					),
				),
			})

		// End Example 3

		require.NoError(t, err)
		require.Len(t, result.InsertedIDs, 3)
	}
}

func QueryToplevelFieldsExamples(t *testing.T, db *mongo.Database) {
	_, err := db.RunCommand(
		context.Background(),
		bson.NewDocument(bson.C.Int32("dropDatabase", 1)),
	)
	require.NoError(t, err)

	coll := db.Collection("inventory")

	{
		// Start Example 6

		docs := []interface{}{
			bson.NewDocument(
				bson.C.String("item", "journal"),
				bson.C.Int32("qty", 25),
				bson.C.SubDocumentFromElements("size",
					bson.C.Int32("h", 14),
					bson.C.Int32("w", 21),
					bson.C.String("uom", "cm"),
				),
				bson.C.String("status", "A"),
			),
			bson.NewDocument(
				bson.C.String("item", "notebook"),
				bson.C.Int32("qty", 50),
				bson.C.SubDocumentFromElements("size",
					bson.C.Double("h", 8.5),
					bson.C.Int32("w", 11),
					bson.C.String("uom", "in"),
				),
				bson.C.String("status", "A"),
			),
			bson.NewDocument(
				bson.C.String("item", "paper"),
				bson.C.Int32("qty", 100),
				bson.C.SubDocumentFromElements("size",
					bson.C.Double("h", 8.5),
					bson.C.Int32("w", 11),
					bson.C.String("uom", "in"),
				),
				bson.C.String("status", "D"),
			),
			bson.NewDocument(
				bson.C.String("item", "planner"),
				bson.C.Int32("qty", 75),
				bson.C.SubDocumentFromElements("size",
					bson.C.Double("h", 22.85),
					bson.C.Int32("w", 30),
					bson.C.String("uom", "cm"),
				),
				bson.C.String("status", "D"),
			),
			bson.NewDocument(
				bson.C.String("item", "postcard"),
				bson.C.Int32("qty", 45),
				bson.C.SubDocumentFromElements("size",
					bson.C.Int32("h", 10),
					bson.C.Double("w", 15.25),
					bson.C.String("uom", "cm"),
				),
				bson.C.String("status", "A"),
			),
		}

		result, err := coll.InsertMany(context.Background(), docs)

		// End Example 6

		require.NoError(t, err)
		require.Len(t, result.InsertedIDs, 5)
	}

	{
		// Start Example 7

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(),
		)

		// End Example 7

		require.NoError(t, err)
		requireCursorLength(t, cursor, 5)
	}

	{
		// Start Example 9

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(bson.C.String("status", "D")),
		)

		// End Example 9

		require.NoError(t, err)
		requireCursorLength(t, cursor, 2)
	}

	{
		// Start Example 10

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.SubDocumentFromElements("status",
					bson.C.ArrayFromElements("$in",
						bson.AC.String("A"),
						bson.AC.String("D"),
					),
				),
			))

		// End Example 10

		require.NoError(t, err)
		requireCursorLength(t, cursor, 5)
	}

	{
		// Start Example 11

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.String("status", "A"),
				bson.C.SubDocumentFromElements("qty",
					bson.C.Int32("$lt", 30),
				),
			))

		// End Example 11

		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)
	}

	{
		// Start Example 12

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.ArrayFromElements("$or",
					bson.AC.DocumentFromElements(
						bson.C.String("status", "A"),
					),
					bson.AC.DocumentFromElements(
						bson.C.SubDocumentFromElements("qty",
							bson.C.Int32("$lt", 30),
						),
					),
				),
			))

		// End Example 12

		require.NoError(t, err)
		requireCursorLength(t, cursor, 3)
	}

	{
		// Start Example 13

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.String("status", "A"),
				bson.C.ArrayFromElements("$or",
					bson.AC.DocumentFromElements(
						bson.C.SubDocumentFromElements("qty",
							bson.C.Int32("$lt", 30),
						),
					),
					bson.AC.DocumentFromElements(
						bson.C.Regex("item", "^p", ""),
					),
				),
			))

		// End Example 13

		require.NoError(t, err)
		requireCursorLength(t, cursor, 2)
	}

}

func QueryEmbeddedDocumentsExamples(t *testing.T, db *mongo.Database) {
	_, err := db.RunCommand(
		context.Background(),
		bson.NewDocument(bson.C.Int32("dropDatabase", 1)),
	)
	require.NoError(t, err)

	coll := db.Collection("inventory")

	{
		// Start Example 14

		docs := []interface{}{
			bson.NewDocument(
				bson.C.String("item", "journal"),
				bson.C.Int32("qty", 25),
				bson.C.SubDocumentFromElements("size",
					bson.C.Int32("h", 14),
					bson.C.Int32("w", 21),
					bson.C.String("uom", "cm"),
				),
				bson.C.String("status", "A"),
			),
			bson.NewDocument(
				bson.C.String("item", "notebook"),
				bson.C.Int32("qty", 50),
				bson.C.SubDocumentFromElements("size",
					bson.C.Double("h", 8.5),
					bson.C.Int32("w", 11),
					bson.C.String("uom", "in"),
				),
				bson.C.String("status", "A"),
			),
			bson.NewDocument(
				bson.C.String("item", "paper"),
				bson.C.Int32("qty", 100),
				bson.C.SubDocumentFromElements("size",
					bson.C.Double("h", 8.5),
					bson.C.Int32("w", 11),
					bson.C.String("uom", "in"),
				),
				bson.C.String("status", "D"),
			),
			bson.NewDocument(
				bson.C.String("item", "planner"),
				bson.C.Int32("qty", 75),
				bson.C.SubDocumentFromElements("size",
					bson.C.Double("h", 22.85),
					bson.C.Int32("w", 30),
					bson.C.String("uom", "cm"),
				),
				bson.C.String("status", "D"),
			),
			bson.NewDocument(
				bson.C.String("item", "postcard"),
				bson.C.Int32("qty", 45),
				bson.C.SubDocumentFromElements("size",
					bson.C.Int32("h", 10),
					bson.C.Double("w", 15.25),
					bson.C.String("uom", "cm"),
				),
				bson.C.String("status", "A"),
			),
		}

		result, err := coll.InsertMany(context.Background(), docs)

		// End Example 14

		require.NoError(t, err)
		require.Len(t, result.InsertedIDs, 5)
	}

	{
		// Start Example 15

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.SubDocumentFromElements("size",
					bson.C.Int32("h", 14),
					bson.C.Int32("w", 21),
					bson.C.String("uom", "cm"),
				),
			))

		// End Example 15

		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)
	}

	{
		// Start Example 16

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.SubDocumentFromElements("size",
					bson.C.Int32("w", 21),
					bson.C.Int32("h", 14),
					bson.C.String("uom", "cm"),
				),
			))

		// End Example 16

		require.NoError(t, err)
		requireCursorLength(t, cursor, 0)
	}

	{
		// Start Example 17

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.String("size.uom", "in"),
			))

		// End Example 17

		require.NoError(t, err)
		requireCursorLength(t, cursor, 2)
	}

	{
		// Start Example 18

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.SubDocumentFromElements("size.h",
					bson.C.Int32("$lt", 15),
				),
			))

		// End Example 18

		require.NoError(t, err)
		requireCursorLength(t, cursor, 4)
	}

	{
		// Start Example 19

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.SubDocumentFromElements("size.h",
					bson.C.Int32("$lt", 15),
				),
				bson.C.String("size.uom", "in"),
				bson.C.String("status", "D"),
			))

		// End Example 19

		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)
	}

}

func QueryArraysExamples(t *testing.T, db *mongo.Database) {
	_, err := db.RunCommand(
		context.Background(),
		bson.NewDocument(bson.C.Int32("dropDatabase", 1)),
	)
	require.NoError(t, err)

	coll := db.Collection("inventory")

	{
		// Start Example 20

		docs := []interface{}{
			bson.NewDocument(
				bson.C.String("item", "journal"),
				bson.C.Int32("qty", 25),
				bson.C.ArrayFromElements("tags",
					bson.AC.String("blank"),
					bson.AC.String("red"),
				),
				bson.C.ArrayFromElements("dim_cm",
					bson.AC.Int32(14),
					bson.AC.Int32(21),
				),
			),
			bson.NewDocument(
				bson.C.String("item", "notebook"),
				bson.C.Int32("qty", 50),
				bson.C.ArrayFromElements("tags",
					bson.AC.String("red"),
					bson.AC.String("blank"),
				),
				bson.C.ArrayFromElements("dim_cm",
					bson.AC.Int32(14),
					bson.AC.Int32(21),
				),
			),
			bson.NewDocument(
				bson.C.String("item", "paper"),
				bson.C.Int32("qty", 100),
				bson.C.ArrayFromElements("tags",
					bson.AC.String("red"),
					bson.AC.String("blank"),
					bson.AC.String("plain"),
				),
				bson.C.ArrayFromElements("dim_cm",
					bson.AC.Int32(14),
					bson.AC.Int32(21),
				),
			),
			bson.NewDocument(
				bson.C.String("item", "planner"),
				bson.C.Int32("qty", 75),
				bson.C.ArrayFromElements("tags",
					bson.AC.String("blank"),
					bson.AC.String("red"),
				),
				bson.C.ArrayFromElements("dim_cm",
					bson.AC.Double(22.85),
					bson.AC.Int32(30),
				),
			),
			bson.NewDocument(
				bson.C.String("item", "postcard"),
				bson.C.Int32("qty", 45),
				bson.C.ArrayFromElements("tags",
					bson.AC.String("blue"),
				),
				bson.C.ArrayFromElements("dim_cm",
					bson.AC.Int32(10),
					bson.AC.Double(15.25),
				),
			),
		}

		result, err := coll.InsertMany(context.Background(), docs)

		// End Example 20

		require.NoError(t, err)
		require.Len(t, result.InsertedIDs, 5)
	}

	{
		// Start Example 21

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.ArrayFromElements("tags",
					bson.AC.String("red"),
					bson.AC.String("blank"),
				),
			))

		// End Example 21

		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)
	}

	{
		// Start Example 22

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.SubDocumentFromElements("tags",
					bson.C.ArrayFromElements("$all",
						bson.AC.String("red"),
						bson.AC.String("blank"),
					),
				),
			))

		// End Example 22

		require.NoError(t, err)
		requireCursorLength(t, cursor, 4)
	}

	{
		// Start Example 23

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.String("tags", "red"),
			))

		// End Example 23

		require.NoError(t, err)
		requireCursorLength(t, cursor, 4)
	}

	{
		// Start Example 24

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.SubDocumentFromElements("dim_cm",
					bson.C.Int32("$gt", 25),
				),
			))

		// End Example 24

		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)
	}

	{
		// Start Example 25

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.SubDocumentFromElements("dim_cm",
					bson.C.Int32("$gt", 15),
					bson.C.Int32("$lt", 20),
				),
			))

		// End Example 25

		require.NoError(t, err)
		requireCursorLength(t, cursor, 4)
	}

	{
		// Start Example 26

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.SubDocumentFromElements("dim_cm",
					bson.C.SubDocumentFromElements("$elemMatch",
						bson.C.Int32("$gt", 22),
						bson.C.Int32("$lt", 30),
					),
				),
			))

		// End Example 26

		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)
	}

	{
		// Start Example 27

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.SubDocumentFromElements("dim_cm.1",
					bson.C.Int32("$gt", 25),
				),
			))

		// End Example 27

		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)
	}

	{
		// Start Example 28

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.SubDocumentFromElements("tags",
					bson.C.Int32("$size", 3),
				),
			))

		// End Example 28

		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)
	}

}

func QueryArrayEmbeddedDocumentsExamples(t *testing.T, db *mongo.Database) {
	_, err := db.RunCommand(
		context.Background(),
		bson.NewDocument(bson.C.Int32("dropDatabase", 1)),
	)
	require.NoError(t, err)

	coll := db.Collection("inventory")

	{
		// Start Example 29

		docs := []interface{}{
			bson.NewDocument(
				bson.C.String("item", "journal"),
				bson.C.ArrayFromElements("instock",
					bson.AC.DocumentFromElements(
						bson.C.String("warehouse", "A"),
						bson.C.Int32("qty", 5),
					),
					bson.AC.DocumentFromElements(
						bson.C.String("warehouse", "C"),
						bson.C.Int32("qty", 15),
					),
				),
			),
			bson.NewDocument(
				bson.C.String("item", "notebook"),
				bson.C.ArrayFromElements("instock",
					bson.AC.DocumentFromElements(
						bson.C.String("warehouse", "C"),
						bson.C.Int32("qty", 5),
					),
				),
			),
			bson.NewDocument(
				bson.C.String("item", "paper"),
				bson.C.ArrayFromElements("instock",
					bson.AC.DocumentFromElements(
						bson.C.String("warehouse", "A"),
						bson.C.Int32("qty", 60),
					),
					bson.AC.DocumentFromElements(
						bson.C.String("warehouse", "B"),
						bson.C.Int32("qty", 15),
					),
				),
			),
			bson.NewDocument(
				bson.C.String("item", "planner"),
				bson.C.ArrayFromElements("instock",
					bson.AC.DocumentFromElements(
						bson.C.String("warehouse", "A"),
						bson.C.Int32("qty", 40),
					),
					bson.AC.DocumentFromElements(
						bson.C.String("warehouse", "B"),
						bson.C.Int32("qty", 5),
					),
				),
			),
			bson.NewDocument(
				bson.C.String("item", "postcard"),
				bson.C.ArrayFromElements("instock",
					bson.AC.DocumentFromElements(
						bson.C.String("warehouse", "B"),
						bson.C.Int32("qty", 15),
					),
					bson.AC.DocumentFromElements(
						bson.C.String("warehouse", "C"),
						bson.C.Int32("qty", 35),
					),
				),
			),
		}

		result, err := coll.InsertMany(context.Background(), docs)

		// End Example 29

		require.NoError(t, err)
		require.Len(t, result.InsertedIDs, 5)
	}

	{
		// Start Example 30

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.SubDocumentFromElements("instock",
					bson.C.String("warehouse", "A"),
					bson.C.Int32("qty", 5),
				),
			))

		// End Example 30

		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)
	}

	{
		// Start Example 31

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.SubDocumentFromElements("instock",
					bson.C.Int32("qty", 5),
					bson.C.String("warehouse", "A"),
				),
			))

		// End Example 31

		require.NoError(t, err)
		requireCursorLength(t, cursor, 0)
	}

	{
		// Start Example 32

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.SubDocumentFromElements("instock.0.qty",
					bson.C.Int32("$lte", 20),
				),
			))

		// End Example 32

		require.NoError(t, err)
		requireCursorLength(t, cursor, 3)
	}

	{
		// Start Example 33

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.SubDocumentFromElements("instock.qty",
					bson.C.Int32("$lte", 20),
				),
			))

		// End Example 33

		require.NoError(t, err)
		requireCursorLength(t, cursor, 5)
	}

	{
		// Start Example 34

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.SubDocumentFromElements("instock",
					bson.C.SubDocumentFromElements("$elemMatch",
						bson.C.Int32("qty", 5),
						bson.C.String("warehouse", "A"),
					),
				),
			))

		// End Example 34

		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)
	}

	{
		// Start Example 35

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.SubDocumentFromElements("instock",
					bson.C.SubDocumentFromElements("$elemMatch",
						bson.C.SubDocumentFromElements("qty",
							bson.C.Int32("$gt", 10),
							bson.C.Int32("$lte", 20),
						),
					),
				),
			))

		// End Example 35

		require.NoError(t, err)
		requireCursorLength(t, cursor, 3)
	}

	{
		// Start Example 36

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.SubDocumentFromElements("instock.qty",
					bson.C.Int32("$gt", 10),
					bson.C.Int32("$lte", 20),
				),
			))

		// End Example 36

		require.NoError(t, err)
		requireCursorLength(t, cursor, 4)
	}

	{
		// Start Example 37

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.Int32("instock.qty", 5),
				bson.C.String("instock.warehouse", "A"),
			))

		// End Example 37

		require.NoError(t, err)
		requireCursorLength(t, cursor, 2)
	}
}

func QueryNullMissingFieldsExamples(t *testing.T, db *mongo.Database) {
	_, err := db.RunCommand(
		context.Background(),
		bson.NewDocument(bson.C.Int32("dropDatabase", 1)),
	)
	require.NoError(t, err)

	coll := db.Collection("inventory")

	{
		// Start Example 38

		docs := []interface{}{
			bson.NewDocument(
				bson.C.Int32("_id", 1),
				bson.C.Null("item"),
			),
			bson.NewDocument(
				bson.C.Int32("_id", 2),
			),
		}

		result, err := coll.InsertMany(context.Background(), docs)

		// End Example 38

		require.NoError(t, err)
		require.Len(t, result.InsertedIDs, 2)
	}

	{
		// Start Example 39

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.Null("item"),
			))

		// End Example 39

		require.NoError(t, err)
		requireCursorLength(t, cursor, 2)
	}

	{
		// Start Example 40

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.SubDocumentFromElements("item",
					bson.C.Int32("$type", 10),
				),
			))

		// End Example 40

		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)
	}

	{
		// Start Example 41

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.SubDocumentFromElements("item",
					bson.C.Boolean("$exists", false),
				),
			))

		// End Example 41

		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)
	}
}

func ProjectionExamples(t *testing.T, db *mongo.Database) {
	_, err := db.RunCommand(
		context.Background(),
		bson.NewDocument(bson.C.Int32("dropDatabase", 1)),
	)
	require.NoError(t, err)

	coll := db.Collection("inventory")

	{
		// Start Example 42

		docs := []interface{}{
			bson.NewDocument(
				bson.C.String("item", "journal"),
				bson.C.String("status", "A"),
				bson.C.SubDocumentFromElements("size",
					bson.C.Int32("h", 14),
					bson.C.Int32("w", 21),
					bson.C.String("uom", "cm"),
				),
				bson.C.ArrayFromElements("instock",
					bson.AC.DocumentFromElements(
						bson.C.String("warehouse", "A"),
						bson.C.Int32("qty", 5),
					),
				),
			),
			bson.NewDocument(
				bson.C.String("item", "notebook"),
				bson.C.String("status", "A"),
				bson.C.SubDocumentFromElements("size",
					bson.C.Double("h", 8.5),
					bson.C.Double("w", 11),
					bson.C.String("uom", "in"),
				),
				bson.C.ArrayFromElements("instock",
					bson.AC.DocumentFromElements(
						bson.C.String("warehouse", "C"),
						bson.C.Int32("qty", 5),
					),
				),
			),
			bson.NewDocument(
				bson.C.String("item", "paper"),
				bson.C.String("status", "D"),
				bson.C.SubDocumentFromElements("size",
					bson.C.Double("h", 8.5),
					bson.C.Double("w", 11),
					bson.C.String("uom", "in"),
				),
				bson.C.ArrayFromElements("instock",
					bson.AC.DocumentFromElements(
						bson.C.String("warehouse", "A"),
						bson.C.Int32("qty", 60),
					),
				),
			),
			bson.NewDocument(
				bson.C.String("item", "planner"),
				bson.C.String("status", "D"),
				bson.C.SubDocumentFromElements("size",
					bson.C.Double("h", 22.85),
					bson.C.Int32("w", 30),
					bson.C.String("uom", "cm"),
				),
				bson.C.ArrayFromElements("instock",
					bson.AC.DocumentFromElements(
						bson.C.String("warehouse", "A"),
						bson.C.Int32("qty", 40),
					),
				),
			),
			bson.NewDocument(
				bson.C.String("item", "postcard"),
				bson.C.String("status", "A"),
				bson.C.SubDocumentFromElements("size",
					bson.C.Int32("h", 10),
					bson.C.Double("w", 15.25),
					bson.C.String("uom", "cm"),
				),
				bson.C.ArrayFromElements("instock",
					bson.AC.DocumentFromElements(
						bson.C.String("warehouse", "B"),
						bson.C.Int32("qty", 15),
					),
					bson.AC.DocumentFromElements(
						bson.C.String("warehouse", "C"),
						bson.C.Int32("qty", 35),
					),
				),
			),
		}

		result, err := coll.InsertMany(context.Background(), docs)

		// End Example 42

		require.NoError(t, err)
		require.Len(t, result.InsertedIDs, 5)
	}

	{
		// Start Example 43

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.String("status", "A"),
			))

		// End Example 43

		require.NoError(t, err)
		requireCursorLength(t, cursor, 3)
	}

	{
		// Start Example 44

		projection, err := mongo.Projection(bson.NewDocument(
			bson.C.Int32("item", 1),
			bson.C.Int32("status", 1),
		))
		require.NoError(t, err)

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.String("status", "A"),
			),
			projection,
		)

		// End Example 44

		require.NoError(t, err)

		doc := bson.NewDocument()
		for cursor.Next(context.Background()) {
			err := cursor.Decode(doc)
			require.NoError(t, err)

			keys, err := doc.Keys(false)
			require.NoError(t, err)

			require.True(t, containsKey(keys, "_id", nil))
			require.True(t, containsKey(keys, "item", nil))
			require.True(t, containsKey(keys, "status", nil))
			require.False(t, containsKey(keys, "size", nil))
			require.False(t, containsKey(keys, "instock", nil))
		}

		require.NoError(t, cursor.Err())
	}

	{
		// Start Example 45

		projection, err := mongo.Projection(bson.NewDocument(
			bson.C.Int32("item", 1),
			bson.C.Int32("status", 1),
			bson.C.Int32("_id", 0),
		))
		require.NoError(t, err)

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.String("status", "A"),
			),
			projection,
		)

		// End Example 45

		require.NoError(t, err)

		doc := bson.NewDocument()
		for cursor.Next(context.Background()) {
			err := cursor.Decode(doc)
			require.NoError(t, err)

			keys, err := doc.Keys(false)
			require.NoError(t, err)

			require.False(t, containsKey(keys, "_id", nil))
			require.True(t, containsKey(keys, "item", nil))
			require.True(t, containsKey(keys, "status", nil))
			require.False(t, containsKey(keys, "size", nil))
			require.False(t, containsKey(keys, "instock", nil))
		}

		require.NoError(t, cursor.Err())
	}

	{
		// Start Example 46

		projection, err := mongo.Projection(bson.NewDocument(
			bson.C.Int32("status", 0),
			bson.C.Int32("instock", 0),
		))
		require.NoError(t, err)

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.String("status", "A"),
			),
			projection,
		)

		// End Example 46

		require.NoError(t, err)

		doc := bson.NewDocument()
		for cursor.Next(context.Background()) {
			err := cursor.Decode(doc)
			require.NoError(t, err)

			keys, err := doc.Keys(false)
			require.NoError(t, err)

			require.True(t, containsKey(keys, "_id", nil))
			require.True(t, containsKey(keys, "item", nil))
			require.False(t, containsKey(keys, "status", nil))
			require.True(t, containsKey(keys, "size", nil))
			require.False(t, containsKey(keys, "instock", nil))
		}

		require.NoError(t, cursor.Err())
	}

	{
		// Start Example 47

		projection, err := mongo.Projection(bson.NewDocument(
			bson.C.Int32("item", 1),
			bson.C.Int32("status", 1),
			bson.C.Int32("size.uom", 1),
		))
		require.NoError(t, err)

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.String("status", "A"),
			),
			projection,
		)

		// End Example 47

		require.NoError(t, err)

		doc := bson.NewDocument()
		for cursor.Next(context.Background()) {
			err := cursor.Decode(doc)
			require.NoError(t, err)

			keys, err := doc.Keys(true)
			require.NoError(t, err)

			require.True(t, containsKey(keys, "_id", nil))
			require.True(t, containsKey(keys, "item", nil))
			require.True(t, containsKey(keys, "status", nil))
			require.True(t, containsKey(keys, "size", nil))
			require.False(t, containsKey(keys, "instock", nil))

			require.True(t, containsKey(keys, "uom", []string{"size"}))
			require.False(t, containsKey(keys, "h", []string{"size"}))
			require.False(t, containsKey(keys, "w", []string{"size"}))

		}

		require.NoError(t, cursor.Err())
	}

	{
		// Start Example 48

		projection, err := mongo.Projection(bson.NewDocument(
			bson.C.Int32("size.uom", 0),
		))
		require.NoError(t, err)

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.String("status", "A"),
			),
			projection,
		)

		// End Example 48

		require.NoError(t, err)

		doc := bson.NewDocument()
		for cursor.Next(context.Background()) {
			err := cursor.Decode(doc)
			require.NoError(t, err)

			keys, err := doc.Keys(true)
			require.NoError(t, err)

			require.True(t, containsKey(keys, "_id", nil))
			require.True(t, containsKey(keys, "item", nil))
			require.True(t, containsKey(keys, "status", nil))
			require.True(t, containsKey(keys, "size", nil))
			require.True(t, containsKey(keys, "instock", nil))

			require.False(t, containsKey(keys, "uom", []string{"size"}))
			require.True(t, containsKey(keys, "h", []string{"size"}))
			require.True(t, containsKey(keys, "w", []string{"size"}))

		}

		require.NoError(t, cursor.Err())
	}

	{
		// Start Example 49

		projection, err := mongo.Projection(bson.NewDocument(
			bson.C.Int32("item", 1),
			bson.C.Int32("status", 1),
			bson.C.Int32("instock.qty", 1),
		))
		require.NoError(t, err)

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.String("status", "A"),
			),
			projection,
		)

		// End Example 49

		require.NoError(t, err)

		doc := bson.NewDocument()
		for cursor.Next(context.Background()) {
			err := cursor.Decode(doc)
			require.NoError(t, err)

			keys, err := doc.Keys(true)
			require.NoError(t, err)

			require.True(t, containsKey(keys, "_id", nil))
			require.True(t, containsKey(keys, "item", nil))
			require.True(t, containsKey(keys, "status", nil))
			require.False(t, containsKey(keys, "size", nil))
			require.True(t, containsKey(keys, "instock", nil))

			instock, err := doc.Lookup("instock")
			require.NoError(t, err)

			arr := instock.Value().MutableArray()

			for i := uint(0); i < uint(arr.Len()); i++ {
				elem, err := arr.Lookup(i)
				require.NoError(t, err)

				require.Equal(t, bson.TypeEmbeddedDocument, elem.Type())
				subdoc := elem.MutableDocument()

				require.Equal(t, 1, subdoc.Len())
				_, err = subdoc.Lookup("qty")
				require.NoError(t, err)
			}
		}

		require.NoError(t, cursor.Err())
	}

	{
		// Start Example 50

		projection, err := mongo.Projection(bson.NewDocument(
			bson.C.Int32("item", 1),
			bson.C.Int32("status", 1),
			bson.C.SubDocumentFromElements("instock",
				bson.C.Int32("$slice", -1),
			),
		))
		require.NoError(t, err)

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.String("status", "A"),
			),
			projection,
		)

		// End Example 50

		require.NoError(t, err)

		doc := bson.NewDocument()
		for cursor.Next(context.Background()) {
			err := cursor.Decode(doc)
			require.NoError(t, err)

			keys, err := doc.Keys(true)
			require.NoError(t, err)

			require.True(t, containsKey(keys, "_id", nil))
			require.True(t, containsKey(keys, "item", nil))
			require.True(t, containsKey(keys, "status", nil))
			require.False(t, containsKey(keys, "size", nil))
			require.True(t, containsKey(keys, "instock", nil))

			instock, err := doc.Lookup("instock")
			require.NoError(t, err)
			require.Equal(t, instock.Value().MutableArray().Len(), 1)
		}

		require.NoError(t, cursor.Err())
	}
}

func UpdateExamples(t *testing.T, db *mongo.Database) {
	_, err := db.RunCommand(
		context.Background(),
		bson.NewDocument(bson.C.Int32("dropDatabase", 1)),
	)
	require.NoError(t, err)

	coll := db.Collection("inventory")

	{
		// Start Example 51

		docs := []interface{}{
			bson.NewDocument(
				bson.C.String("item", "canvas"),
				bson.C.Int32("qty", 100),
				bson.C.SubDocumentFromElements("size",
					bson.C.Int32("h", 28),
					bson.C.Double("w", 35.5),
					bson.C.String("uom", "cm"),
				),
				bson.C.String("status", "A"),
			),
			bson.NewDocument(
				bson.C.String("item", "journal"),
				bson.C.Int32("qty", 25),
				bson.C.SubDocumentFromElements("size",
					bson.C.Int32("h", 14),
					bson.C.Int32("w", 21),
					bson.C.String("uom", "cm"),
				),
				bson.C.String("status", "A"),
			),
			bson.NewDocument(
				bson.C.String("item", "mat"),
				bson.C.Int32("qty", 85),
				bson.C.SubDocumentFromElements("size",
					bson.C.Double("h", 27.9),
					bson.C.Double("w", 35.5),
					bson.C.String("uom", "cm"),
				),
				bson.C.String("status", "A"),
			),
			bson.NewDocument(
				bson.C.String("item", "mousepad"),
				bson.C.Int32("qty", 25),
				bson.C.SubDocumentFromElements("size",
					bson.C.Int32("h", 19),
					bson.C.Double("w", 22.85),
					bson.C.String("uom", "in"),
				),
				bson.C.String("status", "P"),
			),
			bson.NewDocument(
				bson.C.String("item", "notebook"),
				bson.C.Int32("qty", 50),
				bson.C.SubDocumentFromElements("size",
					bson.C.Double("h", 8.5),
					bson.C.Int32("w", 11),
					bson.C.String("uom", "in"),
				),
				bson.C.String("status", "P"),
			),
			bson.NewDocument(
				bson.C.String("item", "paper"),
				bson.C.Int32("qty", 100),
				bson.C.SubDocumentFromElements("size",
					bson.C.Double("h", 8.5),
					bson.C.Int32("w", 11),
					bson.C.String("uom", "in"),
				),
				bson.C.String("status", "D"),
			),
			bson.NewDocument(
				bson.C.String("item", "planner"),
				bson.C.Int32("qty", 75),
				bson.C.SubDocumentFromElements("size",
					bson.C.Double("h", 22.85),
					bson.C.Int32("w", 30),
					bson.C.String("uom", "cm"),
				),
				bson.C.String("status", "D"),
			),
			bson.NewDocument(
				bson.C.String("item", "postcard"),
				bson.C.Int32("qty", 45),
				bson.C.SubDocumentFromElements("size",
					bson.C.Int32("h", 10),
					bson.C.Double("w", 15.25),
					bson.C.String("uom", "cm"),
				),
				bson.C.String("status", "A"),
			),
			bson.NewDocument(
				bson.C.String("item", "sketchbook"),
				bson.C.Int32("qty", 80),
				bson.C.SubDocumentFromElements("size",
					bson.C.Int32("h", 14),
					bson.C.Int32("w", 21),
					bson.C.String("uom", "cm"),
				),
				bson.C.String("status", "A"),
			),
			bson.NewDocument(
				bson.C.String("item", "sketch pad"),
				bson.C.Int32("qty", 95),
				bson.C.SubDocumentFromElements("size",
					bson.C.Double("h", 22.85),
					bson.C.Double("w", 30.5),
					bson.C.String("uom", "cm"),
				),
				bson.C.String("status", "A"),
			),
		}

		result, err := coll.InsertMany(context.Background(), docs)

		// End Example 51

		require.NoError(t, err)
		require.Len(t, result.InsertedIDs, 10)
	}

	{
		// Start Example 52

		result, err := coll.UpdateOne(
			context.Background(),
			bson.NewDocument(
				bson.C.String("item", "paper"),
			),
			bson.NewDocument(
				bson.C.SubDocumentFromElements("$set",
					bson.C.String("size.uom", "cm"),
					bson.C.String("status", "P"),
				),
				bson.C.SubDocumentFromElements("$currentDate",
					bson.C.Boolean("lastModified", true),
				),
			),
		)

		// End Example 52

		require.NoError(t, err)
		require.Equal(t, int64(1), result.MatchedCount)
		require.Equal(t, int64(1), result.ModifiedCount)

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.String("item", "paper"),
			))

		require.NoError(t, err)

		doc := bson.NewDocument()
		for cursor.Next(context.Background()) {
			err := cursor.Decode(doc)
			require.NoError(t, err)

			uom, err := doc.Lookup("size", "uom")
			require.NoError(t, err)
			require.Equal(t, uom.Value().StringValue(), "cm")

			status, err := doc.Lookup("status")
			require.NoError(t, err)
			require.Equal(t, status.Value().StringValue(), "P")

			keys, err := doc.Keys(false)
			require.NoError(t, err)
			require.True(t, containsKey(keys, "lastModified", nil))
		}

		require.NoError(t, cursor.Err())
	}

	{
		// Start Example 53

		result, err := coll.UpdateMany(
			context.Background(),
			bson.NewDocument(
				bson.C.SubDocumentFromElements("qty",
					bson.C.Int32("$lt", 50),
				),
			),
			bson.NewDocument(
				bson.C.SubDocumentFromElements("$set",
					bson.C.String("size.uom", "cm"),
					bson.C.String("status", "P"),
				),
				bson.C.SubDocumentFromElements("$currentDate",
					bson.C.Boolean("lastModified", true),
				),
			),
		)

		// End Example 53

		require.NoError(t, err)
		require.Equal(t, int64(3), result.MatchedCount)
		require.Equal(t, int64(3), result.ModifiedCount)

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.SubDocumentFromElements("qty",
					bson.C.Int32("$lt", 50),
				),
			))

		require.NoError(t, err)

		doc := bson.NewDocument()
		for cursor.Next(context.Background()) {
			err := cursor.Decode(doc)
			require.NoError(t, err)

			uom, err := doc.Lookup("size", "uom")
			require.NoError(t, err)
			require.Equal(t, uom.Value().StringValue(), "cm")

			status, err := doc.Lookup("status")
			require.NoError(t, err)
			require.Equal(t, status.Value().StringValue(), "P")

			keys, err := doc.Keys(false)
			require.NoError(t, err)
			require.True(t, containsKey(keys, "lastModified", nil))
		}

		require.NoError(t, cursor.Err())
	}

	{
		// Start Example 54

		result, err := coll.ReplaceOne(
			context.Background(),
			bson.NewDocument(
				bson.C.String("item", "paper"),
			),
			bson.NewDocument(
				bson.C.String("item", "paper"),
				bson.C.ArrayFromElements("instock",
					bson.AC.DocumentFromElements(
						bson.C.String("warehouse", "A"),
						bson.C.Int32("qty", 60),
					),
					bson.AC.DocumentFromElements(
						bson.C.String("warehouse", "B"),
						bson.C.Int32("qty", 40),
					),
				),
			),
		)

		// End Example 54

		require.NoError(t, err)
		require.Equal(t, int64(1), result.MatchedCount)
		require.Equal(t, int64(1), result.ModifiedCount)

		cursor, err := coll.Find(
			context.Background(),
			bson.NewDocument(
				bson.C.String("item", "paper"),
			))

		require.NoError(t, err)

		doc := bson.NewDocument()
		for cursor.Next(context.Background()) {
			err := cursor.Decode(doc)
			require.NoError(t, err)

			keys, err := doc.Keys(false)
			require.NoError(t, err)
			require.Len(t, keys, 3)

			require.True(t, containsKey(keys, "_id", nil))
			require.True(t, containsKey(keys, "item", nil))
			require.True(t, containsKey(keys, "instock", nil))

			instock, err := doc.Lookup("instock")
			require.NoError(t, err)
			require.Equal(t, instock.Value().MutableArray().Len(), 2)

		}

		require.NoError(t, cursor.Err())
	}

}

func DeleteExamples(t *testing.T, db *mongo.Database) {
	_, err := db.RunCommand(
		context.Background(),
		bson.NewDocument(bson.C.Int32("dropDatabase", 1)),
	)
	require.NoError(t, err)

	coll := db.Collection("inventory")

	{
		// Start Example 55
		docs := []interface{}{
			bson.NewDocument(
				bson.C.String("item", "journal"),
				bson.C.Int32("qty", 25),
				bson.C.SubDocumentFromElements("size",
					bson.C.Int32("h", 14),
					bson.C.Int32("w", 21),
					bson.C.String("uom", "cm"),
				),
				bson.C.String("status", "A"),
			),
			bson.NewDocument(
				bson.C.String("item", "notebook"),
				bson.C.Int32("qty", 50),
				bson.C.SubDocumentFromElements("size",
					bson.C.Double("h", 8.5),
					bson.C.Int32("w", 11),
					bson.C.String("uom", "in"),
				),
				bson.C.String("status", "P"),
			),
			bson.NewDocument(
				bson.C.String("item", "paper"),
				bson.C.Int32("qty", 100),
				bson.C.SubDocumentFromElements("size",
					bson.C.Double("h", 8.5),
					bson.C.Int32("w", 11),
					bson.C.String("uom", "in"),
				),
				bson.C.String("status", "D"),
			),
			bson.NewDocument(
				bson.C.String("item", "planner"),
				bson.C.Int32("qty", 75),
				bson.C.SubDocumentFromElements("size",
					bson.C.Double("h", 22.85),
					bson.C.Int32("w", 30),
					bson.C.String("uom", "cm"),
				),
				bson.C.String("status", "D"),
			),
			bson.NewDocument(
				bson.C.String("item", "postcard"),
				bson.C.Int32("qty", 45),
				bson.C.SubDocumentFromElements("size",
					bson.C.Int32("h", 10),
					bson.C.Double("w", 15.25),
					bson.C.String("uom", "cm"),
				),
				bson.C.String("status", "A"),
			),
		}

		result, err := coll.InsertMany(context.Background(), docs)

		// End Example 55

		require.NoError(t, err)
		require.Len(t, result.InsertedIDs, 5)
	}

	{
		// Start Example 57

		result, err := coll.DeleteMany(
			context.Background(),
			bson.NewDocument(
				bson.C.String("status", "A"),
			),
		)

		// End Example 57

		require.NoError(t, err)
		require.Equal(t, int64(2), result.DeletedCount)
	}

	{
		// Start Example 58

		result, err := coll.DeleteOne(
			context.Background(),
			bson.NewDocument(
				bson.C.String("status", "D"),
			),
		)

		// End Example 58

		require.NoError(t, err)
		require.Equal(t, int64(1), result.DeletedCount)

	}

	{
		// Start Example 56

		result, err := coll.DeleteMany(context.Background(), bson.NewDocument())

		// End Example 56

		require.NoError(t, err)
		require.Equal(t, int64(2), result.DeletedCount)
	}
}
