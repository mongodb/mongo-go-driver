// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/internal/aws/credentials"
	v4signer "go.mongodb.org/mongo-driver/v2/internal/aws/signer/v4"
	"go.mongodb.org/mongo-driver/v2/internal/credproviders"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/auth/creds"
)

// MongoDBAWS is the mechanism name for MongoDBAWS.
const MongoDBAWS = "MONGODB-AWS"

func newMongoDBAWSAuthenticator(cred *Cred, httpClient *http.Client) (Authenticator, error) {
	if cred.Source != "" && cred.Source != sourceExternal {
		return nil, newAuthError("MONGODB-AWS source must be empty or $external", nil)
	}

	if cred.AWSSigner != nil {
		if cred.AWSCredentialsProvider == nil {
			return nil, errors.New("AWSCredentialsProvider is required when AWSSigner is set")
		}
		return &MongoDBAWSAuthenticator{
			signer: customSigner{
				provider: cred.AWSCredentialsProvider,
				signer:   cred.AWSSigner,
			},
		}, nil
	}

	if httpClient == nil {
		return nil, errors.New("httpClient must not be nil")
	}

	providers := []credentials.Provider{
		&credproviders.StaticProvider{
			Value: credentials.Value{
				AccessKeyID:     cred.Username,
				SecretAccessKey: cred.Password,
				SessionToken:    cred.Props["AWS_SESSION_TOKEN"],
			},
		},
	}
	if cred.AWSCredentialsProvider != nil {
		providers = append(providers, cred.AWSCredentialsProvider)
	}

	return &MongoDBAWSAuthenticator{
		signer: v4signer.Signer{
			Credentials: creds.NewAWSCredentialProvider(httpClient, providers...).Cred,
		},
	}, nil
}

// MongoDBAWSAuthenticator uses AWS-IAM credentials over SASL to authenticate a connection.
type MongoDBAWSAuthenticator struct {
	signer awsSigner
}

// Auth authenticates the connection.
func (a *MongoDBAWSAuthenticator) Auth(ctx context.Context, cfg *driver.AuthConfig) error {
	awsSasl := &awsSaslAdapter{
		signer: a.signer,
	}
	err := ConductSaslConversation(ctx, cfg, sourceExternal, awsSasl)
	if err != nil {
		return newAuthError("sasl conversation error", err)
	}
	return nil
}

// Reauth reauthenticates the connection.
func (a *MongoDBAWSAuthenticator) Reauth(_ context.Context, _ *driver.AuthConfig) error {
	return newAuthError("AWS authentication does not support reauthentication", nil)
}

type customSigner struct {
	provider credentials.Provider
	signer   driver.AWSSigner
}

func (s customSigner) Sign(
	ctx context.Context, req *http.Request, body, service, region string,
	signTime time.Time,
) error {
	creds, err := s.provider.Retrieve(ctx)
	if err != nil {
		return err
	}
	return s.signer.Sign(ctx, creds, req, body, service, region, signTime)
}
