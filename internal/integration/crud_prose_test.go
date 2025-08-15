// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/failpoint"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
)

func TestWriteErrorsWithLabels(t *testing.T) {
	clientOpts := options.Client().SetRetryWrites(false).SetWriteConcern(mtest.MajorityWc).
		SetReadConcern(mtest.MajorityRc)
	mtOpts := mtest.NewOptions().ClientOptions(clientOpts).MinServerVersion("4.0").Topologies(mtest.ReplicaSet).
		CreateClient(false)
	mt := mtest.New(t, mtOpts)

	label := "ExampleError"
	mt.Run("InsertMany errors with label", func(mt *mtest.T) {
		mt.SetFailPoint(failpoint.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: failpoint.Mode{
				Times: 1,
			},
			Data: failpoint.Data{
				FailCommands: []string{"insert"},
				WriteConcernError: &failpoint.WriteConcernError{
					Code:        100,
					ErrorLabels: &[]string{label},
				},
			},
		})

		_, err := mt.Coll.InsertMany(context.Background(),
			[]any{
				bson.D{
					{"a", 1},
				},
				bson.D{
					{"a", 2},
				},
			})
		assert.NotNil(mt, err, "expected non-nil error, got nil")

		we, ok := err.(mongo.BulkWriteException)
		assert.True(mt, ok, "expected mongo.BulkWriteException, got %T", err)
		assert.True(mt, we.HasErrorLabel(label), "expected error to have label: %v", label)
	})

	mt.Run("WriteException with label", func(mt *mtest.T) {
		mt.SetFailPoint(failpoint.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: failpoint.Mode{
				Times: 1,
			},
			Data: failpoint.Data{
				FailCommands: []string{"delete"},
				WriteConcernError: &failpoint.WriteConcernError{
					Code:        100,
					ErrorLabels: &[]string{label},
				},
			},
		})

		_, err := mt.Coll.DeleteMany(context.Background(), bson.D{{"a", 1}})
		assert.NotNil(mt, err, "expected non-nil error, got nil")

		we, ok := err.(mongo.WriteException)
		assert.True(mt, ok, "expected mongo.WriteException, got %T", err)
		assert.True(mt, we.HasErrorLabel(label), "expected error to have label: %v", label)
	})

	mt.Run("BulkWriteException with label", func(mt *mtest.T) {
		mt.SetFailPoint(failpoint.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: failpoint.Mode{
				Times: 1,
			},
			Data: failpoint.Data{
				FailCommands: []string{"delete"},
				WriteConcernError: &failpoint.WriteConcernError{
					Code:        100,
					ErrorLabels: &[]string{label},
				},
			},
		})

		models := []mongo.WriteModel{
			&mongo.InsertOneModel{bson.D{{"a", 2}}},
			&mongo.DeleteOneModel{bson.D{{"a", 2}}, nil, nil},
		}
		_, err := mt.Coll.BulkWrite(context.Background(), models)
		assert.NotNil(mt, err, "expected non-nil error, got nil")

		we, ok := err.(mongo.BulkWriteException)
		assert.True(mt, ok, "expected mongo.BulkWriteException, got %T", err)
		assert.True(mt, we.HasErrorLabel(label), "expected error to have label: %v", label)
	})

}

