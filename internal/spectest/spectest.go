// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package spectest

import (
	"io/ioutil"
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

	entries, err := ioutil.ReadDir(dir)
	require.NoError(t, err)

	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".json" {
			continue
		}

		files = append(files, entry.Name())
	}

	return files
}

func TestPath(depth int, testDir string, subDirs ...string) string {
	const basePath = "testdata/specifications/source/"

	// Create a string of "../" repeated 'depth' times
	relativePath := strings.Repeat("../", depth)
	// Construct the full path
	fullPath := relativePath + basePath + testDir + "/tests/" + filepath.Join(subDirs...)

	return fullPath
}
