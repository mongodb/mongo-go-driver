// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package creds

import (
	"context"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	credproviders "go.mongodb.org/mongo-driver/x/mongo/driver/auth/creds/credential_providers"
	"go.mongodb.org/mongo-driver/x/mongo/driver/auth/internal/aws/credentials"
)

const (
	expiryWindow = 5 * time.Minute
)

// AwsCredentialProvider wraps AWS credentials.
type AwsCredentialProvider struct {
	Providers []credentials.Provider
}

// NewAwsCredentialProvider generates new AwsCredentialProvider
func NewAwsCredentialProvider(httpClient *http.Client) AwsCredentialProvider {
	var providers []credentials.Provider
	providers = append(
		providers,
		&credproviders.EnvProvider{},
		credproviders.NewAssumeRoleProvider(httpClient, expiryWindow),
		credproviders.NewEcsProvider(httpClient, expiryWindow),
		credproviders.NewEc2Provider(httpClient, expiryWindow),
	)

	return AwsCredentialProvider{providers}
}

// GetCredentialsDoc generates AWS credentials.
func (p AwsCredentialProvider) GetCredentialsDoc(ctx context.Context) (bsoncore.Document, error) {
	creds, err := credentials.NewChainCredentials(p.Providers).GetWithContext(ctx)
	if err != nil {
		return nil, err
	}
	builder := bsoncore.NewDocumentBuilder().
		AppendString("accessKeyId", creds.AccessKeyID).
		AppendString("secretAccessKey", creds.SecretAccessKey)
	if token := creds.SessionToken; len(token) > 0 {
		builder.AppendString("sessionToken", token)
	}
	return builder.Build(), nil
}
