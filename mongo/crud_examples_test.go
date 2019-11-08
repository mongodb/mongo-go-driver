// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Client examples

func ExampleClient_ListDatabases() {
	var client *mongo.Client

	// use a filter to select non-empty databases
	result, err := client.ListDatabases(context.TODO(), bson.D{{"empty", false}})
	if err != nil {
		log.Fatal(err)
	}

	for _, db := range result.Databases {
		fmt.Printf("db: %v, size: %v\n", db.Name, db.SizeOnDisk)
	}
}

func ExampleClient_ListDatabaseNames() {
	var client *mongo.Client

	// use a filter to only select non-empty databases
	result, err := client.ListDatabaseNames(context.TODO(), bson.D{{"empty", false}})
	if err != nil {
		log.Fatal(err)
	}

	for _, db := range result {
		fmt.Println(db)
	}
}

func ExampleClient_Watch() {
	var client *mongo.Client

	// specify a pipeline that will only match "insert" events
	// specify the MaxAwaitTimeOption to have each attempt wait two seconds for new documents
	matchStage := bson.D{{"$match", bson.D{{"operationType", "insert"}}}}
	opts := options.ChangeStream().SetMaxAwaitTime(2 * time.Second)
	changeStream, err := client.Watch(context.TODO(), mongo.Pipeline{matchStage}, opts)
	if err != nil {
		log.Fatal(err)
	}

	// print out all change stream events in the order they're received
	// see the mongo.ChangeStream documentation for more examples of using change streams
	for changeStream.Next(context.TODO()) {
		fmt.Println(changeStream.Current)
	}
}

// Database examples

func ExampleDatabase_Aggregate() {
	var db *mongo.Database

	// specify a pipeline that will list all sessions on the target server
	// specify the MaxTime option to limit the amount of time the operation can run on the server
	localSessionsStage := bson.D{{"$listLocalSessions", bson.D{{"allUsers", true}}}}
	opts := options.Aggregate().SetMaxTime(2 * time.Second)
	cursor, err := db.Aggregate(context.TODO(), mongo.Pipeline{localSessionsStage}, opts)
	if err != nil {
		log.Fatal(err)
	}

	// get a list of all returned documents and print them out
	// see the mongo.Cursor documentation for more examples of using cursors
	var results []bson.M
	if err = cursor.All(context.TODO(), &results); err != nil {
		log.Fatal(err)
	}
	for _, result := range results {
		fmt.Println(result)
	}
}

func ExampleDatabase_ListCollectionNames() {
	var db *mongo.Database

	// use a filter to only select capped collections
	result, err := db.ListCollectionNames(context.TODO(), bson.D{{"options.capped", true}})
	if err != nil {
		log.Fatal(err)
	}

	for _, coll := range result {
		fmt.Println(coll)
	}
}

func ExampleDatabase_ListCollections() {
	var db *mongo.Database

	// use a filter to only select capped collections
	cursor, err := db.ListCollections(context.TODO(), bson.D{{"options.capped", "true"}})
	if err != nil {
		log.Fatal(err)
	}

	// get a list of all returned specifications and print them out
	// see the mongo.Cursor documentation for more examples of using cursors
	var results []bson.M
	if err = cursor.All(context.TODO(), &results); err != nil {
		log.Fatal(err)
	}
	for _, result := range results {
		fmt.Println(result)
	}
}

func ExampleDatabase_RunCommand() {
	var db *mongo.Database

	// create a capped collection named "capped-coll"
	// specify the ReadPreference option to explicitly set the read preference to primary
	command := bson.D{{"create", "capped-coll"}, {"capped", true}, {"size", 4096}}
	opts := options.RunCmd().SetReadPreference(readpref.Primary())
	var result bson.M
	if err := db.RunCommand(context.TODO(), command, opts).Decode(&result); err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
}

func ExampleDatabase_RunCommandCursor() {
	var db *mongo.Database

	// run a find command against collection "foo"
	// specify the ReadPreference option to explicitly set the read preference to primary
	command := bson.D{{"find", "foo"}}
	opts := options.RunCmd().SetReadPreference(readpref.Primary())
	cursor, err := db.RunCommandCursor(context.TODO(), command, opts)
	if err != nil {
		log.Fatal(err)
	}

	// get a list of all returned documents and print them out
	// see the mongo.Cursor documentation for more examples of using cursors
	var results []bson.M
	if err = cursor.All(context.TODO(), &results); err != nil {
		log.Fatal(err)
	}
	for _, result := range results {
		fmt.Println(result)
	}
}

