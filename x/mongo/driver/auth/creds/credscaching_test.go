// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package creds

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/x/mongo/driver/auth/credproviders"
)

type pipeTransport struct {
	url    string
	param  string
	client *http.Client
}

// RoundTrip reassembles the original request URI into the query parameter and forwards the request.
func (t pipeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	uri, err := url.Parse(t.url)
	if err != nil {
		return nil, err
	}
	values := uri.Query()
	values.Add(t.param, req.URL.String())
	uri.RawQuery = values.Encode()
	req.URL = uri
	return t.client.Do(req)
}

func TestAwsCredentialProviderCaching(t *testing.T) {
	const (
		urienv         = "TEST_CONTAINER_CREDENTIALS_RELATIVE_URI"
		keyenv         = "TEST_ACCESS_KEY"
		awsRelativeURI = "http://169.254.170.2/"
		testEndpoint   = "foo"
		param          = "source"
	)

	credproviders.AwsContainerCredentialsRelativeURIEnv = credproviders.EnvVar(urienv)
	os.Setenv(urienv, testEndpoint)
	credproviders.AwsAccessKeyIDEnv = credproviders.EnvVar(keyenv)
	defer os.Unsetenv(urienv)

	testCases := []struct {
		expiration time.Duration
		reqCount   int
	}{
		{
			expiration: 20 * time.Minute,
			reqCount:   1,
		},
		{
			expiration: 5 * time.Minute,
			reqCount:   2,
		},
		{
			expiration: -1 * time.Minute,
			reqCount:   2,
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("expires in %s", tc.expiration.String()), func(t *testing.T) {
			var cnt int
			// the test server counts the requests and replies mock responses.
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get(param) != awsRelativeURI+testEndpoint {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				cnt++
				t := time.Now().Add(tc.expiration).Format(time.RFC3339)
				_, err := io.WriteString(w, fmt.Sprintf(`{
					"AccessKeyId": "id",
					"SecretAccessKey": "key",
					"Token": "token",
					"Expiration": "%s"
				}`, t))
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
				}
			}))
			defer ts.Close()

			client := &http.Client{
				Transport: pipeTransport{
					url:    ts.URL,
					param:  param,
					client: ts.Client(),
				},
			}

			p := NewAWSCredentialProvider(client)
			var err error
			_, err = p.GetCredentialsDoc(context.Background())
			assert.Nil(t, err, "error in GetCredentialsDoc: %v", err)
			_, err = p.GetCredentialsDoc(context.Background())
			assert.Nil(t, err, "error in GetCredentialsDoc: %v", err)
			assert.Equal(t, tc.reqCount, cnt, "expected retrieval count: %d, actual: %d", tc.reqCount, cnt)
		})
	}
}
