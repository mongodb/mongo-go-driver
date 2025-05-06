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

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/integtest"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	mcopts "go.mongodb.org/mongo-driver/v2/x/mongo/driver/mongocrypt/options"
)

// createDataKeyAndEncrypt creates a data key with the alternate name @keyName.
// Returns a ciphertext encrypted with the data key as test data.
func createDataKeyAndEncrypt(mt *mtest.T, keyName string) bson.Binary {
	mt.Helper()

	kvClientOpts := options.Client().
		ApplyURI(mtest.ClusterURI()).
		SetReadConcern(mtest.MajorityRc).
		SetWriteConcern(mtest.MajorityWc)

	integtest.AddTestServerAPIVersion(kvClientOpts)

	kmsProvidersMap := map[string]map[string]interface{}{
		"local": {"key": localMasterKey},
	}

	kvClient, err := mongo.Connect(kvClientOpts)
	defer kvClient.Disconnect(context.Background())
	assert.Nil(mt, err, "Connect error: %v", err)

	err = kvClient.Database("keyvault").Collection("datakeys").Drop(context.Background())
	assert.Nil(mt, err, "Drop error: %v", err)

	ceOpts := options.ClientEncryption().
		SetKmsProviders(kmsProvidersMap).
		SetKeyVaultNamespace("keyvault.datakeys")

	ce, err := mongo.NewClientEncryption(kvClient, ceOpts)
	assert.Nil(mt, err, "NewClientEncryption error: %v", err)

	dkOpts := options.DataKey().SetKeyAltNames([]string{keyName})
	_, err = ce.CreateDataKey(context.Background(), "local", dkOpts)
	assert.Nil(mt, err, "CreateDataKey error: %v", err)

	in := bson.RawValue{Type: bson.TypeString, Value: bsoncore.AppendString(nil, "test")}
	eOpts := options.Encrypt().
		SetAlgorithm("AEAD_AES_256_CBC_HMAC_SHA_512-Random").
		SetKeyAltName(keyName)

	ciphertext, err := ce.Encrypt(context.Background(), in, eOpts)
	assert.Nil(mt, err, "Encrypt error: %v", err)
	return ciphertext
}

func getLsid(mt *mtest.T, doc bson.Raw) bson.Raw {
	mt.Helper()

	lsid, err := doc.LookupErr("lsid")
	assert.Nil(mt, err, "expected lsid in document: %v", doc)
	lsidDoc, ok := lsid.DocumentOK()
	assert.True(mt, ok, "expected lsid to be document, but got: %v", lsid)
	return lsidDoc
}

func makeMonitor(mt *mtest.T, captured *[]event.CommandStartedEvent) *event.CommandMonitor {
	mt.Helper()
	assert.NotNil(mt, captured, "captured is nil")

	return &event.CommandMonitor{
		Started: func(_ context.Context, cse *event.CommandStartedEvent) {
			assert.NotNil(mt, cse, "expected non-Nil CommandStartedEvent")
			*captured = append(*captured, *cse)
		},
	}
}

