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
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"gonum.org/v1/gonum/mat"
)

type OverrideInfo struct {
	OverrideMainline bool `bson:"override_mainline"`
	BaseOrder        any  `bson:"base_order"`
	Reason           any  `bson:"reason"`
	User             any  `bson:"user"`
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
	TestName     string         `bson:"test_name"`
	Args         map[string]any `bson:"args"`
}

type Stat struct {
	Name     string  `bson:"name"`
	Val      float64 `bson:"val"`
	Metadata any     `bson:"metadata"`
}

type Rollups struct {
	Stats []Stat
}

type RawData struct {
	Info                 Info
	CreatedAt            any `bson:"created_at"`
	CompletedAt          any `bson:"completed_at"`
	Rollups              Rollups
	FailedRollupAttempts int64 `bson:"failed_rollup_attempts"`
}

type TimeSeriesInfo struct {
	Project     string         `bson:"project"`
	Variant     string         `bson:"variant"`
	Task        string         `bson:"task"`
	Test        string         `bson:"test"`
	Measurement string         `bson:"measurement"`
	Args        map[string]any `bson:"args"`
}

type StableRegion struct {
	TimeSeriesInfo         TimeSeriesInfo
	Start                  any       `bson:"start"`
	End                    any       `bson:"end"`
	Values                 []float64 `bson:"values"`
	StartOrder             int64     `bson:"start_order"`
	EndOrder               int64     `bson:"end_order"`
	Mean                   float64   `bson:"mean"`
	Std                    float64   `bson:"std"`
	Median                 float64   `bson:"median"`
	Max                    float64   `bson:"max"`
	Min                    float64   `bson:"min"`
	CoefficientOfVariation float64   `bson:"coefficient_of_variation"`
	LastSuccessfulUpdate   any       `bson:"last_successful_update"`
	Last                   bool      `bson:"last"`
	Contexts               []any     `bson:"contexts"`
}

