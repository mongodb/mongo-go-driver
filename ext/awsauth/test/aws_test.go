// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package awsauthtest

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"go.mongodb.org/mongo-driver/ext/awsauth"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// trackingCredentialsProvider wraps an options.AWSCredentialsProvider and counts calls.
type trackingCredentialsProvider struct {
	inner  options.AWSCredentialsProvider
	called int
}

func (p *trackingCredentialsProvider) Retrieve(ctx context.Context) (awsauth.AWSCredentials, error) {
	p.called++
	return p.inner.Retrieve(ctx)
}

// TestAWSDefaultCustomCredentialProviderAuthenticates is prose test 1:
// "Custom Credential Provider Authenticates" from the MongoDB AWS auth spec.
// https://github.com/mongodb/specifications/blob/master/source/auth/tests/mongodb-aws.md
//
// Uses the AWS SDK default credential chain so all 6 scenarios (Regular, EC2, ECS,
// AssumeRole, WebIdentity, Lambda) are covered in environments where the SDK can
// resolve credentials automatically without needing inline credentials in the URI.
func TestAWSDefaultCustomCredentialProviderAuthenticates(t *testing.T) {
	// The "assume-role" and "regular" scenarios are intentionally skipped. This
	// test exercises the AWS SDK default credential chain and calls SetAuth,
	// which overrides any inline credentials ApplyURI extracted from the URI.
	// Those two scenarios provide credentials inline in MONGODB_URI, which the
	// default chain cannot resolve on its own, so this default-chain approach
	// does not apply to them. They are instead covered in
	// internal/test/aws/aws_test.go, which runs alongside this test via
	// etc/run-mongodb-aws-test.sh.
	if test := os.Getenv("AWS_TEST"); test == "assume-role" || test == "regular" {
		t.Skipf("Skipping test for %s", test)
	}

	rawURI := os.Getenv("MONGODB_URI")
	if rawURI == "" {
		t.Skip("MONGODB_URI not set")
	}

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		t.Fatalf("failed to load AWS config: %v", err)
	}

	tracking := &trackingCredentialsProvider{
		inner: awsauth.NewCredentialsProvider(cfg.Credentials),
	}

	// SetAuth overrides any inline credentials that ApplyURI may have extracted
	// from the URI; the AWS SDK default chain provides credentials instead.
	client, err := mongo.Connect(
		options.Client().
			ApplyURI(rawURI).
			SetAuth(options.Credential{
				AuthMechanism:          "MONGODB-AWS",
				AWSCredentialsProvider: tracking,
			}),
	)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			t.Errorf("Disconnect: %v", err)
		}
	}()

	err = client.Database("aws").Collection("test").
		FindOne(context.Background(), bson.D{{Key: "x", Value: 1}}).Err()
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("unexpected FindOne error: %v", err)
	}

	if tracking.called == 0 {
		t.Fatal("expected custom credential provider to be called at least once")
	}
}
