// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"testing"
)

const dataPath string = "../../data/"

var directories = []string{
	"transactions",
	"convenient-transactions",
	"crud/v2",
	"retryable-reads",
	"sessions",
	"read-write-concern/operation",
	"server-discovery-and-monitoring/integration",
	"atlas-data-lake-testing",
}

func TestUnifiedSpecs(t *testing.T) {
	for _, specDir := range directories {
		t.Run(specDir, func(t *testing.T) {
			RunSpecTestDirectory(t, dataPath, specDir)
		})
	}
}
