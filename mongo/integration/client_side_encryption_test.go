// Copyright (C) MongoDB, Inc. 2021-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// +build cse

package integration

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	mcopts "go.mongodb.org/mongo-driver/x/mongo/driver/mongocrypt/options"
)

// createDataKeyAndEncrypt creates a data key with the alternate name @keyName.
// Returns a ciphertext encrypted with the data key as test data.
func createDataKeyAndEncrypt(mt *mtest.T, keyName string) primitive.Binary {
	mt.Helper()

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

	in := bson.RawValue{Type: bsontype.String, Value: bsoncore.AppendString(nil, "test")}
	eOpts := options.Encrypt().
		SetAlgorithm("AEAD_AES_256_CBC_HMAC_SHA_512-Random").
		SetKeyAltName(keyName)

	ciphertext, err := ce.Encrypt(mtest.Background, in, eOpts)
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

	mt.Run("automatic encryption", func(mt *mtest.T) {
		createDataKeyAndEncrypt(mt, "myKey")

		aeOpts := options.AutoEncryption().
			SetKmsProviders(kmsProvidersMap).
			SetKeyVaultNamespace("keyvault.datakeys").
			SetSchemaMap(schemaMap)

		var capturedEvents []event.CommandStartedEvent

		clientOpts := options.Client().
			ApplyURI(mtest.ClusterURI()).
			SetReadConcern(mtest.MajorityRc).
			SetWriteConcern(mtest.MajorityWc).
			SetAutoEncryptionOptions(aeOpts).
			SetMonitor(makeMonitor(mt, &capturedEvents))

		testutil.AddTestServerAPIVersion(clientOpts)

		client, err := mongo.Connect(mtest.Background, clientOpts)
		assert.Nil(mt, err, "Connect error: %v", err)
		defer client.Disconnect(mtest.Background)

		coll := client.Database("db").Collection("coll")
		err = coll.Drop(mtest.Background)
		assert.Nil(mt, err, "Drop error: %v", err)

		session, err := client.StartSession()
		assert.Nil(mt, err, "StartSession error: %v", err)
		sessionCtx := mongo.NewSessionContext(mtest.Background, session)

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

		testutil.AddTestServerAPIVersion(clientOpts)

		client, err := mongo.Connect(mtest.Background, clientOpts)
		assert.Nil(mt, err, "Connect error: %v", err)
		defer client.Disconnect(mtest.Background)

		coll := client.Database("db").Collection("coll")
		err = coll.Drop(mtest.Background)
		assert.Nil(mt, err, "Drop error: %v", err)
		_, err = coll.InsertOne(mtest.Background, bson.D{{"encryptMe", ciphertext}})
		assert.Nil(mt, err, "InsertOne error: %v", err)

		session, err := client.StartSession()
		assert.Nil(mt, err, "StartSession error: %v", err)
		sessionCtx := mongo.NewSessionContext(mtest.Background, session)

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
	numEncryptCalls              int
	numDecryptCalls              int
	numCreateDataKeyCalls        int
	numEncryptExplicitCalls      int
	numDecryptExplicitCalls      int
	numCloseCalls                int
	numBypassAutoEncryptionCalls int
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

func TestClientSideEncryptionCustomCrypt(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().MinServerVersion("4.2").Enterprise(true).CreateClient(false))
	defer mt.Close()

	kmsProvidersMap := map[string]map[string]interface{}{
		"local": {"key": localMasterKey},
	}

	mt.Run("auto encryption and decryption", func(mt *mtest.T) {
		aeOpts := options.AutoEncryption().
			SetKmsProviders(kmsProvidersMap).
			SetKeyVaultNamespace("keyvault.datakeys")
		clientOpts := options.Client().
			ApplyURI(mtest.ClusterURI()).
			SetAutoEncryptionOptions(aeOpts)
		cc := &customCrypt{}
		clientOpts.Crypt = cc
		testutil.AddTestServerAPIVersion(clientOpts)

		client, err := mongo.Connect(mtest.Background, clientOpts)
		defer client.Disconnect(mtest.Background)
		assert.Nil(mt, err, "Connect error: %v", err)

		coll := client.Database("db").Collection("coll")
		defer func() { _ = coll.Drop(mtest.Background) }()

		doc := bson.D{{"foo", "bar"}, {"ssn", mySSN}}
		_, err = coll.InsertOne(mtest.Background, doc)
		assert.Nil(mt, err, "InsertOne error: %v", err)

		res := coll.FindOne(mtest.Background, bson.D{{"foo", "bar"}})
		assert.Nil(mt, res.Err(), "FindOne error: %v", err)

		rawRes, err := res.DecodeBytes()
		assert.Nil(mt, err, "DecodeBytes error: %v", err)
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
		assert.Equal(mt, cc.numDecryptExplicitCalls, 0,
			"expected 0 calls to DecryptExplicit, got %v", cc.numDecryptExplicitCalls)
		assert.Equal(mt, cc.numCloseCalls, 0,
			"expected 0 calls to Close, got %v", cc.numCloseCalls)
		assert.Equal(mt, cc.numBypassAutoEncryptionCalls, 2,
			"expected 2 calls to BypassAutoEncryption, got %v", cc.numBypassAutoEncryptionCalls)
	})
}
