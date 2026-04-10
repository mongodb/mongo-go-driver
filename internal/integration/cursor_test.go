// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/failpoint"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	errorCursorNotFound = 43
)

func TestCursor(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().CreateClient(false))

	mt.Run("cursor is killed on server", func(mt *mtest.T) {
		initCollection(mt, mt.Coll)
		c, err := mt.Coll.Find(context.Background(), bson.D{}, options.Find().SetBatchSize(2))
		assert.Nil(mt, err, "Find error: %v", err)

		id := c.ID()
		assert.True(mt, c.Next(context.Background()), "expected Next true, got false")
		err = c.Close(context.Background())
		assert.Nil(mt, err, "Close error: %v", err)

		err = mt.DB.RunCommand(context.Background(), bson.D{
			{"getMore", id},
			{"collection", mt.Coll.Name()},
		}).Err()
		ce := err.(mongo.CommandError)
		assert.Equal(mt, int32(errorCursorNotFound), ce.Code, "expected error code %v, got %v", errorCursorNotFound, ce.Code)
	})

	mt.Run("set batchSize", func(mt *mtest.T) {
		initCollection(mt, mt.Coll)
		mt.ClearEvents()

		// create cursor with batchSize 0
		cursor, err := mt.Coll.Find(context.Background(), bson.D{}, options.Find().SetBatchSize(0))
		assert.Nil(mt, err, "Find error: %v", err)
		defer cursor.Close(context.Background())
		evt := mt.GetStartedEvent()
		assert.Equal(mt, "find", evt.CommandName, "expected 'find' event, got '%v'", evt.CommandName)
		sizeVal, err := evt.Command.LookupErr("batchSize")
		assert.Nil(mt, err, "expected find command to have batchSize")
		batchSize := sizeVal.Int32()
		assert.Equal(mt, int32(0), batchSize, "expected batchSize 0, got %v", batchSize)

		// make sure that the getMore sends the new batchSize
		batchCursor := mongo.BatchCursorFromCursor(cursor)
		batchCursor.SetBatchSize(4)
		assert.True(mt, cursor.Next(context.Background()), "expected Next true, got false")
		evt = mt.GetStartedEvent()
		assert.NotNil(mt, evt, "expected getMore event, got nil")
		assert.Equal(mt, "getMore", evt.CommandName, "expected 'getMore' event, got '%v'", evt.CommandName)
		sizeVal, err = evt.Command.LookupErr("batchSize")
		assert.Nil(mt, err, "expected getMore command to have batchSize")
		batchSize = sizeVal.Int32()
		assert.Equal(mt, int32(4), batchSize, "expected batchSize 4, got %v", batchSize)
	})
}

func TestCursor_TryNext(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().CreateClient(false))

	// Skip tests if running against serverless, as capped collections are banned.
	if os.Getenv("SERVERLESS") == "serverless" {
		mt.Skip("skipping as serverless forbids capped collections")
	}

	mt.Run("existing non-empty batch", func(mt *mtest.T) {
		// If there's already documents in the current batch, TryNext should return true without doing a getMore

		initCollection(mt, mt.Coll)
		cursor, err := mt.Coll.Find(context.Background(), bson.D{})
		require.NoError(mt, err, "Find error: %v", err)
		defer cursor.Close(context.Background())
		tryNextExistingBatchTest(mt, cursor)
	})

	cappedCollectionOpts := options.CreateCollection().SetCapped(true).SetSizeInBytes(64 * 1024)
	mt.RunOpts("one getMore sent", mtest.NewOptions().CollectionCreateOptions(cappedCollectionOpts), func(mt *mtest.T) {
		// If the current batch is empty, TryNext should send one getMore and return.

		// insert a document because a tailable cursor will only have a non-zero ID if the initial Find matches
		// at least one document
		_, err := mt.Coll.InsertOne(context.Background(), bson.D{{"x", 1}})
		require.NoError(mt, err, "InsertOne error: %v", err)

		cursor, err := mt.Coll.Find(context.Background(), bson.D{}, options.Find().SetCursorType(options.Tailable))
		require.NoError(mt, err, "Find error: %v", err)
		defer cursor.Close(context.Background())

		// first call to TryNext should return 1 document
		assert.True(mt, cursor.TryNext(context.Background()), "expected Next to return true, got false")
		// TryNext should attempt one getMore
		mt.ClearEvents()
		assert.False(mt, cursor.TryNext(context.Background()), "unexpected document %v", cursor.Current)
		verifyOneGetmoreSent(mt)
	})
	mt.RunOpts("getMore error", mtest.NewOptions().ClientType(mtest.Mock), func(mt *mtest.T) {
		findRes := mtest.CreateCursorResponse(50, "foo.bar", mtest.FirstBatch)
		mt.AddMockResponses(findRes)
		cursor, err := mt.Coll.Find(context.Background(), bson.D{})
		require.NoError(mt, err, "Find error: %v", err)
		defer cursor.Close(context.Background())
		tryNextGetmoreError(mt, cursor)
	})
}

