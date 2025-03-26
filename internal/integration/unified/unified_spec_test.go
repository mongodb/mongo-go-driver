// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"context"
	"strings"
	"testing"
)

var (
	passDirectories = []string{
		//spectest.TestPath(3, "unified-test-format", "valid-pass"),
		//spectest.TestPath(3, "versioned-api"),
		//spectest.TestPath(3, "crud", "unified"),
		//spectest.TestPath(3, "change-streams", "unified"),
		//spectest.TestPath(3, "transactions", "unified"),
		//
		//"load-balancers",
		//
		//spectest.TestPath(3, "collection-management"),
		//"command-monitoring",
		//"command-monitoring/logging",
		//"connection-monitoring-and-pooling/logging",
		//"sessions",
		//"retryable-reads/unified",
		//"retryable-writes/unified",
		//"client-side-encryption/unified",
		//"client-side-operations-timeout",
		//"gridfs",
		//"server-selection/logging",
		//"server-discovery-and-monitoring/unified",
		//"run-command",
		//"index-management",
	}
	failDirectories = []string{
		//"unified-test-format/valid-fail",
	}
)

//const (
//	dataDirectory = "../../../testdata"
//)

func TestUnifiedSpec(t *testing.T) {
	// Ensure the cluster is in a clean state before test execution begins.
	if err := terminateOpenSessions(context.Background()); err != nil {
		t.Fatalf("error terminating open transactions: %v", err)
	}

	for _, testDir := range passDirectories {
		index := strings.Index(testDir, "testdata/")
		testName := testDir[index+len("testdata/"):]

		t.Run(testName, func(t *testing.T) {
			runTestDirectory(t, testDir, false)
		})
	}

	for _, testDir := range failDirectories {
		t.Run(testDir, func(t *testing.T) {
			runTestDirectory(t, testDir, true)
		})
	}
}
