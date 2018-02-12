// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package server

import (
	"time"

	"github.com/mongodb/mongo-go-driver/mongo/private/conn"
)

func newConfig(opts ...Option) (*config, error) {
	cfg := &config{
		opener:            conn.New,
		heartbeatInterval: 10 * time.Second,
		heartbeatTimeout:  30 * time.Second,
		maxConns:          100,
		maxIdleConns:      100,
	}

	err := cfg.apply(opts...)
	return cfg, err
}

// Option configures a server.
type Option func(*config) error

type config struct {
	connOpts          []conn.Option
	opener            conn.Opener
	heartbeatInterval time.Duration
	heartbeatTimeout  time.Duration
	maxConns          uint16
	maxIdleConns      uint16
}

func (c *config) reconfig(opts ...Option) (*config, error) {
	cfg := &config{
		connOpts:          c.connOpts,
		opener:            c.opener,
		heartbeatInterval: c.heartbeatInterval,
		heartbeatTimeout:  c.heartbeatTimeout,
		maxConns:          c.maxConns,
		maxIdleConns:      c.maxIdleConns,
	}

	err := cfg.apply(opts...)
	return cfg, err
}

func (c *config) apply(opts ...Option) error {
	for _, opt := range opts {
		err := opt(c)
		if err != nil {
			return err
		}
	}

	return nil
}

// WithConnectionOpener configures the opener to use
// to create a new connection.
func WithConnectionOpener(opener conn.Opener) Option {
	return func(c *config) error {
		c.opener = opener
		return nil
	}
}

// WithWrappedConnectionOpener configures a new opener to be used
// which wraps the current opener.
func WithWrappedConnectionOpener(wrapper func(conn.Opener) conn.Opener) Option {
	return func(c *config) error {
		c.opener = wrapper(c.opener)
		return nil
	}
}

// WithConnectionOptions configures server's connections. The options provided
// overwrite all previously configured options.
func WithConnectionOptions(opts ...conn.Option) Option {
	return func(c *config) error {
		c.connOpts = opts
		return nil
	}
}

// WithMoreConnectionOptions configures server's connections with
// additional options. The options provided are appended to any
// current options and may override previously configured options.
func WithMoreConnectionOptions(opts ...conn.Option) Option {
	return func(c *config) error {
		c.connOpts = append(c.connOpts, opts...)
		return nil
	}
}

// WithHeartbeatInterval configures a server's heartbeat interval.
// This option will be ignored when creating a Server with a
// pre-existing monitor.
func WithHeartbeatInterval(interval time.Duration) Option {
	return func(c *config) error {
		c.heartbeatInterval = interval
		return nil
	}
}

// WithHeartbeatTimeout configures how long to wait for a heartbeat
// socket to connect.
func WithHeartbeatTimeout(timeout time.Duration) Option {
	return func(c *config) error {
		c.heartbeatTimeout = timeout
		return nil
	}
}

// WithMaxConnections configures maximum number of connections to
// allow for a given server. If max is 0, then there is no upper
// limit on the number of connections.
func WithMaxConnections(max uint16) Option {
	return func(c *config) error {
		c.maxConns = max
		return nil
	}
}

// WithMaxIdleConnections configures the maximum number of idle connections
// allowed for the server.
func WithMaxIdleConnections(size uint16) Option {
	return func(c *config) error {
		c.maxIdleConns = size
		return nil
	}
}
