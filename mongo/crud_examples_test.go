// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// Client examples

func ExampleClient_ListDatabaseNames() {
	var client *mongo.Client

	// Use a filter to only select non-empty databases.
	result, err := client.ListDatabaseNames(
		context.TODO(),
		bson.D{{"empty", false}})
	if err != nil {
		log.Panic(err)
	}

	for _, db := range result {
		fmt.Println(db)
	}
}

func ExampleClient_Watch() {
	var client *mongo.Client

	// Specify a pipeline that will only match "insert" events.
	// Specify the MaxAwaitTimeOption to have each attempt wait two seconds for
	// new documents.
	matchStage := bson.D{{"$match", bson.D{{"operationType", "insert"}}}}
	opts := options.ChangeStream().SetMaxAwaitTime(2 * time.Second)
	changeStream, err := client.Watch(
		context.TODO(),
		mongo.Pipeline{matchStage},
		opts)
	if err != nil {
		log.Panic(err)
	}

	// Print out all change stream events in the order they're received.
	// See the mongo.ChangeStream documentation for more examples of using
	// change streams.
	for changeStream.Next(context.TODO()) {
		fmt.Println(changeStream.Current)
	}
}

// Database examples

func ExampleDatabase_CreateCollection() {
	var db *mongo.Database

	// Create a "users" collection with a JSON schema validator. The validator
	// will ensure that each document in the collection has "name" and "age"
	// fields.
	jsonSchema := bson.M{
		"bsonType": "object",
		"required": []string{"name", "age"},
		"properties": bson.M{
			"name": bson.M{
				"bsonType": "string",
				"description": "the name of the user, which is required and " +
					"must be a string",
			},
			"age": bson.M{
				"bsonType": "int",
				"minimum":  18,
				"description": "the age of the user, which is required and " +
					"must be an integer >= 18",
			},
		},
	}
	validator := bson.M{
		"$jsonSchema": jsonSchema,
	}
	opts := options.CreateCollection().SetValidator(validator)

	err := db.CreateCollection(context.TODO(), "users", opts)
	if err != nil {
		log.Panic(err)
	}
}

func ExampleDatabase_CreateView() {
	var db *mongo.Database

	// Create a view on the "users" collection called "usernames". Specify a
	// pipeline that concatenates the "firstName" and "lastName" fields from
	// each document in "users" and projects the result into the "fullName"
	// field in the view.
	projectStage := bson.D{
		{"$project", bson.D{
			{"_id", 0},
			{"fullName", bson.D{
				{"$concat", []string{"$firstName", " ", "$lastName"}},
			}},
		}},
	}
	pipeline := mongo.Pipeline{projectStage}

	// Specify the Collation option to set a default collation for the view.
	opts := options.CreateView().SetCollation(&options.Collation{
		Locale: "en_US",
	})

	err := db.CreateView(context.TODO(), "usernames", "users", pipeline, opts)
	if err != nil {
		log.Panic(err)
	}
}

func ExampleDatabase_ListCollectionNames() {
	var db *mongo.Database

	// Use a filter to only select capped collections.
	result, err := db.ListCollectionNames(
		context.TODO(),
		bson.D{{"options.capped", true}})
	if err != nil {
		log.Panic(err)
	}

	for _, coll := range result {
		fmt.Println(coll)
	}
}

func ExampleDatabase_RunCommand() {
	var db *mongo.Database

	// Run an explain command to see the query plan for when a "find" is
	// executed on collection "bar" specify the ReadPreference option to
	// explicitly set the read preference to primary.
	findCmd := bson.D{{"find", "bar"}}
	command := bson.D{{"explain", findCmd}}
	opts := options.RunCmd().SetReadPreference(readpref.Primary())
	var result bson.M
	err := db.RunCommand(context.TODO(), command, opts).Decode(&result)
	if err != nil {
		log.Panic(err)
	}
	fmt.Println(result)
}

