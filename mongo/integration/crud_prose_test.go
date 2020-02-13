// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestCrudProse(t *testing.T) {
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
			&mongo.DeleteOneModel{bson.D{{"a", 2}}, nil},
		}
		_, err := mt.Coll.BulkWrite(mtest.Background, models)
		assert.NotNil(mt, err, "expected non-nil error, got nil")

		we, ok := err.(mongo.BulkWriteException)
		assert.True(mt, ok, "expected mongo.BulkWriteException, got %T", err)
		assert.True(mt, we.HasErrorLabel(label), "expected error to have label: %v", label)
	})
}
