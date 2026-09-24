// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package prose

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/internal/aws/credentials"
	v4signer "go.mongodb.org/mongo-driver/v2/internal/aws/signer/v4"
	"go.mongodb.org/mongo-driver/v2/internal/credproviders"
)

// TestAWSCredentialsAuthenticate checks that the AWS credentials exported by
// the SSO login are accepted by AWS, by calling STS GetCallerIdentity with
// them.
func TestAWSCredentialsAuthenticate(t *testing.T) {
	path, err := exportSecrets()
	if err != nil {
		t.Fatalf("failed to export secrets: %v", err)
	}

	secrets, err := readSecrets(path)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name         string
		accessKeyID  string
		secretKey    string
		sessionToken string
	}{
		{
			name:         "session credentials",
			accessKeyID:  "AWS_ACCESS_KEY_ID",
			secretKey:    "AWS_SECRET_ACCESS_KEY",
			sessionToken: "AWS_SESSION_TOKEN",
		},
		{
			name:        "CSFLE credentials",
			accessKeyID: "FLE_AWS_KEY",
			secretKey:   "FLE_AWS_SECRET",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := credentials.Value{
				AccessKeyID:     secrets[test.accessKeyID],
				SecretAccessKey: secrets[test.secretKey],
			}
			if test.sessionToken != "" {
				value.SessionToken = secrets[test.sessionToken]
			}
			if value.AccessKeyID == "" || value.SecretAccessKey == "" {
				t.Fatalf("%s or %s missing from %s", test.accessKeyID, test.secretKey, secretsFileName)
			}

			if err := getCallerIdentity(value); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func getCallerIdentity(value credentials.Value) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	const body = "Action=GetCallerIdentity&Version=2011-06-15"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://sts.amazonaws.com/", strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	creds := credentials.NewCredentials(&credproviders.StaticProvider{Value: value})
	if _, err := v4signer.NewSigner(creds).Sign(req, strings.NewReader(body), "sts", "us-east-1", time.Now()); err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// The error body names the failure (e.g. ExpiredToken) and holds no
		// credential material.
		msg, _ := io.ReadAll(resp.Body)
		return &stsError{status: resp.Status, body: string(msg)}
	}

	return nil
}

type stsError struct {
	status string
	body   string
}

func (e *stsError) Error() string {
	return "STS GetCallerIdentity returned " + e.status + ": " + e.body
}