func ExampleDatabase_Watch() {
	var db *mongo.Database

	// Specify a pipeline that will only match "insert" events.
	// Specify the MaxAwaitTimeOption to have each attempt wait two seconds for
	// new documents.
	matchStage := bson.D{{"$match", bson.D{{"operationType", "insert"}}}}
	opts := options.ChangeStream().SetMaxAwaitTime(2 * time.Second)
	changeStream, err := db.Watch(
		context.TODO(),
		mongo.Pipeline{matchStage},
		opts)
	if err != nil {
		log.Panic(err)
	}

	// Print out all change stream events in the order they're received.
	// See the mongo.ChangeStream documentation for more examples of using
	// change streams
	for changeStream.Next(context.TODO()) {
		fmt.Println(changeStream.Current)
	}
}

// Collection examples

func ExampleCollection_Aggregate() {
	var coll *mongo.Collection

	// Specify a pipeline that will return the number of times each name appears
	// in the collection.
	// Specify the MaxTime option to limit the amount of time the operation can
	// run on the server.
	groupStage := bson.D{
		{"$group", bson.D{
			{"_id", "$name"},
			{"numTimes", bson.D{
				{"$sum", 1},
			}},
		}},
	}
	opts := options.Aggregate()
	cursor, err := coll.Aggregate(
		context.TODO(),
		mongo.Pipeline{groupStage},
		opts)
	if err != nil {
		log.Panic(err)
	}

	// Get a list of all returned documents and print them out.
	// See the mongo.Cursor documentation for more examples of using cursors.
	var results []bson.M
	if err = cursor.All(context.TODO(), &results); err != nil {
		log.Panic(err)
	}
	for _, result := range results {
		fmt.Printf(
			"name %v appears %v times\n",
			result["_id"],
			result["numTimes"])
	}
}

func ExampleCollection_BulkWrite() {
	var coll *mongo.Collection
	var firstID, secondID bson.ObjectID

	// Update the "email" field for two users.
	// For each update, specify the Upsert option to insert a new document if a
	// document matching the filter isn't found.
	// Set the Ordered option to false to allow both operations to happen even
	// if one of them errors.
	firstUpdate := bson.D{
		{"$set", bson.D{
			{"email", "firstEmail@example.com"},
		}},
	}
	secondUpdate := bson.D{
		{"$set", bson.D{
			{"email", "secondEmail@example.com"},
		}},
	}
	models := []mongo.WriteModel{
		mongo.NewUpdateOneModel().SetFilter(bson.D{{"_id", firstID}}).
			SetUpdate(firstUpdate).SetUpsert(true),
		mongo.NewUpdateOneModel().SetFilter(bson.D{{"_id", secondID}}).
			SetUpdate(secondUpdate).SetUpsert(true),
	}
	opts := options.BulkWrite().SetOrdered(false)
	res, err := coll.BulkWrite(context.TODO(), models, opts)
	if err != nil {
		log.Panic(err)
	}

	fmt.Printf(
		"inserted %v and deleted %v documents\n",
		res.InsertedCount,
		res.DeletedCount)
}

func ExampleCollection_CountDocuments() {
	var coll *mongo.Collection

	// Specify a timeout to limit the amount of time the operation can run on
	// the server.
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	// Count the number of times the name "Bob" appears in the collection.
	count, err := coll.CountDocuments(ctx, bson.D{{"name", "Bob"}}, nil)
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("name Bob appears in %v documents", count)
}

func ExampleCollection_DeleteMany() {
	var coll *mongo.Collection

	// Delete all documents in which the "name" field is "Bob" or "bob".
	// Specify the Collation option to provide a collation that will ignore case
	// for string comparisons.
	opts := options.DeleteMany().SetCollation(&options.Collation{
		Locale:    "en_US",
		Strength:  1,
		CaseLevel: false,
	})
	res, err := coll.DeleteMany(context.TODO(), bson.D{{"name", "bob"}}, opts)
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("deleted %v documents\n", res.DeletedCount)
}

func ExampleCollection_DeleteOne() {
	var coll *mongo.Collection

	// Delete at most one document in which the "name" field is "Bob" or "bob".
	// Specify the SetCollation option to provide a collation that will ignore
	// case for string comparisons.
	opts := options.DeleteOne().SetCollation(&options.Collation{
		Locale:    "en_US",
		Strength:  1,
		CaseLevel: false,
	})
	res, err := coll.DeleteOne(context.TODO(), bson.D{{"name", "bob"}}, opts)
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("deleted %v documents\n", res.DeletedCount)
}

