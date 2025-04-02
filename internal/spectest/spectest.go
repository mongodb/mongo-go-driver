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
	"runtime"
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

	if len(files) == 0 {
		t.Fatalf("no JSON files found in %q", dir)
	}

	return files
}

// Path returns the absolute path to the given specifications repo file or
// subdirectory.
func Path(subdir string) string {
	root := os.Getenv("PROJECT_DIRECTORY")
	if root == "" {
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			panic("unable to get current file path from call stack")
		}

		// Get the repository root path from the current Go file path.
		root = filepath.Dir(filepath.Dir(filepath.Dir(file)))
	}

	return filepath.Join(root, "testdata", "specifications", "source", subdir)
}
