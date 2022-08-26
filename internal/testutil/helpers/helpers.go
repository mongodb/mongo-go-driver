// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package helpers

import (
	"io/ioutil"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
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

// RawToInterfaceSlice converts a bson.Raw that is internally an array to []interface{}.
func RawToInterfaceSlice(doc bson.Raw) []interface{} {
	values, _ := doc.Values()

	out := make([]interface{}, 0, len(values))
	for _, val := range values {
		out = append(out, val.Document())
	}

	return out
}