func ExampleDatabase_Watch() {
	var db *mongo.Database

	// specify a pipeline that will only match "insert" events
	// specify the MaxAwaitTimeOption to have each attempt wait two seconds for new documents
	matchStage := bson.D{{"$match", bson.D{{"operationType", "insert"}}}}
	opts := options.ChangeStream().SetMaxAwaitTime(2 * time.Second)
	changeStream, err := db.Watch(context.TODO(), mongo.Pipeline{matchStage}, opts)
	if err != nil {
		log.Fatal(err)
	}

	// print out all change stream events in the order they're received
	// see the mongo.ChangeStream documentation for more examples of using change streams
	for changeStream.Next(context.TODO()) {
		fmt.Println(changeStream.Current)
	}
}

// Collection examples

func ExampleCollection_Aggregate() {
	var coll *mongo.Collection

	// specify a pipeline that will return the number of times each name appears in the collection
	// specify the MaxTime option to limit the amount of time the operation can run on the server
	groupStage := bson.D{
		{"$group", bson.D{
			{"_id", "$name"},
			{"numTimes", bson.D{
				{"$sum", 1},
			}},
		}},
	}
	opts := options.Aggregate().SetMaxTime(2 * time.Second)
	cursor, err := coll.Aggregate(context.TODO(), mongo.Pipeline{groupStage}, opts)
	if err != nil {
		log.Fatal(err)
	}

	// get a list of all returned documents and print them out
	// see the mongo.Cursor documentation for more examples of using cursors
	var results []bson.M
	if err = cursor.All(context.TODO(), &results); err != nil {
		log.Fatal(err)
	}
	for _, result := range results {
		fmt.Printf("name %v appears %v times\n", result["_id"], result["numTimes"])
	}
}

func ExampleCollection_BulkWrite() {
	var coll *mongo.Collection

	// insert {name: "Alice"} and delete {name: "Bob"}
	// set the Ordered option to false to allow both operations to happen even if one of them errors
	models := []mongo.WriteModel{
		mongo.NewInsertOneModel().SetDocument(bson.D{{"name", "Alice"}}),
		mongo.NewDeleteOneModel().SetFilter(bson.D{{"name", "Bob"}}),
	}
	opts := options.BulkWrite().SetOrdered(false)
	res, err := coll.BulkWrite(context.TODO(), models, opts)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("inserted %v and deleted %v documents\n", res.InsertedCount, res.DeletedCount)
}

func ExampleCollection_CountDocuments() {
	var coll *mongo.Collection

	// count the number of times the name "Bob" appears in the collection
	// specify the MaxTime option to limit the amount of time the operation can run on the server
	opts := options.Count().SetMaxTime(2 * time.Second)
	count, err := coll.CountDocuments(context.TODO(), bson.D{{"name", "Bob"}}, opts)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("name Bob appears in %v documents", count)
}

func ExampleCollection_DeleteMany() {
	var coll *mongo.Collection

	// delete all documents in which the "name" field is "Bob" or "bob"
	// specify the SetCollation option to provide a collation that will ignore case for string comparisons
	opts := options.Delete().SetCollation(&options.Collation{
		Locale:   "en_US",
		Strength: 1,
	})
	res, err := coll.DeleteMany(context.TODO(), bson.D{{"name", "bob"}}, opts)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("deleted %v documents\n", res.DeletedCount)
}

func ExampleCollection_DeleteOne() {
	var coll *mongo.Collection

	// delete at most one documents in which the "name" field is "Bob" or "bob"
	// specify the SetCollation option to provide a collation that will ignore case for string comparisons
	opts := options.Delete().SetCollation(&options.Collation{
		Locale:   "en_US",
		Strength: 1,
	})
	res, err := coll.DeleteOne(context.TODO(), bson.D{{"name", "bob"}}, opts)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("deleted %v documents\n", res.DeletedCount)
}

