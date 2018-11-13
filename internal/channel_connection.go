// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package internal

import (
	"context"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
)

// Implements the connection.Connection interface by reading and writing wire messages
// to a channel
type ChannelConn struct {
	T        *testing.T
	WriteErr error
	Written  chan wiremessage.WireMessage
	ReadResp chan wiremessage.WireMessage
	ReadErr  chan error
}

func (c *ChannelConn) WriteWireMessage(ctx context.Context, wm wiremessage.WireMessage) error {
	select {
	case c.Written <- wm:
	default:
		c.T.Error("could not write wiremessage to written channel")
	}
	return c.WriteErr
}

func (c *ChannelConn) ReadWireMessage(ctx context.Context) (wiremessage.WireMessage, error) {
	var wm wiremessage.WireMessage
	var err error
	select {
	case wm = <-c.ReadResp:
	case err = <-c.ReadErr:
	case <-ctx.Done():
	}
	return wm, err
}

func (c *ChannelConn) Close() error {
	return nil
}

func (c *ChannelConn) Expired() bool {
	return false
}

func (c *ChannelConn) Alive() bool {
	return true
}

func (c *ChannelConn) ID() string {
	return "faked"
}

// Create a OP_REPLY wiremessage from a BSON document
func MakeReply(t *testing.T, doc bsonx.Doc) wiremessage.WireMessage {
	rdr, err := doc.MarshalBSON()
	if err != nil {
		t.Fatalf("Could not create document: %v", err)
	}
	return wiremessage.Reply{
		NumberReturned: 1,
		Documents:      []bson.Raw{rdr},
	}
}
