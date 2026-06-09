// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build cse

package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/integtest"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

type cseProse27Config struct {
	encryptedFieldsPrefixSuffixCIDI bson.Raw
	encryptedFieldsSubstringCIDI    bson.Raw
	key1Document                    bson.Raw
}

type cseProse27Test struct {
	keyVaultClient        *mongo.Client
	clientEncryption      *mongo.ClientEncryption
	explicitEncryptClient *mongo.Client
	autoEncryptClient     *mongo.Client
	key1ID                bson.Binary
}

const cseSpecDataDir = "../../testdata/specifications/source/client-side-encryption/etc/data"

func TestClientSideEncryptionProse_27(t *testing.T) {
	mt := newCSE_T(t, newQEOpts().MinServerVersion("8.2"))
	mt.Setup()

	test := setupCSEProse27(mt)

	mt.Run("case 8: can find an auto-encrypted case indexed document by prefix and suffix", func(mt *mtest.T) {
		runCSEProse27Case8(mt, test)
	})
	mt.Run("case 9: can find an auto-encrypted diacritic-insensitively indexed document by prefix and suffix", func(mt *mtest.T) {
		runCSEProse27Case9(mt, test)
	})
	mt.Run("case 10: can find an auto-encrypted case-insensitively indexed document by substring", func(mt *mtest.T) {
		runCSEProse27Case10(mt, test)
	})
	mt.Run("case 11: can find an auto-encrypted diacritic-insensitively indexed document by substring", func(mt *mtest.T) {
		runCSEProse27Case11(mt, test)
	})
}

// =============================================================================
// Test Runners
// =============================================================================

// runCSEProse27Case8 ensures that we can find an auto-encrypted case indexed
// document by prefix and suffix.
//
// Requires server 9.0+ and libmongocrypt 1.19.0+.
//
// TODO(GODRIVER-3943): Add 1.19.0 min guards.
func runCSEProse27Case8(mt *mtest.T, test *cseProse27Test) {
	mt.Helper()

	if mtest.CompareServerVersions(mtest.ServerVersion(), "9.0") < 0 {
		mt.Skip("prefix and suffix query types require server 9.0+")
	}

	const collName = "prefix-suffix-ci-di"
	const dbName = "db"

	// Step 1. Use autoEncryptedClient to insert { "encryptedText": "BingQiLin" }
	// into db.prefix-suffix-ci-di with majority write concern.
	insertEncryptedText(mt, test.autoEncryptClient, dbName, collName, "BingQiLin")

	// Step 2. Use clientEncryption.encrypt() to encrypt "bing" with prefix query
	// type.
	encryptOpts := options.Encrypt().
		SetKeyID(test.key1ID).
		SetAlgorithm("String").
		SetQueryType("prefix").
		SetContentionFactor(0).
		SetTextOptions(options.Text().
			SetCaseSensitive(false).
			SetDiacriticSensitive(false).
			SetPrefix(options.PrefixOptions{StrMinQueryLength: 2, StrMaxQueryLength: 10}))

	bingPlaintextSearchToken := bson.RawValue{Type: bson.TypeString, Value: bsoncore.AppendString(nil, "bing")}

	bingEncSearchToken, err := test.clientEncryption.Encrypt(context.Background(), bingPlaintextSearchToken, encryptOpts)
	require.Nil(mt, err, "error encrypting 'bing': %v", err)

	// Step 3. Use explicitEncryptedClient to run a "find" operation on the
	// db.prefix-suffix-ci-di collection with the following filter:
	//
	// { $expr: { $encStrStartsWith: {input: '$encryptedText', prefix: <encrypted 'bing'>} } }
	bingFilter := bson.D{{Key: "$expr", Value: bson.D{{Key: "$encStrStartsWith", Value: bson.D{
		{Key: "input", Value: "$encryptedText"},
		{Key: "prefix", Value: bingEncSearchToken},
	}}}}}

	encColl := test.explicitEncryptClient.Database(dbName).Collection(collName)

	bingCur, err := encColl.Find(context.Background(), bingFilter)
	require.Nil(mt, err, "error running find on %s.%s: %v", dbName, collName, err)

	var results []bson.M
	err = bingCur.All(context.Background(), &results)
	require.Nil(mt, err, "error decoding find results: %v", err)

	require.Len(mt, results, 1, "expected 1 result, got %d", len(results))
	require.Equal(mt, "BingQiLin", results[0]["encryptedText"])

	// Step 4. Use clientEncryption.encrypt() to encrypt "lin" with suffix query type.
	encryptOpts = options.Encrypt().
		SetKeyID(test.key1ID).
		SetAlgorithm("String").
		SetQueryType("suffix").
		SetContentionFactor(0).
		SetTextOptions(options.Text().
			SetCaseSensitive(false).
			SetDiacriticSensitive(false).
			SetSuffix(options.SuffixOptions{StrMinQueryLength: 2, StrMaxQueryLength: 10}))

	linPlaintextSearchToken := bson.RawValue{Type: bson.TypeString, Value: bsoncore.AppendString(nil, "lin")}

	linEncSearchToken, err := test.clientEncryption.Encrypt(context.Background(), linPlaintextSearchToken, encryptOpts)
	require.Nil(mt, err, "error encrypting 'lin': %v", err)

	// Step 5. Use explicitEncryptedClient to run a "find" operation on the
	// db.prefix-suffix-ci-di collection with the following filter:
	//
	// { $expr: { $encStrEndsWith: {input: '$encryptedText', suffix: <encrypted 'lin'>} } }
	linFilter := bson.D{{Key: "$expr", Value: bson.D{{Key: "$encStrEndsWith", Value: bson.D{
		{Key: "input", Value: "$encryptedText"},
		{Key: "suffix", Value: linEncSearchToken},
	}}}}}

	linCur, err := encColl.Find(context.Background(), linFilter)
	require.Nil(mt, err, "error running find on %s.%s: %v", dbName, collName, err)

	err = linCur.All(context.Background(), &results)
	require.Nil(mt, err, "error decoding find results: %v", err)

	require.Len(mt, results, 1, "expected 1 result, got %d", len(results))
	require.Equal(mt, "BingQiLin", results[0]["encryptedText"])
}

