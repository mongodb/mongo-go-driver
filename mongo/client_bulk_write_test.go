// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
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

func TestModelBatches_processResponseLabels(t *testing.T) {
	// A failed response ("ok": 0) makes processResponse return the exception
	// directly, which is where the labels have to surface.
	newFailedResp := func(t *testing.T) bsoncore.Document {
		t.Helper()

		resp, err := bson.Marshal(bson.D{
			{Key: "ok", Value: 0},
			{Key: "code", Value: 11602},
			{Key: "errmsg", Value: "operation was interrupted"},
		})
		require.NoError(t, err)

		return bsoncore.Document(resp)
	}

	tests := []struct {
		name string
		// errs is the driver error reported for each batch in turn.
		errs []error
		want []string
	}{
		{
			name: "labels from a write command error",
			errs: []error{driver.WriteCommandError{Labels: []string{"RetryableWriteError"}}},
			want: []string{"RetryableWriteError"},
		},
		{
			name: "labels without a write concern error",
			// Labels hang off the command error, so they are collected even
			// when the batch reports no write concern error.
			errs: []error{driver.WriteCommandError{
				Labels:      []string{"TransientTransactionError"},
				WriteErrors: driver.WriteErrors{{Index: 0, Code: 11000}},
			}},
			want: []string{"TransientTransactionError"},
		},
		{
			name: "labels merged across batches",
			errs: []error{
				driver.WriteCommandError{Labels: []string{"RetryableWriteError"}},
				driver.WriteCommandError{Labels: []string{"RetryableWriteError", "UnknownTransactionCommitResult"}},
			},
			want: []string{"RetryableWriteError", "UnknownTransactionCommitResult"},
		},
		{
			name: "no error yields no labels",
			errs: []error{nil},
			want: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			batches := &modelBatches{result: &ClientBulkWriteResult{}}

			var err error
			for _, batchErr := range test.errs {
				err = batches.processResponse(context.Background(), newFailedResp(t),
					driver.ResponseInfo{Error: batchErr})
			}

			assert.Equal(t, test.want, batches.labels)

			bwe, ok := err.(ClientBulkWriteException)
			require.True(t, ok, "expected a ClientBulkWriteException, got %T", err)
			assert.Equal(t, test.want, bwe.Labels)

			for _, label := range test.want {
				assert.True(t, bwe.HasErrorLabel(label), "expected label %q", label)
			}
		})
	}
}

func TestAppendMissingLabels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		dst  []string
		src  []string
		want []string
	}{
		{
			name: "appends to empty",
			src:  []string{"a", "b"},
			want: []string{"a", "b"},
		},
		{
			name: "skips duplicates",
			dst:  []string{"a"},
			src:  []string{"a", "b"},
			want: []string{"a", "b"},
		},
		{
			name: "deduplicates within src",
			src:  []string{"a", "a"},
			want: []string{"a"},
		},
		{
			name: "empty src leaves dst alone",
			dst:  []string{"a"},
			want: []string{"a"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.want, appendMissingLabels(test.dst, test.src))
		})
	}
}
