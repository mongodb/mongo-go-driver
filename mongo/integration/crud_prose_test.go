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
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func TestWriteErrorsWithLabels(t *testing.T) {
	clientOpts := options.Client().SetRetryWrites(false).SetWriteConcern(mtest.MajorityWc).
		SetReadConcern(mtest.MajorityRc)
	mtOpts := mtest.NewOptions().ClientOptions(clientOpts).MinServerVersion("4.0").Topologies(mtest.ReplicaSet).
		CreateClient(false)
	mt := mtest.New(t, mtOpts)
	defer mt.Close()

	label := "ExampleError"
	mt.Run("InsertMany errors with label", func(mt *mtest.T) {
		mt.SetFailPoint(mtest.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: mtest.FailPointMode{
				Times: 1,
			},
			Data: mtest.FailPointData{
				FailCommands: []string{"insert"},
				WriteConcernError: &mtest.WriteConcernErrorData{
					Code:        100,
					ErrorLabels: &[]string{label},
				},
			},
		})

		_, err := mt.Coll.InsertMany(context.Background(),
			[]interface{}{
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
		mt.SetFailPoint(mtest.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: mtest.FailPointMode{
				Times: 1,
			},
			Data: mtest.FailPointData{
				FailCommands: []string{"delete"},
				WriteConcernError: &mtest.WriteConcernErrorData{
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
		mt.SetFailPoint(mtest.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: mtest.FailPointMode{
				Times: 1,
			},
			Data: mtest.FailPointData{
				FailCommands: []string{"delete"},
				WriteConcernError: &mtest.WriteConcernErrorData{
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
	defer mt.Close()

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
					_, err := coll.InsertMany(context.Background(), []interface{}{bson.D{{"nope", 1}}})
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
						[]interface{}{
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
	defer mt.Close()

	expected := errors.New("the 'hint' command parameter requires a minimum server wire version of 5")
	mt.Run("UpdateMany", func(mt *mtest.T) {

		_, got := mt.Coll.UpdateMany(context.Background(), bson.D{{"a", 1}}, bson.D{{"$inc", bson.D{{"a", 1}}}},
			options.Update().SetHint("_id_"))
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
	defer mt.Close()

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
		fp := mtest.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: mtest.FailPointMode{
				Times: 1,
			},
			Data: mtest.FailPointData{
				FailCommands: []string{"insert"},
				WriteConcernError: &mtest.WriteConcernErrorData{
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
	defer mt.Close()

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
