// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

func TestBatches(t *testing.T) {
	t.Parallel()

	batches := &modelBatches{
		writeOps: make([]clientBulkWriteOp, 2),
	}
	batches.AdvanceBatches(3)
	size := batches.Size()
	assert.Equal(t, 0, size, "expected: %d, got: %d", 1, size)
}

func TestAppendBatchSequence(t *testing.T) {
	t.Parallel()

	newBatches := func(t *testing.T) *modelBatches {
		client, err := newClient()
		require.NoError(t, err, "NewClient error: %v", err)
		return &modelBatches{
			client: client,
			writeOps: []clientBulkWriteOp{
				{namespace: "ns0", model: nil},
				{namespace: "ns1", model: &ClientInsertOneModel{
					Document: bson.D{{"foo", 42}},
				}},
				{namespace: "ns2", model: &ClientReplaceOneModel{
					Filter:      bson.D{{"foo", "bar"}},
					Replacement: bson.D{{"foo", "baz"}},
				}},
				{namespace: "ns1", model: &ClientDeleteOneModel{
					Filter: bson.D{{"qux", "quux"}},
				}},
			},
			offset: 1,
			result: &ClientBulkWriteResult{
				Acknowledged: true,
			},
		}
	}
	t.Run("test appendBatches", func(t *testing.T) {
		t.Parallel()

		batches := newBatches(t)
		const limitBigEnough = 16_000
		n, _, err := batches.AppendBatchSequence(nil, 4, limitBigEnough)
		require.NoError(t, err, "AppendBatchSequence error: %v", err)
		require.Equal(t, 3, n, "expected %d appendings, got: %d", 3, n)

		_ = batches.cursorHandlers[0](&cursorInfo{Ok: true, Idx: 0}, nil)
		_ = batches.cursorHandlers[1](&cursorInfo{Ok: true, Idx: 1}, nil)
		_ = batches.cursorHandlers[2](&cursorInfo{Ok: true, Idx: 2}, nil)

		ins, ok := batches.result.InsertResults[1]
		assert.True(t, ok, "expected an insert results")
		assert.NotNil(t, ins.InsertedID, "expected an ID")

		_, ok = batches.result.UpdateResults[2]
		assert.True(t, ok, "expected an insert results")

		_, ok = batches.result.DeleteResults[3]
		assert.True(t, ok, "expected an delete results")
	})
	t.Run("test appendBatches with maxCount", func(t *testing.T) {
		t.Parallel()

		batches := newBatches(t)
		const limitBigEnough = 16_000
		n, _, err := batches.AppendBatchSequence(nil, 2, limitBigEnough)
		require.NoError(t, err, "AppendBatchSequence error: %v", err)
		require.Equal(t, 2, n, "expected %d appendings, got: %d", 2, n)

		_ = batches.cursorHandlers[0](&cursorInfo{Ok: true, Idx: 0}, nil)
		_ = batches.cursorHandlers[1](&cursorInfo{Ok: true, Idx: 1}, nil)

		ins, ok := batches.result.InsertResults[1]
		assert.True(t, ok, "expected an insert results")
		assert.NotNil(t, ins.InsertedID, "expected an ID")

		_, ok = batches.result.UpdateResults[2]
		assert.True(t, ok, "expected an insert results")

		_, ok = batches.result.DeleteResults[3]
		assert.False(t, ok, "expected an delete results")
	})
	t.Run("test appendBatches with totalSize", func(t *testing.T) {
		t.Parallel()

		batches := newBatches(t)
		const limit = 1200 // > ( 166 first two batches + 1000 overhead )
		n, _, err := batches.AppendBatchSequence(nil, 4, limit)
		require.NoError(t, err, "AppendBatchSequence error: %v", err)
		require.Equal(t, 2, n, "expected %d appendings, got: %d", 2, n)

		_ = batches.cursorHandlers[0](&cursorInfo{Ok: true, Idx: 0}, nil)
		_ = batches.cursorHandlers[1](&cursorInfo{Ok: true, Idx: 1}, nil)

		ins, ok := batches.result.InsertResults[1]
		assert.True(t, ok, "expected an insert results")
		assert.NotNil(t, ins.InsertedID, "expected an ID")

		_, ok = batches.result.UpdateResults[2]
		assert.True(t, ok, "expected an insert results")

		_, ok = batches.result.DeleteResults[3]
		assert.False(t, ok, "expected an delete results")
	})
}

