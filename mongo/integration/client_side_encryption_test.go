// Copyright (C) MongoDB, Inc. 2021-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// +build cse

package integration

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// createDataKeyAndEncrypt creates a data key with the alternate name @keyName.
// Returns a ciphertext encrypted with the data key for use as test data.
func createDataKeyAndEncrypt(mt *mtest.T, keyName string) primitive.Binary {
	mt.Helper()

	// Use majority read and write concern since operations will depend on keys being created.
	kvClientOpts := options.Client().
		ApplyURI(mtest.ClusterURI()).
		SetReadConcern(mtest.MajorityRc).
		SetWriteConcern(mtest.MajorityWc)

	testutil.AddTestServerAPIVersion(kvClientOpts)

	kmsProvidersMap := map[string]map[string]interface{}{
		"local": {"key": localMasterKey},
	}

	kvClient, err := mongo.Connect(mtest.Background, kvClientOpts)
	defer kvClient.Disconnect(mtest.Background)
	assert.Nil(mt, err, "Connect error: %v", err)

	err = kvClient.Database("keyvault").Collection("datakeys").Drop(mtest.Background)
	assert.Nil(mt, err, "Drop error: %v", err)

	ceOpts := options.ClientEncryption().
		SetKmsProviders(kmsProvidersMap).
		SetKeyVaultNamespace("keyvault.datakeys")

	ce, err := mongo.NewClientEncryption(kvClient, ceOpts)
	assert.Nil(mt, err, "NewClientEncryption error: %v", err)

	dkOpts := options.DataKey().SetKeyAltNames([]string{keyName})
	_, err = ce.CreateDataKey(mtest.Background, "local", dkOpts)
	assert.Nil(mt, err, "CreateDataKey error: %v", err)

	t, value, err := bson.MarshalValue("test")
	assert.Nil(mt, err, "MarshalValue error: %v", err)
	in := bson.RawValue{Type: t, Value: value}
	encryptOpts := options.Encrypt().
		SetAlgorithm("AEAD_AES_256_CBC_HMAC_SHA_512-Random").
		SetKeyAltName(keyName)

	ciphertext, err := ce.Encrypt(mtest.Background, in, encryptOpts)
	assert.Nil(mt, err, "Encrypt error: %v", err)
	return ciphertext
}

func TestClientSideEncryptionWithExplicitSessions(t *testing.T) {
	verifyClientSideEncryptionVarsSet(t)
	mt := mtest.New(t, mtest.NewOptions().MinServerVersion("4.2").Enterprise(true).CreateClient(false))
	defer mt.Close()

	kmsProvidersMap := map[string]map[string]interface{}{
		"local": {"key": localMasterKey},
	}

	schema := bson.D{
		{"bsonType", "object"},
		{"properties", bson.D{
			{"encryptMe", bson.D{
				{"encrypt", bson.D{
					{"keyId", "/keyName"},
					{"bsonType", "string"},
					{"algorithm", "AEAD_AES_256_CBC_HMAC_SHA_512-Random"},
				}},
			}},
		}},
	}
	schemaMap := map[string]interface{}{"db.coll": schema}

	mt.Run("automatic encryption", func(t *mtest.T) {
		createDataKeyAndEncrypt(mt, "myKey")

		aeOpts := options.AutoEncryption().
			SetKmsProviders(kmsProvidersMap).
			SetKeyVaultNamespace("keyvault.datakeys").
			SetSchemaMap(schemaMap)

		clientOpts := options.Client().
			ApplyURI(mtest.ClusterURI()).
			SetReadConcern(mtest.MajorityRc).
			SetWriteConcern(mtest.MajorityWc).
			SetAutoEncryptionOptions(aeOpts)

		testutil.AddTestServerAPIVersion(clientOpts)

		client, err := mongo.Connect(mtest.Background, clientOpts)
		assert.Nil(mt, err, "Connect error: %v", err)

		session, err := client.StartSession()
		assert.Nil(mt, err, "StartSession error: %v", err)

		sessionCtx := mongo.NewSessionContext(mtest.Background, session)

		coll := client.Database("db").Collection("coll")
		coll.Drop(mtest.Background)
		_, err = coll.InsertOne(sessionCtx, bson.D{{"encryptMe", "test"}, {"keyName", "myKey"}})
		assert.Nil(mt, err, "InsertOne error: %v", err)
	})

	mt.Run("automatic decryption", func(t *mtest.T) {
		ciphertext := createDataKeyAndEncrypt(mt, "myKey")

		aeOpts := options.AutoEncryption().
			SetKmsProviders(kmsProvidersMap).
			SetKeyVaultNamespace("keyvault.datakeys").
			SetBypassAutoEncryption(true)

		clientOpts := options.Client().
			ApplyURI(mtest.ClusterURI()).
			SetReadConcern(mtest.MajorityRc).
			SetWriteConcern(mtest.MajorityWc).
			SetAutoEncryptionOptions(aeOpts)
		testutil.AddTestServerAPIVersion(clientOpts)

		client, err := mongo.Connect(mtest.Background, clientOpts)
		assert.Nil(mt, err, "Connect error: %v", err)

		coll := client.Database("db").Collection("coll")
		coll.Drop(mtest.Background)
		_, err = coll.InsertOne(mtest.Background, bson.D{{"encryptMe", ciphertext}})
		assert.Nil(mt, err, "InsertOne error: %v", err)

		session, err := client.StartSession()
		assert.Nil(mt, err, "StartSession error: %v", err)
		sessionCtx := mongo.NewSessionContext(mtest.Background, session)
		res := coll.FindOne(sessionCtx, bson.D{{}})
		assert.Nil(mt, res.Err(), "FindOne error: %v", res.Err())
	})
}
