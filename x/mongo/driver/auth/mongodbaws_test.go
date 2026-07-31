// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/aws/credentials"
	"go.mongodb.org/mongo-driver/v2/internal/require"
)

func TestGetRegion(t *testing.T) {
	longHost := make([]rune, 256)
	emptyErr := errors.New("invalid STS host: empty")
	tooLongErr := errors.New("invalid STS host: too large")
	emptyPartErr := errors.New("invalid STS host: empty part")
	testCases := []struct {
		name   string
		host   string
		err    error
		region string
	}{
		{"success default", "sts.amazonaws.com", nil, "us-east-1"},
		{"success parse", "first.second", nil, "second"},
		{"success no region", "first", nil, "us-east-1"},
		{"error host too long", string(longHost), tooLongErr, ""},
		{"error host empty", "", emptyErr, ""},
		{"error empty middle part", "abc..def", emptyPartErr, ""},
		{"error empty part", "first.", emptyPartErr, ""},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reg, err := getRegion(tc.host)
			if tc.err == nil {
				assert.Nil(t, err, "error getting region: %v", err)
				assert.Equal(t, tc.region, reg, "expected %v, got %v", tc.region, reg)
				return
			}
			assert.NotNil(t, err, "expected error, got nil")
			assert.Equal(t, err, tc.err, "expected error: %v, got: %v", tc.err, err)
		})
	}
}

type testAWSCredentialsProvider struct {
	cnt int
}

func (a *testAWSCredentialsProvider) Retrieve(_ context.Context) (credentials.Value, error) {
	a.cnt++
	return credentials.Value{}, nil
}

func TestAWSCustomCredentialsProvider(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "AWS_ACCESS_KEY_ID")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "AWS_SECRET_ACCESS_KEY")

	provider := &testAWSCredentialsProvider{}
	for _, tc := range []struct {
		name string
		cred *Cred
		cnt  int
	}{
		{
			name: "provider with cred",
			cred: &Cred{
				Username:               "user",
				Password:               "pass",
				Props:                  map[string]string{"AWS_SESSION_TOKEN": "token"},
				AWSCredentialsProvider: provider,
			},
			cnt: 0,
		},
		{
			name: "provider with empty cred",
			cred: &Cred{
				AWSCredentialsProvider: provider,
			},
			cnt: 1,
		},
	} {
		provider.cnt = 0
		t.Run(tc.name, func(t *testing.T) {
			authenticator, err := newMongoDBAWSAuthenticator(
				tc.cred,
				&http.Client{},
			)
			require.NoErrorf(t, err, "unexpected error %v", err)

			_, err = authenticator.(*MongoDBAWSAuthenticator).credentials.Get(context.Background())
			require.NoError(t, err, "unexpected error getting credentials: %v", err)
			require.Equalf(t, tc.cnt, provider.cnt, "expected provider to be called %v times but got %v", tc.cnt, provider.cnt)
		})
	}
}
