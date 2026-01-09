// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongoaws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type CredentialsProvider struct {
	cfg *aws.Config

	creds aws.Credentials
}

func NewCredentialsProvider(cfg *aws.Config) *CredentialsProvider {
	if cfg == nil {
		cfg = aws.NewConfig()
	}
	return &CredentialsProvider{
		cfg:   cfg,
		creds: aws.Credentials{},
	}
}

func (p *CredentialsProvider) Retrieve(ctx context.Context) (options.AWSCredentials, error) {
	var err error
	p.creds, err = p.cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return options.AWSCredentials{}, err
	}
	return options.AWSCredentials(p.creds), nil
}

func (p *CredentialsProvider) Expired() bool {
	return p.creds.Expired()
}
