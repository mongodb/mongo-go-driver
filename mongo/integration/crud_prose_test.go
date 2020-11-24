// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"bytes"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
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

		_, err := mt.Coll.InsertMany(mtest.Background,
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

		_, err := mt.Coll.DeleteMany(mtest.Background, bson.D{{"a", 1}})
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
		_, err := mt.Coll.BulkWrite(mtest.Background, models)
		assert.NotNil(mt, err, "expected non-nil error, got nil")

		we, ok := err.(mongo.BulkWriteException)
		assert.True(mt, ok, "expected mongo.BulkWriteException, got %T", err)
		assert.True(mt, we.HasErrorLabel(label), "expected error to have label: %v", label)
	})

}

func TestHintErrors(t *testing.T) {
	mtOpts := mtest.NewOptions().MaxServerVersion("3.2").CreateClient(false)
	mt := mtest.New(t, mtOpts)
	defer mt.Close()

	expected := errors.New("the 'hint' command parameter requires a minimum server wire version of 5")
	mt.Run("UpdateMany", func(mt *mtest.T) {

		_, got := mt.Coll.UpdateMany(mtest.Background, bson.D{{"a", 1}}, bson.D{{"$inc", bson.D{{"a", 1}}}},
			options.Update().SetHint("_id_"))
		assert.NotNil(mt, got, "expected non-nil error, got nil")
		assert.Equal(mt, got, expected, "expected: %v got: %v", expected, got)
	})

	mt.Run("ReplaceOne", func(mt *mtest.T) {

		_, got := mt.Coll.ReplaceOne(mtest.Background, bson.D{{"a", 1}}, bson.D{{"a", 2}},
			options.Replace().SetHint("_id_"))
		assert.NotNil(mt, got, "expected non-nil error, got nil")
		assert.Equal(mt, got, expected, "expected: %v got: %v", expected, got)
	})

	mt.Run("BulkWrite", func(mt *mtest.T) {
		models := []mongo.WriteModel{
			&mongo.InsertOneModel{bson.D{{"_id", 2}}},
			&mongo.ReplaceOneModel{Filter: bson.D{{"_id", 2}}, Replacement: bson.D{{"a", 2}}, Hint: "_id_"},
		}
		_, got := mt.Coll.BulkWrite(mtest.Background, models)
		assert.NotNil(mt, got, "expected non-nil error, got nil")
		assert.Equal(mt, got, expected, "expected: %v got: %v", expected, got)
	})
}

type testValueMarshaler struct {
	val []bson.D
}

func (tvm testValueMarshaler) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bson.MarshalValue(tvm.val)
}

func TestAggregatePrimaryPreferredReadPreference(t *testing.T) {
	primaryPrefClientOpts := options.Client().
		SetWriteConcern(mtest.MajorityWc).
		SetReadPreference(readpref.PrimaryPreferred()).
		SetReadConcern(mtest.MajorityRc)
	mtOpts := mtest.NewOptions().
		ClientOptions(primaryPrefClientOpts).
		MinServerVersion("4.1.0") // Consistent with tests in aggregate-out-readConcern.json

	mt := mtest.New(t, mtOpts)
	mt.Run("aggregate $out with non-primary read preference", func(mt *mtest.T) {
		doc, err := bson.Marshal(bson.D{
			{"_id", 1},
			{"x", 11},
		})
		assert.Nil(mt, err, "Marshal error: %v", err)
		outputCollName := "aggregate-read-pref-primary-preferred-output"
		testCases := []struct {
			name     string
			pipeline interface{}
		}{
			{
				"pipeline",
				mongo.Pipeline{bson.D{{"$out", outputCollName}}},
			},
			{
				"doc slice",
				[]bson.D{{{"$out", outputCollName}}},
			},
			{
				"bson a",
				bson.A{bson.D{{"$out", outputCollName}}},
			},
			{
				"valueMarshaler",
				testValueMarshaler{[]bson.D{{{"$out", outputCollName}}}},
			},
		}
		for _, tc := range testCases {
			mt.Run(tc.name, func(mt *mtest.T) {
				_, err = mt.Coll.InsertOne(mtest.Background, doc)
				assert.Nil(mt, err, "InsertOne error: %v", err)

				mt.ClearEvents()

				cursor, err := mt.Coll.Aggregate(mtest.Background, tc.pipeline)
				assert.Nil(mt, err, "Aggregate error: %v", err)
				_ = cursor.Close(mtest.Background)

				// Assert that the output collection contains the document we expect.
				outputColl := mt.CreateCollection(mtest.Collection{Name: outputCollName}, false)
				cursor, err = outputColl.Find(mtest.Background, bson.D{})
				assert.Nil(mt, err, "Find error: %v", err)
				defer cursor.Close(mtest.Background)

				assert.True(mt, cursor.Next(mtest.Background), "expected Next to return true, got false")
				assert.True(mt, bytes.Equal(doc, cursor.Current), "expected document %s, got %s", bson.Raw(doc), cursor.Current)
				assert.False(mt, cursor.Next(mtest.Background), "unexpected document returned by Find: %s", cursor.Current)

				// Assert that no read preference was sent to the server.
				evt := mt.GetStartedEvent()
				assert.Equal(mt, "aggregate", evt.CommandName, "expected command 'aggregate', got '%s'", evt.CommandName)
				_, err = evt.Command.LookupErr("$readPreference")
				assert.NotNil(mt, err, "expected command %s to not contain $readPreference", evt.Command)
			})
		}
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

		_, err := mt.Coll.InsertOne(mtest.Background, bson.D{{"x", 1}})
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
		err := mt.DB.RunCommand(mtest.Background, cmd).Err()
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

		_, err := mt.Coll.InsertOne(mtest.Background, bson.D{})
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
