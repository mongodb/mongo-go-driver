// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"sync"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CustomLogger struct {
	io.Writer
	mu sync.Mutex
}

func (logger *CustomLogger) Info(level int, msg string, _ ...interface{}) {
	logger.mu.Lock()
	defer logger.mu.Unlock()

	fmt.Fprintf(logger, "level=%d msg=%s\n", level, msg)
}

func (logger *CustomLogger) Error(err error, msg string, _ ...interface{}) {
	logger.mu.Lock()
	defer logger.mu.Unlock()

	fmt.Fprintf(logger, "err=%v msg=%s\n", err, msg)
}

func ExampleClientOptions_SetLoggerOptions_customLogger() {
	buf := bytes.NewBuffer(nil)
	sink := &CustomLogger{Writer: buf}

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

	defer func() { _ = client.Disconnect(context.TODO()) }()

	// Make a database request to test our logging solution.
	coll := client.Database("test").Collection("test")

	_, err = coll.InsertOne(context.TODO(), map[string]string{"foo": "bar"})
	if err != nil {
		log.Fatalf("InsertOne failed: %v", err)
	}

	// Print the logs.
	fmt.Println(buf.String())
}
