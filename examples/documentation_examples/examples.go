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
	"github.com/mongodb/mongo-go-driver/bson/primitive"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
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

func containsKey(doc bsonx.Doc, key ...string) bool {
	_, err := doc.LookupErr(key...)
	if err != nil {
		return false
	}
	return true
}

func InsertExamples(t *testing.T, db *mongo.Database) {
	err := db.RunCommand(
		context.Background(),
		bson.D{{"dropDatabase", 1}},
	).Err()
	require.NoError(t, err)

	coll := db.Collection("inventory")

	{
		// Start Example 1

		result, err := coll.InsertOne(
			context.Background(),
			bson.D{
				{"item", "canvas"},
				{"qty", 100},
				{"tags", bson.A{"cotton"}},
				{"size", bson.D{
					{"h", 28},
					{"w", 35.5},
					{"uom", "cm"},
				}},
			})

		// End Example 1

		require.NoError(t, err)
		require.NotNil(t, result.InsertedID)
	}

	{
		// Start Example 2

		cursor, err := coll.Find(
			context.Background(),
			bson.D{{"item", "canvas"}},
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
				bson.D{
					{"item", bsonx.String("journal")},
					{"qty", bsonx.Int32(25)},
					{"tags", bson.A{"blank", "red"}},
					{"size", bson.D{
						{"h", 14},
						{"w", 21},
						{"uom", "cm"},
					}},
				},
				bson.D{
					{"item", bsonx.String("mat")},
					{"qty", bsonx.Int32(25)},
					{"tags", bson.A{"gray"}},
					{"size", bson.D{
						{"h", 27.9},
						{"w", 35.5},
						{"uom", "cm"},
					}},
				},
				bson.D{
					{"item", "mousepad"},
					{"qty", 25},
					{"tags", bson.A{"gel", "blue"}},
					{"size", bson.D{
						{"h", 19},
						{"w", 22.85},
						{"uom", "cm"},
					}},
				},
			})

		// End Example 3

		require.NoError(t, err)
		require.Len(t, result.InsertedIDs, 3)
	}
}

func QueryToplevelFieldsExamples(t *testing.T, db *mongo.Database) {
	err := db.RunCommand(
		context.Background(),
		bson.D{{"dropDatabase", 1}},
	).Err()
	require.NoError(t, err)

	coll := db.Collection("inventory")

	{
		// Start Example 6

		docs := []interface{}{
			bson.D{
				{"item", "journal"},
				{"qty", 25},
				{"size", bson.D{
					{"h", 14},
					{"w", 21},
					{"uom", "cm"},
				}},
				{"status", "A"},
			},
			bson.D{
				{"item", "notebook"},
				{"qty", 50},
				{"size", bson.D{
					{"h", 8.5},
					{"w", 11},
					{"uom", "in"},
				}},
				{"status", "A"},
			},
			bson.D{
				{"item", "paper"},
				{"qty", 100},
				{"size", bson.D{
					{"h", 8.5},
					{"w", 11},
					{"uom", "in"},
				}},
				{"status", "D"},
			},
			bson.D{
				{"item", "planner"},
				{"qty", 75},
				{"size", bson.D{
					{"h", 22.85},
					{"w", 30},
					{"uom", "cm"},
				}},
				{"status", "D"},
			},
			bson.D{
				{"item", "postcard"},
				{"qty", 45},
				{"size", bson.D{
					{"h", 10},
					{"w", 15.25},
					{"uom", "cm"},
				}},
				{"status", "A"},
			},
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
			bson.D{},
		)

		// End Example 7

		require.NoError(t, err)
		requireCursorLength(t, cursor, 5)
	}

	{
		// Start Example 9

		cursor, err := coll.Find(
			context.Background(),
			bson.D{{"status", "D"}},
		)

		// End Example 9

		require.NoError(t, err)
		requireCursorLength(t, cursor, 2)
	}

	{
		// Start Example 10

		cursor, err := coll.Find(
			context.Background(),
			bson.D{{"status", bson.D{{"$in", bson.A{"A", "D"}}}}})

		// End Example 10

		require.NoError(t, err)
		requireCursorLength(t, cursor, 5)
	}

	{
		// Start Example 11

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"status", "A"},
				{"qty", bson.D{{"$lt", 30}}},
			})

		// End Example 11

		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)
	}

	{
		// Start Example 12

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"$or",
					bson.A{
						bson.D{{"status", "A"}},
						bson.D{{"qty", bson.D{{"$lt", 30}}}},
					}},
			})

		// End Example 12

		require.NoError(t, err)
		requireCursorLength(t, cursor, 3)
	}

	{
		// Start Example 13

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"status", "A"},
				{"$or", bson.A{
					bson.D{{"qty", bson.D{{"$lt", 30}}}},
					bson.D{{"item", primitive.Regex{Pattern: "^p", Options: ""}}},
				}},
			})

		// End Example 13

		require.NoError(t, err)
		requireCursorLength(t, cursor, 2)
	}

}

