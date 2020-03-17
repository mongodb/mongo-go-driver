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
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
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

func TestHintWithUnacknowledgedWriteErrors(t *testing.T) {
	mtOpts := mtest.NewOptions().MinServerVersion("3.6").MaxServerVersion("4.0").CreateClient(false).
		CollectionOptions(options.Collection().SetWriteConcern(writeconcern.New(writeconcern.W(0))))
	mt := mtest.New(t, mtOpts)
	defer mt.Close()

	expected := errors.New("the 'hint' command parameter with unacknowledged WriteConcern requires a minimum server wire version of 8")
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
			&mongo.ReplaceOneModel{Filter: bson.D{{"_id", 2}}, Replacement: bson.D{{"a", 2}}, Hint: "_id_"},
		}
		_, got := mt.Coll.BulkWrite(mtest.Background, models)
		assert.NotNil(mt, got, "expected non-nil error, got nil")
		assert.Equal(mt, got, expected, "expected: %v got: %v", expected, got)
	})
}

func TestAggregateSecondaryPreferredReadPreference(t *testing.T) {
	// Use secondaryPreferred instead of secondary because sharded clusters started up by mongo-orchestration have
	// one-node shards, so a secondary read preference is not satisfiable.
	secondaryPrefClientOpts := options.Client().
		SetWriteConcern(mtest.MajorityWc).
		SetReadPreference(readpref.SecondaryPreferred()).
		SetReadConcern(mtest.MajorityRc)
	mtOpts := mtest.NewOptions().
		ClientOptions(secondaryPrefClientOpts).
		MinServerVersion("4.1.0") // Consistent with tests in aggregate-out-readConcern.json

	mt := mtest.New(t, mtOpts)
	mt.Run("aggregate $out with read preference secondary", func(mt *mtest.T) {
		doc, err := bson.Marshal(bson.D{
			{"_id", 1},
			{"x", 11},
		})
		assert.Nil(mt, err, "Marshal error: %v", err)
		_, err = mt.Coll.InsertOne(mtest.Background, doc)
		assert.Nil(mt, err, "InsertOne error: %v", err)

		mt.ClearEvents()
		outputCollName := "aggregate-read-pref-secondary-output"
		outStage := bson.D{
			{"$out", outputCollName},
		}
		cursor, err := mt.Coll.Aggregate(mtest.Background, mongo.Pipeline{outStage})
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
