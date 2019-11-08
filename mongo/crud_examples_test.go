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
)

// Client examples

func ExampleClient_ListDatabases() {
	var client *mongo.Client

	// use a filter to only select the admin database
	// specify the NameOnly option so the returned specificiations only have the name field populated
	opts := options.ListDatabases().SetNameOnly(true)
	result, err := client.ListDatabases(context.TODO(), bson.D{{"name", "admin"}}, opts)
	if err != nil {
		log.Fatal(err)
	}

	for _, db := range result.Databases {
		fmt.Println(db.Name)
	}
}

func ExampleClient_ListDatabaseNames() {
	var client *mongo.Client

	// use a filter to only select the admin database
	result, err := client.ListDatabaseNames(context.TODO(), bson.D{{"name", "admin"}})
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
		fmt.Printf("got event %v\n", changeStream.Current)
	}
}

func ExampleDatabase_Aggregate() {
	var db *mongo.Database

	// specify a pipeline that will list all sessions on the target server
	// specify the MaxTime option to limit the amount of time the query can run on the server
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

	// use a filter to only select the collection "foo"
	result, err := db.ListCollectionNames(context.TODO(), bson.D{{"name", "foo"}})
	if err != nil {
		log.Fatal(err)
	}

	for _, coll := range result {
		fmt.Println(coll)
	}
}

func ExampleDatabase_ListCollections() {
	var db *mongo.Database

	// use a filter to only select the collection "foo"
	// specify the NameOnly option so the returned specificiations only have the name field populated
	opts := options.ListCollections().SetNameOnly(true)
	cursor, err := db.ListCollections(context.TODO(), bson.D{{"name", "foo"}}, opts)
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