func QueryEmbeddedDocumentsExamples(t *testing.T, db *mongo.Database) {
	err := db.RunCommand(
		context.Background(),
		bson.D{{"dropDatabase", 1}},
	).Err()
	require.NoError(t, err)

	coll := db.Collection("inventory")

	{
		// Start Example 14

		docs := []interface{}{
			bson.D{
				{"item", "journal"},
				{"qty", 25},
				{"size", bson.D{
					{"h", 14},
					{"w", 21},
					{"uom", "cm"},
				}},
				{"status", "A"},
			},
			bson.D{
				{"item", "notebook"},
				{"qty", 50},
				{"size", bson.D{
					{"h", 8.5},
					{"w", 11},
					{"uom", "in"},
				}},
				{"status", "A"},
			},
			bson.D{
				{"item", "paper"},
				{"qty", 100},
				{"size", bson.D{
					{"h", 8.5},
					{"w", 11},
					{"uom", "in"},
				}},
				{"status", "D"},
			},
			bson.D{
				{"item", "planner"},
				{"qty", 75},
				{"size", bson.D{
					{"h", 22.85},
					{"w", 30},
					{"uom", "cm"},
				}},
				{"status", "D"},
			},
			bson.D{
				{"item", "postcard"},
				{"qty", 45},
				{"size", bson.D{
					{"h", 10},
					{"w", 15.25},
					{"uom", "cm"},
				}},
				{"status", "A"},
			},
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
			bson.D{
				{"size", bson.D{
					{"h", 14},
					{"w", 21},
					{"uom", "cm"},
				}},
			})

		// End Example 15

		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)
	}

	{
		// Start Example 16

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"size", bson.D{
					{"w", 21},
					{"h", 14},
					{"uom", "cm"},
				}},
			})

		// End Example 16

		require.NoError(t, err)
		requireCursorLength(t, cursor, 0)
	}

	{
		// Start Example 17

		cursor, err := coll.Find(
			context.Background(),
			bson.D{{"size.uom", "in"}},
		)

		// End Example 17

		require.NoError(t, err)
		requireCursorLength(t, cursor, 2)
	}

	{
		// Start Example 18

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"size.h", bson.D{
					{"$lt", 15},
				}},
			})

		// End Example 18

		require.NoError(t, err)
		requireCursorLength(t, cursor, 4)
	}

	{
		// Start Example 19

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"size.h", bson.D{
					{"$lt", 15},
				}},
				{"size.uom", "in"},
				{"status", "D"},
			})

		// End Example 19

		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)
	}

}

