// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package drivertest

import (
	"bytes"
	"errors"
	"net"
	"time"
)

// ChannelNetConn implements the net.Conn interface by reading and writing wire messages to a channel.
type ChannelNetConn struct {
	WriteErr error
	Written  chan []byte
	ReadResp chan []byte
	ReadErr  chan error
	Buf      *bytes.Buffer
}

// Read reads data from the connection
func (c *ChannelNetConn) Read(b []byte) (int, error) {
	if c.Buf.Len() == 0 {
		select {
		case wm := <-c.ReadResp:
			c.Buf.Write(wm)
		case err := <-c.ReadErr:
			return 0, err
		}
	}
	return c.Buf.Read(b)
}

// Write writes data to the connection.
func (c *ChannelNetConn) Write(b []byte) (int, error) {
	copyBuf := make([]byte, len(b))
	copy(copyBuf, b)

	select {
	case c.Written <- copyBuf:
	default:
		c.WriteErr = errors.New("could not write wm to Written channel")
	}
	return len(b), c.WriteErr
}

// Close closes the connection.
func (c *ChannelNetConn) Close() error {
	return nil
}

// LocalAddr returns the local network address.
func (c *ChannelNetConn) LocalAddr() net.Addr {
	return nil
}

// RemoteAddr returns the remote network address.
func (c *ChannelNetConn) RemoteAddr() net.Addr {
	return nil
}

// SetDeadline sets the read and write deadlines associated with the connection.
func (c *ChannelNetConn) SetDeadline(_ time.Time) error {
	return nil
}

// SetReadDeadline sets the read and write deadlines associated with the connection.
func (c *ChannelNetConn) SetReadDeadline(_ time.Time) error {
	return nil
}

// SetWriteDeadline sets the read and write deadlines associated with the connection.
func (c *ChannelNetConn) SetWriteDeadline(_ time.Time) error {
	return nil
}

// GetWrittenMessage gets the last wire message written to the connection
func (c *ChannelNetConn) GetWrittenMessage() []byte {
	select {
	case wm := <-c.Written:
		return wm
	default:
		return nil
	}
}

// AddResponse adds a response to the connection.
func (c *ChannelNetConn) AddResponse(resp []byte) error {
	select {
	case c.ReadResp <- resp[:4]:
	default:
		return errors.New("could not write length bytes")
	}

	select {
	case c.ReadResp <- resp[4:]:
	default:
		return errors.New("could not write response bytes")
	}

	return nil
}
