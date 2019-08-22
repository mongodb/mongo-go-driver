// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// +build cse

package mongo

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

var localMasterKey = []byte("2x44+xduTaBBkY16Er5DuADaghvS4vwdkg8tpPp3tz6gV01A1CwbD9itQ2HFDgPWOp8eMaC1Oi766JzXZBdBdbdMurdonJ1d")
var keyID = os.Getenv(awsAccessKeyID)
var secretAccessKey = os.Getenv(awsSecretAccessKey)
var connStr string

const clientEncryptionProseDir = "../data/client-side-encryption-prose"
const deterministicAlgorithm = "AEAD_AES_256_CBC_HMAC_SHA_512-Deterministic"
const randomAlgorithm = "AEAD_AES_256_CBC_HMAC_SHA_512-Random"
const kvNamespace = "admin.datakeys" // default namespace for the key vault collection
const keySubtype byte = 4            // expected subtype for data keys
const encryptedValueSubtype byte = 6 // expected subtypes for encrypted values
const cryptMaxBsonObjSize = 2097152  // max bytes in BSON object when auto encryption is enabled

func skipCse(t *testing.T) {
	t.Helper()
	dbName := "admin"
	db := createTestDatabase(t, &dbName)

	// check server version is > 4.2
	version, err := getServerVersion(db)
	require.NoError(t, err, "error getting server version: %v", err)
	if compareVersions(t, version, "4.2.0") < 0 {
		t.Skip("skipping because server version is < 4.2.0")
	}

	// check for enterprise build (mongocryptd and auto enc/dec is enterprise-only)
	res, err := db.RunCommand(ctx, bson.D{{"buildInfo", 1}}).DecodeBytes()
	require.NoError(t, err, "buildInfo error: %v", err)

	modulesRaw, err := res.LookupErr("modules")
	if err != nil {
		// older server versions don't report "modules" field in buildInfo result
		return
	}

	modules, err := modulesRaw.Array().Values()
	require.NoError(t, err, "error getting modules values: %v", err)
	for _, module := range modules {
		if module.StringValue() == "enterprise" {
			return
		}
	}
	t.Skip("skipping because server is not enterprise")
}

