// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
//
// Based on github.com/aws/aws-sdk-go by Amazon.com, Inc. with code from:
// - github.com/aws/aws-sdk-go/blob/v1.44.225/aws/credentials/chain_provider.go
// See THIRD-PARTY-NOTICES for original license terms

package credentials

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/internal/aws/awserr"
)

// The ChainProvider provides a way of chaining multiple providers together
// which will pick the first available using priority order of the Providers
// in the list.
//
// If none of the Providers retrieve valid credentials Value, ChainProvider's
// Retrieve() will return an error.
type ChainProvider struct {
	Providers []Provider
}

// NewChainCredentials returns a pointer to a new Credentials object
// wrapping a chain of providers.
func NewChainCredentials(providers []Provider) *Credentials {
	return NewCredentials(&ChainProvider{
		Providers: append([]Provider{}, providers...),
	})
}

// Retrieve returns the credentials value or error if no provider returned
// without error.
func (c *ChainProvider) Retrieve(ctx context.Context) (Value, error) {
	errs := make([]error, 0, len(c.Providers))
	for _, p := range c.Providers {
		creds, err := p.Retrieve(ctx)
		if err == nil {
			if !creds.Expired() && creds.HasKeys() {
				return creds, nil
			}
			err = errors.New("credentials are invalid")
		}
		errs = append(errs, err)
	}

	err := awserr.NewBatchError("NoCredentialProviders", "no valid providers in chain", errs)
	return Value{}, err
}
