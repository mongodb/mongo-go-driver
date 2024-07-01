// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

// This file exists to allow the build scripts (and standard Go builds for some early Go versions)
// to succeed. Without it, the build may encounter an error like:
//
//   go build go.mongodb.org/mongo-driver/mongo/integration: build constraints exclude all Go files in ./go.mongodb.org/mongo-driver/mongo/integration
//

import (
	"context"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func main() {
	opts := options.Client().ApplyURI("mongodb://localhost:27017")

	client, err := mongo.Connect(context.Background(), opts)
	if err != nil {
		log.Fatalf("failed to create connection: %v", err)
	}

	defer client.Disconnect(context.Background())

	database := client.Database("appdb")
	uc := database.Collection("users")

	indexModel := mongo.IndexModel{
		Keys:    bson.M{"username": 1},
		Options: options.Index().SetUnique(true).SetName("myidx"),
	}

	_, err = uc.Indexes().CreateOne(context.TODO(), indexModel)
	if err != nil {
		log.Fatal(err)
	}

	print("hi")
	// Drop index if it exists
	_, err = uc.Indexes().DropKeyOne(context.Background(), bsoncore.Document(bson.Raw(bsoncore.NewDocumentBuilder().AppendInt32("username", 1).Build())))
	if err != nil {
		log.Fatalf("failed to drop index: %v", err)
	}
}
