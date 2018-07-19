// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package connection

import (
	"fmt"
	"net"
	"time"

	"github.com/mongodb/mongo-go-driver/core/address"
)

// Listener is a generic mongodb network protocol listener. It can return connections
// that speak the mongodb wire protocol.
//
// Multiple goroutines may invoke methods on a Listener simultaneously.
//
// TODO(GODRIVER-270): Implement this.
type Listener interface {
	// Accept waits for and returns the next Connection to the listener.
	Accept() (Connection, error)

	// Close closes the listener.
	Close() error

	// Addr returns the listener's network address.
	Addr() address.Address
}

// Listen creates a new listener on the provided network and address.
func Listen(network, address string, opts ...Option) (Listener, error) {
	return newListener(network, address, opts...)
}

type listener struct {
	addr address.Address
	nl   net.Listener

	readTimeout  time.Duration
	writeTimeout time.Duration
}

func newListener(network, addr string, opts ...Option) (*listener, error) {
	cfg, err := newConfig(opts...)
	if err != nil {
		return nil, err
	}

	nl, err := net.Listen(network, addr)
	if err != nil {
		return nil, err
	}

	return &listener{
		addr:         address.Address(nl.Addr().String()),
		nl:           nl,
		readTimeout:  cfg.readTimeout,
		writeTimeout: cfg.writeTimeout,
	}, nil
}

func (l *listener) Accept() (Connection, error) {
	nc, err := l.nl.Accept()
	if err != nil {
		return nil, err
	}

	id := fmt.Sprintf("%s[-%d]", l.addr, nextServerConnectionID())
	c := &connection{
		id:             id,
		conn:           nc,
		compressBuf:    make([]byte, 256),
		addr:           l.addr,
		readTimeout:    l.readTimeout,
		writeTimeout:   l.writeTimeout,
		readBuf:        make([]byte, 256),
		uncompressBuf:  make([]byte, 256),
		writeBuf:       make([]byte, 256),
		wireMessageBuf: make([]byte, 256),
	}

	return c, nil
}

func (l *listener) Close() error {
	return l.nl.Close()
}

func (l *listener) Addr() address.Address {
	return l.addr
}
