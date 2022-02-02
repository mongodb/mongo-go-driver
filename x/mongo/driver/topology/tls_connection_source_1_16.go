// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
//
// +build !go1.17

package topology

import (
	"crypto/tls"
	"net"
)

type tlsConn interface {
	net.Conn

	// Only require Handshake on the interface for Go 1.16 and less.
	Handshake() error
	ConnectionState() tls.ConnectionState
}

var _ tlsConn = (*tls.Conn)(nil)

type tlsConnectionSource interface {
	Client(net.Conn, *tls.Config) tlsConn
}

type tlsConnectionSourceFn func(net.Conn, *tls.Config) tlsConn

var _ tlsConnectionSource = (tlsConnectionSourceFn)(nil)

func (t tlsConnectionSourceFn) Client(nc net.Conn, cfg *tls.Config) tlsConn {
	return t(nc, cfg)
}

var defaultTLSConnectionSource tlsConnectionSourceFn = func(nc net.Conn, cfg *tls.Config) tlsConn {
	return tls.Client(nc, cfg)
}

// clientHandshake will start a handshake goroutine on Go 1.16 and less when HandshakeContext
// is not avaiable.
func clientHandshake(_ context.Context, client tlsConn, errChan chan error) {
	go func() {
		errChan <- client.Handshake()
	}()
}
