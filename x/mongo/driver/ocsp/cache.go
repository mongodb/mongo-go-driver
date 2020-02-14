// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ocsp

import (
	"crypto"
	"sync"
	"time"

	"golang.org/x/crypto/ocsp"
)

type cacheKey struct {
	HashAlgorithm  crypto.Hash
	IssuerNameHash string
	IssuerKeyHash  string
	SerialNumber   string
}

// Cache represents an OCSP cache.
type Cache interface {
	Put(*ocsp.Request, *Response) *Response
	Get(request *ocsp.Request) *Response
}

// ConcurrentCache is an implementation of ocsp.Cache that's safe for concurrent use.
type ConcurrentCache struct {
	cache map[cacheKey]*Response
	sync.Mutex
}

// NewCache creates an empty OCSP cache.
func NewCache() *ConcurrentCache {
	return &ConcurrentCache{
		cache: make(map[cacheKey]*Response),
	}
}

// Put stores the given request into the cache. If the request is already in the cache, the existing entry will be
// replaced.
func (c *ConcurrentCache) Put(request *ocsp.Request, response *Response) *Response {
	unknown := response.Status == ocsp.Unknown
	hasUpdateTime := !response.NextUpdate.IsZero()
	canBeCached := !unknown && hasUpdateTime
	key := createCacheKey(request)

	c.Lock()
	defer c.Unlock()

	current, ok := c.cache[key]
	if !ok {
		if canBeCached {
			c.cache[key] = response
		}

		// Return the provided response even though it might not have been cached because it's the most up-to-date
		// response available.
		return response
	}

	// If the new response is Unknown, we can't cache it. Return the existing cached response.
	if unknown {
		return current
	}

	// If a response has no nextUpdate set, the responder is telling us that newer information is always available.
	// In this case, remove the existing cache entry because it is stale and return the new response because it is
	// more up-to-date.
	if !hasUpdateTime {
		delete(c.cache, key)
		return response
	}

	// If we get here, the new response is conclusive and has a non-empty nextUpdate so it can be cached. Overwrite
	// the existing cache entry if the new one will be valid for longer.
	newest := current
	if response.NextUpdate.After(current.NextUpdate) {
		c.cache[key] = response
		newest = response
	}
	return newest
}

// Get returns the cached response for the request, or nil if there is no cached response. If the cached response has
// expired, it will be removed from the cache and nil will be returned.
func (c *ConcurrentCache) Get(request *ocsp.Request) *Response {
	c.Lock()
	defer c.Unlock()

	key := createCacheKey(request)
	response, ok := c.cache[key]
	if !ok {
		return nil
	}

	if time.Now().UTC().Before(response.NextUpdate) {
		return response
	}
	delete(c.cache, key)
	return nil
}

func createCacheKey(request *ocsp.Request) cacheKey {
	return cacheKey{
		HashAlgorithm:  request.HashAlgorithm,
		IssuerNameHash: string(request.IssuerNameHash), // TODO: is simple stringification ok here?
		IssuerKeyHash:  string(request.IssuerKeyHash),
		SerialNumber:   request.SerialNumber.String(),
	}
}