func ExampleCollection_Distinct() {
	var coll *mongo.Collection

	// find all unique values for the "name" field for documents in which the "age" field is greater than 25
	// specify the MaxTime option to limit the amount of time the operation can run on the server
	filter := bson.D{{"age", bson.D{{"$gt", 25}}}}
	opts := options.Distinct().SetMaxTime(2 * time.Second)
	values, err := coll.Distinct(context.TODO(), "name", filter, opts)
	if err != nil {
		log.Fatal(err)
	}

	for _, value := range values {
		fmt.Println(value)
	}
}

func ExampleCollection_EstimatedDocumentCount() {
	var coll *mongo.Collection

	// get and print an estimated of the number of documents in the collection
	// specify the MaxTime option to limit the amount of time the operation can run on the server
	opts := options.EstimatedDocumentCount().SetMaxTime(2 * time.Second)
	count, err := coll.EstimatedDocumentCount(context.TODO(), opts)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("estimated document count: %v", count)
}

func ExampleCollection_Find() {
	var coll *mongo.Collection

	// find all documents in which the "name" field is "Bob"
	// specify the MaxTime option to limit the amount of time the operation can run on the server
	opts := options.Find().SetMaxTime(2 * time.Second)
	cursor, err := coll.Find(context.TODO(), bson.D{{"name", "Bob"}}, opts)
	if err != nil {
		log.Fatal(err)
	}

	// get a list of all returned documents and print them out
	// see the mongo.Cursor documentation for more examples of using cursors
	var results []bson.M
	if err = cursor.All(context.TODO(), &results); err != nil {
		log.Fatal(err)
	}
	for _, result := range results {
		fmt.Println(result)
	}
}

func ExampleCollection_FindOne() {
	var coll *mongo.Collection

	// find the document in which the "name" field is "Bob"
	// specify the MaxTime option to limit the amount of time the operation can run on the server
	opts := options.FindOne().SetMaxTime(2 * time.Second)
	var result bson.M
	err := coll.FindOne(context.TODO(), bson.D{{"name", "Bob"}}, opts).Decode(&result)
	if err != nil {
		// ErrNoDocuments means that the filter did not match any documents in the collection
		if err == mongo.ErrNoDocuments {
			return
		}
		log.Fatal(err)
	}
	fmt.Printf("found document %v", result)
}

func ExampleCollection_FindOneAndDelete() {
	var coll *mongo.Collection

	// find and delete the document in which the "name" field is "Bob"
	// specify the MaxTime option to limit the amount of time the operation can run on the server
	opts := options.FindOneAndDelete().SetMaxTime(2 * time.Second)
	var deletedDocument bson.M
	err := coll.FindOneAndDelete(context.TODO(), bson.D{{"name", "Bob"}}, opts).Decode(&deletedDocument)
	if err != nil {
		// ErrNoDocuments means that the filter did not match any documents in the collection
		if err == mongo.ErrNoDocuments {
			return
		}
		log.Fatal(err)
	}
	fmt.Printf("deleted document %v", deletedDocument)
}

func ExampleCollection_FindOneAndReplace() {
	var coll *mongo.Collection

	// find and replace the document in which the "name" field is "Bob" with {name: "Alice"}
	// specify the Upsert option to insert {name: "Alice"} if the {name: "Bob"} document isn't found
	opts := options.FindOneAndReplace().SetUpsert(true)
	filter := bson.D{{"name", "Bob"}}
	replacement := bson.D{{"name", "Alice"}}
	var replacedDocument bson.M
	err := coll.FindOneAndReplace(context.TODO(), filter, replacement, opts).Decode(&replacedDocument)
	if err != nil {
		// ErrNoDocuments means that the filter did not match any documents in the collection
		if err == mongo.ErrNoDocuments {
			return
		}
		log.Fatal(err)
	}
	fmt.Printf("replaced document %v", replacedDocument)
}

func ExampleCollection_FindOneAndUpdate() {
	var coll *mongo.Collection

	// find the document in which the "name" field is "Bob" and set the name to "Alice"
	// specify the Upsert option to insert {name: "Alice"} if the {name: "Bob"} document isn't found
	opts := options.FindOneAndUpdate().SetUpsert(true)
	filter := bson.D{{"name", "Bob"}}
	update := bson.D{{"$set", bson.D{{"name", "Alice"}}}}
	var updatedDocument bson.M
	err := coll.FindOneAndUpdate(context.TODO(), filter, update, opts).Decode(&updatedDocument)
	if err != nil {
		// ErrNoDocuments means that the filter did not match any documents in the collection
		if err == mongo.ErrNoDocuments {
			return
		}
		log.Fatal(err)
	}
	fmt.Printf("updated document %v", updatedDocument)
}

