// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/x/mongo/driverlegacy/auth"
	"go.mongodb.org/mongo-driver/x/mongo/driverlegacy/topology"
	"go.mongodb.org/mongo-driver/x/network/address"
	"go.mongodb.org/mongo-driver/x/network/command"
	"go.mongodb.org/mongo-driver/x/network/connection"
)

func TestTopologyServer(t *testing.T) {
	noerr := func(t *testing.T, err error) {
		if err != nil {
			t.Helper()
			t.Errorf("Unepexted error: %v", err)
			t.FailNow()
		}
	}

	t.Run("After close, should not return new connection", func(t *testing.T) {
		s, err := topology.ConnectServer(context.Background(), address.Address(*host), nil, serveropts(t)...)
		noerr(t, err)
		err = s.Disconnect(context.TODO())
		noerr(t, err)
		_, err = s.Connection(context.Background())
		if err != topology.ErrServerClosed {
			t.Errorf("Expected error from getting a connection from closed server, but got %v", err)
		}
	})
	t.Run("Shouldn't be able to get more than max connections", func(t *testing.T) {
		t.Parallel()

		s, err := topology.ConnectServer(context.Background(), address.Address(*host), nil,
			serveropts(
				t,
				topology.WithMaxConnections(func(uint16) uint16 { return 2 }),
				topology.WithMaxIdleConnections(func(uint16) uint16 { return 2 }),
			)...,
		)
		noerr(t, err)
		c1, err := s.Connection(context.Background())
		noerr(t, err)
		defer c1.Close()
		c2, err := s.Connection(context.Background())
		noerr(t, err)
		defer c2.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		_, err = s.Connection(ctx)
		if !strings.Contains(err.Error(), "deadline exceeded") {
			t.Errorf("Expected timeout while trying to open more than max connections, but got %v", err)
		}
	})
	t.Run("Should drain pool when monitor fails", func(t *testing.T) {
		// TODO(GODRIVER-274): Implement this once there is a more testable Dialer.
		t.Skip()
	})
	t.Run("Should drain pool on network error", func(t *testing.T) {
		// TODO(GODRIVER-274): Implement this once there is a more testable Dialer that can return
		// net.Conns that can return specific errors.
		t.Skip()
		t.Run("Read network error", func(t *testing.T) {})
		t.Run("Write network error", func(t *testing.T) {})
	})
	t.Run("Should not drain pool on timeout error", func(t *testing.T) {
		// TODO(GODRIVER-274): Implement this once there is a more testable Dialer that can return
		// net.Conns that can return specific errors.
		t.Skip()
		t.Run("Read network timeout", func(t *testing.T) {})
		t.Run("Write network timeout", func(t *testing.T) {})
	})
	t.Run("Close should close all subscription channels", func(t *testing.T) {
		s, err := topology.ConnectServer(context.Background(), address.Address(*host), nil, serveropts(t)...)
		noerr(t, err)

		var done1, done2 = make(chan struct{}), make(chan struct{})

		sub1, err := s.Subscribe()
		noerr(t, err)

		go func() {
			for range sub1.C {
			}

			close(done1)
		}()

		sub2, err := s.Subscribe()
		noerr(t, err)

		go func() {
			for range sub2.C {
			}

			close(done2)
		}()

		err = s.Disconnect(context.TODO())
		noerr(t, err)

		select {
		case <-done1:
		case <-time.After(50 * time.Millisecond):
			t.Error("Closing server did not close subscription channel 1")
		}

		select {
		case <-done2:
		case <-time.After(50 * time.Millisecond):
			t.Error("Closing server did not close subscription channel 2")
		}
	})
	t.Run("Subscribe after Close should return an error", func(t *testing.T) {
		s, err := topology.ConnectServer(context.Background(), address.Address(*host), nil, serveropts(t)...)
		noerr(t, err)

		sub, err := s.Subscribe()
		noerr(t, err)
		err = s.Disconnect(context.TODO())
		noerr(t, err)

		for range sub.C {
		}

		_, err = s.Subscribe()
		if err != topology.ErrSubscribeAfterClosed {
			t.Errorf("Did not receive expected error. got %v; want %v", err, topology.ErrSubscribeAfterClosed)
		}
	})
	t.Run("Disconnect", func(t *testing.T) {
		t.Run("cannot disconnect before connecting", func(t *testing.T) {
			s, err := topology.NewServer(address.Address(*host), nil, serveropts(t)...)
			noerr(t, err)

			got := s.Disconnect(context.TODO())
			if got != topology.ErrServerClosed {
				t.Errorf("Expected a server disconnected error. got %v; want %v", got, topology.ErrServerClosed)
			}
		})
		t.Run("cannot disconnect twice", func(t *testing.T) {
			s, err := topology.NewServer(address.Address(*host), nil, serveropts(t)...)
			noerr(t, err)
			err = s.Connect(context.TODO())
			noerr(t, err)

			got := s.Disconnect(context.TODO())
			if got != nil {
				t.Errorf("Expected no server disconnected error. got %v; want <nil>", got)
			}
			got = s.Disconnect(context.TODO())
			if got != topology.ErrServerClosed {
				t.Errorf("Expected a server disconnected error. got %v; want %v", got, topology.ErrServerClosed)
			}
		})
		t.Run("all open sockets should be closed after disconnect", func(t *testing.T) {
			d := newdialer(&net.Dialer{})
			s, err := topology.NewServer(
				address.Address(*host), nil,
				serveropts(
					t,
					topology.WithConnectionOptions(func(opts ...connection.Option) []connection.Option {
						return append(opts, connection.WithDialer(func(connection.Dialer) connection.Dialer { return d }))
					}),
				)...,
			)
			noerr(t, err)
			err = s.Connect(context.TODO())
			noerr(t, err)

			conns := [3]connection.Connection{}
			for idx := range [3]struct{}{} {
				conns[idx], err = s.Connection(context.TODO())
				noerr(t, err)
			}
			for idx := range [2]struct{}{} {
				err = conns[idx].Close()
				noerr(t, err)
			}
			if d.lenopened() < 3 {
				t.Errorf("Should have opened at least 3 connections, but didn't. got %d; want >%d", d.lenopened(), 3)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			err = s.Disconnect(ctx)
			noerr(t, err)
			if d.lenclosed() < 3 {
				t.Errorf("Should have closed at least 3 connections, but didn't. got %d; want >%d", d.lenclosed(), 3)
			}
		})
	})
	t.Run("Connect", func(t *testing.T) {
		t.Run("can reconnect a disconnected server", func(t *testing.T) {
			s, err := topology.NewServer(address.Address(*host), nil, serveropts(t)...)
			noerr(t, err)
			err = s.Connect(context.TODO())
			noerr(t, err)

			err = s.Disconnect(context.TODO())
			noerr(t, err)
			err = s.Connect(context.TODO())
			noerr(t, err)
		})
		t.Run("cannot connect multiple times without disconnect", func(t *testing.T) {
			s, err := topology.NewServer(address.Address(*host), nil, serveropts(t)...)
			noerr(t, err)
			err = s.Connect(context.TODO())
			noerr(t, err)

			err = s.Disconnect(context.TODO())
			noerr(t, err)
			err = s.Connect(context.TODO())
			noerr(t, err)
			err = s.Connect(context.TODO())
			if err != topology.ErrServerConnected {
				t.Errorf("Did not receive expected error. got %v; want %v", err, topology.ErrServerConnected)
			}
		})
		t.Run("can disconnect and reconnect multiple times", func(t *testing.T) {
			s, err := topology.NewServer(address.Address(*host), nil, serveropts(t)...)
			noerr(t, err)
			err = s.Connect(context.TODO())
			noerr(t, err)

			err = s.Disconnect(context.TODO())
			noerr(t, err)
			err = s.Connect(context.TODO())
			noerr(t, err)

			err = s.Disconnect(context.TODO())
			noerr(t, err)
			err = s.Connect(context.TODO())
			noerr(t, err)

			err = s.Disconnect(context.TODO())
			noerr(t, err)
			err = s.Connect(context.TODO())
			noerr(t, err)
		})
	})
}

func serveropts(t *testing.T, opts ...topology.ServerOption) []topology.ServerOption {
	noerr := func(t *testing.T, err error) {
		if err != nil {
			t.Errorf("Unepexted error: %v", err)
			t.FailNow()
		}
	}
	cs := testutil.ConnString(t)
	var connOpts []connection.Option
	if cs.Username != "" || cs.AuthMechanism == auth.GSSAPI {
		cred := &auth.Cred{
			Source:      "admin",
			Username:    cs.Username,
			Password:    cs.Password,
			PasswordSet: cs.PasswordSet,
			Props:       cs.AuthMechanismProperties,
		}

		if cs.AuthSource != "" {
			cred.Source = cs.AuthSource
		} else {
			switch cs.AuthMechanism {
			case auth.GSSAPI, auth.PLAIN:
				cred.Source = "$external"
			default:
				cred.Source = cs.Database
			}
		}

		authenticator, err := auth.CreateAuthenticator(cs.AuthMechanism, cred)
		noerr(t, err)

		connOpts = append(connOpts, connection.WithHandshaker(func(h connection.Handshaker) connection.Handshaker {
			return auth.Handshaker(h, &auth.HandshakeOptions{
				AppName:       cs.AppName,
				Authenticator: authenticator,
			})
		}))
	} else {
		connOpts = append(connOpts, connection.WithHandshaker(func(h connection.Handshaker) connection.Handshaker {
			return &command.Handshake{Client: command.ClientDoc(cs.AppName), Compressors: cs.Compressors}
		}))
	}

	if cs.SSL {
		tlsConfig := connection.NewTLSConfig()

		if cs.SSLCaFileSet {
			err := tlsConfig.AddCACertFromFile(cs.SSLCaFile)
			noerr(t, err)
		}

		if cs.SSLInsecure {
			tlsConfig.SetInsecure(true)
		}

		connOpts = append(connOpts, connection.WithTLSConfig(func(*connection.TLSConfig) *connection.TLSConfig { return tlsConfig }))
	}

	if len(cs.Compressors) > 0 {
		connOpts = append(connOpts, connection.WithCompressors(func(compressors []string) []string {
			return append(compressors, cs.Compressors...)
		}))

		for _, comp := range cs.Compressors {
			if comp == "zlib" {
				connOpts = append(connOpts, connection.WithZlibLevel(func(level *int) *int {
					return &cs.ZlibLevel
				}))
			}
		}
	}

	if len(connOpts) > 0 {
		opts = append(opts, topology.WithConnectionOptions(func(opts ...connection.Option) []connection.Option {
			return append(opts, connOpts...)
		}))
	}
	return opts
}
