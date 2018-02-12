// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package conntest

import (
	"context"
	"fmt"
	"net"

	"github.com/mongodb/mongo-go-driver/mongo/model"
	"github.com/mongodb/mongo-go-driver/mongo/private/msg"
)

// MockConnection is used to mock a connection for testing purposes.
type MockConnection struct {
	Dead      bool
	Sent      []msg.Request
	ResponseQ []*msg.Reply
	ReadErr   error
	WriteErr  error

	SkipResponseToFixup bool
}

// Alive returns whether a MockConnection is alive.
func (c *MockConnection) Alive() bool {
	return !c.Dead
}

// Close closes a MockConnection.
func (c *MockConnection) Close() error {
	c.Dead = true
	return nil
}

// CloseIgnoreError closes a MockConnection and ignores any error that occurs.
func (c *MockConnection) CloseIgnoreError() {
	_ = c.Close()
}

// MarkDead marks a MockConnection as dead.
func (c *MockConnection) MarkDead() {
	c.Dead = true
}

// Model returns the description of a MockConnection.
func (c *MockConnection) Model() *model.Conn {
	return &model.Conn{}
}

// Expired returns whether a MockConnection is expired.
func (c *MockConnection) Expired() bool {
	return c.Dead
}

// LocalAddr returns nil.
func (c *MockConnection) LocalAddr() net.Addr {
	return nil
}

// Read reads a server response from the MockConnection.
func (c *MockConnection) Read(ctx context.Context, responseTo int32) (msg.Response, error) {
	if c.ReadErr != nil {
		err := c.ReadErr
		c.ReadErr = nil
		return nil, err
	}

	if len(c.ResponseQ) == 0 {
		return nil, fmt.Errorf("no response queued")
	}
	resp := c.ResponseQ[0]
	c.ResponseQ = c.ResponseQ[1:]
	return resp, nil
}

// Write writes a wire protocol message to MockConnection.
func (c *MockConnection) Write(ctx context.Context, reqs ...msg.Request) error {
	if c.WriteErr != nil {
		err := c.WriteErr
		c.WriteErr = nil
		return err
	}

	for i, req := range reqs {
		c.Sent = append(c.Sent, req)
		if !c.SkipResponseToFixup && i < len(c.ResponseQ) {
			c.ResponseQ[i].RespTo = req.RequestID()
		}
	}
	return nil
}