func TestCursor_RemainingBatchLength(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().CreateClient(false))

	cappedCollectionOpts := options.CreateCollection().SetCapped(true).SetSizeInBytes(64 * 1024)
	cappedMtOpts := mtest.NewOptions().CollectionCreateOptions(cappedCollectionOpts)
	// Skip tests if running against serverless, as capped collections are banned.
	if os.Getenv("SERVERLESS") == "serverless" {
		mt.Skip("skipping as serverless forbids capped collections")
	}

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
		cursor, err := mt.Coll.Find(context.Background(), bson.D{}, findOpts)
		require.NoError(mt, err, "Find error: %v", err)
		defer cursor.Close(context.Background())

		mt.ClearEvents()

		// The initial batch length should be equal to the batchSize. Do batchSize Next calls to exhaust the current
		// batch and assert that no getMore was done.
		assert.Equal(mt,
			batchSize,
			cursor.RemainingBatchLength(),
			"expected remaining batch length to match")
		for i := 0; i < batchSize; i++ {
			prevLength := cursor.RemainingBatchLength()
			if !cursor.Next(context.Background()) {
				mt.Fatalf("expected Next to return true on index %d; cursor err: %v", i, cursor.Err())
			}

			// Each successful Next call should decrement batch length by 1.
			assert.Equal(mt,
				prevLength-1,
				cursor.RemainingBatchLength(),
				"expected remaining batch length to match")
		}
		evt := mt.GetStartedEvent()
		assert.Nil(mt, evt, "expected no events, got %v", evt)

		// The batch is exhausted, so the batch length should be 0. Do one Next call, which should do a getMore and
		// fetch batchSize more documents. The batch length after the call should be (batchSize-1) because Next consumes
		// one document.
		assert.Equal(mt,
			0,
			cursor.RemainingBatchLength(),
			"expected remaining batch length to match")

		assert.True(mt, cursor.Next(context.Background()), "expected Next to return true; cursor err: %v", cursor.Err())
		evt = mt.GetStartedEvent()
		assert.NotNil(mt, evt, "expected CommandStartedEvent, got nil")
		assert.Equal(mt, "getMore", evt.CommandName, "expected command %q, got %q", "getMore", evt.CommandName)

		assert.Equal(mt,
			batchSize-1,
			cursor.RemainingBatchLength(),
			"expected remaining batch length to match")
	})
	mt.RunOpts("first batch is empty", mtest.NewOptions().ClientType(mtest.Mock), func(mt *mtest.T) {
		// Test that the cursor reports the correct value for RemainingBatchLength if the first batch is empty.
		// Using a mock deployment simplifies this test because the server won't create a valid cursor if the
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

		cursor, err := mt.Coll.Find(context.Background(), bson.D{})
		assert.Nil(mt, err, "Find error: %v", err)
		defer cursor.Close(context.Background())
		mt.ClearEvents()

		for !cursor.TryNext(context.Background()) {

			assert.Nil(mt, cursor.Err(), "cursor error: %v", err)
			assert.Equal(mt,
				0,
				cursor.RemainingBatchLength(),
				"expected remaining batch length to match")
		}
		// TryNext consumes one document so the remaining batch size should be len(getMoreBatch)-1.
		assert.Equal(mt,
			len(getMoreBatch)-1,
			cursor.RemainingBatchLength(),
			"expected remaining batch length to match")
	})
}

