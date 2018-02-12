// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package conn

import (
	"context"

	"github.com/mongodb/mongo-go-driver/mongo/model"
	"github.com/mongodb/mongo-go-driver/mongo/private/msg"
)

// Tracked creates a tracked connection.
func Tracked(c Connection) *TrackedConnection {
	return &TrackedConnection{
		c:     c,
		usage: 1,
	}
}

// TrackedConnection is a connection that only closes
// once it's usage count is 0.
type TrackedConnection struct {
	c     Connection
	usage int
}

// Alive indicates if the connection is still alive.
func (tc *TrackedConnection) Alive() bool {
	return tc.c.Alive()
}

// Close closes the connection.
func (tc *TrackedConnection) Close() error {
	tc.usage--
	if tc.usage <= 0 {
		return tc.c.Close()
	}

	return nil
}

// Model gets a description of the connection.
func (tc *TrackedConnection) Model() *model.Conn {
	return tc.c.Model()
}

// Expired indicates if the connection has expired.
func (tc *TrackedConnection) Expired() bool {
	return tc.c.Expired()
}

// Inc increments the usage count.
func (tc *TrackedConnection) Inc() {
	tc.usage++
}

// Read reads a message from the connection.
func (tc *TrackedConnection) Read(ctx context.Context, responseTo int32) (msg.Response, error) {
	return tc.c.Read(ctx, responseTo)
}

// Write writes a number of messages to the connection.
func (tc *TrackedConnection) Write(ctx context.Context, reqs ...msg.Request) error {
	return tc.c.Write(ctx, reqs...)
}
