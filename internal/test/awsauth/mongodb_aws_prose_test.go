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

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/ext/awsauth"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// This file defines one test per AWS_TEST scenario. The scenarios are set by
// .evergreen/config.yml and prepared by drivers-evergreen-tools' aws_setup.sh,
// which shapes MONGODB_URI and the AWS_* environment differently for each.
// Exactly one scenario is live per test process; the rest skip.
//
// The scenarios differ in exactly one respect: where valid credentials come
// from. The assertions are shared.
//
//	AWS_TEST       spec scenario  credential source              obtained via   ext/awsauth
//	-------------  -------------  -----------------------------  -------------  -----------
//	regular        1              inline in MONGODB_URI          credsFromURI   no
//	ec2            2              instance metadata endpoint     credsFromSDK   yes
//	ecs            3              container credentials endpoint credsFromSDK   yes
//	assume-role    4              inline in MONGODB_URI + token  credsFromURI   no
//	web-identity   5              web identity token file        credsFromSDK   yes
//	env-creds      6              AWS_* env vars, no token       credsFromEnv   no
//	session-creds  6              AWS_* env vars, with token     credsFromEnv   no
//
//	It's easier to keep all the prose tests in one file than to split them by
//	scenario. Since some require the AWS SDK and some don't, this file imports
//	it unconditionally.

// requireScenario skips unless MONGODB_URI is set. It returns the raw URI.
func requireScenario(t *testing.T) string {
	t.Helper()

	rawURI := os.Getenv("MONGODB_URI")
	if rawURI == "" {
		t.Skip("Skipping test: MONGODB_URI is not set")
	}

	return rawURI
}

// ---------------------------------------------------------------------------
// Credential sources. One per way a scenario can supply valid credentials.
// ---------------------------------------------------------------------------

// staticProvider returns fixed credentials on every call.
type staticProvider struct {
	creds options.AWSCredentials
}

func (p *staticProvider) Retrieve(context.Context) (options.AWSCredentials, error) {
	return p.creds, nil
}

// trackingCredentialsProvider wraps an options.AWSCredentialsProvider and counts calls.
type trackingCredentialsProvider struct {
	inner  options.AWSCredentialsProvider
	called int
}

func (p *trackingCredentialsProvider) Retrieve(ctx context.Context) (awsauth.AWSCredentials, error) {
	p.called++

	return p.inner.Retrieve(ctx)
}

// credsFromURI returns the credentials embedded in MONGODB_URI, including the
// AWS_SESSION_TOKEN authMechanismProperty when present.
func credsFromURI(t *testing.T, rawURI string) options.AWSCredentialsProvider {
	t.Helper()

	opts := options.Client().ApplyURI(rawURI)
	require.NotNil(t, opts.Auth, "MONGODB_URI carries no credentials")
	require.NotEmpty(t, opts.Auth.Username, "MONGODB_URI carries no inline credentials")

	return &staticProvider{creds: options.AWSCredentials{
		AccessKeyID:     opts.Auth.Username,
		SecretAccessKey: opts.Auth.Password,
		SessionToken:    opts.Auth.AuthMechanismProperties["AWS_SESSION_TOKEN"],
	}}
}

// credsFromEnv returns the credentials exported into the environment.
func credsFromEnv(t *testing.T) options.AWSCredentialsProvider {
	t.Helper()

	creds := options.AWSCredentials{
		AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		SessionToken:    os.Getenv("AWS_SESSION_TOKEN"),
	}
	require.NotEmpty(t, creds.AccessKeyID, "AWS_ACCESS_KEY_ID is not set")
	require.NotEmpty(t, creds.SecretAccessKey, "AWS_SECRET_ACCESS_KEY is not set")

	return &staticProvider{creds: creds}
}

// credsFromSDK returns a provider backed by the AWS SDK default credential
// chain, which resolves the instance metadata, container, and web identity
// sources that cannot be read directly.
func credsFromSDK(t *testing.T) options.AWSCredentialsProvider {
	t.Helper()

	cfg, err := config.LoadDefaultConfig(context.Background())
	require.NoError(t, err, "failed to load AWS config")

	return awsauth.NewCredentialsProvider(cfg.Credentials)
}

// ---------------------------------------------------------------------------
// Shared assertions. These are the prose tests; they do not vary by scenario.
// https://github.com/mongodb/specifications/blob/master/source/auth/tests/mongodb-aws.md
// ---------------------------------------------------------------------------

// connect returns a client authenticating with MONGODB-AWS through the given
// provider. SetAuth overrides any inline credentials ApplyURI parsed from the
// URI, so the resolution chain is: custom provider, then the driver's built-in
// providers.
func connect(t *testing.T, rawURI string, provider options.AWSCredentialsProvider) *mongo.Client {
	t.Helper()

	client, err := mongo.Connect(options.Client().
		ApplyURI(rawURI).
		SetAuth(options.Credential{
			AuthMechanism:          "MONGODB-AWS",
			AWSCredentialsProvider: provider,
		}))
	require.NoError(t, err, "failed to connect")

	t.Cleanup(func() {
		require.NoError(t, client.Disconnect(context.Background()), "Disconnect")
	})

	return client
}