func TestCursor_All(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().CreateClient(false))

	failpointOpts := mtest.NewOptions().Topologies(mtest.ReplicaSet).MinServerVersion("4.0")
	mt.RunOpts("getMore error", failpointOpts, func(mt *mtest.T) {
		failpointData := failpoint.Data{
			FailCommands: []string{"getMore"},
			ErrorCode:    100,
		}
		mt.SetFailPoint(failpoint.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode:               failpoint.ModeAlwaysOn,
			Data:               failpointData,
		})
		initCollection(mt, mt.Coll)
		cursor, err := mt.Coll.Find(context.Background(), bson.D{}, options.Find().SetBatchSize(2))
		assert.Nil(mt, err, "Find error: %v", err)
		defer cursor.Close(context.Background())

		var docs []bson.D
		err = cursor.All(context.Background(), &docs)
		require.Error(mt, err, "expected change stream error, got nil")

		// make sure that a mongo.CommandError is returned instead of a driver.Error
		mongoErr, ok := err.(mongo.CommandError)
		assert.True(mt, ok, "expected mongo.CommandError, got: %T", err)
		assert.Equal(mt, failpointData.ErrorCode, mongoErr.Code, "expected code %v, got: %v", failpointData.ErrorCode, mongoErr.Code)
	})

	mt.Run("deferred Close uses context.Background", func(mt *mtest.T) {
		initCollection(mt, mt.Coll)

		// Find with batchSize 2 so All will run getMore for next 3 docs and error.
		cur, err := mt.Coll.Find(context.Background(), bson.D{},
			options.Find().SetBatchSize(2))
		require.NoError(mt, err, "Find error: %v", err)

		// Create a context and immediately cancel it.
		canceledCtx, cancel := context.WithCancel(context.Background())
		cancel()

		// Clear "insert" and "find" events.
		mt.ClearEvents()

		// Call All with the canceled context and expect context.Canceled.
		var docs []bson.D
		err = cur.All(canceledCtx, &docs)
		require.Error(mt, err, "expected error for All, got nil")
		assert.True(mt, errors.Is(err, context.Canceled),
			"expected context.Canceled error, got %v", err)

		// Assert that a "getMore" command was sent and failed (Next used the
		// canceled context).
		stEvt := mt.GetStartedEvent()
		assert.NotNil(mt, stEvt, `expected a "getMore" started event, got no event`)
		assert.Equal(mt, stEvt.CommandName, "getMore",
			`expected a "getMore" started event, got %q`, stEvt.CommandName)
		fEvt := mt.GetFailedEvent()
		assert.NotNil(mt, fEvt, `expected a failed "getMore" event, got no event`)
		assert.Equal(mt, fEvt.CommandName, "getMore",
			`expected a failed "getMore" event, got %q`, fEvt.CommandName)

		// Assert that a "killCursors" command was sent and was successful (Close
		// used the 2 second Client Timeout).
		stEvt = mt.GetStartedEvent()
		assert.NotNil(mt, stEvt, `expected a "killCursors" started event, got no event`)
		assert.Equal(mt, stEvt.CommandName, "killCursors",
			`expected a "killCursors" started event, got %q`, stEvt.CommandName)
		suEvt := mt.GetSucceededEvent()
		assert.NotNil(mt, suEvt, `expected a successful "killCursors" event, got no event`)
		assert.Equal(mt, suEvt.CommandName, "killCursors",
			`expected a successful "killCursors" event, got %q`, suEvt.CommandName)
	})
}

func TestCursor_Close(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().CreateClient(false))

	failpointOpts := mtest.NewOptions().Topologies(mtest.ReplicaSet).MinServerVersion("4.0")
	mt.RunOpts("killCursors error", failpointOpts, func(mt *mtest.T) {
		failpointData := failpoint.Data{
			FailCommands: []string{"killCursors"},
			ErrorCode:    100,
		}
		mt.SetFailPoint(failpoint.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode:               failpoint.ModeAlwaysOn,
			Data:               failpointData,
		})
		initCollection(mt, mt.Coll)
		cursor, err := mt.Coll.Find(context.Background(), bson.D{}, options.Find().SetBatchSize(2))
		require.NoError(mt, err, "Find error: %v", err)

		err = cursor.Close(context.Background())
		require.Error(mt, err, "expected change stream error, got nil")

		// make sure that a mongo.CommandError is returned instead of a driver.Error
		mongoErr, ok := err.(mongo.CommandError)
		assert.True(mt, ok, "expected mongo.CommandError, got: %T", err)
		assert.Equal(mt, failpointData.ErrorCode, mongoErr.Code, "expected code %v, got: %v", failpointData.ErrorCode, mongoErr.Code)
	})
}

func parseMaxAwaitTime(mt *mtest.T, evt *event.CommandStartedEvent) int64 {
	mt.Helper()

	maxTimeMSRaw, err := evt.Command.LookupErr("maxTimeMS")
	require.NoError(mt, err)

	got, ok := maxTimeMSRaw.AsInt64OK()
	require.True(mt, ok)

	return got
}

