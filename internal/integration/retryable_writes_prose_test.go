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
	"strings"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/eventtest"
	"go.mongodb.org/mongo-driver/v2/internal/failpoint"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/mongoutil"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
)

func TestRetryableWritesProse(t *testing.T) {
	clientOpts := options.Client().SetRetryWrites(true).SetWriteConcern(mtest.MajorityWc).
		SetReadConcern(mtest.MajorityRc)
	mtOpts := mtest.NewOptions().ClientOptions(clientOpts).MinServerVersion("3.6").CreateClient(false)
	mt := mtest.New(t, mtOpts)

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
		res, err := mt.DB.RunCommand(context.Background(), bson.D{{"serverStatus", 1}}).Raw()
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

			err = mongo.WithSession(context.Background(), sess, func(ctx context.Context) error {
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

	tpm := eventtest.NewTestPoolMonitor()
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
		mt.SetFailPoint(failpoint.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: failpoint.Mode{
				Times: 1,
			},
			Data: failpoint.Data{
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

		// Gather ConnectionCheckedOut, ConnectionCheckOutFailed and PoolCleared pool events.
		events := tpm.Events(func(e *event.PoolEvent) bool {
			connectionCheckedOut := e.Type == event.ConnectionCheckedOut
			connectionCheckOutFailed := e.Type == event.ConnectionCheckOutFailed
			poolCleared := e.Type == event.ConnectionPoolCleared
			return connectionCheckedOut || connectionCheckOutFailed || poolCleared
		})

		// Assert that first check out succeeds, pool is cleared, and second check
		// out fails due to connection error.
		assert.True(mt, len(events) >= 3, "expected at least 3 events, got %v", len(events))
		assert.Equal(mt, event.ConnectionCheckedOut, events[0].Type,
			"expected ConnectionCheckedOut event, got %v", events[0].Type)
		assert.Equal(mt, event.ConnectionPoolCleared, events[1].Type,
			"expected ConnectionPoolCleared event, got %v", events[1].Type)
		assert.Equal(mt, event.ConnectionCheckOutFailed, events[2].Type,
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
			mt.SetFailPoint(failpoint.FailPoint{
				ConfigureFailPoint: "failCommand",
				Mode:               failpoint.Mode{Times: 1},
				Data: failpoint.Data{
					WriteConcernError: &failpoint.WriteConcernError{
						Code: shutdownInProgressErrorCode,
					},
					FailCommands: []string{"insert"},
				},
			})

			// secondFailPointConfigured is used to determine if the conditions from the
			// shutdownInProgressErrorCode actually configures the "NoWritablePrimary" fail command.
			var secondFailPointConfigured bool

			// Set a command monitor on the client that configures a failpoint with a "NoWritesPerformed"
			monitor.Succeeded = func(_ context.Context, evt *event.CommandSucceededEvent) {
				var errorCode int32
				if wce := evt.Reply.Lookup("writeConcernError"); wce.Type == bson.TypeEmbeddedDocument {
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

				mt.SetFailPoint(failpoint.FailPoint{
					ConfigureFailPoint: "failCommand",
					Mode:               failpoint.Mode{Times: 1},
					Data: failpoint.Data{
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

	mtOpts = mtest.NewOptions().Topologies(mtest.Sharded).MinServerVersion("4.2")
	mt.RunOpts("retrying in sharded cluster", mtOpts, func(mt *mtest.T) {
		tests := []struct {
			name string

			// Note that setting this value greater than 2 will result in false
			// negatives. The current specification does not account for CSOT, which
			// might allow for an "infinite" number of retries over a period of time.
			// Because of this, we only track the "previous server".
			hostCount            int
			failpointErrorCode   int32
			expectedFailCount    int
			expectedSuccessCount int
		}{
			{
				name:                 "retry on different mongos",
				hostCount:            2,
				failpointErrorCode:   6, // HostUnreachable
				expectedFailCount:    2,
				expectedSuccessCount: 0,
			},
			{
				name:                 "retry on same mongos",
				hostCount:            1,
				failpointErrorCode:   6, // HostUnreachable
				expectedFailCount:    1,
				expectedSuccessCount: 1,
			},
		}

		for _, tc := range tests {
			mt.Run(tc.name, func(mt *mtest.T) {
				hosts, err := mongoutil.HostsFromURI(mtest.ClusterURI())

				require.NoError(mt, err)
				require.GreaterOrEqualf(mt, len(hosts), tc.hostCount,
					"test cluster must have at least %v mongos hosts", tc.hostCount)

				// Configure the failpoint options for each mongos.
				failPoint := failpoint.FailPoint{
					ConfigureFailPoint: "failCommand",
					Mode: failpoint.Mode{
						Times: 1,
					},
					Data: failpoint.Data{
						FailCommands:    []string{"insert"},
						ErrorLabels:     &[]string{"RetryableWriteError"},
						ErrorCode:       tc.failpointErrorCode,
						CloseConnection: false,
					},
				}

				// In order to ensure that each mongos in the hostCount-many mongos
				// hosts are tried at least once (i.e. failures are deprioritized), we
				// set a failpoint on all mongos hosts. The idea is that if we get
				// hostCount-many failures, then by the pigeonhole principal all mongos
				// hosts must have been tried.
				for i := 0; i < tc.hostCount; i++ {
					mt.ResetClient(options.Client().SetHosts([]string{hosts[i]}))
					mt.SetFailPoint(failPoint)

					// The automatic failpoint clearing may not clear failpoints set on
					// specific hosts, so manually clear the failpoint we set on the
					// specific mongos when the test is done.
					defer mt.ResetClient(options.Client().SetHosts([]string{hosts[i]}))
					defer mt.ClearFailPoints()
				}

				failCount := 0
				successCount := 0

				commandMonitor := &event.CommandMonitor{
					Failed: func(context.Context, *event.CommandFailedEvent) {
						failCount++
					},
					Succeeded: func(context.Context, *event.CommandSucceededEvent) {
						successCount++
					},
				}

				// Reset the client with exactly hostCount-many mongos hosts.
				mt.ResetClient(options.Client().
					SetHosts(hosts[:tc.hostCount]).
					SetRetryWrites(true).
					SetMonitor(commandMonitor))

				_, _ = mt.Coll.InsertOne(context.Background(), bson.D{})

				assert.Equal(mt, tc.expectedFailCount, failCount)
				assert.Equal(mt, tc.expectedSuccessCount, successCount)
			})
		}
	})
}

type crudOperation struct {
	Name      string   `bson:"name"`
	Arguments bson.Raw `bson:"arguments"`
}

type crudOutcome struct {
	Error      bool               `bson:"error"` // only used by retryable writes tests
	Result     interface{}        `bson:"result"`
	Collection *outcomeCollection `bson:"collection"`
}

// run a CRUD operation and verify errors and outcomes.
// the test description is needed to see determine if the test is an aggregate with $out
func runCrudOperation(mt *mtest.T, testDescription string, operation crudOperation, outcome crudOutcome) {
	switch operation.Name {
	case "aggregate":
		cursor, err := executeAggregate(mt, mt.Coll, nil, operation.Arguments)
		if outcome.Error {
			assert.NotNil(mt, err, "expected Aggregate error, got nil")
			verifyCrudError(mt, outcome, err)
			break
		}
		assert.Nil(mt, err, "Aggregate error: %v", err)
		// only verify cursor contents for pipelines without $out
		if !strings.Contains(testDescription, "$out") {
			verifyCursorResult(mt, cursor, outcome.Result)
		}
	case "bulkWrite":
		res, err := executeBulkWrite(mt, nil, operation.Arguments)
		if outcome.Error {
			assert.NotNil(mt, err, "expected BulkWrite error, got nil")
			verifyCrudError(mt, outcome, err)
			break
		}
		assert.Nil(mt, err, "BulkWrite error: %v", err)
		verifyBulkWriteResult(mt, res, outcome.Result)
	case "count":
		res, err := executeCountDocuments(mt, nil, operation.Arguments)
		if outcome.Error {
			assert.NotNil(mt, err, "expected CountDocuments error, got nil")
			verifyCrudError(mt, outcome, err)
			break
		}
		assert.Nil(mt, err, "CountDocuments error: %v", err)
		verifyCountResult(mt, res, outcome.Result)
	case "distinct":
		res, err := executeDistinct(mt, nil, operation.Arguments)
		if outcome.Error {
			assert.NotNil(mt, err, "expected Distinct error, got nil")
			verifyCrudError(mt, outcome, err)
			break
		}
		assert.Nil(mt, err, "Distinct error: %v", err)
		verifyDistinctResult(mt, res, outcome.Result)
	case "find":
		cursor, err := executeFind(mt, nil, operation.Arguments)
		if outcome.Error {
			assert.NotNil(mt, err, "expected Find error, got nil")
			verifyCrudError(mt, outcome, err)
			break
		}
		assert.Nil(mt, err, "Find error: %v", err)
		verifyCursorResult(mt, cursor, outcome.Result)
	case "deleteOne":
		res, err := executeDeleteOne(mt, nil, operation.Arguments)
		if outcome.Error {
			assert.NotNil(mt, err, "expected DeleteOne error, got nil")
			verifyCrudError(mt, outcome, err)
			break
		}
		assert.Nil(mt, err, "DeleteOne error: %v", err)
		verifyDeleteResult(mt, res, outcome.Result)
	case "deleteMany":
		res, err := executeDeleteMany(mt, nil, operation.Arguments)
		if outcome.Error {
			assert.NotNil(mt, err, "expected DeleteMany error, got nil")
			verifyCrudError(mt, outcome, err)
			break
		}
		assert.Nil(mt, err, "DeleteMany error: %v", err)
		verifyDeleteResult(mt, res, outcome.Result)
	case "findOneAndDelete":
		res := executeFindOneAndDelete(mt, nil, operation.Arguments)
		err := res.Err()
		if outcome.Error {
			assert.NotNil(mt, err, "expected FindOneAndDelete error, got nil")
			verifyCrudError(mt, outcome, err)
			break
		}
		if outcome.Result == nil {
			assert.Equal(mt, mongo.ErrNoDocuments, err, "expected error %v, got %v", mongo.ErrNoDocuments, err)
			break
		}
		assert.Nil(mt, err, "FindOneAndDelete error: %v", err)
		verifySingleResult(mt, res, outcome.Result)
	case "findOneAndReplace":
		res := executeFindOneAndReplace(mt, nil, operation.Arguments)
		err := res.Err()
		if outcome.Error {
			assert.NotNil(mt, err, "expected FindOneAndReplace error, got nil")
			verifyCrudError(mt, outcome, err)
			break
		}
		if outcome.Result == nil {
			assert.Equal(mt, mongo.ErrNoDocuments, err, "expected error %v, got %v", mongo.ErrNoDocuments, err)
			break
		}
		assert.Nil(mt, err, "FindOneAndReplace error: %v", err)
		verifySingleResult(mt, res, outcome.Result)
	case "findOneAndUpdate":
		res := executeFindOneAndUpdate(mt, nil, operation.Arguments)
		err := res.Err()
		if outcome.Error {
			assert.NotNil(mt, err, "expected FindOneAndUpdate error, got nil")
			verifyCrudError(mt, outcome, err)
			break
		}
		if outcome.Result == nil {
			assert.Equal(mt, mongo.ErrNoDocuments, err, "expected error %v, got %v", mongo.ErrNoDocuments, err)
			break
		}
		assert.Nil(mt, err, "FindOneAndUpdate error: %v", err)
		verifySingleResult(mt, res, outcome.Result)
	case "insertOne":
		res, err := executeInsertOne(mt, nil, operation.Arguments)
		if outcome.Error {
			assert.NotNil(mt, err, "expected InsertOne error, got nil")
			verifyCrudError(mt, outcome, err)
			break
		}
		assert.Nil(mt, err, "InsertOne error: %v", err)
		verifyInsertOneResult(mt, res, outcome.Result)
	case "insertMany":
		res, err := executeInsertMany(mt, nil, operation.Arguments)
		if outcome.Error {
			assert.NotNil(mt, err, "expected InsertMany error, got nil")
			verifyCrudError(mt, outcome, err)
			break
		}
		assert.Nil(mt, err, "InsertMany error: %v", err)
		verifyInsertManyResult(mt, res, outcome.Result)
	case "replaceOne":
		res, err := executeReplaceOne(mt, nil, operation.Arguments)
		if outcome.Error {
			assert.NotNil(mt, err, "expected ReplaceOne error, got nil")
			verifyCrudError(mt, outcome, err)
			break
		}
		assert.Nil(mt, err, "ReplaceOne error: %v", err)
		verifyUpdateResult(mt, res, outcome.Result)
	case "updateOne":
		res, err := executeUpdateOne(mt, nil, operation.Arguments)
		if outcome.Error {
			assert.NotNil(mt, err, "expected UpdateOne error, got nil")
			verifyCrudError(mt, outcome, err)
			break
		}
		assert.Nil(mt, err, "UpdateOne error: %v", err)
		verifyUpdateResult(mt, res, outcome.Result)
	case "updateMany":
		res, err := executeUpdateMany(mt, nil, operation.Arguments)
		if outcome.Error {
			assert.NotNil(mt, err, "expected UpdateMany error, got nil")
			verifyCrudError(mt, outcome, err)
			break
		}
		assert.Nil(mt, err, "UpdateMany error: %v", err)
		verifyUpdateResult(mt, res, outcome.Result)
	case "estimatedDocumentCount":
		res, err := executeEstimatedDocumentCount(mt, nil, operation.Arguments)
		if outcome.Error {
			assert.NotNil(mt, err, "expected EstimatedDocumentCount error, got nil")
			verifyCrudError(mt, outcome, err)
			break
		}
		assert.Nil(mt, err, "EstimatedDocumentCount error: %v", err)
		verifyCountResult(mt, res, outcome.Result)
	case "countDocuments":
		res, err := executeCountDocuments(mt, nil, operation.Arguments)
		if outcome.Error {
			assert.NotNil(mt, err, "expected CountDocuments error, got nil")
			verifyCrudError(mt, outcome, err)
			break
		}
		assert.Nil(mt, err, "CountDocuments error: %v", err)
		verifyCountResult(mt, res, outcome.Result)
	default:
		mt.Fatalf("unrecognized operation: %v", operation.Name)
	}

	if outcome.Collection != nil {
		verifyTestOutcome(mt, outcome.Collection)
	}
}

func verifyCrudError(mt *mtest.T, outcome crudOutcome, err error) {
	opError := errorFromResult(mt, outcome.Result)
	if opError == nil {
		return
	}
	verificationErr := verifyError(opError, err)
	assert.Nil(mt, verificationErr, "error mismatch: %v", verificationErr)
}
