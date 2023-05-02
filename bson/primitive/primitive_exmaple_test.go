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

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		panic(err)
	}

	defer client.Disconnect(ctx)

	// Create a collection
	col := client.Database("test").Collection("test")
	defer col.Drop(ctx)

	// Insert a simple document to lookup using regex.
	doc := bson.D{{"foo", "bar"}}

	if _, err := col.InsertOne(ctx, doc); err != nil {
		panic(err)
	}

	// Create a filter to find a document with key "foo" with a value that
	// starts with "b".
	filter := bson.D{{"foo", primitive.Regex{Pattern: "^b", Options: ""}}}

	// Remove the "_id" from the results.
	options := options.FindOne().SetProjection(bson.D{{"_id", 0}})

	result := col.FindOne(ctx, filter, options)
	if err := result.Err(); err != nil {
		panic(err)
	}

	var got bson.Raw
	if err := result.Decode(&got); err != nil {
		panic(err)
	}

	fmt.Printf(got.String())
	//Output:{"foo": "bar"}
}
