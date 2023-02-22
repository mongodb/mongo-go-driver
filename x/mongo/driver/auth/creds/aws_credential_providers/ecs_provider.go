// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package awscredproviders

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/x/mongo/driver/auth/internal/aws/credentials"
)

const (
	// EcsProviderName provides a name of ECS provider
	EcsProviderName = "EcsProvider"

	awsRelativeURI = "http://169.254.170.2/"
)

// An EcsProvider retrieves credentials from ECS metadata.
type EcsProvider struct {
	httpClient   *http.Client
	expiration   time.Time
	expiryWindow time.Duration
}

// NewEcsProvider returns a pointer to an ECS credential provider.
func NewEcsProvider(httpClient *http.Client, expiryWindow time.Duration) *EcsProvider {
	return &EcsProvider{
		httpClient:   httpClient,
		expiration:   time.Time{},
		expiryWindow: expiryWindow,
	}
}

// RetrieveWithContext retrieves the keys from the AWS service.
func (e *EcsProvider) RetrieveWithContext(ctx context.Context) (credentials.Value, error) {
	const defaultHTTPTimeout = 10 * time.Second

	v := credentials.Value{ProviderName: EcsProviderName}

	relativeEcsURI := os.Getenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI")
	if len(relativeEcsURI) == 0 {
		return v, errors.New("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI is missing")
	}
	fullURI := awsRelativeURI + relativeEcsURI

	req, err := http.NewRequest(http.MethodGet, fullURI, nil)
	if err != nil {
		return v, err
	}
	req.Header.Set("Accept", "application/json")

	ctx, cancel := context.WithTimeout(ctx, defaultHTTPTimeout)
	defer cancel()
	resp, err := e.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return v, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return v, fmt.Errorf("response failure: %s", resp.Status)
	}

	var escResp struct {
		AccessKeyID     string    `json:"AccessKeyId"`
		SecretAccessKey string    `json:"SecretAccessKey"`
		Token           string    `json:"Token"`
		Expiration      time.Time `json:"Expiration"`
	}

	err = json.NewDecoder(resp.Body).Decode(&escResp)
	if err != nil {
		return v, err
	}

	v.AccessKeyID = escResp.AccessKeyID
	v.SecretAccessKey = escResp.SecretAccessKey
	v.SessionToken = escResp.Token
	if !v.HasKeys() {
		return v, errors.New("failed to retrieve ECS keys")
	}
	e.expiration = escResp.Expiration.Add(-e.expiryWindow)

	return v, err
}

// Retrieve retrieves the keys from the AWS service.
func (e *EcsProvider) Retrieve() (credentials.Value, error) {
	return e.RetrieveWithContext(context.Background())
}

// IsExpired returns true if the credentials are expired.
func (e *EcsProvider) IsExpired() bool {
	return e.expiration.Before(time.Now())
}
