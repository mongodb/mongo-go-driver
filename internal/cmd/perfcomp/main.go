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
	"math"
	"os"
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

type StableRegion struct {
	TimeSeriesInfo struct {
		Project     string                 `bson:"project"`
		Variant     string                 `bson:"variant"`
		Task        string                 `bson:"task"`
		Test        string                 `bson:"test"`
		Measurement string                 `bson:"measurement"`
		Args        map[string]interface{} `bson:"args"`
	}
	Start                  interface{}   `bson:"start"`
	End                    interface{}   `bson:"end"`
	Values                 []float64     `bson:"values"`
	StartOrder             int64         `bson:"start_order"`
	EndOrder               int64         `bson:"end_order"`
	Mean                   float64       `bson:"mean"`
	Std                    float64       `bson:"std"`
	Median                 float64       `bson:"median"`
	Max                    float64       `bson:"max"`
	Min                    float64       `bson:"min"`
	CoefficientOfVariation float64       `bson:"coefficient_of_variation"`
	LastSuccessfulUpdate   interface{}   `bson:"last_successful_update"`
	Last                   bool          `bson:"last"`
	Contexts               []interface{} `bson:"contexts"`
}

type EnergyStats struct {
	Benchmark    string
	Measurement  string
	PatchVersion string
	StableRegion StableRegion
	PatchValues  []float64
	PChange      float64
	E            float64
	T            float64
	H            float64
	Z            float64
}

func main() {
	// Connect to analytics node
	uri := os.Getenv("PERF_URI_PRIVATE_ENDPOINT")
	if uri == "" {
		log.Panic("PERF_URI_PRIVATE_ENDPOINT env variable is not set")
	}

	version := os.Args[len(os.Args)-1]
	if version == "" {
		log.Panic("could not get VERSION_ID")
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

	db := client.Database("expanded_metrics")

	// Get raw data, most recent stable region, and calculate energy stats
	patchRawData, err3 := findRawData(version, db.Collection("raw_results"))
	if err3 != nil {
		log.Panicf("Error getting raw data: %v", err3)
	}

	allEnergyStats, err4 := getEnergyStatsForAllBenchMarks(patchRawData, db.Collection("stable_regions"))
	if err4 != nil {
		log.Panicf("Error getting energy statistics: %v", err4)
	}
	fmt.Println(generatePRComment(allEnergyStats, version))

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

	findCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cursor, err := coll.Find(findCtx, filter)
	if err != nil {
		log.Panicf(
			"Error retrieving raw data for version %q: %v",
			version,
			err,
		)
	}
	defer cursor.Close(findCtx)

	fmt.Printf("Successfully retrieved %d docs from version %s.\n", cursor.RemainingBatchLength(), version)

	var rawData []RawData
	err = cursor.All(findCtx, &rawData)
	if err != nil {
		log.Panicf(
			"Error decoding raw data from version %q: %v",
			version,
			err,
		)
	}

	return rawData, nil
}

// Find the most recent stable region of the mainline version for a specific test/measurement
func findLastStableRegion(testname string, measurement string, coll *mongo.Collection) (*StableRegion, error) {
	filter := bson.D{
		{"time_series_info.project", "mongo-go-driver"},
		{"time_series_info.variant", "perf"},
		{"time_series_info.task", "perf"},
		{"time_series_info.test", testname},
		{"time_series_info.measurement", measurement},
		{"last", true},
		{"contexts", bson.D{{"$in", bson.A{"GoDriver perf task"}}}},
	}

	findOptions := options.FindOne().SetSort(bson.D{{"end", -1}})

	findCtx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	var sr *StableRegion
	err := coll.FindOne(findCtx, filter, findOptions).Decode(&sr)
	if err != nil {
		return nil, err
	}
	return sr, nil
}

// For a specific test and measurement
func getEnergyStatsForOneBenchmark(rd RawData, coll *mongo.Collection) ([]*EnergyStats, error) {
	testname := rd.Info.TestName
	var energyStats []*EnergyStats

	for i := range rd.Rollups.Stats {
		measurement := rd.Rollups.Stats[i].Name
		patchVal := []float64{rd.Rollups.Stats[i].Val}

		stableRegion, err := findLastStableRegion(testname, measurement, coll)
		if err != nil {
			log.Panicf(
				"Error finding last stable region for test %q, measurement %q: %v",
				testname,
				measurement,
				err,
			)
		}

		pChange := GetPercentageChange(patchVal[0], stableRegion.Mean)
		e, t, h := GetEnergyStatistics(mat.NewDense(len(stableRegion.Values), 1, stableRegion.Values), mat.NewDense(1, 1, patchVal))
		z := GetZScore(patchVal[0], stableRegion.Mean, stableRegion.Std)

		es := EnergyStats{
			Benchmark:    testname,
			Measurement:  measurement,
			PatchVersion: rd.Info.Version,
			StableRegion: *stableRegion,
			PatchValues:  patchVal,
			PChange:      pChange,
			E:            e,
			T:            t,
			H:            h,
			Z:            z,
		}
		energyStats = append(energyStats, &es)
	}

	return energyStats, nil
}

func getEnergyStatsForAllBenchMarks(patchRawData []RawData, coll *mongo.Collection) ([]*EnergyStats, error) {
	var allEnergyStats []*EnergyStats
	for _, rd := range patchRawData {
		energyStats, err := getEnergyStatsForOneBenchmark(rd, coll)
		if err != nil {
			return nil, err
		}
		allEnergyStats = append(allEnergyStats, energyStats...)
	}
	return allEnergyStats, nil
}

func generatePRComment(energyStats []*EnergyStats, version string) string {
	var comment strings.Builder
	comment.WriteString("# 👋GoDriver Performance\n")
	fmt.Fprintf(&comment, "The following benchmark tests for version %s had statistically significant changes (i.e., |z-score| > 1.96):\n", version)
	comment.WriteString("| Benchmark | Measurement | H-Score | Z-Score | % Change | Stable Reg | Patch Value |\n| --- | --- | --- | --- | --- | --- | --- |\n")

	var testCount int64
	for _, es := range energyStats {
		if math.Abs(es.Z) > 1.96 {
			testCount += 1
			fmt.Fprintf(&comment, "| %s | %s | %.4f | %.4f | %.4f | Avg: %.4f, Stdev: %.4f | %.4f |\n", es.Benchmark, es.Measurement, es.H, es.Z, es.PChange, es.StableRegion.Mean, es.StableRegion.Std, es.PatchValues[0])
		}
	}

	if testCount == 0 {
		comment.Reset()
		comment.WriteString("# 👋GoDriver Performance\n")
		comment.WriteString("There were no significant changes to the performance to report.")
	}

	comment.WriteString("\n*For a comprehensive view of all microbenchmark results for this PR's commit, please check out the Evergreen perf task for this patch.*")
	return comment.String()
}
