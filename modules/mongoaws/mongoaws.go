// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongoaws

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"golang.org/x/sync/singleflight"

	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	// emptyStringSHA256 is a SHA256 of an empty string
	emptyStringSHA256 = `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
)

var _ options.AWSCredentialsProvider = (*CredentialsProvider)(nil)

// CredentialsProvider is an implementation of options.AWSCredentialsProvider in
// MongoDB Go driver. It provides caching and concurrency safe credentials retrieval
// via the provider's retrieve method.
type CredentialsProvider struct {
	sf       singleflight.Group
	m        sync.RWMutex
	provider aws.CredentialsProvider
	creds    *aws.Credentials
}

// NewCredentialsProvider returns a CredentialsProvider that wraps AWS
// CredentialsProvider. Provider is expected to not be nil.
func NewCredentialsProvider(provider aws.CredentialsProvider) *CredentialsProvider {
	return &CredentialsProvider{
		provider: provider,
	}
}

// Retrieve returns the credentials. If the credentials have already been
// retrieved, and not expired the cached credentials will be returned. If the
// credentials have not been retrieved yet, or expired the provider's Retrieve
// method will be called.
func (p *CredentialsProvider) Retrieve(ctx context.Context) (options.AWSCredentials, error) {
	// Check if credentials are cached, and not expired.
	select {
	case curCreds, ok := <-p.asyncIsExpired():
		// ok will only be true, if the credentials were not expired. ok will
		// be false and have no value if the credentials are expired.
		if ok {
			return options.AWSCredentials(curCreds), nil
		}
	case <-ctx.Done():
		return options.AWSCredentials{}, ctx.Err()
	}

	// Cannot pass context down to the actual retrieve, because the first
	// context would cancel the whole group when there is not direct
	// association of items in the group.
	resCh := p.sf.DoChan("", func() (any, error) {
		return p.singleRetrieve(&suppressedContext{ctx})
	})
	select {
	case res := <-resCh:
		if res.Err != nil {
			return options.AWSCredentials{}, res.Err
		}
		creds := res.Val.(aws.Credentials)
		return options.AWSCredentials(creds), nil
	case <-ctx.Done():
		return options.AWSCredentials{}, ctx.Err()
	}
}

func (p *CredentialsProvider) singleRetrieve(ctx context.Context) (any, error) {
	p.m.Lock()
	defer p.m.Unlock()

	if !p.isExpired() {
		curCreds := *p.creds
		return curCreds, nil
	}

	creds, err := p.provider.Retrieve(ctx)
	if err == nil {
		curCreds := creds
		p.creds = &curCreds
	}

	return creds, err
}

// asyncIsExpired returns a channel of credentials Value. If the channel is
// closed the credentials are expired and credentials value are not empty.
func (p *CredentialsProvider) asyncIsExpired() <-chan aws.Credentials {
	ch := make(chan aws.Credentials, 1)
	go func() {
		p.m.RLock()
		defer p.m.RUnlock()

		if !p.isExpired() {
			curCreds := *p.creds
			ch <- curCreds
		}

		close(ch)
	}()

	return ch
}

func (p *CredentialsProvider) isExpired() bool {
	return p.creds == nil || p.creds.Expired()
}

// Expired returns if the cached credentials are no longer valid, and need
// to be retrieved.
func (p *CredentialsProvider) Expired() bool {
	p.m.RLock()
	defer p.m.RUnlock()
	return p.isExpired()
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

var _ options.AWSSigner = (*Signer)(nil)

// Signer is an implementation of options.AWSSigner in MongoDB Go driver.
type Signer struct {
	signer v4.HTTPSigner
}

// NewSigner creates a new Signer from the provided AWS HTTPSigner.
func NewSigner(httpSigner v4.HTTPSigner) Signer {
	return Signer{
		signer: httpSigner,
	}
}

// Sign signs AWS v4 requests.
func (s Signer) Sign(
	ctx context.Context, creds options.AWSCredentials, r *http.Request,
	payload, service, region string, signingTime time.Time) error {
	if len(payload) == 0 {
		payload = emptyStringSHA256
	} else {
		hash := sha256.Sum256([]byte(payload))
		payload = hex.EncodeToString(hash[:])
	}
	r.Header.Set("X-Amz-Security-Token", creds.SessionToken)
	return s.signer.SignHTTP(ctx, aws.Credentials(creds), r, payload, service, region, signingTime)
}
