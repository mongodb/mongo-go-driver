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

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	uri := os.Getenv("MONGODB_URI")
	compressor := os.Getenv("MONGO_GO_DRIVER_COMPRESSOR")

	client, err := mongo.Connect(
		context.Background(),
		options.Client().ApplyURI(uri).SetCompressors([]string{compressor}))
	if err != nil {
		log.Fatal(err)
	}

	coll := client.Database("test").Collection("test")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	count, err := coll.EstimatedDocumentCount(ctx)
	if err != nil {
		log.Fatalf("failed executing count command: %v", err)
	}
	log.Println("Count of test.test:", count)
}
