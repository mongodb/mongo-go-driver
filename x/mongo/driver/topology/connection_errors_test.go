// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build go1.13
// +build go1.13

package topology

import (
	"context"
	"errors"
	"net"
	"testing"

	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo/address"
	"go.mongodb.org/mongo-driver/x/mongo/driver/auth"
)

func TestConnectionErrors(t *testing.T) {
	t.Run("errors are wrapped", func(t *testing.T) {
		t.Run("dial error", func(t *testing.T) {
			dialError := errors.New("foo")

			conn := newConnection(address.Address(""), WithDialer(func(Dialer) Dialer {
				return DialerFunc(func(context.Context, string, string) (net.Conn, error) { return nil, dialError })
			}))

			err := conn.connect(context.Background())
			assert.True(t, errors.Is(err, dialError), "expected error %v, got %v", dialError, err)
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
			defer conn.close()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			err := conn.connect(ctx)
			assert.True(t, errors.Is(err, context.Canceled), "expected error %v, got %v", context.Canceled, err)
		})
		t.Run("write error", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			conn := &connection{id: "foobar", nc: &net.TCPConn{}, state: connConnected}
			err := conn.writeWireMessage(ctx, []byte{})
			assert.True(t, errors.Is(err, context.Canceled), "expected error %v, got %v", context.Canceled, err)
		})
		t.Run("read error", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			conn := &connection{id: "foobar", nc: &net.TCPConn{}, state: connConnected}
			_, err := conn.readWireMessage(ctx, []byte{})
			assert.True(t, errors.Is(err, context.Canceled), "expected error %v, got %v", context.Canceled, err)
		})
	})
}
