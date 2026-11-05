// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/drivertest"
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
				namespace: "db.coll", model: &ClientInsertOneModel{Document: bigDoc},
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
			namespace: "db.coll", model: &ClientInsertOneModel{Document: bson.D{{"a", 1}}},
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
				namespace: "db.coll", model: &ClientInsertOneModel{Document: doc},
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

	// A replacement document whose first key starts with '$' is rejected
	// client-side while building a later batch.
	t.Run("invalid replacement after results are observed", func(t *testing.T) {
		client, err := newClient()
		require.NoError(t, err, "newClient error: %v", err)
		client.deployment = drivertest.NewMockDeployment(okBatch)

		ordered := false
		bw := &clientBulkWrite{client: client, ordered: &ordered}
		bw.result.Acknowledged = true
		// Six documents span at least two batches, so the invalid model is
		// marshalled only after the first batch has been processed.
		for i := 0; i < 6; i++ {
			var model any = &ClientInsertOneModel{Document: bigDoc}
			if i == 5 {
				model = &ClientReplaceOneModel{
					Filter:      bson.D{},
					Replacement: bson.D{{"$a", "b"}},
				}
			}
			bw.writePairs = append(bw.writePairs, clientBulkWritePair{
				namespace: "db.coll", model: model,
			})
		}

		err = wrapErrors(bw.execute(context.Background()))

		var bwe ClientBulkWriteException
		require.True(t, errors.As(err, &bwe),
			"expected a ClientBulkWriteException, got %T: %v", err, err)
		require.NotNil(t, bwe.WriteError, "expected the client-side error to be embedded")
		require.Equal(t, "replacement document cannot contain keys beginning with '$'", bwe.WriteError.Message)
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

	// A response that reports a failed write but records no matching cursor
	// entry leaves nothing to report. The internal errWriteFailed signal must
	// not leak to the caller.
	t.Run("a write failure with no recorded error is not surfaced", func(t *testing.T) {
		emptyBatch := bson.D{
			{Key: "ok", Value: 1},
			{Key: "nErrors", Value: 1},
			{Key: "cursor", Value: cursorReply()},
		}

		err := runOrdered(t, emptyBatch)

		require.NoError(t, err, "expected no error to be surfaced")
	})

	// An unacknowledged write is not a failure. Intermediate batches of a
	// multi-batch unacknowledged write are still read, so resultsObserved is
	// already true by the time the final batch returns ErrUnacknowledgedWrite.
	t.Run("an unacknowledged multi-batch write is not a failure", func(t *testing.T) {
		client, err := newClient()
		require.NoError(t, err, "newClient error: %v", err)
		client.deployment = drivertest.NewMockDeployment(okBatch, okBatch)

		ordered := false
		bw := &clientBulkWrite{
			client:       client,
			ordered:      &ordered,
			writeConcern: writeconcern.Unacknowledged(),
		}
		bw.result.Acknowledged = false
		// Four writes span two batches: the first is read, the last is sent
		// with moreToCome and produces no response.
		for i := 0; i < 4; i++ {
			bw.writePairs = append(bw.writePairs, clientBulkWritePair{
				namespace: "db.coll", model: &ClientInsertOneModel{Document: bigDoc},
			})
		}

		err = wrapErrors(bw.execute(context.Background()))

		require.NoError(t, err, "expected no error for an unacknowledged write")
	})
}

func TestHasTopLevelErr(t *testing.T) {
	testCases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"write failure sentinel", errWriteFailed, false},
		{"wrapped write failure sentinel", fmt.Errorf("wrapped: %w", errWriteFailed), false},
		{"driver error", driver.Error{Code: 189, Message: "PrimarySteppedDown"}, true},
		{"unacknowledged write", driver.ErrUnacknowledgedWrite, false},
		{"write command error with a write concern error", driver.WriteCommandError{
			WriteConcernError: &driver.WriteConcernError{Code: 91},
		}, false},
		{"write command error with write errors", driver.WriteCommandError{
			WriteErrors: driver.WriteErrors{{Code: 11000}},
		}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, hasTopLevelErr(tc.err))
		})
	}
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

	t.Run("size limit counts whole nsInfo documents", func(t *testing.T) {
		// Give every write its own UUID so that each one adds an nsInfo entry.
		pairs := make([]clientBulkWritePair, 100)
		for i := range pairs {
			uuid := &bson.Binary{Subtype: bson.TypeBinaryUUID, Data: fmt.Appendf(nil, "%016d", i)}
			pairs[i] = pair("db.c0", uuid)
		}
		batches := newBatches(t, pairs...)

		const limit = 3000
		n, dst, err := batches.AppendBatchSequence(nil, len(pairs), limit)
		require.NoError(t, err, "AppendBatchSequence error: %v", err)
		require.Less(t, n, len(pairs), "expected the size limit to split the writes")
		assert.LessOrEqual(t, len(dst), limit-1000,
			"expected the batch to fit within the limit minus the 1000-byte reserve")
	})
}
