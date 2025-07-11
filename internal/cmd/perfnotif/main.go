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

type RawData struct {
	Info struct {
		Project      string `bson:"project"`
		Version      string `bson:"version"`
		Variant      string `bson:"variant"`
		Order        int64  `bson:"order"`
		TaskName     string `bson:"task_name"`
		TaskID       string `bson:"task_id"`
		Execution    int64  `bson:"execution"`
		Mainline     bool   `bson:"mainline"`
		OverrideInfo struct {
			OverrideMainline bool        `bson:"override_mainline"`
			BaseOrder        interface{} `bson:"base_order"`
			Reason           interface{} `bson:"reason"`
			User             interface{} `bson:"user"`
		}
		TestName string        `bson:"test_name"`
		Args     []interface{} `bson:"args"`
	}
	CreatedAt   interface{} `bson:"created_at"`
	CompletedAt interface{} `bson:"completed_at"`
	Rollups     struct {
		Stats []struct {
			Name     string      `bson:"name"`
			Val      float64     `bson:"val"`
			Metadata interface{} `bson:"metadata"`
		}
	}
	FailedRollupAttempts int64 `bson:"failed_rollup_attempts"`
}

// findRawData will get all of the rawData for the given version
func findRawData(version string, coll *mongo.Collection) ([]RawData, error) {
	filter := bson.D{
		{"info.project", "mongo-go-driver"},
		{"info.version", version},
		{"info.variant", "perf"},
		{"info.task_name", "perf"},
	}

	findOptions := options.Find()

	findCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cursor, err := coll.Find(findCtx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(findCtx)

	fmt.Printf("Successfully retrieved %d docs from version %s.\n", cursor.RemainingBatchLength(), version)

	var rawData []RawData
	for cursor.Next(findCtx) {
		var rd RawData
		if err := cursor.Decode(&rd); err != nil {
			return nil, err
		}
		rawData = append(rawData, rd)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return rawData, nil
}

func main() {
	uri := os.Getenv("perf_uri_private_endpoint")
	if uri == "" {
		log.Panic("perf_uri_private_endpoint env variable is not set")
	}

	client, err1 := mongo.Connect(options.Client().ApplyURI(uri))
	if err1 != nil {
		log.Panicf("Error connecting client: %v", err1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err2 := client.Ping(ctx, nil)
	if err2 != nil {
		log.Panicf("Error pinging MongoDB Analytics: %v", err2)
	}
	fmt.Println("Successfully connected to MongoDB Analytics node.")

	commit := os.Getenv("COMMIT")
	if commit == "" {
		log.Panic("could not retrieve commit number")
	}

	coll := client.Database("expanded_metrics").Collection("raw_results")
	version := os.Getenv("VERSION")
	if version == "" {
		log.Panic("could not retrieve version")
	}
	rawData, err3 := findRawData(version, coll)
	if err3 != nil {
		log.Panicf("Error getting raw data: %v", err3)
	}
	fmt.Println(rawData)

	err0 := client.Disconnect(context.Background())
	if err0 != nil {
		log.Panicf("Failed to disconnect client: %v", err0)
	}

}
