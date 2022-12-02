// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/internal/testutil/monitor"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
)

func TestRetryableWritesProse(t *testing.T) {
	clientOpts := options.Client().SetRetryWrites(true).SetWriteConcern(mtest.MajorityWc).
		SetReadConcern(mtest.MajorityRc)
	mtOpts := mtest.NewOptions().ClientOptions(clientOpts).MinServerVersion("3.6").CreateClient(false)
	mt := mtest.New(t, mtOpts)
	defer mt.Close()

	includeOpts := mtest.NewOptions().Topologies(mtest.ReplicaSet, mtest.Sharded).CreateClient(false)
	mt.RunOpts("txn number included", includeOpts, func(mt *mtest.T) {
		updateDoc := bson.D{{"$inc", bson.D{{"x", 1}}}}
		insertOneDoc := bson.D{{"x", 1}}
		insertManyOrderedArgs := bson.D{
			{"options", bson.D{{"ordered", true}}},
			{"documents", []interface{}{insertOneDoc}},
		}
		insertManyUnorderedArgs := bson.D{
			{"options", bson.D{{"ordered", true}}},
			{"documents", []interface{}{insertOneDoc}},
		}

		testCases := []struct {
			operationName   string
			args            bson.D
			expectTxnNumber bool
		}{
			{"deleteOne", bson.D{}, true},
			{"deleteMany", bson.D{}, false},
			{"updateOne", bson.D{{"update", updateDoc}}, true},
			{"updateMany", bson.D{{"update", updateDoc}}, false},
			{"replaceOne", bson.D{}, true},
			{"insertOne", bson.D{{"document", insertOneDoc}}, true},
			{"insertMany", insertManyOrderedArgs, true},
			{"insertMany", insertManyUnorderedArgs, true},
			{"findOneAndReplace", bson.D{}, true},
			{"findOneAndUpdate", bson.D{{"update", updateDoc}}, true},
			{"findOneAndDelete", bson.D{}, true},
		}
		for _, tc := range testCases {
			mt.Run(tc.operationName, func(mt *mtest.T) {
				tcArgs, err := bson.Marshal(tc.args)
				assert.Nil(mt, err, "Marshal error: %v", err)
				crudOp := crudOperation{
					Name:      tc.operationName,
					Arguments: tcArgs,
				}

				mt.ClearEvents()
				runCrudOperation(mt, "", crudOp, crudOutcome{})
				started := mt.GetStartedEvent()
				assert.NotNil(mt, started, "expected CommandStartedEvent, got nil")
				_, err = started.Command.LookupErr("txnNumber")
				if tc.expectTxnNumber {
					assert.Nil(mt, err, "expected txnNumber in command %v", started.Command)
					return
				}
				assert.NotNil(mt, err, "did not expect txnNumber in command %v", started.Command)
			})
		}
	})
	errorOpts := mtest.NewOptions().Topologies(mtest.ReplicaSet, mtest.Sharded)
	mt.RunOpts("wrap mmapv1 error", errorOpts, func(mt *mtest.T) {
		res, err := mt.DB.RunCommand(context.Background(), bson.D{{"serverStatus", 1}}).DecodeBytes()
		assert.Nil(mt, err, "serverStatus error: %v", err)
		storageEngine, ok := res.Lookup("storageEngine", "name").StringValueOK()
		if !ok || storageEngine != "mmapv1" {
			mt.Skip("skipping because storage engine is not mmapv1")
		}

		_, err = mt.Coll.InsertOne(context.Background(), bson.D{{"x", 1}})
		assert.Equal(mt, driver.ErrUnsupportedStorageEngine, err,
			"expected error %v, got %v", driver.ErrUnsupportedStorageEngine, err)
	})

	standaloneOpts := mtest.NewOptions().Topologies(mtest.Single).CreateClient(false)
	mt.RunOpts("transaction number not sent on writes", standaloneOpts, func(mt *mtest.T) {
		mt.Run("explicit session", func(mt *mtest.T) {
			// Standalones do not support retryable writes and will error if a transaction number is sent

			sess, err := mt.Client.StartSession()
			assert.Nil(mt, err, "StartSession error: %v", err)
			defer sess.EndSession(context.Background())

			mt.ClearEvents()

			err = mongo.WithSession(context.Background(), sess, func(ctx mongo.SessionContext) error {
				doc := bson.D{{"foo", 1}}
				_, err := mt.Coll.InsertOne(ctx, doc)
				return err
			})
			assert.Nil(mt, err, "InsertOne error: %v", err)

			_, wantID := sess.ID().Lookup("id").Binary()
			command := mt.GetStartedEvent().Command
			lsid, err := command.LookupErr("lsid")
			assert.Nil(mt, err, "Error getting lsid: %v", err)
			_, gotID := lsid.Document().Lookup("id").Binary()
			assert.True(mt, bytes.Equal(wantID, gotID), "expected session ID %v, got %v", wantID, gotID)
			txnNumber, err := command.LookupErr("txnNumber")
			assert.NotNil(mt, err, "expected no txnNumber, got %v", txnNumber)
		})
		mt.Run("implicit session", func(mt *mtest.T) {
			// Standalones do not support retryable writes and will error if a transaction number is sent

			mt.ClearEvents()

			doc := bson.D{{"foo", 1}}
			_, err := mt.Coll.InsertOne(context.Background(), doc)
			assert.Nil(mt, err, "InsertOne error: %v", err)

			command := mt.GetStartedEvent().Command
			lsid, err := command.LookupErr("lsid")
			assert.Nil(mt, err, "Error getting lsid: %v", err)
			_, gotID := lsid.Document().Lookup("id").Binary()
			assert.NotNil(mt, gotID, "expected session ID, got nil")
			txnNumber, err := command.LookupErr("txnNumber")
			assert.NotNil(mt, err, "expected no txnNumber, got %v", txnNumber)
		})
	})

	tpm := monitor.NewTestPoolMonitor()
	// Client options with MaxPoolSize of 1 and RetryWrites used per the test description.
	// Lower HeartbeatInterval used to speed the test up for any server that uses streaming
	// heartbeats. Only connect to first host in list for sharded clusters.
	hosts := mtest.ClusterConnString().Hosts
	pceOpts := options.Client().SetMaxPoolSize(1).SetRetryWrites(true).
		SetPoolMonitor(tpm.PoolMonitor).SetHeartbeatInterval(500 * time.Millisecond).
		SetHosts(hosts[:1])

	mtPceOpts := mtest.NewOptions().ClientOptions(pceOpts).MinServerVersion("4.3").
		Topologies(mtest.ReplicaSet, mtest.Sharded)
	mt.RunOpts("PoolClearedError retryability", mtPceOpts, func(mt *mtest.T) {
		// Force Find to block for 1 second once.
		mt.SetFailPoint(mtest.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: mtest.FailPointMode{
				Times: 1,
			},
			Data: mtest.FailPointData{
				FailCommands:    []string{"insert"},
				ErrorCode:       91,
				BlockConnection: true,
				BlockTimeMS:     1000,
				ErrorLabels:     &[]string{"RetryableWriteError"},
			},
		})

		// Clear CMAP and command events.
		tpm.ClearEvents()
		mt.ClearEvents()

		// Perform an InsertOne on two different threads and assert both operations are
		// successful.
		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := mt.Coll.InsertOne(context.Background(), bson.D{{"x", 1}})
				assert.Nil(mt, err, "InsertOne error: %v", err)
			}()
		}
		wg.Wait()

		// Gather GetSucceeded, GetFailed and PoolCleared pool events.
		events := tpm.Events(func(e *event.PoolEvent) bool {
			getSucceeded := e.Type == event.GetSucceeded
			getFailed := e.Type == event.GetFailed
			poolCleared := e.Type == event.PoolCleared
			return getSucceeded || getFailed || poolCleared
		})

		// Assert that first check out succeeds, pool is cleared, and second check
		// out fails due to connection error.
		assert.True(mt, len(events) >= 3, "expected at least 3 events, got %v", len(events))
		assert.Equal(mt, event.GetSucceeded, events[0].Type,
			"expected ConnectionCheckedOut event, got %v", events[0].Type)
		assert.Equal(mt, event.PoolCleared, events[1].Type,
			"expected ConnectionPoolCleared event, got %v", events[1].Type)
		assert.Equal(mt, event.GetFailed, events[2].Type,
			"expected ConnectionCheckedOutFailed event, got %v", events[2].Type)
		assert.Equal(mt, event.ReasonConnectionErrored, events[2].Reason,
			"expected check out failure due to connection error, failed due to %q", events[2].Reason)

		// Assert that three insert CommandStartedEvents were observed.
		for i := 0; i < 3; i++ {
			cmdEvt := mt.GetStartedEvent()
			assert.NotNil(mt, cmdEvt, "expected an insert event, got nil")
			assert.Equal(mt, cmdEvt.CommandName, "insert",
				"expected an insert event, got a(n) %v event", cmdEvt.CommandName)
		}
	})

	mtNWPOpts := mtest.NewOptions().MinServerVersion("6.0").Topologies(mtest.ReplicaSet)
	mt.RunOpts(fmt.Sprintf("%s label returns original error", driver.NoWritesPerformed), mtNWPOpts,
		func(mt *mtest.T) {
			const shutdownInProgressErrorCode int32 = 91
			const notWritablePrimaryErrorCode int32 = 10107

			monitor := new(event.CommandMonitor)
			mt.ResetClient(options.Client().SetRetryWrites(true).SetMonitor(monitor))

			// Configure a fail point for a "ShutdownInProgress" error.
			mt.SetFailPoint(mtest.FailPoint{
				ConfigureFailPoint: "failCommand",
				Mode:               mtest.FailPointMode{Times: 1},
				Data: mtest.FailPointData{
					WriteConcernError: &mtest.WriteConcernErrorData{
						Code: shutdownInProgressErrorCode,
					},
					FailCommands: []string{"insert"},
				},
			})

			// secondFailPointConfigured is used to determine if the conditions from the
			// shutdownInProgressErrorCode actually configures the "NoWritablePrimary" fail command.
			var secondFailPointConfigured bool

			//Set a command monitor on the client that configures a failpoint with a "NoWritesPerformed"
			monitor.Succeeded = func(_ context.Context, evt *event.CommandSucceededEvent) {
				var errorCode int32
				if wce := evt.Reply.Lookup("writeConcernError"); wce.Type == bsontype.EmbeddedDocument {
					var ok bool
					errorCode, ok = wce.Document().Lookup("code").Int32OK()
					if !ok {
						t.Fatalf("expected code to be an int32, got %v",
							wce.Document().Lookup("code").Type)
						return
					}
				}

				// Do not set a fail point if event was not a writeConcernError with an error code for
				// "ShutdownInProgress".
				if errorCode != shutdownInProgressErrorCode {
					return
				}

				mt.SetFailPoint(mtest.FailPoint{
					ConfigureFailPoint: "failCommand",
					Mode:               mtest.FailPointMode{Times: 1},
					Data: mtest.FailPointData{
						ErrorCode: notWritablePrimaryErrorCode,
						ErrorLabels: &[]string{
							driver.NoWritesPerformed,
							driver.RetryableWriteError,
						},
						FailCommands: []string{"insert"},
					},
				})
				secondFailPointConfigured = true
			}

			// Attempt to insert a document.
			_, err := mt.Coll.InsertOne(context.Background(), bson.D{{"x", 1}})

			require.True(mt, secondFailPointConfigured)

			// Assert that the "ShutdownInProgress" error is returned.
			require.True(mt, err.(mongo.WriteException).HasErrorCode(int(shutdownInProgressErrorCode)))
		})
}