func TestClientSideEncryptionWithExplicitSessions(t *testing.T) {
	verifyClientSideEncryptionVarsSet(t)
	mt := mtest.New(t, mtest.NewOptions().MinServerVersion("4.2").Enterprise(true).CreateClient(false))

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

	mt.Run("automatic encryption", func(mt *mtest.T) {
		createDataKeyAndEncrypt(mt, "myKey")

		aeOpts := options.AutoEncryption().
			SetKmsProviders(kmsProvidersMap).
			SetKeyVaultNamespace("keyvault.datakeys").
			SetSchemaMap(schemaMap).
			SetExtraOptions(getCryptSharedLibExtraOptions())

		var capturedEvents []event.CommandStartedEvent

		clientOpts := options.Client().
			ApplyURI(mtest.ClusterURI()).
			SetReadConcern(mtest.MajorityRc).
			SetWriteConcern(mtest.MajorityWc).
			SetAutoEncryptionOptions(aeOpts).
			SetMonitor(makeMonitor(mt, &capturedEvents))

		integtest.AddTestServerAPIVersion(clientOpts)

		client, err := mongo.Connect(clientOpts)
		assert.Nil(mt, err, "Connect error: %v", err)
		defer client.Disconnect(context.Background())

		coll := client.Database("db").Collection("coll")
		err = coll.Drop(context.Background())
		assert.Nil(mt, err, "Drop error: %v", err)

		session, err := client.StartSession()
		assert.Nil(mt, err, "StartSession error: %v", err)
		sessionCtx := mongo.NewSessionContext(context.Background(), session)

		capturedEvents = make([]event.CommandStartedEvent, 0)
		_, err = coll.InsertOne(sessionCtx, bson.D{{"encryptMe", "test"}, {"keyName", "myKey"}})
		assert.Nil(mt, err, "InsertOne error: %v", err)

		assert.Equal(mt, len(capturedEvents), 2, "expected 2 events, got %v", len(capturedEvents))

		// Assert the first event is a find on the keyvault.datakeys collection.
		event := capturedEvents[0]
		assert.Equal(mt, event.CommandName, "find", "expected command find, got %q", event.CommandName)
		assert.Equal(mt, event.DatabaseName, "keyvault", "expected find on keyvault, got %q", event.DatabaseName)

		// Assert the find used an implicit session with an lsid != session.ID()
		lsid := getLsid(mt, event.Command)
		assert.Nil(mt, err, "lsid not found in %v", event.Command)
		assert.NotEqual(mt, lsid, session.ID(), "expected different lsid, but got %v", lsid)

		// Assert the second event is the original insert.
		event = capturedEvents[1]
		assert.Equal(mt, event.CommandName, "insert", "expected command insert, got %q", event.CommandName)

		// Assert the insert used the explicit session.
		lsid = getLsid(mt, event.Command)
		assert.Nil(mt, err, "lsid not found on %v", event.Command)
		assert.Equal(mt, lsid, session.ID(), "expected lsid %v, but got %v", session.ID(), lsid)

		// Check that encryptMe is encrypted.
		encryptMe, err := event.Command.LookupErr("documents", "0", "encryptMe")
		assert.Nil(mt, err, "could not find encryptMe in %v", event.Command)
		assert.Equal(mt, encryptMe.Type, bson.TypeBinary, "expected Binary, got %v", encryptMe.Type)
	})

	mt.Run("automatic decryption", func(mt *mtest.T) {
		ciphertext := createDataKeyAndEncrypt(mt, "myKey")

		aeOpts := options.AutoEncryption().
			SetKmsProviders(kmsProvidersMap).
			SetKeyVaultNamespace("keyvault.datakeys").
			SetBypassAutoEncryption(true)

		var capturedEvents []event.CommandStartedEvent

		clientOpts := options.Client().
			ApplyURI(mtest.ClusterURI()).
			SetReadConcern(mtest.MajorityRc).
			SetWriteConcern(mtest.MajorityWc).
			SetAutoEncryptionOptions(aeOpts).
			SetMonitor(makeMonitor(mt, &capturedEvents))

		integtest.AddTestServerAPIVersion(clientOpts)

		client, err := mongo.Connect(clientOpts)
		assert.Nil(mt, err, "Connect error: %v", err)
		defer client.Disconnect(context.Background())

		coll := client.Database("db").Collection("coll")
		err = coll.Drop(context.Background())
		assert.Nil(mt, err, "Drop error: %v", err)
		_, err = coll.InsertOne(context.Background(), bson.D{{"encryptMe", ciphertext}})
		assert.Nil(mt, err, "InsertOne error: %v", err)

		session, err := client.StartSession()
		assert.Nil(mt, err, "StartSession error: %v", err)
		sessionCtx := mongo.NewSessionContext(context.Background(), session)

		capturedEvents = make([]event.CommandStartedEvent, 0)
		res := coll.FindOne(sessionCtx, bson.D{{}})
		assert.Nil(mt, res.Err(), "FindOne error: %v", res.Err())

		assert.Equal(mt, len(capturedEvents), 2, "expected 2 events, got %v", len(capturedEvents))

		// Assert the first event is the original find.
		event := capturedEvents[0]
		assert.Equal(mt, event.CommandName, "find", "expected command find, got %q", event.CommandName)
		assert.Equal(mt, event.DatabaseName, "db", "expected find on db, got %q", event.DatabaseName)

		// Assert the find used the explicit session
		lsid := getLsid(mt, event.Command)
		assert.Nil(mt, err, "lsid not found on %v", event.Command)
		assert.Equal(mt, lsid, session.ID(), "expected lsid %v, but got %v", session.ID(), lsid)

		// Assert the second event is the find on the keyvault.datakeys collection.
		event = capturedEvents[1]
		assert.Equal(mt, event.CommandName, "find", "expected command find, got %q", event.CommandName)
		assert.Equal(mt, event.DatabaseName, "keyvault", "expected find on keyvault, got %q", event.DatabaseName)

		// Assert the find used an implicit session with an lsid != session.ID()
		lsid = getLsid(mt, event.Command)
		assert.Nil(mt, err, "lsid not found on %v", event.Command)
		assert.NotEqual(mt, lsid, session.ID(), "expected different lsid, but got %v", lsid)
	})
}

