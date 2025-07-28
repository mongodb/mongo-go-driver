// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package credproviders

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/internal/aws/credentials"
)

const (
	eksProviderName = "EKSProvider"
)

type EKSProvider struct {
	awsContainerCredentialsFullURIEnv EnvVar

	httpClient *http.Client
	expiration time.Time

	// expiryWindow will allow the credentials to trigger refreshing prior to the credentials actually expiring.
	// This is beneficial so expiring credentials do not cause request to fail unexpectedly due to exceptions.
	//
	// So a ExpiryWindow of 10s would cause calls to IsExpired() to return true
	// 10 seconds before the credentials are actually expired.
	expiryWindow time.Duration
}

// NewEKSProvider returns a pointer to an EKS credential provider.
func NewEKSProvider(httpClient *http.Client, expiryWindow time.Duration) *EKSProvider {
	return &EKSProvider{
		// AwsContainerCredentialsFullURIEnv is the environment variable for AWS_CONTAINER_CREDENTIALS_RELATIVE_URI
		awsContainerCredentialsFullURIEnv: EnvVar("AWS_CONTAINER_CREDENTIALS_FULL_URI"),
		httpClient:                        httpClient,
		expiryWindow:                      expiryWindow,
	}
}

// RetrieveWithContext retrieves the keys from the AWS service.
func (e *EKSProvider) RetrieveWithContext(ctx context.Context) (credentials.Value, error) {
	const defaultHTTPTimeout = 10 * time.Second

	v := credentials.Value{ProviderName: eksProviderName}

	fullURI := e.awsContainerCredentialsFullURIEnv.Get()
	if len(fullURI) == 0 {
		return v, fmt.Errorf("AWS_CONTAINER_CREDENTIALS_FULL_URI is missing")
	}

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
	var eksResp struct {
		AccessKeyID     string    `json:"AccessKeyId"`
		SecretAccessKey string    `json:"SecretAccessKey"`
		Token           string    `json:"Token"`
		Expiration      time.Time `json:"Expiration"`
	}

	err = json.NewDecoder(resp.Body).Decode(&eksResp)
	if err != nil {
		return v, err
	}

	v.AccessKeyID = eksResp.AccessKeyID
	v.SecretAccessKey = eksResp.SecretAccessKey
	v.SessionToken = eksResp.Token
	if !v.HasKeys() {
		return v, errors.New("failed to retrieve eks keys")
	}
	e.expiration = eksResp.Expiration.Add(-e.expiryWindow)

	return v, nil
}

// Retrieve retrieves the keys from the AWS service.
func (e *EKSProvider) Retrieve() (credentials.Value, error) {
	return e.RetrieveWithContext(context.Background())
}

// IsExpired returns true if the credentials are expired.
func (e *EKSProvider) IsExpired() bool {
	return time.Now().After(e.expiration)
}
