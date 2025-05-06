// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/connstring"
)

func main() {
	uri := os.Getenv("MONGODB_URI")
	compressor := os.Getenv("MONGO_GO_DRIVER_COMPRESSOR")

	client, err := mongo.Connect(
		options.Client().ApplyURI(uri).SetCompressors([]string{compressor}))
	if err != nil {
		log.Panicf("Error connecting client: %v", err)
	}

	// Use the defaultauthdb (i.e. the database name after the "/") specified in the connection
	// string to run the count operation.
	cs, err := connstring.Parse(uri)
	if err != nil {
		log.Panicf("Error parsing connection string: %v", err)
	}
	if cs.Database == "" {
		log.Panic("Connection string must contain a defaultauthdb.")
	}

	coll := client.Database(cs.Database).Collection("test")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	count, err := coll.EstimatedDocumentCount(ctx)
	if err != nil {
		log.Panicf("failed executing count command: %v", err)
	}
	log.Println("Count of test collection:", count)
}
