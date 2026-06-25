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

func awsFindOne(t *testing.T, client *mongo.Client) {
	t.Helper()
	err := client.Database("aws").Collection("test").FindOne(context.Background(), bson.D{{Key: "x", Value: 1}}).Err()
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("unexpected FindOne error: %v", err)
	}
}

func TestAWS(t *testing.T) {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		t.Skip("Skipping test: MONGODB_URI environment variable is not set")
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))

	defer func() {
		err = client.Disconnect(context.Background())
		require.NoError(t, err)
	}()

	coll := client.Database("aws").Collection("test")

	err = coll.FindOne(context.Background(), bson.D{{Key: "x", Value: 1}}).Err()
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		t.Logf("FindOne error: %v", err)
	}
}

// Custom Credential Provider Authenticates
//
// Scenarios where MONGODB_URI contains no inline credentials (EC2, ECS, WebIdentity)
// are skipped because a valid custom provider for those environments requires the AWS
// SDK, which is not available in this module.
func TestAWSCustomCredentialProviderAuthenticates(t *testing.T) {
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
	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	if accessKeyID == "" || secretAccessKey == "" {
		t.Skip("AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY must be set")
	}

	rawURI := os.Getenv("MONGODB_URI")
	if rawURI == "" {
		rawURI = "mongodb://localhost:27017/?authMechanism=MONGODB-AWS"
	}

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
