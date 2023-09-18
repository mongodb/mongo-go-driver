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
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/internal/aws/awserr"
)

func isExpired(c *Credentials) bool {
	c.m.RLock()
	defer c.m.RUnlock()

	return c.isExpiredLocked(c.creds)
}

type stubProvider struct {
	creds          Value
	retrievedCount int
	expired        bool
	err            error
}

func (s *stubProvider) Retrieve() (Value, error) {
	s.retrievedCount++
	s.expired = false
	s.creds.ProviderName = "stubProvider"
	return s.creds, s.err
}
func (s *stubProvider) IsExpired() bool {
	return s.expired
}

func TestCredentialsGet(t *testing.T) {
	c := NewCredentials(&stubProvider{
		creds: Value{
			AccessKeyID:     "AKID",
			SecretAccessKey: "SECRET",
			SessionToken:    "",
		},
		expired: true,
	})

	creds, err := c.GetWithContext(context.Background())
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
	c := NewCredentials(&stubProvider{err: awserr.New("provider error", "", nil), expired: true})

	_, err := c.GetWithContext(context.Background())
	if e, a := "provider error", err.(awserr.Error).Code(); e != a {
		t.Errorf("Expected provider error, %v got %v", e, a)
	}
}

func TestCredentialsExpire(t *testing.T) {
	stub := &stubProvider{}
	c := NewCredentials(stub)

	stub.expired = false
	if !isExpired(c) {
		t.Errorf("Expected to start out expired")
	}

	_, err := c.GetWithContext(context.Background())
	if err != nil {
		t.Errorf("Expected no err, got %v", err)
	}
	if isExpired(c) {
		t.Errorf("Expected not to be expired")
	}

	stub.expired = true
	if !isExpired(c) {
		t.Errorf("Expected to be expired")
	}
}

func TestCredentialsGetWithProviderName(t *testing.T) {
	stub := &stubProvider{}

	c := NewCredentials(stub)

	creds, err := c.GetWithContext(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if e, a := creds.ProviderName, "stubProvider"; e != a {
		t.Errorf("Expected provider name to match, %v got %v", e, a)
	}
}

type MockProvider struct {
	// The date/time when to expire on
	expiration time.Time

	// If set will be used by IsExpired to determine the current time.
	// Defaults to time.Now if CurrentTime is not set.  Available for testing
	// to be able to mock out the current time.
	CurrentTime func() time.Time
}

// IsExpired returns if the credentials are expired.
func (e *MockProvider) IsExpired() bool {
	curTime := e.CurrentTime
	if curTime == nil {
		curTime = time.Now
	}
	return e.expiration.Before(curTime())
}

func (*MockProvider) Retrieve() (Value, error) {
	return Value{}, nil
}

func TestCredentialsIsExpired_Race(_ *testing.T) {
	creds := NewChainCredentials([]Provider{&MockProvider{}})

	starter := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			<-starter
			for i := 0; i < 100; i++ {
				isExpired(creds)
			}
		}()
	}
	close(starter)

	wg.Wait()
}

type stubProviderConcurrent struct {
	stubProvider
	done chan struct{}
}

func (s *stubProviderConcurrent) Retrieve() (Value, error) {
	<-s.done
	return s.stubProvider.Retrieve()
}

func TestCredentialsGetConcurrent(t *testing.T) {
	stub := &stubProviderConcurrent{
		done: make(chan struct{}),
	}

	c := NewCredentials(stub)
	done := make(chan struct{})

	for i := 0; i < 2; i++ {
		go func() {
			_, err := c.GetWithContext(context.Background())
			if err != nil {
				t.Errorf("Expected no err, got %v", err)
			}
			done <- struct{}{}
		}()
	}

	// Validates that a single call to Retrieve is shared between two calls to Get
	stub.done <- struct{}{}
	<-done
	<-done
}
