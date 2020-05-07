// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	errorCursorNotFound = 43
)

func TestCursor(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().CreateClient(false))
	defer mt.Close()
	cappedCollectionOpts := bson.D{{"capped", true}, {"size", 64 * 1024}}

	// server versions 2.6 and 3.0 use OP_GET_MORE so this works on >= 3.2
	mt.RunOpts("cursor is killed on server", mtest.NewOptions().MinServerVersion("3.2"), func(mt *mtest.T) {
		initCollection(mt, mt.Coll)
		c, err := mt.Coll.Find(mtest.Background, bson.D{}, options.Find().SetBatchSize(2))
		assert.Nil(mt, err, "Find error: %v", err)

		id := c.ID()
		assert.True(mt, c.Next(mtest.Background), "expected Next true, got false")
		err = c.Close(mtest.Background)
		assert.Nil(mt, err, "Close error: %v", err)

		err = mt.DB.RunCommand(mtest.Background, bson.D{
			{"getMore", id},
			{"collection", mt.Coll.Name()},
		}).Err()
		ce := err.(mongo.CommandError)
		assert.Equal(mt, int32(errorCursorNotFound), ce.Code, "expected error code %v, got %v", errorCursorNotFound, ce.Code)
	})
	mt.RunOpts("try next", noClientOpts, func(mt *mtest.T) {
		mt.Run("existing non-empty batch", func(mt *mtest.T) {
			// If there's already documents in the current batch, TryNext should return true without doing a getMore

			initCollection(mt, mt.Coll)
			cursor, err := mt.Coll.Find(mtest.Background, bson.D{})
			assert.Nil(mt, err, "Find error: %v", err)
			defer cursor.Close(mtest.Background)
			tryNextExistingBatchTest(mt, cursor)
		})
		mt.RunOpts("one getMore sent", mtest.NewOptions().CollectionCreateOptions(cappedCollectionOpts), func(mt *mtest.T) {
			// If the current batch is empty, TryNext should send one getMore and return.

			// insert a document because a tailable cursor will only have a non-zero ID if the initial Find matches
			// at least one document
			_, err := mt.Coll.InsertOne(mtest.Background, bson.D{{"x", 1}})
			assert.Nil(mt, err, "InsertOne error: %v", err)

			cursor, err := mt.Coll.Find(mtest.Background, bson.D{}, options.Find().SetCursorType(options.Tailable))
			assert.Nil(mt, err, "Find error: %v", err)
			defer cursor.Close(mtest.Background)

			// first call to TryNext should return 1 document
			assert.True(mt, cursor.TryNext(mtest.Background), "expected Next to return true, got false")
			// TryNext should attempt one getMore
			mt.ClearEvents()
			assert.False(mt, cursor.TryNext(mtest.Background), "unexpected document %v", cursor.Current)
			verifyOneGetmoreSent(mt, cursor)
		})
		mt.RunOpts("getMore error", mtest.NewOptions().ClientType(mtest.Mock), func(mt *mtest.T) {
			findRes := mtest.CreateCursorResponse(50, "foo.bar", mtest.FirstBatch)
			mt.AddMockResponses(findRes)
			cursor, err := mt.Coll.Find(mtest.Background, bson.D{})
			assert.Nil(mt, err, "Find error: %v", err)
			defer cursor.Close(mtest.Background)
			tryNextGetmoreError(mt, cursor)
		})
	})
	mt.RunOpts("RemainingBatchLength", noClientOpts, func(mt *mtest.T) {
		cappedMtOpts := mtest.NewOptions().CollectionCreateOptions(cappedCollectionOpts)
		mt.RunOpts("first batch is non empty", cappedMtOpts, func(mt *mtest.T) {
			// Test that the cursor reports the correct value for RemainingBatchLength at various execution points if
			// the first batch from the server is non-empty.

			initCollection(mt, mt.Coll)

			// Create a tailable await cursor with a low cursor timeout.
			batchSize := 2
			findOpts := options.Find().
				SetBatchSize(int32(batchSize)).
				SetCursorType(options.TailableAwait).
				SetMaxAwaitTime(100 * time.Millisecond)
			cursor, err := mt.Coll.Find(mtest.Background, bson.D{}, findOpts)
			assert.Nil(mt, err, "Find error: %v", err)
			defer cursor.Close(mtest.Background)

			mt.ClearEvents()

			// The initial batch length should be equal to the batchSize. Do batchSize Next calls to exhaust the current
			// batch and assert that no getMore was done.
			assertCursorBatchLength(mt, cursor, batchSize)
			for i := 0; i < batchSize; i++ {
				prevLength := cursor.RemainingBatchLength()
				if !cursor.Next(mtest.Background) {
					mt.Fatalf("expected Next to return true on index %d; cursor err: %v", i, cursor.Err())
				}

				// Each successful Next call should decrement batch length by 1.
				assertCursorBatchLength(mt, cursor, prevLength-1)
			}
			evt := mt.GetStartedEvent()
			assert.Nil(mt, evt, "expected no events, got %v", evt)

			// The batch is exhaused, so the batch length should be 0. Do one Next call, which should do a getMore and
			// fetch batchSize more documents. The batch length after the call should be (batchSize-1) because Next consumes
			// one document.
			assertCursorBatchLength(mt, cursor, 0)

			assert.True(mt, cursor.Next(mtest.Background), "expected Next to return true; cursor err: %v", cursor.Err())
			evt = mt.GetStartedEvent()
			assert.NotNil(mt, evt, "expected CommandStartedEvent, got nil")
			assert.Equal(mt, "getMore", evt.CommandName, "expected command %q, got %q", "getMore", evt.CommandName)

			assertCursorBatchLength(mt, cursor, batchSize-1)
		})
		mt.RunOpts("first batch is empty", mtest.NewOptions().ClientType(mtest.Mock), func(mt *mtest.T) {
			// Test that the cursor reports the correct value for RemainingBatchLength if the first batch is empty.
			// Using a mock deployment simplifies this test becuase the server won't create a valid cursor if the
			// collection is empty when the find is run.

			cursorID := int64(50)
			ns := mt.DB.Name() + "." + mt.Coll.Name()
			getMoreBatch := []bson.D{
				{{"x", 1}},
				{{"x", 2}},
			}

			// Create mock responses.
			find := mtest.CreateCursorResponse(cursorID, ns, mtest.FirstBatch)
			getMore := mtest.CreateCursorResponse(cursorID, ns, mtest.NextBatch, getMoreBatch...)
			killCursors := mtest.CreateSuccessResponse()
			mt.AddMockResponses(find, getMore, killCursors)

			cursor, err := mt.Coll.Find(mtest.Background, bson.D{})
			assert.Nil(mt, err, "Find error: %v", err)
			defer cursor.Close(mtest.Background)
			mt.ClearEvents()

			for {
				if cursor.TryNext(mtest.Background) {
					break
				}

				assert.Nil(mt, cursor.Err(), "cursor error: %v", err)
				assertCursorBatchLength(mt, cursor, 0)
			}
			// TryNext consumes one document so the remaining batch size should be len(getMoreBatch)-1.
			assertCursorBatchLength(mt, cursor, len(getMoreBatch)-1)
		})
	})
}

