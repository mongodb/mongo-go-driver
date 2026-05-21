// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driver_test

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/drivertest"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/operation"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/topology"
)

type TopologyWrapper struct {
	*topology.Topology
	SelectCallCount             atomic.Int32
	RequestImmediateCheckCalled atomic.Bool
	SelectionTimeout            time.Duration
	CheckCh                     chan struct{}
}

func (w *TopologyWrapper) SelectServer(ctx context.Context, ss description.ServerSelector) (driver.Server, error) {
	w.SelectCallCount.Add(1)
	return w.Topology.SelectServer(ctx, ss)
}

// GetServerSelectionTimeout returns SelectionTimeout, overriding the embedded
// topology's value so tests can force quick timeouts.
func (w *TopologyWrapper) GetServerSelectionTimeout() time.Duration {
	return w.SelectionTimeout
}

// RequestImmediateCheck closes CheckCh on the first call (unblocking all
// pending dials) then delegates to the embedded topology.
func (w *TopologyWrapper) RequestImmediateCheck() {
	if w.RequestImmediateCheckCalled.CompareAndSwap(false, true) {
		close(w.CheckCh)
	}
	w.Topology.RequestImmediateCheck()
}

// Build creates a topology whose dialer behaviour is controlled by the returned
// checkCh. Dials fail until checkCh is closed, after which each dial returns a
// valid hello reply and transitions the topology to a Known state.
func Build(t *testing.T) (*topology.Topology, chan struct{}) {
	t.Helper()

	// reply is a mock OP_REPLY hello with a valid wire-version range.
	reply := drivertest.MakeReply(
		bsoncore.NewDocumentBuilder().
			AppendBoolean("isWritablePrimary", true).
			AppendInt32("maxWireVersion", 99).
			AppendInt32("minWireVersion", 0).
			AppendInt32("ok", 1).
			Build(),
	)

	checkCh := make(chan struct{})

	dialerOpt := topology.WithDialer(func(topology.Dialer) topology.Dialer {
		return topology.DialerFunc(func(_ context.Context, _, _ string) (net.Conn, error) {
			select {
			case <-checkCh:
				// Allow the dial to proceed and return a valid hello reply.
			default:
				return nil, errors.New("mock: dials not yet allowed")
			}
			conn := &drivertest.ChannelNetConn{
				Written:  make(chan []byte, 10),
				ReadResp: make(chan []byte, 4),
				ReadErr:  make(chan error, 1),
			}
			_ = conn.AddResponse(reply)
			_ = conn.AddResponse(reply)
			return conn, nil
		})
	})

	handshakerOpt := topology.WithHandshaker(func(topology.Handshaker) topology.Handshaker {
		return operation.NewHello()
	})

	cfg := &topology.Config{
		SeedList:       []string{"localhost:27017"},
		ConnectTimeout: 200 * time.Millisecond,
		ServerOpts: []topology.ServerOption{
			topology.WithConnectionOptions(func(opts ...topology.ConnectionOption) []topology.ConnectionOption {
				return append(opts, dialerOpt, handshakerOpt)
			}),
			topology.WithHeartbeatInterval(func(time.Duration) time.Duration { return 10 * time.Second }),
		},
	}

	topo, err := topology.New(cfg)
	require.NoError(t, err)
	require.NoError(t, topo.Connect())
	t.Cleanup(func() { _ = topo.Disconnect(context.Background()) })

	return topo, checkCh
}

func TestServerSelectionRetry(t *testing.T) {
	cmdErr := errors.New("mock command error")

	// After the first SelectServer enters selectServerFromSubscription and
	// times out after selectionTimeout, operation.Execute is expected to call
	// RequestImmediateCheck, so it transitions the topology to Known, allowing
	// the second SelectServer to return a server.
	topo, checkCh := Build(t)
	wrap := &TopologyWrapper{Topology: topo, SelectionTimeout: time.Second, CheckCh: checkCh}

	op := driver.Operation{
		CommandFn: func(dst []byte, _ description.SelectedServer) ([]byte, error) {
			return dst, cmdErr
		},
		Deployment: wrap,
		Database:   "testdb",
	}

	err := op.Execute(context.Background())

	require.Equal(t, cmdErr, err, "expected command error from second attempt, got %v", err)
	require.Equal(t, int32(2), wrap.SelectCallCount.Load(), "expected SelectServer to be called twice")
	require.True(t, wrap.RequestImmediateCheckCalled.Load(), "expected RequestImmediateCheck to be called")
}
