// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package awsauth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"

	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	// emptyStringSHA256 is a SHA256 of an empty string
	emptyStringSHA256 = `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
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
	if credentialsProvider == nil {
		return nil, errors.New("nil provider")
	}
	return &CredentialsProvider{
		provider: credentialsProvider,
	}, nil
}

// Retrieve returns the credentials.
func (p *CredentialsProvider) Retrieve(ctx context.Context) (options.AWSCredentials, error) {
	creds, err := p.provider.Retrieve(ctx)
	return options.AWSCredentials(creds), err
}

var _ options.AWSSigner = (*Signer)(nil)

// Signer adapts an AWS SDK v2 SigV4 HTTP signer to the options.AWSSigner
// interface used by the MongoDB Go Driver.
type Signer struct {
	signer v4.HTTPSigner
}

// NewSigner creates a new Signer from the provided AWS HTTPSigner. HTTPSigner
// is expected to not be nil.
func NewSigner(httpSigner v4.HTTPSigner) (*Signer, error) {
	if httpSigner == nil {
		return nil, errors.New("nil httpSigner")
	}
	return &Signer{
		signer: httpSigner,
	}, nil
}

// Sign signs AWS SigV4 requests.  The payload parameter should be the raw
// request body. This method computes the SHA256 hash of the payload before
// signing. If payload is empty, the precomputed hash of an empty string is
// used.
func (s *Signer) Sign(
	ctx context.Context, creds options.AWSCredentials, r *http.Request,
	payload, service, region string, signingTime time.Time,
) error {
	if len(payload) == 0 {
		payload = emptyStringSHA256
	} else {
		hash := sha256.Sum256([]byte(payload))
		payload = hex.EncodeToString(hash[:])
	}
	if creds.SessionToken != "" {
		r.Header.Set("X-Amz-Security-Token", creds.SessionToken)
	}
	return s.signer.SignHTTP(ctx, aws.Credentials(creds), r, payload, service, region, signingTime)
}
