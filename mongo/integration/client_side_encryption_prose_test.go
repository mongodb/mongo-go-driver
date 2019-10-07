// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// +build cse

package integration

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

var (
	localMasterKey  = []byte("2x44+xduTaBBkY16Er5DuADaghvS4vwdkg8tpPp3tz6gV01A1CwbD9itQ2HFDgPWOp8eMaC1Oi766JzXZBdBdbdMurdonJ1d")
	keyID           = os.Getenv(awsAccessKeyID)
	secretAccessKey = os.Getenv(awsSecretAccessKey)
)

const (
	clientEncryptionProseDir      = "../../data/client-side-encryption-prose"
	deterministicAlgorithm        = "AEAD_AES_256_CBC_HMAC_SHA_512-Deterministic"
	randomAlgorithm               = "AEAD_AES_256_CBC_HMAC_SHA_512-Random"
	kvNamespace                   = "admin.datakeys" // default namespace for the key vault collection
	keySubtype               byte = 4                // expected subtype for data keys
	encryptedValueSubtype    byte = 6                // expected subtypes for encrypted values
	cryptMaxBsonObjSize           = 2097152          // max bytes in BSON object when auto encryption is enabled
)

func TestClientSideEncryptionProse(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().MinServerVersion("4.2").Enterprise(true).CreateClient(false))
	defer mt.Close()

	assert.NotEqual(mt, keyID, "", "%s env var not set", awsAccessKeyID)
	assert.NotEqual(mt, secretAccessKey, "", "%s env var not set", awsSecretAccessKey)
	defaultKvClientOptions := options.Client().ApplyURI(mt.ConnString())

	mt.RunOpts("data key and double encryption", noClientOpts, func(mt *mtest.T) {
		// set up options structs
		kmsProviders := map[string]map[string]interface{}{
			"aws": {
				"accessKeyId":     keyID,
				"secretAccessKey": secretAccessKey,
			},
			"local": {"key": localMasterKey},
		}
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
		aeo := options.AutoEncryption().SetKmsProviders(kmsProviders).SetKeyVaultNamespace(kvNamespace).SetSchemaMap(schemaMap)
		ceo := options.ClientEncryption().SetKmsProviders(kmsProviders).SetKeyVaultNamespace(kvNamespace)

		awsMasterKey := bson.D{
			{"region", "us-east-1"},
			{"key", "arn:aws:kms:us-east-1:579766882180:key/89fcc2c4-08b0-4bd9-9f25-e30687b580d0"},
		}
		testCases := []struct {
			name       string
			provider   string
			keyAltName string
			masterKey  interface{}
			value      string
		}{
			{"local", "local", "local_altname", nil, "hello local"},
			{"aws", "aws", "aws_altname", awsMasterKey, "hello aws"},
		}
		for _, tc := range testCases {
			mt.Run(tc.name, func(mt *mtest.T) {
				cpt := setup(mt, aeo, defaultKvClientOptions, ceo)
				defer cpt.teardown(mt)

				// create data key
				altNames := []string{tc.keyAltName}
				dataKeyOpts := options.DataKey().SetKeyAltNames(altNames)
				if tc.masterKey != nil {
					dataKeyOpts.SetMasterKey(tc.masterKey)
				}
				dataKeyID, err := cpt.clientEnc.CreateDataKey(mtest.Background, tc.provider, dataKeyOpts)
				assert.Nil(mt, err, "CreateDataKey error: %v", err)
				assert.Equal(mt, keySubtype, dataKeyID.Subtype,
					"expected data key subtype %v, got %v", keySubtype, dataKeyID.Subtype)

				// assert that the key exists in the key vault
				cursor, err := cpt.keyVaultColl.Find(mtest.Background, bson.D{{"_id", dataKeyID}})
				assert.Nil(mt, err, "key vault Find error: %v", err)
				assert.True(mt, cursor.Next(mtest.Background), "no keys found in key vault")
				provider := cursor.Current.Lookup("masterKey", "provider").StringValue()
				assert.Equal(mt, tc.provider, provider, "expected provider %v, got %v", tc.provider, provider)
				assert.False(mt, cursor.Next(mtest.Background), "unexpected document in key vault: %v", cursor.Current)

				// encrypt a value with the new key by ID
				rawVal := bson.RawValue{Type: bson.TypeString, Value: bsoncore.AppendString(nil, tc.value)}
				encrypted, err := cpt.clientEnc.Encrypt(mtest.Background, rawVal,
					options.Encrypt().SetAlgorithm(deterministicAlgorithm).SetKeyID(dataKeyID))
				assert.Nil(mt, err, "Encrypt error while encrypting value by ID: %v", err)
				assert.Equal(mt, encryptedValueSubtype, encrypted.Subtype,
					"expected encrypted value subtype %v, got %v", encryptedValueSubtype, encrypted.Subtype)

				// insert an encrypted value. the value shouldn't be encrypted again because it's not in the schema.
				_, err = cpt.cseColl.InsertOne(mtest.Background, bson.D{{"_id", tc.provider}, {"value", encrypted}})
				assert.Nil(mt, err, "InsertOne error: %v", err)

				// find the inserted document. the value should be decrypted automatically
				resBytes, err := cpt.cseColl.FindOne(mtest.Background, bson.D{{"_id", tc.provider}}).DecodeBytes()
				assert.Nil(mt, err, "Find error: %v", err)
				foundVal := resBytes.Lookup("value").StringValue()
				assert.Equal(mt, tc.value, foundVal, "expected value %v, got %v", tc.value, foundVal)

				// encrypt a value with an alternate name for the new key
				altEncrypted, err := cpt.clientEnc.Encrypt(mtest.Background, rawVal,
					options.Encrypt().SetAlgorithm(deterministicAlgorithm).SetKeyAltName(tc.keyAltName))
				assert.Nil(mt, err, "Encrypt error while encrypting value by alt key name: %v", err)
				assert.Equal(mt, encryptedValueSubtype, altEncrypted.Subtype,
					"expected encrypted value subtype %v, got %v", encryptedValueSubtype, altEncrypted.Subtype)
				assert.Equal(mt, encrypted.Data, altEncrypted.Data,
					"expected data %v, got %v", encrypted.Data, altEncrypted.Data)

				// insert an encrypted value for an auto-encrypted field
				_, err = cpt.cseColl.InsertOne(mtest.Background, bson.D{{"encrypted_placeholder", encrypted}})
				assert.NotNil(mt, err, "expected InsertOne error, got nil")
			})
		}
	})
	mt.RunOpts("external key vault", noClientOpts, func(mt *mtest.T) {
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
				aeo := options.AutoEncryption().SetKmsProviders(kmsProviders).SetKeyVaultNamespace(kvNamespace).SetSchemaMap(schemaMap)
				ceo := options.ClientEncryption().SetKmsProviders(kmsProviders).SetKeyVaultNamespace(kvNamespace)
				kvClientOpts := defaultKvClientOptions

				if tc.externalVault {
					externalKvOpts := options.Client().ApplyURI(mt.ConnString()).SetAuth(options.Credential{
						Username: "fake-user",
						Password: "fake-password",
					})
					aeo.SetKeyVaultClientOptions(externalKvOpts)
					kvClientOpts = externalKvOpts
				}
				cpt := setup(mt, aeo, kvClientOpts, ceo)
				defer cpt.teardown(mt)

				// manually insert data key
				key := readJSONFile(mt, "external-key.json")
				_, err := cpt.keyVaultColl.InsertOne(mtest.Background, key)
				assert.Nil(mt, err, "InsertOne error for data key: %v", err)
				subtype, data := key.Lookup("_id").Binary()
				dataKeyID := primitive.Binary{Subtype: subtype, Data: data}

				doc := bson.D{{"encrypted", "test"}}
				_, insertErr := cpt.cseClient.Database("db").Collection("coll").InsertOne(mtest.Background, doc)
				rawVal := bson.RawValue{Type: bson.TypeString, Value: bsoncore.AppendString(nil, "test")}
				_, encErr := cpt.clientEnc.Encrypt(mtest.Background, rawVal,
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
	mt.Run("bson size limits", func(mt *mtest.T) {
		kmsProviders := map[string]map[string]interface{}{
			"local": {
				"key": localMasterKey,
			},
		}
		aeo := options.AutoEncryption().SetKmsProviders(kmsProviders).SetKeyVaultNamespace(kvNamespace)
		cpt := setup(mt, aeo, nil, nil)
		defer cpt.teardown(mt)

		// create coll with JSON schema
		err := mt.Client.Database("db").RunCommand(mtest.Background, bson.D{
			{"create", "coll"},
			{"validator", bson.D{
				{"$jsonSchema", readJSONFile(mt, "limits-schema.json")},
			}},
		}).Err()
		assert.Nil(mt, err, "create error with validator: %v", err)

		// insert key
		key := readJSONFile(mt, "limits-key.json")
		_, err = cpt.keyVaultColl.InsertOne(mtest.Background, key)
		assert.Nil(mt, err, "InsertOne error for key: %v", err)

		var completeStr []byte
		for i := 0; i < cryptMaxBsonObjSize; i++ {
			completeStr = append(completeStr, 'a')
		}

		// insert a doc smaller than 2MiB
		str := completeStr[:cryptMaxBsonObjSize-18000] // remove last 18000 bytes
		doc := bson.D{{"_id", "no_encryption_under_2mib"}, {"unencrypted", string(str)}}
		_, err = cpt.cseColl.InsertOne(mtest.Background, doc)
		assert.Nil(mt, err, "InsertOne error for document smaller than 2MiB: %v", err)

		// insert a doc larger than 2MiB
		doc = bson.D{{"_id", "no_encryption_over_2mib"}, {"unencrypted", string(completeStr)}}
		_, err = cpt.cseColl.InsertOne(mtest.Background, doc)
		assert.NotNil(mt, err, "expected error for inserting document larger than 2MiB, got nil")

		str = completeStr[:cryptMaxBsonObjSize-20000] // remove last 20000 bytes
		limitsDoc := readJSONFile(mt, "limits-doc.json")

		// insert a doc smaller than 2MiB that is bigger than 2MiB after encryption
		var extendedLimitsDoc []byte
		extendedLimitsDoc = append(extendedLimitsDoc, limitsDoc...)
		extendedLimitsDoc = extendedLimitsDoc[:len(extendedLimitsDoc)-1] // remove last byte to add new fields
		extendedLimitsDoc = bsoncore.AppendStringElement(extendedLimitsDoc, "_id", "encryption_exceeds_2mib")
		extendedLimitsDoc = bsoncore.AppendStringElement(extendedLimitsDoc, "unencrypted", string(str))
		extendedLimitsDoc, _ = bsoncore.AppendDocumentEnd(extendedLimitsDoc, 0)
		_, err = cpt.cseColl.InsertOne(mtest.Background, extendedLimitsDoc)
		assert.Nil(mt, err, "error inserting extended limits document: %v", err)

		// bulk insert two documents
		cpt.cseStarted = cpt.cseStarted[:0]
		str = completeStr[:cryptMaxBsonObjSize-18000]
		firstDoc := bson.D{{"_id", "no_encryption_under_2mib_1"}, {"unencrypted", string(str)}}
		secondDoc := bson.D{{"_id", "no_encryption_under_2mib_2"}, {"unencrypted", string(str)}}
		_, err = cpt.cseColl.InsertMany(mtest.Background, []interface{}{firstDoc, secondDoc})
		assert.Nil(mt, err, "InsertMany error for small documents: %v", err)
		assert.Equal(mt, 2, len(cpt.cseStarted), "expected 2 insert events, got %d", len(cpt.cseStarted))

		// bulk insert two larger documents
		str = completeStr[:cryptMaxBsonObjSize-20000]
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
		_, err = cpt.cseColl.InsertMany(mtest.Background, []interface{}{firstBulkDoc, secondBulkDoc})
		assert.Nil(mt, err, "InsertMany error for large documents: %v", err)
		assert.Equal(mt, 2, len(cpt.cseStarted), "expected 2 insert events, got %d", len(cpt.cseStarted))
	})
	mt.Run("views are prohibited", func(mt *mtest.T) {
		kmsProviders := map[string]map[string]interface{}{
			"local": {
				"key": localMasterKey,
			},
		}
		aeo := options.AutoEncryption().SetKmsProviders(kmsProviders).SetKeyVaultNamespace(kvNamespace)
		cpt := setup(mt, aeo, nil, nil)
		defer cpt.teardown(mt)

		// create view on db.coll
		mt.CreateCollection(mtest.Collection{
			Name:       "view",
			DB:         cpt.cseColl.Database().Name(),
			CreateOpts: bson.D{{"viewOn", "coll"}},
		}, true)

		view := cpt.cseColl.Database().Collection("view")
		_, err := view.InsertOne(mtest.Background, bson.D{{"_id", "insert_on_view"}})
		assert.NotNil(mt, err, "expected InsertOne error on view, got nil")
		errStr := strings.ToLower(err.Error())
		viewErrSubstr := "cannot auto encrypt a view"
		assert.True(mt, strings.Contains(errStr, viewErrSubstr),
			"expected error '%v' to contain substring '%v'", errStr, viewErrSubstr)
	})
	mt.RunOpts("corpus", noClientOpts, func(mt *mtest.T) {
		kmsProviders := map[string]map[string]interface{}{
			"aws": {
				"accessKeyId":     keyID,
				"secretAccessKey": secretAccessKey,
			},
			"local": {"key": localMasterKey},
		}
		corpusSchema := readJSONFile(mt, "corpus-schema.json")
		aeoWithoutSchema := options.AutoEncryption().SetKmsProviders(kmsProviders).SetKeyVaultNamespace(kvNamespace)
		localSchemaMap := map[string]interface{}{
			"db.coll": corpusSchema,
		}
		aeoWithSchema := options.AutoEncryption().SetKmsProviders(kmsProviders).SetKeyVaultNamespace(kvNamespace).
			SetSchemaMap(localSchemaMap)

		testCases := []struct {
			name   string
			aeo    *options.AutoEncryptionOptions
			schema bson.Raw // the schema to create the collection. if nil, the collection won't be explicitly created
		}{
			{"remote schema", aeoWithoutSchema, corpusSchema},
			{"local schema", aeoWithSchema, nil},
		}

		for _, tc := range testCases {
			mt.Run(tc.name, func(mt *mtest.T) {
				ceo := options.ClientEncryption().SetKmsProviders(kmsProviders).SetKeyVaultNamespace(kvNamespace)
				cpt := setup(mt, tc.aeo, defaultKvClientOptions, ceo)
				defer cpt.teardown(mt)

				// create collection with JSON schema
				if tc.schema != nil {
					db := cpt.coll.Database()
					err := db.RunCommand(mtest.Background, bson.D{
						{"create", "coll"},
						{"validator", bson.D{
							{"$jsonSchema", readJSONFile(mt, "corpus-schema.json")},
						}},
					}).Err()
					assert.Nil(mt, err, "create error with validator: %v", err)
				}

				localKeyFile := readJSONFile(mt, "corpus-key-local.json")
				awsKeyFile := readJSONFile(mt, "corpus-key-aws.json")
				// manually insert local and AWS keys into key vault
				_, err := cpt.keyVaultColl.InsertMany(mtest.Background, []interface{}{
					localKeyFile,
					awsKeyFile,
				})
				assert.Nil(mt, err, "InsertMany error for key vault: %v", err)
				sub, data := localKeyFile.Lookup("_id").Binary()
				localKey := primitive.Binary{Subtype: sub, Data: data}
				sub, data = awsKeyFile.Lookup("_id").Binary()
				awsKey := primitive.Binary{Subtype: sub, Data: data}

				// read original corpus and recursively copy over each value to new corpus, encrypting certain values
				// when needed
				corpus := readJSONFile(mt, "corpus.json")
				cidx, copied := bsoncore.AppendDocumentStart(nil)
				elems, _ := corpus.Elements()

				for _, elem := range elems {
					key := elem.Key()
					val := elem.Value()

					// top-level non-document elements that can be copied directly
					if key == "_id" || key == "altname_aws" || key == "altname_local" {
						copied = bsoncore.AppendStringElement(copied, key, val.StringValue())
						continue
					}

					// if method is auto, copy the value directly because it will be auto-encrypted later
					doc := val.Document()
					if doc.Lookup("method").StringValue() == "auto" {
						copied = bsoncore.AppendDocumentElement(copied, key, doc)
						continue
					}

					// explicitly encrypt value
					eo := options.Encrypt()
					algorithm := deterministicAlgorithm
					if doc.Lookup("algo").StringValue() == "rand" {
						algorithm = randomAlgorithm
					}
					eo.SetAlgorithm(algorithm)

					identifier := doc.Lookup("identifier").StringValue()
					kms := doc.Lookup("kms").StringValue()
					switch identifier {
					case "id":
						keyID := localKey
						if kms == "aws" {
							keyID = awsKey
						}

						eo.SetKeyID(keyID)
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

						encrypted, err := cpt.clientEnc.Encrypt(mtest.Background, deVal, eo)
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
				_, err = cpt.cseColl.InsertOne(mtest.Background, copied)
				assert.Nil(mt, err, "InsertOne error for corpus document: %v", err)

				// find document using client with encryption and assert it matches original
				decryptedDoc, err := cpt.cseColl.FindOne(mtest.Background, bson.D{}).DecodeBytes()
				assert.Nil(mt, err, "Find error with encrypted client: %v", err)
				assert.Equal(mt, corpus, decryptedDoc, "expected document %v, got %v", corpus, decryptedDoc)

				// find document using a client without encryption enabled and assert fields remain encrypted
				corpusEncrypted := readJSONFile(mt, "corpus-encrypted.json")
				foundDoc, err := cpt.coll.FindOne(mtest.Background, bson.D{}).DecodeBytes()
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
						expectedDecrypted, err := cpt.clientEnc.Decrypt(mtest.Background, primitive.Binary{Subtype: sub, Data: data})
						assert.Nil(mt, err, "Decrypt error: %v", err)
						sub, data = foundVal.Binary()
						actualDecrypted, err := cpt.clientEnc.Decrypt(mtest.Background, primitive.Binary{Subtype: sub, Data: data})
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
}

type cseProseTest struct {
	coll         *mongo.Collection // collection db.coll
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
	}, false)
	cpt.keyVaultColl = mt.CreateCollection(mtest.Collection{
		Name: "datakeys",
		DB:   "admin",
	}, false)

	if aeo != nil {
		cseMonitor := &event.CommandMonitor{
			Started: func(_ context.Context, evt *event.CommandStartedEvent) {
				cpt.cseStarted = append(cpt.cseStarted, evt)
			},
		}
		opts := options.Client().ApplyURI(mt.ConnString()).SetWriteConcern(mtest.MajorityWc).
			SetReadPreference(mtest.PrimaryRp).SetAutoEncryptionOptions(aeo).SetMonitor(cseMonitor)
		cpt.cseClient, err = mongo.Connect(mtest.Background, opts)
		assert.Nil(mt, err, "Connect error for encrypted client: %v", err)
		cpt.cseColl = cpt.cseClient.Database("db").Collection("coll")
	}
	if ceo != nil {
		kvClient, err := mongo.Connect(mtest.Background, kvClientOpts)
		assert.Nil(mt, err, "Connect error for ClientEncryption key vault client: %v", err)
		cpt.clientEnc, err = mongo.NewClientEncryption(kvClient, ceo)
		assert.Nil(mt, err, "NewClientEncryption error: %v", err)
	}
	return &cpt
}

func (cpt *cseProseTest) teardown(mt *mtest.T) {
	mt.Helper()

	err := cpt.cseClient.Disconnect(mtest.Background)
	assert.Nil(mt, err, "encrypted client Disconnect error: %v", err)
	if cpt.clientEnc != nil {
		err = cpt.clientEnc.Close(mtest.Background)
		assert.Nil(mt, err, "ClientEncryption Close error: %v", err)
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

func rawValueToCoreValue(rv bson.RawValue) bsoncore.Value {
	return bsoncore.Value{Type: rv.Type, Data: rv.Value}
}
