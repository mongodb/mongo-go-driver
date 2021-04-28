// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// +build go1.13

package topology

import (
	"context"
	"errors"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo/address"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/auth"
	"go.mongodb.org/mongo-driver/x/mongo/driver/drivertest"
	"go.mongodb.org/mongo-driver/x/mongo/driver/wiremessage"
)

func makeIsMasterReply() []byte {
	didx, doc := bsoncore.AppendDocumentStart(nil)
	doc = bsoncore.AppendInt32Element(doc, "ok", 1)
	doc, _ = bsoncore.AppendDocumentEnd(doc, didx)
	return drivertest.MakeReply(doc)
}

type channelNetConnDialer struct{}

func (cncd *channelNetConnDialer) DialContext(_ context.Context, _, _ string) (net.Conn, error) {
	cnc := &drivertest.ChannelNetConn{
		Written:  make(chan []byte, 1),
		ReadResp: make(chan []byte, 2),
		ReadErr:  make(chan error, 1),
	}
	if err := cnc.AddResponse(makeIsMasterReply()); err != nil {
		return nil, err
	}

	return cnc, nil
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

	authErr := ConnectionError{Wrapped: &auth.Error{}, init: true}
	netErr := ConnectionError{Wrapped: &net.AddrError{}, init: true}
	for _, tt := range serverTestTable {
		t.Run(tt.name, func(t *testing.T) {
			var returnConnectionError bool
			s, err := NewServer(
				address.Address("localhost"),
				primitive.NewObjectID(),
				WithConnectionOptions(func(connOpts ...ConnectionOption) []ConnectionOption {
					return append(connOpts,
						WithHandshaker(func(Handshaker) Handshaker {
							return &testHandshaker{
								finishHandshake: func(context.Context, driver.Connection) error {
									var err error
									if tt.connectionError && returnConnectionError {
										err = authErr.Wrapped
									}
									return err
								},
							}
						}),
						WithDialer(func(Dialer) Dialer {
							return DialerFunc(func(context.Context, string, string) (net.Conn, error) {
								var err error
								if tt.networkError && returnConnectionError {
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
			err = s.pool.connect()
			require.NoError(t, err, "unable to connect to pool")
			s.connectionstate = connected

			// The internal connection pool resets the generation number once the number of connections in a generation
			// reaches zero, which will cause some of these tests to fail because they assert that the generation
			// number after a connection failure is 1. To workaround this, we call Connection() twice: once to
			// successfully establish a connection and once to actually do the behavior described in the test case.
			_, err = s.Connection(context.Background())
			assert.Nil(t, err, "error getting initial connection: %v", err)
			returnConnectionError = true
			_, err = s.Connection(context.Background())

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

			generation := s.pool.generation.getGeneration(nil)
			if (tt.connectionError || tt.networkError) && generation != 1 {
				t.Errorf("Expected pool to be drained once on connection or network error. got %d; want %d", generation, 1)
			}
		})
	}

	t.Run("multiple connection initialization errors are processed correctly", func(t *testing.T) {
		assertGenerationStats := func(t *testing.T, server *Server, serviceID primitive.ObjectID, generation, numConns uint64) {
			t.Helper()

			stats, ok := server.pool.generation.generationMap[serviceID]
			assert.True(t, ok, "entry for serviceID not found")
			assert.Equal(t, generation, stats.generation, "expected generation number %d, got %d", generation, stats.generation)
			assert.Equal(t, numConns, stats.numConns, "expected connection count %d, got %d", numConns, stats.numConns)
		}

		testCases := []struct {
			name               string
			loadBalanced       bool
			dialErr            error
			getInfoErr         error
			finishHandshakeErr error
			finalGeneration    uint64
			numNewConns        uint64
		}{
			// For LB clusters, errors for dialing and the initial handshake are ignored.
			{"dial errors are ignored for load balancers", true, netErr.Wrapped, nil, nil, 0, 0},
			{"initial handshake errors are ignored for load balancers", true, nil, netErr.Wrapped, nil, 0, 0},
			{"post-handshake errors are not ignored for load balancers", true, nil, nil, netErr.Wrapped, 2, 0},

			// For non-LB clusters, all errors are processed.
			{"dial errors are not ignored for non-lb clusters", false, netErr.Wrapped, nil, nil, 2, 0},
			{"initial handshake errors are not ignored for non-lb clusters", false, nil, netErr.Wrapped, nil, 2, 0},
			{"post-handshake errors are not ignored for non-lb clusters", false, nil, nil, netErr.Wrapped, 2, 0},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var returnConnectionError bool
				var serviceID primitive.ObjectID
				if tc.loadBalanced {
					serviceID = primitive.NewObjectID()
				}

				handshaker := &testHandshaker{
					getHandshakeInformation: func(_ context.Context, addr address.Address, _ driver.Connection) (driver.HandshakeInformation, error) {
						if tc.getInfoErr != nil && returnConnectionError {
							return driver.HandshakeInformation{}, tc.getInfoErr
						}

						desc := description.NewDefaultServer(addr)
						if tc.loadBalanced {
							desc.ServiceID = &serviceID
						}
						return driver.HandshakeInformation{Description: desc}, nil
					},
					finishHandshake: func(context.Context, driver.Connection) error {
						if tc.finishHandshakeErr != nil && returnConnectionError {
							return tc.finishHandshakeErr
						}
						return nil
					},
				}
				connOpts := []ConnectionOption{
					WithDialer(func(Dialer) Dialer {
						return DialerFunc(func(context.Context, string, string) (net.Conn, error) {
							var err error
							if returnConnectionError && tc.dialErr != nil {
								err = tc.dialErr
							}
							return &net.TCPConn{}, err
						})
					}),
					WithHandshaker(func(Handshaker) Handshaker {
						return handshaker
					}),
					WithConnectionLoadBalanced(func(bool) bool {
						return tc.loadBalanced
					}),
				}
				serverOpts := []ServerOption{
					WithServerLoadBalanced(func(bool) bool {
						return tc.loadBalanced
					}),
					WithConnectionOptions(func(...ConnectionOption) []ConnectionOption {
						return connOpts
					}),
					// Disable the monitoring routine because we're only testing pooled connections and we don't want
					// errors in monitoring to clear the pool and make this test flaky.
					withMonitoringDisabled(func(bool) bool {
						return true
					}),
				}

				server, err := ConnectServer(address.Address("localhost:27017"), nil, primitive.NewObjectID(), serverOpts...)
				assert.Nil(t, err, "ConnectServer error: %v", err)

				_, err = server.Connection(context.Background())
				assert.Nil(t, err, "Connection error: %v", err)
				assertGenerationStats(t, server, serviceID, 0, 1)

				returnConnectionError = true
				for i := 0; i < 2; i++ {
					_, err = server.Connection(context.Background())
					switch {
					case tc.dialErr != nil || tc.getInfoErr != nil || tc.finishHandshakeErr != nil:
						assert.NotNil(t, err, "expected Connection error at iteration %d, got nil", i)
					default:
						assert.Nil(t, err, "Connection error at iteration %d: %v", i, err)
					}
				}
				// The final number of connections should be numNewConns+1 to account for the extra one we create above.
				assertGenerationStats(t, server, serviceID, tc.finalGeneration, tc.numNewConns+1)
			})
		}
	})

	t.Run("Cannot starve connection request", func(t *testing.T) {
		cleanup := make(chan struct{})
		addr := bootstrapConnections(t, 3, func(nc net.Conn) {
			<-cleanup
			_ = nc.Close()
		})
		d := newdialer(&net.Dialer{})
		s, err := NewServer(address.Address(addr.String()),
			primitive.NewObjectID(),
			WithConnectionOptions(func(option ...ConnectionOption) []ConnectionOption {
				return []ConnectionOption{WithDialer(func(_ Dialer) Dialer { return d })}
			}),
			WithMaxConnections(func(u uint64) uint64 {
				return 1
			}))
		noerr(t, err)
		s.connectionstate = connected
		err = s.pool.connect()
		noerr(t, err)

		conn, err := s.Connection(context.Background())
		if d.lenopened() != 1 {
			t.Errorf("Should have opened 1 connections, but didn't. got %d; want %d", d.lenopened(), 1)
		}

		var wg sync.WaitGroup

		wg.Add(1)
		ch := make(chan struct{})
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			ch <- struct{}{}
			_, err := s.Connection(ctx)
			if err != nil {
				t.Errorf("Should not be able to starve connection request, but got error: %v", err)
			}
			wg.Done()
		}()
		<-ch
		runtime.Gosched()
		err = conn.Close()
		noerr(t, err)
		wg.Wait()
		close(cleanup)
	})
	t.Run("ProcessError", func(t *testing.T) {
		processID := primitive.NewObjectID()

		// Declare "old" and "new" topology versions and a connection that reports the new version from its
		// Description() method. This connection can be used to test that errors containing a stale topology version
		// do not affect the state of the server.
		oldTV := &description.TopologyVersion{
			ProcessID: processID,
			Counter:   0,
		}
		newTV := &description.TopologyVersion{
			ProcessID: processID,
			Counter:   1,
		}
		oldTVConn := newProcessErrorTestConn(oldTV)
		newTVConn := newProcessErrorTestConn(newTV)

		staleNotMasterError := driver.Error{
			Code:            10107, // NotMaster
			TopologyVersion: oldTV,
		}
		newNotMasterError := driver.Error{
			Code: 10107,
		}
		newShutdownError := driver.Error{
			Code: 11600, // InterruptedAtShutdown
		}
		staleNotMasterWCError := driver.WriteCommandError{
			WriteConcernError: &driver.WriteConcernError{
				Code:            10107,
				TopologyVersion: oldTV,
			},
		}
		newNotMasterWCError := driver.WriteCommandError{
			WriteConcernError: &driver.WriteConcernError{
				Code: 10107,
			},
		}
		newShutdownWCError := driver.WriteCommandError{
			WriteConcernError: &driver.WriteConcernError{
				Code: 11600,
			},
		}
		nonStateChangeError := driver.Error{
			Code: 1,
		}
		networkTimeoutError := driver.Error{
			Labels: []string{driver.NetworkError},
			Wrapped: ConnectionError{
				// Use a net.Error implementation that can return true from its Timeout() function.
				Wrapped: &net.DNSError{
					IsTimeout: true,
				},
			},
		}
		contextCanceledError := driver.Error{
			Labels: []string{driver.NetworkError},
			Wrapped: ConnectionError{
				Wrapped: context.Canceled,
			},
		}
		nonTimeoutNetworkError := driver.Error{
			Labels: []string{driver.NetworkError},
			Wrapped: ConnectionError{
				// Use a net.Error implementation that always returns false from its Timeout() function.
				Wrapped: &net.AddrError{},
			},
		}

		testCases := []struct {
			name   string
			err    error
			conn   driver.Connection
			result driver.ProcessErrorResult
		}{
			// One-off tests for errors that should have no effect.
			{"nil error", nil, oldTVConn, driver.NoChange},
			{"stale connection", errors.New("foo"), newStaleProcessErrorTestConn(), driver.NoChange},
			{"non state change error", nonStateChangeError, oldTVConn, driver.NoChange},

			// Tests for top-level (ok: 0) errors. We test a NotMaster error and a Shutdown error because the former
			// only causes the server to be marked Unknown and the latter causes the pool to be cleared.
			{"stale not master error", staleNotMasterError, newTVConn, driver.NoChange},
			{"new not master error", newNotMasterError, oldTVConn, driver.ServerMarkedUnknown},
			{"new shutdown error", newShutdownError, oldTVConn, driver.ConnectionPoolCleared},

			// Repeat ok:0 tests for write concern errors.
			{"stale not master write concern error", staleNotMasterWCError, newTVConn, driver.NoChange},
			{"new not master write concern error", newNotMasterWCError, oldTVConn, driver.ServerMarkedUnknown},
			{"new shutdown write concern error", newShutdownWCError, oldTVConn, driver.ConnectionPoolCleared},

			// Network/timeout error tests.
			{"network timeout error", networkTimeoutError, oldTVConn, driver.NoChange},
			{"context canceled error", contextCanceledError, oldTVConn, driver.NoChange},
			{"non-timeout network error", nonTimeoutNetworkError, oldTVConn, driver.ConnectionPoolCleared},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				server, err := NewServer(address.Address("localhost"), primitive.NewObjectID())
				assert.Nil(t, err, "NewServer error: %v", err)

				server.connectionstate = connected
				server.pool.connected = connected
				originalDesc := description.Server{
					// The actual Kind value does not matter as long as it's not Unknown so we can detect that it is
					// properly changed to Unknown during the ProcessError call if needed.
					Kind: description.RSPrimary,
				}
				server.desc.Store(originalDesc)

				result := server.ProcessError(tc.err, tc.conn)
				assert.Equal(t, tc.result, result,
					"expected ProcessError result %v, got %v", tc.result, result)

				// Test the ServerChanged() function.
				expectedServerChanged := tc.result != driver.NoChange
				serverChanged := result.ServerChanged()
				assert.Equal(t, expectedServerChanged, serverChanged, "expected ServerChanged() to return %v, got %v",
					expectedServerChanged, serverChanged)

				// Test that the server description fields have been updated to match the ProcessError result.
				expectedKind := originalDesc.Kind
				var expectedError error
				var expectedPoolGeneration uint64
				switch tc.result {
				case driver.ConnectionPoolCleared:
					expectedPoolGeneration = 1
					// This case also implies ServerMarkedUnknown, so any logic in the following case applies as well.
					fallthrough
				case driver.ServerMarkedUnknown:
					expectedKind = description.Unknown
					expectedError = tc.err
				case driver.NoChange:
				default:
					t.Fatalf("unrecognized ProcessErrorResult value %v", tc.result)
				}

				desc := server.Description()
				assert.Equal(t, expectedKind, desc.Kind,
					"expected server kind %q, got %q", expectedKind, desc.Kind)
				assert.Equal(t, expectedError, desc.LastError,
					"expected last error %v, got %v", expectedError, desc.LastError)
				generation := server.pool.generation.getGeneration(nil)
				assert.Equal(t, expectedPoolGeneration, generation,
					"expected pool generation %d, got %d", expectedPoolGeneration, generation)
			})
		}
	})
	t.Run("update topology", func(t *testing.T) {
		var updated atomic.Value // bool
		updated.Store(false)

		updateCallback := func(desc description.Server) description.Server {
			updated.Store(true)
			return desc
		}
		s, err := ConnectServer(address.Address("localhost"), updateCallback, primitive.NewObjectID())
		require.NoError(t, err)
		s.updateDescription(description.Server{Addr: s.address})
		require.True(t, updated.Load().(bool))
	})
	t.Run("heartbeat", func(t *testing.T) {
		// test that client metadata is sent on handshakes but not heartbeats
		dialer := &channelNetConnDialer{}
		dialerOpt := WithDialer(func(Dialer) Dialer {
			return dialer
		})
		serverOpt := WithConnectionOptions(func(connOpts ...ConnectionOption) []ConnectionOption {
			return append(connOpts, dialerOpt)
		})

		s, err := NewServer(address.Address("localhost:27017"), primitive.NewObjectID(), serverOpt)
		if err != nil {
			t.Fatalf("error from NewServer: %v", err)
		}

		// do a heartbeat with a nil connection so a new one will be dialed
		_, err = s.check()
		assert.Nil(t, err, "check error: %v", err)
		assert.NotNil(t, s.conn, "no connection dialed in check")

		channelConn := s.conn.nc.(*drivertest.ChannelNetConn)
		wm := channelConn.GetWrittenMessage()
		if wm == nil {
			t.Fatal("no wire message written for handshake")
		}
		if !includesMetadata(t, wm) {
			t.Fatal("client metadata expected in handshake but not found")
		}

		// do a heartbeat with a non-nil connection
		if err = channelConn.AddResponse(makeIsMasterReply()); err != nil {
			t.Fatalf("error adding response: %v", err)
		}
		_, err = s.check()
		assert.Nil(t, err, "check error: %v", err)

		wm = channelConn.GetWrittenMessage()
		if wm == nil {
			t.Fatal("no wire message written for heartbeat")
		}
		if includesMetadata(t, wm) {
			t.Fatal("client metadata not expected in heartbeat but found")
		}
	})
	t.Run("heartbeat monitoring", func(t *testing.T) {
		var publishedEvents []interface{}

		serverHeartbeatStarted := func(e *event.ServerHeartbeatStartedEvent) {
			publishedEvents = append(publishedEvents, *e)
		}

		serverHeartbeatSucceeded := func(e *event.ServerHeartbeatSucceededEvent) {
			publishedEvents = append(publishedEvents, *e)
		}

		serverHeartbeatFailed := func(e *event.ServerHeartbeatFailedEvent) {
			publishedEvents = append(publishedEvents, *e)
		}

		sdam := &event.ServerMonitor{
			ServerHeartbeatStarted:   serverHeartbeatStarted,
			ServerHeartbeatSucceeded: serverHeartbeatSucceeded,
			ServerHeartbeatFailed:    serverHeartbeatFailed,
		}

		dialer := &channelNetConnDialer{}
		dialerOpt := WithDialer(func(Dialer) Dialer {
			return dialer
		})
		serverOpts := []ServerOption{
			WithConnectionOptions(func(connOpts ...ConnectionOption) []ConnectionOption {
				return append(connOpts, dialerOpt)
			}),
			withMonitoringDisabled(func(bool) bool { return true }),
			WithServerMonitor(func(*event.ServerMonitor) *event.ServerMonitor { return sdam }),
		}

		s, err := NewServer(address.Address("localhost:27017"), primitive.NewObjectID(), serverOpts...)
		if err != nil {
			t.Fatalf("error from NewServer: %v", err)
		}

		// set up heartbeat connection, which doesn't send events
		_, err = s.check()
		assert.Nil(t, err, "check error: %v", err)

		channelConn := s.conn.nc.(*drivertest.ChannelNetConn)
		_ = channelConn.GetWrittenMessage()

		t.Run("success", func(t *testing.T) {
			publishedEvents = nil
			// do a heartbeat with a non-nil connection
			if err = channelConn.AddResponse(makeIsMasterReply()); err != nil {
				t.Fatalf("error adding response: %v", err)
			}
			_, err = s.check()
			_ = channelConn.GetWrittenMessage()
			assert.Nil(t, err, "check error: %v", err)

			assert.Equal(t, len(publishedEvents), 2, "expected %v events, got %v", 2, len(publishedEvents))

			started, ok := publishedEvents[0].(event.ServerHeartbeatStartedEvent)
			assert.True(t, ok, "expected type %T, got %T", event.ServerHeartbeatStartedEvent{}, publishedEvents[0])
			assert.Equal(t, started.ConnectionID, s.conn.ID(), "expected connectionID to match")
			assert.False(t, started.Awaited, "expected awaited to be false")

			succeeded, ok := publishedEvents[1].(event.ServerHeartbeatSucceededEvent)
			assert.True(t, ok, "expected type %T, got %T", event.ServerHeartbeatSucceededEvent{}, publishedEvents[1])
			assert.Equal(t, succeeded.ConnectionID, s.conn.ID(), "expected connectionID to match")
			assert.Equal(t, succeeded.Reply.Addr, s.address, "expected address %v, got %v", s.address, succeeded.Reply.Addr)
			assert.False(t, succeeded.Awaited, "expected awaited to be false")
		})
		t.Run("failure", func(t *testing.T) {
			publishedEvents = nil
			// do a heartbeat with a non-nil connection
			readErr := errors.New("error")
			channelConn.ReadErr <- readErr
			_, err = s.check()
			_ = channelConn.GetWrittenMessage()
			assert.Nil(t, err, "check error: %v", err)

			assert.Equal(t, len(publishedEvents), 2, "expected %v events, got %v", 2, len(publishedEvents))

			started, ok := publishedEvents[0].(event.ServerHeartbeatStartedEvent)
			assert.True(t, ok, "expected type %T, got %T", event.ServerHeartbeatStartedEvent{}, publishedEvents[0])
			assert.Equal(t, started.ConnectionID, s.conn.ID(), "expected connectionID to match")
			assert.False(t, started.Awaited, "expected awaited to be false")

			failed, ok := publishedEvents[1].(event.ServerHeartbeatFailedEvent)
			assert.True(t, ok, "expected type %T, got %T", event.ServerHeartbeatFailedEvent{}, publishedEvents[1])
			assert.Equal(t, failed.ConnectionID, s.conn.ID(), "expected connectionID to match")
			assert.False(t, failed.Awaited, "expected awaited to be false")
			assert.True(t, errors.Is(failed.Failure, readErr), "expected Failure to be %v, got: %v", readErr, failed.Failure)
		})
	})
	t.Run("WithServerAppName", func(t *testing.T) {
		name := "test"

		s, err := NewServer(address.Address("localhost"),
			primitive.NewObjectID(),
			WithServerAppName(func(string) string { return name }))
		require.Nil(t, err, "error from NewServer: %v", err)
		require.Equal(t, name, s.cfg.appname, "expected appname to be: %v, got: %v", name, s.cfg.appname)
	})
	t.Run("createConnection overwrites WithSocketTimeout", func(t *testing.T) {
		socketTimeout := 40 * time.Second

		s, err := NewServer(
			address.Address("localhost"),
			primitive.NewObjectID(),
			WithConnectionOptions(func(connOpts ...ConnectionOption) []ConnectionOption {
				return append(
					connOpts,
					WithReadTimeout(func(time.Duration) time.Duration { return socketTimeout }),
					WithWriteTimeout(func(time.Duration) time.Duration { return socketTimeout }),
				)
			}),
		)
		assert.Nil(t, err, "NewServer error: %v", err)

		conn, err := s.createConnection()
		assert.Nil(t, err, "createConnection error: %v", err)

		assert.Equal(t, s.cfg.heartbeatTimeout, 10*time.Second, "expected heartbeatTimeout to be: %v, got: %v", 10*time.Second, s.cfg.heartbeatTimeout)
		assert.Equal(t, s.cfg.heartbeatTimeout, conn.readTimeout, "expected readTimeout to be: %v, got: %v", s.cfg.heartbeatTimeout, conn.readTimeout)
		assert.Equal(t, s.cfg.heartbeatTimeout, conn.writeTimeout, "expected writeTimeout to be: %v, got: %v", s.cfg.heartbeatTimeout, conn.writeTimeout)
	})
	t.Run("heartbeat contexts are not leaked", func(t *testing.T) {
		// The context created for heartbeats should be cancelled when it is no longer needed to avoid leaks.

		server, err := ConnectServer(
			address.Address("invalid"),
			nil,
			primitive.NewObjectID(),
			withMonitoringDisabled(func(bool) bool {
				return true
			}),
		)
		assert.Nil(t, err, "ConnectServer error: %v", err)

		// Expect check to return an error in the server description because the server address doesn't exist. This is
		// OK because we just want to ensure the heartbeat context is created.
		desc, err := server.check()
		assert.Nil(t, err, "check error: %v", err)
		assert.NotNil(t, desc.LastError, "expected server description to contain an error, got nil")
		assert.NotNil(t, server.heartbeatCtx, "expected heartbeatCtx to be non-nil, got nil")
		assert.Nil(t, server.heartbeatCtx.Err(), "expected heartbeatCtx error to be nil, got %v", server.heartbeatCtx.Err())

		// Override heartbeatCtxCancel with a wrapper that records whether or not it was called.
		oldCancelFn := server.heartbeatCtxCancel
		var previousCtxCancelled bool
		server.heartbeatCtxCancel = func() {
			previousCtxCancelled = true
			oldCancelFn()
		}

		// The second check call should attempt to create a new heartbeat connection and should cancel the previous
		// heartbeatCtx during the process.
		desc, err = server.check()
		assert.Nil(t, err, "check error: %v", err)
		assert.NotNil(t, desc.LastError, "expected server description to contain an error, got nil")
		assert.True(t, previousCtxCancelled, "expected check to cancel previous context but did not")
	})
}

func includesMetadata(t *testing.T, wm []byte) bool {
	var ok bool
	_, _, _, _, wm, ok = wiremessage.ReadHeader(wm)
	if !ok {
		t.Fatal("could not read header")
	}
	_, wm, ok = wiremessage.ReadQueryFlags(wm)
	if !ok {
		t.Fatal("could not read flags")
	}
	_, wm, ok = wiremessage.ReadQueryFullCollectionName(wm)
	if !ok {
		t.Fatal("could not read fullCollectionName")
	}
	_, wm, ok = wiremessage.ReadQueryNumberToSkip(wm)
	if !ok {
		t.Fatal("could not read numberToSkip")
	}
	_, wm, ok = wiremessage.ReadQueryNumberToReturn(wm)
	if !ok {
		t.Fatal("could not read numberToReturn")
	}
	var query bsoncore.Document
	query, wm, ok = wiremessage.ReadQueryQuery(wm)
	if !ok {
		t.Fatal("could not read query")
	}

	if _, err := query.LookupErr("client"); err == nil {
		return true
	}
	if _, err := query.LookupErr("$query", "client"); err == nil {
		return true
	}
	return false
}

// processErrorTestConn is a driver.Connection implementation used by tests
// for Server.ProcessError. This type should not be used for other tests
// because it does not implement all of the functions of the interface.
type processErrorTestConn struct {
	// Embed a driver.Connection to quickly implement the interface without
	// implementing all methods.
	driver.Connection
	stale bool
	tv    *description.TopologyVersion
}

func newProcessErrorTestConn(tv *description.TopologyVersion) *processErrorTestConn {
	return &processErrorTestConn{
		tv: tv,
	}
}

func newStaleProcessErrorTestConn() *processErrorTestConn {
	return &processErrorTestConn{
		stale: true,
	}
}

func (p *processErrorTestConn) Stale() bool {
	return p.stale
}

func (p *processErrorTestConn) Description() description.Server {
	return description.Server{
		WireVersion: &description.VersionRange{
			Max: SupportedWireVersions.Max,
		},
		TopologyVersion: p.tv,
	}
}
