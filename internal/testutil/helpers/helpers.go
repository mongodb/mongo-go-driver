// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package helpers

import (
	"fmt"
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

// RawToDocuments converts a bson.Raw that is internally an array of documents to []bson.Raw.
func RawToDocuments(doc bson.Raw) []bson.Raw {
	values, err := doc.Values()
	if err != nil {
		panic(fmt.Sprintf("error converting BSON document to values: %v", err))
	}

	out := make([]bson.Raw, len(values))
	for i := range values {
		out[i] = values[i].Document()
	}

	return out
}

// RawToInterfaces takes one or many bson.Raw documents and returns them as a []interface{}.
func RawToInterfaces(docs ...bson.Raw) []interface{} {
	out := make([]interface{}, len(docs))
	for i := range docs {
		out[i] = docs[i]
	}
	return out
}
