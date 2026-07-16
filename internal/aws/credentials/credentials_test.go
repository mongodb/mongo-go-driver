// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
//
// Loosely based on github.com/aws/aws-sdk-go-v2 by Amazon.com, Inc. with code from:
// - github.com/aws/aws-sdk-go-v2/blob/v1.28.0/aws/credential_cache_test.go
// See THIRD-PARTY-NOTICES for original license terms

package credentials

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

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

type stubConcurrentProvider struct {
	called uint32
	done   chan struct{}
}

func (s *stubConcurrentProvider) Retrieve(ctx context.Context) (Value, error) {
	atomic.AddUint32(&s.called, 1)
	<-s.done
	return Value{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}, nil
}

func TestCredentialsGetConcurrent(t *testing.T) {
	stub := &stubConcurrentProvider{
		done: make(chan struct{}),
	}

	c := NewCredentials(stub)

	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			_, err := c.Get(context.Background())
			if err != nil {
				t.Errorf("Expected no err, got %v", err)
			}
			wg.Done()
		}()
	}

	// Validates that a single call to Retrieve is shared between two calls to
	// Get method call
	stub.done <- struct{}{}
	wg.Wait()

	if e, a := uint32(1), atomic.LoadUint32(&stub.called); e != a {
		t.Errorf("expected %v, got %v", e, a)
	}
}
