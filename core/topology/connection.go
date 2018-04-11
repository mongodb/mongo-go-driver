// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"context"
	"errors"
	"net"

	"github.com/mongodb/mongo-go-driver/core/connection"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
)

// sconn is a wrapper around a connection.Connection. This type is returned by
// a Server so that it can track network errors and when a non-timeout network
// error is returned, the pool on the server can be cleared.
type sconn struct {
	connection.Connection
	s *Server
}

func (sc *sconn) ReadWireMessage(ctx context.Context) (wiremessage.WireMessage, error) {
	if sc.Connection == nil {
		return nil, errors.New("already closed")
	}
	wm, err := sc.Connection.ReadWireMessage(ctx)
	sc.processErr(err)
	return wm, err
}

func (sc *sconn) WriteWireMessage(ctx context.Context, wm wiremessage.WireMessage) error {
	if sc.Connection == nil {
		return errors.New("already closed")
	}
	err := sc.Connection.WriteWireMessage(ctx, wm)
	sc.processErr(err)
	return err
}

func (sc *sconn) Close() error {
	if sc.Connection == nil {
		return nil
	}
	err := sc.Connection.Close()
	sc.Connection = nil
	return err
}

func (sc *sconn) processErr(err error) {
	ne, ok := err.(connection.NetworkError)
	if !ok {
		return
	}

	if netErr, ok := ne.Wrapped.(net.Error); ok && netErr.Timeout() {
		return
	}
	if ne.Wrapped == context.Canceled || ne.Wrapped == context.DeadlineExceeded {
		return
	}

	_ = sc.s.Drain()
}