// customCrypt is a test implementation of the driver.Crypt interface. It keeps track of the number of times its
// methods have been called.
type customCrypt struct {
	numEncryptCalls                   int
	numDecryptCalls                   int
	numCreateDataKeyCalls             int
	numEncryptExplicitCalls           int
	numEncryptExplicitExpressionCalls int
	numDecryptExplicitCalls           int
	numCloseCalls                     int
	numBypassAutoEncryptionCalls      int
	numRewrapDataKeyCalls             int
}

var (
	_     driver.Crypt = (*customCrypt)(nil)
	mySSN              = "123456789"
)

// Encrypt encrypts the given command.
func (c *customCrypt) Encrypt(_ context.Context, _ string, cmd bsoncore.Document) (bsoncore.Document, error) {
	c.numEncryptCalls++
	elems, err := cmd.Elements()
	if err != nil {
		return nil, err
	}

	encryptedCmd := bsoncore.NewDocumentBuilder()
	for _, elem := range elems {
		// "encrypt" ssn element as "hidden"
		if elem.Key() == "ssn" {
			encryptedCmd = encryptedCmd.AppendString("ssn", "hidden")
		} else {
			encryptedCmd = encryptedCmd.AppendValue(elem.Key(), elem.Value())
		}
	}
	return encryptedCmd.Build(), nil
}

// Decrypt decrypts the given command response.
func (c *customCrypt) Decrypt(_ context.Context, cmdResponse bsoncore.Document) (bsoncore.Document, error) {
	c.numDecryptCalls++
	elems, err := cmdResponse.Elements()
	if err != nil {
		return nil, err
	}

	decryptedCmdResponse := bsoncore.NewDocumentBuilder()
	for _, elem := range elems {
		// "decrypt" ssn element as mySSN
		if elem.Key() == "ssn" {
			decryptedCmdResponse = decryptedCmdResponse.AppendString("ssn", mySSN)
		} else {
			decryptedCmdResponse = decryptedCmdResponse.AppendValue(elem.Key(), elem.Value())
		}
	}
	return decryptedCmdResponse.Build(), nil
}

// CreateDataKey implements the driver.Crypt interface.
func (c *customCrypt) CreateDataKey(_ context.Context, _ string, _ *mcopts.DataKeyOptions) (bsoncore.Document, error) {
	c.numCreateDataKeyCalls++
	return nil, nil
}

// EncryptExplicit implements the driver.Crypt interface.
func (c *customCrypt) EncryptExplicit(_ context.Context, _ bsoncore.Value, _ *mcopts.ExplicitEncryptionOptions) (byte, []byte, error) {
	c.numEncryptExplicitCalls++
	return 0, nil, nil
}