func TestWriteErrorsDetails(t *testing.T) {
	clientOpts := options.Client().
		SetRetryWrites(false).
		SetWriteConcern(mtest.MajorityWc).
		SetReadConcern(mtest.MajorityRc)
	mtOpts := mtest.NewOptions().
		ClientOptions(clientOpts).
		MinServerVersion("5.0").
		Topologies(mtest.ReplicaSet, mtest.Single).
		CreateClient(false)

	mt := mtest.New(t, mtOpts)

	mt.Run("JSON Schema validation", func(mt *mtest.T) {
		// Create a JSON Schema validator document that requires properties "a" and "b". Use it in
		// the collection creation options so that collections created for subtests have the JSON
		// Schema validator applied.
		validator := bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": []string{"a", "b"},
				"properties": bson.M{
					"a": bson.M{"bsonType": "string"},
					"b": bson.M{"bsonType": "int"},
				},
			},
		}

		cco := options.CreateCollection().SetValidator(validator)
		validatorOpts := mtest.NewOptions().CollectionCreateOptions(cco)

		cases := []struct {
			desc                string
			operation           func(*mongo.Collection) error
			expectBulkError     bool
			expectedCommandName string
		}{
			{
				desc: "InsertOne schema validation errors should include Details",
				operation: func(coll *mongo.Collection) error {
					// Try to insert a document that doesn't contain the required properties.
					_, err := coll.InsertOne(context.Background(), bson.D{{"nope", 1}})
					return err
				},
				expectBulkError:     false,
				expectedCommandName: "insert",
			},
			{
				desc: "InsertMany schema validation errors should include Details",
				operation: func(coll *mongo.Collection) error {
					// Try to insert a document that doesn't contain the required properties.
					_, err := coll.InsertMany(context.Background(), []any{bson.D{{"nope", 1}}})
					return err
				},
				expectBulkError:     true,
				expectedCommandName: "insert",
			},
			{
				desc: "UpdateOne schema validation errors should include Details",
				operation: func(coll *mongo.Collection) error {
					// Try to set "a" to be an int, which violates the string type requirement.
					_, err := coll.UpdateOne(
						context.Background(),
						bson.D{},
						bson.D{{"$set", bson.D{{"a", 1}}}})
					return err
				},
				expectBulkError:     false,
				expectedCommandName: "update",
			},
			{
				desc: "UpdateMany schema validation errors should include Details",
				operation: func(coll *mongo.Collection) error {
					// Try to set "a" to be an int in all documents in the collection, which violates
					// the string type requirement.
					_, err := coll.UpdateMany(
						context.Background(),
						bson.D{},
						bson.D{{"$set", bson.D{{"a", 1}}}})
					return err
				},
				expectBulkError:     false,
				expectedCommandName: "update",
			},
		}

		for _, tc := range cases {
			mt.RunOpts(tc.desc, validatorOpts, func(mt *mtest.T) {
				// Insert two valid documents so that the Update* tests can try to update them.
				{
					_, err := mt.Coll.InsertMany(
						context.Background(),
						[]any{
							bson.D{{"a", "str1"}, {"b", 1}},
							bson.D{{"a", "str2"}, {"b", 2}},
						})
					assert.Nil(mt, err, "unexpected error inserting valid documents: %s", err)
				}

				err := tc.operation(mt.Coll)
				assert.NotNil(mt, err, "expected an error from calling the operation")
				sErr := err.(mongo.ServerError)
				assert.True(
					mt,
					sErr.HasErrorCode(121),
					"expected mongo.ServerError to have error code 121 (DocumentValidationFailure)")

				var details bson.Raw
				if tc.expectBulkError {
					bwe, ok := err.(mongo.BulkWriteException)
					assert.True(
						mt,
						ok,
						"expected error to be type mongo.BulkWriteException, got type %T (error %q)",
						err,
						err)
					// Assert that there is one WriteError and that the Details field is populated.
					assert.Equal(
						mt,
						1,
						len(bwe.WriteErrors),
						"expected exactly 1 write error, but got %d write errors (error %q)",
						len(bwe.WriteErrors),
						err)
					details = bwe.WriteErrors[0].Details
				} else {
					we, ok := err.(mongo.WriteException)
					assert.True(
						mt,
						ok,
						"expected error to be type mongo.WriteException, got type %T (error %q)",
						err,
						err)
					// Assert that there is one WriteError and that the Details field is populated.
					assert.Equal(
						mt,
						1,
						len(we.WriteErrors),
						"expected exactly 1 write error, but got %d write errors (error %q)",
						len(we.WriteErrors),
						err)
					details = we.WriteErrors[0].Details
				}

				assert.True(
					mt,
					len(details) > 0,
					"expected WriteError.Details to be populated, but is empty")

				// Assert that the most recent CommandSucceededEvent was triggered by the expected
				// operation and contains the resulting write errors and that
				// "writeErrors[0].errInfo" is the same as "WriteException.WriteErrors[0].Details".
				evts := mt.GetAllSucceededEvents()
				assert.True(
					mt,
					len(evts) >= 2,
					"expected there to be at least 2 CommandSucceededEvent recorded")
				evt := evts[len(evts)-1]
				assert.Equal(
					mt,
					tc.expectedCommandName,
					evt.CommandName,
					"expected the last CommandSucceededEvent to be for %q, was %q",
					tc.expectedCommandName,
					evt.CommandName)
				errInfo, ok := evt.Reply.Lookup("writeErrors", "0", "errInfo").DocumentOK()
				assert.True(
					mt,
					ok,
					"expected evt.Reply to contain writeErrors[0].errInfo but doesn't (evt.Reply = %v)",
					evt.Reply)
				assert.Equal(mt, details, errInfo, "want %v, got %v", details, errInfo)
			})
		}
	})
}

func TestHintErrors(t *testing.T) {
	mtOpts := mtest.NewOptions().MaxServerVersion("3.2").CreateClient(false)
	mt := mtest.New(t, mtOpts)

	expected := errors.New("the 'hint' command parameter requires a minimum server wire version of 5")
	mt.Run("UpdateMany", func(mt *mtest.T) {

		_, got := mt.Coll.UpdateMany(context.Background(), bson.D{{"a", 1}}, bson.D{{"$inc", bson.D{{"a", 1}}}},
			options.UpdateMany().SetHint("_id_"))
		assert.NotNil(mt, got, "expected non-nil error, got nil")
		assert.Equal(mt, got, expected, "expected: %v got: %v", expected, got)
	})

	mt.Run("ReplaceOne", func(mt *mtest.T) {

		_, got := mt.Coll.ReplaceOne(context.Background(), bson.D{{"a", 1}}, bson.D{{"a", 2}},
			options.Replace().SetHint("_id_"))
		assert.NotNil(mt, got, "expected non-nil error, got nil")
		assert.Equal(mt, got, expected, "expected: %v got: %v", expected, got)
	})

	mt.Run("BulkWrite", func(mt *mtest.T) {
		models := []mongo.WriteModel{
			&mongo.InsertOneModel{bson.D{{"_id", 2}}},
			&mongo.ReplaceOneModel{Filter: bson.D{{"_id", 2}}, Replacement: bson.D{{"a", 2}}, Hint: "_id_"},
		}
		_, got := mt.Coll.BulkWrite(context.Background(), models)
		assert.NotNil(mt, got, "expected non-nil error, got nil")
		assert.Equal(mt, got, expected, "expected: %v got: %v", expected, got)
	})
}

