// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package streamprocessing_test contains env-gated integration tests for the
// streamprocessing/ client. These tests do not run unless
// MONGODB_STREAM_PROCESSING_URI is set, since they require a real Atlas
// Stream Processing workspace.
package streamprocessing_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/internal/uuid"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/streamprocessing"
)

const workspaceURIEnv = "MONGODB_STREAM_PROCESSING_URI"

func skipIfNoWorkspaceURI(t *testing.T) string {
	t.Helper()
	uri := os.Getenv(workspaceURIEnv)
	if uri == "" {
		t.Skipf("%s is not set; skipping ASP integration test", workspaceURIEnv)
	}
	return uri
}

func newClient(t *testing.T, uri string) *streamprocessing.Client {
	t.Helper()
	c, err := streamprocessing.Connect(options.Client().ApplyURI(uri))
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = c.Disconnect(ctx)
	})
	return c
}

func uniqueProcessorName(t *testing.T) string {
	t.Helper()
	id, err := uuid.New()
	require.NoError(t, err)
	return fmt.Sprintf("driver-test-%s", id)
}

// TestEndToEnd runs the full lifecycle of a stream processor against a real
// workspace endpoint.
func TestEndToEnd(t *testing.T) {
	uri := skipIfNoWorkspaceURI(t)
	client := newClient(t, uri)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	sps := client.StreamProcessors()
	name := uniqueProcessorName(t)
	t.Logf("using processor name %q", name)

	pipeline := []bson.D{
		{{Key: "$source", Value: bson.D{{Key: "connectionName", Value: "sample_stream_solar"}}}},
	}

	require.NoError(t, sps.Create(ctx, name, pipeline))
	t.Cleanup(func() {
		dropCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = sps.Get(name).Drop(dropCtx)
	})

	info, err := sps.GetInfo(ctx, name)
	require.NoError(t, err)
	assert.Equal(t, name, info.Name)

	sp := sps.Get(name)
	require.NoError(t, sp.Start(ctx, nil))

	// Wait briefly for the processor to start producing samples.
	time.Sleep(2 * time.Second)

	stats, err := sp.Stats(ctx, nil)
	require.NoError(t, err)
	require.Greater(t, len(stats), 0)

	samples, err := sp.GetStreamProcessorSamples(ctx, options.GetStreamProcessorSamples().SetLimit(10))
	require.NoError(t, err)
	t.Logf("first sample batch: cursor=%d documents=%d", samples.CursorID, len(samples.Documents))

	if samples.CursorID != 0 {
		next, err := sp.GetStreamProcessorSamples(ctx,
			options.GetStreamProcessorSamples().SetCursorID(samples.CursorID).SetBatchSize(5))
		require.NoError(t, err)
		t.Logf("second sample batch: cursor=%d documents=%d", next.CursorID, len(next.Documents))
	}

	require.NoError(t, sp.Stop(ctx))
	require.NoError(t, sp.Drop(ctx))
}