// EncryptExplicit implements the driver.Crypt interface.
func (c *customCrypt) EncryptExplicitExpression(_ context.Context, _ bsoncore.Document, _ *mcopts.ExplicitEncryptionOptions) (bsoncore.Document, error) {
	c.numEncryptExplicitExpressionCalls++
	return nil, nil
}

// DecryptExplicit implements the driver.Crypt interface.
func (c *customCrypt) DecryptExplicit(_ context.Context, _ byte, _ []byte) (bsoncore.Value, error) {
	c.numDecryptExplicitCalls++
	return bsoncore.Value{}, nil
}

// Close implements the driver.Crypt interface.
func (c *customCrypt) Close() {
	c.numCloseCalls++
}

// BypassAutoEncryption implements the driver.Crypt interface.
func (c *customCrypt) BypassAutoEncryption() bool {
	c.numBypassAutoEncryptionCalls++
	return false
}

// RewrapDataKey attempts to rewrap the document data keys matching the filter, preparing the re-wrapped documents to
// be returned as a slice of bsoncore.Document.
func (c *customCrypt) RewrapDataKey(_ context.Context, _ []byte,
	_ *mcopts.RewrapManyDataKeyOptions) ([]bsoncore.Document, error) {

	c.numRewrapDataKeyCalls++
	return nil, nil
}

func TestClientSideEncryptionCustomCrypt(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().MinServerVersion("4.2").Enterprise(true).CreateClient(false))

	kmsProvidersMap := map[string]map[string]interface{}{
		"local": {"key": localMasterKey},
	}

	mt.Run("auto encryption and decryption", func(mt *mtest.T) {
		aeOpts := options.AutoEncryption().
			SetKmsProviders(kmsProvidersMap).
			SetKeyVaultNamespace("keyvault.datakeys").
			SetExtraOptions(getCryptSharedLibExtraOptions())
		clientOpts := options.Client().
			ApplyURI(mtest.ClusterURI()).
			SetAutoEncryptionOptions(aeOpts)
		cc := &customCrypt{}
		clientOpts.Crypt = cc
		integtest.AddTestServerAPIVersion(clientOpts)

		client, err := mongo.Connect(clientOpts)
		defer client.Disconnect(context.Background())
		assert.Nil(mt, err, "Connect error: %v", err)

		coll := client.Database("db").Collection("coll")
		defer func() { _ = coll.Drop(context.Background()) }()

		doc := bson.D{{"foo", "bar"}, {"ssn", mySSN}}
		_, err = coll.InsertOne(context.Background(), doc)
		assert.Nil(mt, err, "InsertOne error: %v", err)

		res := coll.FindOne(context.Background(), bson.D{{"foo", "bar"}})
		assert.Nil(mt, res.Err(), "FindOne error: %v", err)

		rawRes, err := res.Raw()
		assert.Nil(mt, err, "Raw error: %v", err)
		ssn, ok := rawRes.Lookup("ssn").StringValueOK()
		assert.True(mt, ok, "expected 'ssn' value to be type string, got %T", ssn)
		assert.Equal(mt, ssn, mySSN, "expected 'ssn' value %q, got %q", mySSN, ssn)

		// Assert customCrypt methods are called the correct number of times.
		assert.Equal(mt, cc.numEncryptCalls, 1,
			"expected 1 call to Encrypt, got %v", cc.numEncryptCalls)
		assert.Equal(mt, cc.numDecryptCalls, 1,
			"expected 1 call to Decrypt, got %v", cc.numDecryptCalls)
		assert.Equal(mt, cc.numCreateDataKeyCalls, 0,
			"expected 0 calls to CreateDataKey, got %v", cc.numCreateDataKeyCalls)
		assert.Equal(mt, cc.numEncryptExplicitCalls, 0,
			"expected 0 calls to EncryptExplicit, got %v", cc.numEncryptExplicitCalls)
		assert.Equal(mt, cc.numEncryptExplicitExpressionCalls, 0,
			"expected 0 calls to EncryptExplicitExpression, got %v", cc.numEncryptExplicitExpressionCalls)
		assert.Equal(mt, cc.numDecryptExplicitCalls, 0,
			"expected 0 calls to DecryptExplicit, got %v", cc.numDecryptExplicitCalls)
		assert.Equal(mt, cc.numCloseCalls, 0,
			"expected 0 calls to Close, got %v", cc.numCloseCalls)
		assert.Equal(mt, cc.numBypassAutoEncryptionCalls, 1,
			"expected 1 call to BypassAutoEncryption, got %v", cc.numBypassAutoEncryptionCalls)
	})
}

