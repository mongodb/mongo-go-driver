// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package credproviders

import (
	"os"

	"go.mongodb.org/mongo-driver/x/mongo/driver/auth/internal/aws/credentials"
)

// EnvProviderName provides a name of Env provider
const EnvProviderName = "EnvProvider"

var (
	// AwsAccessKeyIDEnv is the environment variable for AWS_ACCESS_KEY_ID
	AwsAccessKeyIDEnv = EnvVar("AWS_ACCESS_KEY_ID")
	// AwsSecretAccessKeyEnv is the environment variable for AWS_SECRET_ACCESS_KEY
	AwsSecretAccessKeyEnv = EnvVar("AWS_SECRET_ACCESS_KEY")
	// AwsSessionTokenEnv is the environment variable for AWS_SESSION_TOKEN
	AwsSessionTokenEnv = EnvVar("AWS_SESSION_TOKEN")
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
	retrieved bool
}

// Retrieve retrieves the keys from the environment.
func (e *EnvProvider) Retrieve() (credentials.Value, error) {
	e.retrieved = false

	v := credentials.Value{
		AccessKeyID:     AwsAccessKeyIDEnv.Get(),
		SecretAccessKey: AwsSecretAccessKeyEnv.Get(),
		SessionToken:    AwsSessionTokenEnv.Get(),
		ProviderName:    EnvProviderName,
	}
	err := verify(v)
	if err == nil {
		e.retrieved = true
	}

	return v, err
}

// IsExpired returns if the credentials have been retrieved.
func (e *EnvProvider) IsExpired() bool {
	return !e.retrieved
}