// findOne runs a query that forces a handshake. An empty result set is success;
// only an error means authentication failed.
func findOne(t *testing.T, client *mongo.Client) {
	t.Helper()

	err := client.Database("aws").Collection("test").
		FindOne(context.Background(), bson.D{{Key: "x", Value: 1}}).Err()
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		require.NoError(t, err, "unexpected FindOne error")
	}
}

func assertScenarioAuthenticates(t *testing.T, rawURI string) {
	t.Helper()

	client, err := mongo.Connect(options.Client().ApplyURI(rawURI))
	require.NoError(t, err, "failed to connect")

	t.Cleanup(func() {
		require.NoError(t, client.Disconnect(context.Background()), "Disconnect")
	})

	findOne(t, client)
}

func assertCustomProviderAuthenticates(t *testing.T, rawURI string, provider options.AWSCredentialsProvider) {
	t.Helper()

	tracking := &trackingCredentialsProvider{inner: provider}
	findOne(t, connect(t, rawURI, tracking))

	require.NotZero(t, tracking.called, "expected the custom credential provider to be called at least once")
}

// runScenarioTests runs everything the spec requires of every scenario: the
// top-level authentication requirement, then prose test 1.
func runScenarioTests(t *testing.T, rawURI string, provider options.AWSCredentialsProvider) {
	t.Helper()

	t.Run("scenario authenticates", func(t *testing.T) {
		assertScenarioAuthenticates(t, rawURI)
	})
	t.Run("1. custom credential provider authenticates", func(t *testing.T) {
		assertCustomProviderAuthenticates(t, rawURI, provider)
	})
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// Regular Credentials: Auth via an AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY
// pair.
//
// Drivers MUST be able to authenticate when a valid access key id and secret
// access key pair are present in the environment.
func TestAWSProse_1_RegularCredentials(t *testing.T) {
	rawURI := requireScenario(t)
	runScenarioTests(t, rawURI, credsFromURI(t, rawURI))
}

// EC2 Credentials: Auth from an EC2 instance via temporary credentials assigned
// to the machine
//
// Drivers MUST be able to authenticate from an EC2 instance via temporary
// credentials assigned to the machine.
func TestAWSProse_2_EC2Credentials(t *testing.T) {
	rawURI := requireScenario(t)
	runScenarioTests(t, rawURI, credsFromSDK(t))
}

// ECS Credentials: Auth from an ECS instance via temporary credentials assigned
// to the task
//
// Drivers MUST be able to authenticate from an ECS container via temporary
// credentials.
func TestAWSProse_3_ECSCredentials(t *testing.T) {
	rawURI := requireScenario(t)
	runScenarioTests(t, rawURI, credsFromSDK(t))
}

// Assume Role: Auth via temporary credentials obtained from an STS AssumeRole
// request
//
// Drivers MUST be able to authenticate using temporary credentials returned
// from an assume role request. These temporary credentials consist of an access
// key ID, a secret access key, and a security token present in the environment.
func TestAWSProse_4_AssumeRole(t *testing.T) {
	rawURI := requireScenario(t)
	runScenarioTests(t, rawURI, credsFromURI(t, rawURI))
}

// Assume Role with Web Identity: Auth via temporary credentials obtained from
// an STS AssumeRoleWithWebIdentity request
//
// Drivers MUST test with and without AWS_ROLE_SESSION_NAME set.
//
// Both cases are the same drivers-evergreen-tools scenario, so they are two
// invocations of this test rather than two tests; see the pair of
// run-aws-auth-test-with-aws-web-identity-credentials commands in
// .evergreen/config.yml.
func TestAWSProse_5_AssumeRoleWithWebIdentity(t *testing.T) {
	rawURI := requireScenario(t)
	runScenarioTests(t, rawURI, credsFromSDK(t))
}

// AWS Lambda: Auth via environment variables AWS_ACCESS_KEY_ID,
// AWS_SECRET_ACCESS_KEY, and AWS_SESSION_TOKEN.
//
// Sample URIs both with and without optional session tokens set are shown
// below. Drivers MUST test both cases.
//
// The two cases are separate drivers-evergreen-tools scenarios: "env-creds"
// exports the pair alone, "session-creds" adds AWS_SESSION_TOKEN.
func TestAWSProse_6_AWSLambda(t *testing.T) {
	rawURI := requireScenario(t)
	provider := credsFromEnv(t)

	runScenarioTests(t, rawURI, provider)

	// Prose test 2 case 2, "Custom Provider Takes Precedence Over Environment
	// Variables". Only this scenario satisfies the spec's precondition of "an
	// environment with AWS credentials configured as environment variables".
	//
	// Runs last: it invalidates the environment credentials, which are the only
	// valid ones in this scenario. provider was built above and still holds them,
	// so a successful query proves the driver used the custom provider rather
	// than falling back to the environment.
	t.Run("2. custom credential provider precedence", func(t *testing.T) {
		t.Setenv("AWS_ACCESS_KEY_ID", "AWS_ACCESS_KEY_ID")
		t.Setenv("AWS_SECRET_ACCESS_KEY", "AWS_SECRET_ACCESS_KEY")

		tracking := &trackingCredentialsProvider{inner: provider}
		findOne(t, connect(t, rawURI, tracking))

		require.NotZero(t, tracking.called, "expected the custom credential provider to be called at least once")
	})
}