func TestClientSideEncryptionProse(t *testing.T) {
	skipCse(t)

	if keyID == "" {
		t.Fatalf("%s env var not set", awsAccessKeyID)
	}
	if secretAccessKey == "" {
		t.Fatalf("%s env var not set", secretAccessKey)
	}

	connStr = testutil.ConnString(t).Original
	defaultKvClientOptions := options.Client().ApplyURI(connStr)

	t.Run("data key and double encryption", func(t *testing.T) {
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
		cpt := setup(t, aeo, defaultKvClientOptions, ceo)
		defer teardown(t, cpt)

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
			t.Run(tc.name, func(t *testing.T) {
				// create data key
				altNames := []string{tc.keyAltName}
				dataKeyOpts := options.DataKey().SetKeyAltNames(altNames)
				if tc.masterKey != nil {
					dataKeyOpts.SetMasterKey(tc.masterKey)
				}
				dataKeyID, err := cpt.clientEnc.CreateDataKey(ctx, tc.provider, dataKeyOpts)
				require.NoError(t, err, "error creating data key: %v", err)
				require.Equal(t, keySubtype, dataKeyID.Subtype,
					"subtype mismatch; expected %v, got %v", keySubtype, dataKeyID.Subtype)

				// assert that the key exists in the key vault
				cursor, err := cpt.keyVaultColl.Find(ctx, bson.D{{"_id", dataKeyID}})
				require.NoError(t, err, "error running find on key vault: %v", err)
				require.True(t, cursor.Next(ctx), "no keys found in the key vault")
				provider := cursor.Current.Lookup("masterKey", "provider").StringValue()
				require.Equal(t, tc.provider, provider,
					"provider mismatch; expected %v, got %v", tc.provider, provider)
				require.False(t, cursor.Next(ctx), "found extra document in key vault: %v", cursor.Current)

				// encrypt a value with the new key by ID
				rawVal := bson.RawValue{Type: bson.TypeString, Value: bsoncore.AppendString(nil, tc.value)}
				encrypted, err := cpt.clientEnc.Encrypt(ctx, rawVal, options.Encrypt().SetAlgorithm(deterministicAlgorithm).SetKeyID(dataKeyID))
				require.NoError(t, err, "error encrypting value by ID: %v", err)
				require.Equal(t, encryptedValueSubtype, encrypted.Subtype,
					"subtype mismatch; expected %v, got %v", encryptedValueSubtype, encrypted.Subtype)

				// insert an encrypted value. the value shouldn't be encrypted again because it's not in the schema.
				_, err = cpt.cseColl.InsertOne(ctx, bson.D{{"_id", tc.provider}, {"value", encrypted}})
				require.NoError(t, err, "error inserting document: %v", err)

				// find the inserted document. the value should be decrypted automatically
				resBytes, err := cpt.cseColl.FindOne(ctx, bson.D{{"_id", tc.provider}}).DecodeBytes()
				require.NoError(t, err, "error running find: %v", err)
				foundVal := resBytes.Lookup("value").StringValue()
				require.Equal(t, tc.value, foundVal, "value mismatch; expected %v, got %v", tc.value, foundVal)

				// encrypt a value with an alternate name for the new key
				altEncrypted, err := cpt.clientEnc.Encrypt(ctx, rawVal,
					options.Encrypt().SetAlgorithm(deterministicAlgorithm).SetKeyAltName(tc.keyAltName))
				require.NoError(t, err, "error encrypting value with alt key name: %v", err)
				require.Equal(t, encryptedValueSubtype, altEncrypted.Subtype,
					"subtype mismatch; expected %v, got %v", encryptedValueSubtype, altEncrypted.Subtype)
				require.Equal(t, encrypted.Data, altEncrypted.Data,
					"data mismatch; expected %v, got %v", encrypted.Data, altEncrypted.Data)

				// insert an encrypted value for an auto-encrypted field
				_, err = cpt.cseColl.InsertOne(ctx, bson.D{{"encrypted_placeholder", encrypted}})
				require.Error(t, err, "expected error inserting document but got nil")
			})
		}
	})
	t.Run("external key vault", func(t *testing.T) {
		testCases := []struct {
			name          string
			externalVault bool
		}{
			{"with external vault", true},
			{"without external vault", false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// setup options structs
				kmsProviders := map[string]map[string]interface{}{
					"local": {
						"key": localMasterKey,
					},
				}
				schemaMap := map[string]interface{}{"db.coll": readJSONFile(t, "external-schema.json")}
				aeo := options.AutoEncryption().SetKmsProviders(kmsProviders).SetKeyVaultNamespace(kvNamespace).SetSchemaMap(schemaMap)
				ceo := options.ClientEncryption().SetKmsProviders(kmsProviders).SetKeyVaultNamespace(kvNamespace)
				kvClientOpts := defaultKvClientOptions

				if tc.externalVault {
					externalKvOpts := options.Client().ApplyURI(connStr).SetAuth(options.Credential{
						Username: "fake-user",
						Password: "fake-password",
					})
					aeo.SetKeyVaultClientOptions(externalKvOpts)
					kvClientOpts = externalKvOpts
				}
				cpt := setup(t, aeo, kvClientOpts, ceo)
				defer teardown(t, cpt)

				// manually insert data key
				key := readJSONFile(t, "external-key.json")
				_, err := cpt.keyVaultColl.InsertOne(ctx, key)
				require.NoError(t, err, "error inserting key: %v", err)
				subtype, data := key.Lookup("_id").Binary()
				dataKeyID := primitive.Binary{Subtype: subtype, Data: data}

				doc := bson.D{{"encrypted", "test"}}
				_, insertErr := cpt.cseClient.Database("db").Collection("coll").InsertOne(ctx, doc)
				rawVal := bson.RawValue{Type: bson.TypeString, Value: bsoncore.AppendString(nil, "test")}
				_, encErr := cpt.clientEnc.Encrypt(ctx, rawVal, options.Encrypt().SetKeyID(dataKeyID).SetAlgorithm(deterministicAlgorithm))

				if tc.externalVault {
					require.Error(t, insertErr, "expected auth error for insert but got nil")
					require.Error(t, encErr, "expected auth error for encryption but got nil")
					require.Contains(t, insertErr.Error(), "auth error", "expected auth error but got %v", insertErr)
					require.Contains(t, encErr.Error(), "auth error", "expected auth error but got %v", encErr)
					return
				}
				require.NoError(t, insertErr, "error inserting document: %v", insertErr)
				require.NoError(t, encErr, "error encrypting value: %v", encErr)
			})
		}
	})
	t.Run("bson size limits", func(t *testing.T) {
		kmsProviders := map[string]map[string]interface{}{
			"local": {
				"key": localMasterKey,
			},
		}
		aeo := options.AutoEncryption().SetKmsProviders(kmsProviders).SetKeyVaultNamespace(kvNamespace)
		cpt := setup(t, aeo, nil, nil)
		defer teardown(t, cpt)

		// create coll with JSON schema
		err := cpt.client.Database("db").RunCommand(ctx, bson.D{
			{"create", "coll"},
			{"validator", bson.D{
				{"$jsonSchema", readJSONFile(t, "limits-schema.json")},
			}},
		}).Err()
		require.NoError(t, err, "error creating coll with validator: %v", err)

		// insert key
		key := readJSONFile(t, "limits-key.json")
		_, err = cpt.keyVaultColl.InsertOne(ctx, key)
		require.NoError(t, err, "error inserting key: %v", err)

		var completeStr []byte
		for i := 0; i < cryptMaxBsonObjSize; i++ {
			completeStr = append(completeStr, 'a')
		}

		// insert a doc smaller than 2MiB
		str := completeStr[:cryptMaxBsonObjSize-18000] // remove last 18000 bytes
		doc := bson.D{{"_id", "no_encryption_under_2mib"}, {"unencrypted", string(str)}}
		_, err = cpt.cseColl.InsertOne(ctx, doc)
		require.NoError(t, err, "error inserting document: %v", err)

		// insert a doc larger than 2MiB
		doc = bson.D{{"_id", "no_encryption_over_2mib"}, {"unencrypted", string(completeStr)}}
		_, err = cpt.cseColl.InsertOne(ctx, doc)
		require.Error(t, err, "expected error for inserting large doc but got nil")

		str = completeStr[:cryptMaxBsonObjSize-20000] // remove last 20000 bytes
		limitsDoc := readJSONFile(t, "limits-doc.json")

		// insert a doc smaller than 2MiB that is bigger than 2MiB after encryption
		var extendedLimitsDoc []byte
		extendedLimitsDoc = append(extendedLimitsDoc, limitsDoc...)
		extendedLimitsDoc = extendedLimitsDoc[:len(extendedLimitsDoc)-1] // remove last byte to add new fields
		extendedLimitsDoc = bsoncore.AppendStringElement(extendedLimitsDoc, "_id", "encryption_exceeds_2mib")
		extendedLimitsDoc = bsoncore.AppendStringElement(extendedLimitsDoc, "unencrypted", string(str))
		extendedLimitsDoc, _ = bsoncore.AppendDocumentEnd(extendedLimitsDoc, 0)
		_, err = cpt.cseColl.InsertOne(ctx, extendedLimitsDoc)
		require.NoError(t, err, "error inserting extended limits doc: %v", err)

		// add command monitor for next tests
		var started []*event.CommandStartedEvent
		cpt.cseClient.monitor = &event.CommandMonitor{
			Started: func(_ context.Context, cse *event.CommandStartedEvent) {
				if cse.CommandName == "insert" {
					started = append(started, cse)
				}
			},
		}
		// bulk insert two documents
		str = completeStr[:cryptMaxBsonObjSize-18000]
		firstDoc := bson.D{{"_id", "no_encryption_under_2mib_1"}, {"unencrypted", string(str)}}
		secondDoc := bson.D{{"_id", "no_encryption_under_2mib_2"}, {"unencrypted", string(str)}}
		_, err = cpt.cseColl.InsertMany(ctx, []interface{}{firstDoc, secondDoc})
		require.NoError(t, err, "insertMany error for small docs: %v", err)
		require.Equal(t, 2, len(started), "expected 2 insert events but got %d", len(started))

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

		started = started[:0]
		_, err = cpt.cseColl.InsertMany(ctx, []interface{}{firstBulkDoc, secondBulkDoc})
		require.NoError(t, err, "insertMany error for large docs: %v", err)
		require.Equal(t, 2, len(started), "expected 2 insert events but got %d", len(started))
	})
	t.Run("views are prohibited", func(t *testing.T) {
		kmsProviders := map[string]map[string]interface{}{
			"local": {
				"key": localMasterKey,
			},
		}
		aeo := options.AutoEncryption().SetKmsProviders(kmsProviders).SetKeyVaultNamespace(kvNamespace)
		cpt := setup(t, aeo, nil, nil)
		defer teardown(t, cpt)

		// drop and create view on db.coll
		db := cpt.cseColl.Database()
		view := db.Collection("view")
		err := view.Drop(ctx)
		require.NoError(t, err, "error dropping view: %v", err)
		err = db.RunCommand(ctx, bson.D{
			{"create", "view"},
			{"viewOn", "coll"},
		}).Err()
		require.NoError(t, err, "error creating view: %v", err)

		_, err = view.InsertOne(ctx, bson.D{{"_id", "insert_on_view"}})
		require.Error(t, err, "expected error inserting a document into a view but got nil")
		errStr := strings.ToLower(err.Error())
		viewErrSubstr := "cannot auto encrypt a view"
		require.Contains(t, errStr, viewErrSubstr, "expected err '%v' to contain substring '%v'", errStr, viewErrSubstr)
	})
	t.Run("corpus", func(t *testing.T) {
		kmsProviders := map[string]map[string]interface{}{
			"aws": {
				"accessKeyId":     keyID,
				"secretAccessKey": secretAccessKey,
			},
			"local": {"key": localMasterKey},
		}
		corpusSchema := readJSONFile(t, "corpus-schema.json")
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
			t.Run(tc.name, func(t *testing.T) {
				ceo := options.ClientEncryption().SetKmsProviders(kmsProviders).SetKeyVaultNamespace(kvNamespace)
				cpt := setup(t, tc.aeo, defaultKvClientOptions, ceo)
				defer teardown(t, cpt)

				// create collection with JSON schema
				if tc.schema != nil {
					db := cpt.coll.Database()
					err := db.RunCommand(ctx, bson.D{
						{"create", "coll"},
						{"validator", bson.D{
							{"$jsonSchema", readJSONFile(t, "corpus-schema.json")},
						}},
					}).Err()
					require.NoError(t, err, "error creating collection with JSON schema: %v", err)
				}

				localKeyFile := readJSONFile(t, "corpus-key-local.json")
				awsKeyFile := readJSONFile(t, "corpus-key-aws.json")
				// manually insert local and AWS keys into key vault
				_, err := cpt.keyVaultColl.InsertMany(ctx, []interface{}{
					localKeyFile,
					awsKeyFile,
				})
				require.NoError(t, err, "error inserting keys: %v", err)
				sub, data := localKeyFile.Lookup("_id").Binary()
				localKey := primitive.Binary{Subtype: sub, Data: data}
				sub, data = awsKeyFile.Lookup("_id").Binary()
				awsKey := primitive.Binary{Subtype: sub, Data: data}

				// read original corpus and recursively copy over each value to new corpus, encrypting certain values
				// when needed
				corpus := readJSONFile(t, "corpus.json")
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
						t.Fatalf("unrecognized identifier: %v", identifier)
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

						encrypted, err := cpt.clientEnc.Encrypt(ctx, deVal, eo)
						if !doc.Lookup("allowed").Boolean() {
							// if allowed is false, encryption should error. in this case, the unencrypted value should be
							// copied over
							require.Error(t, err, "expected error encrypting value for %v but got nil", key)
							copied = bsoncore.AppendValueElement(copied, deKey, rawValueToCoreValue(deVal))
							continue
						}

						// copy encrypted value
						require.NoError(t, err, "error encrypting value for key %v: %v", key, err)
						copied = bsoncore.AppendBinaryElement(copied, deKey, encrypted.Subtype, encrypted.Data)
					}
					copied, _ = bsoncore.AppendDocumentEnd(copied, nestedIdx)
				}
				copied, _ = bsoncore.AppendDocumentEnd(copied, cidx)

				// insert document with encrypted values
				_, err = cpt.cseColl.InsertOne(ctx, copied)
				require.NoError(t, err, "error inserting corpus document: %v", err)

				// find document using client with encryption and assert it matches original
				unencryptedDoc, err := cpt.cseColl.FindOne(ctx, bson.D{}).DecodeBytes()
				require.NoError(t, err, "error running find with encrypted client: %v", err)
				require.Equal(t, corpus, unencryptedDoc, "document mismatch; expected %v, got %v", corpus, unencryptedDoc)

				// find document using a client without encryption enabled and assert fields remain encrypted
				corpusEncrypted := readJSONFile(t, "corpus-encrypted.json")
				foundDoc, err := cpt.coll.FindOne(ctx, bson.D{}).DecodeBytes()
				require.NoError(t, err, "error running find with unencrypted client: %v", err)

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
						require.True(t, expectedVal.Equal(foundVal),
							"value mismatch for key %v; expected %v, got %v", expectedKey, expectedVal, foundVal)
					case "rand":
						if allowed {
							require.False(t, expectedVal.Equal(foundVal),
								"expected values for key %v to be different but but were %v", expectedVal)
						}
					}

					// if allowed is true, decrypt both values with clientEnc and validate equality
					if allowed {
						sub, data := expectedVal.Binary()
						expectedDecrypted, err := cpt.clientEnc.Decrypt(ctx, primitive.Binary{Subtype: sub, Data: data})
						require.NoError(t, err, "error decrypting expected value: %v", err)
						sub, data = foundVal.Binary()
						actualDecrypted, err := cpt.clientEnc.Decrypt(ctx, primitive.Binary{Subtype: sub, Data: data})
						require.NoError(t, err, "error decrypting actual value: %v", err)

						require.True(t, expectedDecrypted.Equal(actualDecrypted),
							"decrypted value mismatch for key %v; expected %v, got %v", expectedKey, expectedDecrypted, actualDecrypted)
						continue
					}

					// if allowed is false, validate found value equals the original value in corpus
					corpusVal := corpus.Lookup(expectedKey).Document().Lookup("value")
					require.True(t, corpusVal.Equal(foundVal),
						"value mismatch for key %v; expected %v, got %v", expectedKey, corpusVal, foundVal)
				}
			})
		}
	})
}

