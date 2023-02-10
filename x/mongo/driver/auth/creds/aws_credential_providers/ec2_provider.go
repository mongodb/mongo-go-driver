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
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/x/mongo/driver/auth/internal/aws/credentials"
)

const (
	// Ec2ProviderName provides a name of EC2 provider
	Ec2ProviderName = "Ec2Provider"

	awsEC2URI       = "http://169.254.169.254/"
	awsEC2RolePath  = "latest/meta-data/iam/security-credentials/"
	awsEC2TokenPath = "latest/api/token"
)

// An Ec2Provider retrieves credentials from EC2 metadata.
type Ec2Provider struct {
	httpClient   *http.Client
	expiration   time.Time
	expiryWindow time.Duration
}

// NewEc2Provider returns a pointer to an EC2 credential provider.
func NewEc2Provider(httpClient *http.Client, expiryWindow time.Duration) *Ec2Provider {
	return &Ec2Provider{
		httpClient:   httpClient,
		expiration:   time.Time{},
		expiryWindow: expiryWindow,
	}
}

func (e *Ec2Provider) executeAWSHTTPRequest(ctx context.Context, req *http.Request) (io.ReadCloser, error) {
	const defaultHTTPTimeout = 10 * time.Second

	ctx, cancel := context.WithTimeout(ctx, defaultHTTPTimeout)
	defer cancel()
	resp, err := e.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response failure: %s", resp.Status)
	}
	return resp.Body, nil
}

func (e *Ec2Provider) getToken(ctx context.Context) (string, error) {
	req, err := http.NewRequest(http.MethodPut, awsEC2URI+awsEC2TokenPath, nil)
	if err != nil {
		return "", err
	}
	const defaultEC2TTLSeconds = "30"
	req.Header.Set("X-aws-ec2-metadata-token-ttl-seconds", defaultEC2TTLSeconds)

	r, err := e.executeAWSHTTPRequest(ctx, req)
	if err != nil {
		return "", err
	}
	defer r.Close()

	token, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}
	if len(token) == 0 {
		return "", errors.New("unable to retrieve token from EC2 metadata")
	}
	return string(token), nil
}

func (e *Ec2Provider) getRoleName(ctx context.Context, token string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, awsEC2URI+awsEC2RolePath, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-aws-ec2-metadata-token", token)

	r, err := e.executeAWSHTTPRequest(ctx, req)
	if err != nil {
		return "", err
	}
	defer r.Close()

	role, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}
	if len(role) == 0 {
		return "", errors.New("unable to retrieve role_name from EC2 metadata")
	}
	return string(role), nil
}

func (e *Ec2Provider) getCredentials(ctx context.Context, token string, role string) (credentials.Value, time.Time, error) {
	v := credentials.Value{ProviderName: Ec2ProviderName}

	pathWithRole := awsEC2URI + awsEC2RolePath + role
	req, err := http.NewRequest(http.MethodGet, pathWithRole, nil)
	if err != nil {
		return v, time.Time{}, err
	}
	req.Header.Set("X-aws-ec2-metadata-token", token)
	resp, err := e.executeAWSHTTPRequest(ctx, req)
	if err != nil {
		return v, time.Time{}, err
	}
	defer resp.Close()

	var es2Resp struct {
		AccessKeyID     string    `json:"AccessKeyId"`
		SecretAccessKey string    `json:"SecretAccessKey"`
		Token           string    `json:"Token"`
		Expiration      time.Time `json:"Expiration"`
	}

	err = json.NewDecoder(resp).Decode(&es2Resp)
	if err != nil {
		return v, time.Time{}, err
	}

	v.AccessKeyID = es2Resp.AccessKeyID
	v.SecretAccessKey = es2Resp.SecretAccessKey
	v.SessionToken = es2Resp.Token

	return v, es2Resp.Expiration, nil
}

// RetrieveWithContext retrieves the keys from the AWS service.
func (e *Ec2Provider) RetrieveWithContext(ctx context.Context) (credentials.Value, error) {
	v := credentials.Value{ProviderName: Ec2ProviderName}

	token, err := e.getToken(ctx)
	if err != nil {
		return v, err
	}

	role, err := e.getRoleName(ctx, token)
	if err != nil {
		return v, err
	}

	v, exp, err := e.getCredentials(ctx, token, role)
	if err != nil {
		return v, err
	}
	if !v.HasKeys() {
		return v, errors.New("failed to retrieve EC2 keys")
	}
	e.expiration = exp.Add(-e.expiryWindow)

	return v, err
}

// Retrieve retrieves the keys from the AWS service.
func (e *Ec2Provider) Retrieve() (credentials.Value, error) {
	return e.RetrieveWithContext(context.Background())
}

// IsExpired returns if the credentials have been retrieved.
func (e *Ec2Provider) IsExpired() bool {
	return e.expiration.Before(time.Now())
}
