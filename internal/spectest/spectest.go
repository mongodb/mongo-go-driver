// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package spectest

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/require"
)

// FindJSONFilesInDir finds the JSON files in a directory.
func FindJSONFilesInDir(t *testing.T, dir string) []string {
	t.Helper()

	files := make([]string, 0)

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".json" {
			continue
		}

		files = append(files, entry.Name())
	}

	return files
}

// TestPath will construct a path to a test file in the specifications git
// submodule. The path will be relative to the current package so a depth should
// be provided to indicate how many directories to go up.
func TestPath(depth int, testDir string, subDirs ...string) string {
	const basePath = "testdata/specifications/source/"

	// Create a string of "../" repeated 'depth' times
	relativePath := strings.Repeat("../", depth)
	// Construct the full path
	fullPath := filepath.Join(relativePath, basePath, testDir, "tests", filepath.Join(subDirs...))

	return fullPath
}
