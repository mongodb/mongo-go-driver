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

	"go.mongodb.org/mongo-driver/v2/internal/aws/credentials"
	"go.mongodb.org/mongo-driver/v2/internal/credproviders"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

const (
	// expiryWindow will allow the credentials to trigger refreshing prior to the credentials actually expiring.
	// This is beneficial so expiring credentials do not cause request to fail unexpectedly due to exceptions.
	//
	// Set an early expiration of 5 minutes before the credentials are actually expired.
	expiryWindow = 5 * time.Minute
)

// NewAWSCredentials generates new AWSCredentialDocSource
func NewAWSCredentials(httpClient *http.Client, providers ...credentials.Provider) *credentials.Credentials {
	providers = append(
		providers,
		credproviders.NewEnvProvider(),
		credproviders.NewAssumeRoleProvider(httpClient, expiryWindow),
		credproviders.NewECSProvider(httpClient, expiryWindow),
		credproviders.NewEC2Provider(httpClient, expiryWindow),
	)

	return credentials.NewChainCredentials(providers)
}

// AWSCredentialDocSource wraps AWS credentials.
type AWSCredentialDocSource struct {
	Creds *credentials.Credentials
}

// GetCredentialsDoc generates AWS credentials.
func (p AWSCredentialDocSource) GetCredentialsDoc(ctx context.Context) (bsoncore.Document, error) {
	creds, err := p.Creds.Get(ctx)
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
