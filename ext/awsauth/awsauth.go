// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package awsauth

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// CredentialsProvider adapts an AWS SDK v2 CredentialsProvider to the
// options.AWSCredentialsProvider interface used by the MongoDB Go Driver.
//
// CredentialsProvider is expected to implement the options.AWSCredentialsProvider interface:
//
// import "go.mongodb.org/mongo-driver/v2/mongo/options"
// var _ options.AWSCredentialsProvider = (*CredentialsProvider)(nil)
type CredentialsProvider struct {
	provider aws.CredentialsProvider
}

// NewCredentialsProvider returns a CredentialsProvider that wraps the provided
// AWS SDK v2 CredentialsProvider. credentialsProvider must not be nil.
func NewCredentialsProvider(credentialsProvider aws.CredentialsProvider) *CredentialsProvider {
	return &CredentialsProvider{
		provider: credentialsProvider,
	}
}

// AWSCredentials is a struct that contains AWS credentials information.
//
// This is a copy of the AWS SDK v2 aws.Credentials struct:
// https://pkg.go.dev/github.com/aws/aws-sdk-go-v2@v1.28.0/aws#Credentials
//
// It is declared as a type alias to an anonymous struct so that its field layout
// matches aws.Credentials exactly. That lets us convert an aws.Credentials value
// with a plain type conversion in Retrieve, while keeping this package's public
// API free of any dependency on the AWS SDK's named type.
type AWSCredentials = struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Source          string
	CanExpire       bool
	Expires         time.Time
	AccountID       string
}

// Retrieve returns the credentials.
func (p *CredentialsProvider) Retrieve(ctx context.Context) (AWSCredentials, error) {
	creds, err := p.provider.Retrieve(ctx)
	return AWSCredentials(creds), err
}
