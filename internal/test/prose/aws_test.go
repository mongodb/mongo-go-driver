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
	"time"

	"go.mongodb.org/mongo-driver/v2/internal/aws/credentials"
	v4signer "go.mongodb.org/mongo-driver/v2/internal/aws/signer/v4"
	"go.mongodb.org/mongo-driver/v2/internal/credproviders"

	"github.com/joho/godotenv"
)

// getCallerIdentity calls STS GetCallerIdentity, which succeeds for any valid
// credentials regardless of their permissions.
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
