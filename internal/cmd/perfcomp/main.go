// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// This module cannot be included in the workspace since it requires a version of Gonum that is not compatible with the Go Driver.
// Must use GOWORK=off to run this test.

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

type OverrideInfo struct {
	OverrideMainline bool        `bson:"override_mainline"`
	BaseOrder        interface{} `bson:"base_order"`
	Reason           interface{} `bson:"reason"`
	User             interface{} `bson:"user"`
}

type Info struct {
	Project      string `bson:"project"`
	Version      string `bson:"version"`
	Variant      string `bson:"variant"`
	Order        int64  `bson:"order"`
	TaskName     string `bson:"task_name"`
	TaskID       string `bson:"task_id"`
	Execution    int64  `bson:"execution"`
	Mainline     bool   `bson:"mainline"`
	OverrideInfo OverrideInfo
	TestName     string                 `bson:"test_name"`
	Args         map[string]interface{} `bson:"args"`
}

type Stat struct {
	Name     string      `bson:"name"`
	Val      float64     `bson:"val"`
	Metadata interface{} `bson:"metadata"`
}

type Rollups struct {
	Stats []Stat
}

type RawData struct {
	Info                 Info
	CreatedAt            interface{} `bson:"created_at"`
	CompletedAt          interface{} `bson:"completed_at"`
	Rollups              Rollups
	FailedRollupAttempts int64 `bson:"failed_rollup_attempts"`
}

type TimeSeriesInfo struct {
	Project     string                 `bson:"project"`
	Variant     string                 `bson:"variant"`
	Task        string                 `bson:"task"`
	Test        string                 `bson:"test"`
	Measurement string                 `bson:"measurement"`
	Args        map[string]interface{} `bson:"args"`
}

type StableRegion struct {
	TimeSeriesInfo         TimeSeriesInfo
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
	Benchmark       string
	Measurement     string
	PatchVersion    string
	StableRegion    StableRegion
	PatchValues     []float64
	PercentChange   float64
	EnergyStatistic float64
	TestStatistic   float64
	HScore          float64
	ZScore          float64
}

const expandedMetricsDB = "expanded_metrics"
const rawResultsColl = "raw_results"
const stableRegionsColl = "stable_regions"

func main() {
	// Check for variables
	uri := os.Getenv("PERF_URI_PRIVATE_ENDPOINT")
	if uri == "" {
		log.Fatal("PERF_URI_PRIVATE_ENDPOINT env variable is not set")
	}

	version := os.Args[len(os.Args)-1]
	if version == "" {
		log.Fatal("could not get VERSION_ID")
	}

	// Connect to analytics node
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("Error connecting client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("Error pinging MongoDB Analytics: %v", err)
	}
	log.Println("Successfully connected to MongoDB Analytics node.")

	db := client.Database(expandedMetricsDB)

	// Get raw data, most recent stable region, and calculate energy stats
	patchRawData, err := findRawData(version, db.Collection(rawResultsColl))
	if err != nil {
		log.Fatalf("Error getting raw data: %v", err)
	}

	allEnergyStats, err := getEnergyStatsForAllBenchMarks(patchRawData, db.Collection(stableRegionsColl))
	if err != nil {
		log.Fatalf("Error getting energy statistics: %v", err)
	}
	log.Println(generatePRComment(allEnergyStats, version))

	// Disconnect client
	err = client.Disconnect(context.Background())
	if err != nil {
		log.Fatalf("Failed to disconnect client: %v", err)
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
		log.Fatalf(
			"Error retrieving raw data for version %q: %v",
			version,
			err,
		)
	}
	defer func() { err = cursor.Close(findCtx) }()

	log.Printf("Successfully retrieved %d docs from version %s.\n", cursor.RemainingBatchLength(), version)

	var rawData []RawData
	err = cursor.All(findCtx, &rawData)
	if err != nil {
		log.Fatalf(
			"Error decoding raw data from version %q: %v",
			version,
			err,
		)
	}

	return rawData, err
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
			log.Fatalf(
				"Error finding last stable region for test %q, measurement %q: %v",
				testname,
				measurement,
				err,
			)
		}

		pChange := getPercentageChange(patchVal[0], stableRegion.Mean)
		e, t, h := getEnergyStatistics(mat.NewDense(len(stableRegion.Values), 1, stableRegion.Values), mat.NewDense(1, 1, patchVal))
		z := getZScore(patchVal[0], stableRegion.Mean, stableRegion.Std)

		es := EnergyStats{
			Benchmark:       testname,
			Measurement:     measurement,
			PatchVersion:    rd.Info.Version,
			StableRegion:    *stableRegion,
			PatchValues:     patchVal,
			PercentChange:   pChange,
			EnergyStatistic: e,
			TestStatistic:   t,
			HScore:          h,
			ZScore:          z,
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
		if math.Abs(es.ZScore) > 1.96 {
			testCount += 1
			fmt.Fprintf(&comment, "| %s | %s | %.4f | %.4f | %.4f | Avg: %.4f, Med: %.4f, Stdev: %.4f | %.4f |\n", es.Benchmark, es.Measurement, es.HScore, es.ZScore, es.PercentChange, es.StableRegion.Mean, es.StableRegion.Median, es.StableRegion.Std, es.PatchValues[0])
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

// Given two matrices, this function returns
// (e, t, h) = (E-statistic, test statistic, e-coefficient of inhomogeneity)
func getEnergyStatistics(x, y *mat.Dense) (float64, float64, float64) {
	n, _ := x.Dims()
	m, _ := y.Dims()
	nf := float64(n)
	mf := float64(m)

	var A float64 // E|X-Y|
	if nf > 0 && mf > 0 {
		A = getDistance(x, y) / (nf * mf)
	} else {
		A = 0
	}
	var B float64 // E|X-X'|
	if nf > 0 {
		B = getDistance(x, x) / (nf * nf)
	} else {
		B = 0
	}
	var C float64 // E|Y-Y'|
	if mf > 0 {
		C = getDistance(y, y) / (mf * mf)
	} else {
		C = 0
	}

	E := 2*A - B - C // D^2(F_x, F_y)
	T := ((nf * mf) / (nf + mf)) * E
	var H float64
	if A > 0 {
		H = E / (2 * A)
	} else {
		H = 0
	}
	return E, T, H
}

// Given two vectors (expected 1 col),
// this function returns the sum of distances between each pair.
func getDistance(x, y *mat.Dense) float64 {
	xrows, _ := x.Dims()
	yrows, _ := y.Dims()

	var sum float64

	for i := 0; i < xrows; i++ {
		for j := 0; j < yrows; j++ {
			sum += math.Sqrt(math.Pow((x.At(i, 0) - y.At(j, 0)), 2))
		}
	}
	return sum
}

// Get Z score for result x, compared to mean u and st dev o.
func getZScore(x, mu, sigma float64) float64 {
	return (x - mu) / sigma
}

// Get percentage change for result x compared to mean u.
func getPercentageChange(x, mu float64) float64 {
	return ((x - mu) / mu) * 100
}