func ExampleCollection_InsertMany() {
	var coll *mongo.Collection

	// insert documents {name: "Alice"} and {name: "Bob"}
	// set the Ordered option to false to allow both operations to happen even if one of them errors
	docs := []interface{}{
		bson.D{{"name", "Alice"}},
		bson.D{{"name", "Bob"}},
	}
	opts := options.InsertMany().SetOrdered(false)
	res, err := coll.InsertMany(context.TODO(), docs, opts)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("inserted documents with IDs %v\n", res.InsertedIDs)
}

func ExampleCollection_InsertOne() {
	var coll *mongo.Collection

	// insert the document {name: "Alice"}
	// set the BypassDocumentValidation option to opt-out of document-level validation
	opts := options.InsertOne().SetBypassDocumentValidation(true)
	res, err := coll.InsertOne(context.TODO(), bson.D{{"name", "Alice"}}, opts)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("inserted document with ID %v\n", res.InsertedID)
}

func ExampleCollection_ReplaceOne() {
	var coll *mongo.Collection

	// replace the document in which the "name" field is "Bob" with {name: "Alice"}
	// specify the Upsert option to insert {name: "Alice"} if the {name: "Bob"} document isn't found
	opts := options.Replace().SetUpsert(true)
	filter := bson.D{{"name", "Bob"}}
	replacement := bson.D{{"name", "Alice"}}
	result, err := coll.ReplaceOne(context.TODO(), filter, replacement, opts)
	if err != nil {
		log.Fatal(err)
	}

	if result.MatchedCount != 0 {
		fmt.Println("matched and replaced an existing document")
		return
	}
	if result.UpsertedCount != 0 {
		fmt.Printf("inserted a new document with ID %v\n", result.UpsertedID)
	}
}

func ExampleCollection_UpdateMany() {
	var coll *mongo.Collection

	// find all documents in which the "name" field is "Bob" and set the name to "Alice"
	// specify the Upsert option to insert {name: "Alice"} if the {name: "Bob"} document isn't found
	opts := options.Update().SetUpsert(true)
	filter := bson.D{{"name", "Bob"}}
	update := bson.D{{"$set", bson.D{{"name", "Alice"}}}}

	result, err := coll.UpdateOne(context.TODO(), filter, update, opts)
	if err != nil {
		log.Fatal(err)
	}

	if result.MatchedCount != 0 {
		fmt.Println("matched and replaced an existing document")
		return
	}
	if result.UpsertedCount != 0 {
		fmt.Printf("inserted a new document with ID %v\n", result.UpsertedID)
	}
}

func ExampleCollection_UpdateOne() {
	var coll *mongo.Collection

	// find the document in which the "name" field is "Bob" and set the name to "Alice"
	// specify the Upsert option to insert {name: "Alice"} if the {name: "Bob"} document isn't found
	opts := options.Update().SetUpsert(true)
	filter := bson.D{{"name", "Bob"}}
	update := bson.D{{"$set", bson.D{{"name", "Alice"}}}}

	result, err := coll.UpdateOne(context.TODO(), filter, update, opts)
	if err != nil {
		log.Fatal(err)
	}

	if result.MatchedCount != 0 {
		fmt.Println("matched and replaced an existing document")
		return
	}
	if result.UpsertedCount != 0 {
		fmt.Printf("inserted a new document with ID %v\n", result.UpsertedID)
	}
}

func ExampleCollection_Watch() {
	var collection *mongo.Collection

	// specify a pipeline that will only match "insert" events
	// specify the MaxAwaitTimeOption to have each attempt wait two seconds for new documents
	matchStage := bson.D{{"$match", bson.D{{"operationType", "insert"}}}}
	opts := options.ChangeStream().SetMaxAwaitTime(2 * time.Second)
	changeStream, err := collection.Watch(context.TODO(), mongo.Pipeline{matchStage}, opts)
	if err != nil {
		log.Fatal(err)
	}

	// print out all change stream events in the order they're received
	// see the mongo.ChangeStream documentation for more examples of using change streams
	for changeStream.Next(context.TODO()) {
		fmt.Println(changeStream.Current)
	}
}
