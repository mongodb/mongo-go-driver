// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package operation

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
)

// buildSPCommand wraps op.command with the BSON document framing so the
// returned byte slice is a valid root-level BSON document.
func buildSPCommand(t *testing.T, build func(dst []byte) ([]byte, error)) bsoncore.Document {
	t.Helper()
	idx, dst := bsoncore.AppendDocumentStart(nil)
	dst, err := build(dst)
	require.NoError(t, err)
	dst, err = bsoncore.AppendDocumentEnd(dst, idx)
	require.NoError(t, err)
	doc, _, ok := bsoncore.ReadDocument(dst)
	require.True(t, ok)
	return doc
}

func mustBSON(t *testing.T, d bson.D) bsoncore.Document {
	t.Helper()
	raw, err := bson.Marshal(d)
	require.NoError(t, err)
	return raw
}

func TestCreateStreamProcessor_Command(t *testing.T) {
	t.Run("minimal", func(t *testing.T) {
		stage := mustBSON(t, bson.D{{Key: "$source", Value: bson.D{{Key: "connectionName", Value: "kafka"}}}})
		arrIdx, arr := bsoncore.AppendArrayStart(nil)
		arr = bsoncore.AppendDocumentElement(arr, "0", stage)
		arr, err := bsoncore.AppendArrayEnd(arr, arrIdx)
		require.NoError(t, err)
		op := NewCreateStreamProcessor("proc1", bsoncore.Document(arr))

		got := buildSPCommand(t, func(dst []byte) ([]byte, error) {
			return op.command(dst, description.SelectedServer{})
		})

		// Verify the command name comes first and pipeline is included.
		gotD := bson.D{}
		require.NoError(t, bson.Unmarshal(got, &gotD))
		require.GreaterOrEqual(t, len(gotD), 2)
		assert.Equal(t, "createStreamProcessor", gotD[0].Key)
		assert.Equal(t, "proc1", gotD[0].Value)
		assert.Equal(t, "pipeline", gotD[1].Key)
	})

	t.Run("with options", func(t *testing.T) {
		dlq := mustBSON(t, bson.D{{Key: "connectionName", Value: "lostMessages"}})
		op := NewCreateStreamProcessor("proc2", nil).
			DLQ(dlq).
			StreamMetaFieldName("_stream").
			Tier("SP10").
			FailoverEnabled(true)

		got := buildSPCommand(t, func(dst []byte) ([]byte, error) {
			return op.command(dst, description.SelectedServer{})
		})

		gotD := bson.D{}
		require.NoError(t, bson.Unmarshal(got, &gotD))

		// First key is the command name.
		assert.Equal(t, "createStreamProcessor", gotD[0].Key)
		assert.Equal(t, "proc2", gotD[0].Value)

		// Find the options sub-doc.
		var opts bson.D
		for _, e := range gotD {
			if e.Key == "options" {
				opts = e.Value.(bson.D)
			}
		}
		require.NotNil(t, opts)
		seen := map[string]any{}
		for _, e := range opts {
			seen[e.Key] = e.Value
		}
		assert.Equal(t, "_stream", seen["streamMetaFieldName"])
		assert.Equal(t, "SP10", seen["tier"])
		assert.Equal(t, true, seen["failover"])
		_, hasDLQ := seen["dlq"]
		assert.True(t, hasDLQ, "expected dlq in options")
	})
}

