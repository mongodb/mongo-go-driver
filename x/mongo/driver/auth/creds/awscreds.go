// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package creds

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/credentials/endpointcreds"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"go.mongodb.org/mongo-driver/internal/uuid"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

const (
	awsRelativeURI = "http://169.254.170.2/"
	expiryWindow   = 5 * time.Minute
)

// AwsCredentialProvider wraps AWS credentials.
type AwsCredentialProvider struct {
	Value *credentials.Credentials
}

// NewAwsCredentialProvider generates new AwsCredentialProvider
func NewAwsCredentialProvider(v credentials.Value) (AwsCredentialProvider, error) {
	getProvider := func(providers []credentials.Provider) AwsCredentialProvider {
		return AwsCredentialProvider{credentials.NewChainCredentials(providers)}
	}

	providers := []credentials.Provider{
		&credentials.StaticProvider{Value: v},
		&credentials.EnvProvider{},
	}

	sess, err := session.NewSession()
	if err != nil {
		return getProvider(providers), err
	}

	// Generate Web Identity Role provider.
	if provider, err := getRoleProvider(sess); err != nil {
		return getProvider(providers), err
	} else if provider != nil {
		providers = append(providers, provider)
	}

	// Generate ECS provider.
	if relativeEcsURI := os.Getenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI"); len(relativeEcsURI) > 0 {
		fullURI := awsRelativeURI + relativeEcsURI
		providers = append(
			providers,
			endpointcreds.NewProviderClient(*sess.Config, sess.Handlers, fullURI,
				func(p *endpointcreds.Provider) {
					p.ExpiryWindow = expiryWindow
				},
			),
		)
	}

	// Generate EC2 Role provider.
	providers = append(providers, &ec2rolecreds.EC2RoleProvider{
		Client:       ec2metadata.New(sess),
		ExpiryWindow: expiryWindow,
	})

	return getProvider(providers), nil
}

// getRoleProvider generates Web Identity Role provider.
func getRoleProvider(p client.ConfigProvider) (credentials.Provider, error) {
	roleArn := os.Getenv("AWS_ROLE_ARN")
	tokenFile := os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE")
	if tokenFile != "" && roleArn == "" {
		return nil, errors.New("AWS_WEB_IDENTITY_TOKEN_FILE is set, but AWS_ROLE_ARN is missing")
	}
	if tokenFile == "" && roleArn != "" {
		return nil, errors.New("AWS_ROLE_ARN is set, but AWS_WEB_IDENTITY_TOKEN_FILE is missing")
	}
	if tokenFile != "" && roleArn != "" {
		sessionName := os.Getenv("AWS_ROLE_SESSION_NAME")
		if sessionName == "" {
			// Use a UUID if the RoleSessionName is not given.
			id, err := uuid.New()
			if err != nil {
				return nil, err
			}
			sessionName = id.String()
		}
		svc := sts.New(p)
		provider := stscreds.NewWebIdentityRoleProvider(svc, roleArn, sessionName, tokenFile)
		return provider, nil
	}
	return nil, nil
}

// GetCredentialsDoc generates AWS credentials.
func (p AwsCredentialProvider) GetCredentialsDoc(ctx context.Context) (bsoncore.Document, error) {
	creds, err := p.Value.GetWithContext(ctx)
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