// runCSEProse27Case9 ensures that we can find an auto-encrypted
// diacritic-insensitively indexed document by prefix and suffix.
//
// Requires server 9.0+ and libmongocrypt 1.19.0+.
//
// TODO(GODRIVER-3943): Add 1.19.0 min guards.
func runCSEProse27Case9(mt *mtest.T, test *cseProse27Test) {
	mt.Helper()

	if mtest.CompareServerVersions(mtest.ServerVersion(), "9.0") < 0 {
		mt.Skip("prefix and suffix query types require server 9.0+")
	}

	const collName = "prefix-suffix-ci-di"
	const dbName = "db"

	// Step 1. Use autoEncryptedClient to insert { "encryptedText": "cafébarbäz" }
	// into db.prefix-suffix-ci-di with majority write concern.
	insertEncryptedText(mt, test.autoEncryptClient, dbName, collName, "cafébarbäz")

	// Step 2. Use clientEncryption.encrypt() to encrypt "cafe" with prefix query
	// type.
	encryptOpts := options.Encrypt().
		SetKeyID(test.key1ID).
		SetAlgorithm("String").
		SetQueryType("prefix").
		SetContentionFactor(0).
		SetTextOptions(options.Text().
			SetCaseSensitive(false).
			SetDiacriticSensitive(false).
			SetPrefix(options.PrefixOptions{StrMinQueryLength: 2, StrMaxQueryLength: 10}))

	cafePlaintextSearchToken := bson.RawValue{Type: bson.TypeString, Value: bsoncore.AppendString(nil, "cafe")}

	cafeEncSearchToken, err := test.clientEncryption.Encrypt(context.Background(), cafePlaintextSearchToken, encryptOpts)
	require.Nil(mt, err, "error encrypting 'cafe': %v", err)

	// Step 3. Use explicitEncryptedClient to run a "find" operation on the
	// db.prefix-suffix-ci-di collection with the following filter:
	//
	// { $expr: { $encStrStartsWith: {input: '$encryptedText', prefix: <encrypted 'cafe'>} } }
	cafeFilter := bson.D{{Key: "$expr", Value: bson.D{{Key: "$encStrStartsWith", Value: bson.D{
		{Key: "input", Value: "$encryptedText"},
		{Key: "prefix", Value: cafeEncSearchToken},
	}}}}}

	encColl := test.explicitEncryptClient.Database(dbName).Collection(collName)

	cafeCur, err := encColl.Find(context.Background(), cafeFilter)
	require.Nil(mt, err, "error running find on %s.%s: %v", dbName, collName, err)

	var results []bson.M

	err = cafeCur.All(context.Background(), &results)
	require.Nil(mt, err, "error decoding find results: %v", err)

	require.Len(mt, results, 1, "expected 1 result, got %d", len(results))
	require.Equal(mt, "cafébarbäz", results[0]["encryptedText"])

	// Step 4. Use clientEncryption.encrypt() to encrypt "baz" with suffix query
	// type.
	encryptOpts = options.Encrypt().
		SetKeyID(test.key1ID).
		SetAlgorithm("String").
		SetQueryType("suffix").
		SetContentionFactor(0).
		SetTextOptions(options.Text().
			SetCaseSensitive(false).
			SetDiacriticSensitive(false).
			SetSuffix(options.SuffixOptions{StrMinQueryLength: 2, StrMaxQueryLength: 10}))

	bazPlaintextSearchToken := bson.RawValue{Type: bson.TypeString, Value: bsoncore.AppendString(nil, "baz")}

	bazEncSearchToken, err := test.clientEncryption.Encrypt(context.Background(), bazPlaintextSearchToken, encryptOpts)
	require.Nil(mt, err, "error encrypting 'baz': %v", err)

	// Step 5. Use explicitEncryptedClient to run a "find" operation on the
	// db.prefix-suffix-ci-di collection with the following filter:
	//
	// { $expr: { $encStrEndsWith: {input: '$encryptedText', suffix: <encrypted 'baz'>} } }
	bazFilter := bson.D{{Key: "$expr", Value: bson.D{{Key: "$encStrEndsWith", Value: bson.D{
		{Key: "input", Value: "$encryptedText"},
		{Key: "suffix", Value: bazEncSearchToken},
	}}}}}

	bazCur, err := encColl.Find(context.Background(), bazFilter)
	require.Nil(mt, err, "error running find on %s.%s: %v", dbName, collName, err)

	err = bazCur.All(context.Background(), &results)
	require.Nil(mt, err, "error decoding find results: %v", err)

	require.Len(mt, results, 1, "expected 1 result, got %d", len(results))
	require.Equal(mt, "cafébarbäz", results[0]["encryptedText"])
}

