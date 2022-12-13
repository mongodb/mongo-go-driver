// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package creds

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/x/mongo/driver/auth/internal/awsv4"
)

type ecsResponse struct {
	AccessKeyID     string `json:"AccessKeyId"`
	SecretAccessKey string `json:"SecretAccessKey"`
	Token           string `json:"Token"`
}

const (
	awsRelativeURI     = "http://169.254.170.2/"
	awsEC2URI          = "http://169.254.169.254/"
	awsEC2RolePath     = "latest/meta-data/iam/security-credentials/"
	awsEC2TokenPath    = "latest/api/token"
	defaultHTTPTimeout = 10 * time.Second
)

// AwsCredentials contains AWS credential fields.
type AwsCredentials struct {
	Username string
	Password string
	Token    string
}

// ValidateAndMakeCredentials validates credential fields and packs them into awsv4.StaticProvider.
func (ac AwsCredentials) ValidateAndMakeCredentials() (*awsv4.StaticProvider, error) {
	if ac.Username != "" && ac.Password == "" {
		return nil, errors.New("ACCESS_KEY_ID is set, but SECRET_ACCESS_KEY is missing")
	}
	if ac.Username == "" && ac.Password != "" {
		return nil, errors.New("SECRET_ACCESS_KEY is set, but ACCESS_KEY_ID is missing")
	}
	if ac.Username == "" && ac.Password == "" && ac.Token != "" {
		return nil, errors.New("AWS_SESSION_TOKEN is set, but ACCESS_KEY_ID and SECRET_ACCESS_KEY are missing")
	}
	if ac.Username != "" || ac.Password != "" || ac.Token != "" {
		return &awsv4.StaticProvider{Value: awsv4.Value{
			AccessKeyID:     ac.Username,
			SecretAccessKey: ac.Password,
			SessionToken:    ac.Token,
		}}, nil
	}
	return nil, nil
}

// AwsCredentialProvider provides AWS credentials.
type AwsCredentialProvider struct {
	HTTPClient *http.Client
}

func executeAWSHTTPRequest(ctx context.Context, httpClient *http.Client, req *http.Request) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultHTTPTimeout)
	defer cancel()
	resp, err := httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

// getEC2Credentials generates EC2 credentials.
func (p *AwsCredentialProvider) getEC2Credentials(ctx context.Context) (*awsv4.StaticProvider, error) {
	// get token
	req, err := http.NewRequest(http.MethodPut, awsEC2URI+awsEC2TokenPath, nil)
	if err != nil {
		return nil, err
	}
	const defaultEC2TTLSeconds = "30"
	req.Header.Set("X-aws-ec2-metadata-token-ttl-seconds", defaultEC2TTLSeconds)

	token, err := executeAWSHTTPRequest(ctx, p.HTTPClient, req)
	if err != nil {
		return nil, err
	}
	if len(token) == 0 {
		return nil, errors.New("unable to retrieve token from EC2 metadata")
	}
	tokenStr := string(token)

	// get role name
	req, err = http.NewRequest(http.MethodGet, awsEC2URI+awsEC2RolePath, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-aws-ec2-metadata-token", tokenStr)

	role, err := executeAWSHTTPRequest(ctx, p.HTTPClient, req)
	if err != nil {
		return nil, err
	}
	if len(role) == 0 {
		return nil, errors.New("unable to retrieve role_name from EC2 metadata")
	}

	// get credentials
	pathWithRole := awsEC2URI + awsEC2RolePath + string(role)
	req, err = http.NewRequest(http.MethodGet, pathWithRole, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-aws-ec2-metadata-token", tokenStr)
	creds, err := executeAWSHTTPRequest(ctx, p.HTTPClient, req)
	if err != nil {
		return nil, err
	}

	var es2Resp ecsResponse
	err = json.Unmarshal(creds, &es2Resp)
	if err != nil {
		return nil, err
	}
	return AwsCredentials{
		Username: es2Resp.AccessKeyID,
		Password: es2Resp.SecretAccessKey,
		Token:    es2Resp.Token,
	}.ValidateAndMakeCredentials()
}

// GetCredentials generates AWS credentials.
func (p *AwsCredentialProvider) GetCredentials(ctx context.Context) (*awsv4.StaticProvider, error) {
	// Credentials from environment variables
	ac := AwsCredentials{
		Username: os.Getenv("AWS_ACCESS_KEY_ID"),
		Password: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		Token:    os.Getenv("AWS_SESSION_TOKEN"),
	}

	creds, err := ac.ValidateAndMakeCredentials()
	if creds != nil || err != nil {
		return creds, err
	}

	// Credentials from ECS metadata
	relativeEcsURI := os.Getenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI")
	if len(relativeEcsURI) > 0 {
		fullURI := awsRelativeURI + relativeEcsURI

		req, err := http.NewRequest(http.MethodGet, fullURI, nil)
		if err != nil {
			return nil, err
		}

		body, err := executeAWSHTTPRequest(ctx, p.HTTPClient, req)
		if err != nil {
			return nil, err
		}

		var espResp ecsResponse
		err = json.Unmarshal(body, &espResp)
		if err != nil {
			return nil, err
		}
		ac.Username = espResp.AccessKeyID
		ac.Password = espResp.SecretAccessKey
		ac.Token = espResp.Token

		creds, err = ac.ValidateAndMakeCredentials()
		if creds != nil || err != nil {
			return creds, err
		}
	}

	// Credentials from EC2 metadata
	creds, err = p.getEC2Credentials(context.Background())
	if creds == nil && err == nil {
		return nil, errors.New("unable to get credentials")
	}
	return creds, err
}
