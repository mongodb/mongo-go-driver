// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/require"
)

func TestBatches(t *testing.T) {
	t.Run("test Addvancing", func(t *testing.T) {
		batches := &modelBatches{
			models: make([]clientWriteModel, 2),
		}
		batches.AdvanceBatches(3)
		size := batches.Size()
		assert.Equal(t, 0, size, "expected: %d, got: %d", 1, size)
	})
	t.Run("test appendBatches", func(t *testing.T) {
		client, err := NewClient()
		require.NoError(t, err, "NewClient error: %v", err)
		batches := &modelBatches{
			client: client,
			models: []clientWriteModel{
				{"ns0", nil},
				{"ns1", &ClientInsertOneModel{
					Document: bson.D{{"foo", 42}},
				}},
				{"ns2", &ClientReplaceOneModel{
					Filter:      bson.D{{"foo", "bar"}},
					Replacement: bson.D{{"foo", "baz"}},
				}},
				{"ns1", &ClientDeleteOneModel{
					Filter: bson.D{{"qux", "quux"}},
				}},
			},
			offset: 1,
			result: &ClientBulkWriteResult{},
		}
		var n int
		n, _, err = batches.AppendBatchSequence(nil, 4, 16_000, 16_000)
		require.NoError(t, err, "AppendBatchSequence error: %v", err)
		assert.Equal(t, 3, n, "expected %d appendings, got: %d", 3, n)

		_ = batches.cursorHandlers[0](&cursorInfo{Ok: true, Idx: 0}, nil)
		_ = batches.cursorHandlers[1](&cursorInfo{Ok: true, Idx: 1}, nil)
		_ = batches.cursorHandlers[2](&cursorInfo{Ok: true, Idx: 2}, nil)

		ins, ok := batches.result.InsertResults[1]
		assert.True(t, ok, "expected an insert results")
		assert.NotNil(t, ins.InsertedID, "expected an ID")

		_, ok = batches.result.UpdateResults[2]
		assert.True(t, ok, "expected an insert results")

		_, ok = batches.result.DeleteResults[3]
		assert.True(t, ok, "expected an insert results")
	})
}