// runCSEProse27Case10 ensures that we can find an auto-encrypted
// case-insensitively indexed document by substring.
//
// Requires libmongocrypt 1.19.0+.
//
// TODO(GODRIVER-3943): Add 1.19.0 min guards.
func runCSEProse27Case10(mt *mtest.T, test *cseProse27Test) {
	mt.Helper()

	const collName = "substring-ci-di"
	const dbName = "db"

	// Step 1. Use autoEncryptedClient to insert { "encryptedText": "FooBarBaz" }
	// into db.substring-ci-di with majority write concern.
	insertEncryptedText(mt, test.autoEncryptClient, dbName, collName, "FooBarBaz")

	// Step 2. Use clientEncryption.encrypt() to encrypt "bar" with substring
	// query type. Substring search is still a preview feature, so the query type
	// remains "substringPreview" even on libmongocrypt 1.19.0+.
	encryptOpts := options.Encrypt().
		SetKeyID(test.key1ID).
		SetAlgorithm("String").
		SetQueryType("substringPreview").
		SetContentionFactor(0).
		SetTextOptions(options.Text().
			SetCaseSensitive(false).
			SetDiacriticSensitive(false).
			SetSubstring(options.SubstringOptions{StrMaxLength: 10, StrMinQueryLength: 2, StrMaxQueryLength: 10}))

	barPlaintextSearchToken := bson.RawValue{Type: bson.TypeString, Value: bsoncore.AppendString(nil, "bar")}

	barEncSearchToken, err := test.clientEncryption.Encrypt(context.Background(), barPlaintextSearchToken, encryptOpts)
	require.Nil(mt, err, "error encrypting 'bar': %v", err)

	// Step 3. Use explicitEncryptedClient to run a "find" operation on the
	// db.substring-ci-di collection with the following filter:
	//
	// { $expr: { $encStrContains: {input: '$encryptedText', substring: <encrypted 'bar'>} } }
	barFilter := bson.D{{Key: "$expr", Value: bson.D{{Key: "$encStrContains", Value: bson.D{
		{Key: "input", Value: "$encryptedText"},
		{Key: "substring", Value: barEncSearchToken},
	}}}}}

	encColl := test.explicitEncryptClient.Database(dbName).Collection(collName)

	barCur, err := encColl.Find(context.Background(), barFilter)
	require.Nil(mt, err, "error running find on %s.%s: %v", dbName, collName, err)

	var results []bson.M

	err = barCur.All(context.Background(), &results)
	require.Nil(mt, err, "error decoding find results: %v", err)

	require.Len(mt, results, 1, "expected 1 result, got %d", len(results))
	require.Equal(mt, "FooBarBaz", results[0]["encryptedText"])
}

