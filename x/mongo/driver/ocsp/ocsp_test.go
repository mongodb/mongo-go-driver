// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build go1.13
// +build go1.13

package ocsp

import (
	"context"
	"crypto/x509"
	"net"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/internal"
	"go.mongodb.org/mongo-driver/internal/assert"
)

func TestContactResponders(t *testing.T) {
	t.Run("context cancellation is honored", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		serverCert := &x509.Certificate{
			OCSPServer: []string{"https://localhost:5000"},
		}
		cfg := config{
			serverCert: serverCert,
			issuer:     &x509.Certificate{},
			cache:      NewCache(),
			httpClient: internal.DefaultHTTPClient,
		}

		res := contactResponders(ctx, cfg)
		assert.Nil(t, res, "expected nil response details, but got %v", res)
	})
	t.Run("context timeout is honored", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Create a TCP listener on a random port that doesn't accept any connections, causing
		// connection attempts to hang indefinitely from the client's perspective.
		l, err := net.Listen("tcp", "localhost:0")
		assert.Nil(t, err, "tls.Listen() error: %v", err)
		defer l.Close()

		serverCert := &x509.Certificate{
			OCSPServer: []string{"https://" + l.Addr().String()},
		}
		cfg := config{
			serverCert: serverCert,
			issuer:     &x509.Certificate{},
			cache:      NewCache(),
			httpClient: internal.DefaultHTTPClient,
		}

		// Expect that contactResponders() returns a nil response but does not cause any errors when
		// the passed-in context times out.
		start := time.Now()
		res := contactResponders(ctx, cfg)
		duration := time.Since(start)
		assert.Nil(t, res, "expected nil response, but got: %v", res)
		assert.True(t, duration <= 5*time.Second, "expected duration to be <= 5s, but was %v", duration)
	})
}
