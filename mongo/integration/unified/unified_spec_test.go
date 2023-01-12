// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"context"
	"path"
	"testing"
)

var (
	passDirectories = []string{
		"unified-test-format/valid-pass",
		"versioned-api",
		"crud/unified",
		"change-streams",
		"transactions/unified",
		"load-balancers",
		"collection-management",
		"command-monitoring",
		"sessions",
		"retryable-writes/unified",
		"client-side-encryption/unified",
		"client-side-operations-timeout",
		"gridfs",
	}
	failDirectories = []string{
		"unified-test-format/valid-fail",
	}
)

const (
	dataDirectory = "../../../testdata"
)

func TestUnifiedSpec(t *testing.T) {
	// Ensure the cluster is in a clean state before test execution begins.
	if err := terminateOpenSessions(context.Background()); err != nil {
		t.Fatalf("error terminating open transactions: %v", err)
	}

	for _, testDir := range passDirectories {
		t.Run(testDir, func(t *testing.T) {
			runTestDirectory(t, path.Join(dataDirectory, testDir), false)
		})
	}

	for _, testDir := range failDirectories {
		t.Run(testDir, func(t *testing.T) {
			runTestDirectory(t, path.Join(dataDirectory, testDir), true)
		})
	}
}