// runCSEProse27Case11 ensures that we can find an auto-encrypted
// diacritic-insensitively indexed document by substring.
//
// Requires libmongocrypt 1.19.0+.
//
// TODO(GODRIVER-3943): Add 1.19.0 min guards.
func runCSEProse27Case11(mt *mtest.T, test *cseProse27Test) {
	mt.Helper()

	const collName = "substring-ci-di"
	const dbName = "db"

	// Step 1. Use autoEncryptedClient to insert { "encryptedText": "foocafébaz" }
	// into db.substring-ci-di with majority write concern.
	insertEncryptedText(mt, test.autoEncryptClient, dbName, collName, "foocafébaz")

	// Step 2. Use clientEncryption.encrypt() to encrypt "cafe" with substring query
	// type. Substring search is still a preview feature, so the query type remains
	// "substringPreview" even on libmongocrypt 1.19.0+.
	encryptOpts := options.Encrypt().
		SetKeyID(test.key1ID).
		SetAlgorithm("String").
		SetQueryType("substringPreview").
		SetContentionFactor(0).
		SetTextOptions(options.Text().
			SetCaseSensitive(false).
			SetDiacriticSensitive(false).
			SetSubstring(options.SubstringOptions{StrMaxLength: 10, StrMinQueryLength: 2, StrMaxQueryLength: 10}))

	cafePlaintextSearchToken := bson.RawValue{Type: bson.TypeString, Value: bsoncore.AppendString(nil, "cafe")}

	cafeEncSearchToken, err := test.clientEncryption.Encrypt(context.Background(), cafePlaintextSearchToken, encryptOpts)
	require.Nil(mt, err, "error encrypting 'cafe': %v", err)

	// Step 3. Use explicitEncryptedClient to run a "find" operation on the
	// db.substring-ci-di collection with the following filter:
	//
	// { $expr: { $encStrContains: {input: '$encryptedText', substring: <encrypted 'cafe'>} } }
	cafeFilter := bson.D{{Key: "$expr", Value: bson.D{{Key: "$encStrContains", Value: bson.D{
		{Key: "input", Value: "$encryptedText"},
		{Key: "substring", Value: cafeEncSearchToken},
	}}}}}

	encColl := test.explicitEncryptClient.Database(dbName).Collection(collName)

	cafeCur, err := encColl.Find(context.Background(), cafeFilter)
	require.Nil(mt, err, "error running find on %s.%s: %v", dbName, collName, err)

	var results []bson.M

	err = cafeCur.All(context.Background(), &results)
	require.Nil(mt, err, "error decoding find results: %v", err)

	require.Len(mt, results, 1, "expected 1 result, got %d", len(results))
	require.Equal(mt, "foocafébaz", results[0]["encryptedText"])
}

