// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/internal/failpoint"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Tests in this file replace the following unified spec tests, which are no
// longer applicable to server versions > 8.99:
//
// TestUnifiedSpec/client-side-operations-timeout/tests/override-operation-timeoutMS.json/timeoutMS_can_be_configured_for_an_operation_-_createChangeStream_on_client
// TestUnifiedSpec/client-side-operations-timeout/tests/override-operation-timeoutMS.json/timeoutMS_can_be_configured_for_an_operation_-_createChangeStream_on_database
// TestUnifiedSpec/client-side-operations-timeout/tests/override-operation-timeoutMS.json/timeoutMS_can_be_configured_for_an_operation_-_createChangeStream_on_collection
//
// TestUnifiedSpec/client-side-operations-timeout/tests/retryability-timeoutMS.json/operation_is_retried_multiple_times_for_non-zero_timeoutMS_-_createChangeStream_on_client
// TestUnifiedSpec/client-side-operations-timeout/tests/retryability-timeoutMS.json/operation_is_retried_multiple_times_for_non-zero_timeoutMS_-_createChangeStream_on_database
// TestUnifiedSpec/client-side-operations-timeout/tests/retryability-timeoutMS.json/operation_is_retried_multiple_times_for_non-zero_timeoutMS_-_createChangeStream_on_collection

type changeStreamWatcher interface {
	Watch(context.Context, any, ...options.Lister[options.ChangeStreamOptions]) (*mongo.ChangeStream, error)
}

func TestCSOT_OverrideOperationTimeout_ChangeStreamOnClient(t *testing.T) {
	requireCSOTOverrideOperationTimeoutChangeStream(t, func(mt *mtest.T) changeStreamWatcher { return mt.Client })
}

func TestCSOT_OverrideOperationTimeout_ChangeStreamOnDatabase(t *testing.T) {
	requireCSOTOverrideOperationTimeoutChangeStream(t, func(mt *mtest.T) changeStreamWatcher { return mt.DB })
}

func TestCSOT_OverrideOperationTimeout_ChangeStreamOnCollection(t *testing.T) {
	requireCSOTOverrideOperationTimeoutChangeStream(t, func(mt *mtest.T) changeStreamWatcher { return mt.Coll })
}

func TestCSOT_Retry_ChangeStreamOnClient(t *testing.T) {
	requireCSOTRetriedChangeStream(t, func(mt *mtest.T) changeStreamWatcher { return mt.Client })
}

func TestCSOT_Retry_ChangeStreamOnDatabase(t *testing.T) {
	requireCSOTRetriedChangeStream(t, func(mt *mtest.T) changeStreamWatcher { return mt.DB })
}

func TestCSOT_Retry_ChangeStreamOnCollection(t *testing.T) {
	requireCSOTRetriedChangeStream(t, func(mt *mtest.T) changeStreamWatcher { return mt.Coll })
}

// =============================================================================
// Test Runner Helpers
// =============================================================================

func changeStreamTarget(mt *mtest.T, watcher changeStreamWatcher) (wantDB, wantColl string) {
	mt.Helper()

	switch v := watcher.(type) {
	case *mongo.Client:
		wantDB = "admin"
	case *mongo.Database:
		wantDB = v.Name()
	case *mongo.Collection:
		wantDB = v.Database().Name()
		wantColl = v.Name()
	default:
		mt.Fatalf("unsupported change stream watcher type %T", watcher)
	}

	return wantDB, wantColl
}

func requireChangeStreamAggregateEvents(mt *mtest.T, wantDB, wantColl string) {
	mt.Helper()

	evts := mt.GetAllStartedEvents()
	for _, evt := range evts {
		_, err := evt.Command.LookupErr("maxTimeMS")
		require.NoError(mt, err, "expected maxTimeMS to be set on aggregate command")

		maxTimeMS := evt.Command.Lookup("maxTimeMS").Int64()
		require.Positive(mt, maxTimeMS, "expected maxTimeMS to be positive")
		require.Equal(mt, wantDB, evt.DatabaseName,
			"expected aggregate command to be sent to the correct database")

		aggregateVal := evt.Command.Lookup("aggregate")
		if wantColl != "" {
			require.Equal(mt, wantColl, aggregateVal.StringValue(),
				"expected aggregate command to target the collection")
		} else {
			require.Equal(mt, int32(1), aggregateVal.Int32(),
				"expected aggregate command to target the whole database")
		}
	}
}

