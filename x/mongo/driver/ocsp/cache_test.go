// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ocsp

import (
	"context"
	"crypto"
	"crypto/tls"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/internal/assert"
	"golang.org/x/crypto/ocsp"
)

var (
	ctx = context.Background()
)

func TestCache(t *testing.T) {
	testRequest := &ocsp.Request{
		HashAlgorithm:  crypto.SHA1,
		IssuerNameHash: []byte("issuerNameHash"),
		IssuerKeyHash:  []byte("issuerKeyHash"),
	}
	testRequestKey := createCacheKey(testRequest)

	t.Run("verification errors if cache is nil", func(t *testing.T) {
		err := Verify(ctx, tls.ConnectionState{}, &VerifyOptions{})
		assert.NotNil(t, err, "expected error, got nil")

		ocspErr, ok := err.(*Error)
		assert.True(t, ok, "expected error of type %T, got %v of type %T", &Error{}, err, err)
		expected := &Error{
			wrapped: errors.New("no OCSP cache provided"),
		}
		assert.Equal(t, expected.Error(), ocspErr.Error(), "expected error %v, got %v", expected, ocspErr)
	})
	t.Run("put", func(t *testing.T) {
		t.Run("empty cache", func(t *testing.T) {
			good := &ResponseDetails{
				Status:     ocsp.Good,
				NextUpdate: futureTime(10),
			}
			revoked := &ResponseDetails{
				Status:     ocsp.Revoked,
				NextUpdate: futureTime(10),
			}
			unknown := &ResponseDetails{
				Status:     ocsp.Unknown,
				NextUpdate: futureTime(10),
			}
			goodNoUpdate := &ResponseDetails{Status: ocsp.Good}
			revokedNoUpdate := &ResponseDetails{Status: ocsp.Revoked}
			unknownNoUpdate := &ResponseDetails{Status: ocsp.Unknown}

			testCases := []struct {
				name     string
				response *ResponseDetails
				cached   bool
			}{
				{"good response is cached", good, true},
				{"revoked response is cached", revoked, true},
				{"good response without nextUpdate is not cached", goodNoUpdate, false},
				{"revoked response without nextUpdate is not cached", revokedNoUpdate, false},
				{"unknown response with nextUpdate is not cached", unknown, false},
				{"unknown response without nextUpdate is not cached", unknownNoUpdate, false},
			}
			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					cache := NewCache()
					putRes := cache.Update(testRequest, tc.response)
					// Put should always return tc.response even if it wasn't added to the cache because it is the most
					// up-to-date response.
					assert.Equal(t, tc.response, putRes, "expected Put to return %v, got %v", tc.response, putRes)

					current, ok := cache.cache[testRequestKey]
					if tc.cached {
						assert.True(t, ok, "expected cache to contain entry but did not")
						assert.Equal(t, tc.response, current, "expected cache to contain %v, got %v", tc.response, current)
						return
					}
					assert.False(t, ok, "expected cache to contain nil, got %v", current)
					assert.Nil(t, current, "expected cache to contain no entry, got %v", current)
				})
			}
		})
		t.Run("non-empty cache", func(t *testing.T) {
			cachedUpdateMinutes := 10
			originalCached := &ResponseDetails{
				Status:     ocsp.Good,
				NextUpdate: futureTime(cachedUpdateMinutes),
			}

			unknownUpdate := &ResponseDetails{
				Status:     ocsp.Unknown,
				NextUpdate: futureTime(cachedUpdateMinutes + 5),
			}
			goodNoUpdate := &ResponseDetails{Status: ocsp.Good}
			revokedNoUpdate := &ResponseDetails{Status: ocsp.Revoked}
			goodEarlierUpdate := &ResponseDetails{
				Status:     ocsp.Good,
				NextUpdate: futureTime(cachedUpdateMinutes - 5),
			}
			revokedEarlierUpdate := &ResponseDetails{
				Status:     ocsp.Revoked,
				NextUpdate: futureTime(cachedUpdateMinutes - 5),
			}
			goodLaterUpdate := &ResponseDetails{
				Status:     ocsp.Good,
				NextUpdate: futureTime(cachedUpdateMinutes + 5),
			}
			revokedLaterUpdate := &ResponseDetails{
				Status:     ocsp.Revoked,
				NextUpdate: futureTime(cachedUpdateMinutes + 5),
			}

			testCases := []struct {
				name      string
				response  *ResponseDetails
				putReturn *ResponseDetails
				cached    *ResponseDetails
			}{
				{"unknown with later nextUpdate is not cached", unknownUpdate, originalCached, originalCached},
				{"good with no nextUpdate clears cache", goodNoUpdate, goodNoUpdate, nil},
				{"revoked with no nextUpdate clears cache", revokedNoUpdate, revokedNoUpdate, nil},
				{"good with earlier nextUpdate is not cached", goodEarlierUpdate, originalCached, originalCached},
				{"revoked with earlier nextUpdate is not cached", revokedEarlierUpdate, originalCached, originalCached},
				{"good with later nextUpdate is cached", goodLaterUpdate, goodLaterUpdate, goodLaterUpdate},
				{"revoked with later nextUpdate is cached", revokedLaterUpdate, revokedLaterUpdate, revokedLaterUpdate},
			}
			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					cache := NewCache()
					_ = cache.Update(testRequest, originalCached)

					putReturn := cache.Update(testRequest, tc.response)
					assert.Equal(t, tc.putReturn, putReturn, "expected Put to return %v, got %v", tc.putReturn, putReturn)

					current, ok := cache.cache[testRequestKey]
					if tc.cached == nil {
						assert.False(t, ok, "expected cache to contain no entry, got %v", current)
						return
					}

					assert.True(t, ok, "expected cache to contain %v, got nil", tc.cached)
					assert.Equal(t, tc.cached, current, "expected cache to contain %v, got %v", tc.cached, current)
				})
			}
		})
	})
	t.Run("get", func(t *testing.T) {
		t.Run("empty cache returns nil", func(t *testing.T) {
			cache := NewCache()
			res := cache.Get(testRequest)
			assert.Nil(t, res, "expected Get to return nil, got %v", res)
		})
		t.Run("valid entry returned", func(t *testing.T) {
			cache := NewCache()
			originalResponse := &ResponseDetails{
				Status:     ocsp.Good,
				NextUpdate: futureTime(10),
			}
			cache.cache[testRequestKey] = originalResponse

			res := cache.Get(testRequest)
			assert.Equal(t, originalResponse, res, "expected Get to return %v, got %v", originalResponse, res)
		})
		t.Run("expired entry is deleted", func(t *testing.T) {
			cache := NewCache()
			originalResponse := &ResponseDetails{
				Status:     ocsp.Good,
				NextUpdate: futureTime(-10),
			}
			cache.cache[testRequestKey] = originalResponse

			res := cache.Get(testRequest)
			assert.Nil(t, res, "expected Get to return nil, got %v", res)
			cached, ok := cache.cache[testRequestKey]
			assert.False(t, ok, "expected cache to contain no entry, got %v", cached)
		})
	})
}

func futureTime(minutes int) time.Time {
	return time.Now().Add(time.Duration(minutes) * time.Minute).UTC()
}
