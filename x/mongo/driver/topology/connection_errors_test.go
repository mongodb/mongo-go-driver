// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build go1.13

package topology

import (
	"context"
	"crypto/x509"
	"errors"
	"net"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo/address"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/auth"
)

func TestConnectionErrors(t *testing.T) {
	type labeledError interface {
		HasErrorLabel(string) bool
	}

	t.Run("errors are wrapped", func(t *testing.T) {
		t.Run("dial error", func(t *testing.T) {
			dialError := errors.New("foo")

			conn := newConnection(address.Address(""), WithDialer(func(Dialer) Dialer {
				return DialerFunc(func(context.Context, string, string) (net.Conn, error) { return nil, dialError })
			}))

			err := conn.connect(context.Background())
			assert.True(t, errors.Is(err, dialError), "expected error %v, got %v", dialError, err)

			le, ok := err.(labeledError)
			require.True(t, ok, "error should implement LabeledError interface")

			assert.True(t, le.HasErrorLabel(driver.ErrSystemOverloadedError), "expected label %s", driver.ErrSystemOverloadedError)
			assert.True(t, le.HasErrorLabel(driver.ErrRetryableError), "expected label %s", driver.ErrRetryableError)
		})
		t.Run("handshake error", func(t *testing.T) {
			conn := newConnection(address.Address(""),
				WithHandshaker(func(Handshaker) Handshaker {
					return auth.Handshaker(nil, &auth.HandshakeOptions{})
				}),
				WithDialer(func(Dialer) Dialer {
					return DialerFunc(func(context.Context, string, string) (net.Conn, error) {
						return &net.TCPConn{}, nil
					})
				}),
			)
			defer func() { require.NoError(t, conn.close()) }()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			err := conn.connect(ctx)
			assert.True(t, errors.Is(err, context.Canceled), "expected error %v, got %v", context.Canceled, err)

			le, ok := err.(labeledError)
			require.True(t, ok, "error should implement LabeledError interface")

			assert.True(t, le.HasErrorLabel(driver.ErrSystemOverloadedError), "expected label %s", driver.ErrSystemOverloadedError)
			assert.True(t, le.HasErrorLabel(driver.ErrRetryableError), "expected label %s", driver.ErrRetryableError)
		})
		t.Run("dns error", func(t *testing.T) {
			dnsError := &net.DNSError{}

			conn := newConnection(address.Address(""), WithDialer(func(Dialer) Dialer {
				return DialerFunc(func(context.Context, string, string) (net.Conn, error) { return nil, dnsError })
			}))

			err := conn.connect(context.Background())
			assert.True(t, errors.Is(err, dnsError), "expected error %v, got %v", dnsError, err)

			le, ok := err.(labeledError)
			require.True(t, ok, "error should implement LabeledError interface")

			assert.False(t, le.HasErrorLabel(driver.ErrSystemOverloadedError), "unexpected label %s", driver.ErrSystemOverloadedError)
			assert.False(t, le.HasErrorLabel(driver.ErrRetryableError), "unexpected label %s", driver.ErrRetryableError)
		})
		t.Run("tls certificate error", func(t *testing.T) {
			certErr := x509.CertificateInvalidError{}

			c := &connection{id: "test"}
			err := c.wrapError(certErr, !isTLSCertError(certErr), "tls failure")

			le, ok := err.(labeledError)
			require.True(t, ok)
			assert.False(t, le.HasErrorLabel(driver.ErrSystemOverloadedError), "unexpected label %s", driver.ErrSystemOverloadedError)
		})
		t.Run("tls hostname error", func(t *testing.T) {
			hostErr := x509.HostnameError{}

			c := &connection{id: "test"}
			err := c.wrapError(hostErr, !isTLSCertError(hostErr), "tls failure")

			le, ok := err.(labeledError)
			require.True(t, ok)
			assert.False(t, le.HasErrorLabel(driver.ErrSystemOverloadedError), "unexpected label %s", driver.ErrSystemOverloadedError)
		})
	})
}
