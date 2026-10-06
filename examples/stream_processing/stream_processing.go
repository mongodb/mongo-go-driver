// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Example use of the streamprocessing client. Set MONGODB_STREAM_PROCESSING_URI
// to a workspace endpoint (mongodb://atlas-stream-...) and run with:
//
//	MONGODB_STREAM_PROCESSING_URI=... go run ./examples/stream_processing
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/streamprocessing"
)

func main() {
	uri := os.Getenv("MONGODB_STREAM_PROCESSING_URI")
	if uri == "" {
		log.Fatal("set MONGODB_STREAM_PROCESSING_URI to a workspace endpoint")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	client, err := streamprocessing.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Disconnect(context.Background())

	sps := client.StreamProcessors()
	name := fmt.Sprintf("example-%d", time.Now().Unix())

	pipeline := []bson.D{
		{{Key: "$source", Value: bson.D{{Key: "connectionName", Value: "sample_stream_solar"}}}},
	}

	if err := sps.Create(ctx, name, pipeline); err != nil {
		log.Fatalf("create: %v", err)
	}
	defer func() {
		_ = sps.Get(name).Drop(context.Background())
	}()

	sp := sps.Get(name)
	if err := sp.Start(ctx, nil); err != nil {
		log.Fatalf("start: %v", err)
	}

	time.Sleep(2 * time.Second)

	samples, err := sp.GetStreamProcessorSamples(ctx, options.GetStreamProcessorSamples().SetLimit(5))
	if err != nil {
		log.Fatalf("sample: %v", err)
	}
	fmt.Printf("got %d sample document(s); cursor=%d\n", len(samples.Documents), samples.CursorID)

	if err := sp.Stop(ctx); err != nil {
		log.Fatalf("stop: %v", err)
	}
}