func TestFLE2CreateCollection(t *testing.T) {
	// FLE 2 (aka Queryable Encryption) is not supported on Standalone topology.
	mtOpts := mtest.NewOptions().
		MinServerVersion("7.0").
		Enterprise(true).
		CreateClient(false).
		Topologies(mtest.ReplicaSet,
			mtest.Sharded,
			mtest.LoadBalanced,
			mtest.ShardedReplicaSet)
	mt := mtest.New(t, mtOpts)

	efJSON := `
	{
		"escCollection": "enxcol_.encryptedCollection.esc",
		"ecocCollection": "enxcol_.encryptedCollection.ecoc",
		"fields": [
		  {
			"path": "firstName",
			"bsonType": "string",
			"keyId": {
			  "$binary": {
				"subType": "04",
				"base64": "AAAAAAAAAAAAAAAAAAAAAA=="
			  }
			}
		  }
		]
	  }
	`
	var efBSON bson.Raw
	err := bson.UnmarshalExtJSON([]byte(efJSON), true /* canonical */, &efBSON)
	assert.Nil(mt, err, "UnmarshalExtJSON error: %v", err)

	// Test the behavior in the specification test fle2-CreateCollection.json: "CreateCollection from encryptedFields.".
	// The Go driver does not support encryptedFields as an option to Drop. See: GODRIVER-2413.
	mt.Run("CreateCollection from encryptedFields", func(mt *mtest.T) {
		// Drop data and state collections to clean up from a prior test run.
		{
			err = mt.DB.Collection("enxcol_.encryptedCollection.esc").Drop(context.Background())
			assert.Nil(mt, err, "error in Drop: %v", err)
			err = mt.DB.Collection("enxcol_.encryptedCollection.ecoc").Drop(context.Background())
			assert.Nil(mt, err, "error in Drop: %v", err)
			err := mt.DB.Collection("coll").Drop(context.Background())
			assert.Nil(mt, err, "error in Drop: %v", err)
		}

		mt.DB.CreateCollection(context.Background(), "coll", options.CreateCollection().SetEncryptedFields(efBSON))

		// Check expected collections and index exist.
		{
			got, err := mt.DB.ListCollectionNames(context.Background(), bson.D{{"name", "coll"}})
			assert.Nil(mt, err, "error in ListCollectionNames")
			assert.Equal(mt, got, []string{"coll"}, "expected ['coll'], got: %v", got)

			got, err = mt.DB.ListCollectionNames(context.Background(), bson.D{{"name", "enxcol_.encryptedCollection.esc"}})
			assert.Nil(mt, err, "error in ListCollectionNames")
			assert.Equal(mt, got, []string{"enxcol_.encryptedCollection.esc"}, "expected ['encryptedCollection.esc'], got: %v", got)

			got, err = mt.DB.ListCollectionNames(context.Background(), bson.D{{"name", "enxcol_.encryptedCollection.ecoc"}})
			assert.Nil(mt, err, "error in ListCollectionNames")
			assert.Equal(mt, got, []string{"enxcol_.encryptedCollection.ecoc"}, "expected ['encryptedCollection.ecoc'], got: %v", got)

			indexSpecs, err := mt.DB.Collection("coll").Indexes().ListSpecifications(context.Background())
			assert.Nil(mt, err, "error in Indexes().ListSpecifications: %v", err)
			assert.Equal(mt, len(indexSpecs), 2, "expected two indexes on 'coll', got: %v", indexSpecs)
			assert.Equal(mt, indexSpecs[1].Name, "__safeContent___1", "expected second index to be '__safeContent___1', got %v", indexSpecs[1].Name)
		}
	})
}

