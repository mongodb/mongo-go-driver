// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"gonum.org/v1/gonum/mat"
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
		TestName string                 `bson:"test_name"`
		Args     map[string]interface{} `bson:"args"`
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

type EnergyStats struct {
	Benchmark       string
	PatchVersion    string
	MainlineVersion string
	E               float64
	T               float64
	H               float64
}

func main() {
	// Connect to analytics node
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

	coll := client.Database("expanded_metrics").Collection("raw_results")
	version := os.Getenv("VERSION_ID")
	if version == "" {
		log.Panic("could not retrieve version")
	}

	// Get and pre-process raw data
	patchRawData, err3 := findRawData(version, coll)
	if err3 != nil {
		log.Panicf("Error getting raw data: %v", err3)
	}

	mainlineCommits, err4 := parseMainelineCommits(patchRawData)
	if err4 != nil {
		log.Panicf("Error parsing commits: %v", err4)
	}

	mainlineVersion := "mongo_go_driver_" + mainlineCommits[0]
	mainlineRawData, err5 := findRawData(mainlineVersion, coll)
	if err5 != nil {
		log.Panicf("Could not retrieve mainline raw data")
	}

	if len(mainlineRawData) != len(patchRawData) {
		log.Panicf("Path and mainline data length do not match.")
	}

	// Calculate energy statistics
	energyStats, err := getEnergyStatsForAllBenchMarks(patchRawData, mainlineRawData)
	if err != nil {
		log.Panicf("Error calculating energy stats: %v", err)
	}
	fmt.Printf("Successfully retrieved %d energy stats.\n", len(energyStats))

	// Disconnect client
	err0 := client.Disconnect(context.Background())
	if err0 != nil {
		log.Panicf("Failed to disconnect client: %v", err0)
	}

}

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

func parseMainelineCommits(rawData []RawData) ([]string, error) {
	commits := make([]string, 0, len(rawData))
	for i, rd := range rawData {
		taskID := rd.Info.TaskID
		pieces := strings.Split(taskID, "_") // Format: mongo_go_driver_perf_perf_patch_<commit-SHA>_<version>_<timestamp>
		for j, p := range pieces {
			if p == "patch" {
				if len(pieces) < j+2 {
					return nil, errors.New("task ID doesn't hold commit SHA")
				}
				commits = append(commits, pieces[j+1])
				break
			}
		}
		if len(commits) < i+1 { // didn't find SHA in task_ID
			return nil, errors.New("task ID doesn't hold commit SHA")
		}
	}
	return commits, nil
}

func getEnergyStatsForOneBenchmark(xRaw RawData, yRaw RawData) (*EnergyStats, error) {

	var x []float64
	var y []float64
	for _, stat := range xRaw.Rollups.Stats {
		x = append(x, stat.Val)
	}
	for _, stat := range yRaw.Rollups.Stats {
		y = append(y, stat.Val)
	}

	for i := range (int)(math.Min((float64)(len(xRaw.Rollups.Stats)), float64(len(yRaw.Rollups.Stats)))) {
		if xRaw.Rollups.Stats[i].Name != yRaw.Rollups.Stats[i].Name {
			return nil, errors.New("measurements do not match")
		}
	}

	e, t, h := GetEnergyStatistics(mat.NewDense(len(x), 1, x), mat.NewDense(len(y), 1, y))
	return &EnergyStats{
		Benchmark:       xRaw.Info.TestName,
		PatchVersion:    xRaw.Info.Version,
		MainlineVersion: yRaw.Info.Version,
		E:               e,
		T:               t,
		H:               h,
	}, nil
}

func getEnergyStatsForAllBenchMarks(patchRawData []RawData, mainlineRawData []RawData) ([]*EnergyStats, error) {

	sort.Slice(patchRawData, func(i, j int) bool {
		return patchRawData[i].Info.TestName < patchRawData[j].Info.TestName
	})
	sort.Slice(mainlineRawData, func(i, j int) bool {
		return mainlineRawData[i].Info.TestName < mainlineRawData[j].Info.TestName
	})

	var energyStats []*EnergyStats
	for i := range patchRawData {
		if testname := patchRawData[i].Info.TestName; testname != mainlineRawData[i].Info.TestName {
			return nil, errors.New("tests do not match")
		}

		es, err := getEnergyStatsForOneBenchmark(patchRawData[i], mainlineRawData[i])
		if err != nil {
			return nil, err
		}
		energyStats = append(energyStats, es)

		fmt.Printf("%s | H-score: %.4f\n", es.Benchmark, es.H)
	}
	return energyStats, nil
}
