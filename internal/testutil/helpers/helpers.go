// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package helpers

import (
	"context"
	"fmt"
	"io/ioutil"
	"path"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/require"
)

// AssertSoon runs the provided callback and fails the passed-in test if the callback
// does not complete within timeout. The provided callback should respect the
// passed-in context and cease execution when it has expired.
//
// Deprecated: This function will be removed with GODRIVER-2667, use assert.Eventually
// instead.
func AssertSoon(t testing.TB, callback func(ctx context.Context), timeout time.Duration) {
	t.Helper()

	// Create context to manually cancel callback after Soon assertion.
	callbackCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	fullCallback := func() {
		callback(callbackCtx)
		done <- struct{}{}
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	go fullCallback()

	select {
	case <-done:
		return
	case <-timer.C:
		t.Fatalf("timed out in %s waiting for callback", timeout)
	}
}

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