func QueryArraysExamples(t *testing.T, db *mongo.Database) {
	err := db.RunCommand(
		context.Background(),
		bson.D{{"dropDatabase", 1}},
	).Err()
	require.NoError(t, err)

	coll := db.Collection("inventory")

	{
		// Start Example 20

		docs := []interface{}{
			bson.D{
				{"item", "journal"},
				{"qty", 25},
				{"tags", bson.A{"blank", "red"}},
				{"dim_cm", bson.A{14, 21}},
			},
			bson.D{
				{"item", "notebook"},
				{"qty", 50},
				{"tags", bson.A{"red", "blank"}},
				{"dim_cm", bson.A{14, 21}},
			},
			bson.D{
				{"item", "paper"},
				{"qty", 100},
				{"tags", bson.A{"red", "blank", "plain"}},
				{"dim_cm", bson.A{14, 21}},
			},
			bson.D{
				{"item", "planner"},
				{"qty", 75},
				{"tags", bson.A{"blank", "red"}},
				{"dim_cm", bson.A{22.85, 30}},
			},
			bson.D{
				{"item", "postcard"},
				{"qty", 45},
				{"tags", bson.A{"blue"}},
				{"dim_cm", bson.A{10, 15.25}},
			},
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
			bson.D{{"tags", bson.A{"red", "blank"}}},
		)

		// End Example 21

		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)
	}

	{
		// Start Example 22

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"tags", bson.D{{"$all", bson.A{"red", "blank"}}}},
			})

		// End Example 22

		require.NoError(t, err)
		requireCursorLength(t, cursor, 4)
	}

	{
		// Start Example 23

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"tags", "red"},
			})

		// End Example 23

		require.NoError(t, err)
		requireCursorLength(t, cursor, 4)
	}

	{
		// Start Example 24

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"dim_cm", bson.D{
					{"$gt", 25},
				}},
			})

		// End Example 24

		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)
	}

	{
		// Start Example 25

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"dim_cm", bson.D{
					{"$gt", 15},
					{"$lt", 20},
				}},
			})

		// End Example 25

		require.NoError(t, err)
		requireCursorLength(t, cursor, 4)
	}

	{
		// Start Example 26

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"dim_cm", bson.D{
					{"$elemMatch", bson.D{
						{"$gt", 22},
						{"$lt", 30},
					}},
				}},
			})

		// End Example 26

		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)
	}

	{
		// Start Example 27

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"dim_cm.1", bson.D{
					{"$gt", 25},
				}},
			})

		// End Example 27

		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)
	}

	{
		// Start Example 28

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"tags", bson.D{
					{"$size", 3},
				}},
			})

		// End Example 28

		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)
	}

}

