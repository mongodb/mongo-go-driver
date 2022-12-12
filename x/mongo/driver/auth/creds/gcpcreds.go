// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package creds

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"go.mongodb.org/mongo-driver/internal"
)

// GcpCredentialProvider provides GCP credentials.
type GcpCredentialProvider struct {
	HTTPClient *http.Client
}

// GetCredentials generates GCP credentials.
func (p *GcpCredentialProvider) GetCredentials(ctx context.Context) (string, error) {
	metadataHost := "metadata.google.internal"
	if envhost := os.Getenv("GCE_METADATA_HOST"); envhost != "" {
		metadataHost = envhost
	}
	url := fmt.Sprintf("http://%s/computeMetadata/v1/instance/service-accounts/default/token", metadataHost)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", internal.WrapErrorf(err, "unable to retrieve GCP credentials")
	}
	req.Header.Set("Metadata-Flavor", "Google")
	resp, err := p.HTTPClient.Do(req.WithContext(ctx))
	if err != nil {
		return "", internal.WrapErrorf(err, "unable to retrieve GCP credentials")
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", internal.WrapErrorf(err, "unable to retrieve GCP credentials: error reading response body")
	}
	if resp.StatusCode != http.StatusOK {
		return "", internal.WrapErrorf(err, "unable to retrieve GCP credentials: expected StatusCode 200, got StatusCode: %v. Response body: %s", resp.StatusCode, body)
	}
	var tokenResponse struct {
		AccessToken string `json:"access_token"`
	}
	// Attempt to read body as JSON
	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		return "", internal.WrapErrorf(err, "unable to retrieve GCP credentials: error reading body JSON. Response body: %s", body)
	}
	if tokenResponse.AccessToken == "" {
		return "", fmt.Errorf("unable to retrieve GCP credentials: got unexpected empty accessToken from GCP Metadata Server. Response body: %s", body)
	}
	return tokenResponse.AccessToken, nil
}
