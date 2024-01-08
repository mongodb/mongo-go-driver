// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build go1.13
// +build go1.13

package topology

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"net"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/eventtest"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/mongo/address"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/auth"
	"go.mongodb.org/mongo-driver/x/mongo/driver/drivertest"
	"go.mongodb.org/mongo-driver/x/mongo/driver/wiremessage"
)

type channelNetConnDialer struct{}

func (cncd *channelNetConnDialer) DialContext(_ context.Context, _, _ string) (net.Conn, error) {
	cnc := &drivertest.ChannelNetConn{
		Written:  make(chan []byte, 1),
		ReadResp: make(chan []byte, 2),
		ReadErr:  make(chan error, 1),
	}
	if err := cnc.AddResponse(makeHelloReply()); err != nil {
		return nil, err
	}

	return cnc, nil
}

type errorQueue struct {
	errors []error
	mutex  sync.Mutex
}

func (eq *errorQueue) head() error {
	eq.mutex.Lock()
	defer eq.mutex.Unlock()
	if len(eq.errors) > 0 {
		return eq.errors[0]
	}
	return nil
}

func (eq *errorQueue) dequeue() bool {
	eq.mutex.Lock()
	defer eq.mutex.Unlock()
	if len(eq.errors) > 0 {
		eq.errors = eq.errors[1:]
		return true
	}
	return false
}

type timeoutConn struct {
	net.Conn
	errors *errorQueue
}

func (c *timeoutConn) Read(b []byte) (int, error) {
	n, err := 0, c.errors.head()
	if err == nil {
		n, err = c.Conn.Read(b)
	}
	return n, err
}

func (c *timeoutConn) Write(b []byte) (int, error) {
	n, err := 0, c.errors.head()
	if err == nil {
		n, err = c.Conn.Write(b)
	}
	return n, err
}

type timeoutDialer struct {
	Dialer
	errors *errorQueue
}

func (d *timeoutDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	c, e := d.Dialer.DialContext(ctx, network, address)

	if caFile := os.Getenv("MONGO_GO_DRIVER_CA_FILE"); len(caFile) > 0 {
		pem, err := ioutil.ReadFile(caFile)
		if err != nil {
			return nil, err
		}

		ca := x509.NewCertPool()
		if !ca.AppendCertsFromPEM(pem) {
			return nil, errors.New("unable to load CA file")
		}

		config := &tls.Config{
			InsecureSkipVerify: true,
			RootCAs:            ca,
		}
		c = tls.Client(c, config)
	}
	return &timeoutConn{c, d.errors}, e
}

// TestServerHeartbeatTimeout tests timeout retry and preemptive canceling.
func TestServerHeartbeatTimeout(t *testing.T) {
	if os.Getenv("DOCKER_RUNNING") != "" {
		t.Skip("Skipping this test in docker.")
	}

	networkTimeoutError := &net.DNSError{
		IsTimeout: true,
	}

	testCases := []struct {
		desc                string
		ioErrors            []error
		expectInterruptions int
	}{
		{
			desc:                "one single timeout should not clear the pool",
			ioErrors:            []error{nil, networkTimeoutError, nil, networkTimeoutError, nil},
			expectInterruptions: 0,
		},
		{
			desc:                "continuous timeouts should clear the pool with interruption",
			ioErrors:            []error{nil, networkTimeoutError, networkTimeoutError, nil},
			expectInterruptions: 1,
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			var wg sync.WaitGroup
			wg.Add(1)

			errors := &errorQueue{errors: tc.ioErrors}
			tpm := eventtest.NewTestPoolMonitor()
			server := NewServer(
				address.Address("localhost:27017"),
				primitive.NewObjectID(),
				WithConnectionPoolMonitor(func(*event.PoolMonitor) *event.PoolMonitor {
					return tpm.PoolMonitor
				}),
				WithConnectionOptions(func(opts ...ConnectionOption) []ConnectionOption {
					return append(opts,
						WithDialer(func(d Dialer) Dialer {
							var dialer net.Dialer
							return &timeoutDialer{&dialer, errors}
						}))
				}),
				WithServerMonitor(func(*event.ServerMonitor) *event.ServerMonitor {
					return &event.ServerMonitor{
						ServerHeartbeatSucceeded: func(e *event.ServerHeartbeatSucceededEvent) {
							if !errors.dequeue() {
								wg.Done()
							}
						},
						ServerHeartbeatFailed: func(e *event.ServerHeartbeatFailedEvent) {
							if !errors.dequeue() {
								wg.Done()
							}
						},
					}
				}),
				WithHeartbeatInterval(func(time.Duration) time.Duration {
					return 200 * time.Millisecond
				}),
			)
			require.NoError(t, server.Connect(nil))
			wg.Wait()
			interruptions := tpm.Interruptions()
			assert.Equal(t, tc.expectInterruptions, interruptions, "expected %d interruption but got %d", tc.expectInterruptions, interruptions)
		})
	}
}