func TestAppendBatchesNamespaceUUIDs(t *testing.T) {
	t.Parallel()

	uuid1 := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	uuid2 := []byte{17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}

	type batchResult struct {
		Ops    []bson.Raw
		NsInfo []bson.Raw
	}

	// decodeBatches runs AppendBatchArray and returns both the ops and nsInfo
	// arrays as slices of raw BSON documents.
	decodeBatches := func(t *testing.T, batches *modelBatches) batchResult {
		t.Helper()
		_, data, err := batches.AppendBatchArray(nil, 100, 16_000)
		require.NoError(t, err, "AppendBatchArray error: %v", err)

		// AppendBatchArray returns two concatenated array-element payloads ("ops"
		// then "nsInfo").  Wrap them in a document so bson.Unmarshal can parse them.
		idx, doc := bsoncore.AppendDocumentStart(nil)
		doc = append(doc, data...)
		doc, _ = bsoncore.AppendDocumentEnd(doc, idx)

		var result struct {
			Ops    []bson.Raw `bson:"ops"`
			NsInfo []bson.Raw `bson:"nsInfo"`
		}
		require.NoError(t, bson.Unmarshal(doc, &result), "unmarshal error")
		return batchResult{Ops: result.Ops, NsInfo: result.NsInfo}
	}

	// decodeNsInfo returns the namespace string and, if present, the raw UUID bytes
	// from a collectionUUID binary element (subtype is not checked here).
	decodeNsInfo := func(t *testing.T, raw bson.Raw) (ns string, uuid []byte) {
		t.Helper()
		var entry struct {
			Ns             string       `bson:"ns"`
			CollectionUUID *bson.Binary `bson:"collectionUUID"`
		}
		require.NoError(t, bson.Unmarshal(raw, &entry))
		if entry.CollectionUUID != nil {
			return entry.Ns, entry.CollectionUUID.Data
		}
		return entry.Ns, nil
	}

	// nsIdxFromOp returns the namespace index embedded in any op document
	// (the value of whichever of "insert"/"update"/"delete" is present).
	nsIdxFromOp := func(t *testing.T, raw bson.Raw) int {
		t.Helper()
		var op struct {
			Insert *int32 `bson:"insert"`
			Update *int32 `bson:"update"`
			Delete *int32 `bson:"delete"`
		}
		require.NoError(t, bson.Unmarshal(raw, &op))
		switch {
		case op.Insert != nil:
			return int(*op.Insert)
		case op.Update != nil:
			return int(*op.Update)
		case op.Delete != nil:
			return int(*op.Delete)
		default:
			t.Fatal("op has no insert/update/delete field")
			return -1
		}
	}

	client, err := newClient()
	require.NoError(t, err, "newClient error: %v", err)

	t.Run("UUID on single entry", func(t *testing.T) {
		t.Parallel()

		batches := &modelBatches{
			client: client,
			writeOps: []clientBulkWriteOp{
				{namespace: "db.coll", model: &ClientInsertOneModel{Document: bson.D{{"x", 1}}}, collectionUUID: uuid1},
			},
			result: &ClientBulkWriteResult{Acknowledged: true},
		}
		res := decodeBatches(t, batches)
		require.Len(t, res.NsInfo, 1)
		ns, uuid := decodeNsInfo(t, res.NsInfo[0])
		assert.Equal(t, "db.coll", ns)
		require.NotNil(t, uuid)
		assert.Equal(t, uuid1, uuid)
	})

	t.Run("no UUID set", func(t *testing.T) {
		t.Parallel()

		batches := &modelBatches{
			client: client,
			writeOps: []clientBulkWriteOp{
				{namespace: "db.coll", model: &ClientInsertOneModel{Document: bson.D{{"x", 1}}}},
			},
			result: &ClientBulkWriteResult{Acknowledged: true},
		}
		res := decodeBatches(t, batches)
		require.Len(t, res.NsInfo, 1)
		ns, uuid := decodeNsInfo(t, res.NsInfo[0])
		assert.Equal(t, "db.coll", ns)
		assert.Nil(t, uuid, "expected collectionUUID to be absent")
	})

	t.Run("mixed: some entries have UUID, some do not", func(t *testing.T) {
		t.Parallel()

		batches := &modelBatches{
			client: client,
			writeOps: []clientBulkWriteOp{
				{namespace: "db.with_uuid", model: &ClientInsertOneModel{Document: bson.D{{"x", 1}}}, collectionUUID: uuid1},
				{namespace: "db.no_uuid", model: &ClientInsertOneModel{Document: bson.D{{"x", 2}}}},
			},
			result: &ClientBulkWriteResult{Acknowledged: true},
		}
		res := decodeBatches(t, batches)
		require.Len(t, res.NsInfo, 2)

		ns0, uuid0 := decodeNsInfo(t, res.NsInfo[0])
		assert.Equal(t, "db.with_uuid", ns0)
		require.NotNil(t, uuid0)
		assert.Equal(t, uuid1, uuid0)

		ns1, uuid1Got := decodeNsInfo(t, res.NsInfo[1])
		assert.Equal(t, "db.no_uuid", ns1)
		assert.Nil(t, uuid1Got, "expected collectionUUID to be absent for db.no_uuid")
	})

	t.Run("same namespace string, different UUIDs → two nsInfo entries, each op points to its UUID", func(t *testing.T) {
		t.Parallel()

		// Simulates two collections with the same db.collection name but different
		// UUIDs (e.g. the collection was dropped and recreated between events).
		batches := &modelBatches{
			client: client,
			writeOps: []clientBulkWriteOp{
				{namespace: "db.coll", model: &ClientInsertOneModel{Document: bson.D{{"x", 1}}}, collectionUUID: uuid1},
				{namespace: "db.coll", model: &ClientInsertOneModel{Document: bson.D{{"x", 2}}}, collectionUUID: uuid2},
			},
			result: &ClientBulkWriteResult{Acknowledged: true},
		}
		res := decodeBatches(t, batches)

		// Verify nsInfo: one entry per UUID, both with the same namespace string.
		require.Len(t, res.NsInfo, 2, "expected one nsInfo entry per distinct UUID")
		ns0, u0 := decodeNsInfo(t, res.NsInfo[0])
		assert.Equal(t, "db.coll", ns0)
		require.NotNil(t, u0)
		assert.Equal(t, uuid1, u0)
		ns1, u1 := decodeNsInfo(t, res.NsInfo[1])
		assert.Equal(t, "db.coll", ns1)
		require.NotNil(t, u1)
		assert.Equal(t, uuid2, u1)

		// Verify each op's namespace index correctly references its UUID's nsInfo entry.
		require.Len(t, res.Ops, 2)
		assert.Equal(t, 0, nsIdxFromOp(t, res.Ops[0]), "op[0] should reference nsInfo[0] (uuid1)")
		assert.Equal(t, 1, nsIdxFromOp(t, res.Ops[1]), "op[1] should reference nsInfo[1] (uuid2)")
	})

	t.Run("same namespace and UUID → deduplicated nsInfo", func(t *testing.T) {
		t.Parallel()

		batches := &modelBatches{
			client: client,
			writeOps: []clientBulkWriteOp{
				{namespace: "db.coll", model: &ClientInsertOneModel{Document: bson.D{{"x", 1}}}, collectionUUID: uuid1},
				{namespace: "db.coll", model: &ClientInsertOneModel{Document: bson.D{{"x", 2}}}, collectionUUID: uuid1},
			},
			result: &ClientBulkWriteResult{Acknowledged: true},
		}
		res := decodeBatches(t, batches)
		require.Len(t, res.NsInfo, 1, "expected one nsInfo entry for same namespace+UUID")

		ns, uuid := decodeNsInfo(t, res.NsInfo[0])
		assert.Equal(t, "db.coll", ns)
		require.NotNil(t, uuid)
		assert.Equal(t, uuid1, uuid)
	})
}
