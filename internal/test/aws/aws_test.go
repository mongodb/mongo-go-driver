// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// staticAWSProvider returns fixed credentials on every call.
type staticAWSProvider struct {
	creds struct {
		AccessKeyID     string
		SecretAccessKey string
		SessionToken    string
		Source          string
		CanExpire       bool
		Expires         time.Time
		AccountID       string
	}
}

func (p *staticAWSProvider) Retrieve(_ context.Context) (struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Source          string
	CanExpire       bool
	Expires         time.Time
	AccountID       string
}, error,
) {
	return p.creds, nil
}

// trackingAWSProvider wraps another provider and counts invocations.
type trackingAWSProvider struct {
	inner  options.AWSCredentialsProvider
	called int
}

func (p *trackingAWSProvider) Retrieve(ctx context.Context) (struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Source          string
	CanExpire       bool
	Expires         time.Time
	AccountID       string
}, error,
) {
	p.called++
	return p.inner.Retrieve(ctx)
}

// requireAWSAuthSupport skips t if the server at the given URI does not support
// MONGODB-AWS (introduced in MongoDB 4.4, maxWireVersion 9). The check is performed
// without auth so that it works even when the URI carries AWS credentials.
// The original URI's TLS and topology settings are preserved so that the probe
// connection is valid on TLS-required servers.
func requireAWSAuthSupport(t *testing.T, rawURI string) {
	t.Helper()
	// Reuse all options from the original URI (TLS, replica set, etc.) but clear
	// auth so hello can be sent without triggering AWS credential negotiation.
	checkOpts := options.Client().ApplyURI(rawURI)
	checkOpts.Auth = nil
	client, err := mongo.Connect(checkOpts)
	if err != nil {
		return // cannot probe; let the real connection surface any error
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	var result struct {
		MaxWireVersion int32 `bson:"maxWireVersion"`
	}
	if err := client.Database("admin").RunCommand(context.Background(), bson.D{{Key: "hello", Value: 1}}).Decode(&result); err != nil {
		return
	}
	if result.MaxWireVersion < 9 {
		t.Skip("MONGODB-AWS requires MongoDB 4.4+ (maxWireVersion 9)")
	}
}

func awsFindOne(t *testing.T, client *mongo.Client) {
	t.Helper()
	err := client.Database("aws").Collection("test").FindOne(context.Background(), bson.D{{Key: "x", Value: 1}}).Err()
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("unexpected FindOne error: %v", err)
	}
}

func TestAWS(t *testing.T) {
	if os.Getenv("AWS_TEST") == "" {
		t.Skip("Skipping test: AWS_TEST environment variable is not set")
	}

	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		t.Skip("Skipping test: MONGODB_URI environment variable is not set")
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))

	defer func() {
		err = client.Disconnect(context.Background())
		require.NoError(t, err)
	}()

	awsFindOne(t, client)
}

// Custom Credential Provider Authenticates
//
// Scenarios where MONGODB_URI contains no inline credentials (EC2, ECS, WebIdentity)
// are skipped because a valid custom provider for those environments requires the AWS
// SDK, which is not available in this module.
func TestAWSCustomCredentialProviderAuthenticates(t *testing.T) {
	if os.Getenv("AWS_TEST") == "" {
		t.Skip("Skipping test: AWS_TEST environment variable is not set")
	}

	rawURI := os.Getenv("MONGODB_URI")
	if rawURI == "" {
		t.Skip("MONGODB_URI not set")
	}

	// Extract inline credentials from the URI, if any. For scenarios where the CI
	// tooling embeds credentials in the URI (Regular, AssumeRole, Lambda), we return
	// them directly from the custom provider rather than using the AWS SDK default
	// provider, per the spec allowance.
	parsedOpts := options.Client().ApplyURI(rawURI)
	if parsedOpts.Auth == nil || parsedOpts.Auth.Username == "" {
		t.Skip("no inline credentials in MONGODB_URI; scenario requires AWS SDK default provider")
	}
	requireAWSAuthSupport(t, rawURI)

	creds := struct {
		AccessKeyID     string
		SecretAccessKey string
		SessionToken    string
		Source          string
		CanExpire       bool
		Expires         time.Time
		AccountID       string
	}{
		AccessKeyID:     parsedOpts.Auth.Username,
		SecretAccessKey: parsedOpts.Auth.Password,
	}
	if parsedOpts.Auth.AuthMechanismProperties != nil {
		creds.SessionToken = parsedOpts.Auth.AuthMechanismProperties["AWS_SESSION_TOKEN"]
	}

	tracking := &trackingAWSProvider{inner: &staticAWSProvider{creds: creds}}

	client, err := mongo.Connect(
		options.Client().
			ApplyURI(rawURI).
			SetAuth(options.Credential{
				AuthMechanism:          "MONGODB-AWS",
				AWSCredentialsProvider: tracking,
			}),
	)
	require.NoError(t, err)
	defer func() { require.NoError(t, client.Disconnect(context.Background())) }()

	awsFindOne(t, client)
	assert.Greater(t, tracking.called, 0, "expected custom credential provider to be called at least once")
}

// Custom Credential Provider Authentication Precedence — Custom Provider Takes
// Precedence Over Environment Variables
func TestAWSCustomCredentialProviderPrecedence(t *testing.T) {
	if os.Getenv("AWS_TEST") == "" {
		t.Skip("Skipping test: AWS_TEST environment variable is not set")
	}

	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	if accessKeyID == "" || secretAccessKey == "" {
		t.Skip("AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY must be set")
	}

	rawURI := os.Getenv("MONGODB_URI")
	if rawURI == "" {
		t.Skip("MONGODB_URI not set")
	}
	// Ensure the URI targets a server configured for MONGODB-AWS; without this
	// guard the test runs (and fails) on any 4.4+ server where AWS env vars
	// happen to be set in the developer's shell.
	if uriOpts := options.Client().ApplyURI(rawURI); uriOpts.Auth == nil ||
		uriOpts.Auth.AuthMechanism != "MONGODB-AWS" {
		t.Skip("MONGODB_URI does not specify MONGODB-AWS authentication mechanism")
	}
	requireAWSAuthSupport(t, rawURI)

	// The custom provider supplies credentials drawn from the same env vars that the
	// built-in env-var provider would read. Verifying that the custom provider is called
	// confirms it takes precedence in the credential chain (static -> custom -> env vars).
	creds := struct {
		AccessKeyID     string
		SecretAccessKey string
		SessionToken    string
		Source          string
		CanExpire       bool
		Expires         time.Time
		AccountID       string
	}{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		SessionToken:    os.Getenv("AWS_SESSION_TOKEN"),
	}
	tracking := &trackingAWSProvider{inner: &staticAWSProvider{creds: creds}}

	// SetAuth overrides any inline credentials that ApplyURI may have parsed from the
	// URI, so the chain is: static(empty) -> custom provider -> env-var provider.
	client, err := mongo.Connect(
		options.Client().
			ApplyURI(rawURI).
			SetAuth(options.Credential{
				AuthMechanism:          "MONGODB-AWS",
				AWSCredentialsProvider: tracking,
			}),
	)
	require.NoError(t, err)
	defer func() { require.NoError(t, client.Disconnect(context.Background())) }()

	awsFindOne(t, client)
	assert.Greater(t, tracking.called, 0, "expected custom credential provider to be called at least once")
}
