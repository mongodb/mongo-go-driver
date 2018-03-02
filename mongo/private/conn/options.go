// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package conn

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/mongodb/mongo-go-driver/mongo/private/msg"
)

func newConfig(opts ...Option) (*config, error) {
	cfg := &config{
		codec:          msg.NewWireProtocolCodec(),
		connectTimeout: 30 * time.Second,
		dialer:         dialWithoutTLS,
		idleTimeout:    10 * time.Minute,
		lifeTimeout:    30 * time.Minute,
	}

	for _, opt := range opts {
		err := opt(cfg)
		if err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// Option configures a connection.
type Option func(*config) error

type config struct {
	appName        string
	codec          msg.Codec
	connectTimeout time.Duration
	dialer         Dialer
	idleTimeout    time.Duration
	lifeTimeout    time.Duration
	readTimeout    time.Duration
	writeTimeout   time.Duration
}

// WithAppName sets the application name which gets
// sent to MongoDB on first connection.
func WithAppName(name string) Option {
	return func(c *config) error {
		c.appName = name
		return nil
	}
}

// WithCodec sets the codec to use to encode and
// decode messages.
func WithCodec(codec msg.Codec) Option {
	return func(c *config) error {
		c.codec = codec
		return nil
	}
}

// WithConnectTimeout configures the maximum amount of time
// a dial will wait for a connect to complete. The default
// is 30 seconds.
func WithConnectTimeout(timeout time.Duration) Option {
	return func(c *config) error {
		c.connectTimeout = timeout
		return nil
	}
}

// WithDialer defines the dialer for endpoints.
func WithDialer(dialer Dialer) Option {
	return func(c *config) error {
		c.dialer = dialer
		return nil
	}
}

// WithWrappedDialer wraps the current dialer.
func WithWrappedDialer(wrapper func(dialer Dialer) Dialer) Option {
	return func(c *config) error {
		c.dialer = wrapper(c.dialer)
		return nil
	}
}

// WithIdleTimeout configures the maximum idle time
// to allow for a connection.
func WithIdleTimeout(timeout time.Duration) Option {
	return func(c *config) error {
		c.idleTimeout = timeout
		return nil
	}
}

// WithLifeTimeout configures the maximum life of a
// connection.
func WithLifeTimeout(timeout time.Duration) Option {
	return func(c *config) error {
		c.lifeTimeout = timeout
		return nil
	}
}

// WithReadTimeout configures the maximum read time
// for a connection.
func WithReadTimeout(timeout time.Duration) Option {
	return func(c *config) error {
		c.readTimeout = timeout
		return nil
	}
}

// WithWriteTimeout configures the maximum read time
// for a connection.
func WithWriteTimeout(timeout time.Duration) Option {
	return func(c *config) error {
		c.writeTimeout = timeout
		return nil
	}
}

// WithTLSConfig configures the SSL options for a connection.
func WithTLSConfig(tlsDefault *TLSConfig) Option {
	return func(c *config) error {
		c.dialer = func(ctx context.Context, dialer *net.Dialer, network, address string) (net.Conn, error) {
			conn, err := dialer.DialContext(ctx, network, address)
			if err != nil {
				return nil, err
			}

			tlsConfig := tlsDefault.MakeConfig()

			if !tlsConfig.InsecureSkipVerify {
				colonPos := strings.LastIndex(address, ":")
				if colonPos == -1 {
					colonPos = len(address)
				}

				hostname := address[:colonPos]
				tlsConfig.ServerName = hostname
			}

			client := tls.Client(conn, tlsConfig)

			errChan := make(chan error, 1)
			go func() {
				errChan <- client.Handshake()
			}()

			for {
				select {
				case err := <-errChan:
					if err == nil {
						return client, nil
					}

					return nil, err

				case <-ctx.Done():
					return nil, errors.New("server connection cancelled/timeout during TLS handshake")
				}
			}
		}

		return nil
	}
}
