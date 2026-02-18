// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package awsauth

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"

	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var _ options.AWSCredentialsProvider = (*CredentialsProvider)(nil)

// CredentialsProvider adapts an AWS SDK v2 CredentialsProvider to the
// options.AWSCredentialsProvider interface used by the MongoDB Go Driver.
type CredentialsProvider struct {
	provider aws.CredentialsProvider
}

// NewCredentialsProvider returns a CredentialsProvider that wraps AWS
// CredentialsProvider. CredentialsProvider is expected to not be nil.
func NewCredentialsProvider(credentialsProvider aws.CredentialsProvider) (*CredentialsProvider, error) {
	return &CredentialsProvider{
		provider: credentialsProvider,
	}, nil
}

// Retrieve returns the credentials.
func (p *CredentialsProvider) Retrieve(ctx context.Context) (options.AWSCredentials, error) {
	creds, err := p.provider.Retrieve(ctx)
	return options.AWSCredentials(creds), err
}
