// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo_test

import (
	"context"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func connect(t *testing.T, uri string) error {
	t.Helper()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if client != nil {
		require.NoError(t, client.Disconnect(context.Background()))
	}

	return err
}

// TestCredentialDisclosureProse runs the prose tests of the connection string
// specification's "Disclosure of Sensitive Values" section.
func TestCredentialDisclosureProse(t *testing.T) {
	const (
		sentinelUser     = "SENTINELuser5678"
		sentinelPassword = "SENTINELpassword1234"
		validURI         = "mongodb://" + sentinelUser + ":" + sentinelPassword + "@localhost:27017/db"
	)

	t.Run("1. redacted string representation", func(t *testing.T) {
		t.Skip("the driver does not provide a redacted connection-string representation")
	})

	// This subtest asserts only that ClientOptions.GetURI returns the
	// credentials unredacted.
	t.Run("2. explicitly named unredacted API", func(t *testing.T) {
		opts := options.Client().ApplyURI(validURI)
		require.NoError(t, opts.Validate(), "expected %q to parse", validURI)

		require.Equal(t, validURI, opts.GetURI(),
			"GetURI is documented as returning the connection string unredacted; "+
				"if that changed, update its doc comment")
	})

	t.Run("3. canonical authentication mechanism diagnostic", func(t *testing.T) {
		// A lowercase spelling of a recognized mechanism in a URI that fails
		// credential validation. The diagnostic must name the driver's
		// canonical mechanism, not the supplied spelling.
		err := connect(t,
			"mongodb://user:pw@localhost:27017/?authMechanism=mongodb-oidc&authMechanismProperties=ALLOWED_HOSTS:x")
		require.Error(t, err, "expected the OIDC credential combination to be rejected")
		require.Contains(t, err.Error(), "MONGODB-OIDC",
			"the diagnostic did not use the canonical mechanism name")
		require.NotContains(t, err.Error(), "mongodb-oidc",
			"the diagnostic copied the supplied mechanism spelling")

		// An argument that names no mechanism the driver knows must not appear.
		err = connect(t, "mongodb://localhost:27017/?authMechanism="+sentinelPassword)
		require.Error(t, err)
		require.NotContains(t, err.Error(), sentinelPassword,
			"an unrecognized authMechanism argument must not appear in its diagnostic")
	})

	t.Run("4. parsed integer diagnostic", func(t *testing.T) {
		// The driver reports only a static class of failure, so no integer is
		// included and only the prohibitions apply.
		err := connect(t, "mongodb://localhost:27017/?maxPoolSize=%2D00005")
		require.Error(t, err)
		require.NotContains(t, err.Error(), "00005")
		require.NotContains(t, err.Error(), "%2D")

		// An argument that failed integer conversion is not reported.
		err = connect(t, "mongodb://localhost:27017/?maxPoolSize="+sentinelPassword)
		require.Error(t, err)
		require.NotContains(t, err.Error(), sentinelPassword)

		// Nor is the argument of a sensitive option.
		err = connect(t, "mongodb://localhost:27017/?tlsCertificateKeyFilePassword="+sentinelPassword+"%zz")
		require.Error(t, err)
		require.NotContains(t, err.Error(), sentinelPassword)
	})

	t.Run("5. representation of a connection string that did not parse", func(t *testing.T) {
		opts := options.Client().
			ApplyURI("mongodb://" + sentinelUser + ":" + sentinelPassword + "@localhost:99999/db")

		// A client cannot be constructed from a URI that did not parse.
		client, err := mongo.Connect(opts)
		require.Error(t, err, "expected the out-of-range port to fail to parse")
		require.Nil(t, client, "expected no client for a URI that did not parse")
		require.NotContains(t, err.Error(), sentinelUser)
		require.NotContains(t, err.Error(), sentinelPassword)

		// The configuration object retains no URI, so it has nothing to expose.
		require.Empty(t, opts.GetURI(), "GetURI must not return a URI that did not parse")
	})
}

// TestValidateErrorsDoNotDiscloseURIContents covers failures reported after the
// connection string parses.
func TestValidateErrorsDoNotDiscloseURIContents(t *testing.T) {
	const (
		sentinel        = "SENTINELpassword1234"
		sentinelEncoded = "%53%45%4E%54%49%4E%45%4C%70%61%73%73%77%6F%72%64%31%32%33%34"
	)

	tests := []struct {
		name string
		uri  string
	}{
		{
			name: "unsupported auth mechanism property",
			uri: "mongodb://localhost:27017/?authMechanism=MONGODB-OIDC&authMechanismProperties=" +
				"ENVIRONMENT:test," + sentinel + ":value",
		},
		{
			name: "unsupported OIDC environment",
			uri: "mongodb://localhost:27017/?authMechanism=MONGODB-OIDC&authMechanismProperties=" +
				"ENVIRONMENT:" + sentinelEncoded,
		},
		{
			name: "invalid read preference",
			uri:  "mongodb://localhost:27017/?readPreference=" + sentinel,
		},
		{
			name: "unreadable TLS certificate key file",
			uri:  "mongodb://localhost:27017/?tlsCertificateKeyFile=" + sentinel + ".pem",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := connect(t, test.uri)
			require.Error(t, err, "expected %q to be rejected", test.uri)

			message := strings.ToLower(err.Error())
			require.NotContains(t, message, strings.ToLower(sentinel),
				"the error for %q disclosed a credential", test.uri)
			require.NotContains(t, message, strings.ToLower(sentinelEncoded),
				"the error for %q disclosed a percent-encoded credential", test.uri)
		})
	}
}