func tadcFindFactory(ctx context.Context, mt *mtest.T, coll mongo.Collection) *mongo.Cursor {
	mt.Helper()

	initCollection(mt, &coll)
	cur, err := coll.Find(ctx, bson.D{{"__nomatch", 1}},
		options.Find().SetBatchSize(1).SetCursorType(options.TailableAwait))
	require.NoError(mt, err, "Find error: %v", err)

	return cur
}

func tadcAggregateFactory(ctx context.Context, mt *mtest.T, coll mongo.Collection) *mongo.Cursor {
	mt.Helper()

	initCollection(mt, &coll)
	opts := options.Aggregate()
	pipeline := mongo.Pipeline{
		{{"$changeStream", bson.D{{"fullDocument", "default"}}}},
		{{"$match", bson.D{
			{"operationType", "insert"},
			{"fullDocment.__nomatch", 1},
		}}},
	}

	cursor, err := coll.Aggregate(ctx, pipeline, opts)
	require.NoError(mt, err, "Aggregate error: %v", err)

	return cursor
}

func tadcRunCommandCursorFactory(ctx context.Context, mt *mtest.T, coll mongo.Collection) *mongo.Cursor {
	mt.Helper()

	initCollection(mt, &coll)

	cur, err := coll.Database().RunCommandCursor(ctx, bson.D{
		{"find", coll.Name()},
		{"filter", bson.D{{"__nomatch", 1}}},
		{"tailable", true},
		{"awaitData", true},
		{"batchSize", int32(1)},
	})
	require.NoError(mt, err, "RunCommandCursor error: %v", err)

	return cur
}

// For tailable awaitData cursors, the maxTimeMS for a getMore should be
// min(maxAwaitTimeMS, remaining timeoutMS - minRoundTripTime) to allow the
// server more opportunities to respond with an empty batch before a
// client-side timeout.
func TestCursor_tailableAwaitData_applyRemainingTimeout(t *testing.T) {
	// These values reflect what is used in the unified spec tests, see
	// DRIVERS-2868.
	const timeoutMS = 200
	const maxAwaitTimeMS = 100
	const blockTimeMS = 30
	const getMoreBound = 71

	// TODO(GODRIVER-3328): mongos doesn't honor a failpoint's full blockTimeMS.
	baseTopologies := []mtest.TopologyKind{mtest.Single, mtest.LoadBalanced, mtest.ReplicaSet}

	type testCase struct {
		name       string
		factory    func(ctx context.Context, mt *mtest.T, coll mongo.Collection) *mongo.Cursor
		opTimeout  bool
		topologies []mtest.TopologyKind
	}

	cases := []testCase{
		// TODO(GODRIVER-2944): "find" cursors are tested in the CSOT unified spec
		// tests for tailable/awaitData cursors and so these tests can be removed
		// once the driver supports timeoutMode.
		{
			name:       "find client-level timeout",
			factory:    tadcFindFactory,
			topologies: baseTopologies,
			opTimeout:  false,
		},
		{
			name:       "find operation-level timeout",
			factory:    tadcFindFactory,
			topologies: baseTopologies,
			opTimeout:  true,
		},

		// There is no analogue to tailable/awaiData cursor unified spec tests for
		// aggregate and runnCommand.
		{
			name:       "aggregate with changeStream client-level timeout",
			factory:    tadcAggregateFactory,
			topologies: []mtest.TopologyKind{mtest.ReplicaSet, mtest.LoadBalanced},
			opTimeout:  false,
		},
		{
			name:       "aggregate with changeStream operation-level timeout",
			factory:    tadcAggregateFactory,
			topologies: []mtest.TopologyKind{mtest.ReplicaSet, mtest.LoadBalanced},
			opTimeout:  true,
		},
		{
			name:       "runCommandCursor client-level timeout",
			factory:    tadcRunCommandCursorFactory,
			topologies: baseTopologies,
			opTimeout:  false,
		},
		{
			name:       "runCommandCursor operation-level timeout",
			factory:    tadcRunCommandCursorFactory,
			topologies: baseTopologies,
			opTimeout:  true,
		},
	}

	mt := mtest.New(t, mtest.NewOptions().CreateClient(false).MinServerVersion("4.2"))

	for _, tc := range cases {
		// Reset the collection between test cases to avoid leaking timeouts
		// between tests.
		cappedOpts := options.CreateCollection().SetCapped(true).SetSizeInBytes(1024 * 64)
		caseOpts := mtest.NewOptions().
			CollectionCreateOptions(cappedOpts).
			Topologies(tc.topologies...).
			CreateClient(true)

		if !tc.opTimeout {
			caseOpts = caseOpts.ClientOptions(options.Client().SetTimeout(timeoutMS * time.Millisecond))
		}

		mt.RunOpts(tc.name, caseOpts, func(mt *mtest.T) {
			mt.SetFailPoint(failpoint.FailPoint{
				ConfigureFailPoint: "failCommand",
				Mode:               failpoint.Mode{Times: 1},
				Data: failpoint.Data{
					FailCommands:    []string{"getMore"},
					BlockConnection: true,
					BlockTimeMS:     int32(blockTimeMS),
				},
			})

			ctx := context.Background()

			var cancel context.CancelFunc
			if tc.opTimeout {
				ctx, cancel = context.WithTimeout(ctx, timeoutMS*time.Millisecond)
				defer cancel()
			}

			cur := tc.factory(ctx, mt, *mt.Coll)
			defer func() { assert.NoError(mt, cur.Close(context.Background())) }()

			require.NoError(mt, cur.Err())

			cur.SetMaxAwaitTime(maxAwaitTimeMS * time.Millisecond)

			mt.ClearEvents()

			assert.False(mt, cur.Next(ctx))

			require.Error(mt, cur.Err(), "expected error from cursor.Next")
			assert.ErrorIs(mt, cur.Err(), context.DeadlineExceeded, "expected context deadline exceeded error")

			getMoreEvts := []*event.CommandStartedEvent{}
			for _, evt := range mt.GetAllStartedEvents() {
				if evt.CommandName == "getMore" {
					getMoreEvts = append(getMoreEvts, evt)
				}
			}

			// It's possible that three getMore events are called: 100ms, 70ms, and
			// then some small leftover of remaining time (e.g. 20µs).
			require.GreaterOrEqual(mt, len(getMoreEvts), 2)

			// The first getMore should have a maxTimeMS of <= 100ms but greater
			// than 71ms, indicating that the maxAwaitTimeMS was used.
			assert.LessOrEqual(mt, parseMaxAwaitTime(mt, getMoreEvts[0]), int64(maxAwaitTimeMS))
			assert.Greater(mt, parseMaxAwaitTime(mt, getMoreEvts[0]), int64(getMoreBound))

			// The second getMore should have a maxTimeMS of <=71, indicating that we
			// are using the time remaining in the context rather than the
			// maxAwaitTimeMS.
			assert.LessOrEqual(mt, parseMaxAwaitTime(mt, getMoreEvts[1]), int64(getMoreBound))
		})
	}
}

