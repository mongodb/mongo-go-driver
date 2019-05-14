// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"context"
	"net"
	"sync/atomic"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/address"
	"go.mongodb.org/mongo-driver/x/mongo/driver/auth"
	"go.mongodb.org/mongo-driver/x/mongo/driver/description"
	connectionlegacy "go.mongodb.org/mongo-driver/x/network/connection"
	"go.mongodb.org/mongo-driver/x/network/result"
)

type testpool struct {
	connectionError bool
	drainCalled     atomic.Value
	networkError    bool
	desc            *description.Server
}

func (p *testpool) Get(ctx context.Context) (connectionlegacy.Connection, *description.Server, error) {
	if p.connectionError {
		return nil, p.desc, &auth.Error{}
	}
	if p.networkError {
		return nil, p.desc, &connectionlegacy.NetworkError{}
	}
	return nil, p.desc, nil
}

func (p *testpool) Connect(ctx context.Context) error {
	return nil
}

func (p *testpool) Disconnect(ctx context.Context) error {
	return nil
}

func (p *testpool) Drain() error {
	p.drainCalled.Store(true)
	return nil
}

func NewTestPool(connectionError bool, networkError bool, desc *description.Server) (connectionlegacy.Pool, error) {
	p := &testpool{
		connectionError: connectionError,
		networkError:    networkError,
		desc:            desc,
	}
	p.drainCalled.Store(false)
	return p, nil
}

func TestServer(t *testing.T) {
	var serverTestTable = []struct {
		name            string
		connectionError bool
		networkError    bool
		hasDesc         bool
	}{
		{"auth_error", true, false, false},
		{"no_error", false, false, false},
		{"network_error_no_desc", false, true, false},
		{"network_error_desc", false, true, true},
	}

	authErr := ConnectionError{Wrapped: &auth.Error{}}
	netErr := ConnectionError{Wrapped: &net.AddrError{}}
	for _, tt := range serverTestTable {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewServer(
				address.Address("localhost"),
				WithConnectionOptions(func(connOpts ...ConnectionOption) []ConnectionOption {
					return append(connOpts,
						WithHandshaker(func(Handshaker) Handshaker {
							return HandshakerFunc(func(context.Context, address.Address, driver.Connection) (description.Server, error) {
								var err error
								if tt.connectionError {
									err = authErr.Wrapped
								}
								return description.Server{}, err
							})
						}),
						WithDialer(func(Dialer) Dialer {
							return DialerFunc(func(context.Context, string, string) (net.Conn, error) {
								var err error
								if tt.networkError {
									err = netErr.Wrapped
								}
								return &net.TCPConn{}, err
							})
						}),
					)
				}),
			)
			require.NoError(t, err)

			var desc *description.Server
			descript := s.Description()
			if tt.hasDesc {
				desc = &descript
				require.Nil(t, desc.LastError)
			}
			s.connectionstate = connected
			s.pool.connected = connected

			_, err = s.ConnectionLegacy(context.Background())

			switch {
			case tt.connectionError && !cmp.Equal(err, authErr, cmp.Comparer(compareErrors)):
				t.Errorf("Expected connection error. got %v; want %v", err, authErr)
			case tt.networkError && !cmp.Equal(err, netErr, cmp.Comparer(compareErrors)):
				t.Errorf("Expected network error. got %v; want %v", err, netErr)
			case !tt.connectionError && !tt.networkError && err != nil:
				t.Errorf("Expected error to be nil. got %v; want %v", err, "<nil>")
			}

			if tt.hasDesc {
				require.Equal(t, s.Description().Kind, (description.ServerKind)(description.Unknown))
				require.NotNil(t, s.Description().LastError)
			}

			if (tt.connectionError || tt.networkError) && s.pool.generation != 1 {
				t.Errorf("Expected pool to be drained once on connection or network error. got %d; want %d", s.pool.generation, 1)
			}
		})
	}
	t.Run("WriteConcernError", func(t *testing.T) {
		s, err := NewServer(address.Address("localhost"))
		require.NoError(t, err)

		var desc *description.Server
		descript := s.Description()
		desc = &descript
		require.Nil(t, desc.LastError)
		s.connectionstate = connected
		s.pool.connected = connected

		wce := result.WriteConcernError{10107, "not master", []byte{}}
		require.Equal(t, wceIsNotMasterOrRecovering(&wce), true)
		s.ProcessWriteConcernError(&wce)

		// should set ServerDescription to Unknown
		resultDesc := s.Description()
		require.Equal(t, resultDesc.Kind, (description.ServerKind)(description.Unknown))
		require.Equal(t, resultDesc.LastError, &wce)

		// pool should be drained
		if s.pool.generation != 1 {
			t.Errorf("Expected pool to be drained once from a write concern error. got %d; want %d", s.pool.generation, 1)
		}
	})
	t.Run("no WriteConcernError", func(t *testing.T) {
		s, err := NewServer(address.Address("localhost"))
		require.NoError(t, err)

		var desc *description.Server
		descript := s.Description()
		desc = &descript
		require.Nil(t, desc.LastError)
		s.connectionstate = connected
		s.pool.connected = connected

		wce := result.WriteConcernError{}
		require.Equal(t, wceIsNotMasterOrRecovering(&wce), false)
		s.ProcessWriteConcernError(&wce)

		// should not be a LastError
		require.Nil(t, s.Description().LastError)

		// pool should not be drained
		if s.pool.generation != 0 {
			t.Errorf("Expected pool to not be drained. got %d; want %d", s.pool.generation, 0)
		}
	})
	t.Run("update topology", func(t *testing.T) {
		var updated atomic.Value // bool
		updated.Store(false)
		s, err := ConnectServer(address.Address("localhost"), func(description.Server) { updated.Store(true) })
		require.NoError(t, err)
		s.updateDescription(description.Server{Addr: s.address}, false)
		require.True(t, updated.Load().(bool))
	})
}