func ExampleCollection_Distinct() {
	var coll *mongo.Collection

	// Specify a timeout to limit the amount of time the operation can run on
	// the server.
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	// Find all unique values for the "name" field for documents in which the
	// "age" field is greater than 25.
	filter := bson.D{{"age", bson.D{{"$gt", 25}}}}
	res := coll.Distinct(ctx, "name", filter)
	if err := res.Err(); err != nil {
		log.Panic(err)
	}

	values, err := res.Raw()
	if err != nil {
		log.Panic(err)
	}

	for _, value := range values {
		fmt.Println(value)
	}
}

func ExampleCollection_EstimatedDocumentCount() {
	var coll *mongo.Collection

	// Specify a timeout to limit the amount of time the operation can run on
	// the server.
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	// Get and print an estimated of the number of documents in the collection.
	count, err := coll.EstimatedDocumentCount(ctx, nil)
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("estimated document count: %v", count)
}

func ExampleCollection_Find() {
	var coll *mongo.Collection

	// Find all documents in which the "name" field is "Bob".
	// Specify the Sort option to sort the returned documents by age in
	// ascending order.
	opts := options.Find().SetSort(bson.D{{"age", 1}})
	cursor, err := coll.Find(context.TODO(), bson.D{{"name", "Bob"}}, opts)
	if err != nil {
		log.Panic(err)
	}

	// Get a list of all returned documents and print them out.
	// See the mongo.Cursor documentation for more examples of using cursors.
	var results []bson.M
	if err = cursor.All(context.TODO(), &results); err != nil {
		log.Panic(err)
	}
	for _, result := range results {
		fmt.Println(result)
	}
}

func ExampleCollection_FindOne() {
	var coll *mongo.Collection
	var id bson.ObjectID

	// Find the document for which the _id field matches id.
	// Specify the Sort option to sort the documents by age.
	// The first document in the sorted order will be returned.
	opts := options.FindOne().SetSort(bson.D{{"age", 1}})
	var result bson.M
	err := coll.FindOne(
		context.TODO(),
		bson.D{{"_id", id}},
		opts,
	).Decode(&result)
	if err != nil {
		// ErrNoDocuments means that the filter did not match any documents in
		// the collection.
		if errors.Is(err, mongo.ErrNoDocuments) {
			return
		}
		log.Panic(err)
	}
	fmt.Printf("found document %v", result)
}

func ExampleCollection_FindOneAndDelete() {
	var coll *mongo.Collection
	var id bson.ObjectID

	// Find and delete the document for which the _id field matches id.
	// Specify the Projection option to only include the name and age fields in
	// the returned document.
	opts := options.FindOneAndDelete().
		SetProjection(bson.D{{"name", 1}, {"age", 1}})
	var deletedDocument bson.M
	err := coll.FindOneAndDelete(
		context.TODO(),
		bson.D{{"_id", id}},
		opts,
	).Decode(&deletedDocument)
	if err != nil {
		// ErrNoDocuments means that the filter did not match any documents in
		// the collection.
		if errors.Is(err, mongo.ErrNoDocuments) {
			return
		}
		log.Panic(err)
	}
	fmt.Printf("deleted document %v", deletedDocument)
}

func ExampleCollection_FindOneAndReplace() {
	var coll *mongo.Collection
	var id bson.ObjectID

	// Find the document for which the _id field matches id and add a field
	// called "location".
	// Specify the Upsert option to insert a new document if a document matching
	// the filter isn't found.
	opts := options.FindOneAndReplace().SetUpsert(true)
	filter := bson.D{{"_id", id}}
	replacement := bson.D{{"location", "NYC"}}
	var replacedDocument bson.M
	err := coll.FindOneAndReplace(
		context.TODO(),
		filter,
		replacement,
		opts,
	).Decode(&replacedDocument)
	if err != nil {
		// ErrNoDocuments means that the filter did not match any documents in
		// the collection.
		if errors.Is(err, mongo.ErrNoDocuments) {
			return
		}
		log.Panic(err)
	}
	fmt.Printf("replaced document %v", replacedDocument)
}

