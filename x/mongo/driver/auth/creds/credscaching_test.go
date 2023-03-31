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
	"sync/atomic"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/aws/credentials"
	"go.mongodb.org/mongo-driver/internal/credproviders"
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

func TestAWSCredentialProviderCaching(t *testing.T) {
	const (
		urienv         = "TEST_CONTAINER_CREDENTIALS_RELATIVE_URI"
		keyenv         = "TEST_ACCESS_KEY"
		awsRelativeURI = "http://169.254.170.2/"
		testEndpoint   = "foo"
		param          = "source"
	)

	t.Setenv(urienv, testEndpoint)

	testCases := []struct {
		expiration time.Duration
		reqCount   uint32
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
			var cnt uint32
			// the test server counts the requests and replies mock responses.
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get(param) != awsRelativeURI+testEndpoint {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				atomic.AddUint32(&cnt, 1)
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

			env := credproviders.NewEnvProvider()
			env.AwsAccessKeyIDEnv = credproviders.EnvVar(keyenv)
			ecs := credproviders.NewECSProvider(client, expiryWindow)
			ecs.AwsContainerCredentialsRelativeURIEnv = credproviders.EnvVar(urienv)

			p := AWSCredentialProvider{credentials.NewChainCredentials([]credentials.Provider{env, ecs})}
			var err error
			_, err = p.GetCredentialsDoc(context.Background())
			assert.NoError(t, err, "error in GetCredentialsDoc")
			_, err = p.GetCredentialsDoc(context.Background())
			assert.NoError(t, err, "error in GetCredentialsDoc")
			assert.Equal(t, tc.reqCount, atomic.LoadUint32(&cnt), "expected and actual credentials retrieval count don't match")
		})
	}
}
