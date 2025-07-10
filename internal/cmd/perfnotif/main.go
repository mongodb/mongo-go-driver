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

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type RawData struct {
	ID   string `json:"_id"`
	Info struct {
		Project      string `json:"project"`
		Version      string `json:"version"`
		Variant      string `json:"variant"`
		Order        int64  `json:"order"`
		TaskName     string `json:"task_name"`
		TaskID       string `json:"task_id"`
		Execution    int64  `json:"execution"`
		Mainline     bool   `json:"mainline"`
		OverrideInfo struct {
			OverrideMainline bool        `json:"override_mainline"`
			BaseOrder        interface{} `json:"base_order"`
			Reason           interface{} `json:"reason"`
			User             interface{} `json:"user"`
		}
		TestName string        `json:"test_name"`
		Args     []interface{} `json:"args"`
	}
	CreatedAt   interface{} `json:"created_at"`
	CompletedAt interface{} `json:"completed_at"`
	Rollups     struct {
		Stats struct {
			Name     string      `json:"name"`
			Val      float64     `json:"val"`
			Metadata interface{} `json:"metadata"`
		}
	}
	FailedRollupAttempts int64 `json:"failed_rollup_attempts"`
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

	commit := os.Getenv("COMMIT")
	if commit == "" {
		log.Panic("could not retrieve commit number")
	}

	// coll := client.Database("expanded_metrics").Collection("raw_results")

	err = client.Disconnect(context.Background())
	if err != nil {
		log.Panicf("Failed to disconnect client: %v", err)
	}

}

// func getMarkdownComment(changePoints []ChangePoint) bytes.Buffer {
// 	var buffer bytes.Buffer

// 	buffer.WriteString("# 👋 GoDriver Performance Notification\n")

// 	if len(changePoints) > 0 {
// 		buffer.WriteString("The following benchmark tests had statistically significant changes (i.e., h-score > 0.6):\n")
// 		buffer.WriteString("| Benchmark Test | Measurement | H-Score | Performance Baron |\n")
// 		buffer.WriteString("|---|---|---|---|\n")

// 		for _, cp := range changePoints {
// 			// TODO: update this to dynamically generate link
// 			var perfBaronLink = "https://performance-monitoring-and-analysis.server-tig.prod.corp.mongodb.com/baron"
// 			fmt.Fprintf(&buffer, "| %s | %s | %f | [linked here](%s) |\n", cp.TimeSeriesInfo.Test, cp.TimeSeriesInfo.Measurement, cp.HScore, perfBaronLink)
// 		}
// 	} else {
// 		buffer.WriteString("There were no significant changes to the performance to report.\n")
// 	}
// 	// TODO: update this to dynamically generate link
// 	buffer.WriteString("*For a comprehensive view of all microbenchmark results for this PR's commit, please visit [this link](https://performance-monitoring-and-analysis.server-tig.prod.corp.mongodb.com/baron?change_point_filters=%5B%7B%22active%22%3Atrue%2C%22name%22%3A%22commit%22%2C%22operator%22%3A%22matches%22%2C%22type%22%3A%22regex%22%2C%22value%22%3A%22%22%7D%2C%7B%22active%22%3Atrue%2C%22name%22%3A%22commit_date%22%2C%22operator%22%3A%22after%22%2C%22type%22%3A%22date%22%7D%2C%7B%22active%22%3Atrue%2C%22name%22%3A%22calculated_on%22%2C%22operator%22%3A%22after%22%2C%22type%22%3A%22date%22%7D%2C%7B%22active%22%3Atrue%2C%22name%22%3A%22project%22%2C%22operator%22%3A%22matches%22%2C%22type%22%3A%22regex%22%2C%22value%22%3A%22mongo-go-driver%22%7D%2C%7B%22active%22%3Atrue%2C%22name%22%3A%22variant%22%2C%22operator%22%3A%22matches%22%2C%22type%22%3A%22regex%22%2C%22value%22%3A%22perf%22%7D%2C%7B%22active%22%3Atrue%2C%22name%22%3A%22task%22%2C%22operator%22%3A%22matches%22%2C%22type%22%3A%22regex%22%2C%22value%22%3A%22perf%22%7D%2C%7B%22active%22%3Atrue%2C%22name%22%3A%22test%22%2C%22operator%22%3A%22matches%22%2C%22type%22%3A%22regex%22%2C%22value%22%3A%22%22%7D%2C%7B%22active%22%3Atrue%2C%22name%22%3A%22measurement%22%2C%22operator%22%3A%22matches%22%2C%22type%22%3A%22regex%22%2C%22value%22%3A%22%22%7D%2C%7B%22active%22%3Atrue%2C%22name%22%3A%22args%22%2C%22operator%22%3A%22eq%22%2C%22type%22%3A%22json%22%7D%2C%7B%22active%22%3Atrue%2C%22name%22%3A%22percent_change%22%2C%22operator%22%3A%22gt%22%2C%22type%22%3A%22number%22%7D%2C%7B%22active%22%3Atrue%2C%22name%22%3A%22z_score_change%22%2C%22operator%22%3A%22gt%22%2C%22type%22%3A%22number%22%7D%2C%7B%22active%22%3Atrue%2C%22name%22%3A%22h_score%22%2C%22operator%22%3A%22gt%22%2C%22type%22%3A%22number%22%7D%2C%7B%22active%22%3Atrue%2C%22name%22%3A%22absolute_change%22%2C%22operator%22%3A%22gt%22%2C%22type%22%3A%22number%22%7D%2C%7B%22active%22%3Atrue%2C%22name%22%3A%22build_failures%22%2C%22operator%22%3A%22matches%22%2C%22type%22%3A%22regex%22%2C%22value%22%3A%22%22%7D%2C%7B%22active%22%3Atrue%2C%22name%22%3A%22bf_suggestions%22%2C%22operator%22%3A%22inlist%22%2C%22type%22%3A%22listSelect%22%7D%2C%7B%22active%22%3Atrue%2C%22name%22%3A%22triage_status%22%2C%22operator%22%3A%22inlist%22%2C%22type%22%3A%22listSelect%22%7D%2C%7B%22active%22%3Atrue%2C%22name%22%3A%22changeType%22%2C%22operator%22%3A%22inlist%22%2C%22type%22%3A%22listSelect%22%7D%2C%7B%22active%22%3Atrue%2C%22name%22%3A%22triage_contexts%22%2C%22operator%22%3A%22inlist%22%2C%22type%22%3A%22listSelect%22%2C%22value%22%3A%5B%22GoDriver+perf+%28h-score%29%22%5D%7D%5D).*")

// 	return buffer
// }