func TestFLE2DocsExample(t *testing.T) {
	// FLE 2 is not supported on Standalone topology.
	// Only test MongoDB Server 7.0+. MongoDB Server 7.0 introduced a backwards breaking change to the Queryable Encryption (QE) protocol: QEv2.
	// libmongocrypt is configured to use the QEv2 protocol.
	mtOpts := mtest.NewOptions().
		MinServerVersion("7.0").
		Enterprise(true).
		CreateClient(false).
		Topologies(mtest.ReplicaSet,
			mtest.Sharded,
			mtest.LoadBalanced,
			mtest.ShardedReplicaSet)
	mt := mtest.New(t, mtOpts)

	mt.Run("Auto Encryption", func(mt *mtest.T) {
		// Drop data from prior test runs.
		{
			err := mt.Client.Database("keyvault").Collection("datakeys").Drop(context.Background())
			assert.Nil(mt, err, "error in Drop: %v", err)
			err = mt.Client.Database("docsExamples").Drop(context.Background())
			assert.Nil(mt, err, "error in Drop: %v", err)
		}

		kmsProvidersMap := map[string]map[string]interface{}{
			"local": {"key": localMasterKey},
		}

		var key1ID bson.Binary
		var key2ID bson.Binary

		// Create two data keys.
		{
			cOpts := options.Client().ApplyURI(mtest.ClusterURI())
			integtest.AddTestServerAPIVersion(cOpts)
			keyVaultClient, err := mongo.Connect(cOpts)
			assert.Nil(mt, err, "error in Connect: %v", err)
			defer keyVaultClient.Disconnect(context.Background())
			ceOpts := options.ClientEncryption().SetKmsProviders(kmsProvidersMap).SetKeyVaultNamespace("keyvault.datakeys")
			ce, err := mongo.NewClientEncryption(keyVaultClient, ceOpts)
			assert.Nil(mt, err, "error in NewClientEncryption: %v", err)
			defer ce.Close(context.Background())
			key1ID, err = ce.CreateDataKey(context.Background(), "local")
			assert.Nil(mt, err, "error in CreateDataKey: %v", err)
			key2ID, err = ce.CreateDataKey(context.Background(), "local")
			assert.Nil(mt, err, "error in CreateDataKey: %v", err)
		}

		// Create an encryptedFieldsMap.
		encryptedFieldsMap := bson.M{
			"docsExamples.encrypted": bson.M{
				"fields": []bson.M{
					{
						"path":     "encryptedIndexed",
						"bsonType": "string",
						"keyId":    key1ID,
						"queries": []bson.M{
							{
								"queryType": "equality",
							},
						},
					},
					{
						"path":     "encryptedUnindexed",
						"bsonType": "string",
						"keyId":    key2ID,
					},
				},
			},
		}

		// Create an FLE 2 collection.
		var encryptedColl *mongo.Collection
		{
			cOpts := options.Client().ApplyURI(mtest.ClusterURI())
			integtest.AddTestServerAPIVersion(cOpts)
			aeOpts := options.AutoEncryption().SetKmsProviders(kmsProvidersMap).SetKeyVaultNamespace("keyvault.datakeys").SetEncryptedFieldsMap(encryptedFieldsMap).SetExtraOptions(getCryptSharedLibExtraOptions())
			cOpts.SetAutoEncryptionOptions(aeOpts)
			encryptedClient, err := mongo.Connect(cOpts)
			defer encryptedClient.Disconnect(context.Background())
			assert.Nil(mt, err, "error in Connect: %v", err)
			// Create the FLE 2 collection docsExample.encrypted.
			db := encryptedClient.Database("docsExamples")
			// Because docsExample.encrypted is in encryptedFieldsMap, it is created with FLE 2 support.
			err = db.CreateCollection(context.Background(), "encrypted")
			assert.Nil(mt, err, "error in CreateCollection")
			encryptedColl = db.Collection("encrypted")
		}

		// Auto encrypt an insert and find.
		{
			// Encrypt an insert.
			_, err := encryptedColl.InsertOne(context.Background(), bson.M{
				"_id":                1,
				"encryptedIndexed":   "indexedValue",
				"encryptedUnindexed": "unindexedValue",
			})
			assert.Nil(mt, err, "error in InsertOne")

			// Encrypt a find.
			res := encryptedColl.FindOne(context.Background(), bson.M{
				"encryptedIndexed": "indexedValue",
			})
			assert.Nil(mt, res.Err(), "error in FindOne: %v", res.Err())
			var resBSON bson.M
			err = res.Decode(&resBSON)
			assert.Nil(mt, err, "error in Decode: %v", err)
			assert.Equal(mt, resBSON["encryptedIndexed"], "indexedValue", "expected 'indexedValue', got %q", resBSON["encryptedIndexed"])
			assert.Equal(mt, resBSON["encryptedUnindexed"], "unindexedValue", "expected 'unindexedValue', got %q", resBSON["encryptedUnindexed"])
		}

		// Find documents without decryption.
		{
			unencryptedColl := mt.Client.Database("docsExamples").Collection("encrypted")
			res := unencryptedColl.FindOne(context.Background(), bson.M{"_id": 1})
			assert.Nil(mt, res.Err(), "error in FindOne: %v", res.Err())
			resBSON, err := res.Raw()
			assert.Nil(mt, err, "error in Raw: %v", err)

			val := resBSON.Lookup("encryptedIndexed")
			assert.Equal(mt, val.Type, bson.TypeBinary, "expected encryptedIndexed to be Binary, got %v", val.Type)
			val = resBSON.Lookup("encryptedUnindexed")
			assert.Equal(mt, val.Type, bson.TypeBinary, "expected encryptedUnindexed to be Binary, got %v", val.Type)
		}
	})
}

