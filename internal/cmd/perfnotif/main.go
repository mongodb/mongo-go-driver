// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func main() {
	uri := os.Getenv("perf_uri_private_endpoint")
	if uri == "" {
		log.Panic("perf_uri_private_endpoint env variable is not set")
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		log.Panicf("Error connecting client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Panicf("Error pinging MongoDB Analytics: %v", err)
	}
	fmt.Println("Successfully connected to MongoDB Analytics node.")

	coll := client.Database("expanded_metrics").Collection("change_points")
	var cursor *mongo.Cursor
	cursor, err = getDocsWithContext(coll)
	if err != nil {
		log.Panicf("Error retrieving documents from collection.")
	}
	fmt.Printf("Successfully retrieved %d documents.", cursor.RemainingBatchLength())

	err = client.Disconnect(context.Background())
	if err != nil {
		log.Panicf("Failed to disconnect client: %v", err)
	}

}

func getDocsWithContext(coll *mongo.Collection) (*mongo.Cursor, error) {
	filter := bson.D{
		{"time_series_info.project", "mongo-go-driver"},
		{"time_series_info.variant", "perf"},
		{"time_series_info.task", "perf"},
		// {"commit", commitName},
		{"triage_contexts", bson.M{"$in": []string{"GoDriver perf (h-score)"}}},
	}

	projection := bson.D{
		{"time_series_info.project", 1},
		{"time_series_info.task", 1},
		{"time_series_info.test", 1},
		{"triage_contexts", 1},
		{"h_score", 1},
	}

	findOptions := options.Find().SetProjection(projection)

	findCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cursor, err := coll.Find(findCtx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(findCtx)

	return cursor, err
}
