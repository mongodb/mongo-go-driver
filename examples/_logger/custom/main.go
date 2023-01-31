// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build customlog

package main

import (
	"context"
	"fmt"
	"io"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CustomLogger struct{ io.Writer }

func (logger CustomLogger) Info(level int, msg string, keysAndValues ...interface{}) {
	logger.Write([]byte(fmt.Sprintf("level=%d msg=%s keysAndValues=%v", level, msg, keysAndValues)))
}

func (logger CustomLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	logger.Write([]byte(fmt.Sprintf("err=%v msg=%s keysAndValues=%v", err, msg, keysAndValues)))
}

func main() {
	//sink := CustomLogger{os.Stdout}

	// Create a client with our logger options.
	loggerOptions := options.
		Logger().
		SetSink(sink).
		SetMaxDocumentLength(25).
		SetComponentLevel(options.LogComponentCommand, options.LogLevelDebug)

	clientOptions := options.
		Client().
		ApplyURI("mongodb://localhost:27017").
		SetMinPoolSize(1).
		SetMaxPoolSize(5).
		SetMaxConnIdleTime(10_000)

	client, err := mongo.Connect(context.TODO(), clientOptions)

	if err != nil {
		log.Fatalf("error connecting to MongoDB: %v", err)
	}

	defer client.Disconnect(context.TODO())

	// Make a database request to test our logging solution.
	coll := client.Database("test").Collection("test")

	_, err = coll.InsertOne(context.TODO(), bson.D{{"Alice", "123"}})
	if err != nil {
		log.Fatalf("InsertOne failed: %v", err)
	}
}
