// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/drivertest"
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

func TestClientBulkWrite_executeTopLevelError(t *testing.T) {
	// bigDoc is large enough that four writes span two batches under the mock
	// deployment's maximum message size.
	bigDoc := bson.D{{Key: "a", Value: strings.Repeat("x", 12_000_000)}}

	cursorReply := func(entries ...bson.D) bson.D {
		return bson.D{
			{Key: "id", Value: int64(0)},
			{Key: "ns", Value: "admin.$cmd.bulkWrite"},
			{Key: "firstBatch", Value: func() bson.A {
				arr := bson.A{}
				for _, e := range entries {
					arr = append(arr, e)
				}
				return arr
			}()},
		}
	}
	// writeErrBatch is acknowledged, but its reported operation failed.
	writeErrBatch := bson.D{
		{Key: "ok", Value: 1},
		{Key: "nErrors", Value: 1},
		{Key: "cursor", Value: cursorReply(bson.D{
			{Key: "ok", Value: 0},
			{Key: "idx", Value: int32(0)},
			{Key: "code", Value: int32(11000)},
			{Key: "errmsg", Value: "duplicate key"},
			{Key: "n", Value: int32(0)},
		})},
	}
	// okBatch is acknowledged and every reported operation succeeded.
	okBatch := bson.D{
		{Key: "ok", Value: 1},
		{Key: "nErrors", Value: 0},
		{Key: "nInserted", Value: 3},
		{Key: "cursor", Value: cursorReply(
			bson.D{{Key: "ok", Value: 1}, {Key: "idx", Value: int32(0)}, {Key: "n", Value: int32(1)}},
			bson.D{{Key: "ok", Value: 1}, {Key: "idx", Value: int32(1)}, {Key: "n", Value: int32(1)}},
			bson.D{{Key: "ok", Value: 1}, {Key: "idx", Value: int32(2)}, {Key: "n", Value: int32(1)}},
		)},
	}
	// topLevelErr is a command error, the way a failed batch is reported.
	topLevelErr := bson.D{
		{Key: "ok", Value: 0},
		{Key: "code", Value: 189},
		{Key: "errmsg", Value: "PrimarySteppedDown"},
	}

	// run executes a bulk write of writes documents against the given replies
	// and returns what Client.BulkWrite would return. The top-level error is
	// queued more than once so the test does not depend on the exact point the
	// input is split into batches.
	run := func(t *testing.T, writes int, replies ...bson.D) error {
		t.Helper()

		client, err := newClient()
		require.NoError(t, err, "newClient error: %v", err)
		client.deployment = drivertest.NewMockDeployment(replies...)

		ordered := false
		bw := &clientBulkWrite{client: client, ordered: &ordered}
		// Client.BulkWrite sets this, and results are only recorded for
		// acknowledged writes.
		bw.result.Acknowledged = true
		for i := 0; i < writes; i++ {
			bw.writePairs = append(bw.writePairs, clientBulkWritePair{
				"db.coll", &ClientInsertOneModel{Document: bigDoc},
			})
		}

		return wrapErrors(bw.execute(context.Background()))
	}

	// runOrdered is run with an ordered bulk write, so a write error halts
	// execution after the batch that reported it.
	runOrdered := func(t *testing.T, replies ...bson.D) error {
		t.Helper()

		client, err := newClient()
		require.NoError(t, err, "newClient error: %v", err)
		client.deployment = drivertest.NewMockDeployment(replies...)

		ordered := true
		bw := &clientBulkWrite{client: client, ordered: &ordered}
		bw.result.Acknowledged = true
		bw.writePairs = append(bw.writePairs, clientBulkWritePair{
			"db.coll", &ClientInsertOneModel{Document: bson.D{{"a", 1}}},
		})

		return wrapErrors(bw.execute(context.Background()))
	}

	t.Run("a clean top-level failure stays a CommandError", func(t *testing.T) {
		err := run(t, 1, topLevelErr)

		var ce CommandError
		require.True(t, errors.As(err, &ce),
			"expected a CommandError, got %T: %v", err, err)

		var bwe ClientBulkWriteException
		require.False(t, errors.As(err, &bwe),
			"expected no ClientBulkWriteException, got: %v", bwe)
	})

	t.Run("top-level error after results are observed", func(t *testing.T) {
		err := run(t, 4, okBatch, topLevelErr, topLevelErr, topLevelErr)

		var bwe ClientBulkWriteException
		require.True(t, errors.As(err, &bwe),
			"expected a ClientBulkWriteException, got %T: %v", err, err)
		require.NotNil(t, bwe.WriteError, "expected the top-level error to be embedded")
		require.Equal(t, 189, bwe.WriteError.Code)
		require.NotEmpty(t, bwe.WriteError.Raw, "expected the raw server reply")
		require.NotNil(t, bwe.PartialResult, "expected the observed results to be retained")
		require.Equal(t, int64(3), bwe.PartialResult.InsertedCount)
	})

	t.Run("top-level error after write errors are observed", func(t *testing.T) {
		err := run(t, 4, writeErrBatch, topLevelErr, topLevelErr, topLevelErr)

		var bwe ClientBulkWriteException
		require.True(t, errors.As(err, &bwe),
			"expected a ClientBulkWriteException, got %T: %v", err, err)
		require.NotNil(t, bwe.WriteError, "expected the top-level error to be embedded")
		require.Equal(t, 189, bwe.WriteError.Code)
		require.NotEmpty(t, bwe.WriteError.Raw, "expected the raw server reply")
		require.Len(t, bwe.WriteErrors, 1, "expected the observed write error to be retained")
	})

	t.Run("client-side error after results are observed", func(t *testing.T) {
		client, err := newClient()
		require.NoError(t, err, "newClient error: %v", err)
		client.deployment = drivertest.NewMockDeployment(okBatch)

		ordered := false
		bw := &clientBulkWrite{client: client, ordered: &ordered}
		bw.result.Acknowledged = true
		// Six documents span at least two batches, so the last one is
		// marshalled only after the first batch has been processed.
		for i := 0; i < 6; i++ {
			var doc any = bigDoc
			if i == 5 {
				// Cannot be marshalled, so building a later batch fails.
				doc = make(chan int)
			}
			bw.writePairs = append(bw.writePairs, clientBulkWritePair{
				"db.coll", &ClientInsertOneModel{Document: doc},
			})
		}

		err = wrapErrors(bw.execute(context.Background()))

		var bwe ClientBulkWriteException
		require.True(t, errors.As(err, &bwe),
			"expected a ClientBulkWriteException, got %T: %v", err, err)
		require.NotNil(t, bwe.WriteError, "expected the client-side error to be embedded")
		require.NotEmpty(t, bwe.WriteError.Message, "expected the client-side error message")
		require.Equal(t, 0, bwe.WriteError.Code, "expected no code for a client-side error")
		require.NotNil(t, bwe.PartialResult, "expected the observed results to be retained")
	})

	t.Run("ordered write errors are not a top-level error", func(t *testing.T) {
		err := runOrdered(t, writeErrBatch)

		var bwe ClientBulkWriteException
		require.True(t, errors.As(err, &bwe),
			"expected a ClientBulkWriteException, got %T: %v", err, err)
		require.Len(t, bwe.WriteErrors, 1, "expected the write error to be reported")
		require.Nil(t, bwe.WriteError, "expected no top-level error")
	})

	t.Run("write concern errors are not a top-level error", func(t *testing.T) {
		wceBatch := bson.D{
			{Key: "ok", Value: 1},
			{Key: "nErrors", Value: 0},
			{Key: "cursor", Value: cursorReply(bson.D{
				{Key: "ok", Value: 1}, {Key: "idx", Value: int32(0)}, {Key: "n", Value: int32(1)},
			})},
			{Key: "writeConcernError", Value: bson.D{
				{Key: "code", Value: 91},
				{Key: "errmsg", Value: "Replication is being shut down"},
			}},
		}

		err := run(t, 1, wceBatch)

		var bwe ClientBulkWriteException
		require.True(t, errors.As(err, &bwe),
			"expected a ClientBulkWriteException, got %T: %v", err, err)
		require.Len(t, bwe.WriteConcernErrors, 1, "expected the write concern error to be reported")
		require.Nil(t, bwe.WriteError, "expected no top-level error")
		require.Equal(t, []int{91}, bwe.ErrorCodes(),
			"expected only the write concern error code")
	})
}
