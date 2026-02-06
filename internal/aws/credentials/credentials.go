// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
//
// Based on github.com/aws/aws-sdk-go by Amazon.com, Inc. with code from:
// - github.com/aws/aws-sdk-go/blob/v1.44.225/aws/credentials/credentials.go
// See THIRD-PARTY-NOTICES for original license terms

package credentials

import (
	"context"
	"sync/atomic"
	"time"

	"go.mongodb.org/mongo-driver/v2/internal/aws/awserr"
	"golang.org/x/sync/singleflight"
)

// A Value is the AWS credentials value for individual credential fields.
//
// A Value is also used to represent Azure credentials.
// Azure credentials only consist of an access token, which is stored in the `SessionToken` field.
type Value struct {
	// AWS Access key ID
	AccessKeyID string

	// AWS Secret Access Key
	SecretAccessKey string

	// AWS Session Token
	SessionToken string

	// Source of the credentials
	Source string

	// States if the credentials can expire or not.
	CanExpire bool

	// The time the credentials will expire at. Should be ignored if CanExpire
	// is false.
	Expires time.Time

	// The ID of the account for the credentials.
	AccountID string
}

// Expired returns if the credentials have expired.
func (v Value) Expired() bool {
	if v.CanExpire {
		// Calling Round(0) on the current time will truncate the monotonic
		// reading only. Ensures credential expiry time is always based on
		// reported wall-clock time.
		return !v.Expires.After(time.Now().Round(0))
	}

	return false
}

// HasKeys returns if the credentials Value has both AccessKeyID and
// SecretAccessKey value set.
func (v Value) HasKeys() bool {
	return len(v.AccessKeyID) != 0 && len(v.SecretAccessKey) != 0
}

// A Provider is the interface for any component which will provide credentials
// Value. A provider is required to manage its own Expired state, and what to
// be expired means.
//
// The Provider should not need to implement its own mutexes, because
// that will be managed by Credentials.
type Provider interface {
	// Retrieve returns nil if it successfully retrieved the value.
	// Error is returned if the value were not obtainable, or empty.
	Retrieve(context.Context) (Value, error)
}

// A Credentials provides concurrency safe retrieval of AWS credentials Value.
//
// A Credentials is also used to fetch Azure credentials Value.
//
// Credentials will cache the credentials value until they expire. Once the value
// expires the next Get will attempt to retrieve valid credentials.
//
// Credentials is safe to use across multiple goroutines and will manage the
// synchronous state so the Providers do not need to implement their own
// synchronization.
//
// The first Credentials.Get() will always call Provider.Retrieve() to get the
// first instance of the credentials Value. All calls to Get() after that
// will return the cached credentials Value until Expired() returns true.
type Credentials struct {
	provider Provider

	creds atomic.Value
	sf    singleflight.Group
}

// NewCredentials returns a pointer to a new Credentials with the provider set.
func NewCredentials(provider Provider) *Credentials {
	c := &Credentials{
		provider: provider,
	}
	return c
}

// Get returns the credentials value, or error if the credentials
// Value failed to be retrieved. Will return early if the passed in context is
// canceled.
//
// Will return the cached credentials Value if it has not expired. If the
// credentials Value has expired the Provider's Retrieve() will be called
// to refresh the credentials.
func (c *Credentials) Get(ctx context.Context) (Value, error) {
	// Cannot pass context down to the actual retrieve, because the first
	// context would cancel the whole group when there is not direct
	// association of items in the group.
	resCh := c.sf.DoChan("", func() (any, error) {
		return c.singleRetrieve(&suppressedContext{ctx})
	})
	select {
	case res := <-resCh:
		return res.Val.(Value), res.Err
	case <-ctx.Done():
		return Value{}, awserr.New("RequestCanceled",
			"request context canceled", ctx.Err())
	}
}

func (c *Credentials) singleRetrieve(ctx context.Context) (any, error) {
	if currCreds, ok := c.getCreds(); ok && !currCreds.Expired() {
		return currCreds, nil
	}

	newCreds, err := c.provider.Retrieve(ctx)
	if err == nil {
		c.creds.Store(&newCreds)
	}

	return newCreds, err
}

// getCreds returns the currently stored credentials and true. Returning false
// if no credentials were stored.
func (c *Credentials) getCreds() (Value, bool) {
	v := c.creds.Load()
	if v == nil {
		return Value{}, false
	}

	val := v.(*Value)
	if val == nil || !val.HasKeys() {
		return Value{}, false
	}

	return *val, true
}

type suppressedContext struct {
	context.Context
}

func (s *suppressedContext) Deadline() (deadline time.Time, ok bool) {
	return time.Time{}, false
}

func (s *suppressedContext) Done() <-chan struct{} {
	return nil
}

func (s *suppressedContext) Err() error {
	return nil
}
