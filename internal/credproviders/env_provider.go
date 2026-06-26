// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package credproviders

import (
	"context"
	"os"

	"go.mongodb.org/mongo-driver/v2/internal/aws/credentials"
)

// EnvVar is an environment variable
type EnvVar string

// Get retrieves the environment variable
func (ev EnvVar) Get() string {
	return os.Getenv(string(ev))
}

// A EnvProvider retrieves credentials from the environment variables of the
// running process. Environment credentials never expire.
type EnvProvider struct {
	AwsAccessKeyIDEnv     EnvVar
	AwsSecretAccessKeyEnv EnvVar
	AwsSessionTokenEnv    EnvVar
}

// NewEnvProvider returns a pointer to an ECS credential provider.
func NewEnvProvider() *EnvProvider {
	return &EnvProvider{
		// AwsAccessKeyIDEnv is the environment variable for AWS_ACCESS_KEY_ID
		AwsAccessKeyIDEnv: EnvVar("AWS_ACCESS_KEY_ID"),
		// AwsSecretAccessKeyEnv is the environment variable for AWS_SECRET_ACCESS_KEY
		AwsSecretAccessKeyEnv: EnvVar("AWS_SECRET_ACCESS_KEY"),
		// AwsSessionTokenEnv is the environment variable for AWS_SESSION_TOKEN
		AwsSessionTokenEnv: EnvVar("AWS_SESSION_TOKEN"),
	}
}

// Retrieve retrieves the keys from the environment.
func (e *EnvProvider) Retrieve(_ context.Context) (credentials.Value, error) {
	v := credentials.Value{
		AccessKeyID:     e.AwsAccessKeyIDEnv.Get(),
		SecretAccessKey: e.AwsSecretAccessKeyEnv.Get(),
		SessionToken:    e.AwsSessionTokenEnv.Get(),
		CanExpire:       false,
	}
	err := verify(v)
	if err != nil {
		// Expire the credentials if invalid.
		v.CanExpire = true
	}

	return v, err
}
