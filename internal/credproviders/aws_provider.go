// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package credproviders

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/internal/aws/credentials"
)

const awsProviderName = "AwsProvider"

// AwsCredentialsProvider is a function that retrieves AWS credentials.
type AwsCredentialsProvider func(context.Context) (AwsCredentials, error)

// AwsCredentials represents AWS credentials with an expiration callback.
type AwsCredentials struct {
	AccessKeyID        string
	SecretAccessKey    string
	SessionToken       string
	ExpirationCallback func() bool
}

// AwsProvider retrieves credentials from the given AWS credentials provider.
type AwsProvider struct {
	credentials *AwsCredentials

	Provider AwsCredentialsProvider
}

// Retrieve retrieves the keys from the given AWS credentials provider.
func (a *AwsProvider) Retrieve(ctx context.Context) (credentials.Value, error) {
	var value credentials.Value
	if a.credentials == nil {
		creds, err := a.Provider(ctx)
		if err != nil {
			return value, err
		}
		a.credentials = &creds
	}
	value.AccessKeyID = a.credentials.AccessKeyID
	value.SecretAccessKey = a.credentials.SecretAccessKey
	value.SessionToken = a.credentials.SessionToken
	value.ProviderName = awsProviderName
	return value, nil
}

// IsExpired returns true if the credentials have not been retrieved.
func (a *AwsProvider) IsExpired() bool {
	if a.credentials == nil || a.credentials.ExpirationCallback == nil {
		return true
	}
	return a.credentials.ExpirationCallback()
}
