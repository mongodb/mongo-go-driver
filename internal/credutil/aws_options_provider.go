// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package credutil

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/internal/aws/credentials"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// AWSOptionsProvider adapts options.AWSCredentialsProvider to the
// credentials.Provider interface used internally by the driver.
type AWSOptionsProvider struct {
	Provider options.AWSCredentialsProvider
}

// Retrieve returns the credentials from the wrapped options.AWSCredentialsProvider.
func (p AWSOptionsProvider) Retrieve(ctx context.Context) (credentials.Value, error) {
	creds, err := p.Provider.Retrieve(ctx)
	if err != nil {
		return credentials.Value{}, err
	}
	return credentials.Value{
		AccessKeyID:     creds.AccessKeyID,
		SecretAccessKey: creds.SecretAccessKey,
		SessionToken:    creds.SessionToken,
		Source:          creds.Source,
		CanExpire:       creds.CanExpire,
		Expires:         creds.Expires,
		AccountID:       creds.AccountID,
	}, nil
}
