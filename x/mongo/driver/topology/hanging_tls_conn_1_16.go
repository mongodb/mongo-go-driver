// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build !go1.17
// +build !go1.17

package topology

import (
	"crypto/tls"
	"time"
)

// hangingTLSConn is an implementation of tlsConn that wraps the tls.Conn type and overrides the Handshake function to
// sleep for a fixed amount of time.
type hangingTLSConn struct {
	*tls.Conn
	sleepTime time.Duration
}

var _ tlsConn = (*hangingTLSConn)(nil)

func newHangingTLSConn(conn *tls.Conn, sleepTime time.Duration) *hangingTLSConn {
	return &hangingTLSConn{
		Conn:      conn,
		sleepTime: sleepTime,
	}
}

// Handshake implements the tlsConn interface on Go 1.16 and less.
func (h *hangingTLSConn) Handshake() error {
	time.Sleep(h.sleepTime)
	return h.Conn.Handshake()
}
