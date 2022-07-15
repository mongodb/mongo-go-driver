// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build cse
// +build cse

package integration

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal"
	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

var (
	localMasterKey = []byte("2x44+xduTaBBkY16Er5DuADaghvS4vwdkg8tpPp3tz6gV01A1CwbD9itQ2HFDgPWOp8eMaC1Oi766JzXZBdBdbdMurdonJ1d")
)

const (
	clientEncryptionProseDir      = "../../data/client-side-encryption-prose"
	deterministicAlgorithm        = "AEAD_AES_256_CBC_HMAC_SHA_512-Deterministic"
	randomAlgorithm               = "AEAD_AES_256_CBC_HMAC_SHA_512-Random"
	kvNamespace                   = "keyvault.datakeys" // default namespace for the key vault collection
	keySubtype               byte = 4                   // expected subtype for data keys
	encryptedValueSubtype    byte = 6                   // expected subtypes for encrypted values
	cryptMaxBatchSizeBytes        = 2097152             // max bytes in write batch when auto encryption is enabled
	maxBsonObjSize                = 16777216            // max bytes in BSON object
)

func TestClientSideEncryptionProse(t *testing.T) {
	verifyClientSideEncryptionVarsSet(t)
	mt := mtest.New(t, mtest.NewOptions().MinServerVersion("4.2").Enterprise(true).CreateClient(false))
	defer mt.Close()

	defaultKvClientOptions := options.Client().ApplyURI(mtest.ClusterURI())
	testutil.AddTestServerAPIVersion(defaultKvClientOptions)
	fullKmsProvidersMap := map[string]map[string]interface{}{
		"aws": {
			"accessKeyId":     awsAccessKeyID,
			"secretAccessKey": awsSecretAccessKey,
		},
		"azure": {
			"tenantId":     azureTenantID,
			"clientId":     azureClientID,
			"clientSecret": azureClientSecret,
		},
		"gcp": {
			"email":      gcpEmail,
			"privateKey": gcpPrivateKey,
		},
		"local": {"key": localMasterKey},
		"kmip": {
			"endpoint": "localhost:5698",
		},
	}

	runOpts := mtest.NewOptions().MinServerVersion("6.0").Topologies(mtest.ReplicaSet, mtest.LoadBalanced, mtest.ShardedReplicaSet)
	mt.Run("1. custom key material test", func(mt *mtest.T) {
		const (
			dkCollection = "datakeys"
			idKey        = "_id"
			kvDatabase   = "keyvault"
		)

		// Create a ClientEncryption object (referred to as client_encryption) with client set as the keyVaultClient.
		// Using client, drop the collection keyvault.datakeys.
		cse := setup(mt, nil, defaultKvClientOptions, options.ClientEncryption().
			SetKmsProviders(fullKmsProvidersMap).
			SetKeyVaultNamespace(kvNamespace))

		err := cse.kvClient.Database(kvDatabase).Collection(dkCollection).Drop(context.Background())
		assert.Nil(mt, err, "error dropping %q namespace: %v", kvNamespace, err)

		// Using client_encryption, create a data key with a local KMS provider and the declared b64 custom key material
		// (given as base64).
		const b641 = `xPTAjBRG5JiPm+d3fj6XLi2q5DMXUS/f1f+SMAlhhwkhDRL0kr8r9GDLIGTAGlvC+HVjSIgdL+RKwZCvpXSyxTICWSXT` +
			`UYsWYPyu3IoHbuBZdmw2faM3WhcRIgbMReU5`

		// Decode the base64-encoded keyMaterial string.
		km, err := base64.StdEncoding.DecodeString(b641)
		assert.Nil(mt, err, "error decoding b64: %v", err)

		_, err = cse.clientEnc.CreateDataKey(context.Background(), "local", options.DataKey().SetKeyMaterial(km))
		assert.Nil(mt, err, "error creating data key: %v", err)

		// Find the resulting key document in keyvault.datakeys, save a copy of the key document, then remove the key
		// document from the collection.
		coll := cse.kvClient.Database(kvDatabase).Collection(dkCollection)

		keydoc, err := coll.FindOne(context.Background(), bson.D{}).DecodeBytes()
		assert.Nil(mt, err, "error in decoding bytes: %v", err)

		// Remove the key document from the collection.
		id, err := keydoc.LookupErr(idKey)
		assert.Nil(mt, err, "error looking up %s: %v", idKey, err)

		_, err = coll.DeleteOne(context.Background(), bson.D{{idKey, id}})
		assert.Nil(mt, err, "error deleting key document: %v", err)

		// Replace the _id field in the copied key document with a UUID with base64 value AAAAAAAAAAAAAAAAAAAAAA== (16
		// bytes all equal to 0x00) and insert the modified key document into keyvault.datakeys with majority write
		// concern.
		cidx, alteredKeydoc := bsoncore.AppendDocumentStart(nil)
		docElems, _ := keydoc.Elements()
		for _, element := range docElems {
			if key := element.Key(); key != idKey {
				alteredKeydoc = bsoncore.AppendValueElement(alteredKeydoc, key, rawValueToCoreValue(element.Value()))
			}
		}
		empty := [16]byte{}
		uuidSubtype, _ := keydoc.Lookup(idKey).Binary()
		alteredKeydoc = bsoncore.AppendBinaryElement(alteredKeydoc, idKey, uuidSubtype, empty[:])
		alteredKeydoc, _ = bsoncore.AppendDocumentEnd(alteredKeydoc, cidx)

		// Insert the copied key document into keyvault.datakeys with majority write concern.
		wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(1*time.Second))
		wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)
		wcmColl := cse.kvClient.Database(kvDatabase).Collection(dkCollection, wcMajorityCollectionOpts)
		_, err = wcmColl.InsertOne(context.Background(), alteredKeydoc)
		assert.Nil(mt, err, "error inserting altered key document: %v", err)

		// Using client_encryption, encrypt the string "test" with the modified data key using the
		// AEAD_AES_256_CBC_HMAC_SHA_512-Deterministic algorithm and assert the resulting value is equal to the
		// declared b64 constant.
		const b642 = `AQAAAAAAAAAAAAAAAAAAAAACz0ZOLuuhEYi807ZXTdhbqhLaS2/t9wLifJnnNYwiw79d75QYIZ6M/aYC1h9nCzCjZ7pG` +
			`UpAuNnkUhnIXM3PjrA==`

		empty = [16]byte{}
		keyid := primitive.Binary{Subtype: 0x04, Data: empty[:]}

		encOpts := options.Encrypt().SetAlgorithm(deterministicAlgorithm).SetKeyID(keyid)
		testVal := bson.RawValue{
			Type:  bson.TypeString,
			Value: bsoncore.AppendString(nil, "test"),
		}
		actual, err := cse.clientEnc.Encrypt(context.Background(), testVal, encOpts)
		assert.Nil(mt, err, "error encrypting data: %v", err)

		expected := primitive.Binary{Subtype: 0x06}
		expected.Data, _ = base64.StdEncoding.DecodeString(b642)

		assert.Equal(t, actual, expected, "expected: %v, got: %v", actual, expected)
	})
	mt.RunOpts("2. data key and double encryption", noClientOpts, func(mt *mtest.T) {
		// set up options structs
		schema := bson.D{
			{"bsonType", "object"},
			{"properties", bson.D{
				{"encrypted_placeholder", bson.D{
					{"encrypt", bson.D{
						{"keyId", "/placeholder"},
						{"bsonType", "string"},
						{"algorithm", "AEAD_AES_256_CBC_HMAC_SHA_512-Random"},
					}},
				}},
			}},
		}
		schemaMap := map[string]interface{}{"db.coll": schema}
		tlsConfig := make(map[string]*tls.Config)
		if tlsCAFileKMIP != "" && tlsClientCertificateKeyFileKMIP != "" {
			tlsOpts := map[string]interface{}{
				"tlsCertificateKeyFile": tlsClientCertificateKeyFileKMIP,
				"tlsCAFile":             tlsCAFileKMIP,
			}
			kmipConfig, err := options.BuildTLSConfig(tlsOpts)
			assert.Nil(mt, err, "BuildTLSConfig error: %v", err)
			tlsConfig["kmip"] = kmipConfig
		}

		aeo := options.AutoEncryption().
			SetKmsProviders(fullKmsProvidersMap).
			SetKeyVaultNamespace(kvNamespace).
			SetSchemaMap(schemaMap).
			SetTLSConfig(tlsConfig).
			SetExtraOptions(getCryptSharedLibExtraOptions())

		ceo := options.ClientEncryption().
			SetKmsProviders(fullKmsProvidersMap).
			SetKeyVaultNamespace(kvNamespace).
			SetTLSConfig(tlsConfig)

		awsMasterKey := bson.D{
			{"region", "us-east-1"},
			{"key", "arn:aws:kms:us-east-1:579766882180:key/89fcc2c4-08b0-4bd9-9f25-e30687b580d0"},
		}
		azureMasterKey := bson.D{
			{"keyVaultEndpoint", "key-vault-csfle.vault.azure.net"},
			{"keyName", "key-name-csfle"},
		}
		gcpMasterKey := bson.D{
			{"projectId", "devprod-drivers"},
			{"location", "global"},
			{"keyRing", "key-ring-csfle"},
			{"keyName", "key-name-csfle"},
		}
		kmipMasterKey := bson.D{}
		testCases := []struct {
			provider  string
			masterKey interface{}
		}{
			{"local", nil},
			{"aws", awsMasterKey},
			{"azure", azureMasterKey},
			{"gcp", gcpMasterKey},
			{"kmip", kmipMasterKey},
		}
		for _, tc := range testCases {
			mt.Run(tc.provider, func(mt *mtest.T) {
				if tc.provider == "kmip" && "" == os.Getenv("KMS_MOCK_SERVERS_RUNNING") {
					mt.Skipf("Skipping test as KMS_MOCK_SERVERS_RUNNING is not set")
				}
				var startedEvents []*event.CommandStartedEvent
				monitor := &event.CommandMonitor{
					Started: func(_ context.Context, evt *event.CommandStartedEvent) {
						startedEvents = append(startedEvents, evt)
					},
				}
				kvClientOpts := options.Client().ApplyURI(mtest.ClusterURI()).SetMonitor(monitor)
				testutil.AddTestServerAPIVersion(kvClientOpts)
				cpt := setup(mt, aeo, kvClientOpts, ceo)
				defer cpt.teardown(mt)

				// create data key
				keyAltName := fmt.Sprintf("%s_altname", tc.provider)
				dataKeyOpts := options.DataKey().SetKeyAltNames([]string{keyAltName})
				if tc.masterKey != nil {
					dataKeyOpts.SetMasterKey(tc.masterKey)
				}
				dataKeyID, err := cpt.clientEnc.CreateDataKey(context.Background(), tc.provider, dataKeyOpts)
				assert.Nil(mt, err, "CreateDataKey error: %v", err)
				assert.Equal(mt, keySubtype, dataKeyID.Subtype,
					"expected data key subtype %v, got %v", keySubtype, dataKeyID.Subtype)

				// assert that the key exists in the key vault
				cursor, err := cpt.keyVaultColl.Find(context.Background(), bson.D{{"_id", dataKeyID}})
				assert.Nil(mt, err, "key vault Find error: %v", err)
				assert.True(mt, cursor.Next(context.Background()), "no keys found in key vault")
				provider := cursor.Current.Lookup("masterKey", "provider").StringValue()
				assert.Equal(mt, tc.provider, provider, "expected provider %v, got %v", tc.provider, provider)
				assert.False(mt, cursor.Next(context.Background()), "unexpected document in key vault: %v", cursor.Current)

				// verify that the key was inserted using write concern majority
				assert.Equal(mt, 1, len(startedEvents), "expected 1 CommandStartedEvent, got %v", len(startedEvents))
				evt := startedEvents[0]
				assert.Equal(mt, "insert", evt.CommandName, "expected command 'insert', got '%v'", evt.CommandName)
				writeConcernVal, err := evt.Command.LookupErr("writeConcern")
				assert.Nil(mt, err, "expected writeConcern in command %s", evt.Command)
				wString := writeConcernVal.Document().Lookup("w").StringValue()
				assert.Equal(mt, "majority", wString, "expected write concern 'majority', got %v", wString)

				// encrypt a value with the new key by ID
				valueToEncrypt := fmt.Sprintf("hello %s", tc.provider)
				rawVal := bson.RawValue{Type: bson.TypeString, Value: bsoncore.AppendString(nil, valueToEncrypt)}
				encrypted, err := cpt.clientEnc.Encrypt(context.Background(), rawVal,
					options.Encrypt().SetAlgorithm(deterministicAlgorithm).SetKeyID(dataKeyID))
				assert.Nil(mt, err, "Encrypt error while encrypting value by ID: %v", err)
				assert.Equal(mt, encryptedValueSubtype, encrypted.Subtype,
					"expected encrypted value subtype %v, got %v", encryptedValueSubtype, encrypted.Subtype)

				// insert an encrypted value. the value shouldn't be encrypted again because it's not in the schema.
				_, err = cpt.cseColl.InsertOne(context.Background(), bson.D{{"_id", tc.provider}, {"value", encrypted}})
				assert.Nil(mt, err, "InsertOne error: %v", err)

				// find the inserted document. the value should be decrypted automatically
				resBytes, err := cpt.cseColl.FindOne(context.Background(), bson.D{{"_id", tc.provider}}).DecodeBytes()
				assert.Nil(mt, err, "Find error: %v", err)
				foundVal := resBytes.Lookup("value").StringValue()
				assert.Equal(mt, valueToEncrypt, foundVal, "expected value %v, got %v", valueToEncrypt, foundVal)

				// encrypt a value with an alternate name for the new key
				altEncrypted, err := cpt.clientEnc.Encrypt(context.Background(), rawVal,
					options.Encrypt().SetAlgorithm(deterministicAlgorithm).SetKeyAltName(keyAltName))
				assert.Nil(mt, err, "Encrypt error while encrypting value by alt key name: %v", err)
				assert.Equal(mt, encryptedValueSubtype, altEncrypted.Subtype,
					"expected encrypted value subtype %v, got %v", encryptedValueSubtype, altEncrypted.Subtype)
				assert.Equal(mt, encrypted.Data, altEncrypted.Data,
					"expected data %v, got %v", encrypted.Data, altEncrypted.Data)

				// insert an encrypted value for an auto-encrypted field
				_, err = cpt.cseColl.InsertOne(context.Background(), bson.D{{"encrypted_placeholder", encrypted}})
				assert.NotNil(mt, err, "expected InsertOne error, got nil")
			})
		}
	})
	mt.RunOpts("3. external key vault", noClientOpts, func(mt *mtest.T) {
		testCases := []struct {
			name          string
			externalVault bool
		}{
			{"with external vault", true},
			{"without external vault", false},
		}

		for _, tc := range testCases {
			mt.Run(tc.name, func(mt *mtest.T) {
				// setup options structs
				kmsProviders := map[string]map[string]interface{}{
					"local": {
						"key": localMasterKey,
					},
				}
				schemaMap := map[string]interface{}{"db.coll": readJSONFile(mt, "external-schema.json")}
				aeo := options.AutoEncryption().
					SetKmsProviders(kmsProviders).
					SetKeyVaultNamespace(kvNamespace).
					SetSchemaMap(schemaMap).
					SetExtraOptions(getCryptSharedLibExtraOptions())
				ceo := options.ClientEncryption().
					SetKmsProviders(kmsProviders).
					SetKeyVaultNamespace(kvNamespace)
				kvClientOpts := defaultKvClientOptions

				if tc.externalVault {
					externalKvOpts := options.Client().ApplyURI(mtest.ClusterURI()).SetAuth(options.Credential{
						Username: "fake-user",
						Password: "fake-password",
					})
					testutil.AddTestServerAPIVersion(externalKvOpts)
					aeo.SetKeyVaultClientOptions(externalKvOpts)
					kvClientOpts = externalKvOpts
				}
				cpt := setup(mt, aeo, kvClientOpts, ceo)
				defer cpt.teardown(mt)

				// manually insert data key
				key := readJSONFile(mt, "external-key.json")
				_, err := cpt.keyVaultColl.InsertOne(context.Background(), key)
				assert.Nil(mt, err, "InsertOne error for data key: %v", err)
				subtype, data := key.Lookup("_id").Binary()
				dataKeyID := primitive.Binary{Subtype: subtype, Data: data}

				doc := bson.D{{"encrypted", "test"}}
				_, insertErr := cpt.cseClient.Database("db").Collection("coll").InsertOne(context.Background(), doc)
				rawVal := bson.RawValue{Type: bson.TypeString, Value: bsoncore.AppendString(nil, "test")}
				_, encErr := cpt.clientEnc.Encrypt(context.Background(), rawVal,
					options.Encrypt().SetKeyID(dataKeyID).SetAlgorithm(deterministicAlgorithm))

				if tc.externalVault {
					assert.NotNil(mt, insertErr, "expected InsertOne auth error, got nil")
					assert.NotNil(mt, encErr, "expected Encrypt auth error, got nil")
					assert.True(mt, strings.Contains(insertErr.Error(), "auth error"),
						"expected InsertOne auth error, got %v", insertErr)
					assert.True(mt, strings.Contains(encErr.Error(), "auth error"),
						"expected Encrypt auth error, got %v", insertErr)
					return
				}
				assert.Nil(mt, insertErr, "InsertOne error: %v", insertErr)
				assert.Nil(mt, encErr, "Encrypt error: %v", err)
			})
		}
	})
	mt.Run("4. bson size limits", func(mt *mtest.T) {
		kmsProviders := map[string]map[string]interface{}{
			"local": {
				"key": localMasterKey,
			},
		}
		aeo := options.AutoEncryption().
			SetKmsProviders(kmsProviders).
			SetKeyVaultNamespace(kvNamespace).
			SetExtraOptions(getCryptSharedLibExtraOptions())
		cpt := setup(mt, aeo, nil, nil)
		defer cpt.teardown(mt)

		// create coll with JSON schema
		err := mt.Client.Database("db").RunCommand(context.Background(), bson.D{
			{"create", "coll"},
			{"validator", bson.D{
				{"$jsonSchema", readJSONFile(mt, "limits-schema.json")},
			}},
		}).Err()
		assert.Nil(mt, err, "create error with validator: %v", err)

		// insert key
		key := readJSONFile(mt, "limits-key.json")
		_, err = cpt.keyVaultColl.InsertOne(context.Background(), key)
		assert.Nil(mt, err, "InsertOne error for key: %v", err)

		var builder2mb, builder16mb strings.Builder
		for i := 0; i < cryptMaxBatchSizeBytes; i++ {
			builder2mb.WriteByte('a')
		}
		for i := 0; i < maxBsonObjSize; i++ {
			builder16mb.WriteByte('a')
		}
		complete2mbStr := builder2mb.String()
		complete16mbStr := builder16mb.String()

		// insert a document over 2MiB
		doc := bson.D{{"over_2mib_under_16mib", complete2mbStr}}
		_, err = cpt.cseColl.InsertOne(context.Background(), doc)
		assert.Nil(mt, err, "InsertOne error for 2MiB document: %v", err)

		str := complete2mbStr[:cryptMaxBatchSizeBytes-2000] // remove last 2000 bytes
		limitsDoc := readJSONFile(mt, "limits-doc.json")

		// insert a doc smaller than 2MiB that is bigger than 2MiB after encryption
		var extendedLimitsDoc []byte
		extendedLimitsDoc = append(extendedLimitsDoc, limitsDoc...)
		extendedLimitsDoc = extendedLimitsDoc[:len(extendedLimitsDoc)-1] // remove last byte to add new fields
		extendedLimitsDoc = bsoncore.AppendStringElement(extendedLimitsDoc, "_id", "encryption_exceeds_2mib")
		extendedLimitsDoc = bsoncore.AppendStringElement(extendedLimitsDoc, "unencrypted", str)
		extendedLimitsDoc, _ = bsoncore.AppendDocumentEnd(extendedLimitsDoc, 0)
		_, err = cpt.cseColl.InsertOne(context.Background(), extendedLimitsDoc)
		assert.Nil(mt, err, "error inserting extended limits document: %v", err)

		// bulk insert two 2MiB documents, each over 2 MiB
		// each document should be split into its own batch because the documents are bigger than 2MiB but smaller
		// than 16MiB
		cpt.cseStarted = cpt.cseStarted[:0]
		firstDoc := bson.D{{"_id", "over_2mib_1"}, {"unencrypted", complete2mbStr}}
		secondDoc := bson.D{{"_id", "over_2mib_2"}, {"unencrypted", complete2mbStr}}
		_, err = cpt.cseColl.InsertMany(context.Background(), []interface{}{firstDoc, secondDoc})
		assert.Nil(mt, err, "InsertMany error for small documents: %v", err)
		assert.Equal(mt, 2, len(cpt.cseStarted), "expected 2 insert events, got %d", len(cpt.cseStarted))

		// bulk insert two documents
		str = complete2mbStr[:cryptMaxBatchSizeBytes-20000]
		firstBulkDoc := make([]byte, len(limitsDoc))
		copy(firstBulkDoc, limitsDoc)
		firstBulkDoc = firstBulkDoc[:len(firstBulkDoc)-1] // remove last byte to append new fields
		firstBulkDoc = bsoncore.AppendStringElement(firstBulkDoc, "_id", "encryption_exceeds_2mib_1")
		firstBulkDoc = bsoncore.AppendStringElement(firstBulkDoc, "unencrypted", string(str))
		firstBulkDoc, _ = bsoncore.AppendDocumentEnd(firstBulkDoc, 0)

		secondBulkDoc := make([]byte, len(limitsDoc))
		copy(secondBulkDoc, limitsDoc)
		secondBulkDoc = secondBulkDoc[:len(secondBulkDoc)-1] // remove last byte to append new fields
		secondBulkDoc = bsoncore.AppendStringElement(secondBulkDoc, "_id", "encryption_exceeds_2mib_2")
		secondBulkDoc = bsoncore.AppendStringElement(secondBulkDoc, "unencrypted", string(str))
		secondBulkDoc, _ = bsoncore.AppendDocumentEnd(secondBulkDoc, 0)

		cpt.cseStarted = cpt.cseStarted[:0]
		_, err = cpt.cseColl.InsertMany(context.Background(), []interface{}{firstBulkDoc, secondBulkDoc})
		assert.Nil(mt, err, "InsertMany error for large documents: %v", err)
		assert.Equal(mt, 2, len(cpt.cseStarted), "expected 2 insert events, got %d", len(cpt.cseStarted))

		// insert a document slightly smaller than 16MiB and expect the operation to succeed
		doc = bson.D{{"_id", "under_16mib"}, {"unencrypted", complete16mbStr[:maxBsonObjSize-2000]}}
		_, err = cpt.cseColl.InsertOne(context.Background(), doc)
		assert.Nil(mt, err, "InsertOne error: %v", err)

		// insert a document over 16MiB and expect the operation to fail
		var over16mb []byte
		over16mb = append(over16mb, limitsDoc...)
		over16mb = over16mb[:len(over16mb)-1] // remove last byte
		over16mb = bsoncore.AppendStringElement(over16mb, "_id", "encryption_exceeds_16mib")
		over16mb = bsoncore.AppendStringElement(over16mb, "unencrypted", complete16mbStr[:maxBsonObjSize-2000])
		over16mb, _ = bsoncore.AppendDocumentEnd(over16mb, 0)
		_, err = cpt.cseColl.InsertOne(context.Background(), over16mb)
		assert.NotNil(mt, err, "expected InsertOne error for document over 16MiB, got nil")
	})
	mt.Run("5. views are prohibited", func(mt *mtest.T) {
		kmsProviders := map[string]map[string]interface{}{
			"local": {
				"key": localMasterKey,
			},
		}
		aeo := options.AutoEncryption().
			SetKmsProviders(kmsProviders).
			SetKeyVaultNamespace(kvNamespace).
			SetExtraOptions(getCryptSharedLibExtraOptions())
		cpt := setup(mt, aeo, nil, nil)
		defer cpt.teardown(mt)

		// create view on db.coll
		mt.CreateCollection(mtest.Collection{
			Name:         "view",
			DB:           cpt.cseColl.Database().Name(),
			ViewOn:       "coll",
			ViewPipeline: mongo.Pipeline{},
		}, true)

		view := cpt.cseColl.Database().Collection("view")
		_, err := view.InsertOne(context.Background(), bson.D{{"_id", "insert_on_view"}})
		assert.NotNil(mt, err, "expected InsertOne error on view, got nil")
		errStr := strings.ToLower(err.Error())
		viewErrSubstr := "cannot auto encrypt a view"
		assert.True(mt, strings.Contains(errStr, viewErrSubstr),
			"expected error '%v' to contain substring '%v'", errStr, viewErrSubstr)
	})
	mt.RunOpts("6. corpus test", noClientOpts, func(mt *mtest.T) {
		if "" == os.Getenv("KMS_MOCK_SERVERS_RUNNING") {
			mt.Skipf("Skipping test as KMS_MOCK_SERVERS_RUNNING is not set")
		}
		corpusSchema := readJSONFile(mt, "corpus-schema.json")
		localSchemaMap := map[string]interface{}{
			"db.coll": corpusSchema,
		}

		tlsConfig := make(map[string]*tls.Config)
		if tlsCAFileKMIP != "" && tlsClientCertificateKeyFileKMIP != "" {
			tlsOpts := map[string]interface{}{
				"tlsCertificateKeyFile": tlsClientCertificateKeyFileKMIP,
				"tlsCAFile":             tlsCAFileKMIP,
			}
			kmipConfig, err := options.BuildTLSConfig(tlsOpts)
			assert.Nil(mt, err, "BuildTLSConfig error: %v", err)
			tlsConfig["kmip"] = kmipConfig
		}

		getBaseAutoEncryptionOpts := func() *options.AutoEncryptionOptions {
			return options.AutoEncryption().
				SetKmsProviders(fullKmsProvidersMap).
				SetKeyVaultNamespace(kvNamespace).
				SetTLSConfig(tlsConfig).
				SetExtraOptions(getCryptSharedLibExtraOptions())
		}

		testCases := []struct {
			name   string
			aeo    *options.AutoEncryptionOptions
			schema bson.Raw // the schema to create the collection. if nil, the collection won't be explicitly created
		}{
			{"remote schema", getBaseAutoEncryptionOpts(), corpusSchema},
			{"local schema", getBaseAutoEncryptionOpts().SetSchemaMap(localSchemaMap), nil},
		}

		for _, tc := range testCases {
			mt.Run(tc.name, func(mt *mtest.T) {
				ceo := options.ClientEncryption().
					SetKmsProviders(fullKmsProvidersMap).
					SetKeyVaultNamespace(kvNamespace).
					SetTLSConfig(tlsConfig)

				cpt := setup(mt, tc.aeo, defaultKvClientOptions, ceo)
				defer cpt.teardown(mt)

				// create collection with JSON schema
				if tc.schema != nil {
					db := cpt.coll.Database()
					err := db.RunCommand(context.Background(), bson.D{
						{"create", "coll"},
						{"validator", bson.D{
							{"$jsonSchema", readJSONFile(mt, "corpus-schema.json")},
						}},
					}).Err()
					assert.Nil(mt, err, "create error with validator: %v", err)
				}

				// Manually insert keys for each KMS provider into the key vault.
				_, err := cpt.keyVaultColl.InsertMany(context.Background(), []interface{}{
					readJSONFile(mt, "corpus-key-local.json"),
					readJSONFile(mt, "corpus-key-aws.json"),
					readJSONFile(mt, "corpus-key-azure.json"),
					readJSONFile(mt, "corpus-key-gcp.json"),
					readJSONFile(mt, "corpus-key-kmip.json"),
				})
				assert.Nil(mt, err, "InsertMany error for key vault: %v", err)

				// read original corpus and recursively copy over each value to new corpus, encrypting certain values
				// when needed
				corpus := readJSONFile(mt, "corpus.json")
				cidx, copied := bsoncore.AppendDocumentStart(nil)
				elems, _ := corpus.Elements()

				// Keys for top-level non-document elements that should be copied directly.
				copiedKeys := map[string]struct{}{
					"_id":           {},
					"altname_aws":   {},
					"altname_local": {},
					"altname_azure": {},
					"altname_gcp":   {},
					"altname_kmip":  {},
				}

				for _, elem := range elems {
					key := elem.Key()
					val := elem.Value()

					if _, ok := copiedKeys[key]; ok {
						copied = bsoncore.AppendStringElement(copied, key, val.StringValue())
						continue
					}

					doc := val.Document()
					switch method := doc.Lookup("method").StringValue(); method {
					case "auto":
						// Copy the value directly because it will be auto-encrypted later.
						copied = bsoncore.AppendDocumentElement(copied, key, doc)
						continue
					case "explicit":
						// Handled below.
					default:
						mt.Fatalf("unrecognized 'method' value %q", method)
					}

					// explicitly encrypt value
					algorithm := deterministicAlgorithm
					if doc.Lookup("algo").StringValue() == "rand" {
						algorithm = randomAlgorithm
					}
					eo := options.Encrypt().SetAlgorithm(algorithm)

					identifier := doc.Lookup("identifier").StringValue()
					kms := doc.Lookup("kms").StringValue()
					switch identifier {
					case "id":
						var keyID string
						switch kms {
						case "local":
							keyID = "LOCALAAAAAAAAAAAAAAAAA=="
						case "aws":
							keyID = "AWSAAAAAAAAAAAAAAAAAAA=="
						case "azure":
							keyID = "AZUREAAAAAAAAAAAAAAAAA=="
						case "gcp":
							keyID = "GCPAAAAAAAAAAAAAAAAAAA=="
						case "kmip":
							keyID = "KMIPAAAAAAAAAAAAAAAAAA=="
						default:
							mt.Fatalf("unrecognized KMS provider %q", kms)
						}

						keyIDBytes, err := base64.StdEncoding.DecodeString(keyID)
						assert.Nil(mt, err, "base64 DecodeString error: %v", err)
						eo.SetKeyID(primitive.Binary{Subtype: 4, Data: keyIDBytes})
					case "altname":
						eo.SetKeyAltName(kms) // alt name for a key is the same as the KMS name
					default:
						mt.Fatalf("unrecognized identifier: %v", identifier)
					}

					// iterate over all elements in the document. copy elements directly, except for ones that need to
					// be encrypted, which should be copied after encryption.
					var nestedIdx int32
					nestedIdx, copied = bsoncore.AppendDocumentElementStart(copied, key)
					docElems, _ := doc.Elements()
					for _, de := range docElems {
						deKey := de.Key()
						deVal := de.Value()

						// element to encrypt has key "value"
						if deKey != "value" {
							copied = bsoncore.AppendValueElement(copied, deKey, rawValueToCoreValue(deVal))
							continue
						}

						encrypted, err := cpt.clientEnc.Encrypt(context.Background(), deVal, eo)
						if !doc.Lookup("allowed").Boolean() {
							// if allowed is false, encryption should error. in this case, the unencrypted value should be
							// copied over
							assert.NotNil(mt, err, "expected error encrypting value for key %v, got nil", key)
							copied = bsoncore.AppendValueElement(copied, deKey, rawValueToCoreValue(deVal))
							continue
						}

						// copy encrypted value
						assert.Nil(mt, err, "Encrypt error for key %v: %v", key, err)
						copied = bsoncore.AppendBinaryElement(copied, deKey, encrypted.Subtype, encrypted.Data)
					}
					copied, _ = bsoncore.AppendDocumentEnd(copied, nestedIdx)
				}
				copied, _ = bsoncore.AppendDocumentEnd(copied, cidx)

				// insert document with encrypted values
				_, err = cpt.cseColl.InsertOne(context.Background(), copied)
				assert.Nil(mt, err, "InsertOne error for corpus document: %v", err)

				// find document using client with encryption and assert it matches original
				decryptedDoc, err := cpt.cseColl.FindOne(context.Background(), bson.D{}).DecodeBytes()
				assert.Nil(mt, err, "Find error with encrypted client: %v", err)
				assert.Equal(mt, corpus, decryptedDoc, "expected document %v, got %v", corpus, decryptedDoc)

				// find document using a client without encryption enabled and assert fields remain encrypted
				corpusEncrypted := readJSONFile(mt, "corpus-encrypted.json")
				foundDoc, err := cpt.coll.FindOne(context.Background(), bson.D{}).DecodeBytes()
				assert.Nil(mt, err, "Find error with unencrypted client: %v", err)

				encryptedElems, _ := corpusEncrypted.Elements()
				for _, encryptedElem := range encryptedElems {
					// skip non-document fields
					encryptedDoc, ok := encryptedElem.Value().DocumentOK()
					if !ok {
						continue
					}

					allowed := encryptedDoc.Lookup("allowed").Boolean()
					expectedKey := encryptedElem.Key()
					expectedVal := encryptedDoc.Lookup("value")
					foundVal := foundDoc.Lookup(expectedKey).Document().Lookup("value")

					// for deterministic encryption, the value should be exactly equal
					// for random encryption, the value should not be equal if allowed is true
					algo := encryptedDoc.Lookup("algo").StringValue()
					switch algo {
					case "det":
						assert.True(mt, expectedVal.Equal(foundVal),
							"expected value %v for key %v, got %v", expectedVal, expectedKey, foundVal)
					case "rand":
						if allowed {
							assert.False(mt, expectedVal.Equal(foundVal),
								"expected values for key %v to be different but were %v", expectedKey, expectedVal)
						}
					}

					// if allowed is true, decrypt both values with clientEnc and validate equality
					if allowed {
						sub, data := expectedVal.Binary()
						expectedDecrypted, err := cpt.clientEnc.Decrypt(context.Background(), primitive.Binary{Subtype: sub, Data: data})
						assert.Nil(mt, err, "Decrypt error: %v", err)
						sub, data = foundVal.Binary()
						actualDecrypted, err := cpt.clientEnc.Decrypt(context.Background(), primitive.Binary{Subtype: sub, Data: data})
						assert.Nil(mt, err, "Decrypt error: %v", err)

						assert.True(mt, expectedDecrypted.Equal(actualDecrypted),
							"expected decrypted value %v for key %v, got %v", expectedDecrypted, expectedKey, actualDecrypted)
						continue
					}

					// if allowed is false, validate found value equals the original value in corpus
					corpusVal := corpus.Lookup(expectedKey).Document().Lookup("value")
					assert.True(mt, corpusVal.Equal(foundVal),
						"expected value %v for key %v, got %v", corpusVal, expectedKey, foundVal)
				}
			})
		}
	})
	mt.Run("7. custom endpoint", func(mt *mtest.T) {
		validKmsProviders := map[string]map[string]interface{}{
			"aws": {
				"accessKeyId":     awsAccessKeyID,
				"secretAccessKey": awsSecretAccessKey,
			},
			"azure": {
				"tenantId":                 azureTenantID,
				"clientId":                 azureClientID,
				"clientSecret":             azureClientSecret,
				"identityPlatformEndpoint": "login.microsoftonline.com:443",
			},
			"gcp": {
				"email":      gcpEmail,
				"privateKey": gcpPrivateKey,
				"endpoint":   "oauth2.googleapis.com:443",
			},
			"kmip": {
				"endpoint": "localhost:5698",
			},
		}

		tlsConfig := make(map[string]*tls.Config)
		if tlsCAFileKMIP != "" && tlsClientCertificateKeyFileKMIP != "" {
			tlsOpts := map[string]interface{}{
				"tlsCertificateKeyFile": tlsClientCertificateKeyFileKMIP,
				"tlsCAFile":             tlsCAFileKMIP,
			}
			kmipConfig, err := options.BuildTLSConfig(tlsOpts)
			assert.Nil(mt, err, "BuildTLSConfig error: %v", err)
			tlsConfig["kmip"] = kmipConfig
		}

		validClientEncryptionOptions := options.ClientEncryption().
			SetKmsProviders(validKmsProviders).
			SetKeyVaultNamespace(kvNamespace).
			SetTLSConfig(tlsConfig)

		invalidKmsProviders := map[string]map[string]interface{}{
			"azure": {
				"tenantId":                 azureTenantID,
				"clientId":                 azureClientID,
				"clientSecret":             azureClientSecret,
				"identityPlatformEndpoint": "doesnotexist.invalid:443",
			},
			"gcp": {
				"email":      gcpEmail,
				"privateKey": gcpPrivateKey,
				"endpoint":   "doesnotexist.invalid:443",
			},
			"kmip": {
				"endpoint": "doesnotexist.local:5698",
			},
		}

		invalidClientEncryptionOptions := options.ClientEncryption().
			SetKmsProviders(invalidKmsProviders).
			SetKeyVaultNamespace(kvNamespace).
			SetTLSConfig(tlsConfig)

		awsSuccessWithoutEndpoint := map[string]interface{}{
			"region": "us-east-1",
			"key":    "arn:aws:kms:us-east-1:579766882180:key/89fcc2c4-08b0-4bd9-9f25-e30687b580d0",
		}
		awsSuccessWithEndpoint := map[string]interface{}{
			"region":   "us-east-1",
			"key":      "arn:aws:kms:us-east-1:579766882180:key/89fcc2c4-08b0-4bd9-9f25-e30687b580d0",
			"endpoint": "kms.us-east-1.amazonaws.com",
		}
		awsSuccessWithHTTPSEndpoint := map[string]interface{}{
			"region":   "us-east-1",
			"key":      "arn:aws:kms:us-east-1:579766882180:key/89fcc2c4-08b0-4bd9-9f25-e30687b580d0",
			"endpoint": "kms.us-east-1.amazonaws.com:443",
		}
		awsFailureConnectionError := map[string]interface{}{
			"region":   "us-east-1",
			"key":      "arn:aws:kms:us-east-1:579766882180:key/89fcc2c4-08b0-4bd9-9f25-e30687b580d0",
			"endpoint": "kms.us-east-1.amazonaws.com:12345",
		}
		awsFailureInvalidEndpoint := map[string]interface{}{
			"region":   "us-east-1",
			"key":      "arn:aws:kms:us-east-1:579766882180:key/89fcc2c4-08b0-4bd9-9f25-e30687b580d0",
			"endpoint": "kms.us-east-2.amazonaws.com",
		}
		awsFailureParseError := map[string]interface{}{
			"region":   "us-east-1",
			"key":      "arn:aws:kms:us-east-1:579766882180:key/89fcc2c4-08b0-4bd9-9f25-e30687b580d0",
			"endpoint": "doesnotexist.invalid",
		}
		azure := map[string]interface{}{
			"keyVaultEndpoint": "key-vault-csfle.vault.azure.net",
			"keyName":          "key-name-csfle",
		}
		gcpSuccess := map[string]interface{}{
			"projectId": "devprod-drivers",
			"location":  "global",
			"keyRing":   "key-ring-csfle",
			"keyName":   "key-name-csfle",
			"endpoint":  "cloudkms.googleapis.com:443",
		}
		gcpFailure := map[string]interface{}{
			"projectId": "devprod-drivers",
			"location":  "global",
			"keyRing":   "key-ring-csfle",
			"keyName":   "key-name-csfle",
			"endpoint":  "doesnotexist.invalid:443",
		}
		kmipSuccessWithoutEndpoint := map[string]interface{}{
			"keyId": "1",
		}
		kmipSuccessWithEndpoint := map[string]interface{}{
			"keyId":    "1",
			"endpoint": "localhost:5698",
		}
		kmipFailureInvalidEndpoint := map[string]interface{}{
			"keyId":    "1",
			"endpoint": "doesnotexist.local:5698",
		}

		testCases := []struct {
			name                                  string
			provider                              string
			masterKey                             interface{}
			errorSubstring                        string
			testInvalidClientEncryption           bool
			invalidClientEncryptionErrorSubstring string
		}{
			{"Case 1: aws success without endpoint", "aws", awsSuccessWithoutEndpoint, "", false, ""},
			{"Case 2: aws success with endpoint", "aws", awsSuccessWithEndpoint, "", false, ""},
			{"Case 3: aws success with https endpoint", "aws", awsSuccessWithHTTPSEndpoint, "", false, ""},
			{"Case 4: aws failure with connection error", "aws", awsFailureConnectionError, "connection refused", false, ""},
			{"Case 5: aws failure with wrong endpoint", "aws", awsFailureInvalidEndpoint, "mongocrypt error", false, ""},
			{"Case 6: aws failure with parse error", "aws", awsFailureParseError, "no such host", false, ""},
			{"Case 7: azure success", "azure", azure, "", true, "no such host"},
			{"Case 8: gcp success", "gcp", gcpSuccess, "", true, "no such host"},
			{"Case 9: gcp failure", "gcp", gcpFailure, "Invalid KMS response", false, ""},
			{"Case 10: kmip success without endpoint", "kmip", kmipSuccessWithoutEndpoint, "", true, "no such host"},
			{"Case 11: kmip success with endpoint", "kmip", kmipSuccessWithEndpoint, "", false, ""},
			{"Case 12: kmip failure with invalid endpoint", "kmip", kmipFailureInvalidEndpoint, "no such host", false, ""},
		}
		for _, tc := range testCases {
			mt.Run(tc.name, func(mt *mtest.T) {
				if strings.Contains(tc.name, "kmip") && "" == os.Getenv("KMS_MOCK_SERVERS_RUNNING") {
					mt.Skipf("Skipping test as KMS_MOCK_SERVERS_RUNNING is not set")
				}
				cpt := setup(mt, nil, defaultKvClientOptions, validClientEncryptionOptions)
				defer cpt.teardown(mt)

				dkOpts := options.DataKey().SetMasterKey(tc.masterKey)
				createdKey, err := cpt.clientEnc.CreateDataKey(context.Background(), tc.provider, dkOpts)
				if tc.errorSubstring != "" {
					assert.NotNil(mt, err, "expected error, got nil")
					errSubstr := tc.errorSubstring
					if runtime.GOOS == "windows" && errSubstr == "connection refused" {
						// tls.Dial returns an error that does not contain the substring "connection refused"
						// on Windows machines
						errSubstr = "No connection could be made because the target machine actively refused it"
					}
					assert.True(mt, strings.Contains(err.Error(), errSubstr),
						"expected error '%s' to contain '%s'", err.Error(), errSubstr)
					return
				}
				assert.Nil(mt, err, "CreateDataKey error: %v", err)

				encOpts := options.Encrypt().SetKeyID(createdKey).SetAlgorithm(deterministicAlgorithm)
				testVal := bson.RawValue{
					Type:  bson.TypeString,
					Value: bsoncore.AppendString(nil, "test"),
				}
				encrypted, err := cpt.clientEnc.Encrypt(context.Background(), testVal, encOpts)
				assert.Nil(mt, err, "Encrypt error: %v", err)
				decrypted, err := cpt.clientEnc.Decrypt(context.Background(), encrypted)
				assert.Nil(mt, err, "Decrypt error: %v", err)
				assert.Equal(mt, testVal, decrypted, "expected value %s, got %s", testVal, decrypted)

				if !tc.testInvalidClientEncryption {
					return
				}

				invalidClientEncryption, err := mongo.NewClientEncryption(cpt.kvClient, invalidClientEncryptionOptions)
				assert.Nil(mt, err, "error creating invalidClientEncryption object: %v", err)
				defer invalidClientEncryption.Close(context.Background())

				invalidKeyOpts := options.DataKey().SetMasterKey(tc.masterKey)
				_, err = invalidClientEncryption.CreateDataKey(context.Background(), tc.provider, invalidKeyOpts)
				assert.NotNil(mt, err, "expected CreateDataKey error, got nil")
				assert.True(mt, strings.Contains(err.Error(), tc.invalidClientEncryptionErrorSubstring),
					"expected error %v to contain substring '%v'", err, tc.invalidClientEncryptionErrorSubstring)
			})
		}
	})
	mt.RunOpts("8. bypass mongocryptd spawning", noClientOpts, func(mt *mtest.T) {
		kmsProviders := map[string]map[string]interface{}{
			"local": {
				"key": localMasterKey,
			},
		}
		schemaMap := map[string]interface{}{
			"db.coll": readJSONFile(mt, "external-schema.json"),
		}

		// All mongocryptd options use port 27021 instead of the default 27020 to avoid interference
		// with mongocryptd instances spawned by previous tests. Explicitly disable loading the
		// crypt_shared library to make sure we're testing mongocryptd spawning behavior that is not
		// influenced by loading the crypt_shared library.
		mongocryptdBypassSpawnTrue := map[string]interface{}{
			"mongocryptdBypassSpawn":              true,
			"mongocryptdURI":                      "mongodb://localhost:27021/db?serverSelectionTimeoutMS=1000",
			"mongocryptdSpawnArgs":                []string{"--pidfilepath=bypass-spawning-mongocryptd.pid", "--port=27021"},
			"__cryptSharedLibDisabledForTestOnly": true, // Disable loading the crypt_shared library.
		}
		mongocryptdBypassSpawnFalse := map[string]interface{}{
			"mongocryptdBypassSpawn":              false,
			"mongocryptdSpawnArgs":                []string{"--pidfilepath=bypass-spawning-mongocryptd.pid", "--port=27021"},
			"__cryptSharedLibDisabledForTestOnly": true, // Disable loading the crypt_shared library.
		}
		mongocryptdBypassSpawnNotSet := map[string]interface{}{
			"mongocryptdSpawnArgs":                []string{"--pidfilepath=bypass-spawning-mongocryptd.pid", "--port=27021"},
			"__cryptSharedLibDisabledForTestOnly": true, // Disable loading the crypt_shared library.
		}

		testCases := []struct {
			name                    string
			mongocryptdOpts         map[string]interface{}
			setBypassAutoEncryption bool
			bypassAutoEncryption    bool
			bypassQueryAnalysis     bool
		}{
			{
				name:            "mongocryptdBypassSpawn only",
				mongocryptdOpts: mongocryptdBypassSpawnTrue,
			},
			{
				name:                    "bypassAutoEncryption only",
				mongocryptdOpts:         mongocryptdBypassSpawnNotSet,
				setBypassAutoEncryption: true,
				bypassAutoEncryption:    true,
			},
			{
				name:                    "mongocryptdBypassSpawn false, bypassAutoEncryption true",
				mongocryptdOpts:         mongocryptdBypassSpawnFalse,
				setBypassAutoEncryption: true,
				bypassAutoEncryption:    true,
			},
			{
				name:                    "mongocryptdBypassSpawn true, bypassAutoEncryption false",
				mongocryptdOpts:         mongocryptdBypassSpawnTrue,
				setBypassAutoEncryption: true,
				bypassAutoEncryption:    false,
			},
			{
				name:                "bypassQueryAnalysis only",
				mongocryptdOpts:     mongocryptdBypassSpawnNotSet,
				bypassQueryAnalysis: true,
			},
		}
		for _, tc := range testCases {
			mt.Run(tc.name, func(mt *mtest.T) {
				aeo := options.AutoEncryption().
					SetKmsProviders(kmsProviders).
					SetKeyVaultNamespace(kvNamespace).
					SetSchemaMap(schemaMap).
					SetExtraOptions(tc.mongocryptdOpts)
				if tc.setBypassAutoEncryption {
					aeo.SetBypassAutoEncryption(tc.bypassAutoEncryption)
				}
				aeo.SetBypassQueryAnalysis(tc.bypassQueryAnalysis)
				cpt := setup(mt, aeo, nil, nil)
				defer cpt.teardown(mt)

				_, err := cpt.cseColl.InsertOne(context.Background(), bson.D{{"unencrypted", "test"}})

				// Check for mongocryptd server selection error if auto encryption was not bypassed.
				if !(tc.setBypassAutoEncryption && tc.bypassAutoEncryption) && !tc.bypassQueryAnalysis {
					assert.NotNil(mt, err, "expected InsertOne error, got nil")
					mcryptErr, ok := err.(mongo.MongocryptdError)
					assert.True(mt, ok, "expected error type %T, got %v of type %T", mongo.MongocryptdError{}, err, err)
					assert.True(mt, strings.Contains(mcryptErr.Error(), "server selection error"),
						"expected mongocryptd server selection error, got %v", err)
					return
				}

				// If auto encryption is bypassed, the command should succeed. Create a new client to connect to
				// mongocryptd and verify it is not running.
				assert.Nil(mt, err, "InsertOne error: %v", err)

				mcryptOpts := options.Client().ApplyURI("mongodb://localhost:27021").
					SetServerSelectionTimeout(1 * time.Second)
				testutil.AddTestServerAPIVersion(mcryptOpts)
				mcryptClient, err := mongo.Connect(context.Background(), mcryptOpts)
				assert.Nil(mt, err, "mongocryptd Connect error: %v", err)

				err = mcryptClient.Database("admin").RunCommand(context.Background(), bson.D{{internal.LegacyHelloLowercase, 1}}).Err()
				assert.NotNil(mt, err, "expected mongocryptd legacy hello error, got nil")
				assert.True(mt, strings.Contains(err.Error(), "server selection error"),
					"expected mongocryptd server selection error, got %v", err)
			})
		}
	})
	changeStreamOpts := mtest.NewOptions().
		CreateClient(false).
		Topologies(mtest.ReplicaSet)
	mt.RunOpts("change streams", changeStreamOpts, func(mt *mtest.T) {
		// Change streams can't easily fit into the spec test format because of their tailable nature, so there are two
		// prose tests for them instead:
		//
		// 1. Auto-encryption errors for Watch operations. Collection-level change streams error because the
		// $changeStream aggregation stage is not valid for encryption. Client and database-level streams error because
		// only collection-level operations are valid for encryption.
		//
		// 2. Events are automatically decrypted: If the Watch() is done with BypassAutoEncryption=true, the Watch
		// should succeed and subsequent getMore calls should decrypt documents when necessary.

		var testConfig struct {
			JSONSchema        bson.Raw   `bson:"json_schema"`
			KeyVaultData      []bson.Raw `bson:"key_vault_data"`
			EncryptedDocument bson.Raw   `bson:"encrypted_document"`
			DecryptedDocument bson.Raw   `bson:"decrytped_document"`
		}
		decodeJSONFile(mt, "change-streams-test.json", &testConfig)

		schemaMap := map[string]interface{}{
			"db.coll": testConfig.JSONSchema,
		}
		kmsProviders := map[string]map[string]interface{}{
			"aws": {
				"accessKeyId":     awsAccessKeyID,
				"secretAccessKey": awsSecretAccessKey,
			},
		}

		testCases := []struct {
			name       string
			streamType mongo.StreamType
		}{
			{"client", mongo.ClientStream},
			{"database", mongo.DatabaseStream},
			{"collection", mongo.CollectionStream},
		}
		mt.RunOpts("auto encryption errors", noClientOpts, func(mt *mtest.T) {
			for _, tc := range testCases {
				mt.Run(tc.name, func(mt *mtest.T) {
					autoEncryptionOpts := options.AutoEncryption().
						SetKmsProviders(kmsProviders).
						SetKeyVaultNamespace(kvNamespace).
						SetSchemaMap(schemaMap).
						SetExtraOptions(getCryptSharedLibExtraOptions())
					cpt := setup(mt, autoEncryptionOpts, nil, nil)
					defer cpt.teardown(mt)

					_, err := getWatcher(mt, tc.streamType, cpt).Watch(context.Background(), mongo.Pipeline{})
					assert.NotNil(mt, err, "expected Watch error: %v", err)
				})
			}
		})
		mt.RunOpts("events are automatically decrypted", noClientOpts, func(mt *mtest.T) {
			for _, tc := range testCases {
				mt.Run(tc.name, func(mt *mtest.T) {
					autoEncryptionOpts := options.AutoEncryption().
						SetKmsProviders(kmsProviders).
						SetKeyVaultNamespace(kvNamespace).
						SetSchemaMap(schemaMap).
						SetBypassAutoEncryption(true)
					cpt := setup(mt, autoEncryptionOpts, nil, nil)
					defer cpt.teardown(mt)

					// Insert key vault data so the key can be accessed when starting the change stream.
					insertDocuments(mt, cpt.keyVaultColl, testConfig.KeyVaultData)

					stream, err := getWatcher(mt, tc.streamType, cpt).Watch(context.Background(), mongo.Pipeline{})
					assert.Nil(mt, err, "Watch error: %v", err)
					defer stream.Close(context.Background())

					// Insert already encrypted data and verify that it is automatically decrypted by Next().
					insertDocuments(mt, cpt.coll, []bson.Raw{testConfig.EncryptedDocument})
					assert.True(mt, stream.Next(context.Background()), "expected Next to return true, got false")
					gotDocument := stream.Current.Lookup("fullDocument").Document()
					err = compareDocs(mt, testConfig.DecryptedDocument, gotDocument)
					assert.Nil(mt, err, "compareDocs error: %v", err)
				})
			}
		})
	})
	mt.RunOpts("9. deadlock tests", noClientOpts, func(mt *mtest.T) {
		testcases := []struct {
			description                            string
			maxPoolSize                            uint64
			bypassAutoEncryption                   bool
			keyVaultClientSet                      bool
			clientEncryptedTopologyOpeningExpected int
			clientEncryptedCommandStartedExpected  []startedEvent
			clientKeyVaultCommandStartedExpected   []startedEvent
		}{
			// In the following comments, "special auto encryption options" refers to the "bypassAutoEncryption" and
			// "keyVaultClient" options
			{
				// If the client has a limited maxPoolSize, and no special auto-encryption options are set, the
				// driver should create an internal Client for metadata/keyVault operations.
				"deadlock case 1", 1, false, false, 2,
				[]startedEvent{{"listCollections", "db"}, {"find", "keyvault"}, {"insert", "db"}, {"find", "db"}},
				nil,
			},
			{
				// If the client has a limited maxPoolSize, and a keyVaultClient is set, the driver should create
				// an internal Client for metadata operations.
				"deadlock case 2", 1, false, true, 2,
				[]startedEvent{{"listCollections", "db"}, {"insert", "db"}, {"find", "db"}},
				[]startedEvent{{"find", "keyvault"}},
			},
			{
				// If the client has a limited maxPoolSize, and a bypassAutomaticEncryption=true, the driver should
				// create an internal Client for keyVault operations.
				"deadlock case 3", 1, true, false, 2,
				[]startedEvent{{"find", "db"}, {"find", "keyvault"}},
				nil,
			},
			{
				// If the client has a limited maxPoolSize, bypassAutomaticEncryption=true, and a keyVaultClient is set,
				// the driver should not create an internal Client.
				"deadlock case 4", 1, true, true, 1,
				[]startedEvent{{"find", "db"}},
				[]startedEvent{{"find", "keyvault"}},
			},
			{
				// If the client has an unlimited maxPoolSize, and no special auto-encryption options are set,  the
				// driver should reuse the client for metadata/keyVault operations
				"deadlock case 5", 0, false, false, 1,
				[]startedEvent{{"listCollections", "db"}, {"listCollections", "keyvault"}, {"find", "keyvault"}, {"insert", "db"}, {"find", "db"}},
				nil,
			},
			{
				// If the client has an unlimited maxPoolSize, and a keyVaultClient is set, the driver should reuse the
				// client for metadata operations.
				"deadlock case 6", 0, false, true, 1,
				[]startedEvent{{"listCollections", "db"}, {"insert", "db"}, {"find", "db"}},
				[]startedEvent{{"find", "keyvault"}},
			},
			{
				// If the client has an unlimited maxPoolSize, and bypassAutomaticEncryption=true, the driver should
				// reuse the client for keyVault operations
				"deadlock case 7", 0, true, false, 1,
				[]startedEvent{{"find", "db"}, {"find", "keyvault"}},
				nil,
			},
			{
				// If the client has an unlimited maxPoolSize, bypassAutomaticEncryption=true, and a keyVaultClient is
				// set, the driver should not create an internal Client.
				"deadlock case 8", 0, true, true, 1,
				[]startedEvent{{"find", "db"}},
				[]startedEvent{{"find", "keyvault"}},
			},
		}

		for _, tc := range testcases {
			mt.Run(tc.description, func(mt *mtest.T) {
				var clientEncryptedEvents []startedEvent
				var clientEncryptedTopologyOpening int

				d := newDeadlockTest(mt)
				defer d.disconnect(mt)

				kmsProviders := map[string]map[string]interface{}{
					"local": {"key": localMasterKey},
				}
				aeOpts := options.AutoEncryption()
				aeOpts.SetKeyVaultNamespace("keyvault.datakeys").
					SetKmsProviders(kmsProviders).
					SetBypassAutoEncryption(tc.bypassAutoEncryption)

				// Only set the crypt_shared library extra options if bypassAutoEncryption isn't
				// true because it's invalid to set cryptSharedLibRequired=true and
				// bypassAutoEncryption=true together.
				if !tc.bypassAutoEncryption {
					aeOpts.SetExtraOptions(getCryptSharedLibExtraOptions())
				}

				if tc.keyVaultClientSet {
					testutil.AddTestServerAPIVersion(d.clientKeyVaultOpts)
					aeOpts.SetKeyVaultClientOptions(d.clientKeyVaultOpts)
				}

				ceOpts := options.Client().ApplyURI(mtest.ClusterURI()).
					SetMonitor(&event.CommandMonitor{
						Started: func(ctx context.Context, event *event.CommandStartedEvent) {
							clientEncryptedEvents = append(clientEncryptedEvents, startedEvent{event.CommandName, event.DatabaseName})
						},
					}).
					SetServerMonitor(&event.ServerMonitor{
						TopologyOpening: func(event *event.TopologyOpeningEvent) {
							clientEncryptedTopologyOpening++
						},
					}).
					SetMaxPoolSize(tc.maxPoolSize).
					SetAutoEncryptionOptions(aeOpts)

				testutil.AddTestServerAPIVersion(ceOpts)
				clientEncrypted, err := mongo.Connect(context.Background(), ceOpts)
				assert.Nil(mt, err, "Connect error: %v", err)
				defer clientEncrypted.Disconnect(context.Background())

				coll := clientEncrypted.Database("db").Collection("coll")
				if !tc.bypassAutoEncryption {
					_, err = coll.InsertOne(context.Background(), bson.M{"_id": 0, "encrypted": "string0"})
				} else {
					unencryptedColl := d.clientTest.Database("db").Collection("coll")
					_, err = unencryptedColl.InsertOne(context.Background(), bson.M{"_id": 0, "encrypted": d.ciphertext})
				}
				assert.Nil(mt, err, "InsertOne error: %v", err)

				raw, err := coll.FindOne(context.Background(), bson.M{"_id": 0}).DecodeBytes()
				assert.Nil(mt, err, "FindOne error: %v", err)

				expected := bsoncore.NewDocumentBuilder().
					AppendInt32("_id", 0).
					AppendString("encrypted", "string0").
					Build()
				assert.Equal(mt, bson.Raw(expected), raw, "returned value unequal, expected: %v, got: %v", expected, raw)

				assert.Equal(mt, clientEncryptedEvents, tc.clientEncryptedCommandStartedExpected, "mismatched events for clientEncrypted. Expected %v, got %v", clientEncryptedEvents, tc.clientEncryptedCommandStartedExpected)
				assert.Equal(mt, d.clientKeyVaultEvents, tc.clientKeyVaultCommandStartedExpected, "mismatched events for clientKeyVault. Expected %v, got %v", d.clientKeyVaultEvents, tc.clientKeyVaultCommandStartedExpected)
				assert.Equal(mt, clientEncryptedTopologyOpening, tc.clientEncryptedTopologyOpeningExpected, "wrong number of TopologyOpening events. Expected %v, got %v", tc.clientEncryptedTopologyOpeningExpected, clientEncryptedTopologyOpening)
			})
		}
	})

	// These tests only run when a KMS mock server is running on localhost:8000.
	mt.RunOpts("10. kms tls tests", noClientOpts, func(mt *mtest.T) {
		kmsTlsTestcase := os.Getenv("KMS_TLS_TESTCASE")
		if kmsTlsTestcase == "" {
			mt.Skipf("Skipping test as KMS_TLS_TESTCASE is not set")
		}

		testcases := []struct {
			name       string
			envValue   string
			errMessage string
		}{
			{
				"invalid certificate",
				"INVALID_CERT",
				"expired",
			},
			{
				"invalid hostname",
				"INVALID_HOSTNAME",
				"SANs",
			},
		}

		for _, tc := range testcases {
			mt.Run(tc.name, func(mt *mtest.T) {
				// Only run test if correct KMS mock server is running.
				if kmsTlsTestcase != tc.envValue {
					mt.Skipf("Skipping test as KMS_TLS_TESTCASE is set to %q, expected %v", kmsTlsTestcase, tc.envValue)
				}

				ceo := options.ClientEncryption().
					SetKmsProviders(fullKmsProvidersMap).
					SetKeyVaultNamespace(kvNamespace)
				cpt := setup(mt, nil, nil, ceo)
				defer cpt.teardown(mt)

				_, err := cpt.clientEnc.CreateDataKey(context.Background(), "aws", options.DataKey().SetMasterKey(
					bson.D{
						{"region", "us-east-1"},
						{"key", "arn:aws:kms:us-east-1:579766882180:key/89fcc2c4-08b0-4bd9-9f25-e30687b580d0"},
						{"endpoint", "127.0.0.1:8000"},
					},
				))
				assert.NotNil(mt, err, "expected CreateDataKey error, got nil")
				assert.True(mt, strings.Contains(err.Error(), tc.errMessage),
					"expected CreateDataKey error to contain %v, got %v", tc.errMessage, err.Error())
			})
		}
	})

	// These tests only run when 3 KMS HTTP servers and 1 KMS KMIP server are running. See specification for port numbers and necessary arguments:
	// https://github.com/mongodb/specifications/blob/master/source/client-side-encryption/tests/README.rst#kms-tls-options-tests
	mt.RunOpts("11. kms tls options tests", noClientOpts, func(mt *mtest.T) {
		if os.Getenv("KMS_MOCK_SERVERS_RUNNING") == "" {
			mt.Skipf("Skipping test as KMS_MOCK_SERVERS_RUNNING is not set")
		}
		validKmsProviders := map[string]map[string]interface{}{
			"aws": {
				"accessKeyId":     awsAccessKeyID,
				"secretAccessKey": awsSecretAccessKey,
			},
			"azure": {
				"tenantId":                 azureTenantID,
				"clientId":                 azureClientID,
				"clientSecret":             azureClientSecret,
				"identityPlatformEndpoint": "127.0.0.1:8002",
			},
			"gcp": {
				"email":      gcpEmail,
				"privateKey": gcpPrivateKey,
				"endpoint":   "127.0.0.1:8002",
			},
			"kmip": {
				"endpoint": "127.0.0.1:5698",
			},
		}

		expiredKmsProviders := map[string]map[string]interface{}{
			"aws": {
				"accessKeyId":     awsAccessKeyID,
				"secretAccessKey": awsSecretAccessKey,
			},
			"azure": {
				"tenantId":                 azureTenantID,
				"clientId":                 azureClientID,
				"clientSecret":             azureClientSecret,
				"identityPlatformEndpoint": "127.0.0.1:8000",
			},
			"gcp": {
				"email":      gcpEmail,
				"privateKey": gcpPrivateKey,
				"endpoint":   "127.0.0.1:8000",
			},
			"kmip": {
				"endpoint": "127.0.0.1:8000",
			},
		}

		invalidKmsProviders := map[string]map[string]interface{}{
			"aws": {
				"accessKeyId":     awsAccessKeyID,
				"secretAccessKey": awsSecretAccessKey,
			},
			"azure": {
				"tenantId":                 azureTenantID,
				"clientId":                 azureClientID,
				"clientSecret":             azureClientSecret,
				"identityPlatformEndpoint": "127.0.0.1:8001",
			},
			"gcp": {
				"email":      gcpEmail,
				"privateKey": gcpPrivateKey,
				"endpoint":   "127.0.0.1:8001",
			},
			"kmip": {
				"endpoint": "127.0.0.1:8001",
			},
		}

		// create valid Client Encryption options without a client certificate
		validClientEncryptionOptionsWithoutClientCert := options.ClientEncryption().
			SetKmsProviders(validKmsProviders).
			SetKeyVaultNamespace(kvNamespace)

		// make TLS opts containing client certificate and CA file
		tlsConfig := make(map[string]*tls.Config)
		if tlsCAFileKMIP != "" && tlsClientCertificateKeyFileKMIP != "" {
			clientAndCATlsMap := map[string]interface{}{
				"tlsCertificateKeyFile": tlsClientCertificateKeyFileKMIP,
				"tlsCAFile":             tlsCAFileKMIP,
			}
			certConfig, err := options.BuildTLSConfig(clientAndCATlsMap)
			assert.Nil(mt, err, "BuildTLSConfig error: %v", err)
			tlsConfig["aws"] = certConfig
			tlsConfig["azure"] = certConfig
			tlsConfig["gcp"] = certConfig
			tlsConfig["kmip"] = certConfig
		}

		// create valid Client Encryption options and set valid TLS options
		validClientEncryptionOptionsWithTLS := options.ClientEncryption().
			SetKmsProviders(validKmsProviders).
			SetKeyVaultNamespace(kvNamespace).
			SetTLSConfig(tlsConfig)

		// make TLS opts containing only CA file
		if tlsCAFileKMIP != "" {
			caTlsMap := map[string]interface{}{
				"tlsCAFile": tlsCAFileKMIP,
			}
			certConfig, err := options.BuildTLSConfig(caTlsMap)
			assert.Nil(mt, err, "BuildTLSConfig error: %v", err)
			tlsConfig["aws"] = certConfig
			tlsConfig["azure"] = certConfig
			tlsConfig["gcp"] = certConfig
			tlsConfig["kmip"] = certConfig
		}

		// create invalid Client Encryption options with expired credentials
		expiredClientEncryptionOptions := options.ClientEncryption().
			SetKmsProviders(expiredKmsProviders).
			SetKeyVaultNamespace(kvNamespace).
			SetTLSConfig(tlsConfig)

		// create invalid Client Encryption options with invalid hostnames
		invalidHostnameClientEncryptionOptions := options.ClientEncryption().
			SetKmsProviders(invalidKmsProviders).
			SetKeyVaultNamespace(kvNamespace).
			SetTLSConfig(tlsConfig)

		awsMasterKeyNoClientCert := map[string]interface{}{
			"region":   "us-east-1",
			"key":      "arn:aws:kms:us-east-1:579766882180:key/89fcc2c4-08b0-4bd9-9f25-e30687b580d0",
			"endpoint": "127.0.0.1:8002",
		}
		awsMasterKeyWithTLS := map[string]interface{}{
			"region":   "us-east-1",
			"key":      "arn:aws:kms:us-east-1:579766882180:key/89fcc2c4-08b0-4bd9-9f25-e30687b580d0",
			"endpoint": "127.0.0.1:8002",
		}
		awsMasterKeyExpired := map[string]interface{}{
			"region":   "us-east-1",
			"key":      "arn:aws:kms:us-east-1:579766882180:key/89fcc2c4-08b0-4bd9-9f25-e30687b580d0",
			"endpoint": "127.0.0.1:8000",
		}
		awsMasterKeyInvalidHostname := map[string]interface{}{
			"region":   "us-east-1",
			"key":      "arn:aws:kms:us-east-1:579766882180:key/89fcc2c4-08b0-4bd9-9f25-e30687b580d0",
			"endpoint": "127.0.0.1:8001",
		}
		azureMasterKey := map[string]interface{}{
			"keyVaultEndpoint": "doesnotexist.local",
			"keyName":          "foo",
		}
		gcpMasterKey := map[string]interface{}{
			"projectId": "foo",
			"location":  "bar",
			"keyRing":   "baz",
			"keyName":   "foo",
		}
		kmipMasterKey := map[string]interface{}{}

		testCases := []struct {
			name                     string
			masterKeyNoClientCert    interface{}
			masterKeyWithTLS         interface{}
			masterKeyExpired         interface{}
			masterKeyInvalidHostname interface{}
			tlsError                 string
			expiredError             string
			invalidHostnameError     string
		}{
			{"aws", awsMasterKeyNoClientCert, awsMasterKeyWithTLS, awsMasterKeyExpired, awsMasterKeyInvalidHostname, "parse error", "certificate has expired", "cannot validate certificate"},
			{"azure", azureMasterKey, azureMasterKey, azureMasterKey, azureMasterKey, "HTTP status=404", "certificate has expired", "cannot validate certificate"},
			{"gcp", gcpMasterKey, gcpMasterKey, gcpMasterKey, gcpMasterKey, "HTTP status=404", "certificate has expired", "cannot validate certificate"},
			{"kmip", kmipMasterKey, kmipMasterKey, kmipMasterKey, kmipMasterKey, "", "certificate has expired", "cannot validate certificate"},
		}

		for _, tc := range testCases {
			mt.Run(tc.name, func(mt *mtest.T) {
				if tc.name == "kmip" && "" == os.Getenv("KMS_MOCK_SERVERS_RUNNING") {
					mt.Skipf("Skipping test as KMS_MOCK_SERVERS_RUNNING is not set")
				}
				// call CreateDataKey with CEO no TLS with each provider and corresponding master key
				cpt := setup(mt, nil, defaultKvClientOptions, validClientEncryptionOptionsWithoutClientCert)
				defer cpt.teardown(mt)

				dkOpts := options.DataKey().SetMasterKey(tc.masterKeyNoClientCert)
				_, err := cpt.clientEnc.CreateDataKey(context.Background(), tc.name, dkOpts)

				assert.NotNil(mt, err, "expected error, got nil")
				assert.True(mt, strings.Contains(err.Error(), "certificate signed by unknown authority"),
					"expected error '%s' to contain '%s'", err.Error(), "certificate signed by unknown authority")

				// call CreateDataKey with CEO & TLS with each provider and corresponding master key
				cpt = setup(mt, nil, defaultKvClientOptions, validClientEncryptionOptionsWithTLS)

				dkOpts = options.DataKey().SetMasterKey(tc.masterKeyWithTLS)
				_, err = cpt.clientEnc.CreateDataKey(context.Background(), tc.name, dkOpts)
				// check if current test case is KMIP, which should pass
				if tc.name == "kmip" {
					assert.Nil(mt, err, "expected no error, got err: %v", err)
				} else {
					assert.NotNil(mt, err, "expected error, got nil")
					assert.True(mt, strings.Contains(err.Error(), tc.tlsError),
						"expected error '%s' to contain '%s'", err.Error(), tc.tlsError)
				}

				// call CreateDataKey with expired CEO each provider and same masterKey
				cpt = setup(mt, nil, defaultKvClientOptions, expiredClientEncryptionOptions)

				dkOpts = options.DataKey().SetMasterKey(tc.masterKeyExpired)
				_, err = cpt.clientEnc.CreateDataKey(context.Background(), tc.name, dkOpts)
				assert.NotNil(mt, err, "expected error, got nil")
				assert.True(mt, strings.Contains(err.Error(), tc.expiredError),
					"expected error '%s' to contain '%s'", err.Error(), tc.expiredError)

				// call CreateDataKey with invalid hostname CEO with each provider and same masterKey
				cpt = setup(mt, nil, defaultKvClientOptions, invalidHostnameClientEncryptionOptions)

				dkOpts = options.DataKey().SetMasterKey(tc.masterKeyInvalidHostname)
				_, err = cpt.clientEnc.CreateDataKey(context.Background(), tc.name, dkOpts)
				assert.NotNil(mt, err, "expected error, got nil")
				assert.True(mt, strings.Contains(err.Error(), tc.invalidHostnameError),
					"expected error '%s' to contain '%s'", err.Error(), tc.invalidHostnameError)
			})
		}
	})
	mt.RunOpts("12. explicit encryption", runOpts, func(mt *mtest.T) {
		// Test Setup ... begin
		encryptedFields := readJSONFile(mt, "encrypted-fields.json")
		key1Document := readJSONFile(mt, "key1-document.json")
		var key1ID primitive.Binary
		{
			subtype, data := key1Document.Lookup("_id").Binary()
			key1ID = primitive.Binary{Subtype: subtype, Data: data}
		}

		testSetup := func() (*mongo.Client, *mongo.ClientEncryption) {
			mtest.DropEncryptedCollection(mt, mt.Client.Database("db").Collection("explicit_encryption"), encryptedFields)
			cco := options.CreateCollection().SetEncryptedFields(encryptedFields)
			err := mt.Client.Database("db").CreateCollection(context.Background(), "explicit_encryption", cco)
			assert.Nil(mt, err, "error on CreateCollection: %v", err)
			err = mt.Client.Database("keyvault").Collection("datakeys").Drop(context.Background())
			assert.Nil(mt, err, "error on Drop: %v", err)
			keyVaultClient, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(mtest.ClusterURI()))
			assert.Nil(mt, err, "error on Connect: %v", err)
			datakeysColl := keyVaultClient.Database("keyvault").Collection("datakeys", options.Collection().SetWriteConcern(mtest.MajorityWc))
			_, err = datakeysColl.InsertOne(context.TODO(), key1Document)
			assert.Nil(mt, err, "error on InsertOne: %v", err)
			// Create a ClientEncryption.
			ceo := options.ClientEncryption().
				SetKeyVaultNamespace("keyvault.datakeys").
				SetKmsProviders(fullKmsProvidersMap)
			clientEncryption, err := mongo.NewClientEncryption(keyVaultClient, ceo)
			assert.Nil(mt, err, "error on NewClientEncryption: %v", err)

			// Create a MongoClient with AutoEncryptionOpts and bypassQueryAnalysis=true.
			aeo := options.AutoEncryption().
				SetKeyVaultNamespace("keyvault.datakeys").
				SetKmsProviders(fullKmsProvidersMap).
				SetBypassQueryAnalysis(true)
			co := options.Client().SetAutoEncryptionOptions(aeo).ApplyURI(mtest.ClusterURI())
			encryptedClient, err := mongo.Connect(context.Background(), co)
			assert.Nil(mt, err, "error on Connect: %v", err)
			return encryptedClient, clientEncryption
		}
		// Test Setup ... end

		mt.Run("case 1: can insert encrypted indexed and find", func(mt *mtest.T) {
			encryptedClient, clientEncryption := testSetup()
			defer clientEncryption.Close(context.Background())
			defer encryptedClient.Disconnect(context.Background())

			// Explicit encrypt the value "encrypted indexed value" with algorithm: "Indexed".
			eo := options.Encrypt().SetAlgorithm("Indexed").SetKeyID(key1ID).SetContentionFactor(0)
			valueToEncrypt := "encrypted indexed value"
			rawVal := bson.RawValue{Type: bson.TypeString, Value: bsoncore.AppendString(nil, valueToEncrypt)}
			insertPayload, err := clientEncryption.Encrypt(context.Background(), rawVal, eo)
			assert.Nil(mt, err, "error in Encrypt: %v", err)
			// Insert.
			coll := encryptedClient.Database("db").Collection("explicit_encryption")
			_, err = coll.InsertOne(context.Background(), bson.D{{"_id", 1}, {"encryptedIndexed", insertPayload}})
			assert.Nil(mt, err, "Error in InsertOne: %v", err)
			// Explicit encrypt an indexed value to find.
			eo = options.Encrypt().SetAlgorithm("Indexed").SetKeyID(key1ID).SetQueryType(options.QueryTypeEquality).SetContentionFactor(0)
			findPayload, err := clientEncryption.Encrypt(context.Background(), rawVal, eo)
			assert.Nil(mt, err, "error in Encrypt: %v", err)
			// Find.
			res := coll.FindOne(context.Background(), bson.D{{"encryptedIndexed", findPayload}})
			assert.Nil(mt, res.Err(), "Error in FindOne: %v", res.Err())
			got, err := res.DecodeBytes()
			assert.Nil(mt, err, "error in DecodeBytes: %v", err)
			gotValue, err := got.LookupErr("encryptedIndexed")
			assert.Nil(mt, err, "error in LookupErr: %v", err)
			assert.Equal(mt, gotValue.StringValue(), valueToEncrypt, "expected %q, got %q", valueToEncrypt, gotValue.StringValue())
		})
		mt.Run("case 2: can insert encrypted indexed and find with non-zero contention", func(mt *mtest.T) {
			encryptedClient, clientEncryption := testSetup()
			defer clientEncryption.Close(context.Background())
			defer encryptedClient.Disconnect(context.Background())

			coll := encryptedClient.Database("db").Collection("explicit_encryption")
			valueToEncrypt := "encrypted indexed value"
			rawVal := bson.RawValue{Type: bson.TypeString, Value: bsoncore.AppendString(nil, valueToEncrypt)}

			for i := 0; i < 10; i++ {
				// Explicit encrypt the value "encrypted indexed value" with algorithm: "Indexed".
				eo := options.Encrypt().SetAlgorithm("Indexed").SetKeyID(key1ID).SetContentionFactor(10)
				insertPayload, err := clientEncryption.Encrypt(context.Background(), rawVal, eo)
				assert.Nil(mt, err, "error in Encrypt: %v", err)
				// Insert.
				_, err = coll.InsertOne(context.Background(), bson.D{{"_id", i}, {"encryptedIndexed", insertPayload}})
				assert.Nil(mt, err, "Error in InsertOne: %v", err)
			}

			// Explicit encrypt an indexed value to find with default contentionFactor 0.
			{
				eo := options.Encrypt().SetAlgorithm("Indexed").SetKeyID(key1ID).SetQueryType(options.QueryTypeEquality).SetContentionFactor(0)
				findPayload, err := clientEncryption.Encrypt(context.Background(), rawVal, eo)
				assert.Nil(mt, err, "error in Encrypt: %v", err)
				// Find with contentionFactor=0.
				cursor, err := coll.Find(context.Background(), bson.D{{"encryptedIndexed", findPayload}})
				assert.Nil(mt, err, "error in Find: %v", err)
				var got []bson.Raw
				err = cursor.All(context.Background(), &got)
				assert.Nil(mt, err, "error in All: %v", err)
				assert.True(mt, len(got) < 10, "expected len(got) < 10, got: %v", len(got))
				for _, doc := range got {
					gotValue, err := doc.LookupErr("encryptedIndexed")
					assert.Nil(mt, err, "error in LookupErr: %v", err)
					assert.Equal(mt, gotValue.StringValue(), valueToEncrypt, "expected %q, got %q", valueToEncrypt, gotValue.StringValue())
				}
			}

			// Explicit encrypt an indexed value to find with contentionFactor 10.
			{
				eo := options.Encrypt().SetAlgorithm("Indexed").SetKeyID(key1ID).SetQueryType(options.QueryTypeEquality).SetContentionFactor(10)
				findPayload2, err := clientEncryption.Encrypt(context.Background(), rawVal, eo)
				assert.Nil(mt, err, "error in Encrypt: %v", err)
				// Find with contentionFactor=10.
				cursor, err := coll.Find(context.Background(), bson.D{{"encryptedIndexed", findPayload2}})
				assert.Nil(mt, err, "error in Find: %v", err)
				var got []bson.Raw
				err = cursor.All(context.Background(), &got)
				assert.Nil(mt, err, "error in All: %v", err)
				assert.True(mt, len(got) == 10, "expected len(got) == 10, got: %v", len(got))
				for _, doc := range got {
					gotValue, err := doc.LookupErr("encryptedIndexed")
					assert.Nil(mt, err, "error in LookupErr: %v", err)
					assert.Equal(mt, gotValue.StringValue(), valueToEncrypt, "expected %q, got %q", valueToEncrypt, gotValue.StringValue())
				}
			}
		})
		mt.Run("case 3: can insert encrypted unindexed", func(mt *mtest.T) {
			encryptedClient, clientEncryption := testSetup()
			defer clientEncryption.Close(context.Background())
			defer encryptedClient.Disconnect(context.Background())

			// Explicit encrypt the value "encrypted indexed value" with algorithm: "Indexed".
			eo := options.Encrypt().SetAlgorithm("Unindexed").SetKeyID(key1ID)
			valueToEncrypt := "encrypted unindexed value"
			rawVal := bson.RawValue{Type: bson.TypeString, Value: bsoncore.AppendString(nil, valueToEncrypt)}
			insertPayload, err := clientEncryption.Encrypt(context.Background(), rawVal, eo)
			assert.Nil(mt, err, "error in Encrypt: %v", err)
			// Insert.
			coll := encryptedClient.Database("db").Collection("explicit_encryption")
			_, err = coll.InsertOne(context.Background(), bson.D{{"_id", 1}, {"encryptedUnindexed", insertPayload}})
			assert.Nil(mt, err, "Error in InsertOne: %v", err)
			// Find.
			res := coll.FindOne(context.Background(), bson.D{{"_id", 1}})
			assert.Nil(mt, res.Err(), "Error in FindOne: %v", res.Err())
			got, err := res.DecodeBytes()
			assert.Nil(mt, err, "error in DecodeBytes: %v", err)
			gotValue, err := got.LookupErr("encryptedUnindexed")
			assert.Nil(mt, err, "error in LookupErr: %v", err)
			assert.Equal(mt, gotValue.StringValue(), valueToEncrypt, "expected %q, got %q", valueToEncrypt, gotValue.StringValue())
		})
		mt.Run("case 4: can roundtrip encrypted indexed", func(mt *mtest.T) {
			encryptedClient, clientEncryption := testSetup()
			defer clientEncryption.Close(context.Background())
			defer encryptedClient.Disconnect(context.Background())

			// Explicit encrypt the value "encrypted indexed value" with algorithm: "Indexed".
			eo := options.Encrypt().SetAlgorithm("Indexed").SetKeyID(key1ID).SetContentionFactor(0)
			valueToEncrypt := "encrypted indexed value"
			rawVal := bson.RawValue{Type: bson.TypeString, Value: bsoncore.AppendString(nil, valueToEncrypt)}
			payload, err := clientEncryption.Encrypt(context.Background(), rawVal, eo)
			assert.Nil(mt, err, "error in Encrypt: %v", err)
			gotValue, err := clientEncryption.Decrypt(context.Background(), payload)
			assert.Nil(mt, err, "error in Decrypt: %v", err)
			assert.Equal(mt, gotValue.StringValue(), valueToEncrypt, "expected %q, got %q", valueToEncrypt, gotValue.StringValue())
		})
		mt.Run("case 5: can roundtrip encrypted unindexed", func(mt *mtest.T) {
			encryptedClient, clientEncryption := testSetup()
			defer clientEncryption.Close(context.Background())
			defer encryptedClient.Disconnect(context.Background())

			// Explicit encrypt the value "encrypted indexed value" with algorithm: "Indexed".
			eo := options.Encrypt().SetAlgorithm("Unindexed").SetKeyID(key1ID)
			valueToEncrypt := "encrypted unindexed value"
			rawVal := bson.RawValue{Type: bson.TypeString, Value: bsoncore.AppendString(nil, valueToEncrypt)}
			payload, err := clientEncryption.Encrypt(context.Background(), rawVal, eo)
			assert.Nil(mt, err, "error in Encrypt: %v", err)
			gotValue, err := clientEncryption.Decrypt(context.Background(), payload)
			assert.Nil(mt, err, "error in Decrypt: %v", err)
			assert.Equal(mt, gotValue.StringValue(), valueToEncrypt, "expected %q, got %q", valueToEncrypt, gotValue.StringValue())
		})
	})

	runOpts = mtest.NewOptions().MinServerVersion("4.2").
		Topologies(mtest.ReplicaSet, mtest.LoadBalanced, mtest.ShardedReplicaSet)
	mt.RunOpts("13. Unique Index on keyAltNames", runOpts, func(mt *mtest.T) {
		const (
			dkCollection  = "datakeys"
			idKey         = "_id"
			kvDatabase    = "keyvault"
			defKeyAltName = "def"
			abcKeyAltName = "abc"
		)

		var cse *cseProseTest

		var initialize = func() primitive.Binary {
			// Create a ClientEncryption object (referred to as client_encryption) with client set as the keyVaultClient.
			// Using client, drop the collection keyvault.datakeys.
			cse = setup(mt, nil, defaultKvClientOptions, options.ClientEncryption().
				SetKmsProviders(fullKmsProvidersMap).
				SetKeyVaultNamespace(kvNamespace))

			err := cse.kvClient.Database(kvDatabase).Collection(dkCollection).Drop(context.Background())
			assert.Nil(mt, err, "error dropping %q namespace: %v", kvNamespace, err)

			// Using client, create a unique index on keyAltNames with a partial index filter for only documents where
			// keyAltNames exists using writeConcern "majority".
			keyVaultIndex := mongo.IndexModel{
				Keys: bson.D{{"keyAltNames", 1}},
				Options: options.Index().
					SetUnique(true).
					SetName("keyAltNames_1").
					SetPartialFilterExpression(bson.D{
						{"keyAltNames", bson.D{
							{"$exists", true},
						}},
					}),
			}

			wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(1*time.Second))
			wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)
			wcmColl := cse.kvClient.Database(kvDatabase).Collection(dkCollection, wcMajorityCollectionOpts)
			_, err = wcmColl.Indexes().CreateOne(context.TODO(), keyVaultIndex)
			assert.Nil(mt, err, "error creating keyAltNames index: %v", err)

			// Using client_encryption, create a data key with a local KMS provider and the keyAltName "def".
			opts := options.DataKey().SetKeyAltNames([]string{defKeyAltName})
			defKeyID, err := cse.clientEnc.CreateDataKey(context.Background(), "local", opts)
			assert.Nil(mt, err, "error creating %q data key: %v", defKeyAltName, err)
			return defKeyID
		}

		var validateAddKeyAltName = func(mt *mtest.T, cse *cseProseTest, res *mongo.SingleResult, expected ...string) {
			assert.Nil(mt, res.Err(), "error adding key alt name: %v", res.Err())

			resbytes, err := res.DecodeBytes()
			assert.Nil(mt, err, "error decoding result bytes: %v", err)

			idsubtype, iddata := bson.RawValue{Type: bsontype.EmbeddedDocument, Value: resbytes}.
				Document().Lookup("_id").Binary()
			filter := bsoncore.NewDocumentBuilder().AppendBinary("_id", idsubtype, iddata).Build()

			ctx := context.Background()
			updatedData, err := cse.keyVaultColl.FindOne(ctx, filter).DecodeBytes()
			assert.Nil(mt, err, "error decoding result bytes: %v", err)

			updated := bson.RawValue{Type: bsontype.EmbeddedDocument, Value: updatedData}
			updatedKeyAltNames, err := updated.Document().Lookup("keyAltNames").Array().Values()
			assert.Nil(mt, err, "error looking up raw keyAltNames: %v", err)
			assert.Equal(mt, len(updatedKeyAltNames), len(expected), "expected raw keyAltNames length to be 1")

			for idx, keyAltName := range updatedKeyAltNames {
				str := keyAltName.StringValue()
				assert.Equal(mt, str, expected[idx], "expected keyAltName to be %q, got: %q", expected[idx], str)
			}
		}

		mt.Run("case 1: createKey()", func(mt *mtest.T) {
			initialize()

			// Use client_encryption to create a new local data key with a keyAltName "abc" and assert the operation
			// does not fail.
			opts := options.DataKey().SetKeyAltNames([]string{abcKeyAltName})
			_, err := cse.clientEnc.CreateDataKey(context.Background(), "local", opts)
			assert.Nil(mt, err, "error creating %q data key: %v", abcKeyAltName, err)

			// Repeat Step 1 and assert the operation fails due to a duplicate key server error (error code 11000).
			opts = options.DataKey().SetKeyAltNames([]string{abcKeyAltName})
			_, err = cse.clientEnc.CreateDataKey(context.Background(), "local", opts)
			assert.NotNil(mt, err, "duplicate %q key did not propagate expected error", abcKeyAltName)

			e110000 := "E11000 duplicate key"
			correctError := strings.Contains(err.Error(), e110000)
			assert.True(t, correctError, "expected error to contain %q, got: %v", e110000, err)

			// Use client_encryption to create a new local data key with a keyAltName "def" and assert the operation
			// fails due to a duplicate key server error (error code 11000).
			opts = options.DataKey().SetKeyAltNames([]string{defKeyAltName})
			_, err = cse.clientEnc.CreateDataKey(context.Background(), "local", opts)
			assert.NotNil(mt, err, "duplicate %q key did not propagate expected error", defKeyAltName)

			e110000 = "E11000 duplicate key"
			correctError = strings.Contains(err.Error(), e110000)
			assert.True(t, correctError, "expected error to contain %q, got: %v", e110000, err)
		})

		mt.Run("case 2: addKeyAltName()", func(t *mtest.T) {
			defKeyID := initialize()

			var someNewKeyID primitive.Binary
			// Use client_encryption to create a new local data key and assert the operation does not fail.
			var err error
			someNewKeyID, err = cse.clientEnc.CreateDataKey(context.Background(), "local")
			assert.Nil(mt, err, "error creating data key: %v", err)

			// Use client_encryption to add a keyAltName "abc" to the key created in Step 1 and assert the operation
			// does not fail.
			res := cse.clientEnc.AddKeyAltName(context.Background(), someNewKeyID, abcKeyAltName)
			validateAddKeyAltName(mt, cse, res, abcKeyAltName)

			// Repeat Step 2, assert the operation does not fail, and assert the returned key document contains the
			// keyAltName "abc" added in Step 2.
			res = cse.clientEnc.AddKeyAltName(context.Background(), someNewKeyID, abcKeyAltName)
			validateAddKeyAltName(mt, cse, res, abcKeyAltName)

			// Use client_encryption to add a keyAltName "def" to the key created in Step 1 and assert the operation
			// fails due to a duplicate key server error (error code 11000).
			res = cse.clientEnc.AddKeyAltName(context.Background(), someNewKeyID, defKeyAltName)
			assert.NotNil(mt, res.Err(), "duplicate %q key did not propagate expected error", defKeyAltName)

			e110000 := "E11000 duplicate key"
			correctError := strings.Contains(res.Err().Error(), e110000)
			assert.True(t, correctError, "expected error to contain %q, got: %v", e110000, res.Err())

			// Use client_encryption to add a keyAltName "def" to the existing key, assert the operation does not fail,
			// and assert the returned key document contains the keyAltName "def" added during Setup.
			res = cse.clientEnc.AddKeyAltName(context.Background(), defKeyID, defKeyAltName)
			validateAddKeyAltName(mt, cse, res, defKeyAltName)
		})

	})
}