func TestWriteConcernError(t *testing.T) {
	mt := mtest.New(t, noClientOpts)

	errInfoOpts := mtest.NewOptions().MinServerVersion("4.0").Topologies(mtest.ReplicaSet)
	mt.RunOpts("errInfo is propagated", errInfoOpts, func(mt *mtest.T) {
		wcDoc := bsoncore.BuildDocumentFromElements(nil,
			bsoncore.AppendInt32Element(nil, "w", 2),
			bsoncore.AppendInt32Element(nil, "wtimeout", 0),
			bsoncore.AppendStringElement(nil, "provenance", "clientSupplied"),
		)
		errInfoDoc := bsoncore.BuildDocumentFromElements(nil,
			bsoncore.AppendDocumentElement(nil, "writeConcern", wcDoc),
		)
		fp := failpoint.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: failpoint.Mode{
				Times: 1,
			},
			Data: failpoint.Data{
				FailCommands: []string{"insert"},
				WriteConcernError: &failpoint.WriteConcernError{
					Code:    100,
					Name:    "UnsatisfiableWriteConcern",
					Errmsg:  "Not enough data-bearing nodes",
					ErrInfo: errInfoDoc,
				},
			},
		}
		mt.SetFailPoint(fp)

		_, err := mt.Coll.InsertOne(context.Background(), bson.D{{"x", 1}})
		assert.NotNil(mt, err, "expected InsertOne error, got nil")
		writeException, ok := err.(mongo.WriteException)
		assert.True(mt, ok, "expected WriteException, got error %v of type %T", err, err)
		wcError := writeException.WriteConcernError
		assert.NotNil(mt, wcError, "expected write-concern error, got %v", err)
		assert.True(mt, bytes.Equal(wcError.Details, errInfoDoc), "expected errInfo document %v, got %v",
			bson.Raw(errInfoDoc), wcError.Details)
	})
}

func TestErrorsCodeNamePropagated(t *testing.T) {
	// Ensure the codeName field is propagated for both command and write concern errors.

	mtOpts := mtest.NewOptions().
		Topologies(mtest.ReplicaSet).
		CreateClient(false)
	mt := mtest.New(t, mtOpts)

	mt.RunOpts("command error", mtest.NewOptions().MinServerVersion("3.4"), func(mt *mtest.T) {
		// codeName is propagated in an ok:0 error.

		cmd := bson.D{
			{"insert", mt.Coll.Name()},
			{"documents", []bson.D{}},
		}
		err := mt.DB.RunCommand(context.Background(), cmd).Err()
		assert.NotNil(mt, err, "expected RunCommand error, got nil")

		ce, ok := err.(mongo.CommandError)
		assert.True(mt, ok, "expected error of type %T, got %v of type %T", mongo.CommandError{}, err, err)
		expectedCodeName := "InvalidLength"
		assert.Equal(mt, expectedCodeName, ce.Name, "expected error code name %q, got %q", expectedCodeName, ce.Name)
	})

	wcCollOpts := options.Collection().
		SetWriteConcern(impossibleWc)
	wcMtOpts := mtest.NewOptions().
		CollectionOptions(wcCollOpts)
	mt.RunOpts("write concern error", wcMtOpts, func(mt *mtest.T) {
		// codeName is propagated for write concern errors.

		_, err := mt.Coll.InsertOne(context.Background(), bson.D{})
		assert.NotNil(mt, err, "expected InsertOne error, got nil")

		we, ok := err.(mongo.WriteException)
		assert.True(mt, ok, "expected error of type %T, got %v of type %T", mongo.WriteException{}, err, err)
		wce := we.WriteConcernError
		assert.NotNil(mt, wce, "expected write concern error, got %v", we)

		var expectedCodeName string
		if codeNameVal, err := mt.GetSucceededEvent().Reply.LookupErr("writeConcernError", "codeName"); err == nil {
			expectedCodeName = codeNameVal.StringValue()
		}

		assert.Equal(mt, expectedCodeName, wce.Name, "expected code name %q, got %q", expectedCodeName, wce.Name)
	})
}

