// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongoaws

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

type stubProvider struct {
	creds aws.Credentials
	err   error
}

func (s *stubProvider) Retrieve(_ context.Context) (aws.Credentials, error) {
	return s.creds, s.err
}

func TestCredentialsRetrieve(t *testing.T) {
	p := &CredentialsProvider{
		provider: &stubProvider{
			creds: aws.Credentials{
				AccessKeyID:     "AKID",
				SecretAccessKey: "SECRET",
				SessionToken:    "Token",
			},
		},
	}

	creds, err := p.Retrieve(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got %q", err)
	}
	if e, a := "AKID", creds.AccessKeyID; e != a {
		t.Errorf("Expect access key ID to match, %q got %q", e, a)
	}
	if e, a := "SECRET", creds.SecretAccessKey; e != a {
		t.Errorf("Expect secret access key to match, %q got %q", e, a)
	}
	if e, a := "Token", creds.SessionToken; e != a {
		t.Errorf("Expect session token to match, %q got %q", e, a)
	}
}

func TestCredentialsRetrieveWithError(t *testing.T) {
	p := &CredentialsProvider{
		provider: &stubProvider{
			err: fmt.Errorf("provider error"),
		},
	}

	_, err := p.Retrieve(context.Background())
	if e, a := "provider error", err.Error(); e != a {
		t.Errorf("Expected provider error, %q got %q", e, a)
	}
}

func TestCredentialsExpire(t *testing.T) {
	stub := &stubProvider{}
	p := &CredentialsProvider{
		provider: stub,
	}

	if !p.Expired() {
		t.Errorf("Expected to start out expired")
	}

	_, err := p.Retrieve(context.Background())
	if err != nil {
		t.Errorf("Expected no err, got %v", err)
	}
	if p.Expired() {
		t.Errorf("Expected not to be expired")
	}
}

type stubProviderConcurrent struct {
	stubProvider
	done chan struct{}
}

func (s *stubProviderConcurrent) Retrieve(ctx context.Context) (aws.Credentials, error) {
	<-s.done
	return s.stubProvider.Retrieve(ctx)
}

func TestCredentialsRetrieveConcurrent(t *testing.T) {
	stub := &stubProviderConcurrent{
		done: make(chan struct{}),
	}

	p := &CredentialsProvider{
		provider: stub,
	}
	done := make(chan struct{})

	for range 2 {
		go func() {
			_, err := p.Retrieve(context.Background())
			if err != nil {
				t.Errorf("Expected no err, got %q", err)
			}
			done <- struct{}{}
		}()
	}

	// Validates that a single call to Retrieve is shared between two calls to Get
	stub.done <- struct{}{}
	<-done
	<-done
}