func getWatcher(mt *mtest.T, streamType mongo.StreamType, cpt *cseProseTest) watcher {
	mt.Helper()

	switch streamType {
	case mongo.ClientStream:
		return cpt.cseClient
	case mongo.DatabaseStream:
		return cpt.cseColl.Database()
	case mongo.CollectionStream:
		return cpt.cseColl
	default:
		mt.Fatalf("unknown stream type %v", streamType)
	}
	return nil
}

type cseProseTest struct {
	coll         *mongo.Collection // collection db.coll
	kvClient     *mongo.Client
	keyVaultColl *mongo.Collection
	cseClient    *mongo.Client     // encrypted client
	cseColl      *mongo.Collection // db.coll with encrypted client
	clientEnc    *mongo.ClientEncryption
	cseStarted   []*event.CommandStartedEvent
}

func setup(mt *mtest.T, aeo *options.AutoEncryptionOptions, kvClientOpts *options.ClientOptions,
	ceo *options.ClientEncryptionOptions) *cseProseTest {
	mt.Helper()
	var cpt cseProseTest
	var err error
	cpt.coll = mt.CreateCollection(mtest.Collection{
		Name: "coll",
		DB:   "db",
		Opts: options.Collection().SetWriteConcern(mtest.MajorityWc),
	}, false)
	cpt.keyVaultColl = mt.CreateCollection(mtest.Collection{
		Name: "datakeys",
		DB:   "keyvault",
		Opts: options.Collection().SetWriteConcern(mtest.MajorityWc),
	}, false)

	if aeo != nil {
		cseMonitor := &event.CommandMonitor{
			Started: func(_ context.Context, evt *event.CommandStartedEvent) {
				cpt.cseStarted = append(cpt.cseStarted, evt)
			},
		}
		opts := options.Client().ApplyURI(mtest.ClusterURI()).SetWriteConcern(mtest.MajorityWc).
			SetReadPreference(mtest.PrimaryRp).SetAutoEncryptionOptions(aeo).SetMonitor(cseMonitor)
		testutil.AddTestServerAPIVersion(opts)
		cpt.cseClient, err = mongo.Connect(context.Background(), opts)
		assert.Nil(mt, err, "Connect error for encrypted client: %v", err)
		cpt.cseColl = cpt.cseClient.Database("db").Collection("coll")
	}
	if ceo != nil {
		cpt.kvClient, err = mongo.Connect(context.Background(), kvClientOpts)
		assert.Nil(mt, err, "Connect error for ClientEncryption key vault client: %v", err)
		cpt.clientEnc, err = mongo.NewClientEncryption(cpt.kvClient, ceo)
		assert.Nil(mt, err, "NewClientEncryption error: %v", err)
	}
	return &cpt
}

