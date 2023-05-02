package primitive_test

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ExampleRegex() {
	ctx := context.TODO()
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")

	// Connect to a mongodb server.
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		panic(err)
	}

	defer client.Disconnect(ctx)

	col := client.Database("test").Collection("test")
	defer col.Drop(ctx)

	// Create a slice of documents to insert. We will lookup a subset of
	// these documents using regex.
	toInsert := []interface{}{
		bson.D{{"foo", "bar"}},
		bson.D{{"foo", "baz"}},
		bson.D{{"foo", "qux"}},
	}

	if _, err := col.InsertMany(ctx, toInsert); err != nil {
		panic(err)
	}

	// Create a filter to find a document with key "foo" and any value that
	// starts with letter "b".
	filter := bson.D{{"foo", primitive.Regex{Pattern: "^b", Options: ""}}}

	// Remove "_id" from the results.
	options := options.Find().SetProjection(bson.D{{"_id", 0}})

	cursor, err := col.Find(ctx, filter, options)
	if err != nil {
		panic(err)
	}

	// Iterate over the cursor, printing the extended json for each result
	// returned by the filter.
	for cursor.Next(ctx) {
		var got bson.Raw
		if err := cursor.Decode(&got); err != nil {
			panic(err)
		}

		fmt.Println(got.String())
	}

	//Output:
	//{"foo": "bar"}
	//{"foo": "baz"}
}