func QueryArrayEmbeddedDocumentsExamples(t *testing.T, db *mongo.Database) {
	err := db.RunCommand(
		context.Background(),
		bson.D{{"dropDatabase", 1}},
	).Err()
	require.NoError(t, err)

	coll := db.Collection("inventory")

	{
		// Start Example 29

		docs := []interface{}{
			bson.D{
				{"item", "journal"},
				{"instock", bson.A{
					bson.D{
						{"warehouse", "A"},
						{"qty", 5},
					},
					bson.D{
						{"warehouse", "C"},
						{"qty", 15},
					},
				}},
			},
			bson.D{
				{"item", "notebook"},
				{"instock", bson.A{
					bson.D{
						{"warehouse", "C"},
						{"qty", 5},
					},
				}},
			},
			bson.D{
				{"item", "paper"},
				{"instock", bson.A{
					bson.D{
						{"warehouse", "A"},
						{"qty", 60},
					},
					bson.D{
						{"warehouse", "B"},
						{"qty", 15},
					},
				}},
			},
			bson.D{
				{"item", "planner"},
				{"instock", bson.A{
					bson.D{
						{"warehouse", "A"},
						{"qty", 40},
					},
					bson.D{
						{"warehouse", "B"},
						{"qty", 5},
					},
				}},
			},
			bson.D{
				{"item", "postcard"},
				{"instock", bson.A{
					bson.D{
						{"warehouse", "B"},
						{"qty", 15},
					},
					bson.D{
						{"warehouse", "C"},
						{"qty", 35},
					},
				}},
			},
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
			bson.D{
				{"instock", bson.D{
					{"warehouse", "A"},
					{"qty", 5},
				}},
			})

		// End Example 30

		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)
	}

	{
		// Start Example 31

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"instock", bson.D{
					{"qty", 5},
					{"warehouse", "A"},
				}},
			})

		// End Example 31

		require.NoError(t, err)
		requireCursorLength(t, cursor, 0)
	}

	{
		// Start Example 32

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"instock.0.qty", bson.D{
					{"$lte", 20},
				}},
			})

		// End Example 32

		require.NoError(t, err)
		requireCursorLength(t, cursor, 3)
	}

	{
		// Start Example 33

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"instock.qty", bson.D{
					{"$lte", 20},
				}},
			})

		// End Example 33

		require.NoError(t, err)
		requireCursorLength(t, cursor, 5)
	}

	{
		// Start Example 34

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"instock", bson.D{
					{"$elemMatch", bson.D{
						{"qty", 5},
						{"warehouse", "A"},
					}},
				}},
			})

		// End Example 34

		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)
	}

	{
		// Start Example 35

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"instock", bson.D{
					{"$elemMatch", bson.D{
						{"qty", bson.D{
							{"$gt", 10},
							{"$lte", 20},
						}},
					}},
				}},
			})

		// End Example 35

		require.NoError(t, err)
		requireCursorLength(t, cursor, 3)
	}

	{
		// Start Example 36

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"instock.qty", bson.D{
					{"$gt", 10},
					{"$lte", 20},
				}},
			})

		// End Example 36

		require.NoError(t, err)
		requireCursorLength(t, cursor, 4)
	}

	{
		// Start Example 37

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"instock.qty", 5},
				{"instock.warehouse", "A"},
			})

		// End Example 37

		require.NoError(t, err)
		requireCursorLength(t, cursor, 2)
	}
}

func QueryNullMissingFieldsExamples(t *testing.T, db *mongo.Database) {
	err := db.RunCommand(
		context.Background(),
		bson.D{{"dropDatabase", 1}},
	).Err()
	require.NoError(t, err)

	coll := db.Collection("inventory")

	{
		// Start Example 38

		docs := []interface{}{
			bson.D{
				{"_id", 1},
				{"item", nil},
			},
			bson.D{
				{"_id", 2},
			},
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
			bson.D{
				{"item", nil},
			})

		// End Example 39

		require.NoError(t, err)
		requireCursorLength(t, cursor, 2)
	}

	{
		// Start Example 40

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"item", bson.D{
					{"$type", 10},
				}},
			})

		// End Example 40

		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)
	}

	{
		// Start Example 41

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"item", bson.D{
					{"$exists", false},
				}},
			})

		// End Example 41

		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)
	}
}

