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
	"os"
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
	Benchmark          string
	Measurement        string
	PatchVersion       string
	StableRegionValues []float64
	PatchValues        []float64
	E                  float64
	T                  float64
	H                  float64
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

	db := client.Database("expanded_metrics")
	version := os.Getenv("VERSION_ID")
	if version == "" {
		log.Panic("could not retrieve version")
	}

	// Get raw data, most recent stable region, and calculate energy stats
	patchRawData, err3 := findRawData(version, db.Collection("raw_results"))
	if err3 != nil {
		log.Panicf("Error getting raw data: %v", err3)
	}

	allEnergyStats, err4 := getEnergyStatsForAllBenchMarks(patchRawData, db.Collection("stable_regions"))
	if err4 != nil {
		log.Panicf("Error getting raw data: %v", err4)
	}
	for _, es := range allEnergyStats {
		fmt.Printf("%s | %s | E: %f | T: %f | H: %f\n", es.Benchmark, es.Measurement, es.E, es.T, es.H)
	}

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

func findLastStableRegion(testname string, measurement string, coll *mongo.Collection) ([]float64, error) {
	filter := bson.D{
		{"time_series_info.project", "mongo-go-driver"},
		{"time_series_info.variant", "perf"},
		{"time_series_info.task", "perf"},
		{"time_series_info.test", testname},
		{"time_series_info.measurement", measurement},
		{"last", true},
		{"contexts", []string{"GoDriver perf (h-score)"}},
	}
	projection := bson.D{
		{"values", 1},
	}
	findOptions := options.FindOne().SetSort(bson.D{{"end", -1}}).SetProjection(projection)

	findCtx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	var result bson.M
	err := coll.FindOne(findCtx, filter, findOptions).Decode(&result)
	if err != nil {
		return nil, err
	}

	valuesSlice, ok := result["values"].(bson.A)
	if !ok {
		return nil, errors.New("values is not of type bson.A")
	}
	var values []float64
	for _, v := range valuesSlice {
		number, ok := v.(float64)
		if !ok {
			return nil, errors.New("value is not float64")
		}
		values = append(values, number)
	}
	return values, nil
}

// For a specific test and measurement
func getEnergyStatsForOneBenchmark(rd RawData, coll *mongo.Collection) ([]*EnergyStats, error) {
	testname := rd.Info.TestName
	var energyStats []*EnergyStats

	for i := range rd.Rollups.Stats {
		measurement := rd.Rollups.Stats[i].Name
		patchVal := []float64{rd.Rollups.Stats[i].Val}
		stableRegionVals, err := findLastStableRegion(testname, measurement, coll)
		if err != nil {
			return nil, err
		}
		e, t, h := GetEnergyStatistics(mat.NewDense(len(stableRegionVals), 1, stableRegionVals), mat.NewDense(1, 1, patchVal))
		es := EnergyStats{
			Benchmark:          testname,
			Measurement:        measurement,
			PatchVersion:       rd.Info.Version,
			StableRegionValues: stableRegionVals,
			PatchValues:        patchVal,
			E:                  e,
			T:                  t,
			H:                  h,
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
