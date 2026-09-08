// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package sfp contains connectivity tests for the Secure Forwarding Protocol
// (SFP, aka monguard), a transparent authentication proxy that sits in front of
// an Atlas cluster. The driver connects through the SFP exactly as it would to
// a normal deployment, so these tests assert only that connection,
// authentication, and basic CRUD succeed end-to-end through the proxy.
package sfp

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"os"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestSFP_Unauthenticated(t *testing.T) {
	connString := os.Getenv("SFP_ATLAS_URI")
	if connString == "" {
		t.Skip("SFP_ATLAS_URI environment variable is not set")
	}

	client, err := mongo.Connect(options.Client().ApplyURI(connString))
	require.NoError(t, err, "failed to connect to SFP deployment")
	defer func() {
		require.NoError(t, client.Disconnect(context.Background()))
	}()

	require.NoError(t, client.Ping(context.Background(), nil), "ping failed")
	requireConnStatus(t, client, false)
}

func TestSFP_Authenticated_SCRAM_SHA256(t *testing.T) {
	client := newScramSHA256Client(t)
	defer func() {
		require.NoError(t, client.Disconnect(context.Background()))
	}()

	require.NoError(t, client.Ping(context.Background(), nil), "ping failed")
	requireConnStatus(t, client, true)
	requireInsertFind(t, client)
}

func TestSFP_Authenticated_SCRAM_SHA256_WithCompressor(t *testing.T) {
	client := newScramSHA256Client(t, options.Client().SetCompressors([]string{"zstd"}))
	defer func() {
		require.NoError(t, client.Disconnect(context.Background()))
	}()

	require.NoError(t, client.Ping(context.Background(), nil), "ping failed")
	requireConnStatus(t, client, true)
	requireInsertFind(t, client)
}

func TestSFP_Authenticated_SCRAM_SHA256_WithServerAPIV1(t *testing.T) {
	client := newScramSHA256Client(t, options.Client().SetServerAPIOptions(options.ServerAPI(options.ServerAPIVersion1)))
	defer func() {
		require.NoError(t, client.Disconnect(context.Background()))
	}()

	require.NoError(t, client.Ping(context.Background(), nil), "ping failed")
	requireConnStatus(t, client, true)
	requireInsertFind(t, client)
}

func TestSFP_Authenticated_X509(t *testing.T) {
	client := newX509Client(t)
	defer func() {
		require.NoError(t, client.Disconnect(context.Background()))
	}()

	require.NoError(t, client.Ping(context.Background(), nil), "ping failed")
	requireConnStatus(t, client, true)
	requireInsertFind(t, client)
}

func TestSFP_Authenticated_X509_WithCompressor(t *testing.T) {
	client := newX509Client(t, options.Client().SetCompressors([]string{"zstd"}))
	defer func() {
		require.NoError(t, client.Disconnect(context.Background()))
	}()

	require.NoError(t, client.Ping(context.Background(), nil), "ping failed")
	requireConnStatus(t, client, true)
	requireInsertFind(t, client)
}

func TestSFP_Authenticated_X509_WithServerAPIV1(t *testing.T) {
	client := newX509Client(t, options.Client().SetServerAPIOptions(options.ServerAPI(options.ServerAPIVersion1)))
	defer func() {
		require.NoError(t, client.Disconnect(context.Background()))
	}()

	require.NoError(t, client.Ping(context.Background(), nil), "ping failed")
	requireConnStatus(t, client, true)
	requireInsertFind(t, client)
}

// =============================================================================
// Test Runner Helpers
// =============================================================================

