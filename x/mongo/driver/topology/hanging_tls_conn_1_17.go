// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build go1.17
// +build go1.17

package topology

import (
	"context"
	"crypto/tls"
	"time"
)

// hangingTLSConn is an implementation of tlsConn that wraps the tls.Conn type and overrides the HandshakeContext function to
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

// HandshakeContext implements the tlsConn interface on Go 1.17 and higher.
func (h *hangingTLSConn) HandshakeContext(ctx context.Context) error {
	timer := time.NewTimer(h.sleepTime)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
	}

	return h.Conn.HandshakeContext(ctx)
}
