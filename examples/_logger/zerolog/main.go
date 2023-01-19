// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build zerolog

package main

import (
	"context"
	"log"
	"os"

	"github.com/go-logr/zerologr"
	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	logger := zerolog.New(os.Stderr).With().Caller().Timestamp().Logger()
	sink := zerologr.New(&logger).GetSink()

	// Create a client with our logger options.
	loggerOptions := options.
		Logger().
		SetSink(sink).
		SetMaxDocumentLength(25).
		SetComponentLevel(options.LogComponentCommand, options.LogLevelDebug)

	clientOptions := options.
		Client().
		ApplyURI("mongodb://localhost:27017").
		SetLoggerOptions(loggerOptions)

	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatalf("error connecting to MongoDB: %v", err)
	}

	defer client.Disconnect(context.TODO())

	// Make a database request to test our logging solution
	coll := client.Database("test").Collection("test")

	_, err = coll.InsertOne(context.TODO(), bson.D{{"Alice", "123"}})
	if err != nil {
		log.Fatalf("InsertOne failed: %v", err)
	}
}