type tryNextCursor interface {
	TryNext(context.Context) bool
	Err() error
}

func tryNextExistingBatchTest(mt *mtest.T, cursor tryNextCursor) {
	mt.Helper()

	mt.ClearEvents()
	assert.True(mt, cursor.TryNext(mtest.Background), "expected TryNext to return true, got false")
	evt := mt.GetStartedEvent()
	if evt != nil {
		mt.Fatalf("unexpected event sent during TryNext: %v", evt.CommandName)
	}
}

// use command monitoring to verify that a single getMore was sent
func verifyOneGetmoreSent(mt *mtest.T, cursor tryNextCursor) {
	mt.Helper()

	evt := mt.GetStartedEvent()
	assert.NotNil(mt, evt, "expected getMore event, got nil")
	assert.Equal(mt, "getMore", evt.CommandName, "expected 'getMore' event, got '%v'", evt.CommandName)
	evt = mt.GetStartedEvent()
	if evt != nil {
		mt.Fatalf("unexpected event sent during TryNext: %v", evt.CommandName)
	}
}

// should be called in a test run with a mock deployment
func tryNextGetmoreError(mt *mtest.T, cursor tryNextCursor) {
	getMoreRes := mtest.CreateCommandErrorResponse(mtest.CommandError{
		Code:    100,
		Message: "getMore error",
		Name:    "CursorError",
		Labels:  []string{"NonResumableChangeStreamError"},
	})
	mt.AddMockResponses(getMoreRes)

	// first call to TryNext should return false because first batch was empty so batch cursor returns false
	// without doing a getMore
	// next call to TryNext should attempt a getMore
	for i := 0; i < 2; i++ {
		assert.False(mt, cursor.TryNext(mtest.Background), "TryNext returned true on iteration %v", i)
	}

	err := cursor.Err()
	assert.NotNil(mt, err, "expected change stream error, got nil")
}

func assertCursorBatchLength(mt *mtest.T, cursor *mongo.Cursor, expected int) {
	batchLen := cursor.RemainingBatchLength()
	assert.Equal(mt, expected, batchLen, "expected remaining batch length %d, got %d", expected, batchLen)
}
