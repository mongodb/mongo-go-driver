// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/spectest"
)

var (
	passDirectories = []string{
		"unified-test-format/tests/valid-pass",
		"versioned-api/tests",
		"crud/tests/unified",
		"change-streams/tests/unified",
		"transactions/tests/unified",
		"load-balancers/tests",
		"collection-management/tests",
		"command-logging-and-monitoring/tests/monitoring",
		"command-logging-and-monitoring/tests/logging",
		"connection-monitoring-and-pooling/tests/logging",
		"sessions/tests",
		"retryable-reads/tests/unified",
		"retryable-writes/tests/unified",
		"client-side-encryption/tests/unified",
		"client-side-operations-timeout/tests",
		"gridfs/tests",
		"server-selection/tests/logging",
		"server-discovery-and-monitoring/tests/unified",
		"run-command/tests/unified",
		"index-management/tests",
		"transactions-convenient-api/tests/unified",
		"atlas-data-lake-testing/tests/unified",
	}
	failDirectories = []string{
		"unified-test-format/tests/valid-fail",
	}
)

func TestUnifiedSpec(t *testing.T) {
	// Ensure the cluster is in a clean state before test execution begins.
	if err := terminateOpenSessions(context.Background()); err != nil {
		t.Fatalf("error terminating open transactions: %v", err)
	}

	for _, testDir := range passDirectories {
		t.Run(testDir, func(t *testing.T) {
			runTestDirectory(t, spectest.Path(testDir), false)
		})
	}

	for _, testDir := range failDirectories {
		t.Run(testDir, func(t *testing.T) {
			runTestDirectory(t, spectest.Path(testDir), true)
		})
	}
}