// =============================================================================
// Test Runner Helpers
// =============================================================================

// insertEncryptedText inserts { "encryptedText": <text> } into db.<coll> via
// the given client with majority write concern.
func insertEncryptedText(mt *mtest.T, client *mongo.Client, dbName, collName, text string) {
	mt.Helper()

	collOpts := options.Collection().SetWriteConcern(mtest.MajorityWc)
	coll := client.Database(dbName).Collection(collName, collOpts)

	_, err := coll.InsertOne(context.Background(), bson.D{{Key: "encryptedText", Value: text}})
	require.Nil(mt, err, "error inserting document into %s.%s: %v", dbName, collName, err)
}

// =============================================================================
// Setup
// =============================================================================

// loadCSEProse27Config loads the JSON fixtures required by prose test 27.
func loadCSEProse27Config() (cseProse27Config, error) {
	load := func(path string) (bson.Raw, error) {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var doc bson.Raw
		if err := bson.UnmarshalExtJSON(content, true, &doc); err != nil {
			return nil, err
		}
		return doc, nil
	}

	var cfg cseProse27Config
	files := []struct {
		path string
		dest *bson.Raw
	}{
		{filepath.Join(cseSpecDataDir, "encryptedFields-prefix-suffix-ci-di.json"), &cfg.encryptedFieldsPrefixSuffixCIDI},
		{filepath.Join(cseSpecDataDir, "encryptedFields-substring-ci-di.json"), &cfg.encryptedFieldsSubstringCIDI},
		{filepath.Join(cseSpecDataDir, "keys", "key1-document.json"), &cfg.key1Document},
	}

	for _, f := range files {
		doc, err := load(f.path)
		if err != nil {
			return cseProse27Config{}, err
		}
		*f.dest = doc
	}

	return cfg, nil
}

