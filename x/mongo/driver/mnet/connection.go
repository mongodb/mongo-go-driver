// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mnet

import (
	"sync"

	"go.mongodb.org/mongo-driver/internal/driverutil"
	"go.mongodb.org/mongo-driver/mongo/address"
)

// Connection represents a connection to a MongoDB server.
type Connection struct {
	WireMessageReadWriteCloser
	Describer
	Streamer
	WireMessageCompressor
	Pinner

	// conn contains the underlying connection logic for the default
	// implementation of mnet.Connection. conn is thread-unsafe to avoid
	// lock contention, but the default usage should be made thread-safe.
	unsafe *driverutil.UnsafeConnection

	// mu guards logic that wrapps the thread-unsafe conn.
	mu sync.RWMutex
}

// NewConnection will construct a connection object with the default
// implementations.
func NewConnection(addr address.Address, _ *Options) *Connection {
	// Convert opts to UnsafeConnectionOptions.
	unsafeOpts := &driverutil.UnsafeConnectionOptions{}

	conn := &Connection{
		unsafe: driverutil.NewUnsafeConnection(addr, unsafeOpts),
	}

	conn.Describer = &defaultDescriber{}
	conn.WireMessageReadWriteCloser = &defaultIO{}
	conn.Pinner = &defaultPinner{}
	conn.Streamer = &defaultStreamer{}
	conn.WireMessageCompressor = &defaultCompressor{}

	return conn
}

// TODO: Add logic
func (c *Connection) Expire() error                 { return nil }
func (c *Connection) Alive() bool                   { return false }
func (c *Connection) LocalAddress() address.Address { return "" }
func (c *Connection) IsClosed() bool                { return false }