func ProjectionExamples(t *testing.T, db *mongo.Database) {
	err := db.RunCommand(
		context.Background(),
		bson.D{{"dropDatabase", 1}},
	).Err()
	require.NoError(t, err)

	coll := db.Collection("inventory")

	{
		// Start Example 42

		docs := []interface{}{
			bson.D{
				{"item", "journal"},
				{"status", "A"},
				{"size", bson.D{
					{"h", 14},
					{"w", 21},
					{"uom", "cm"},
				}},
				{"instock", bson.A{
					bson.D{
						{"warehouse", "A"},
						{"qty", 5},
					},
				}},
			},
			bson.D{
				{"item", "notebook"},
				{"status", "A"},
				{"size", bson.D{
					{"h", 8.5},
					{"w", 11},
					{"uom", "in"},
				}},
				{"instock", bson.A{
					bson.D{
						{"warehouse", "EC"},
						{"qty", 5},
					},
				}},
			},
			bson.D{
				{"item", "paper"},
				{"status", "D"},
				{"size", bson.D{
					{"h", 8.5},
					{"w", 11},
					{"uom", "in"},
				}},
				{"instock", bson.A{
					bson.D{
						{"warehouse", "A"},
						{"qty", 60},
					},
				}},
			},
			bson.D{
				{"item", "planner"},
				{"status", "D"},
				{"size", bson.D{
					{"h", 22.85},
					{"w", 30},
					{"uom", "cm"},
				}},
				{"instock", bson.A{
					bson.D{
						{"warehouse", "A"},
						{"qty", 40},
					},
				}},
			},
			bson.D{
				{"item", "postcard"},
				{"status", "A"},
				{"size", bson.D{
					{"h", 10},
					{"w", 15.25},
					{"uom", "cm"},
				}},
				{"instock", bson.A{
					bson.D{
						{"warehouse", "B"},
						{"qty", 15},
					},
					bson.D{
						{"warehouse", "EC"},
						{"qty", 35},
					},
				}},
			},
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
			bson.D{{"status", "A"}},
		)

		// End Example 43

		require.NoError(t, err)
		requireCursorLength(t, cursor, 3)
	}

	{
		// Start Example 44

		projection := bson.D{
			{"item", 1},
			{"status", 1},
		}

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"status", "A"},
			},
			options.Find().SetProjection(projection),
		)

		// End Example 44

		require.NoError(t, err)

		doc := bsonx.Doc{}
		for cursor.Next(context.Background()) {
			doc = doc[:0]
			err := cursor.Decode(doc)
			require.NoError(t, err)

			require.True(t, containsKey(doc, "_id"))
			require.True(t, containsKey(doc, "item"))
			require.True(t, containsKey(doc, "status"))
			require.False(t, containsKey(doc, "size"))
			require.False(t, containsKey(doc, "instock"))
		}

		require.NoError(t, cursor.Err())
	}

	{
		// Start Example 45

		projection := bson.D{
			{"item", 1},
			{"status", 1},
			{"_id", 0},
		}

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"status", "A"},
			},
			options.Find().SetProjection(projection),
		)

		// End Example 45

		require.NoError(t, err)

		doc := bsonx.Doc{}
		for cursor.Next(context.Background()) {
			doc = doc[:0]
			err := cursor.Decode(doc)
			require.NoError(t, err)

			require.False(t, containsKey(doc, "_id"))
			require.True(t, containsKey(doc, "item"))
			require.True(t, containsKey(doc, "status"))
			require.False(t, containsKey(doc, "size"))
			require.False(t, containsKey(doc, "instock"))
		}

		require.NoError(t, cursor.Err())
	}

	{
		// Start Example 46

		projection := bson.D{
			{"status", 0},
			{"instock", 0},
		}

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"status", "A"},
			},
			options.Find().SetProjection(projection),
		)

		// End Example 46

		require.NoError(t, err)

		doc := bsonx.Doc{}
		for cursor.Next(context.Background()) {
			doc = doc[:0]
			err := cursor.Decode(doc)
			require.NoError(t, err)

			require.True(t, containsKey(doc, "_id"))
			require.True(t, containsKey(doc, "item"))
			require.False(t, containsKey(doc, "status"))
			require.True(t, containsKey(doc, "size"))
			require.False(t, containsKey(doc, "instock"))
		}

		require.NoError(t, cursor.Err())
	}

	{
		// Start Example 47

		projection := bson.D{
			{"item", 1},
			{"status", 1},
			{"size.uom", 1},
		}

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"status", "A"},
			},
			options.Find().SetProjection(projection),
		)

		// End Example 47

		require.NoError(t, err)

		doc := bsonx.Doc{}
		for cursor.Next(context.Background()) {
			doc = doc[:0]
			err := cursor.Decode(doc)
			require.NoError(t, err)

			require.True(t, containsKey(doc, "_id"))
			require.True(t, containsKey(doc, "item"))
			require.True(t, containsKey(doc, "status"))
			require.True(t, containsKey(doc, "size"))
			require.False(t, containsKey(doc, "instock"))

			require.True(t, containsKey(doc, "uom", "size"))
			require.False(t, containsKey(doc, "h", "size"))
			require.False(t, containsKey(doc, "w", "size"))

		}

		require.NoError(t, cursor.Err())
	}

	{
		// Start Example 48

		projection := bson.D{
			{"size.uom", 0},
		}

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"status", "A"},
			},
			options.Find().SetProjection(projection),
		)

		// End Example 48

		require.NoError(t, err)

		doc := bsonx.Doc{}
		for cursor.Next(context.Background()) {
			doc = doc[:0]
			err := cursor.Decode(doc)
			require.NoError(t, err)

			require.True(t, containsKey(doc, "_id"))
			require.True(t, containsKey(doc, "item"))
			require.True(t, containsKey(doc, "status"))
			require.True(t, containsKey(doc, "size"))
			require.True(t, containsKey(doc, "instock"))

			require.False(t, containsKey(doc, "uom", "size"))
			require.True(t, containsKey(doc, "h", "size"))
			require.True(t, containsKey(doc, "w", "size"))

		}

		require.NoError(t, cursor.Err())
	}

	{
		// Start Example 49

		projection := bson.D{
			{"item", 1},
			{"status", 1},
			{"instock.qty", 1},
		}

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"status", "A"},
			},
			options.Find().SetProjection(projection),
		)

		// End Example 49

		require.NoError(t, err)

		doc := bsonx.Doc{}
		for cursor.Next(context.Background()) {
			doc = doc[:0]
			err := cursor.Decode(doc)
			require.NoError(t, err)

			require.True(t, containsKey(doc, "_id"))
			require.True(t, containsKey(doc, "item"))
			require.True(t, containsKey(doc, "status"))
			require.False(t, containsKey(doc, "size"))
			require.True(t, containsKey(doc, "instock"))

			instock, err := doc.LookupErr("instock")
			require.NoError(t, err)

			arr := instock.Array()

			for _, val := range arr {
				require.Equal(t, bson.TypeEmbeddedDocument, val.Type())
				subdoc := val.Document()

				require.Equal(t, 1, len(subdoc))
				_, err = subdoc.LookupErr("qty")
				require.NoError(t, err)
			}
		}

		require.NoError(t, cursor.Err())
	}

	{
		// Start Example 50

		projection := bson.D{
			{"item", 1},
			{"status", 1},
			{"instock", bson.D{
				{"$slice", -1},
			}},
		}

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"status", "A"},
			},
			options.Find().SetProjection(projection),
		)

		// End Example 50

		require.NoError(t, err)

		doc := bsonx.Doc{}
		for cursor.Next(context.Background()) {
			doc = doc[:0]
			err := cursor.Decode(doc)
			require.NoError(t, err)

			require.True(t, containsKey(doc, "_id"))
			require.True(t, containsKey(doc, "item"))
			require.True(t, containsKey(doc, "status"))
			require.False(t, containsKey(doc, "size"))
			require.True(t, containsKey(doc, "instock"))

			instock, err := doc.LookupErr("instock")
			require.NoError(t, err)
			require.Equal(t, len(instock.Array()), 1)
		}

		require.NoError(t, cursor.Err())
	}
}