func (cpt *cseProseTest) teardown(mt *mtest.T) {
	mt.Helper()

	if cpt.cseClient != nil {
		_ = cpt.cseClient.Disconnect(context.Background())
	}
	if cpt.clientEnc != nil {
		_ = cpt.clientEnc.Close(context.Background())
	}
}

func readJSONFile(mt *mtest.T, file string) bson.Raw {
	mt.Helper()

	content, err := ioutil.ReadFile(filepath.Join(clientEncryptionProseDir, file))
	assert.Nil(mt, err, "ReadFile error for %v: %v", file, err)

	var doc bson.Raw
	err = bson.UnmarshalExtJSON(content, true, &doc)
	assert.Nil(mt, err, "UnmarshalExtJSON error for file %v: %v", file, err)
	return doc
}

func decodeJSONFile(mt *mtest.T, file string, val interface{}) bson.Raw {
	mt.Helper()

	content, err := ioutil.ReadFile(filepath.Join(clientEncryptionProseDir, file))
	assert.Nil(mt, err, "ReadFile error for %v: %v", file, err)

	var doc bson.Raw
	err = bson.UnmarshalExtJSON(content, true, val)
	assert.Nil(mt, err, "UnmarshalExtJSON error for file %v: %v", file, err)
	return doc
}