// TestServerConnectionTimeout tests how different timeout errors are handled during connection
// creation and server handshake.
func TestServerConnectionTimeout(t *testing.T) {
	testCases := []struct {
		desc              string
		dialer            func(Dialer) Dialer
		handshaker        func(Handshaker) Handshaker
		operationTimeout  time.Duration
		connectTimeout    time.Duration
		expectErr         bool
		expectPoolCleared bool
	}{
		{
			desc:              "successful connection should not clear the pool",
			expectErr:         false,
			expectPoolCleared: false,
		},
		{
			desc: "timeout error during dialing should clear the pool",
			dialer: func(Dialer) Dialer {
				var d net.Dialer
				return DialerFunc(func(ctx context.Context, network, addr string) (net.Conn, error) {
					// Wait for the passed in context to time out. Expect the error returned by
					// DialContext() to be treated as a timeout caused by reaching connectTimeoutMS.
					<-ctx.Done()
					return d.DialContext(ctx, network, addr)
				})
			},
			operationTimeout:  1 * time.Minute,
			connectTimeout:    100 * time.Millisecond,
			expectErr:         true,
			expectPoolCleared: true,
		},
		{
			desc: "timeout error during dialing with no operation timeout should clear the pool",
			dialer: func(Dialer) Dialer {
				var d net.Dialer
				return DialerFunc(func(ctx context.Context, network, addr string) (net.Conn, error) {
					// Wait for the passed in context to time out. Expect the error returned by
					// DialContext() to be treated as a timeout caused by reaching connectTimeoutMS.
					<-ctx.Done()
					return d.DialContext(ctx, network, addr)
				})
			},
			operationTimeout:  0, // Uses a context.Background() with no timeout.
			connectTimeout:    100 * time.Millisecond,
			expectErr:         true,
			expectPoolCleared: true,
		},
		{
			desc: "dial errors unrelated to context timeouts should clear the pool",
			dialer: func(Dialer) Dialer {
				var d net.Dialer
				return DialerFunc(func(ctx context.Context, _, _ string) (net.Conn, error) {
					// Try to dial an invalid TCP address and expect an error.
					return d.DialContext(ctx, "tcp", "300.0.0.0:nope")
				})
			},
			expectErr:         true,
			expectPoolCleared: true,
		},
		{
			desc: "operation context timeout with unrelated dial errors should clear the pool",
			dialer: func(Dialer) Dialer {
				var d net.Dialer
				return DialerFunc(func(ctx context.Context, _, _ string) (net.Conn, error) {
					// Try to dial an invalid TCP address and expect an error.
					c, err := d.DialContext(ctx, "tcp", "300.0.0.0:nope")
					// Wait for the passed in context to time out. Expect that the context error is
					// ignored because the dial error is not a timeout.
					<-ctx.Done()
					return c, err
				})
			},
			operationTimeout:  1 * time.Millisecond,
			connectTimeout:    100 * time.Millisecond,
			expectErr:         true,
			expectPoolCleared: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			// Create a TCP listener on a random port. The listener will accept connections but not
			// read or write to them.
			l, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			defer func() {
				_ = l.Close()
			}()

			tpm := eventtest.NewTestPoolMonitor()
			server := NewServer(
				address.Address(l.Addr().String()),
				primitive.NewObjectID(),
				WithConnectionPoolMonitor(func(*event.PoolMonitor) *event.PoolMonitor {
					return tpm.PoolMonitor
				}),
				// Replace the default dialer and handshaker with the test dialer and handshaker, if
				// present.
				WithConnectionOptions(func(opts ...ConnectionOption) []ConnectionOption {
					if tc.connectTimeout > 0 {
						opts = append(opts, WithConnectTimeout(func(time.Duration) time.Duration { return tc.connectTimeout }))
					}
					if tc.dialer != nil {
						opts = append(opts, WithDialer(tc.dialer))
					}
					if tc.handshaker != nil {
						opts = append(opts, WithHandshaker(tc.handshaker))
					}
					return opts
				}),
				// Disable monitoring to prevent unrelated failures from the RTT monitor and
				// heartbeats from unexpectedly clearing the connection pool.
				withMonitoringDisabled(func(bool) bool { return true }),
			)
			require.NoError(t, server.Connect(nil))

			// Create a context with the operation timeout if one is specified in the test case.
			ctx := context.Background()
			if tc.operationTimeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tc.operationTimeout)
				defer cancel()
			}
			_, err = server.Connection(ctx)
			if tc.expectErr {
				assert.NotNil(t, err, "expected an error but got nil")
			} else {
				assert.Nil(t, err, "expected no error but got %s", err)
			}

			// If we expect the pool to be cleared, watch for all events until we get a
			// "ConnectionPoolCleared" event or until we hit a 10s time limit.
			if tc.expectPoolCleared {
				assert.Eventually(t,
					tpm.IsPoolCleared,
					10*time.Second,
					100*time.Millisecond,
					"expected pool to be cleared within 10s but was not cleared")
			}

			// Disconnect the server then close the events channel and expect that no more events
			// are sent on the channel.
			_ = server.Disconnect(context.Background())

			// If we don't expect the pool to be cleared, check all events after the server is
			// disconnected and make sure none were "ConnectionPoolCleared".
			if !tc.expectPoolCleared {
				assert.False(t, tpm.IsPoolCleared(), "expected pool to not be cleared but was cleared")
			}
		})
	}
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
			s := NewServer(
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

			var desc *description.Server
			descript := s.Description()
			if tt.hasDesc {
				desc = &descript
				require.Nil(t, desc.LastError)
			}
			err := s.pool.ready()
			require.NoError(t, err, "pool.ready() error")
			defer s.pool.close(context.Background())

			s.state = serverConnected

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

			generation, _ := s.pool.generation.getGeneration(nil)
			if (tt.connectionError || tt.networkError) && generation != 1 {
				t.Errorf("Expected pool to be drained once on connection or network error. got %d; want %d", generation, 1)
			}
		})
	}

	t.Run("multiple connection initialization errors are processed correctly", func(t *testing.T) {
		assertGenerationStats := func(t *testing.T, server *Server, serviceID primitive.ObjectID, wantGeneration, wantNumConns uint64) {
			t.Helper()

			getGeneration := func(serviceIDPtr *primitive.ObjectID) uint64 {
				generation, _ := server.pool.generation.getGeneration(serviceIDPtr)
				return generation
			}

			// On connection failure, the connection is removed and closed after delivering the
			// error to Connection(), so it may still count toward the generation connection count
			// briefly. Wait up to 100ms for the generation connection count to reach the target.
			assert.Eventuallyf(t,
				func() bool {
					generation, _ := server.pool.generation.getGeneration(&serviceID)
					numConns := server.pool.generation.getNumConns(&serviceID)
					return generation == wantGeneration && numConns == wantNumConns
				},
				100*time.Millisecond,
				10*time.Millisecond,
				"expected generation number %v, got %v; expected connection count %v, got %v",
				wantGeneration,
				getGeneration(&serviceID),
				wantNumConns,
				server.pool.generation.getNumConns(&serviceID))
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
			{"dial errors are ignored for load balancers", true, netErr.Wrapped, nil, nil, 0, 1},
			{"initial handshake errors are ignored for load balancers", true, nil, netErr.Wrapped, nil, 0, 1},

			// For LB clusters, post-handshake errors clear the pool, but do not update the server
			// description or pause the pool.
			{"post-handshake errors are not ignored for load balancers", true, nil, nil, netErr.Wrapped, 5, 1},

			// For non-LB clusters, the first error sets the server to Unknown and clears and pauses
			// the pool. All subsequent attempts to check out a connection without updating the
			// server description return an error because the pool is paused.
			{"dial errors are not ignored for non-lb clusters", false, netErr.Wrapped, nil, nil, 1, 1},
			{"initial handshake errors are not ignored for non-lb clusters", false, nil, netErr.Wrapped, nil, 1, 1},
			{"post-handshake errors are not ignored for non-lb clusters", false, nil, nil, netErr.Wrapped, 1, 1},
		}
		for _, tc := range testCases {
			tc := tc // Capture range variable.

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
					// With the default maxConnecting (2), there are multiple goroutines creating
					// connections. Because handshake errors are processed after returning the error
					// to checkOut(), it's possible for extra connection requests to be processed
					// after a handshake error before the connection pool is cleared and paused. Set
					// maxConnecting=1 to minimize the number of extra connection requests processed
					// before the connection pool is cleared and paused.
					WithMaxConnecting(func(uint64) uint64 { return 1 }),
				}

				server, err := ConnectServer(address.Address("localhost:27017"), nil, primitive.NewObjectID(), serverOpts...)
				assert.Nil(t, err, "ConnectServer error: %v", err)
				defer func() {
					_ = server.Disconnect(context.Background())
				}()

				_, err = server.Connection(context.Background())
				assert.Nil(t, err, "Connection error: %v", err)
				assertGenerationStats(t, server, serviceID, 0, 1)

				returnConnectionError = true
				for i := 0; i < 5; i++ {
					_, err = server.Connection(context.Background())
					switch {
					case tc.dialErr != nil || tc.getInfoErr != nil || tc.finishHandshakeErr != nil:
						assert.NotNil(t, err, "expected Connection error at iteration %d, got nil", i)
					default:
						assert.Nil(t, err, "Connection error at iteration %d: %v", i, err)
					}
				}
				assertGenerationStats(t, server, serviceID, tc.finalGeneration, tc.numNewConns)
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
		s := NewServer(address.Address(addr.String()),
			primitive.NewObjectID(),
			WithConnectionOptions(func(option ...ConnectionOption) []ConnectionOption {
				return []ConnectionOption{WithDialer(func(_ Dialer) Dialer { return d })}
			}),
			WithMaxConnections(func(u uint64) uint64 {
				return 1
			}))
		s.state = serverConnected
		err := s.pool.ready()
		noerr(t, err)
		defer s.pool.close(context.Background())

		conn, err := s.Connection(context.Background())
		noerr(t, err)
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

		s := NewServer(address.Address("localhost:27017"), primitive.NewObjectID(), serverOpt)

		// do a heartbeat with a nil connection so a new one will be dialed
		_, err := s.check()
		assert.Nil(t, err, "check error: %v", err)
		assert.NotNil(t, s.conn, "no connection dialed in check")

		channelConn := s.conn.nc.(*drivertest.ChannelNetConn)
		wm := channelConn.GetWrittenMessage()
		if wm == nil {
			t.Fatal("no wire message written for handshake")
		}
		if !includesClientMetadata(t, wm) {
			t.Fatal("client metadata expected in handshake but not found")
		}

		// do a heartbeat with a non-nil connection
		if err = channelConn.AddResponse(makeHelloReply()); err != nil {
			t.Fatalf("error adding response: %v", err)
		}
		_, err = s.check()
		assert.Nil(t, err, "check error: %v", err)

		wm = channelConn.GetWrittenMessage()
		if wm == nil {
			t.Fatal("no wire message written for heartbeat")
		}
		if includesClientMetadata(t, wm) {
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

		s := NewServer(address.Address("localhost:27017"), primitive.NewObjectID(), serverOpts...)

		// set up heartbeat connection, which doesn't send events
		_, err := s.check()
		assert.Nil(t, err, "check error: %v", err)

		channelConn := s.conn.nc.(*drivertest.ChannelNetConn)
		_ = channelConn.GetWrittenMessage()

		t.Run("success", func(t *testing.T) {
			publishedEvents = nil
			// do a heartbeat with a non-nil connection
			if err = channelConn.AddResponse(makeHelloReply()); err != nil {
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

		s := NewServer(address.Address("localhost"),
			primitive.NewObjectID(),
			WithServerAppName(func(string) string { return name }))
		require.Equal(t, name, s.cfg.appname, "expected appname to be: %v, got: %v", name, s.cfg.appname)
	})
	t.Run("createConnection overwrites WithSocketTimeout", func(t *testing.T) {
		socketTimeout := 40 * time.Second

		s := NewServer(
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

		conn := s.createConnection()
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

func TestServer_ProcessError(t *testing.T) {
	t.Parallel()

	processID := primitive.NewObjectID()
	newProcessID := primitive.NewObjectID()

	testCases := []struct {
		name string

		startDescription description.Server // Initial server description at the start of the test.

		inputErr  error             // ProcessError error input.
		inputConn driver.Connection // ProcessError conn input.

		want            driver.ProcessErrorResult // Expected ProcessError return value.
		wantGeneration  uint64                    // Expected resulting connection pool generation.
		wantDescription description.Server        // Expected resulting server description.
	}{
		// Test that a nil error does not change the Server state.
		{
			name: "nil error",
			startDescription: description.Server{
				Kind: description.RSPrimary,
			},
			inputErr:       nil,
			want:           driver.NoChange,
			wantGeneration: 0,
			wantDescription: description.Server{
				Kind: description.RSPrimary,
			},
		},
		// Test that errors that occur on stale connections are ignored.
		{
			name: "stale connection",
			startDescription: description.Server{
				Kind: description.RSPrimary,
			},
			inputErr: errors.New("foo"),
			inputConn: newProcessErrorTestConn(
				&description.VersionRange{
					Max: 17,
				},
				true),
			want:           driver.NoChange,
			wantGeneration: 0,
			wantDescription: description.Server{
				Kind: description.RSPrimary,
			},
		},
		// Test that errors that do not indicate a database state change or connection error are
		// ignored.
		{
			name: "non state change error",
			startDescription: description.Server{
				Kind: description.RSPrimary,
			},
			inputErr: driver.Error{
				Code: 1,
			},
			inputConn:      newProcessErrorTestConn(&description.VersionRange{Max: 17}, false),
			want:           driver.NoChange,
			wantGeneration: 0,
			wantDescription: description.Server{
				Kind: description.RSPrimary,
			},
		},
		// Test that a "not writable primary" error with an old topology version is ignored.
		{
			name:             "stale not writable primary error",
			startDescription: newServerDescription(description.RSPrimary, processID, 1, nil),
			inputErr: driver.Error{
				Code: 10107, // NotWritablePrimary
				TopologyVersion: &description.TopologyVersion{
					ProcessID: processID,
					Counter:   0,
				},
			},
			inputConn:       newProcessErrorTestConn(&description.VersionRange{Max: 17}, false),
			want:            driver.NoChange,
			wantGeneration:  0,
			wantDescription: newServerDescription(description.RSPrimary, processID, 1, nil),
		},
		// Test that a "not writable primary" error with an newer topology version marks the Server
		// as "unknown" and updates its topology version.
		{
			name:             "new not writable primary error",
			startDescription: newServerDescription(description.RSPrimary, processID, 0, nil),
			inputErr: driver.Error{
				Code: 10107, // NotWritablePrimary
				TopologyVersion: &description.TopologyVersion{
					ProcessID: processID,
					Counter:   1,
				},
			},
			inputConn:      newProcessErrorTestConn(&description.VersionRange{Max: 17}, false),
			want:           driver.ServerMarkedUnknown,
			wantGeneration: 0,
			wantDescription: newServerDescription(description.Unknown, processID, 1, driver.Error{
				Code: 10107, // NotWritablePrimary
				TopologyVersion: &description.TopologyVersion{
					ProcessID: processID,
					Counter:   1,
				},
			}),
		},
		// Test that a "not writable primary" error with an different topology process ID marks the Server as
		// "unknown" and updates its topology version.
		{
			name:             "new process ID not writable primary error",
			startDescription: newServerDescription(description.RSPrimary, processID, 0, nil),
			inputErr: driver.Error{
				Code: 10107, // NotWritablePrimary
				TopologyVersion: &description.TopologyVersion{
					ProcessID: newProcessID,
					Counter:   0,
				},
			},
			inputConn:      newProcessErrorTestConn(&description.VersionRange{Max: 17}, false),
			want:           driver.ServerMarkedUnknown,
			wantGeneration: 0,
			wantDescription: newServerDescription(description.Unknown, newProcessID, 0, driver.Error{
				Code: 10107, // NotWritablePrimary
				TopologyVersion: &description.TopologyVersion{
					ProcessID: newProcessID,
					Counter:   0,
				},
			}),
		},
		// Test that a connection with a newer topology version overrides the server topology
		// version and causes an error with the same topology version to be ignored.
		// TODO(GODRIVER-2841): Remove this test case.
		{
			name:             "newer connection topology version",
			startDescription: newServerDescription(description.RSPrimary, processID, 0, nil),
			inputErr: driver.Error{
				Code: 10107, // NotWritablePrimary
				TopologyVersion: &description.TopologyVersion{
					ProcessID: processID,
					Counter:   1,
				},
			},
			inputConn: &processErrorTestConn{
				description: description.Server{
					WireVersion: &description.VersionRange{Max: 17},
					TopologyVersion: &description.TopologyVersion{
						ProcessID: processID,
						Counter:   1,
					},
				},
				stale: false,
			},
			want:            driver.NoChange,
			wantGeneration:  0,
			wantDescription: newServerDescription(description.RSPrimary, processID, 0, nil),
		},
		// Test that a "node is shutting down" error with a newer topology version clears the
		// connection pool, marks the Server as "unknown", and updates its topology version.
		{
			name:             "new shutdown error",
			startDescription: newServerDescription(description.RSPrimary, processID, 0, nil),
			inputErr: driver.Error{
				Code: 11600, // InterruptedAtShutdown
				TopologyVersion: &description.TopologyVersion{
					ProcessID: processID,
					Counter:   1,
				},
			},
			inputConn:      newProcessErrorTestConn(&description.VersionRange{Max: 17}, false),
			want:           driver.ConnectionPoolCleared,
			wantGeneration: 1,
			wantDescription: newServerDescription(description.Unknown, processID, 1, driver.Error{
				Code: 11600, // InterruptedAtShutdown
				TopologyVersion: &description.TopologyVersion{
					ProcessID: processID,
					Counter:   1,
				},
			}),
		},
		// Test that a "not writable primary" error with a stale topology version is ignored.
		{
			name:             "stale not writable primary write concern error",
			startDescription: newServerDescription(description.RSPrimary, processID, 1, nil),
			inputErr: driver.WriteCommandError{
				WriteConcernError: &driver.WriteConcernError{
					Code: 10107, // NotWritablePrimary
					TopologyVersion: &description.TopologyVersion{
						ProcessID: processID,
						Counter:   0,
					},
				},
			},
			inputConn:       newProcessErrorTestConn(&description.VersionRange{Max: 17}, false),
			want:            driver.NoChange,
			wantGeneration:  0,
			wantDescription: newServerDescription(description.RSPrimary, processID, 1, nil),
		},
		// Test that a "not writable primary" error with a newer topology version marks the Server
		// as "unknown" and updates its topology version.
		{
			name:             "new not writable primary write concern error",
			startDescription: newServerDescription(description.RSPrimary, processID, 0, nil),
			inputErr: driver.WriteCommandError{
				WriteConcernError: &driver.WriteConcernError{
					Code: 10107, // NotWritablePrimary
					TopologyVersion: &description.TopologyVersion{
						ProcessID: processID,
						Counter:   1,
					},
				},
			},
			inputConn:      newProcessErrorTestConn(&description.VersionRange{Max: 17}, false),
			want:           driver.ServerMarkedUnknown,
			wantGeneration: 0,
			wantDescription: newServerDescription(description.Unknown, processID, 1, driver.WriteCommandError{
				WriteConcernError: &driver.WriteConcernError{
					Code: 10107, // NotWritablePrimary
					TopologyVersion: &description.TopologyVersion{
						ProcessID: processID,
						Counter:   1,
					},
				},
			}),
		},
		// Test that "node is shutting down" errors that have a newer topology version than the
		// local Server topology version mark the Server as "unknown" and clear the connection pool.
		{
			name:             "new shutdown write concern error",
			startDescription: newServerDescription(description.RSPrimary, processID, 0, nil),
			inputErr: driver.WriteCommandError{
				WriteConcernError: &driver.WriteConcernError{
					Code: 11600, // InterruptedAtShutdown
					TopologyVersion: &description.TopologyVersion{
						ProcessID: processID,
						Counter:   1,
					},
				},
			},
			inputConn:      newProcessErrorTestConn(&description.VersionRange{Max: 17}, false),
			want:           driver.ConnectionPoolCleared,
			wantGeneration: 1,
			wantDescription: newServerDescription(description.Unknown, processID, 1, driver.WriteCommandError{
				WriteConcernError: &driver.WriteConcernError{
					Code: 11600, // InterruptedAtShutdown
					TopologyVersion: &description.TopologyVersion{
						ProcessID: processID,
						Counter:   1,
					},
				},
			}),
		},
		// Test that "node is recovering" or "not writable primary" errors that have a newer
		// topology version than the local Server topology version and appear to be from MongoDB
		// servers before 4.2 mark the Server as "unknown" and clear the connection pool.
		{
			name:             "older than 4.2 write concern error",
			startDescription: newServerDescription(description.RSPrimary, processID, 0, nil),
			inputErr: driver.WriteCommandError{
				WriteConcernError: &driver.WriteConcernError{
					Code: 10107, // NotWritablePrimary
					TopologyVersion: &description.TopologyVersion{
						ProcessID: processID,
						Counter:   1,
					},
				},
			},
			inputConn:      newProcessErrorTestConn(&description.VersionRange{Max: 7}, false),
			want:           driver.ConnectionPoolCleared,
			wantGeneration: 1,
			wantDescription: newServerDescription(description.Unknown, processID, 1, driver.WriteCommandError{
				WriteConcernError: &driver.WriteConcernError{
					Code: 10107, // NotWritablePrimary
					TopologyVersion: &description.TopologyVersion{
						ProcessID: processID,
						Counter:   1,
					},
				},
			}),
		},
		// Test that a network timeout error, such as a DNS lookup timeout error, is ignored.
		{
			name:             "network timeout error",
			startDescription: newServerDescription(description.RSPrimary, processID, 0, nil),
			inputErr: driver.Error{
				Labels: []string{driver.NetworkError},
				Wrapped: ConnectionError{
					// Use a net.Error implementation that can return true from its Timeout() function.
					Wrapped: &net.DNSError{
						IsTimeout: true,
					},
				},
			},
			inputConn:       newProcessErrorTestConn(&description.VersionRange{Max: 17}, false),
			want:            driver.NoChange,
			wantGeneration:  0,
			wantDescription: newServerDescription(description.RSPrimary, processID, 0, nil),
		},
		// Test that a context canceled error is ignored.
		{
			name:             "context canceled error",
			startDescription: newServerDescription(description.RSPrimary, processID, 0, nil),
			inputErr: driver.Error{
				Labels: []string{driver.NetworkError},
				Wrapped: ConnectionError{
					Wrapped: context.Canceled,
				},
			},
			inputConn:       newProcessErrorTestConn(&description.VersionRange{Max: 17}, false),
			want:            driver.NoChange,
			wantGeneration:  0,
			wantDescription: newServerDescription(description.RSPrimary, processID, 0, nil),
		},
		// Test that a non-timeout network error, such as an address lookup error, marks the server
		// as "unknown" and sets its topology version to nil.
		{
			name:             "non-timeout network error",
			startDescription: newServerDescription(description.RSPrimary, processID, 0, nil),
			inputErr: driver.Error{
				Labels: []string{driver.NetworkError},
				Wrapped: ConnectionError{
					// Use a net.Error implementation that always returns false from its Timeout() function.
					Wrapped: &net.AddrError{},
				},
			},
			inputConn:      newProcessErrorTestConn(&description.VersionRange{Max: 17}, false),
			want:           driver.ConnectionPoolCleared,
			wantGeneration: 1,
			wantDescription: description.Server{
				Kind: description.Unknown,
				LastError: driver.Error{
					Labels: []string{driver.NetworkError},
					Wrapped: ConnectionError{
						Wrapped: &net.AddrError{},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable.

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := NewServer(address.Address(""), primitive.NewObjectID())
			server.state = serverConnected
			err := server.pool.ready()
			require.Nil(t, err, "pool.ready() error: %v", err)

			server.desc.Store(tc.startDescription)

			got := server.ProcessError(tc.inputErr, tc.inputConn)
			assert.Equal(t, tc.want, got, "expected and actual ProcessError result are different")

			desc := server.Description()
			assert.Equal(t,
				tc.wantDescription,
				desc,
				"expected and actual server descriptions are different")

			generation, _ := server.pool.generation.getGeneration(nil)
			assert.Equal(t,
				tc.wantGeneration,
				generation,
				"expected and actual pool generation are different")
		})
	}
}

// includesClientMetadata will return true if the wire message includes the
// "client" field.
func includesClientMetadata(t *testing.T, wm []byte) bool {
	t.Helper()

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
	description description.Server
	stale       bool
}

func newProcessErrorTestConn(wireVersion *description.VersionRange, stale bool) *processErrorTestConn {
	return &processErrorTestConn{
		description: description.Server{
			WireVersion: wireVersion,
		},
		stale: stale,
	}
}

func (p *processErrorTestConn) Stale() bool {
	return p.stale
}

func (p *processErrorTestConn) Description() description.Server {
	return p.description
}

// newServerDescription is a convenience function for creating a server description with a specified
// kind, topology version process ID and counter, and last error.
func newServerDescription(
	kind description.ServerKind,
	processID primitive.ObjectID,
	counter int64,
	lastError error,
) description.Server {
	return description.Server{
		Kind: kind,
		TopologyVersion: &description.TopologyVersion{
			ProcessID: processID,
			Counter:   counter,
		},
		LastError: lastError,
	}
}