func ExampleCollection_FindOneAndUpdate() {
	var coll *mongo.Collection
	var id bson.ObjectID

	// Find the document for which the _id field matches id and set the email to
	// "newemail@example.com".
	// Specify the Upsert option to insert a new document if a document matching
	// the filter isn't found.
	opts := options.FindOneAndUpdate().SetUpsert(true)
	filter := bson.D{{"_id", id}}
	update := bson.D{{"$set", bson.D{{"email", "newemail@example.com"}}}}
	var updatedDocument bson.M
	err := coll.FindOneAndUpdate(
		context.TODO(),
		filter,
		update,
		opts,
	).Decode(&updatedDocument)
	if err != nil {
		// ErrNoDocuments means that the filter did not match any documents in
		// the collection.
		if errors.Is(err, mongo.ErrNoDocuments) {
			return
		}
		log.Panic(err)
	}
	fmt.Printf("updated document %v", updatedDocument)
}

func ExampleCollection_InsertMany() {
	var coll *mongo.Collection

	// Insert documents {name: "Alice"} and {name: "Bob"}.
	// Set the Ordered option to false to allow both operations to happen even
	// if one of them errors.
	docs := []interface{}{
		bson.D{{"name", "Alice"}},
		bson.D{{"name", "Bob"}},
	}
	opts := options.InsertMany().SetOrdered(false)
	res, err := coll.InsertMany(context.TODO(), docs, opts)
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("inserted documents with IDs %v\n", res.InsertedIDs)
}

func ExampleCollection_InsertOne() {
	var coll *mongo.Collection

	// Insert the document {name: "Alice"}.
	res, err := coll.InsertOne(context.TODO(), bson.D{{"name", "Alice"}})
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("inserted document with ID %v\n", res.InsertedID)
}