// For tailable awaitData cursors, the maxTimeMS for a getMore should be
func TestCursor_tailableAwaitData_ShortCircuitingGetMore(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().CreateClient(false))

	cappedOpts := options.CreateCollection().SetCapped(true).
		SetSizeInBytes(1024 * 64)

	mtOpts := mtest.NewOptions().CollectionCreateOptions(cappedOpts)
	tests := []struct {
		name             string
		deadline         time.Duration
		maxAwaitTime     time.Duration
		wantShortCircuit bool
	}{
		{
			name:             "maxAwaitTime less than operation timeout",
			deadline:         200 * time.Millisecond,
			maxAwaitTime:     100 * time.Millisecond,
			wantShortCircuit: false,
		},
		{
			name:             "maxAwaitTime equal to operation timeout",
			deadline:         200 * time.Millisecond,
			maxAwaitTime:     200 * time.Millisecond,
			wantShortCircuit: true,
		},
		{
			name:             "maxAwaitTime greater than operation timeout",
			deadline:         200 * time.Millisecond,
			maxAwaitTime:     300 * time.Millisecond,
			wantShortCircuit: true,
		},
	}

	for _, tt := range tests {
		mt.Run(tt.name, func(mt *mtest.T) {
			mt.RunOpts("find", mtOpts, func(mt *mtest.T) {
				initCollection(mt, mt.Coll)

				// Create a find cursor
				opts := options.Find().
					SetBatchSize(1).
					SetMaxAwaitTime(tt.maxAwaitTime).
					SetCursorType(options.TailableAwait)

				ctx, cancel := context.WithTimeout(context.Background(), tt.deadline)
				defer cancel()

				cur, err := mt.Coll.Find(ctx, bson.D{{Key: "x", Value: 3}}, opts)
				require.NoError(mt, err, "Find error: %v", err)

				// Close to return the session to the pool.
				defer cur.Close(context.Background())

				ok := cur.Next(ctx)
				if tt.wantShortCircuit {
					assert.False(mt, ok, "expected Next to return false, got true")
					assert.EqualError(t, cur.Err(), "MaxAwaitTime must be less than the operation timeout")
				} else {
					assert.True(mt, ok, "expected Next to return true, got false")
					assert.NoError(mt, cur.Err(), "expected no error, got %v", cur.Err())
				}
			})

			mt.RunOpts("aggregate", mtOpts, func(mt *mtest.T) {
				initCollection(mt, mt.Coll)

				// Create a find cursor
				opts := options.Aggregate().
					SetBatchSize(1).
					SetMaxAwaitTime(tt.maxAwaitTime)

				ctx, cancel := context.WithTimeout(context.Background(), tt.deadline)
				defer cancel()

				cur, err := mt.Coll.Aggregate(ctx, []bson.D{}, opts)
				require.NoError(mt, err, "Aggregate error: %v", err)

				// Close to return the session to the pool.
				defer cur.Close(context.Background())

				ok := cur.Next(ctx)
				if tt.wantShortCircuit {
					assert.False(mt, ok, "expected Next to return false, got true")
					assert.EqualError(t, cur.Err(), "MaxAwaitTime must be less than the operation timeout")
				} else {
					assert.True(mt, ok, "expected Next to return true, got false")
					assert.NoError(mt, cur.Err(), "expected no error, got %v", cur.Err())
				}
			})

			// The $changeStream stage is only supported on replica sets.
			watchOpts := mtOpts.Topologies(mtest.ReplicaSet, mtest.Sharded)
			mt.RunOpts("watch", watchOpts, func(mt *mtest.T) {
				initCollection(mt, mt.Coll)

				// Create a find cursor
				opts := options.ChangeStream().SetMaxAwaitTime(tt.maxAwaitTime)

				ctx, cancel := context.WithTimeout(context.Background(), tt.deadline)
				defer cancel()

				cur, err := mt.Coll.Watch(ctx, []bson.D{}, opts)
				require.NoError(mt, err, "Watch error: %v", err)

				// Close to return the session to the pool.
				defer cur.Close(context.Background())

				if tt.wantShortCircuit {
					ok := cur.Next(ctx)

					assert.False(mt, ok, "expected Next to return false, got true")
					assert.EqualError(mt, cur.Err(), "MaxAwaitTime must be less than the operation timeout")
				}
			})
		})
	}
}