func TestClientBulkWriteProse(t *testing.T) {
	mtOpts := mtest.NewOptions().MinServerVersion("8.0").AtlasDataLake(false).ClientType(mtest.Pinned)
	mt := mtest.New(t, mtOpts)

	mt.Run("3. MongoClient.bulkWrite batch splits a writeModels input with greater than maxWriteBatchSize operations", func(mt *mtest.T) {
		var opsCnt []int
		monitor := &event.CommandMonitor{
			Started: func(_ context.Context, e *event.CommandStartedEvent) {
				if e.CommandName == "bulkWrite" {
					var c struct {
						Ops []bson.D
					}
					err := bson.Unmarshal(e.Command, &c)
					require.NoError(mt, err)
					opsCnt = append(opsCnt, len(c.Ops))
				}
			},
		}
		mt.ResetClient(options.Client().SetMonitor(monitor))
		var hello struct {
			MaxWriteBatchSize int
		}
		err := mt.DB.RunCommand(context.Background(), bson.D{{"hello", 1}}).Decode(&hello)
		require.NoError(mt, err, "Hello error: %v", err)
		var writes []mongo.ClientBulkWrite
		num := hello.MaxWriteBatchSize + 1
		for i := 0; i < num; i++ {
			writes = append(writes, mongo.ClientBulkWrite{
				Database:   "db",
				Collection: "coll",
				Model: &mongo.ClientInsertOneModel{
					Document: bson.D{{"a", "b"}},
				},
			})
		}
		result, err := mt.Client.BulkWrite(context.Background(), writes)
		require.NoError(mt, err, "BulkWrite error: %v", err)
		assert.Equal(mt, num, int(result.InsertedCount), "expected InsertedCount: %d, got %d", num, result.InsertedCount)
		require.Len(mt, opsCnt, 2, "expected %d bulkWrite commands, got: %d", 2, len(opsCnt))
		assert.Equal(mt, num-1, opsCnt[0], "expected %d firstEvent.command.ops, got: %d", num-1, opsCnt[0])
		assert.Equal(mt, 1, opsCnt[1], "expected %d secondEvent.command.ops, got: %d", 1, opsCnt[1])
	})

	mt.Run("4. MongoClient.bulkWrite batch splits when an ops payload exceeds maxMessageSizeBytes", func(mt *mtest.T) {
		var opsCnt []int
		monitor := &event.CommandMonitor{
			Started: func(_ context.Context, e *event.CommandStartedEvent) {
				if e.CommandName == "bulkWrite" {
					var c struct {
						Ops []bson.D
					}
					err := bson.Unmarshal(e.Command, &c)
					require.NoError(mt, err)
					opsCnt = append(opsCnt, len(c.Ops))
				}
			},
		}
		mt.ResetClient(options.Client().SetMonitor(monitor))
		var hello struct {
			MaxBsonObjectSize   int
			MaxMessageSizeBytes int
		}
		err := mt.DB.RunCommand(context.Background(), bson.D{{"hello", 1}}).Decode(&hello)
		require.NoError(mt, err, "Hello error: %v", err)
		var writes []mongo.ClientBulkWrite
		num := hello.MaxMessageSizeBytes/hello.MaxBsonObjectSize + 1
		for i := 0; i < num; i++ {
			writes = append(writes, mongo.ClientBulkWrite{
				Database:   "db",
				Collection: "coll",
				Model: &mongo.ClientInsertOneModel{
					Document: bson.D{{"a", strings.Repeat("b", hello.MaxBsonObjectSize-500)}},
				},
			})
		}
		result, err := mt.Client.BulkWrite(context.Background(), writes)
		require.NoError(mt, err, "BulkWrite error: %v", err)
		assert.Equal(mt, num, int(result.InsertedCount), "expected InsertedCount: %d, got: %d", num, result.InsertedCount)
		require.Len(mt, opsCnt, 2, "expected %d bulkWrite commands, got: %d", 2, len(opsCnt))
		assert.Equal(mt, num-1, opsCnt[0], "expected %d firstEvent.command.ops, got: %d", num-1, opsCnt[0])
		assert.Equal(mt, 1, opsCnt[1], "expected %d secondEvent.command.ops, got: %d", 1, opsCnt[1])
	})

	// TODO(GODRIVER-3328): FailPoints are not currently reliable on sharded
	// topologies. Allow running on sharded topologies once that is fixed.
	noShardedOpts := mtest.NewOptions().Topologies(mtest.Single, mtest.ReplicaSet, mtest.LoadBalanced)
	mt.RunOpts("5. MongoClient.bulkWrite collects WriteConcernErrors across batches", noShardedOpts, func(mt *mtest.T) {
		var eventCnt int
		monitor := &event.CommandMonitor{
			Started: func(_ context.Context, e *event.CommandStartedEvent) {
				if e.CommandName == "bulkWrite" {
					eventCnt++
				}
			},
		}
		mt.ResetClient(options.Client().SetRetryWrites(false).SetMonitor(monitor))
		var hello struct {
			MaxWriteBatchSize int
		}
		err := mt.DB.RunCommand(context.Background(), bson.D{{"hello", 1}}).Decode(&hello)
		require.NoError(mt, err, "Hello error: %v", err)

		mt.SetFailPoint(failpoint.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: failpoint.Mode{
				Times: 2,
			},
			Data: failpoint.Data{
				FailCommands: []string{"bulkWrite"},
				WriteConcernError: &failpoint.WriteConcernError{
					Code:   91,
					Errmsg: "Replication is being shut down",
				},
			},
		})

		var writes []mongo.ClientBulkWrite
		num := hello.MaxWriteBatchSize + 1
		for i := 0; i < num; i++ {
			writes = append(writes, mongo.ClientBulkWrite{
				Database:   "db",
				Collection: "coll",
				Model: &mongo.ClientInsertOneModel{
					Document: bson.D{{"a", "b"}},
				},
			})
		}
		_, err = mt.Client.BulkWrite(context.Background(), writes)
		require.Error(mt, err, "expected a BulkWrite error")
		bwe, ok := err.(mongo.ClientBulkWriteException)
		require.True(mt, ok, "expected a BulkWriteException, got %T: %v", err, err)
		assert.Len(mt, bwe.WriteConcernErrors, 2, "expected %d writeConcernErrors, got: %d", 2, len(bwe.WriteConcernErrors))
		require.NotNil(mt, bwe.PartialResult)
		assert.Equal(mt, num, int(bwe.PartialResult.InsertedCount),
			"expected InsertedCount: %d, got: %d", num, bwe.PartialResult.InsertedCount)
		require.Equal(mt, 2, eventCnt, "expected %d bulkWrite commands, got: %d", 2, eventCnt)
	})

	mt.Run("6. MongoClient.bulkWrite handles individual WriteErrors across batches", func(mt *mtest.T) {
		var eventCnt int
		monitor := &event.CommandMonitor{
			Started: func(_ context.Context, e *event.CommandStartedEvent) {
				if e.CommandName == "bulkWrite" {
					eventCnt++
				}
			},
		}

		mt.ResetClient(options.Client())
		var hello struct {
			MaxWriteBatchSize int
		}
		err := mt.DB.RunCommand(context.Background(), bson.D{{"hello", 1}}).Decode(&hello)
		require.NoError(mt, err, "Hello error: %v", err)

		coll := mt.CreateCollection(mtest.Collection{DB: "db", Name: "coll"}, false)
		err = coll.Drop(context.Background())
		require.NoError(mt, err, "Drop error: %v", err)
		_, err = coll.InsertOne(context.Background(), bson.D{{"_id", 1}})
		require.NoError(mt, err, "InsertOne error: %v", err)

		var writes []mongo.ClientBulkWrite
		numModels := hello.MaxWriteBatchSize + 1
		for i := 0; i < numModels; i++ {
			writes = append(writes, mongo.ClientBulkWrite{
				Database:   "db",
				Collection: "coll",
				Model: &mongo.ClientInsertOneModel{
					Document: bson.D{{"_id", 1}},
				},
			})
		}

		mt.Run("unordered", func(mt *mtest.T) {
			eventCnt = 0
			mt.ResetClient(options.Client().SetMonitor(monitor))
			_, err := mt.Client.BulkWrite(context.Background(), writes, options.ClientBulkWrite().SetOrdered(false))
			require.Error(mt, err, "expected a BulkWrite error")
			bwe, ok := err.(mongo.ClientBulkWriteException)
			require.True(mt, ok, "expected a BulkWriteException, got %T: %v", err, err)
			assert.Len(mt, bwe.WriteErrors, numModels, "expected %d writeErrors, got %d", numModels, len(bwe.WriteErrors))
			require.Equal(mt, 2, eventCnt, "expected %d bulkWrite commands, got: %d", 2, eventCnt)
		})
		mt.Run("ordered", func(mt *mtest.T) {
			eventCnt = 0
			mt.ResetClient(options.Client().SetMonitor(monitor))
			_, err := mt.Client.BulkWrite(context.Background(), writes, options.ClientBulkWrite().SetOrdered(true))
			require.Error(mt, err, "expected a BulkWrite error")
			bwe, ok := err.(mongo.ClientBulkWriteException)
			require.True(mt, ok, "expected a BulkWriteException, got %T: %v", err, err)
			assert.Len(mt, bwe.WriteErrors, 1, "expected %d writeErrors, got: %d", 1, len(bwe.WriteErrors))
			require.Equal(mt, 1, eventCnt, "expected %d bulkWrite commands, got: %d", 1, eventCnt)
		})
	})

	mt.Run("7. MongoClient.bulkWrite handles a cursor requiring a getMore", func(mt *mtest.T) {
		var getMoreCalled int
		monitor := &event.CommandMonitor{
			Started: func(_ context.Context, e *event.CommandStartedEvent) {
				if e.CommandName == "getMore" {
					getMoreCalled++
				}
			},
		}
		mt.ResetClient(options.Client().SetMonitor(monitor))
		var hello struct {
			MaxBsonObjectSize int
		}
		err := mt.DB.RunCommand(context.Background(), bson.D{{"hello", 1}}).Decode(&hello)
		require.NoError(mt, err, "Hello error: %v", err)

		coll := mt.CreateCollection(mtest.Collection{DB: "db", Name: "coll"}, false)
		err = coll.Drop(context.Background())
		require.NoError(mt, err, "Drop error: %v", err)

		upsert := true
		models := []mongo.ClientBulkWrite{
			{
				Database:   "db",
				Collection: "coll",
				Model: &mongo.ClientUpdateOneModel{
					Filter: bson.D{{"_id", strings.Repeat("a", hello.MaxBsonObjectSize/2)}},
					Update: bson.D{{"$set", bson.D{{"x", 1}}}},
					Upsert: &upsert,
				},
			},
			{
				Database:   "db",
				Collection: "coll",
				Model: &mongo.ClientUpdateOneModel{
					Filter: bson.D{{"_id", strings.Repeat("b", hello.MaxBsonObjectSize/2)}},
					Update: bson.D{{"$set", bson.D{{"x", 1}}}},
					Upsert: &upsert,
				},
			},
		}
		result, err := mt.Client.BulkWrite(context.Background(), models, options.ClientBulkWrite().SetVerboseResults(true))
		require.NoError(mt, err, "BulkWrite error: %v", err)
		assert.Equal(mt, int64(2), result.UpsertedCount, "expected InsertedCount: %d, got: %d", 2, result.UpsertedCount)
		assert.Len(mt, result.UpdateResults, 2, "expected %d UpdateResults, got: %d", 2, len(result.UpdateResults))
		assert.Equal(mt, 1, getMoreCalled, "expected %d getMore call, got: %d", 1, getMoreCalled)
	})

	mt.RunOpts("8. MongoClient.bulkWrite handles a cursor requiring getMore within a transaction",
		mtest.NewOptions().MinServerVersion("8.0").AtlasDataLake(false).ClientType(mtest.Pinned).
			Topologies(mtest.ReplicaSet, mtest.Sharded, mtest.LoadBalanced, mtest.ShardedReplicaSet),
		func(mt *mtest.T) {
			var getMoreCalled int
			monitor := &event.CommandMonitor{
				Started: func(_ context.Context, e *event.CommandStartedEvent) {
					if e.CommandName == "getMore" {
						getMoreCalled++
					}
				},
			}
			mt.ResetClient(options.Client().SetMonitor(monitor))
			var hello struct {
				MaxBsonObjectSize int
			}
			err := mt.DB.RunCommand(context.Background(), bson.D{{"hello", 1}}).Decode(&hello)
			require.NoError(mt, err, "Hello error: %v", err)

			coll := mt.CreateCollection(mtest.Collection{DB: "db", Name: "coll"}, false)
			err = coll.Drop(context.Background())
			require.NoError(mt, err, "Drop error: %v", err)

			session, err := mt.Client.StartSession()
			require.NoError(mt, err, "StartSession error: %v", err)
			defer session.EndSession(context.Background())

			upsert := true
			models := []mongo.ClientBulkWrite{
				{
					Database:   "db",
					Collection: "coll",
					Model: &mongo.ClientUpdateOneModel{
						Filter: bson.D{{"_id", strings.Repeat("a", hello.MaxBsonObjectSize/2)}},
						Update: bson.D{{"$set", bson.D{{"x", 1}}}},
						Upsert: &upsert,
					},
				},
				{
					Database:   "db",
					Collection: "coll",
					Model: &mongo.ClientUpdateOneModel{
						Filter: bson.D{{"_id", strings.Repeat("b", hello.MaxBsonObjectSize/2)}},
						Update: bson.D{{"$set", bson.D{{"x", 1}}}},
						Upsert: &upsert,
					},
				},
			}
			result, err := session.WithTransaction(context.Background(), func(ctx context.Context) (any, error) {
				return mt.Client.BulkWrite(ctx, models, options.ClientBulkWrite().SetVerboseResults(true))
			})
			require.NoError(mt, err, "BulkWrite error: %v", err)
			cbwResult, ok := result.(*mongo.ClientBulkWriteResult)
			require.True(mt, ok, "expected a ClientBulkWriteResult, got %T", result)
			assert.Equal(mt, int64(2), cbwResult.UpsertedCount, "expected InsertedCount: %d, got: %d", 2, cbwResult.UpsertedCount)
			assert.Len(mt, cbwResult.UpdateResults, 2, "expected %d UpdateResults, got: %d", 2, len(cbwResult.UpdateResults))
			assert.Equal(mt, 1, getMoreCalled, "expected %d getMore call, got: %d", 1, getMoreCalled)
		})

	// TODO(GODRIVER-3328): FailPoints are not currently reliable on sharded
	// topologies. Allow running on sharded topologies once that is fixed.
	mt.RunOpts("9. MongoClient.bulkWrite handles a getMore error", noShardedOpts, func(mt *mtest.T) {
		var getMoreCalled int
		var killCursorsCalled int
		monitor := &event.CommandMonitor{
			Started: func(_ context.Context, e *event.CommandStartedEvent) {
				switch e.CommandName {
				case "getMore":
					getMoreCalled++
				case "killCursors":
					killCursorsCalled++
				}
			},
		}
		mt.ResetClient(options.Client().SetMonitor(monitor))
		var hello struct {
			MaxBsonObjectSize int
		}
		err := mt.DB.RunCommand(context.Background(), bson.D{{"hello", 1}}).Decode(&hello)
		require.NoError(mt, err, "Hello error: %v", err)

		mt.SetFailPoint(failpoint.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: failpoint.Mode{
				Times: 1,
			},
			Data: failpoint.Data{
				FailCommands: []string{"getMore"},
				ErrorCode:    8,
			},
		})

		coll := mt.CreateCollection(mtest.Collection{DB: "db", Name: "coll"}, false)
		err = coll.Drop(context.Background())
		require.NoError(mt, err, "Drop error: %v", err)

		upsert := true
		models := []mongo.ClientBulkWrite{
			{
				Database:   "db",
				Collection: "coll",
				Model: &mongo.ClientUpdateOneModel{
					Filter: bson.D{{"_id", strings.Repeat("a", hello.MaxBsonObjectSize/2)}},
					Update: bson.D{{"$set", bson.D{{"x", 1}}}},
					Upsert: &upsert,
				},
			},
			{
				Database:   "db",
				Collection: "coll",
				Model: &mongo.ClientUpdateOneModel{
					Filter: bson.D{{"_id", strings.Repeat("b", hello.MaxBsonObjectSize/2)}},
					Update: bson.D{{"$set", bson.D{{"x", 1}}}},
					Upsert: &upsert,
				},
			},
		}
		_, err = mt.Client.BulkWrite(context.Background(), models, options.ClientBulkWrite().SetVerboseResults(true))
		assert.Error(mt, err, "expected a BulkWrite error")
		bwe, ok := err.(mongo.ClientBulkWriteException)
		require.True(mt, ok, "expected a BulkWriteException, got %T: %v", err, err)
		require.NotNil(mt, bwe.WriteError)
		assert.Equal(mt, 8, bwe.WriteError.Code, "expected top level error code: %d, got; %d", 8, bwe.WriteError.Code)
		require.NotNil(mt, bwe.PartialResult)
		assert.Equal(mt, int64(2), bwe.PartialResult.UpsertedCount, "expected UpsertedCount: %d, got: %d", 2, bwe.PartialResult.UpsertedCount)
		assert.Len(mt, bwe.PartialResult.UpdateResults, 1, "expected %d UpdateResults, got: %d", 1, len(bwe.PartialResult.UpdateResults))
		assert.Equal(mt, 1, getMoreCalled, "expected %d getMore call, got: %d", 1, getMoreCalled)
		assert.Equal(mt, 1, killCursorsCalled, "expected %d killCursors call, got: %d", 1, killCursorsCalled)
	})

	mt.Run("11. MongoClient.bulkWrite batch splits when the addition of a new namespace exceeds the maximum message size", func(mt *mtest.T) {
		type cmd struct {
			Ops    []bson.D
			NsInfo []struct {
				Ns string
			}
		}
		var bwCmd []cmd
		monitor := &event.CommandMonitor{
			Started: func(_ context.Context, e *event.CommandStartedEvent) {
				if e.CommandName == "bulkWrite" {
					var c cmd
					err := bson.Unmarshal(e.Command, &c)
					require.NoError(mt, err, "Unmarshal error: %v", err)
					bwCmd = append(bwCmd, c)
				}
			},
		}
		mt.ResetClient(options.Client())
		var hello struct {
			MaxBsonObjectSize   int
			MaxMessageSizeBytes int
		}
		err := mt.DB.RunCommand(context.Background(), bson.D{{"hello", 1}}).Decode(&hello)
		require.NoError(mt, err, "Hello error: %v", err)

		newWrites := func() (int, []mongo.ClientBulkWrite) {
			maxBsonObjectSize := hello.MaxBsonObjectSize
			opsBytes := hello.MaxMessageSizeBytes - 1122
			num := opsBytes / maxBsonObjectSize

			var writes []mongo.ClientBulkWrite
			for i := 0; i < num; i++ {
				writes = append(writes, mongo.ClientBulkWrite{
					Database:   "db",
					Collection: "coll",
					Model: &mongo.ClientInsertOneModel{
						Document: bson.D{{"a", strings.Repeat("b", maxBsonObjectSize-57)}},
					},
				})
			}
			if remainderBytes := opsBytes % maxBsonObjectSize; remainderBytes > 217 {
				num++
				writes = append(writes, mongo.ClientBulkWrite{
					Database:   "db",
					Collection: "coll",
					Model: &mongo.ClientInsertOneModel{
						Document: bson.D{{"a", strings.Repeat("b", remainderBytes-57)}},
					},
				})
			}
			return num, writes
		}
		mt.Run("Case 1: No batch-splitting required", func(mt *mtest.T) {
			bwCmd = bwCmd[:0]
			mt.ResetClient(options.Client().SetMonitor(monitor))

			num, writes := newWrites()
			writes = append(writes, mongo.ClientBulkWrite{
				Database:   "db",
				Collection: "coll",
				Model: &mongo.ClientInsertOneModel{
					Document: bson.D{{"a", "b"}},
				},
			})
			result, err := mt.Client.BulkWrite(context.Background(), writes)
			require.NoError(mt, err, "BulkWrite error: %v", err)
			assert.Equal(mt, num+1, int(result.InsertedCount), "expected insertedCound: %d, got: %d", num+1, result.InsertedCount)
			require.Len(mt, bwCmd, 1, "expected %d bulkWrite call, got: %d", 1, len(bwCmd))

			assert.Len(mt, bwCmd[0].Ops, num+1, "expected %d ops, got: %d", num+1, len(bwCmd[0].Ops))
			require.Len(mt, bwCmd[0].NsInfo, 1, "expected %d nsInfo, got: %d", 1, len(bwCmd[0].NsInfo))
			assert.Equal(mt, "db.coll", bwCmd[0].NsInfo[0].Ns, "expected namespace: %s, got: %s", "db.coll", bwCmd[0].NsInfo[0].Ns)
		})
		mt.Run("Case 2: Batch-splitting required", func(mt *mtest.T) {
			bwCmd = bwCmd[:0]
			mt.ResetClient(options.Client().SetMonitor(monitor))

			coll := strings.Repeat("c", 200)
			num, writes := newWrites()
			writes = append(writes, mongo.ClientBulkWrite{
				Database:   "db",
				Collection: coll,
				Model: &mongo.ClientInsertOneModel{
					Document: bson.D{{"a", "b"}},
				},
			})
			result, err := mt.Client.BulkWrite(context.Background(), writes)
			require.NoError(mt, err, "BulkWrite error: %v", err)
			assert.Equal(mt, num+1, int(result.InsertedCount), "expected insertedCound: %d, got: %d", num+1, result.InsertedCount)
			require.Len(mt, bwCmd, 2, "expected %d bulkWrite calls, got: %d", 2, len(bwCmd))

			assert.Len(mt, bwCmd[0].Ops, num, "expected %d ops, got: %d", num, len(bwCmd[0].Ops))
			require.Len(mt, bwCmd[0].NsInfo, 1, "expected %d nsInfo, got: %d", 1, len(bwCmd[0].NsInfo))
			assert.Equal(mt, "db.coll", bwCmd[0].NsInfo[0].Ns, "expected namespace: %s, got: %s", "db.coll", bwCmd[0].NsInfo[0].Ns)

			assert.Len(mt, bwCmd[1].Ops, 1, "expected %d ops, got: %d", 1, len(bwCmd[1].Ops))
			require.Len(mt, bwCmd[1].NsInfo, 1, "expected %d nsInfo, got: %d", 1, len(bwCmd[1].NsInfo))
			assert.Equal(mt, "db."+coll, bwCmd[1].NsInfo[0].Ns, "expected namespace: %s, got: %s", "db."+coll, bwCmd[1].NsInfo[0].Ns)
		})
	})

	mt.Run("12. MongoClient.bulkWrite returns an error if no operations can be added to ops", func(mt *mtest.T) {
		mt.ResetClient(options.Client())
		var hello struct {
			MaxMessageSizeBytes int
		}
		err := mt.DB.RunCommand(context.Background(), bson.D{{"hello", 1}}).Decode(&hello)
		require.NoError(mt, err, "Hello error: %v", err)
		mt.Run("Case 1: document too large", func(mt *mtest.T) {
			writes := []mongo.ClientBulkWrite{{
				Database:   "db",
				Collection: "coll",
				Model: &mongo.ClientInsertOneModel{
					Document: bson.D{{"a", strings.Repeat("b", hello.MaxMessageSizeBytes)}},
				},
			}}
			_, err := mt.Client.BulkWrite(context.Background(), writes)
			require.EqualError(mt, err, driver.ErrDocumentTooLarge.Error())
			var cbwe mongo.ClientBulkWriteException
			if errors.As(err, &cbwe) {
				assert.Nil(mt, cbwe.PartialResult, "expected nil PartialResult in ClientBulkWriteException")
			}
		})
		mt.Run("Case 2: namespace too large", func(mt *mtest.T) {
			writes := []mongo.ClientBulkWrite{{
				Database:   "db",
				Collection: strings.Repeat("c", hello.MaxMessageSizeBytes),
				Model: &mongo.ClientInsertOneModel{
					Document: bson.D{{"a", "b"}},
				},
			}}
			_, err := mt.Client.BulkWrite(context.Background(), writes)
			require.EqualError(mt, err, driver.ErrDocumentTooLarge.Error())
			var cbwe mongo.ClientBulkWriteException
			if errors.As(err, &cbwe) {
				assert.Nil(mt, cbwe.PartialResult, "expected nil PartialResult in ClientBulkWriteException")
			}
		})
	})

	mt.Run("13. MongoClient.bulkWrite returns an error if auto-encryption is configured", func(mt *mtest.T) {
		if !mtest.IsCSFLEEnabled() {
			mt.Skip("CSFLE is not enabled")
		}
		if os.Getenv("DOCKER_RUNNING") != "" {
			mt.Skip("skipping test in docker environment")
		}

		autoEncryptionOpts := options.AutoEncryption().
			SetKeyVaultNamespace("db.coll").
			SetKmsProviders(map[string]map[string]any{
				"aws": {
					"accessKeyId":     "foo",
					"secretAccessKey": "bar",
				},
			})
		mt.ResetClient(options.Client().SetAutoEncryptionOptions(autoEncryptionOpts))
		writes := []mongo.ClientBulkWrite{{
			Database:   "db",
			Collection: "coll",
			Model: &mongo.ClientInsertOneModel{
				Document: bson.D{{"a", "b"}},
			},
		}}
		_, err := mt.Client.BulkWrite(context.Background(), writes)
		require.ErrorContains(mt, err, "bulkWrite does not currently support automatic encryption")
		var cbwe mongo.ClientBulkWriteException
		if errors.As(err, &cbwe) {
			assert.Nil(mt, cbwe.PartialResult, "expected nil PartialResult in ClientBulkWriteException")
		}
	})

	mt.Run("15. MongoClient.bulkWrite with unacknowledged write concern uses w:0 for all batches", func(mt *mtest.T) {
		type cmd struct {
			Ops          []bson.D
			WriteConcern struct {
				W any
			}
		}
		var bwCmd []cmd
		monitor := &event.CommandMonitor{
			Started: func(_ context.Context, e *event.CommandStartedEvent) {
				if e.CommandName == "bulkWrite" {
					var c cmd
					err := bson.Unmarshal(e.Command, &c)
					require.NoError(mt, err, "Unmarshal error: %v", err)

					bwCmd = append(bwCmd, c)
				}
			},
		}
		mt.ResetClient(options.Client().SetMonitor(monitor))
		var hello struct {
			MaxBsonObjectSize   int
			MaxMessageSizeBytes int
		}
		err := mt.DB.RunCommand(context.Background(), bson.D{{"hello", 1}}).Decode(&hello)
		require.NoError(mt, err, "Hello error: %v", err)

		coll := mt.CreateCollection(mtest.Collection{DB: "db", Name: "coll"}, false)
		err = coll.Drop(context.Background())
		require.NoError(mt, err, "Drop error: %v", err)

		num := hello.MaxMessageSizeBytes/hello.MaxBsonObjectSize + 1
		var writes []mongo.ClientBulkWrite
		for i := 0; i < num; i++ {
			writes = append(writes, mongo.ClientBulkWrite{
				Database:   "db",
				Collection: "coll",
				Model: &mongo.ClientInsertOneModel{
					Document: bson.D{{"a", strings.Repeat("b", hello.MaxBsonObjectSize-500)}},
				},
			})
		}
		result, err := mt.Client.BulkWrite(context.Background(), writes, options.ClientBulkWrite().SetOrdered(false).SetWriteConcern(writeconcern.Unacknowledged()))
		require.NoError(mt, err, "BulkWrite error: %v", err)
		assert.False(mt, result.Acknowledged)
		require.Len(mt, bwCmd, 2, "expected %d bulkWrite calls, got: %d", 2, len(bwCmd))

		assert.Len(mt, bwCmd[0].Ops, num-1, "expected %d ops, got: %d", num-1, len(bwCmd[0].Ops))
		assert.Equal(mt, int32(0), bwCmd[0].WriteConcern.W, "expected writeConcern: %d, got: %v", 0, bwCmd[0].WriteConcern.W)

		assert.Len(mt, bwCmd[1].Ops, 1, "expected %d ops, got: %d", 1, len(bwCmd[1].Ops))
		assert.Equal(mt, int32(0), bwCmd[1].WriteConcern.W, "expected writeConcern: %d, got: %v", 0, bwCmd[1].WriteConcern.W)

		n, err := coll.CountDocuments(context.Background(), bson.D{})
		require.NoError(mt, err, "CountDocuments error: %v", err)
		assert.Equal(mt, num, int(n), "expected %d documents, got: %d", num, n)
	})
}