type cseProseTest struct {
	client       *Client
	coll         *Collection
	keyVaultColl *Collection
	cseClient    *Client
	cseColl      *Collection
	clientEnc    *ClientEncryption
}

func setup(t *testing.T, aeo *options.AutoEncryptionOptions, kvClientOpts *options.ClientOptions,
	ceo *options.ClientEncryptionOptions) cseProseTest {

	cpt := cseProseTest{}
	var err error

	clientWc := writeconcern.New(writeconcern.WMajority())
	clientRp := readpref.Primary()
	cpt.client, err = Connect(ctx, options.Client().ApplyURI(connStr).SetWriteConcern(clientWc).SetReadPreference(clientRp))
	require.NoError(t, err, "error creating client: %v", err)

	cpt.keyVaultColl = cpt.client.Database("admin").Collection("datakeys")
	err = cpt.keyVaultColl.Drop(ctx)
	require.NoError(t, err, "error dropping admin.datakeys: %v", err)
	cpt.coll = cpt.client.Database("db").Collection("coll")
	err = cpt.coll.Drop(ctx)
	require.NoError(t, err, "error dropping admin.datakeys: %v", err)

	if aeo != nil {
		opts := options.Client().ApplyURI(connStr).SetWriteConcern(clientWc).SetReadPreference(clientRp).
			SetAutoEncryptionOptions(aeo)
		cpt.cseClient, err = Connect(ctx, opts)
		require.NoError(t, err, "error creating encrypted client: %v", err)
		cpt.cseColl = cpt.cseClient.Database("db").Collection("coll")
	}
	if ceo != nil {
		kvClient, err := Connect(ctx, kvClientOpts)
		require.NoError(t, err, "error creating key vault client for ClientEncrytpion: %v", err)
		cpt.clientEnc, err = NewClientEncryption(kvClient, ceo)
		require.NoError(t, err, "error creating ClientEncryption: %v", err)
	}
	return cpt
}

func teardown(t *testing.T, cpt cseProseTest) {
	t.Helper()

	err := cpt.client.Disconnect(ctx)
	require.NoError(t, err, "error disconnecting client: %v", err)
	err = cpt.cseClient.Disconnect(ctx)
	require.NoError(t, err, "error disconnecting encrypted client: %v", err)
	if cpt.clientEnc != nil {
		err = cpt.clientEnc.Close(ctx)
		require.NoError(t, err, "error closing ClientEncryption: %v", err)
	}
}

func rawValueToCoreValue(rv bson.RawValue) bsoncore.Value {
	return bsoncore.Value{Type: rv.Type, Data: rv.Value}
}

func readJSONFile(t *testing.T, file string) bson.Raw {
	t.Helper()

	content, err := ioutil.ReadFile(filepath.Join(clientEncryptionProseDir, file))
	require.NoError(t, err, "error reading file %v: %v", file, err)

	var doc bson.Raw
	err = bson.UnmarshalExtJSON(content, true, &doc)
	require.NoError(t, err, "error creating document from file %v: %v", file, err)
	return doc
}