func doSetupCSEProse27(mt *mtest.T) (*cseProse27Test, error) {
	cfg, err := loadCSEProse27Config()
	if err != nil {
		return nil, err
	}

	test := &cseProse27Test{}

	// Using QE CreateCollection() and Collection.Drop(), drop and create the
	// test collections with majority write concern. Only the case- and
	// diacritic-insensitive (ci-di) collections are needed by cases 8-11.
	db := mt.Client.Database("db")
	encryptedColls := []struct {
		name         string
		fields       bson.Raw
		minServerVer string
	}{
		// prefix and suffix query types require server 9.0+.
		{"prefix-suffix-ci-di", cfg.encryptedFieldsPrefixSuffixCIDI, "9.0"},
		{"substring-ci-di", cfg.encryptedFieldsSubstringCIDI, ""},
	}
	for _, c := range encryptedColls {
		// Always drop to ensure a clean state from any previous run.
		mtest.DropEncryptedCollection(mt, db.Collection(c.name), c.fields)

		// Skip creating collections that require a higher server version.
		if c.minServerVer != "" && mtest.CompareServerVersions(mtest.ServerVersion(), c.minServerVer) < 0 {
			continue
		}

		cco := options.CreateCollection().SetEncryptedFields(c.fields)
		if err := db.CreateCollection(context.Background(), c.name, cco); err != nil {
			return nil, fmt.Errorf("error creating db.%s: %w", c.name, err)
		}
	}

	// Drop and create keyvault.datakeys with majority write concern.
	keyVaultDB := mt.Client.Database("keyvault")
	keyVaultDropCollOpts := options.Collection().SetWriteConcern(mtest.MajorityWc)

	if err = keyVaultDB.Collection("datakeys", keyVaultDropCollOpts).Drop(context.Background()); err != nil {
		return nil, fmt.Errorf("error dropping keyvault.datakeys: %w", err)
	}

	if err = keyVaultDB.CreateCollection(context.Background(), "datakeys"); err != nil {
		return nil, fmt.Errorf("error creating keyvault.datakeys: %w", err)
	}

	// Insert key1Document in keyvault.datakeys with majority write concern.
	keyVaultInsertCollOpts := options.Collection().SetWriteConcern(mtest.MajorityWc)
	keyVaultColl := keyVaultDB.Collection("datakeys", keyVaultInsertCollOpts)

	if _, err = keyVaultColl.InsertOne(context.Background(), cfg.key1Document); err != nil {
		return nil, fmt.Errorf("error inserting key1Document into keyvault.datakeys: %w", err)
	}

	// Create a MongoClient named keyVaultClient.
	keyVaultClientOpts := options.Client().ApplyURI(mtest.ClusterURI())
	integtest.AddTestServerAPIVersion(keyVaultClientOpts)

	if test.keyVaultClient, err = mongo.Connect(keyVaultClientOpts); err != nil {
		return nil, fmt.Errorf("error connecting keyVaultClient: %w", err)
	}
	mt.Cleanup(func() { _ = test.keyVaultClient.Disconnect(context.Background()) })

	// Create a ClientEncryption object named clientEncryption.
	ceo := options.ClientEncryption().
		SetKeyVaultNamespace(kvNamespace).
		SetKmsProviders(map[string]map[string]any{
			"local": {"key": localMasterKey},
		})

	if test.clientEncryption, err = mongo.NewClientEncryption(test.keyVaultClient, ceo); err != nil {
		return nil, fmt.Errorf("error creating clientEncryption: %w", err)
	}
	mt.Cleanup(func() { _ = test.clientEncryption.Close(context.Background()) })

	// Create a MongoClient named explicitEncryptedClient with bypassQueryAnalysis: true.
	explicitAEO := options.AutoEncryption().
		SetKeyVaultNamespace(kvNamespace).
		SetKmsProviders(map[string]map[string]any{
			"local": {"key": localMasterKey},
		}).
		SetBypassQueryAnalysis(true)

	explicitEncryptClientOpts := options.Client().
		ApplyURI(mtest.ClusterURI()).
		SetAutoEncryptionOptions(explicitAEO)
	integtest.AddTestServerAPIVersion(explicitEncryptClientOpts)

	if test.explicitEncryptClient, err = mongo.Connect(explicitEncryptClientOpts); err != nil {
		return nil, fmt.Errorf("error connecting explicitEncryptedClient: %w", err)
	}
	mt.Cleanup(func() { _ = test.explicitEncryptClient.Disconnect(context.Background()) })

	// Create a MongoClient named autoEncryptedClient with auto-encryption enabled.
	// Use the crypt_shared library for query analysis when CRYPT_SHARED_LIB_PATH
	// is set; otherwise the driver spawns mongocryptd, which must be new enough to
	// recognize the prefix/suffix query types.
	autoAEO := options.AutoEncryption().
		SetKeyVaultNamespace(kvNamespace).
		SetKmsProviders(map[string]map[string]any{
			"local": {"key": localMasterKey},
		}).
		SetExtraOptions(getCryptSharedLibExtraOptions())

	autoEncryptClientOpts := options.Client().
		ApplyURI(mtest.ClusterURI()).
		SetAutoEncryptionOptions(autoAEO)
	integtest.AddTestServerAPIVersion(autoEncryptClientOpts)

	if test.autoEncryptClient, err = mongo.Connect(autoEncryptClientOpts); err != nil {
		return nil, fmt.Errorf("error connecting autoEncryptedClient: %w", err)
	}
	mt.Cleanup(func() { _ = test.autoEncryptClient.Disconnect(context.Background()) })

	// Read the "_id" field of key1Document as key1ID, used by the test runners
	// to build explicit-encryption search tokens.
	subtype, data := cfg.key1Document.Lookup("_id").Binary()
	test.key1ID = bson.Binary{Subtype: subtype, Data: data}

	return test, nil
}

func setupCSEProse27(mt *mtest.T) *cseProse27Test {
	mt.Helper()

	// doSetupCSEProse27 registers cleanup for each client as it is created, so a
	// partial failure still tears down whatever was already connected.
	test, err := doSetupCSEProse27(mt)
	require.Nil(mt, err, "failed to set up CSE prose 27: %v", err)

	return test
}