// `TestFLE2CreateCollectionWithAutoEncryption` is a regression test for a bug fixed in GODRIVER-2413.
// Prior to GODRIVER-2413, the `IndexView.CreateMany` operation was not processed for automatic encryption. This resulted in no "listCollections" command sent.
func TestFLE2CreateCollectionWithAutoEncryption(t *testing.T) {
	mtOpts := mtest.NewOptions().
		MinServerVersion("7.0").
		Enterprise(true).
		CreateClient(false).
		Topologies(mtest.ReplicaSet,
			mtest.Sharded,
			mtest.LoadBalanced,
			mtest.ShardedReplicaSet)
	mt := mtest.New(t, mtOpts)

	mt.Run("TestFLE2CreateCollectionWithAutoEncryption", func(mt *mtest.T) {
		// Drop data from prior test runs.
		{
			err := mt.Client.Database("keyvault").Collection("datakeys").Drop(context.Background())
			assert.Nil(mt, err, "error in Drop: %v", err)
			err = mt.Client.Database("db").Drop(context.Background())
			assert.Nil(mt, err, "error in Drop: %v", err)
		}

		kmsProvidersMap := map[string]map[string]interface{}{
			"local": {"key": localMasterKey},
		}

		// Use an empty encryptedFields.
		encryptedFields := bson.M{
			"fields": []bson.M{},
		}

		// Store names of started commands.
		startedCommands := make([]string, 0)
		cmdMonitor := &event.CommandMonitor{
			Started: func(_ context.Context, evt *event.CommandStartedEvent) {
				startedCommands = append(startedCommands, evt.CommandName)
			},
		}

		// Create a Client with Auto Encryption enabled.
		var encryptedClient *mongo.Client
		{
			aeOpts := options.AutoEncryption().
				SetKmsProviders(kmsProvidersMap).
				SetKeyVaultNamespace("keyvault.datakeys").
				SetExtraOptions(getCryptSharedLibExtraOptions())

			cOpts := options.Client().
				ApplyURI(mtest.ClusterURI()).
				SetMonitor(cmdMonitor).
				SetAutoEncryptionOptions(aeOpts)

			integtest.AddTestServerAPIVersion(cOpts)

			var err error
			encryptedClient, err = mongo.Connect(cOpts)
			defer encryptedClient.Disconnect(context.Background())
			assert.Nil(mt, err, "error in Connect: %v", err)
		}

		// Create a collection with the encrypted fields.
		encryptedClient.Database("db").CreateCollection(context.Background(), "coll", options.CreateCollection().SetEncryptedFields(encryptedFields))

		// Check resulting events sent.
		assert.Equal(mt, []string{
			"create",          // Create ESC collection.
			"create",          // Create ECOC collection.
			"create",          // Create 'coll' collection.
			"listCollections", // Run listCollections when processing `createIndexes` command for automatic encryption.
			"createIndexes",
		}, startedCommands)
	})
}