func requireCSOTOverrideOperationTimeoutChangeStream(t *testing.T, entity func(mt *mtest.T) changeStreamWatcher) {
	clientOpts := options.Client().SetTimeout(10 * time.Millisecond).SetMinPoolSize(1)

	mtOpts := mtest.NewOptions().
		MinServerVersion("4.4").
		Topologies(mtest.ReplicaSet, mtest.Sharded).
		AllowFailPointsOnSharded().
		ClientOptions(clientOpts).
		ClientType(mtest.Pinned)

	mt := mtest.New(t, mtOpts)
	mt.Setup()

	watcher := entity(mt)
	wantDB, wantColl := changeStreamTarget(mt, watcher)

	// Create a failpoint that blocks aggregate for 15 ms 1 time.
	mt.SetFailPoint(failpoint.FailPoint{
		ConfigureFailPoint: "failCommand",
		Mode: failpoint.Mode{
			Times: 1,
		},
		Data: failpoint.Data{
			FailCommands:    []string{"aggregate"},
			BlockConnection: true,
			BlockTimeMS:     15,
		},
	})

	// As of 9.0+ on sharded clusters, the config server's cluster time must
	// be advanced to the mongos cluster time so that a change stream open
	// doesn't block waiting for it.
	//
	// See DRIVERS-3556 / SERVER-129623 for more details.
	if mtest.ClusterTopologyKind() == mtest.Sharded {
		require.NoError(mt, mtest.AdvanceConfigClusterTime(context.Background()))
	}

	// Create a change stream with an operation timeout of 1000ms.
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()

	cs, err := watcher.Watch(ctx, mongo.Pipeline{})
	require.NoError(mt, err)

	defer cs.Close(context.Background())

	// Check an aggregate event was sent with maxTimeMS set to 1000ms, proving
	// that the client-level timeout was overridden by the context timeout.
	mt.FilterStartedEvents(func(evt *event.CommandStartedEvent) bool {
		return evt.CommandName == "aggregate"
	})

	require.Len(mt, mt.GetAllStartedEvents(), 1, "expected 1 aggregate command to be sent")

	requireChangeStreamAggregateEvents(mt, wantDB, wantColl)
}

func requireCSOTRetriedChangeStream(t *testing.T, entity func(mt *mtest.T) changeStreamWatcher) {
	clientOptions := options.Client().SetTimeout(100 * time.Millisecond).SetMinPoolSize(1)

	mtOpts := mtest.NewOptions().
		MinServerVersion("4.3.1"). // failCommand errorLabels option
		Topologies(mtest.ReplicaSet, mtest.Sharded).
		AllowFailPointsOnSharded().
		ClientOptions(clientOptions).
		ClientType(mtest.Pinned)

	mt := mtest.New(t, mtOpts)
	mt.Setup()

	// Establish a connection before the failpoint so that connection setup
	// doesn't consume the operation's timeout budget.
	require.NoError(mt, mt.Client.Ping(context.Background(), nil))

	watcher := entity(mt)
	wantDB, wantColl := changeStreamTarget(mt, watcher)

	// Create a failpoint that fails "aggregate" with a retryable error twice.
	mt.SetFailPoint(failpoint.FailPoint{
		ConfigureFailPoint: "failCommand",
		Mode: failpoint.Mode{
			Times: 2,
		},
		Data: failpoint.Data{
			FailCommands:    []string{"aggregate"},
			ErrorCode:       7,
			CloseConnection: false,
			ErrorLabels:     &[]string{"RetryableWriteError"},
		},
	})

	// As of 9.0+ on sharded clusters, the config server's cluster time must
	// be advanced to the mongos cluster time so that a change stream open
	// doesn't block waiting for it.
	//
	// See DRIVERS-3556 / SERVER-129623 for more details.
	if mtest.ClusterTopologyKind() == mtest.Sharded {
		require.NoError(mt, mtest.AdvanceConfigClusterTime(context.Background()))
	}

	// Create a change stream with an operation timeout of 1000ms. The first
	// two aggregate attempts fail with a retryable error; the third succeeds.
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()

	cs, err := watcher.Watch(ctx, mongo.Pipeline{})
	require.NoError(mt, err)

	defer cs.Close(context.Background())

	mt.FilterStartedEvents(func(evt *event.CommandStartedEvent) bool {
		return evt.CommandName == "aggregate"
	})

	require.Len(mt, mt.GetAllStartedEvents(), 3, "expected 3 aggregate commands to be sent (2 retries)")

	requireChangeStreamAggregateEvents(mt, wantDB, wantColl)
}