func TestStartStreamProcessor_Command(t *testing.T) {
	t.Run("minimal", func(t *testing.T) {
		op := NewStartStreamProcessor("proc1")
		got := buildSPCommand(t, func(dst []byte) ([]byte, error) {
			return op.command(dst, description.SelectedServer{})
		})
		gotD := bson.D{}
		require.NoError(t, bson.Unmarshal(got, &gotD))
		require.Len(t, gotD, 1)
		assert.Equal(t, "startStreamProcessor", gotD[0].Key)
		assert.Equal(t, "proc1", gotD[0].Value)
	})

	t.Run("with failover and options", func(t *testing.T) {
		op := NewStartStreamProcessor("proc1").
			Workers(4).
			ClearCheckpoints(true).
			Tier("SP30").
			EnableAutoScaling(true).
			FailoverRegion("us-east-1").
			FailoverMode("GRACEFUL").
			FailoverDryRun(false)
		got := buildSPCommand(t, func(dst []byte) ([]byte, error) {
			return op.command(dst, description.SelectedServer{})
		})

		gotD := bson.D{}
		require.NoError(t, bson.Unmarshal(got, &gotD))

		keys := make(map[string]any)
		for _, e := range gotD {
			keys[e.Key] = e.Value
		}
		assert.Equal(t, "proc1", keys["startStreamProcessor"])
		assert.Equal(t, int32(4), keys["workers"])

		opts, ok := keys["options"].(bson.D)
		require.True(t, ok)
		optsMap := map[string]any{}
		for _, e := range opts {
			optsMap[e.Key] = e.Value
		}
		assert.Equal(t, true, optsMap["clearCheckpoints"])
		assert.Equal(t, "SP30", optsMap["tier"])
		assert.Equal(t, true, optsMap["enableAutoScaling"])

		fail, ok := keys["failover"].(bson.D)
		require.True(t, ok)
		failMap := map[string]any{}
		for _, e := range fail {
			failMap[e.Key] = e.Value
		}
		assert.Equal(t, "us-east-1", failMap["region"])
		assert.Equal(t, "GRACEFUL", failMap["mode"])
		assert.Equal(t, false, failMap["dryRun"])
	})

	t.Run("failover without region errors", func(t *testing.T) {
		op := NewStartStreamProcessor("proc1").FailoverMode("FORCED")
		_, err := op.command(nil, description.SelectedServer{})
		require.Error(t, err)
	})

	t.Run("startAtOperationTime is serialized as timestamp", func(t *testing.T) {
		op := NewStartStreamProcessor("proc1").StartAtOperationTime(42, 7)
		got := buildSPCommand(t, func(dst []byte) ([]byte, error) {
			return op.command(dst, description.SelectedServer{})
		})
		gotD := bson.D{}
		require.NoError(t, bson.Unmarshal(got, &gotD))
		var opts bson.D
		for _, e := range gotD {
			if e.Key == "options" {
				opts = e.Value.(bson.D)
			}
		}
		require.NotNil(t, opts)
		var ts bson.Timestamp
		for _, e := range opts {
			if e.Key == "startAtOperationTime" {
				ts = e.Value.(bson.Timestamp)
			}
		}
		assert.Equal(t, uint32(42), ts.T)
		assert.Equal(t, uint32(7), ts.I)
	})
}

func TestStopStreamProcessor_Command(t *testing.T) {
	op := NewStopStreamProcessor("proc1")
	got := buildSPCommand(t, func(dst []byte) ([]byte, error) {
		return op.command(dst, description.SelectedServer{})
	})
	gotD := bson.D{}
	require.NoError(t, bson.Unmarshal(got, &gotD))
	require.Len(t, gotD, 1)
	assert.Equal(t, "stopStreamProcessor", gotD[0].Key)
	assert.Equal(t, "proc1", gotD[0].Value)
}

func TestDropStreamProcessor_Command(t *testing.T) {
	op := NewDropStreamProcessor("proc1")
	got := buildSPCommand(t, func(dst []byte) ([]byte, error) {
		return op.command(dst, description.SelectedServer{})
	})
	gotD := bson.D{}
	require.NoError(t, bson.Unmarshal(got, &gotD))
	require.Len(t, gotD, 1)
	assert.Equal(t, "dropStreamProcessor", gotD[0].Key)
	assert.Equal(t, "proc1", gotD[0].Value)
}

func TestGetStreamProcessor_Command(t *testing.T) {
	op := NewGetStreamProcessor("proc1")
	got := buildSPCommand(t, func(dst []byte) ([]byte, error) {
		return op.command(dst, description.SelectedServer{})
	})
	gotD := bson.D{}
	require.NoError(t, bson.Unmarshal(got, &gotD))
	require.Len(t, gotD, 1)
	assert.Equal(t, "getStreamProcessor", gotD[0].Key)
	assert.Equal(t, "proc1", gotD[0].Value)
}

