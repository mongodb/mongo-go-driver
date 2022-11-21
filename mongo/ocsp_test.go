// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"crypto/tls"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func TestOCSP(t *testing.T) {
	successEnvVar := os.Getenv("OCSP_TLS_SHOULD_SUCCEED")
	if successEnvVar == "" {
		t.Skip("skipping because OCSP_TLS_SHOULD_SUCCEED not set")
	}
	shouldSucceed, err := strconv.ParseBool(successEnvVar)
	assert.Nil(t, err, "invalid value for OCSP_TLS_SHOULD_SUCCEED; expected true or false, got %v", successEnvVar)

	cs := testutil.ConnString(t)

	t.Run("tls", func(t *testing.T) {
		clientOpts := createOCSPClientOptions(cs.Original)
		client, err := Connect(bgCtx, clientOpts)
		assert.Nil(t, err, "Connect error: %v", err)
		defer client.Disconnect(bgCtx)

		err = client.Ping(bgCtx, readpref.Primary())
		if shouldSucceed {
			assert.Nil(t, err, "Ping error: %v", err)
			return
		}
		// Log the error we got so it's visible in Evergreen and we can verify the tests are running as expected there.
		t.Logf("got Ping error: %v\n", err)
		assert.NotNil(t, err, "expected Ping error, got nil")
	})
	t.Run("tlsInsecure", func(t *testing.T) {
		clientOpts := createInsecureOCSPClientOptions(cs.Original)
		client, err := Connect(bgCtx, clientOpts)
		assert.Nil(t, err, "Connect error: %v", err)
		defer client.Disconnect(bgCtx)

		err = client.Ping(bgCtx, readpref.Primary())
		assert.Nil(t, err, "Ping error: %v", err)
	})
}

func createOCSPClientOptions(uri string) *options.ClientOptions {
	opts := options.Client().ApplyURI(uri)

	timeout := 500 * time.Millisecond
	if runtime.GOOS == "windows" {
		// Non-stapled OCSP endpoint checks are slow on Windows.
		timeout = 5 * time.Second
	}
	opts.SetServerSelectionTimeout(timeout)
	return opts
}

func createInsecureOCSPClientOptions(uri string) *options.ClientOptions {
	opts := createOCSPClientOptions(uri)

	if opts.TLSConfig != nil {
		opts.TLSConfig.InsecureSkipVerify = true
		return opts
	}
	return opts.SetTLSConfig(&tls.Config{
		InsecureSkipVerify: true,
	})
}
