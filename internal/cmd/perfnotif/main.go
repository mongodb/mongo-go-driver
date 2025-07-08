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

type ChangePoint struct {
	TimeSeriesInfo struct {
		Project     string `bson:"project"`
		Task        string `bson:"task"`
		Test        string `bson:"test"`
		Measurement string `bson:"measurement"`
	} `bson:"time_series_info"`
	TriageContexts []string `bson:"triage_contexts"`
	HScore         float64  `bson:"h_score"`
}

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

	commit := os.Getenv("COMMIT") // TODO: get PR from evergreen instead of just the commit, and use PR to get latest commit
	if commit == "" {
		log.Panic("could not retrieve commit number")
	}

	coll := client.Database("expanded_metrics").Collection("change_points")
	var changePoints []ChangePoint
	changePoints, err = getDocsWithContext(coll, "50cf0c20d228975074c0010bfb688917e25934a4") // TODO: restore test commit
	if err != nil {
		log.Panicf("Error retrieving and decoding documents from collection: %v.", err)
	}

	if len(changePoints) == 0 {
		log.Panicf("Nothing was decoded")
	}

	fmt.Print("Documents:")
	for _, cp := range changePoints {
		fmt.Printf("  Project: %s, Task: %s, Test: %s, Measurement: %s, Triage Contexts: %v, H-Score: %f\n",
			cp.TimeSeriesInfo.Project,
			cp.TimeSeriesInfo.Task,
			cp.TimeSeriesInfo.Test,
			cp.TimeSeriesInfo.Measurement,
			cp.TriageContexts,
			cp.HScore,
		)
	}

	err = client.Disconnect(context.Background())
	if err != nil {
		log.Panicf("Failed to disconnect client: %v", err)
	}

}

func getDocsWithContext(coll *mongo.Collection, commit string) ([]ChangePoint, error) {
	filter := bson.D{
		{"time_series_info.project", "mongo-go-driver"},
		{"time_series_info.variant", "perf"},
		{"time_series_info.task", "perf"},
		{"commit", commit},
		{"triage_contexts", bson.M{"$in": []string{"GoDriver perf (h-score)"}}},
	}

	projection := bson.D{
		{"time_series_info.project", 1},
		{"time_series_info.task", 1},
		{"time_series_info.test", 1},
		{"time_series_info.measurement", 1},
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

	fmt.Printf("Successfully retrieved %d documents from commit %s.", cursor.RemainingBatchLength(), commit)

	var changePoints []ChangePoint
	for cursor.Next(findCtx) {
		var cp ChangePoint
		if err := cursor.Decode(&cp); err != nil {
			return nil, err
		}
		changePoints = append(changePoints, cp)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return changePoints, nil
}
