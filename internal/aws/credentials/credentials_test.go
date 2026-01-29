// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
//
// Based on github.com/aws/aws-sdk-go by Amazon.com, Inc. with code from:
// - github.com/aws/aws-sdk-go/blob/v1.44.225/aws/credentials/credentials_test.go
// See THIRD-PARTY-NOTICES for original license terms

package credentials

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/internal/aws/awserr"
)

type stubProvider struct {
	creds Value
	err   error
}

func (s *stubProvider) Retrieve(_ context.Context) (Value, error) {
	return s.creds, s.err
}

func TestCredentialsGet(t *testing.T) {
	c := NewCredentials(&stubProvider{
		creds: Value{
			AccessKeyID:     "AKID",
			SecretAccessKey: "SECRET",
			SessionToken:    "",
		},
	})

	creds, err := c.Get(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if e, a := "AKID", creds.AccessKeyID; e != a {
		t.Errorf("Expect access key ID to match, %v got %v", e, a)
	}
	if e, a := "SECRET", creds.SecretAccessKey; e != a {
		t.Errorf("Expect secret access key to match, %v got %v", e, a)
	}
	if v := creds.SessionToken; len(v) != 0 {
		t.Errorf("Expect session token to be empty, %v", v)
	}
}

func TestCredentialsGetWithError(t *testing.T) {
	c := NewCredentials(&stubProvider{err: awserr.New("provider error", "", nil)})

	_, err := c.Get(context.Background())
	if e, a := "provider error", err.(awserr.Error).Code(); e != a {
		t.Errorf("Expected provider error, %v got %v", e, a)
	}
}

type stubProviderConcurrent struct {
	stubProvider
	done chan struct{}
}

func (s *stubProviderConcurrent) Retrieve(ctx context.Context) (Value, error) {
	<-s.done
	return s.stubProvider.Retrieve(ctx)
}

func TestCredentialsGetConcurrent(t *testing.T) {
	stub := &stubProviderConcurrent{
		done: make(chan struct{}),
	}

	c := NewCredentials(stub)
	done := make(chan struct{})

	for i := 0; i < 2; i++ {
		go func() {
			_, err := c.Get(context.Background())
			if err != nil {
				t.Errorf("Expected no err, got %v", err)
			}
			done <- struct{}{}
		}()
	}

	// Validates that a single call to Retrieve is shared between two calls to Get
	time.Sleep(10 * time.Millisecond)
	stub.done <- struct{}{}
	<-done
	<-done
}
