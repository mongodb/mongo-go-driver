// Copyright (C) MongoDB, Inc. 2021-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build cse
// +build cse

package integration

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestCreateEncryptedCollection tests creating a collection with automatic encryption.
func TestCreateEncryptedCollection(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().MinServerVersion("6.0").Enterprise(true).CreateClient(false).Topologies(mtest.ReplicaSet, mtest.Sharded))
	defer mt.Close()

	kmsProvidersMap := map[string]map[string]interface{}{
		"local": {"key": localMasterKey},
	}

	mt.Run("createEncryptedCollection", func(mt *mtest.T) {
		var encryptedFields bson.D
		err := bson.UnmarshalExtJSON([]byte(`{
			"escCollection": "foo_esc",
			"eccCollection": "foo_ecc",
			"ecocCollection": "foo_ecoc",
			"fields": [
				{
					"path": "firstName",
					"keyId": { "$binary": { "subType": "04", "base64": "AAAAAAAAAAAAAAAAAAAAAA==" }},
					"bsonType": "string",
					"queries": {"queryType": "equality"}
				}
			]
		}`), true /* canonical */, &encryptedFields)
		assert.Nil(mt, err, "error in UnmarshalExtJSON: %v", err)

		encryptedFieldsMap := map[string]interface{}{
			"db.coll": encryptedFields,
		}
		aeo := options.AutoEncryption().SetKmsProviders(kmsProvidersMap).SetEncryptedFieldsMap(encryptedFieldsMap)

		// Create a data client.
		co := options.Client().ApplyURI(mtest.ClusterURI()).SetAutoEncryptionOptions(aeo)
		testutil.AddTestServerAPIVersion(co)
		client, err := mongo.Connect(context.TODO(), co)
		assert.Nil(t, err, "error in data client Connect: %v", err)
		defer client.Disconnect(context.TODO())

		db := client.Database("db")
		err = db.Drop(context.TODO())
		assert.Nil(mt, err, "error in database Drop: %v", err)
		err = db.CreateCollection(context.TODO(), "coll")
		assert.Nil(mt, err, "error in CreateCollection: %v", err)

		// Check that expected data collection was created with encryptedFields. */
		{
			got, err := db.ListCollectionSpecifications(context.TODO(), bson.M{"name": "coll"})
			assert.Nil(mt, err, "error in ListCollectionSpecifications: %v", err)
			assert.Equal(mt, len(got), 1, "expected to get one CollectionSpecification, got: %v", got)
			opts := got[0].Options
			_, err = opts.LookupErr("encryptedFields")
			assert.Nil(mt, err, "expected to find encryptedFields in coll options, got: %v", opts)
		}

		// Check that expected state collections were created.
		{
			got, err := db.ListCollectionNames(context.TODO(), bson.M{"name": "foo_esc"})
			assert.Nil(mt, err, "error in ListCollectionNames")
			assert.Equal(mt, len(got), 1, "expected to get match for foo_esc, got: %v", got)
			got, err = db.ListCollectionNames(context.TODO(), bson.M{"name": "foo_ecc"})
			assert.Nil(mt, err, "error in ListCollectionNames")
			assert.Equal(mt, len(got), 1, "expected to get match for foo_ecc, got: %v", got)
			got, err = db.ListCollectionNames(context.TODO(), bson.M{"name": "foo_ecoc"})
			assert.Nil(mt, err, "error in ListCollectionNames")
			assert.Equal(mt, len(got), 1, "expected to get match for foo_ecoc, got: %v", got)
		}

		// Check that index was created on __safeContents__.
		{
			got, err := db.Collection("coll").Indexes().ListSpecifications(context.TODO())
			assert.Nil(mt, err, "error in ListSpecifications: %v", err)
			found := false
			names_found := make([]string, 0)
			for _, indexSpec := range got {
				names_found = append(names_found, indexSpec.Name)
				rawValue, err := indexSpec.KeysDocument.LookupErr("__safeContent__")
				/* Skip other index values. */
				if err != nil {
					continue
				}
				found = true
				indexVal, ok := rawValue.AsInt32OK()
				assert.True(mt, ok, "index value expected to be Int32, got: %v", rawValue.Type)
				assert.Equal(mt, indexVal, int32(1), "expected index value to be 1, got: %v", indexVal)
			}
			assert.True(mt, found, "unable to find index on __safeContent__, got: %v", names_found)
		}
	})
}

