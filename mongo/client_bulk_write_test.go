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
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/wiremessage"
)

func TestBatches(t *testing.T) {
	t.Parallel()

	batches := &modelBatches{
		writePairs: make([]clientBulkWritePair, 2),
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
			writePairs: []clientBulkWritePair{
				{namespace: "ns0"},
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

// nsInfoEntry is a decoded nsInfo document. UUID is nil when the entry has no
// collectionUUID element.
type nsInfoEntry struct {
	Namespace string       `bson:"ns"`
	UUID      *bson.Binary `bson:"collectionUUID"`
}

// readBatch decodes the output of AppendBatchSequence into its nsInfo entries
// and the namespace index each op points at. It walks the OP_MSG document
// sequences the same way the driver does for command monitoring.
func readBatch(t *testing.T, dst []byte) ([]nsInfoEntry, []int32) {
	t.Helper()

	var nsInfo []nsInfoEntry
	var nsIdxs []int32
	for len(dst) > 0 {
		stype, data, ok := wiremessage.ReadMsgSectionType(dst)
		require.True(t, ok, "expected a section type")
		require.Equal(t, wiremessage.DocumentSequence, stype, "expected a document sequence")

		var identifier string
		identifier, data, dst, ok = wiremessage.ReadMsgSectionRawDocumentSequence(data)
		require.True(t, ok, "expected a raw document sequence")

		for len(data) > 0 {
			var doc bsoncore.Document
			doc, data, ok = bsoncore.ReadDocument(data)
			require.True(t, ok, "expected a document in %q", identifier)

			switch identifier {
			case "nsInfo":
				var entry nsInfoEntry
				err := bson.Unmarshal(doc, &entry)
				require.NoError(t, err, "Unmarshal error: %v", err)
				nsInfo = append(nsInfo, entry)
			case "ops":
				// The first element of an op names its type and holds its
				// namespace index.
				elem, err := doc.IndexErr(0)
				require.NoError(t, err, "IndexErr error: %v", err)
				nsIdxs = append(nsIdxs, elem.Value().Int32())
			}
		}
	}
	return nsInfo, nsIdxs
}

func TestAppendBatchSequenceCollectionUUIDs(t *testing.T) {
	uuid1 := &bson.Binary{Subtype: bson.TypeBinaryUUID, Data: []byte("0123456789abcdef")}
	uuid2 := &bson.Binary{Subtype: bson.TypeBinaryUUID, Data: []byte("fedcba9876543210")}
	empty := &bson.Binary{Subtype: bson.TypeBinaryUUID, Data: []byte{}}

	// pair builds an insert into ns. The _id is fixed so that rebuilding a
	// batch produces the same bytes.
	pair := func(ns string, uuid *bson.Binary) clientBulkWritePair {
		p := clientBulkWritePair{
			namespace: ns,
			model:     &ClientInsertOneModel{Document: bson.D{{"_id", 1}}},
		}
		if uuid != nil {
			p.collectionUUID = uuid.Data
		}
		return p
	}

	newBatches := func(t *testing.T, pairs ...clientBulkWritePair) *modelBatches {
		t.Helper()

		client, err := newClient()
		require.NoError(t, err, "newClient error: %v", err)
		return &modelBatches{
			client:     client,
			writePairs: pairs,
			result:     &ClientBulkWriteResult{Acknowledged: true},
		}
	}

	const limitBigEnough = 16_000

	testCases := []struct {
		name       string
		pairs      []clientBulkWritePair
		wantNsInfo []nsInfoEntry
		wantNsIdxs []int32
	}{
		{
			name:       "no UUIDs",
			pairs:      []clientBulkWritePair{pair("db.c0", nil), pair("db.c1", nil), pair("db.c0", nil)},
			wantNsInfo: []nsInfoEntry{{Namespace: "db.c0"}, {Namespace: "db.c1"}},
			wantNsIdxs: []int32{0, 1, 0},
		},
		{
			name:       "same namespace different UUIDs",
			pairs:      []clientBulkWritePair{pair("db.c0", uuid1), pair("db.c0", uuid2)},
			wantNsInfo: []nsInfoEntry{{Namespace: "db.c0", UUID: uuid1}, {Namespace: "db.c0", UUID: uuid2}},
			wantNsIdxs: []int32{0, 1},
		},
		{
			name:       "same namespace same UUID",
			pairs:      []clientBulkWritePair{pair("db.c0", uuid1), pair("db.c0", uuid1)},
			wantNsInfo: []nsInfoEntry{{Namespace: "db.c0", UUID: uuid1}},
			wantNsIdxs: []int32{0, 0},
		},
		{
			name:       "same namespace with and without a UUID",
			pairs:      []clientBulkWritePair{pair("db.c0", uuid1), pair("db.c0", nil)},
			wantNsInfo: []nsInfoEntry{{Namespace: "db.c0", UUID: uuid1}, {Namespace: "db.c0"}},
			wantNsIdxs: []int32{0, 1},
		},
		{
			name:       "empty UUID is distinct from an unset one",
			pairs:      []clientBulkWritePair{pair("db.c0", nil), pair("db.c0", empty)},
			wantNsInfo: []nsInfoEntry{{Namespace: "db.c0"}, {Namespace: "db.c0", UUID: empty}},
			wantNsIdxs: []int32{0, 1},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			batches := newBatches(t, tc.pairs...)
			n, dst, err := batches.AppendBatchSequence(nil, len(tc.pairs), limitBigEnough)
			require.NoError(t, err, "AppendBatchSequence error: %v", err)
			require.Equal(t, len(tc.pairs), n, "expected %d appendings, got %d", len(tc.pairs), n)

			nsInfo, nsIdxs := readBatch(t, dst)
			assert.Equal(t, tc.wantNsInfo, nsInfo, "unexpected nsInfo")
			assert.Equal(t, tc.wantNsIdxs, nsIdxs, "unexpected op namespace indexes")

			// A retry rebuilds the message without advancing the offset, so it
			// must be byte-identical to the first build.
			_, retry, err := batches.AppendBatchSequence(nil, len(tc.pairs), limitBigEnough)
			require.NoError(t, err, "AppendBatchSequence error: %v", err)
			assert.Equal(t, dst, retry, "expected a retry to rebuild an identical message")
		})
	}

	t.Run("batch split restarts namespace indexes", func(t *testing.T) {
		batches := newBatches(t, pair("db.c0", uuid1), pair("db.c1", nil), pair("db.c0", uuid2))

		n, dst, err := batches.AppendBatchSequence(nil, 2, limitBigEnough)
		require.NoError(t, err, "AppendBatchSequence error: %v", err)
		require.Equal(t, 2, n, "expected %d appendings, got %d", 2, n)
		nsInfo, nsIdxs := readBatch(t, dst)
		assert.Equal(t, []nsInfoEntry{{Namespace: "db.c0", UUID: uuid1}, {Namespace: "db.c1"}}, nsInfo,
			"unexpected nsInfo in the first batch")
		assert.Equal(t, []int32{0, 1}, nsIdxs, "unexpected indexes in the first batch")

		batches.AdvanceBatches(n)

		// The second batch builds its own namespace map, so the remaining write
		// to db.c0 is index 0 again, now with the second UUID.
		n, dst, err = batches.AppendBatchSequence(nil, 2, limitBigEnough)
		require.NoError(t, err, "AppendBatchSequence error: %v", err)
		require.Equal(t, 1, n, "expected %d appendings, got %d", 1, n)
		nsInfo, nsIdxs = readBatch(t, dst)
		assert.Equal(t, []nsInfoEntry{{Namespace: "db.c0", UUID: uuid2}}, nsInfo,
			"unexpected nsInfo in the second batch")
		assert.Equal(t, []int32{0}, nsIdxs, "unexpected indexes in the second batch")
	})
}