func requireConnStatus(t *testing.T, client *mongo.Client, auth bool) {
	t.Helper()

	// Execute a `connectionStatus` command against the `admin` database.
	result := client.Database("admin").RunCommand(context.Background(), bson.D{{Key: "connectionStatus", Value: 1}})
	require.NoError(t, result.Err(), "connectionStatus command failed")

	doc := struct {
		Ok       float64 `bson:"ok"`
		AuthInfo struct {
			AuthenticatedUsers bson.A `bson:"authenticatedUsers"`
		}
	}{}
	require.NoError(t, result.Decode(&doc), "failed to decode connectionStatus result")

	// Assert that the command succeeds with `ok: 1`
	require.Equal(t, 1.0, doc.Ok, "connectionStatus command returned ok != 1")

	// If authenticated, assert that `authInfo.authenticatedUsers` contains at least one user
	if auth {
		require.Greater(t, len(doc.AuthInfo.AuthenticatedUsers), 0, "no authenticated users found when expected")
	} else {
		require.Equal(t, 0, len(doc.AuthInfo.AuthenticatedUsers), "authenticated users found when none expected")
	}
}

func requireInsertFind(t *testing.T, client *mongo.Client) {
	t.Helper()

	// Insert a document into a test collection and assert the insert succeeds
	coll := client.Database("test").Collection("sfp_test")
	defer func() { require.NoError(t, coll.Drop(context.Background()), "failed to drop test collection") }()

	_, err := coll.InsertOne(context.Background(), bson.D{{Key: "name", Value: "test"}})
	require.NoError(t, err, "insert failed")

	// Query the collection using `find` and assert the inserted document is returned
	result := coll.FindOne(context.Background(), bson.D{{Key: "name", Value: "test"}})
	require.NoError(t, result.Err(), "find failed")

	doc := struct {
		Name string `bson:"name"`
	}{}
	require.NoError(t, result.Decode(&doc), "failed to decode find result")
	require.Equal(t, "test", doc.Name, "find returned unexpected document")
}

func newScramSHA256Client(t *testing.T, extOpts ...*options.ClientOptions) *mongo.Client {
	t.Helper()

	connString := os.Getenv("SFP_ATLAS_URI")
	if connString == "" {
		t.Skip("SFP_ATLAS_URI environment variable is not set")
	}

	user := os.Getenv("SFP_ATLAS_USER")
	if user == "" {
		t.Skip("SFP_ATLAS_USER environment variable is not set")
	}

	password := os.Getenv("SFP_ATLAS_PASSWORD")
	if password == "" {
		t.Skip("SFP_ATLAS_PASSWORD environment variable is not set")
	}

	baseOpts := options.Client().
		ApplyURI(connString).
		SetAuth(options.Credential{
			AuthMechanism: "SCRAM-SHA-256",
			Username:      user,
			Password:      password,
			AuthSource:    "admin",
		})

	allOpts := append([]*options.ClientOptions{baseOpts}, extOpts...)

	client, err := mongo.Connect(options.MergeClientOptions(allOpts...))
	require.NoError(t, err, "failed to connect to SFP deployment with SCRAM-SHA-256 credentials")

	return client
}

func newX509Client(t *testing.T, extOpts ...*options.ClientOptions) *mongo.Client {
	t.Helper()

	connString := os.Getenv("SFP_ATLAS_X509_URI")
	if connString == "" {
		t.Skip("SFP_ATLAS_X509_URI environment variable is not set")
	}

	certPEMBase64 := os.Getenv("SFP_ATLAS_X509_BASE64")
	if certPEMBase64 == "" {
		t.Skip("SFP_ATLAS_X509_BASE64 environment variable is not set")
	}

	certPEM, err := base64.StdEncoding.DecodeString(certPEMBase64)
	require.NoError(t, err, "failed to base64-decode SFP_ATLAS_X509_BASE64")

	cert, err := tls.X509KeyPair(certPEM, certPEM)
	require.NoError(t, err, "failed to parse X.509 client certificate and key")

	baseOpts := options.Client().
		ApplyURI(connString).
		SetAuth(options.Credential{
			AuthMechanism: "MONGODB-X509",
			AuthSource:    "$external",
		}).
		SetTLSConfig(&tls.Config{Certificates: []tls.Certificate{cert}})

	allOpts := append([]*options.ClientOptions{baseOpts}, extOpts...)

	client, err := mongo.Connect(options.MergeClientOptions(allOpts...))
	require.NoError(t, err, "failed to connect to SFP deployment with X.509 credentials")

	return client
}
