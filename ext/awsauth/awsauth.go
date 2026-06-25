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

// NewCredentialsProvider returns a CredentialsProvider that wraps AWS
// CredentialsProvider. CredentialsProvider is expected to not be nil.
func NewCredentialsProvider(credentialsProvider aws.CredentialsProvider) *CredentialsProvider {
	return &CredentialsProvider{
		provider: credentialsProvider,
	}
}

// Retrieve returns the credentials.
func (p *CredentialsProvider) Retrieve(ctx context.Context) (struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Source          string
	CanExpire       bool
	Expires         time.Time
	AccountID       string
}, error,
) {
	creds, err := p.provider.Retrieve(ctx)
	return struct {
		AccessKeyID     string
		SecretAccessKey string
		SessionToken    string
		Source          string
		CanExpire       bool
		Expires         time.Time
		AccountID       string
	}{
		AccessKeyID:     creds.AccessKeyID,
		SecretAccessKey: creds.SecretAccessKey,
		SessionToken:    creds.SessionToken,
		Source:          creds.Source,
		CanExpire:       creds.CanExpire,
		Expires:         creds.Expires,
		AccountID:       creds.AccountID,
	}, err
}
