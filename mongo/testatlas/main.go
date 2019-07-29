// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"context"
	"log"

	"flag"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
)

func main() {
	flag.Parse()
	uris := flag.Args()

	for _, uri := range uris {
		ctx := context.Background()
		cs, err := connstring.Parse(uri)
		if err != nil {
			log.Fatal(err)
		}

		client, err := mongo.NewClient(options.Client().ApplyURI(cs.String()))
		if err != nil {
			log.Fatal(err)
		}

		err = client.Connect(ctx)
		if err != nil {
			log.Fatal(err)
		}

		db := client.Database("test")
		coll := db.Collection("test")

		err = db.RunCommand(
			ctx,
			bson.D{{"isMaster", bsonx.Int32(1)}},
		).Err()
		if err != nil {
			log.Fatalf("failed executing isMaster command: %v", err)
		}

		_, err = coll.InsertOne(ctx, bson.D{{"x", bsonx.Int32(1)}})
		if err != nil {
			log.Fatalf("failed executing insertOne command: %v", err)
		}

		res := coll.FindOne(ctx, bsonx.Doc{{"x", bsonx.Int32(1)}})
		if res.Err() != nil {
			log.Fatalf("failed executing findOne command: %v", res.Err())
		}
	}
}