func UpdateExamples(t *testing.T, db *mongo.Database) {
	err := db.RunCommand(
		context.Background(),
		bson.D{{"dropDatabase", 1}},
	).Err()
	require.NoError(t, err)

	coll := db.Collection("inventory")

	{
		// Start Example 51

		docs := []interface{}{
			bson.D{
				{"item", "canvas"},
				{"qty", 100},
				{"size", bson.D{
					{"h", 28},
					{"w", 35.5},
					{"uom", "cm"},
				}},
				{"status", "A"},
			},
			bson.D{
				{"item", "journal"},
				{"qty", 25},
				{"size", bson.D{
					{"h", 14},
					{"w", 21},
					{"uom", "cm"},
				}},
				{"status", "A"},
			},
			bson.D{
				{"item", "mat"},
				{"qty", 85},
				{"size", bson.D{
					{"h", 27.9},
					{"w", 35.5},
					{"uom", "cm"},
				}},
				{"status", "A"},
			},
			bson.D{
				{"item", "mousepad"},
				{"qty", 25},
				{"size", bson.D{
					{"h", 19},
					{"w", 22.85},
					{"uom", "in"},
				}},
				{"status", "P"},
			},
			bson.D{
				{"item", "notebook"},
				{"qty", 50},
				{"size", bson.D{
					{"h", 8.5},
					{"w", 11},
					{"uom", "in"},
				}},
				{"status", "P"},
			},
			bson.D{
				{"item", "paper"},
				{"qty", 100},
				{"size", bson.D{
					{"h", 8.5},
					{"w", 11},
					{"uom", "in"},
				}},
				{"status", "D"},
			},
			bson.D{
				{"item", "planner"},
				{"qty", 75},
				{"size", bson.D{
					{"h", 22.85},
					{"w", 30},
					{"uom", "cm"},
				}},
				{"status", "D"},
			},
			bson.D{
				{"item", "postcard"},
				{"qty", 45},
				{"size", bson.D{
					{"h", 10},
					{"w", 15.25},
					{"uom", "cm"},
				}},
				{"status", "A"},
			},
			bson.D{
				{"item", "sketchbook"},
				{"qty", 80},
				{"size", bson.D{
					{"h", 14},
					{"w", 21},
					{"uom", "cm"},
				}},
				{"status", "A"},
			},
			bson.D{
				{"item", "sketch pad"},
				{"qty", 95},
				{"size", bson.D{
					{"h", 22.85},
					{"w", 30.5},
					{"uom", "cm"},
				}},
				{"status", "A"},
			},
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
			bson.D{
				{"item", "paper"},
			},
			bson.D{
				{"$set", bson.D{
					{"size.uom", "cm"},
					{"status", "P"},
				}},
				{"$currentDate", bson.D{
					{"lastModified", true},
				}},
			},
		)

		// End Example 52

		require.NoError(t, err)
		require.Equal(t, int64(1), result.MatchedCount)
		require.Equal(t, int64(1), result.ModifiedCount)

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"item", "paper"},
			})

		require.NoError(t, err)

		doc := bsonx.Doc{}
		for cursor.Next(context.Background()) {
			doc = doc[:0]
			err := cursor.Decode(doc)
			require.NoError(t, err)

			uom, err := doc.LookupErr("size", "uom")
			require.NoError(t, err)
			require.Equal(t, uom.StringValue(), "cm")

			status, err := doc.LookupErr("status")
			require.NoError(t, err)
			require.Equal(t, status.StringValue(), "P")

			require.True(t, containsKey(doc, "lastModified"))
		}

		require.NoError(t, cursor.Err())
	}

	{
		// Start Example 53

		result, err := coll.UpdateMany(
			context.Background(),
			bson.D{
				{"qty", bson.D{
					{"$lt", 50},
				}},
			},
			bson.D{
				{"$set", bson.D{
					{"size.uom", "cm"},
					{"status", "P"},
				}},
				{"$currentDate", bson.D{
					{"lastModified", true},
				}},
			},
		)

		// End Example 53

		require.NoError(t, err)
		require.Equal(t, int64(3), result.MatchedCount)
		require.Equal(t, int64(3), result.ModifiedCount)

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"qty", bson.D{
					{"$lt", 50},
				}},
			})

		require.NoError(t, err)

		doc := bsonx.Doc{}
		for cursor.Next(context.Background()) {
			doc = doc[:0]
			err := cursor.Decode(doc)
			require.NoError(t, err)

			uom, err := doc.LookupErr("size", "uom")
			require.NoError(t, err)
			require.Equal(t, uom.StringValue(), "cm")

			status, err := doc.LookupErr("status")
			require.NoError(t, err)
			require.Equal(t, status.StringValue(), "P")

			require.True(t, containsKey(doc, "lastModified"))
		}

		require.NoError(t, cursor.Err())
	}

	{
		// Start Example 54

		result, err := coll.ReplaceOne(
			context.Background(),
			bson.D{
				{"item", "paper"},
			},
			bson.D{
				{"item", "paper"},
				{"instock", bson.A{
					bson.D{
						{"warehouse", "A"},
						{"qty", 60},
					},
					bson.D{
						{"warehouse", "B"},
						{"qty", 40},
					},
				}},
			},
		)

		// End Example 54

		require.NoError(t, err)
		require.Equal(t, int64(1), result.MatchedCount)
		require.Equal(t, int64(1), result.ModifiedCount)

		cursor, err := coll.Find(
			context.Background(),
			bson.D{
				{"item", "paper"},
			})

		require.NoError(t, err)

		doc := bsonx.Doc{}
		for cursor.Next(context.Background()) {
			doc = doc[:0]
			err := cursor.Decode(doc)
			require.NoError(t, err)

			require.True(t, containsKey(doc, "_id"))
			require.True(t, containsKey(doc, "item"))
			require.True(t, containsKey(doc, "instock"))

			instock, err := doc.LookupErr("instock")
			require.NoError(t, err)
			require.Equal(t, len(instock.Array()), 2)

		}

		require.NoError(t, cursor.Err())
	}

}

