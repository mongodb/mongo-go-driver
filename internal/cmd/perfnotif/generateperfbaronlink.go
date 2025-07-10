// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"encoding/json"
	"fmt"
	"net/url"
)

const baseURL = "https://performance-monitoring-and-analysis.server-tig.prod.corp.mongodb.com/baron"

type CPFilter struct {
	Active   bool        `json:"active"`
	Name     string      `json:"name"`
	Operator string      `json:"operator"`
	Type     string      `json:"type"`
	Value    interface{} `json:"value,omitempty"` // Exclude if no value, following structure of change_point_filters
}

func createBaseChangePointFilters() []CPFilter {
	return []CPFilter{
		{Active: true, Name: "commit", Operator: "matches", Type: "regex"},
		{Active: true, Name: "commit_date", Operator: "after", Type: "date"},
		{Active: true, Name: "calculated_on", Operator: "after", Type: "date"},
		{Active: true, Name: "project", Operator: "matches", Type: "regex"},
		{Active: true, Name: "variant", Operator: "matches", Type: "regex"},
		{Active: true, Name: "task", Operator: "matches", Type: "regex"},
		{Active: true, Name: "test", Operator: "matches", Type: "regex"},
		{Active: true, Name: "measurement", Operator: "matches", Type: "regex"},
		{Active: true, Name: "args", Operator: "eq", Type: "json"},
		{Active: true, Name: "percent_change", Operator: "gt", Type: "number"},
		{Active: true, Name: "z_score_change", Operator: "gt", Type: "number"},
		{Active: true, Name: "h_score", Operator: "gt", Type: "number"},
		{Active: true, Name: "absolute_change", Operator: "gt", Type: "number"},
		{Active: true, Name: "build_failures", Operator: "matches", Type: "regex"},
		{Active: true, Name: "bf_suggestions", Operator: "inlist", Type: "listSelect"},
		{Active: true, Name: "triage_status", Operator: "inlist", Type: "listSelect"},
		{Active: true, Name: "changeType", Operator: "inlist", Type: "listSelect"},
		{Active: true, Name: "triage_contexts", Operator: "inlist", Type: "listSelect"},
	}
}

func updateFilterString(filters []CPFilter, name string, val string) {
	for i := range filters {
		if filters[i].Name == name {
			filters[i].Value = val
		}
	}
}

func updateFilterList(filters []CPFilter, name string, val []string) {
	for i := range filters {
		if filters[i].Name == name {
			filters[i].Value = val
		}
	}
}

func createGoDriverContextFilters() []CPFilter {
	baseFilters := createBaseChangePointFilters()
	updateFilterString(baseFilters, "project", "mongo-go-driver")
	updateFilterString(baseFilters, "variant", "perf")
	updateFilterString(baseFilters, "task", "perf")
	updateFilterList(baseFilters, "triage_contexts", []string{"GoDriver perf (h-score)"})
	return baseFilters
}

func GeneratePerfBaronLink(commitName string, testName string) (string, error) { // If empty parameters, then doesn't filter for commit or test
	cpFilters := createGoDriverContextFilters()
	if commitName != "" {
		updateFilterString(cpFilters, "commit", commitName)
	}
	if testName != "" {
		updateFilterString(cpFilters, "test", testName)
	}

	cpFiltersJSON, err := json.Marshal(cpFilters)
	if err != nil {
		return "", fmt.Errorf("failed to marshal change_point_filters to json: %v", err)
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse base URL: %v", err)
	}

	q := u.Query()
	q.Set("change_point_filters", string(cpFiltersJSON))
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func TestGeneratePerfBaronLink() {
	fmt.Println()
	// just mongo-go-driver, perf, perf
	link1, err1 := GeneratePerfBaronLink("", "")
	if err1 == nil {
		fmt.Println(link1)
	}
	// add commit
	link2, err2 := GeneratePerfBaronLink("50cf0c20d228975074c0010bfb688917e25934a4", "")
	if err2 == nil {
		fmt.Println(link2)
	}
	// add test
	link3, err3 := GeneratePerfBaronLink("", "BenchmarkMultiInsertLargeDocument")
	if err3 == nil {
		fmt.Println(link3)
	}
	// add both
	link4, err4 := GeneratePerfBaronLink("50cf0c20d228975074c0010bfb688917e25934a4", "BenchmarkMultiInsertLargeDocument")
	if err4 == nil {
		fmt.Println(link4)
	}
}