type tryNextCursor interface {
	TryNext(context.Context) bool
	Err() error
}

func tryNextExistingBatchTest(mt *mtest.T, cursor tryNextCursor) {
	mt.Helper()

	mt.ClearEvents()
	assert.True(mt, cursor.TryNext(context.Background()), "expected TryNext to return true, got false")
	evt := mt.GetStartedEvent()
	if evt != nil {
		mt.Fatalf("unexpected event sent during TryNext: %v", evt.CommandName)
	}
}

// use command monitoring to verify that a single getMore was sent
func verifyOneGetmoreSent(mt *mtest.T) {
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
	testErr := mtest.CommandError{
		Code:    100,
		Message: "getMore error",
		Name:    "CursorError",
		Labels:  []string{"NonResumableChangeStreamError"},
	}
	getMoreRes := mtest.CreateCommandErrorResponse(testErr)
	mt.AddMockResponses(getMoreRes)

	// first call to TryNext should return false because first batch was empty so batch cursor returns false
	// without doing a getMore
	// next call to TryNext should attempt a getMore
	for i := 0; i < 2; i++ {
		assert.False(mt, cursor.TryNext(context.Background()), "TryNext returned true on iteration %v", i)
	}

	err := cursor.Err()
	assert.NotNil(mt, err, "expected change stream error, got nil")

	// make sure that a mongo.CommandError is returned instead of a driver.Error
	mongoErr, ok := err.(mongo.CommandError)
	assert.True(mt, ok, "expected mongo.CommandError, got: %T", err)
	assert.Equal(mt, testErr.Code, mongoErr.Code, "expected code %v, got: %v", testErr.Code, mongoErr.Code)
	assert.Equal(mt, testErr.Message, mongoErr.Message, "expected message %v, got: %v", testErr.Message, mongoErr.Message)
	assert.Equal(mt, testErr.Name, mongoErr.Name, "expected name %v, got: %v", testErr.Name, mongoErr.Name)
	assert.Equal(mt, testErr.Labels, mongoErr.Labels, "expected labels %v, got: %v", testErr.Labels, mongoErr.Labels)
}