func DeleteExamples(t *testing.T, db *mongo.Database) {
	err := db.RunCommand(
		context.Background(),
		bson.D{{"dropDatabase", 1}},
	).Err()
	require.NoError(t, err)

	coll := db.Collection("inventory")

	{
		// Start Example 55
		docs := []interface{}{
			bson.D{
				{"item", "journal"},
				{"qty", 25},
				{"size", bson.D{
					{"h", 14},
					{"w", 21},
					{"uom", "cm"},
				}},
				{"status", "A"},
			},
			bson.D{
				{"item", "notebook"},
				{"qty", 50},
				{"size", bson.D{
					{"h", 8.5},
					{"w", 11},
					{"uom", "in"},
				}},
				{"status", "P"},
			},
			bson.D{
				{"item", "paper"},
				{"qty", 100},
				{"size", bson.D{
					{"h", 8.5},
					{"w", 11},
					{"uom", "in"},
				}},
				{"status", "D"},
			},
			bson.D{
				{"item", "planner"},
				{"qty", 75},
				{"size", bson.D{
					{"h", 22.85},
					{"w", 30},
					{"uom", "cm"},
				}},
				{"status", "D"},
			},
			bson.D{
				{"item", "postcard"},
				{"qty", 45},
				{"size", bson.D{
					{"h", 10},
					{"w", 15.25},
					{"uom", "cm"},
				}},
				{"status", "A"},
			},
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
			bson.D{
				{"status", "A"},
			},
		)

		// End Example 57

		require.NoError(t, err)
		require.Equal(t, int64(2), result.DeletedCount)
	}

	{
		// Start Example 58

		result, err := coll.DeleteOne(
			context.Background(),
			bson.D{
				{"status", "D"},
			},
		)

		// End Example 58

		require.NoError(t, err)
		require.Equal(t, int64(1), result.DeletedCount)

	}

	{
		// Start Example 56

		result, err := coll.DeleteMany(context.Background(), bsonx.Doc{})

		// End Example 56

		require.NoError(t, err)
		require.Equal(t, int64(2), result.DeletedCount)
	}
}