func TestFLEIndexView(t *testing.T) {
	verifyClientSideEncryptionVarsSet(t)

	mt := mtest.New(t, mtest.NewOptions().MinServerVersion("4.2").Enterprise(true).CreateClient(false))

	opts := options.Client().ApplyURI(mtest.ClusterURI()).SetWriteConcern(mtest.MajorityWc).
		SetReadPreference(mtest.PrimaryRp)

	cc := &customCrypt{}
	opts.Crypt = cc

	integtest.AddTestServerAPIVersion(opts)

	client, err := mongo.Connect(opts)
	assert.NoError(mt, err)

	mt.Cleanup(func() { client.Disconnect(context.Background()) })

	coll := client.Database("db").Collection("coll")

	err = coll.Drop(context.Background())
	assert.Nil(mt, err, "Drop error: %v", err)

	mt.Run("create many", func(mt *mtest.T) {
		createIndexes(mt, coll, 2)

		assert.Equal(mt, cc.numEncryptCalls, 2, "expected 2 calls to Encrypt, got %v", cc.numEncryptCalls)
	})

	mt.Run("list", func(mt *mtest.T) {
		cc.numEncryptCalls = 0 // Reset Encrypt calls from createIndexes

		_, err := coll.Indexes().List(context.Background(), options.ListIndexes().SetBatchSize(2))
		assert.NoError(mt, err, "error creating list cursor: %v", err)

		assert.Equal(mt, cc.numEncryptCalls, 1, "expected 1 call to Encrypt, got %v", cc.numEncryptCalls)
	})

	mt.Run("list specifications", func(mt *mtest.T) {
		cc.numEncryptCalls = 0 // Reset Encrypt calls from createIndexes

		_, err := coll.Indexes().ListSpecifications(context.Background())
		assert.NoError(mt, err, "error listing specifications : %v", err)

		assert.Equal(mt, cc.numEncryptCalls, 1, "expected 1 call to Encrypt, got %v", cc.numEncryptCalls)
	})

	mt.Run("drop one", func(mt *mtest.T) {
		createIndexes(mt, coll, 1)

		cc.numEncryptCalls = 0 // Reset Encrypt calls from createIndexes

		err := coll.Indexes().DropOne(context.Background(), "a_1")
		assert.NoError(mt, err, "error dropping one index: %v", err)

		assert.Equal(mt, cc.numEncryptCalls, 1, "expected 1 call to Encrypt, got %v", cc.numEncryptCalls)
	})

	mt.Run("drop all", func(mt *mtest.T) {
		createIndexes(mt, coll, 2)

		cc.numEncryptCalls = 0 // Reset Encrypt calls from createIndexes

		err := coll.Indexes().DropAll(context.Background())
		assert.NoError(mt, err, "error dropping all indexes: %v", err)

		assert.Equal(mt, cc.numEncryptCalls, 1, "expected 1 call to Encrypt, got %v", cc.numEncryptCalls)
	})
}
