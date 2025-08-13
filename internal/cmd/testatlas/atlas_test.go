// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build atlastest
// +build atlastest

package main

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/handshake"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestAtlas(t *testing.T) {
	cases := []struct {
		name        string
		envVar      string
		certKeyFile string
		wantErr     string
	}{
		{
			name:        "Atlas with TLS",
			envVar:      "ATLAS_REPL",
			certKeyFile: "",
			wantErr:     "",
		},
		{
			name:        "Atlas with TLS and shared cluster",
			envVar:      "ATLAS_SHRD",
			certKeyFile: "",
			wantErr:     "",
		},
		{
			name:        "Atlas with free tier",
			envVar:      "ATLAS_FREE",
			certKeyFile: "",
			wantErr:     "",
		},
		{
			name:        "Atlas with TLS 1.1",
			envVar:      "ATLAS_TLS11",
			certKeyFile: "",
			wantErr:     "",
		},
		{
			name:        "Atlas with TLS 1.2",
			envVar:      "ATLAS_TLS12",
			certKeyFile: "",
			wantErr:     "",
		},
		{
			name:        "Atlas with serverless",
			envVar:      "ATLAS_SERVERLESS",
			certKeyFile: "",
			wantErr:     "",
		},
		{
			name:        "Atlas with srv file on replica set",
			envVar:      "ATLAS_SRV_REPL",
			certKeyFile: "",
			wantErr:     "",
		},
		{
			name:        "Atlas with srv file on shared cluster",
			envVar:      "ATLAS_SRV_SHRD",
			certKeyFile: "",
			wantErr:     "",
		},
		{
			name:        "Atlas with srv file on free tier",
			envVar:      "ATLAS_SRV_FREE",
			certKeyFile: "",
			wantErr:     "",
		},
		{
			name:        "Atlas with srv file on TLS 1.1",
			envVar:      "ATLAS_SRV_TLS11",
			certKeyFile: "",
			wantErr:     "",
		},
		{
			name:        "Atlas with srv file on TLS 1.2",
			envVar:      "ATLAS_SRV_TLS12",
			certKeyFile: "",
			wantErr:     "",
		},
		{
			name:        "Atlas with srv file on serverless",
			envVar:      "ATLAS_SRV_SERVERLESS",
			certKeyFile: "",
			wantErr:     "",
		},
		{
			name:        "Atlas with X509 Dev",
			envVar:      "ATLAS_X509_DEV",
			certKeyFile: createAtlasX509DevCertKeyFile(t),
			wantErr:     "",
		},
		{
			name:        "Atlas with X509 Dev no user",
			envVar:      "ATLAS_X509_DEV",
			certKeyFile: createAtlasX509DevCertKeyFileNoUser(t),
			wantErr:     "UserNotFound",
		},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s (%s)", tc.name, tc.envVar), func(t *testing.T) {
			uri := os.Getenv(tc.envVar)
			require.NotEmpty(t, uri, "Environment variable %s is not set", tc.envVar)

			if tc.certKeyFile != "" {
				uri = addTLSCertKeyFile(t, tc.certKeyFile, uri)
			}

			// Set a low server selection timeout so we fail fast if there are errors.
			clientOpts := options.Client().
				ApplyURI(uri).
				SetServerSelectionTimeout(1 * time.Second)

			// Run basic connectivity test.
			err := runTest(context.Background(), clientOpts)
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr, "expected error to contain %q", tc.wantErr)

				return
			}
			require.NoError(t, err, "error running test with TLS")

			orig := clientOpts.TLSConfig
			if orig == nil {
				orig = &tls.Config{}
			}

			insecure := orig.Clone()
			insecure.InsecureSkipVerify = true

			// Run the connectivity test with InsecureSkipVerify to ensure SNI is done
			// correctly even if verification is disabled.
			insecureClientOpts := options.Client().
				ApplyURI(uri).
				SetServerSelectionTimeout(1 * time.Second).
				SetTLSConfig(insecure)

			err = runTest(context.Background(), insecureClientOpts)
			require.NoError(t, err, "error running test with tlsInsecure")
		})
	}
}

func runTest(ctx context.Context, clientOpts *options.ClientOptions) error {
	client, err := mongo.Connect(clientOpts)
	if err != nil {
		return fmt.Errorf("Connect error: %w", err)
	}

	defer func() {
		_ = client.Disconnect(ctx)
	}()

	db := client.Database("test")
	cmd := bson.D{{handshake.LegacyHello, 1}}
	err = db.RunCommand(ctx, cmd).Err()
	if err != nil {
		return fmt.Errorf("legacy hello error: %w", err)
	}

	coll := db.Collection("test")
	if err = coll.FindOne(ctx, bson.D{{"x", 1}}).Err(); err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return fmt.Errorf("FindOne error: %w", err)
	}
	return nil
}

func createAtlasX509DevCertKeyFile(t *testing.T) string {
	t.Helper()

	b64 := os.Getenv("ATLAS_X509_DEV_CERT_BASE64")
	assert.NotEmpty(t, b64, "Environment variable ATLAS_X509_DEV_CERT_BASE64 is not set")

	certBytes, err := base64.StdEncoding.DecodeString(b64)
	require.NoError(t, err, "failed to decode ATLAS_X509_DEV_CERT_BASE64")

	certFilePath := t.TempDir() + "/atlas_x509_dev_cert.pem"

	err = os.WriteFile(certFilePath, certBytes, 0600)
	require.NoError(t, err, "failed to write ATLAS_X509_DEV_CERT_BASE64 to file")

	return certFilePath
}

func createAtlasX509DevCertKeyFileNoUser(t *testing.T) string {
	t.Helper()

	b64 := os.Getenv("ATLAS_X509_DEV_CERT_NOUSER_BASE64")
	assert.NotEmpty(t, b64, "Environment variable ATLAS_X509_DEV_CERT_NOUSER_BASE64 is not set")

	keyBytes, err := base64.StdEncoding.DecodeString(b64)
	require.NoError(t, err, "failed to decode ATLAS_X509_DEV_CERT_NOUSER_BASE64")

	keyFilePath := t.TempDir() + "/atlas_x509_dev_cert_no_user.pem"

	err = os.WriteFile(keyFilePath, keyBytes, 0600)
	require.NoError(t, err, "failed to write ATLAS_X509_DEV_CERT_NOUSER_BASE64 to file")

	return keyFilePath
}

func addTLSCertKeyFile(t *testing.T, certKeyFile, uri string) string {
	t.Helper()

	u, err := url.Parse(uri)
	require.NoError(t, err, "failed to parse uri")

	q := u.Query()
	q.Set("tlsCertificateKeyFile", filepath.ToSlash(certKeyFile))

	u.RawQuery = q.Encode()

	return u.String()
}