func TestDropEncryptedCollection(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().MinServerVersion("6.0").Enterprise(true).CreateClient(false).Topologies(mtest.ReplicaSet, mtest.Sharded))
	defer mt.Close()

	kmsProvidersMap := map[string]map[string]interface{}{
		"local": {"key": localMasterKey},
	}

	mt.Run("createEncryptedCollection", func(mt *mtest.T) {
		var encryptedFields bson.D
		err := bson.UnmarshalExtJSON([]byte(`{
			"escCollection": "foo_esc",
			"eccCollection": "foo_ecc",
			"ecocCollection": "foo_ecoc",
			"fields": [
				{
					"path": "firstName",
					"keyId": { "$binary": { "subType": "04", "base64": "AAAAAAAAAAAAAAAAAAAAAA==" }},
					"bsonType": "string",
					"queries": {"queryType": "equality"}
				}
			]
		}`), true /* canonical */, &encryptedFields)
		assert.Nil(mt, err, "error in UnmarshalExtJSON: %v", err)

		encryptedFieldsMap := map[string]interface{}{
			"db.coll": encryptedFields,
		}
		aeo := options.AutoEncryption().SetKmsProviders(kmsProvidersMap).SetEncryptedFieldsMap(encryptedFieldsMap)

		// Create a data client.
		co := options.Client().ApplyURI(mtest.ClusterURI()).SetAutoEncryptionOptions(aeo)
		testutil.AddTestServerAPIVersion(co)
		client, err := mongo.Connect(context.TODO(), co)
		assert.Nil(t, err, "error in data client Connect: %v", err)
		defer client.Disconnect(context.TODO())

		db := client.Database("db")
		err = db.Drop(context.TODO())
		assert.Nil(mt, err, "error in database Drop: %v", err)
		err = db.CreateCollection(context.TODO(), "coll")
		assert.Nil(mt, err, "error in CreateCollection: %v", err)

		// Check that expected data collection was created with encryptedFields. */
		{
			got, err := db.ListCollectionSpecifications(context.TODO(), bson.M{"name": "coll"})
			assert.Nil(mt, err, "error in ListCollectionSpecifications: %v", err)
			assert.Equal(mt, len(got), 1, "expected to get one CollectionSpecification, got: %v", got)
			opts := got[0].Options
			_, err = opts.LookupErr("encryptedFields")
			assert.Nil(mt, err, "expected to find encryptedFields in coll options, got: %v", opts)
		}

		// Check that expected state collections were created.
		{
			got, err := db.ListCollectionNames(context.TODO(), bson.M{"name": "foo_esc"})
			assert.Nil(mt, err, "error in ListCollectionNames")
			assert.Equal(mt, len(got), 1, "expected to get match for foo_esc, got: %v", got)
			got, err = db.ListCollectionNames(context.TODO(), bson.M{"name": "foo_ecc"})
			assert.Nil(mt, err, "error in ListCollectionNames")
			assert.Equal(mt, len(got), 1, "expected to get match for foo_ecc, got: %v", got)
			got, err = db.ListCollectionNames(context.TODO(), bson.M{"name": "foo_ecoc"})
			assert.Nil(mt, err, "error in ListCollectionNames")
			assert.Equal(mt, len(got), 1, "expected to get match for foo_ecoc, got: %v", got)
		}

		// Check that index was created on __safeContents__.
		{
			got, err := db.Collection("coll").Indexes().ListSpecifications(context.TODO())
			assert.Nil(mt, err, "error in ListSpecifications: %v", err)
			found := false
			names_found := make([]string, 0)
			for _, indexSpec := range got {
				names_found = append(names_found, indexSpec.Name)
				rawValue, err := indexSpec.KeysDocument.LookupErr("__safeContent__")
				/* Skip other index values. */
				if err != nil {
					continue
				}
				found = true
				indexVal, ok := rawValue.AsInt32OK()
				assert.True(mt, ok, "index value expected to be Int32, got: %v", rawValue.Type)
				assert.Equal(mt, indexVal, int32(1), "expected index value to be 1, got: %v", indexVal)
			}
			assert.True(mt, found, "unable to find index on __safeContent__, got: %v", names_found)
		}

		err = db.Collection("coll").Drop(context.TODO())
		assert.Nil(mt, err, "error in Drop: %v", err)

		// Check that the data collection does not exist.
		{
			got, err := db.ListCollectionNames(context.TODO(), bson.M{"name": "coll"})
			assert.Nil(mt, err, "error in ListCollectionNames")
			assert.Equal(mt, len(got), 0, "expected to get no match for coll, got: %v", got)
		}

		// Check that the state collections do not exist.
		{
			got, err := db.ListCollectionNames(context.TODO(), bson.M{"name": "foo_esc"})
			assert.Nil(mt, err, "error in ListCollectionNames")
			assert.Equal(mt, len(got), 0, "expected to get no match for foo_esc, got: %v", got)
			got, err = db.ListCollectionNames(context.TODO(), bson.M{"name": "foo_ecc"})
			assert.Nil(mt, err, "error in ListCollectionNames")
			assert.Equal(mt, len(got), 0, "expected to get no match for foo_ecc, got: %v", got)
			got, err = db.ListCollectionNames(context.TODO(), bson.M{"name": "foo_ecoc"})
			assert.Nil(mt, err, "error in ListCollectionNames")
			assert.Equal(mt, len(got), 0, "expected to get no match for foo_ecoc, got: %v", got)
		}
	})
}
