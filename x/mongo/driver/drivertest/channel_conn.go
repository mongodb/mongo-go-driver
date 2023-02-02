// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package drivertest

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/mongo/address"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/wiremessage"
)

// ChannelConn implements the driver.Connection interface by reading and writing wire messages
// to a channel
type ChannelConn struct {
	WriteErr error
	Written  chan []byte
	ReadResp chan []byte
	ReadErr  chan error
	Desc     description.Server
}

// WriteWireMessage implements the driver.Connection interface.
func (c *ChannelConn) WriteWireMessage(ctx context.Context, wm []byte) error {
	// Copy wm in case it came from a buffer pool.
	b := make([]byte, len(wm))
	copy(b, wm)
	select {
	case c.Written <- b:
	default:
		c.WriteErr = errors.New("could not write wiremessage to written channel")
	}
	return c.WriteErr
}

// ReadWireMessage implements the driver.Connection interface.
func (c *ChannelConn) ReadWireMessage(ctx context.Context, dst []byte) ([]byte, error) {
	dst = dst[:0]
	var wm []byte
	var err error
	select {
	case wm = <-c.ReadResp:
	case err = <-c.ReadErr:
	case <-ctx.Done():
	}
	if l := len(wm); l > 0 {
		if l > cap(dst) {
			dst = make([]byte, 0, l)
		}
		dst = append(dst, wm...)
	}
	return dst, err
}

// Description implements the driver.Connection interface.
func (c *ChannelConn) Description() description.Server { return c.Desc }

// Close implements the driver.Connection interface.
func (c *ChannelConn) Close() error {
	return nil
}

// ID implements the driver.Connection interface.
func (c *ChannelConn) ID() string {
	return "faked"
}

// ServerConnectionID implements the driver.Connection interface.
func (c *ChannelConn) ServerConnectionID() *int32 {
	serverConnectionID := int32(42)
	return &serverConnectionID
}

// Address implements the driver.Connection interface.
func (c *ChannelConn) Address() address.Address { return address.Address("0.0.0.0") }

// Stale implements the driver.Connection interface.
func (c *ChannelConn) Stale() bool {
	return false
}

// MakeReply creates an OP_REPLY wiremessage from a BSON document
func MakeReply(doc bsoncore.Document) []byte {
	var dst []byte
	idx, dst := bsoncore.ReserveLength(dst)
	dst = bsoncore.AppendInt32(dst,
		10,                         // reqid
		9,                          // respto
		int32(wiremessage.OpReply), // opcode
		0,                          // reply flag
	)
	dst = bsoncore.AppendInt64(dst, 0) // reply cursor ID
	dst = bsoncore.AppendInt32(dst,
		0, // reply starting from
		1, // reply number returned
	)
	dst = append(dst, doc...)
	return bsoncore.UpdateLength(dst, idx, int32(len(dst[idx:])))
}

// GetCommandFromQueryWireMessage returns the command sent in an OP_QUERY wire message.
func GetCommandFromQueryWireMessage(wm []byte) (bsoncore.Document, error) {
	var ok bool
	_, wm, ok = bsoncore.ReadBytes(wm, 16)
	if !ok {
		return nil, errors.New("could not read header")
	}
	_, wm, ok = bsoncore.ReadInt32(wm)
	if !ok {
		return nil, errors.New("could not read flags")
	}
	_, wm, ok = bsoncore.ReadCString(wm)
	if !ok {
		return nil, errors.New("could not read fullCollectionName")
	}
	_, wm, ok = bsoncore.ReadInt32(wm)
	if !ok {
		return nil, errors.New("could not read numberToSkip")
	}
	_, wm, ok = bsoncore.ReadInt32(wm)
	if !ok {
		return nil, errors.New("could not read numberToReturn")
	}

	var query bsoncore.Document
	query, _, ok = bsoncore.ReadDocument(wm)
	if !ok {
		return nil, errors.New("could not read query")
	}
	return query, nil
}

// GetCommandFromMsgWireMessage returns the command document sent in an OP_MSG wire message.
func GetCommandFromMsgWireMessage(wm []byte) (bsoncore.Document, error) {
	var ok bool
	_, wm, ok = bsoncore.ReadBytes(wm, 16)
	if !ok {
		return nil, errors.New("could not read header")
	}

	_, wm, ok = bsoncore.ReadInt32(wm)
	if !ok {
		return nil, errors.New("could not read flags")
	}
	_, wm, ok = bsoncore.ReadByte(wm)
	if !ok {
		return nil, errors.New("could not read section type")
	}

	cmdDoc, _, ok := bsoncore.ReadDocument(wm)
	if !ok {
		return nil, errors.New("could not read command document")
	}
	return cmdDoc, nil
}