func ExampleCollection_ReplaceOne() {
	var coll *mongo.Collection
	var id bson.ObjectID

	// Find the document for which the _id field matches id and add a field
	// called "location".
	// Specify the Upsert option to insert a new document if a document matching
	// the filter isn't found.
	opts := options.Replace().SetUpsert(true)
	filter := bson.D{{"_id", id}}
	replacement := bson.D{{"location", "NYC"}}
	result, err := coll.ReplaceOne(context.TODO(), filter, replacement, opts)
	if err != nil {
		log.Panic(err)
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

	// Increment the age for all users whose birthday is today.
	today := time.Now().Format("01-01-1970")
	filter := bson.D{{"birthday", today}}
	update := bson.D{{"$inc", bson.D{{"age", 1}}}}

	result, err := coll.UpdateMany(context.TODO(), filter, update)
	if err != nil {
		log.Panic(err)
	}

	if result.MatchedCount != 0 {
		fmt.Println("matched and replaced an existing document")
		return
	}
}

func ExampleCollection_UpdateOne() {
	var coll *mongo.Collection
	var id bson.ObjectID

	// Find the document for which the _id field matches id and set the email to
	// "newemail@example.com".
	// Specify the Upsert option to insert a new document if a document matching
	// the filter isn't found.
	opts := options.UpdateOne().SetUpsert(true)
	filter := bson.D{{"_id", id}}
	update := bson.D{{"$set", bson.D{{"email", "newemail@example.com"}}}}

	result, err := coll.UpdateOne(context.TODO(), filter, update, opts)
	if err != nil {
		log.Panic(err)
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

	// Specify a pipeline that will only match "insert" events.
	// Specify the MaxAwaitTimeOption to have each attempt wait two seconds for
	// new documents.
	matchStage := bson.D{{"$match", bson.D{{"operationType", "insert"}}}}
	opts := options.ChangeStream().SetMaxAwaitTime(2 * time.Second)
	changeStream, err := collection.Watch(
		context.TODO(),
		mongo.Pipeline{matchStage},
		opts)
	if err != nil {
		log.Panic(err)
	}

	// Print out all change stream events in the order they're received.
	// See the mongo.ChangeStream documentation for more examples of using
	// change streams.
	for changeStream.Next(context.TODO()) {
		fmt.Println(changeStream.Current)
	}
}

// Session examples

func ExampleWithSession() {
	// Assume client is configured with write concern majority and read
	// preference primary.
	var client *mongo.Client

	// Specify the DefaultReadConcern option so any transactions started through
	// the session will have read concern majority.
	// The DefaultReadPreference and DefaultWriteConcern options aren't
	// specified so they will be inheritied from client and be set to primary
	// and majority, respectively.
	txnOpts := options.Transaction().SetReadConcern(readconcern.Majority())
	opts := options.Session().SetDefaultTransactionOptions(txnOpts)
	sess, err := client.StartSession(opts)
	if err != nil {
		log.Panic(err)
	}
	defer sess.EndSession(context.TODO())

	// Call WithSession to start a transaction within the new session.
	err = mongo.WithSession(
		context.TODO(),
		sess,
		func(ctx context.Context) error {
			// Use the context.Context as the Context parameter for
			// InsertOne and FindOne so both operations are run under the new
			// Session.

			if err := sess.StartTransaction(); err != nil {
				return err
			}

			coll := client.Database("db").Collection("coll")
			res, err := coll.InsertOne(ctx, bson.D{{"x", 1}})
			if err != nil {
				// Abort the transaction after an error. Use
				// context.Background() to ensure that the abort can complete
				// successfully even if the context passed to mongo.WithSession
				// is changed to have a timeout.
				_ = sess.AbortTransaction(context.Background())
				return err
			}

			var result bson.M
			err = coll.FindOne(
				ctx,
				bson.D{{"_id", res.InsertedID}},
			).Decode(result)
			if err != nil {
				// Abort the transaction after an error. Use
				// context.Background() to ensure that the abort can complete
				// successfully even if the context passed to mongo.WithSession
				// is changed to have a timeout.
				_ = sess.AbortTransaction(context.Background())
				return err
			}
			fmt.Println(result)

			// Use context.Background() to ensure that the commit can complete
			// successfully even if the context passed to mongo.WithSession is
			// changed to have a timeout.
			return sess.CommitTransaction(context.Background())
		})
	if err != nil {
		log.Panic(err)
	}
}

func ExampleClient_UseSessionWithOptions() {
	var client *mongo.Client

	// Specify the DefaultReadConcern option so any transactions started through
	// the session will have read concern majority.
	// The DefaultReadPreference and DefaultWriteConcern options aren't
	// specified so they will be inheritied from client and be set to primary
	// and majority, respectively.
	txnOpts := options.Transaction().SetReadConcern(readconcern.Majority())
	opts := options.Session().SetDefaultTransactionOptions(txnOpts)
	err := client.UseSessionWithOptions(
		context.TODO(),
		opts,
		func(ctx context.Context) error {
			sess := mongo.SessionFromContext(ctx)
			// Use the context.Context as the Context parameter for
			// InsertOne and FindOne so both operations are run under the new
			// Session.

			if err := sess.StartTransaction(); err != nil {
				return err
			}

			coll := client.Database("db").Collection("coll")
			res, err := coll.InsertOne(ctx, bson.D{{"x", 1}})
			if err != nil {
				// Abort the transaction after an error. Use
				// context.Background() to ensure that the abort can complete
				// successfully even if the context passed to mongo.WithSession
				// is changed to have a timeout.
				_ = sess.AbortTransaction(context.Background())
				return err
			}

			var result bson.M
			err = coll.FindOne(
				ctx,
				bson.D{{"_id", res.InsertedID}},
			).Decode(result)
			if err != nil {
				// Abort the transaction after an error. Use
				// context.Background() to ensure that the abort can complete
				// successfully even if the context passed to mongo.WithSession
				// is changed to have a timeout.
				_ = sess.AbortTransaction(context.Background())
				return err
			}
			fmt.Println(result)

			// Use context.Background() to ensure that the commit can complete
			// successfully even if the context passed to mongo.WithSession is
			// changed to have a timeout.
			return sess.CommitTransaction(context.Background())
		})
	if err != nil {
		log.Panic(err)
	}
}

func ExampleClient_StartSession_withTransaction() {
	// Assume client is configured with write concern majority and read
	// preference primary.
	var client *mongo.Client

	// Specify the DefaultReadConcern option so any transactions started through
	// the session will have read concern majority.
	// The DefaultReadPreference and DefaultWriteConcern options aren't
	// specified so they will be inheritied from client and be set to primary
	// and majority, respectively.
	txnOpts := options.Transaction().SetReadConcern(readconcern.Majority())
	opts := options.Session().SetDefaultTransactionOptions(txnOpts)
	sess, err := client.StartSession(opts)
	if err != nil {
		log.Panic(err)
	}
	defer sess.EndSession(context.TODO())

	// Specify the ReadPreference option to set the read preference to primary
	// preferred for this transaction.
	txnOpts.SetReadPreference(readpref.PrimaryPreferred())
	result, err := sess.WithTransaction(
		context.TODO(),
		func(ctx context.Context) (interface{}, error) {
			// Use the context.Context as the Context parameter for
			// InsertOne and FindOne so both operations are run in the same
			// transaction.

			coll := client.Database("db").Collection("coll")
			res, err := coll.InsertOne(ctx, bson.D{{"x", 1}})
			if err != nil {
				return nil, err
			}

			var result bson.M
			err = coll.FindOne(
				ctx,
				bson.D{{"_id", res.InsertedID}},
			).Decode(result)
			if err != nil {
				return nil, err
			}
			return result, err
		},
		txnOpts)
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("result: %v\n", result)
}

func ExampleNewSessionContext() {
	var client *mongo.Client

	// Create a new Session and SessionContext.
	sess, err := client.StartSession()
	if err != nil {
		panic(err)
	}
	defer sess.EndSession(context.TODO())
	ctx := mongo.NewSessionContext(context.TODO(), sess)

	// Start a transaction and use the context.Context as the Context
	// parameter for InsertOne and FindOne so both operations are run in the
	// transaction.
	if err = sess.StartTransaction(); err != nil {
		panic(err)
	}

	coll := client.Database("db").Collection("coll")
	res, err := coll.InsertOne(ctx, bson.D{{"x", 1}})
	if err != nil {
		// Abort the transaction after an error. Use context.Background() to
		// ensure that the abort can complete successfully even if the context
		// passed to NewSessionContext is changed to have a timeout.
		_ = sess.AbortTransaction(context.Background())
		panic(err)
	}

	var result bson.M
	err = coll.FindOne(
		ctx,
		bson.D{{"_id", res.InsertedID}},
	).Decode(&result)
	if err != nil {
		// Abort the transaction after an error. Use context.Background() to
		// ensure that the abort can complete successfully even if the context
		// passed to NewSessionContext is changed to have a timeout.
		_ = sess.AbortTransaction(context.Background())
		panic(err)
	}
	fmt.Printf("result: %v\n", result)

	// Commit the transaction so the inserted document will be stored. Use
	// context.Background() to ensure that the commit can complete successfully
	// even if the context passed to NewSessionContext is changed to have a
	// timeout.
	if err = sess.CommitTransaction(context.Background()); err != nil {
		panic(err)
	}
}

// Cursor examples

func ExampleCursor_All() {
	var cursor *mongo.Cursor

	var results []bson.M
	if err := cursor.All(context.TODO(), &results); err != nil {
		log.Panic(err)
	}
	fmt.Println(results)
}

func ExampleCursor_Next() {
	var cursor *mongo.Cursor
	defer cursor.Close(context.TODO())

	// Iterate the cursor and print out each document until the cursor is
	// exhausted or there is an error getting the next document.
	for cursor.Next(context.TODO()) {
		// A new result variable should be declared for each document.
		var result bson.M
		if err := cursor.Decode(&result); err != nil {
			log.Panic(err)
		}
		fmt.Println(result)
	}
	if err := cursor.Err(); err != nil {
		log.Panic(err)
	}
}

func ExampleCursor_TryNext() {
	var cursor *mongo.Cursor
	defer cursor.Close(context.TODO())

	// Iterate the cursor and print out each document until the cursor is
	// exhausted or there is an error getting the next document.
	for {
		if cursor.TryNext(context.TODO()) {
			// A new result variable should be declared for each document.
			var result bson.M
			if err := cursor.Decode(&result); err != nil {
				log.Panic(err)
			}
			fmt.Println(result)
			continue
		}

		// If TryNext returns false, the next document is not yet available, the
		// cursor was exhausted and was closed, or an error occurred. TryNext
		// should only be called again for the empty batch case.
		if err := cursor.Err(); err != nil {
			log.Panic(err)
		}
		if cursor.ID() == 0 {
			break
		}
	}
}

func ExampleCursor_RemainingBatchLength() {
	// Because we're using a tailable cursor, this must be a handle to a capped
	// collection.
	var coll *mongo.Collection

	// Create a tailable await cursor. Specify the MaxAwaitTime option so
	// requests to get more data will return if there are no documents available
	// after two seconds.
	findOpts := options.Find().
		SetCursorType(options.TailableAwait).
		SetMaxAwaitTime(2 * time.Second)
	cursor, err := coll.Find(context.TODO(), bson.D{}, findOpts)
	if err != nil {
		panic(err)
	}

	for {
		// Iterate the cursor using TryNext.
		if cursor.TryNext(context.TODO()) {
			fmt.Println(cursor.Current)
		}

		// Handle cursor errors or the cursor being closed by the server.
		if err = cursor.Err(); err != nil {
			panic(err)
		}
		if cursor.ID() == 0 {
			panic("cursor was unexpectedly closed by the server")
		}

		// Use the RemainingBatchLength function to rate-limit the number of
		// network requests the driver does. If the current batch is empty,
		// sleep for a short amount of time to let documents build up on the
		// server before the next TryNext call, which will do a network request.
		if cursor.RemainingBatchLength() == 0 {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// ChangeStream examples

func ExampleChangeStream_Next() {
	var stream *mongo.ChangeStream
	defer stream.Close(context.TODO())

	// Iterate the change stream and print out each event.
	// Because the Next call blocks until an event is available, another way to
	// iterate the change stream is to call Next in a goroutine and pass in a
	// context that can be cancelled to abort the call.

	for stream.Next(context.TODO()) {
		// A new event variable should be declared for each event.
		var event bson.M
		if err := stream.Decode(&event); err != nil {
			log.Panic(err)
		}
		fmt.Println(event)
	}
	if err := stream.Err(); err != nil {
		log.Panic(err)
	}
}

func ExampleChangeStream_TryNext() {
	var stream *mongo.ChangeStream
	defer stream.Close(context.TODO())

	// Iterate the change stream and print out each event until the change
	// stream is closed by the server or there is an error getting the next
	// event.
	for {
		if stream.TryNext(context.TODO()) {
			// A new event variable should be declared for each event.
			var event bson.M
			if err := stream.Decode(&event); err != nil {
				log.Panic(err)
			}
			fmt.Println(event)
			continue
		}

		// If TryNext returns false, the next change is not yet available, the
		// change stream was closed by the server, or an error occurred. TryNext
		// should only be called again for the empty batch case.
		if err := stream.Err(); err != nil {
			log.Panic(err)
		}
		if stream.ID() == 0 {
			break
		}
	}
}

func ExampleChangeStream_ResumeToken() {
	var client *mongo.Client

	// Assume stream was created via client.Watch()
	var stream *mongo.ChangeStream

	cancelCtx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	var wg sync.WaitGroup
	wg.Add(1)

	// Run a goroutine to process events.
	go func() {
		for stream.Next(cancelCtx) {
			fmt.Println(stream.Current)
		}
		wg.Done()
	}()

	// Assume client needs to be disconnected. Cancel the context being used by
	// the goroutine to abort any in-progres Next calls and wait for the
	// goroutine to exit.
	cancel()
	wg.Wait()

	// Before disconnecting the client, store the last seen resume token for the
	// change stream.
	resumeToken := stream.ResumeToken()
	_ = client.Disconnect(context.TODO())

	// Once a new client is created, the change stream can be re-created.
	// Specify resumeToken as the ResumeAfter option so only events that
	// occurred after resumeToken will be returned.
	var newClient *mongo.Client
	opts := options.ChangeStream().SetResumeAfter(resumeToken)
	newStream, err := newClient.Watch(context.TODO(), mongo.Pipeline{}, opts)
	if err != nil {
		log.Panic(err)
	}
	defer newStream.Close(context.TODO())
}

// IndexView examples

func ExampleIndexView_CreateMany() {
	var indexView *mongo.IndexView

	// Create two indexes: {name: 1, email: 1} and {name: 1, age: 1}
	// For the first index, specify no options. The name will be generated as
	// "name_1_email_1" by the driver.
	// For the second index, specify the Name option to explicitly set the name
	// to "nameAge".
	models := []mongo.IndexModel{
		{
			Keys: bson.D{{"name", 1}, {"email", 1}},
		},
		{
			Keys:    bson.D{{"name", 1}, {"age", 1}},
			Options: options.Index().SetName("nameAge"),
		},
	}

	// Specify the MaxTime option to limit the amount of time the operation can
	// run on the server
	names, err := indexView.CreateMany(context.TODO(), models, nil)
	if err != nil {
		log.Panic(err)
	}

	fmt.Printf("created indexes %v\n", names)
}

func ExampleIndexView_List() {
	var indexView *mongo.IndexView

	// Specify a timeout to limit the amount of time the operation can run on
	// the server.
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	cursor, err := indexView.List(ctx, nil)
	if err != nil {
		log.Panic(err)
	}

	// Get a slice of all indexes returned and print them out.
	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		log.Panic(err)
	}
	fmt.Println(results)
}

func ExampleCollection_Find_primitiveRegex() {
	ctx := context.TODO()
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")

	// Connect to a mongodb server.
	client, err := mongo.Connect(clientOptions)
	if err != nil {
		panic(err)
	}

	defer func() { _ = client.Disconnect(ctx) }()

	type Pet struct {
		Type string `bson:"type"`
		Name string `bson:"name"`
	}

	// Create a slice of documents to insert. We will lookup a subset of
	// these documents using regex.
	toInsert := []interface{}{
		Pet{Type: "cat", Name: "Mo"},
		Pet{Type: "dog", Name: "Loki"},
	}

	coll := client.Database("test").Collection("test")

	if _, err := coll.InsertMany(ctx, toInsert); err != nil {
		panic(err)
	}

	// Create a filter to find a document with key "name" and any value that
	// starts with letter "m". Use the "i" option to indicate
	// case-insensitivity.
	filter := bson.D{{"name", bson.Regex{Pattern: "^m", Options: "i"}}}

	_, err = coll.Find(ctx, filter)
	if err != nil {
		panic(err)
	}
}

func ExampleCollection_Find_regex() {
	ctx := context.TODO()
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")

	// Connect to a mongodb server.
	client, err := mongo.Connect(clientOptions)
	if err != nil {
		panic(err)
	}

	defer func() { _ = client.Disconnect(ctx) }()

	type Pet struct {
		Type string `bson:"type"`
		Name string `bson:"name"`
	}

	// Create a slice of documents to insert. We will lookup a subset of
	// these documents using regex.
	toInsert := []interface{}{
		Pet{Type: "cat", Name: "Mo"},
		Pet{Type: "dog", Name: "Loki"},
	}

	coll := client.Database("test").Collection("test")

	if _, err := coll.InsertMany(ctx, toInsert); err != nil {
		panic(err)
	}

	// Create a filter to find a document with key "name" and any value that
	// starts with letter "m". Use the "i" option to indicate
	// case-insensitivity.
	filter := bson.D{{"name", bson.D{{"$regex", "^m"}, {"$options", "i"}}}}

	_, err = coll.Find(ctx, filter)
	if err != nil {
		panic(err)
	}
}