type EnergyStats struct {
	Project         string
	Benchmark       string
	Measurement     string
	PatchVersion    string
	StableRegion    StableRegion
	MeasurementVal  float64
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

	// TODO (GODRIVER-3102): Map each project to a unique performance context,
	// necessary for project switching to work since it's required for querying the stable region.
	project := flag.String("project", "mongo-go-driver", "specify the name of an existing Evergreen project")
	if project == nil {
		log.Fatalf("must provide project")
	}

	// Connect to analytics node
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("Error connecting client: %v", err)
	}

	defer func() { // Defer disconnect client
		err = client.Disconnect(context.Background())
		if err != nil {
			log.Fatalf("Failed to disconnect client: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("Error pinging MongoDB Analytics: %v", err)
	}
	log.Println("Successfully connected to MongoDB Analytics node.")

	db := client.Database(expandedMetricsDB)

	// Get raw data, most recent stable region, and calculate energy stats
	findCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	patchRawData, err := findRawData(findCtx, *project, version, db.Collection(rawResultsColl))
	if err != nil {
		log.Fatalf("Error getting raw data: %v", err)
	}

	allEnergyStats, err := getEnergyStatsForAllBenchMarks(findCtx, patchRawData, db.Collection(stableRegionsColl))
	if err != nil {
		log.Fatalf("Error getting energy statistics: %v", err)
	}
	log.Println(generatePRComment(allEnergyStats, version))
}

func findRawData(ctx context.Context, project string, version string, coll *mongo.Collection) ([]RawData, error) {
	filter := bson.D{
		{"info.project", project},
		{"info.version", version},
		{"info.variant", "perf"},
		{"info.task_name", "perf"},
	}

	cursor, err := coll.Find(ctx, filter)
	if err != nil {
		log.Fatalf(
			"Error retrieving raw data for version %q: %v",
			version,
			err,
		)
	}
	defer func() {
		err = cursor.Close(ctx)
		if err != nil {
			log.Fatalf("Error closing cursor while retrieving raw data for version %q: %v", version, err)
		}
	}()

	log.Printf("Successfully retrieved %d docs from version %s.\n", cursor.RemainingBatchLength(), version)

	var rawData []RawData
	err = cursor.All(ctx, &rawData)
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
func findLastStableRegion(ctx context.Context, project string, testname string, measurement string, coll *mongo.Collection) (*StableRegion, error) {
	filter := bson.D{
		{"time_series_info.project", project},
		{"time_series_info.variant", "perf"},
		{"time_series_info.task", "perf"},
		{"time_series_info.test", testname},
		{"time_series_info.measurement", measurement},
		{"last", true},
		{"contexts", bson.D{{"$in", bson.A{"GoDriver perf task"}}}},
	}

	findOptions := options.FindOne().SetSort(bson.D{{"end", -1}})

	var sr *StableRegion
	err := coll.FindOne(ctx, filter, findOptions).Decode(&sr)
	if err != nil {
		return nil, err
	}
	return sr, nil
}

// For a specific test and measurement
func getEnergyStatsForOneBenchmark(ctx context.Context, rd RawData, coll *mongo.Collection) ([]*EnergyStats, error) {
	testname := rd.Info.TestName
	var energyStats []*EnergyStats

	for i := range rd.Rollups.Stats {
		project := rd.Info.Project
		measName := rd.Rollups.Stats[i].Name
		measVal := rd.Rollups.Stats[i].Val

		stableRegion, err := findLastStableRegion(ctx, project, testname, measName, coll)
		if err != nil {
			log.Fatalf(
				"Error finding last stable region for test %q, measurement %q: %v",
				testname,
				measName,
				err,
			)
		}

		// The performance analyzer compares the measurement value from the patch to a stable region that succeeds the latest change point.
		// For example, if there were 5 measurements since the last change point, then the stable region is the 5 latest values for the measurement.
		stableRegionVec := mat.NewDense(len(stableRegion.Values), 1, stableRegion.Values)
		measValVec := mat.NewDense(1, 1, []float64{measVal}) // singleton

		estat, tstat, hscore, err := getEnergyStatistics(stableRegionVec, measValVec)
		var zscore float64
		var pChange float64
		if err != nil {
			log.Printf("Could not calculate energy stats for test %q, measurement %q: %v", testname, measName, err)
			zscore = 0
			pChange = 0
		} else {
			zscore = getZScore(measVal, stableRegion.Mean, stableRegion.Std)
			pChange = getPercentageChange(measVal, stableRegion.Mean)
		}

		es := EnergyStats{
			Project:         project,
			Benchmark:       testname,
			Measurement:     measName,
			PatchVersion:    rd.Info.Version,
			StableRegion:    *stableRegion,
			MeasurementVal:  measVal,
			PercentChange:   pChange,
			EnergyStatistic: estat,
			TestStatistic:   tstat,
			HScore:          hscore,
			ZScore:          zscore,
		}
		energyStats = append(energyStats, &es)
	}

	return energyStats, nil
}

func getEnergyStatsForAllBenchMarks(ctx context.Context, patchRawData []RawData, coll *mongo.Collection) ([]*EnergyStats, error) {
	var allEnergyStats []*EnergyStats
	for _, rd := range patchRawData {
		energyStats, err := getEnergyStatsForOneBenchmark(ctx, rd, coll)
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

	w := tabwriter.NewWriter(&comment, 0, 0, 1, ' ', 0)
	fmt.Fprintln(w, "| Benchmark\t| Measurement\t| H-Score\t| Z-Score\t| % Change\t| Stable Reg\t| Patch Value\t|")
	fmt.Fprintln(w, "| ---------\t| -----------\t| -------\t| -------\t| --------\t| ----------\t| -----------\t|")

	var testCount int64
	for _, es := range energyStats {
		if math.Abs(es.ZScore) > 1.96 {
			testCount += 1
			fmt.Fprintf(w, "| %s\t| %s\t| %.4f\t| %.4f\t| %.4f\t| Avg: %.4f, Med: %.4f, Stdev: %.4f\t| %.4f\t|\n", es.Benchmark, es.Measurement, es.HScore, es.ZScore, es.PercentChange, es.StableRegion.Mean, es.StableRegion.Median, es.StableRegion.Std, es.MeasurementVal)
		}
	}
	w.Flush()

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
func getEnergyStatistics(x, y *mat.Dense) (float64, float64, float64, error) {
	xrows, xcols := x.Dims()
	yrows, ycols := y.Dims()

	if xcols != ycols {
		return 0, 0, 0, fmt.Errorf("both inputs must have the same number of columns")
	}
	if xrows == 0 || yrows == 0 {
		return 0, 0, 0, fmt.Errorf("inputs cannot be empty")
	}

	xrowsf := float64(xrows)
	yrowsf := float64(yrows)

	var A float64 // E|X-Y|
	if xrowsf > 0 && yrowsf > 0 {
		dist, err := getDistance(x, y)
		if err != nil {
			return 0, 0, 0, err
		}
		A = dist / (xrowsf * yrowsf)
	} else {
		A = 0
	}

	var B float64 // E|X-X'|
	if xrowsf > 0 {
		dist, err := getDistance(x, x)
		if err != nil {
			return 0, 0, 0, err
		}
		B = dist / (xrowsf * xrowsf)
	} else {
		B = 0
	}

	var C float64 // E|Y-Y'|
	if yrowsf > 0 {
		dist, err := getDistance(y, y)
		if err != nil {
			return 0, 0, 0, err
		}
		C = dist / (yrowsf * yrowsf)
	} else {
		C = 0
	}

	E := 2*A - B - C // D^2(F_x, F_y)
	T := ((xrowsf * yrowsf) / (xrowsf + yrowsf)) * E
	var H float64
	if A > 0 {
		H = E / (2 * A)
	} else {
		H = 0
	}
	return E, T, H, nil
}

// Given two vectors (expected 1 col),
// this function returns the sum of distances between each pair.
func getDistance(x, y *mat.Dense) (float64, error) {
	xrows, xcols := x.Dims()
	yrows, ycols := y.Dims()

	if xcols != 1 || ycols != 1 {
		return 0, fmt.Errorf("both inputs must be column vectors")
	}

	var sum float64

	for i := 0; i < xrows; i++ {
		for j := 0; j < yrows; j++ {
			sum += math.Abs(x.At(i, 0) - y.At(j, 0))
		}
	}
	return sum, nil
}

// Get Z score for result x, compared to mean u and st dev o.
func getZScore(x, mu, sigma float64) float64 {
	if sigma == 0 {
		return math.NaN()
	}
	return (x - mu) / sigma
}

// Get percentage change for result x compared to mean u.
func getPercentageChange(x, mu float64) float64 {
	if mu == 0 {
		return math.NaN()
	}
	return ((x - mu) / mu) * 100
}