func TestGetStreamProcessorStats_Command(t *testing.T) {
	t.Run("minimal", func(t *testing.T) {
		op := NewGetStreamProcessorStats("proc1")
		got := buildSPCommand(t, func(dst []byte) ([]byte, error) {
			return op.command(dst, description.SelectedServer{})
		})
		gotD := bson.D{}
		require.NoError(t, bson.Unmarshal(got, &gotD))
		require.Len(t, gotD, 1)
		assert.Equal(t, "getStreamProcessorStats", gotD[0].Key)
	})

	t.Run("verbose", func(t *testing.T) {
		op := NewGetStreamProcessorStats("proc1").Verbose(true)
		got := buildSPCommand(t, func(dst []byte) ([]byte, error) {
			return op.command(dst, description.SelectedServer{})
		})
		gotD := bson.D{}
		require.NoError(t, bson.Unmarshal(got, &gotD))
		var opts bson.D
		for _, e := range gotD {
			if e.Key == "options" {
				opts = e.Value.(bson.D)
			}
		}
		require.NotNil(t, opts)
		assert.Equal(t, "verbose", opts[0].Key)
		assert.Equal(t, true, opts[0].Value)
	})
}

func TestStartSampleStreamProcessor_Command(t *testing.T) {
	t.Run("no limit", func(t *testing.T) {
		op := NewStartSampleStreamProcessor("proc1")
		got := buildSPCommand(t, func(dst []byte) ([]byte, error) {
			return op.command(dst, description.SelectedServer{})
		})
		gotD := bson.D{}
		require.NoError(t, bson.Unmarshal(got, &gotD))
		require.Len(t, gotD, 1)
		assert.Equal(t, "startSampleStreamProcessor", gotD[0].Key)
	})

	t.Run("with limit", func(t *testing.T) {
		op := NewStartSampleStreamProcessor("proc1").Limit(100)
		got := buildSPCommand(t, func(dst []byte) ([]byte, error) {
			return op.command(dst, description.SelectedServer{})
		})
		gotD := bson.D{}
		require.NoError(t, bson.Unmarshal(got, &gotD))
		require.Len(t, gotD, 2)
		assert.Equal(t, "startSampleStreamProcessor", gotD[0].Key)
		assert.Equal(t, "limit", gotD[1].Key)
		assert.Equal(t, int32(100), gotD[1].Value)
	})
}

func TestGetMoreSampleStreamProcessor_Command(t *testing.T) {
	op := NewGetMoreSampleStreamProcessor("proc1", 42).BatchSize(50)
	got := buildSPCommand(t, func(dst []byte) ([]byte, error) {
		return op.command(dst, description.SelectedServer{})
	})
	gotD := bson.D{}
	require.NoError(t, bson.Unmarshal(got, &gotD))
	require.Len(t, gotD, 3)
	assert.Equal(t, "getMoreSampleStreamProcessor", gotD[0].Key)
	assert.Equal(t, "proc1", gotD[0].Value)
	assert.Equal(t, "cursorId", gotD[1].Key)
	assert.Equal(t, int64(42), gotD[1].Value)
	assert.Equal(t, "batchSize", gotD[2].Key)
	assert.Equal(t, int32(50), gotD[2].Value)
}

func TestStartSampleStreamProcessor_ParseResponse(t *testing.T) {
	resp := mustBSON(t, bson.D{{Key: "ok", Value: 1.0}, {Key: "cursorId", Value: int64(99)}})
	op := NewStartSampleStreamProcessor("proc1")
	require.NoError(t, op.processResponse(nil, resp, driver.ResponseInfo{}))
	assert.Equal(t, int64(99), op.CursorID())
}

func TestGetMoreSampleStreamProcessor_ParseResponse(t *testing.T) {
	doc1 := mustBSON(t, bson.D{{Key: "x", Value: 1}})
	doc2 := mustBSON(t, bson.D{{Key: "x", Value: 2}})
	resp := mustBSON(t, bson.D{
		{Key: "ok", Value: 1.0},
		{Key: "cursorId", Value: int64(0)}, // exhausted
		{Key: "nextBatch", Value: bson.A{
			bson.Raw(doc1),
			bson.Raw(doc2),
		}},
	})
	op := NewGetMoreSampleStreamProcessor("proc1", 42)
	require.NoError(t, op.processResponse(nil, resp, driver.ResponseInfo{}))
	assert.Equal(t, int64(0), op.ResultCursorID())
	require.Len(t, op.ResultBatch(), 2)
}
