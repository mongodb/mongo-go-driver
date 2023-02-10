// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package awscredproviders

import (
	"os"

	"go.mongodb.org/mongo-driver/x/mongo/driver/auth/internal/aws/credentials"
)

// EnvProviderName provides a name of Env provider
const EnvProviderName = "EnvProvider"

// A EnvProvider retrieves credentials from the environment variables of the
// running process. Environment credentials never expire.
type EnvProvider struct {
	retrieved bool
}

// Retrieve retrieves the keys from the environment.
func (e *EnvProvider) Retrieve() (credentials.Value, error) {
	e.retrieved = false

	v, err := (&StaticProvider{credentials.Value{
		AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		SessionToken:    os.Getenv("AWS_SESSION_TOKEN"),
	}}).Retrieve()

	if err == nil {
		e.retrieved = true
	}

	v.ProviderName = EnvProviderName

	return v, err
}

// IsExpired returns if the credentials have been retrieved.
func (e *EnvProvider) IsExpired() bool {
	return !e.retrieved
}