func rawValueToCoreValue(rv bson.RawValue) bsoncore.Value {
	return bsoncore.Value{Type: rv.Type, Data: rv.Value}
}

type deadlockTest struct {
	clientTest           *mongo.Client
	clientKeyVaultOpts   *options.ClientOptions
	clientKeyVaultEvents []startedEvent
	clientEncryption     *mongo.ClientEncryption
	ciphertext           primitive.Binary
}

type startedEvent struct {
	Command  string
	Database string
}

func newDeadlockTest(mt *mtest.T) *deadlockTest {
	mt.Helper()

	var d deadlockTest
	var err error

	clientTestOpts := options.Client().ApplyURI(mtest.ClusterURI()).SetWriteConcern(mtest.MajorityWc)
	testutil.AddTestServerAPIVersion(clientTestOpts)
	if d.clientTest, err = mongo.Connect(context.Background(), clientTestOpts); err != nil {
		mt.Fatalf("Connect error: %v", err)
	}

	clientKeyVaultMonitor := &event.CommandMonitor{
		Started: func(ctx context.Context, event *event.CommandStartedEvent) {
			startedEvent := startedEvent{event.CommandName, event.DatabaseName}
			d.clientKeyVaultEvents = append(d.clientKeyVaultEvents, startedEvent)
		},
	}

	d.clientKeyVaultOpts = options.Client().ApplyURI(mtest.ClusterURI()).
		SetMaxPoolSize(1).SetMonitor(clientKeyVaultMonitor)

	keyvaultColl := d.clientTest.Database("keyvault").Collection("datakeys")
	dataColl := d.clientTest.Database("db").Collection("coll")
	err = dataColl.Drop(context.Background())
	assert.Nil(mt, err, "Drop error for collection db.coll: %v", err)

	err = keyvaultColl.Drop(context.Background())
	assert.Nil(mt, err, "Drop error for key vault collection: %v", err)

	keyDoc := readJSONFile(mt, "external-key.json")
	_, err = keyvaultColl.InsertOne(context.Background(), keyDoc)
	assert.Nil(mt, err, "InsertOne error into key vault collection: %v", err)

	schemaDoc := readJSONFile(mt, "external-schema.json")
	createOpts := options.CreateCollection().SetValidator(bson.M{"$jsonSchema": schemaDoc})
	err = d.clientTest.Database("db").CreateCollection(context.Background(), "coll", createOpts)
	assert.Nil(mt, err, "CreateCollection error: %v", err)

	kmsProviders := map[string]map[string]interface{}{
		"local": {"key": localMasterKey},
	}
	ceOpts := options.ClientEncryption().SetKmsProviders(kmsProviders).SetKeyVaultNamespace("keyvault.datakeys")
	d.clientEncryption, err = mongo.NewClientEncryption(d.clientTest, ceOpts)
	assert.Nil(mt, err, "NewClientEncryption error: %v", err)

	t, value, err := bson.MarshalValue("string0")
	assert.Nil(mt, err, "MarshalValue error: %v", err)
	in := bson.RawValue{Type: t, Value: value}
	eopts := options.Encrypt().SetAlgorithm("AEAD_AES_256_CBC_HMAC_SHA_512-Deterministic").SetKeyAltName("local")
	d.ciphertext, err = d.clientEncryption.Encrypt(context.Background(), in, eopts)
	assert.Nil(mt, err, "Encrypt error: %v", err)

	return &d
}

func (d *deadlockTest) disconnect(mt *mtest.T) {
	mt.Helper()
	err := d.clientEncryption.Close(context.Background())
	assert.Nil(mt, err, "clientEncryption Close error: %v", err)
	d.clientTest.Disconnect(context.Background())
	assert.Nil(mt, err, "clientTest Disconnect error: %v", err)
}
