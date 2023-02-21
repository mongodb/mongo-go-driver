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
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/internal/uuid"
	"go.mongodb.org/mongo-driver/x/mongo/driver/auth/internal/aws/credentials"
)

const (
	// AssumeRoleProviderName provides a name of assume role provider
	AssumeRoleProviderName = "AssumeRoleProvider"

	stsURI = `https://sts.amazonaws.com/?Action=AssumeRoleWithWebIdentity&RoleSessionName=%s&RoleArn=%s&WebIdentityToken=%s&Version=2011-06-15`
)

// An AssumeRoleProvider retrieves credentials for assume role with web identity.
type AssumeRoleProvider struct {
	httpClient   *http.Client
	expiration   time.Time
	expiryWindow time.Duration
}

// NewAssumeRoleProvider returns a pointer to an assume role provider.
func NewAssumeRoleProvider(httpClient *http.Client, expiryWindow time.Duration) *AssumeRoleProvider {
	return &AssumeRoleProvider{
		httpClient:   httpClient,
		expiration:   time.Time{},
		expiryWindow: expiryWindow,
	}
}

// RetrieveWithContext retrieves the keys from the AWS service.
func (a *AssumeRoleProvider) RetrieveWithContext(ctx context.Context) (credentials.Value, error) {
	const defaultHTTPTimeout = 10 * time.Second

	v := credentials.Value{ProviderName: AssumeRoleProviderName}

	roleArn := os.Getenv("AWS_ROLE_ARN")
	tokenFile := os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE")
	if tokenFile == "" && roleArn == "" {
		return v, errors.New("AWS_WEB_IDENTITY_TOKEN_FILE and AWS_ROLE_ARN are missing")
	}
	if tokenFile != "" && roleArn == "" {
		return v, errors.New("AWS_WEB_IDENTITY_TOKEN_FILE is set, but AWS_ROLE_ARN is missing")
	}
	if tokenFile == "" && roleArn != "" {
		return v, errors.New("AWS_ROLE_ARN is set, but AWS_WEB_IDENTITY_TOKEN_FILE is missing")
	}
	token, err := ioutil.ReadFile(tokenFile)
	if err != nil {
		return v, err
	}

	sessionName := os.Getenv("AWS_ROLE_SESSION_NAME")
	if sessionName == "" {
		// Use a UUID if the RoleSessionName is not given.
		id, err := uuid.New()
		if err != nil {
			return v, err
		}
		sessionName = id.String()
	}

	fullURI := fmt.Sprintf(stsURI, sessionName, roleArn, string(token))

	req, err := http.NewRequest(http.MethodPost, fullURI, nil)
	if err != nil {
		return v, err
	}
	req.Header.Set("Accept", "application/json")

	ctx, cancel := context.WithTimeout(ctx, defaultHTTPTimeout)
	defer cancel()
	resp, err := a.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return v, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return v, fmt.Errorf("response failure: %s", resp.Status)
	}

	var stsResp struct {
		Response struct {
			Result struct {
				Credentials struct {
					AccessKeyID     string  `json:"AccessKeyId"`
					SecretAccessKey string  `json:"SecretAccessKey"`
					Token           string  `json:"SessionToken"`
					Expiration      float64 `json:"Expiration"`
				} `json:"Credentials"`
			} `json:"AssumeRoleWithWebIdentityResult"`
		} `json:"AssumeRoleWithWebIdentityResponse"`
	}

	err = json.NewDecoder(resp.Body).Decode(&stsResp)
	if err != nil {
		return v, err
	}
	v.AccessKeyID = stsResp.Response.Result.Credentials.AccessKeyID
	v.SecretAccessKey = stsResp.Response.Result.Credentials.SecretAccessKey
	v.SessionToken = stsResp.Response.Result.Credentials.Token
	if !v.HasKeys() {
		return v, errors.New("failed to retrieve web identity keys")
	}
	sec := int64(stsResp.Response.Result.Credentials.Expiration)
	a.expiration = time.Unix(sec, 0).Add(-a.expiryWindow)

	return v, err
}

// Retrieve retrieves the keys from the AWS service.
func (a *AssumeRoleProvider) Retrieve() (credentials.Value, error) {
	return a.RetrieveWithContext(context.Background())
}

// IsExpired returns if the credentials have been retrieved.
func (a *AssumeRoleProvider) IsExpired() bool {
	return a.expiration.Before(time.Now())
}
