// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/internal"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo/address"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
)

type testHandshaker struct {
	getHandshakeInformation func(context.Context, address.Address, driver.Connection) (driver.HandshakeInformation, error)
	finishHandshake         func(context.Context, driver.Connection) error
}

// GetHandshakeInformation implements the Handshaker interface.
func (th *testHandshaker) GetHandshakeInformation(ctx context.Context, addr address.Address, conn driver.Connection) (driver.HandshakeInformation, error) {
	if th.getHandshakeInformation != nil {
		return th.getHandshakeInformation(ctx, addr, conn)
	}
	return driver.HandshakeInformation{}, nil
}

// FinishHandshake implements the Handshaker interface.
func (th *testHandshaker) FinishHandshake(ctx context.Context, conn driver.Connection) error {
	if th.finishHandshake != nil {
		return th.finishHandshake(ctx, conn)
	}
	return nil
}

var _ driver.Handshaker = &testHandshaker{}

func TestConnection(t *testing.T) {
	t.Run("connection", func(t *testing.T) {
		t.Run("newConnection", func(t *testing.T) {
			t.Run("config error", func(t *testing.T) {
				want := errors.New("config error")
				_, got := newConnection(address.Address(""), ConnectionOption(func(*connectionConfig) error { return want }))
				if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
					t.Errorf("errors do not match. got %v; want %v", got, want)
				}
			})
			t.Run("no default idle timeout", func(t *testing.T) {
				conn, err := newConnection(address.Address(""))
				assert.Nil(t, err, "newConnection error: %v", err)
				wantTimeout := time.Duration(0)
				assert.Equal(t, wantTimeout, conn.idleTimeout, "expected idle timeout %v, got %v", wantTimeout,
					conn.idleTimeout)
			})
		})
		t.Run("connect", func(t *testing.T) {
			t.Run("dialer error", func(t *testing.T) {
				err := errors.New("dialer error")
				var want error = ConnectionError{Wrapped: err, init: true}
				conn, got := newConnection(address.Address(""), WithDialer(func(Dialer) Dialer {
					return DialerFunc(func(context.Context, string, string) (net.Conn, error) { return nil, err })
				}))
				if got != nil {
					t.Errorf("newConnection shouldn't error. got %v; want nil", got)
				}
				conn.connect(context.Background())
				got = conn.wait()
				if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
					t.Errorf("errors do not match. got %v; want %v", got, want)
				}
				connState := atomic.LoadInt32(&conn.connected)
				assert.Equal(t, disconnected, connState, "expected connection state %v, got %v", disconnected, connState)
			})
			t.Run("handshaker error", func(t *testing.T) {
				err := errors.New("handshaker error")
				var want error = ConnectionError{Wrapped: err, init: true}
				conn, got := newConnection(address.Address(""),
					WithHandshaker(func(Handshaker) Handshaker {
						return &testHandshaker{
							finishHandshake: func(context.Context, driver.Connection) error {
								return err
							},
						}
					}),
					WithDialer(func(Dialer) Dialer {
						return DialerFunc(func(context.Context, string, string) (net.Conn, error) {
							return &net.TCPConn{}, nil
						})
					}),
				)
				if got != nil {
					t.Errorf("newConnection shouldn't error. got %v; want nil", got)
				}
				conn.connect(context.Background())
				got = conn.wait()
				if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
					t.Errorf("errors do not match. got %v; want %v", got, want)
				}
				connState := atomic.LoadInt32(&conn.connected)
				assert.Equal(t, disconnected, connState, "expected connection state %v, got %v", disconnected, connState)
			})
			t.Run("calls error callback", func(t *testing.T) {
				handshakerError := errors.New("handshaker error")
				var got error

				conn, err := newConnection(address.Address(""),
					WithHandshaker(func(Handshaker) Handshaker {
						return &testHandshaker{
							getHandshakeInformation: func(context.Context, address.Address, driver.Connection) (driver.HandshakeInformation, error) {
								return driver.HandshakeInformation{}, handshakerError
							},
						}
					}),
					WithDialer(func(Dialer) Dialer {
						return DialerFunc(func(context.Context, string, string) (net.Conn, error) {
							return &net.TCPConn{}, nil
						})
					}),
					withErrorHandlingCallback(func(err error, _ uint64, _ *primitive.ObjectID) {
						got = err
					}),
				)
				noerr(t, err)
				conn.connect(context.Background())

				var want error = ConnectionError{Wrapped: handshakerError, init: true}
				err = conn.wait()
				assert.NotNil(t, err, "expected connect error %v, got nil", want)
				assert.Equal(t, want, got, "expected error %v, got %v", want, got)
			})
			t.Run("context is not pinned by connect", func(t *testing.T) {
				// connect creates a cancel-able version of the context passed to it and stores the CancelFunc on the
				// connection. The CancelFunc must be set to nil once the connection has been established so the driver
				// does not pin the memory associated with the context for the connection's lifetime.

				t.Run("connect succeeds", func(t *testing.T) {
					// In the case where connect finishes successfully, it unpins the CancelFunc.

					conn, err := newConnection(address.Address(""),
						WithDialer(func(Dialer) Dialer {
							return DialerFunc(func(context.Context, string, string) (net.Conn, error) {
								return &net.TCPConn{}, nil
							})
						}),
						WithHandshaker(func(Handshaker) Handshaker {
							return &testHandshaker{}
						}),
					)
					assert.Nil(t, err, "newConnection error: %v", err)

					conn.connect(context.Background())
					err = conn.wait()
					assert.Nil(t, err, "error establishing connection: %v", err)
					assert.Nil(t, conn.cancelConnectContext, "cancellation function was not cleared")
				})
				t.Run("connect cancelled", func(t *testing.T) {
					// In the case where connection establishment is cancelled, the closeConnectContext function
					// unpins the CancelFunc.

					// Create a connection that will block in connect until doneChan is closed. This prevents
					// connect from succeeding and unpinning the CancelFunc.
					doneChan := make(chan struct{})
					conn, err := newConnection(address.Address(""),
						WithDialer(func(Dialer) Dialer {
							return DialerFunc(func(context.Context, string, string) (net.Conn, error) {
								<-doneChan
								return &net.TCPConn{}, nil
							})
						}),
						WithHandshaker(func(Handshaker) Handshaker {
							return &testHandshaker{}
						}),
					)
					assert.Nil(t, err, "newConnection error: %v", err)

					// Call connect in a goroutine because it will block.
					var wg sync.WaitGroup
					wg.Add(1)
					go func() {
						defer wg.Done()
						conn.connect(context.Background())
					}()

					// Simulate cancelling connection establishment and assert that this cleares the CancelFunc.
					conn.closeConnectContext()
					assert.Nil(t, conn.cancelConnectContext, "cancellation function was not cleared")
					close(doneChan)
					wg.Wait()
				})
			})
			t.Run("tls", func(t *testing.T) {
				t.Run("connection source is set to default if unspecified", func(t *testing.T) {
					conn, err := newConnection(address.Address(""))
					assert.Nil(t, err, "newConnection error: %v", err)
					assert.NotNil(t, conn.config.tlsConnectionSource, "expected tlsConnectionSource to be set but was not")
				})
				t.Run("server name", func(t *testing.T) {
					testCases := []struct {
						name               string
						addr               address.Address
						cfg                *tls.Config
						expectedServerName string
					}{
						{"set to connection address if empty", "localhost:27017", &tls.Config{}, "localhost"},
						{"left alone if non-empty", "localhost:27017", &tls.Config{ServerName: "other"}, "other"},
					}
					for _, tc := range testCases {
						t.Run(tc.name, func(t *testing.T) {
							var sentCfg *tls.Config
							var testTLSConnectionSource tlsConnectionSourceFn = func(nc net.Conn, cfg *tls.Config) tlsConn {
								sentCfg = cfg
								return tls.Client(nc, cfg)
							}

							connOpts := []ConnectionOption{
								WithDialer(func(Dialer) Dialer {
									return DialerFunc(func(context.Context, string, string) (net.Conn, error) {
										return &net.TCPConn{}, nil
									})
								}),
								WithHandshaker(func(Handshaker) Handshaker {
									return &testHandshaker{}
								}),
								WithTLSConfig(func(*tls.Config) *tls.Config {
									return tc.cfg
								}),
								withTLSConnectionSource(func(tlsConnectionSource) tlsConnectionSource {
									return testTLSConnectionSource
								}),
							}
							conn, err := newConnection(tc.addr, connOpts...)
							assert.Nil(t, err, "newConnection error: %v", err)

							conn.connect(context.Background())
							err = conn.wait()
							assert.NotNil(t, sentCfg, "expected TLS config to be set, but was not")
							assert.Equal(t, tc.expectedServerName, sentCfg.ServerName, "expected ServerName %s, got %s",
								tc.expectedServerName, sentCfg.ServerName)
						})
					}
				})
			})
			t.Run("connectTimeout is applied correctly", func(t *testing.T) {
				testCases := []struct {
					name           string
					contextTimeout time.Duration
					connectTimeout time.Duration
					maxConnectTime time.Duration
				}{
					// The timeout to dial a connection should be min(context timeout, connectTimeoutMS), so 1ms for
					// both of the tests declared below. Both tests also specify a 10ms max connect time to provide
					// a large buffer for lag and avoid test flakiness.

					{"context timeout is lower", 1 * time.Millisecond, 100 * time.Millisecond, 10 * time.Millisecond},
					{"connect timeout is lower", 100 * time.Millisecond, 1 * time.Millisecond, 10 * time.Millisecond},
				}

				for _, tc := range testCases {
					t.Run("timeout applied to socket establishment: "+tc.name, func(t *testing.T) {
						// Ensure the initial connection dial can be timed out and the connection propagates the error
						// from the dialer in this case.

						connOpts := []ConnectionOption{
							WithDialer(func(Dialer) Dialer {
								return DialerFunc(func(ctx context.Context, _, _ string) (net.Conn, error) {
									<-ctx.Done()
									return nil, ctx.Err()
								})
							}),
							WithConnectTimeout(func(time.Duration) time.Duration {
								return tc.connectTimeout
							}),
						}
						conn, err := newConnection("", connOpts...)
						assert.Nil(t, err, "newConnection error: %v", err)

						ctx, cancel := context.WithTimeout(context.Background(), tc.contextTimeout)
						defer cancel()
						var connectErr error
						callback := func() {
							conn.connect(ctx)
							connectErr = conn.wait()
						}
						assert.Soon(t, callback, tc.maxConnectTime)

						ce, ok := connectErr.(ConnectionError)
						assert.True(t, ok, "expected error %v to be of type %T", connectErr, ConnectionError{})
						assert.Equal(t, context.DeadlineExceeded, ce.Unwrap(), "expected wrapped error to be %v, got %v",
							context.DeadlineExceeded, ce.Unwrap())
					})
					t.Run("timeout applied to TLS handshake: "+tc.name, func(t *testing.T) {
						// Ensure the TLS handshake can be timed out and the connection propagates the error from the
						// tlsConn in this case.

						var hangingTLSConnectionSource tlsConnectionSourceFn = func(nc net.Conn, cfg *tls.Config) tlsConn {
							tlsConn := tls.Client(nc, cfg)
							return newHangingTLSConn(tlsConn, tc.maxConnectTime)
						}

						connOpts := []ConnectionOption{
							WithConnectTimeout(func(time.Duration) time.Duration {
								return tc.connectTimeout
							}),
							WithDialer(func(Dialer) Dialer {
								return DialerFunc(func(context.Context, string, string) (net.Conn, error) {
									return &net.TCPConn{}, nil
								})
							}),
							WithTLSConfig(func(*tls.Config) *tls.Config {
								return &tls.Config{}
							}),
							withTLSConnectionSource(func(tlsConnectionSource) tlsConnectionSource {
								return hangingTLSConnectionSource
							}),
						}
						conn, err := newConnection("", connOpts...)
						assert.Nil(t, err, "newConnection error: %v", err)

						ctx, cancel := context.WithTimeout(context.Background(), tc.contextTimeout)
						defer cancel()
						var connectErr error
						callback := func() {
							conn.connect(ctx)
							connectErr = conn.wait()
						}
						assert.Soon(t, callback, tc.maxConnectTime)

						ce, ok := connectErr.(ConnectionError)
						assert.True(t, ok, "expected error %v to be of type %T", connectErr, ConnectionError{})
						assert.Equal(t, context.DeadlineExceeded, ce.Unwrap(), "expected wrapped error to be %v, got %v",
							context.DeadlineExceeded, ce.Unwrap())
					})
					t.Run("timeout is not applied to handshaker: "+tc.name, func(t *testing.T) {
						// Ensure that no additional timeout is applied to the handshake after the connection has been
						// established.

						var getInfoCtx, finishCtx context.Context
						handshaker := &testHandshaker{
							getHandshakeInformation: func(ctx context.Context, _ address.Address, _ driver.Connection) (driver.HandshakeInformation, error) {
								getInfoCtx = ctx
								return driver.HandshakeInformation{}, nil
							},
							finishHandshake: func(ctx context.Context, _ driver.Connection) error {
								finishCtx = ctx
								return nil
							},
						}

						connOpts := []ConnectionOption{
							WithConnectTimeout(func(time.Duration) time.Duration {
								return tc.connectTimeout
							}),
							WithDialer(func(Dialer) Dialer {
								return DialerFunc(func(context.Context, string, string) (net.Conn, error) {
									return &net.TCPConn{}, nil
								})
							}),
							WithHandshaker(func(Handshaker) Handshaker {
								return handshaker
							}),
						}
						conn, err := newConnection("", connOpts...)
						assert.Nil(t, err, "newConnection error: %v", err)

						bgCtx := context.Background()
						conn.connect(bgCtx)
						err = conn.wait()
						assert.Nil(t, err, "connect error: %v", err)

						assertNoContextTimeout := func(t *testing.T, ctx context.Context) {
							t.Helper()
							dl, ok := ctx.Deadline()
							assert.False(t, ok, "expected context to have no deadline, but got deadline %v", dl)
						}
						assertNoContextTimeout(t, getInfoCtx)
						assertNoContextTimeout(t, finishCtx)
					})
				}
			})
		})
		t.Run("writeWireMessage", func(t *testing.T) {
			t.Run("closed connection", func(t *testing.T) {
				conn := &connection{id: "foobar"}
				want := ConnectionError{ConnectionID: "foobar", message: "connection is closed"}
				got := conn.writeWireMessage(context.Background(), []byte{})
				if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
					t.Errorf("errors do not match. got %v; want %v", got, want)
				}
			})
			t.Run("completed context", func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				conn := &connection{id: "foobar", nc: &net.TCPConn{}, connected: connected}
				want := ConnectionError{ConnectionID: "foobar", Wrapped: ctx.Err(), message: "failed to write"}
				got := conn.writeWireMessage(ctx, []byte{})
				if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
					t.Errorf("errors do not match. got %v; want %v", got, want)
				}
			})
			t.Run("deadlines", func(t *testing.T) {
				testCases := []struct {
					name        string
					ctxDeadline time.Duration
					timeout     time.Duration
					deadline    time.Time
				}{
					{"no deadline", 0, 0, time.Now().Add(1 * time.Second)},
					{"ctx deadline", 5 * time.Second, 0, time.Now().Add(6 * time.Second)},
					{"timeout", 0, 10 * time.Second, time.Now().Add(11 * time.Second)},
					{"both (ctx wins)", 15 * time.Second, 20 * time.Second, time.Now().Add(16 * time.Second)},
					{"both (timeout wins)", 30 * time.Second, 25 * time.Second, time.Now().Add(26 * time.Second)},
				}

				for _, tc := range testCases {
					t.Run(tc.name, func(t *testing.T) {
						ctx := context.Background()
						if tc.ctxDeadline > 0 {
							var cancel context.CancelFunc
							ctx, cancel = context.WithTimeout(ctx, tc.ctxDeadline)
							defer cancel()
						}
						want := ConnectionError{
							ConnectionID: "foobar",
							Wrapped:      errors.New("set writeDeadline error"),
							message:      "failed to set write deadline",
						}
						tnc := &testNetConn{deadlineerr: errors.New("set writeDeadline error")}
						conn := &connection{id: "foobar", nc: tnc, writeTimeout: tc.timeout, connected: connected}
						got := conn.writeWireMessage(ctx, []byte{})
						if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
							t.Errorf("errors do not match. got %v; want %v", got, want)
						}
						if !tc.deadline.After(tnc.writeDeadline) {
							t.Errorf("write deadline not properly set. got %v; want %v", tnc.writeDeadline, tc.deadline)
						}
					})
				}
			})
			t.Run("Write", func(t *testing.T) {
				writeErrMsg := "unable to write wire message to network"

				t.Run("error", func(t *testing.T) {
					err := errors.New("Write error")
					tnc := &testNetConn{writeerr: err}
					conn := &connection{id: "foobar", nc: tnc, connected: connected}
					listener := newTestCancellationListener(false)
					conn.cancellationListener = listener

					want := ConnectionError{ConnectionID: "foobar", Wrapped: err, message: writeErrMsg}
					got := conn.writeWireMessage(context.Background(), []byte{})
					if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
						t.Errorf("errors do not match. got %v; want %v", got, want)
					}
					if !tnc.closed {
						t.Errorf("failed to closeConnection net.Conn after error writing bytes.")
					}
					listener.assertMethodsCalled(t, 1, 1)
				})
				t.Run("success", func(t *testing.T) {
					tnc := &testNetConn{}
					conn := &connection{id: "foobar", nc: tnc, connected: connected}
					listener := newTestCancellationListener(false)
					conn.cancellationListener = listener

					want := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A}
					err := conn.writeWireMessage(context.Background(), want)
					noerr(t, err)
					got := tnc.buf
					if !cmp.Equal(got, want) {
						t.Errorf("writeWireMessage did not write the proper bytes. got %v; want %v", got, want)
					}
					listener.assertMethodsCalled(t, 1, 1)
				})
				t.Run("cancel in-progress write", func(t *testing.T) {
					// Simulate context cancellation during a network write.

					nc := newCancellationWriteConn(&testNetConn{}, 0)
					conn := &connection{id: "foobar", nc: nc, connected: connected}
					listener := newTestCancellationListener(false)
					conn.cancellationListener = listener

					ctx, cancel := context.WithCancel(context.Background())
					var err error

					var wg sync.WaitGroup
					wg.Add(1)
					go func() {
						defer wg.Done()
						err = conn.writeWireMessage(ctx, []byte("foobar"))
					}()

					<-nc.operationStartedChan
					cancel()
					nc.continueChan <- struct{}{}

					wg.Wait()
					want := ConnectionError{ConnectionID: conn.id, Wrapped: context.Canceled, message: writeErrMsg}
					assert.Equal(t, want, err, "expected error %v, got %v", want, err)
					assert.Equal(t, disconnected, conn.connected, "expected connection state %v, got %v", disconnected,
						conn.connected)
				})
				t.Run("connection is closed if context is cancelled even if network write succeeds", func(t *testing.T) {
					// Test the race condition between Write and the cancellation listener. The socket write will
					// succeed, but we set the abortedForCancellation flag to true to simulate the context being
					// cancelled immediately after the Write finishes.

					tnc := &testNetConn{}
					conn := &connection{id: "foobar", nc: tnc, connected: connected}
					listener := newTestCancellationListener(true)
					conn.cancellationListener = listener

					want := ConnectionError{ConnectionID: conn.id, Wrapped: context.Canceled, message: writeErrMsg}
					err := conn.writeWireMessage(context.Background(), []byte("foobar"))
					assert.Equal(t, want, err, "expected error %v, got %v", want, err)
					assert.Equal(t, conn.connected, disconnected, "expected connection state %v, got %v", disconnected,
						conn.connected)
				})
			})
		})
		t.Run("readWireMessage", func(t *testing.T) {
			t.Run("closed connection", func(t *testing.T) {
				conn := &connection{id: "foobar"}
				want := ConnectionError{ConnectionID: "foobar", message: "connection is closed"}
				_, got := conn.readWireMessage(context.Background(), []byte{})
				if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
					t.Errorf("errors do not match. got %v; want %v", got, want)
				}
			})
			t.Run("completed context", func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				conn := &connection{id: "foobar", nc: &net.TCPConn{}, connected: connected}
				want := ConnectionError{ConnectionID: "foobar", Wrapped: ctx.Err(), message: "failed to read"}
				_, got := conn.readWireMessage(ctx, []byte{})
				if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
					t.Errorf("errors do not match. got %v; want %v", got, want)
				}
			})
			t.Run("deadlines", func(t *testing.T) {
				testCases := []struct {
					name        string
					ctxDeadline time.Duration
					timeout     time.Duration
					deadline    time.Time
				}{
					{"no deadline", 0, 0, time.Now().Add(1 * time.Second)},
					{"ctx deadline", 5 * time.Second, 0, time.Now().Add(6 * time.Second)},
					{"timeout", 0, 10 * time.Second, time.Now().Add(11 * time.Second)},
					{"both (ctx wins)", 15 * time.Second, 20 * time.Second, time.Now().Add(16 * time.Second)},
					{"both (timeout wins)", 30 * time.Second, 25 * time.Second, time.Now().Add(26 * time.Second)},
				}

				for _, tc := range testCases {
					t.Run(tc.name, func(t *testing.T) {
						ctx := context.Background()
						if tc.ctxDeadline > 0 {
							var cancel context.CancelFunc
							ctx, cancel = context.WithTimeout(ctx, tc.ctxDeadline)
							defer cancel()
						}
						want := ConnectionError{
							ConnectionID: "foobar",
							Wrapped:      errors.New("set readDeadline error"),
							message:      "failed to set read deadline",
						}
						tnc := &testNetConn{deadlineerr: errors.New("set readDeadline error")}
						conn := &connection{id: "foobar", nc: tnc, readTimeout: tc.timeout, connected: connected}
						_, got := conn.readWireMessage(ctx, []byte{})
						if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
							t.Errorf("errors do not match. got %v; want %v", got, want)
						}
						if !tc.deadline.After(tnc.readDeadline) {
							t.Errorf("read deadline not properly set. got %v; want %v", tnc.readDeadline, tc.deadline)
						}
					})
				}
			})
			t.Run("Read", func(t *testing.T) {
				t.Run("size read errors", func(t *testing.T) {
					err := errors.New("Read error")
					tnc := &testNetConn{readerr: err}
					conn := &connection{id: "foobar", nc: tnc, connected: connected}
					listener := newTestCancellationListener(false)
					conn.cancellationListener = listener

					want := ConnectionError{ConnectionID: "foobar", Wrapped: err, message: "incomplete read of message header"}
					_, got := conn.readWireMessage(context.Background(), []byte{})
					if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
						t.Errorf("errors do not match. got %v; want %v", got, want)
					}
					if !tnc.closed {
						t.Errorf("failed to closeConnection net.Conn after error writing bytes.")
					}
					listener.assertMethodsCalled(t, 1, 1)
				})
				t.Run("full message read errors", func(t *testing.T) {
					err := errors.New("Read error")
					tnc := &testNetConn{readerr: err, buf: []byte{0x11, 0x00, 0x00, 0x00}}
					conn := &connection{id: "foobar", nc: tnc, connected: connected}
					listener := newTestCancellationListener(false)
					conn.cancellationListener = listener

					want := ConnectionError{ConnectionID: "foobar", Wrapped: err, message: "incomplete read of full message"}
					_, got := conn.readWireMessage(context.Background(), []byte{})
					if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
						t.Errorf("errors do not match. got %v; want %v", got, want)
					}
					if !tnc.closed {
						t.Errorf("failed to closeConnection net.Conn after error writing bytes.")
					}
					listener.assertMethodsCalled(t, 1, 1)
				})
				t.Run("message too large errors", func(t *testing.T) {
					testCases := []struct {
						name   string
						buffer []byte
						desc   description.Server
					}{
						{
							"message too large errors with small max message size",
							[]byte{0x0A, 0x00, 0x00, 0x00}, // defines a message size of 10 in hex with the first four bytes.
							description.Server{MaxMessageSize: 9},
						},
						{
							"message too large errors with default max message size",
							[]byte{0x01, 0x6C, 0xDC, 0x02}, // defines a message size of 48000001 in hex with the first four bytes.
							description.Server{},
						},
					}
					for _, tc := range testCases {
						t.Run(tc.name, func(t *testing.T) {
							err := errors.New("length of read message too large")
							tnc := &testNetConn{buf: make([]byte, len(tc.buffer))}
							copy(tnc.buf, tc.buffer)
							conn := &connection{id: "foobar", nc: tnc, connected: connected, desc: tc.desc}
							listener := newTestCancellationListener(false)
							conn.cancellationListener = listener

							want := ConnectionError{ConnectionID: "foobar", Wrapped: err, message: err.Error()}
							_, got := conn.readWireMessage(context.Background(), nil)
							if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
								t.Errorf("errors do not match. got %v; want %v", got, want)
							}
							listener.assertMethodsCalled(t, 1, 1)
						})
					}
				})
				t.Run("success", func(t *testing.T) {
					want := []byte{0x0A, 0x00, 0x00, 0x00, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A}
					tnc := &testNetConn{buf: make([]byte, len(want))}
					copy(tnc.buf, want)
					conn := &connection{id: "foobar", nc: tnc, connected: connected}
					listener := newTestCancellationListener(false)
					conn.cancellationListener = listener

					got, err := conn.readWireMessage(context.Background(), nil)
					noerr(t, err)
					if !cmp.Equal(got, want) {
						t.Errorf("did not read full wire message. got %v; want %v", got, want)
					}
					listener.assertMethodsCalled(t, 1, 1)
				})
				t.Run("cancel in-progress read", func(t *testing.T) {
					// Simulate context cancellation during a network read. This has two sub-tests to test cancellation
					// when reading the msg size and when reading the rest of the msg.

					testCases := []struct {
						name   string
						skip   int
						errmsg string
					}{
						{"cancel size read", 0, "incomplete read of message header"},
						{"cancel full message read", 1, "incomplete read of full message"},
					}
					for _, tc := range testCases {
						t.Run(tc.name, func(t *testing.T) {
							// In the full message case, the size read needs to succeed and return a non-zero size, so
							// we set readBuf to indicate that the full message will have 10 bytes.
							readBuf := []byte{10, 0, 0, 0}
							nc := newCancellationReadConn(&testNetConn{}, tc.skip, readBuf)

							conn := &connection{id: "foobar", nc: nc, connected: connected}
							listener := newTestCancellationListener(false)
							conn.cancellationListener = listener

							ctx, cancel := context.WithCancel(context.Background())
							var err error

							var wg sync.WaitGroup
							wg.Add(1)
							go func() {
								defer wg.Done()
								_, err = conn.readWireMessage(ctx, nil)
							}()

							<-nc.operationStartedChan
							cancel()
							nc.continueChan <- struct{}{}

							wg.Wait()
							want := ConnectionError{ConnectionID: conn.id, Wrapped: context.Canceled, message: tc.errmsg}
							assert.Equal(t, want, err, "expected error %v, got %v", want, err)
							assert.Equal(t, disconnected, conn.connected, "expected connection state %v, got %v", disconnected,
								conn.connected)
						})
					}
				})
				t.Run("closes connection if context is cancelled even if the socket read succeeds", func(t *testing.T) {
					tnc := &testNetConn{buf: []byte{0x0A, 0x00, 0x00, 0x00, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A}}
					conn := &connection{id: "foobar", nc: tnc, connected: connected}
					listener := newTestCancellationListener(true)
					conn.cancellationListener = listener

					want := ConnectionError{ConnectionID: conn.id, Wrapped: context.Canceled, message: "unable to read server response"}
					_, err := conn.readWireMessage(context.Background(), nil)
					assert.Equal(t, want, err, "expected error %v, got %v", want, err)
					assert.Equal(t, disconnected, conn.connected, "expected connection state %v, got %v", disconnected,
						conn.connected)
				})
			})
		})
		t.Run("close", func(t *testing.T) {
			t.Run("can close a connection that failed handshaking", func(t *testing.T) {
				conn, err := newConnection(address.Address(""),
					WithHandshaker(func(Handshaker) Handshaker {
						return &testHandshaker{
							finishHandshake: func(context.Context, driver.Connection) error {
								return errors.New("handshake err")
							},
						}
					}),
					WithDialer(func(Dialer) Dialer {
						return DialerFunc(func(context.Context, string, string) (net.Conn, error) {
							return &net.TCPConn{}, nil
						})
					}),
				)
				assert.Nil(t, err, "newConnection error: %v", err)

				conn.connect(context.Background())
				err = conn.wait()
				assert.NotNil(t, err, "expected handshake error from wait, got nil")
				connState := atomic.LoadInt32(&conn.connected)
				assert.Equal(t, disconnected, connState, "expected connection state %v, got %v", disconnected, connState)

				err = conn.close()
				assert.Nil(t, err, "close error: %v", err)
			})
		})
		t.Run("cancellation listener callback", func(t *testing.T) {
			t.Run("closes connection", func(t *testing.T) {
				tnc := &testNetConn{}
				conn := &connection{connected: connected, nc: tnc}

				conn.cancellationListenerCallback()
				assert.True(t, conn.connected == disconnected, "expected connection state %v, got %v", disconnected,
					conn.connected)
				assert.True(t, tnc.closed, "expected net.Conn to be closed but was not")
			})
		})
	})
	t.Run("Connection", func(t *testing.T) {
		t.Run("nil connection does not panic", func(t *testing.T) {
			conn := &Connection{}
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Methods on a Connection with a nil *connection should not panic, but panicked with %v", r)
				}
			}()

			var want, got interface{}

			want = ErrConnectionClosed
			got = conn.WriteWireMessage(context.Background(), nil)
			if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
				t.Errorf("errors do not match. got %v; want %v", got, want)
			}
			_, got = conn.ReadWireMessage(context.Background(), nil)
			if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
				t.Errorf("errors do not match. got %v; want %v", got, want)
			}

			want = description.Server{}
			got = conn.Description()
			if !cmp.Equal(got, want) {
				t.Errorf("descriptions do not match. got %v; want %v", got, want)
			}

			want = nil
			got = conn.Close()
			if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
				t.Errorf("errors do not match. got %v; want %v", got, want)
			}

			got = conn.Expire()
			if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
				t.Errorf("errors do not match. got %v; want %v", got, want)
			}

			want = false
			got = conn.Alive()
			if !cmp.Equal(got, want) {
				t.Errorf("Alive does not match. got %v; want %v", got, want)
			}

			want = "<closed>"
			got = conn.ID()
			if !cmp.Equal(got, want) {
				t.Errorf("IDs do not match. got %v; want %v", got, want)
			}

			want = address.Address("0.0.0.0")
			got = conn.Address()
			if !cmp.Equal(got, want) {
				t.Errorf("Addresses do not match. got %v; want %v", got, want)
			}

			want = address.Address("0.0.0.0")
			got = conn.LocalAddress()
			if !cmp.Equal(got, want) {
				t.Errorf("LocalAddresses do not match. got %v; want %v", got, want)
			}
		})

		t.Run("pinning", func(t *testing.T) {
			makeMultipleConnections := func(t *testing.T, numConns int) (*pool, []*Connection) {
				t.Helper()

				addr := address.Address("")
				pool, err := newPool(poolConfig{Address: addr})
				assert.Nil(t, err, "newPool error: %v", err)

				err = pool.sem.Acquire(context.Background(), int64(numConns))
				assert.Nil(t, err, "error acquiring semaphore: %v", err)

				conns := make([]*Connection, 0, numConns)
				for i := 0; i < numConns; i++ {
					conn, err := newConnection(addr)
					assert.Nil(t, err, "newConnection error: %v", err)
					conn.pool = pool
					conns = append(conns, &Connection{connection: conn})
				}
				return pool, conns
			}
			makeOneConnection := func(t *testing.T) (*pool, *Connection) {
				t.Helper()

				pool, conns := makeMultipleConnections(t, 1)
				return pool, conns[0]
			}

			assertPoolPinnedStats := func(t *testing.T, p *pool, cursorConns, txnConns uint64) {
				t.Helper()

				assert.Equal(t, cursorConns, p.pinnedCursorConnections, "expected %d connections to be pinned to cursors, got %d",
					cursorConns, p.pinnedCursorConnections)
				assert.Equal(t, txnConns, p.pinnedTransactionConnections, "expected %d connections to be pinned to transactions, got %d",
					txnConns, p.pinnedTransactionConnections)
			}

			t.Run("cursors", func(t *testing.T) {
				pool, conn := makeOneConnection(t)
				err := conn.PinToCursor()
				assert.Nil(t, err, "PinToCursor error: %v", err)
				assertPoolPinnedStats(t, pool, 1, 0)

				err = conn.UnpinFromCursor()
				assert.Nil(t, err, "UnpinFromCursor error: %v", err)

				err = conn.Close()
				assert.Nil(t, err, "Close error: %v", err)
				assertPoolPinnedStats(t, pool, 0, 0)
			})
			t.Run("transactions", func(t *testing.T) {
				pool, conn := makeOneConnection(t)
				err := conn.PinToTransaction()
				assert.Nil(t, err, "PinToTransaction error: %v", err)
				assertPoolPinnedStats(t, pool, 0, 1)

				err = conn.UnpinFromTransaction()
				assert.Nil(t, err, "UnpinFromTransaction error: %v", err)

				err = conn.Close()
				assert.Nil(t, err, "Close error: %v", err)
				assertPoolPinnedStats(t, pool, 0, 0)
			})
			t.Run("pool is only updated for first reference", func(t *testing.T) {
				pool, conn := makeOneConnection(t)
				err := conn.PinToTransaction()
				assert.Nil(t, err, "PinToTransaction error: %v", err)
				assertPoolPinnedStats(t, pool, 0, 1)

				err = conn.PinToCursor()
				assert.Nil(t, err, "PinToCursor error: %v", err)
				assertPoolPinnedStats(t, pool, 0, 1)

				err = conn.UnpinFromCursor()
				assert.Nil(t, err, "UnpinFromCursor error: %v", err)
				assertPoolPinnedStats(t, pool, 0, 1)

				err = conn.UnpinFromTransaction()
				assert.Nil(t, err, "UnpinFromTransaction error: %v", err)
				assertPoolPinnedStats(t, pool, 0, 1)

				err = conn.Close()
				assert.Nil(t, err, "Close error: %v", err)
				assertPoolPinnedStats(t, pool, 0, 0)
			})
			t.Run("multiple connections from a pool", func(t *testing.T) {
				pool, conns := makeMultipleConnections(t, 2)
				first, second := conns[0], conns[1]

				err := first.PinToTransaction()
				assert.Nil(t, err, "PinToTransaction error: %v", err)
				err = second.PinToCursor()
				assert.Nil(t, err, "PinToCursor error: %v", err)
				assertPoolPinnedStats(t, pool, 1, 1)

				err = first.UnpinFromTransaction()
				assert.Nil(t, err, "UnpinFromTransaction error: %v", err)
				err = first.Close()
				assert.Nil(t, err, "Close error: %v", err)
				assertPoolPinnedStats(t, pool, 1, 0)

				err = second.UnpinFromCursor()
				assert.Nil(t, err, "UnpinFromCursor error: %v", err)
				err = second.Close()
				assert.Nil(t, err, "Close error: %v", err)
				assertPoolPinnedStats(t, pool, 0, 0)
			})
			t.Run("close is ignored if connection is pinned", func(t *testing.T) {
				pool, conn := makeOneConnection(t)
				err := conn.PinToCursor()
				assert.Nil(t, err, "PinToCursor error: %v", err)

				err = conn.Close()
				assert.Nil(t, err, "Close error")
				assert.NotNil(t, conn.connection, "expected connection to be pinned but it was released to the pool")
				assertPoolPinnedStats(t, pool, 1, 0)
			})
			t.Run("expire forcefully returns connection to pool", func(t *testing.T) {
				pool, conn := makeOneConnection(t)
				err := conn.PinToCursor()
				assert.Nil(t, err, "PinToCursor error: %v", err)

				err = conn.Expire()
				assert.Nil(t, err, "Expire error")
				assert.Nil(t, conn.connection, "expected connection to be released to the pool but was not")
				assertPoolPinnedStats(t, pool, 0, 0)
			})
		})
	})
}

// cancellationTestNetConn is a net.Conn implementation that is used to test context.Cancellation during an in-progress
// network read or write. This type has two unbuffered channels: operationStartedChan and continueChan. When Read/Write
// starts, the type will write to operationStartedChan, which will block until the test reads from it. This signals to
// the test that the connection has entered the net.Conn read/write. After that unblocks, the type will then read from
// continueChan, which blocks until the test writes to it. This allows the test to perform operations with the guarantee
// that they will complete before the read/write functions exit. Sample usage:
//
// nc := newCancellationWriteConn(&testNetConn{}, 0)
// conn := &connection{nc}
// go func() { _ = conn.writeWireMessage(ctx, []byte{"hello world"})}()
// <-nc.operationStartedChan
// log.Println("This print will happen inside net.Conn.Write")
// nc.continueChan <- struct{}{}
//
// By default, the read/write methods will error after they can read from continueChan to simulate a connection being
// closed after context cancellation. This type also supports skipping to allow a number of successfull read/write calls
// before one fails.
type cancellationTestNetConn struct {
	net.Conn

	shouldSkip           int
	skipCount            int
	readBuf              []byte
	operationStartedChan chan struct{}
	continueChan         chan struct{}
}

// create a cancellationTestNetConn to test cancelling net.Conn.Write().
// skip specifies the number of writes that should succeed. Successful writes will return len(writeBuffer), nil.
func newCancellationWriteConn(nc net.Conn, skip int) *cancellationTestNetConn {
	return &cancellationTestNetConn{
		Conn:                 nc,
		shouldSkip:           skip,
		operationStartedChan: make(chan struct{}),
		continueChan:         make(chan struct{}),
	}
}

// create a cancellationTestNetConn to test cancelling net.Conn.Read().
// skip specifies the number of reads that should succeed. Successful reads will copy the contents of readBuf into the
// buffer provided to Read and will return len(readBuf), nil.
func newCancellationReadConn(nc net.Conn, skip int, readBuf []byte) *cancellationTestNetConn {
	return &cancellationTestNetConn{
		Conn:                 nc,
		shouldSkip:           skip,
		readBuf:              readBuf,
		operationStartedChan: make(chan struct{}),
		continueChan:         make(chan struct{}),
	}
}

func (c *cancellationTestNetConn) Read(b []byte) (int, error) {
	if c.skipCount < c.shouldSkip {
		c.skipCount++
		copy(b, c.readBuf)
		return len(c.readBuf), nil
	}

	c.operationStartedChan <- struct{}{}
	<-c.continueChan
	return 0, errors.New("cancelled read")
}

func (c *cancellationTestNetConn) Write(b []byte) (n int, err error) {
	if c.skipCount < c.shouldSkip {
		c.skipCount++
		return len(b), nil
	}

	c.operationStartedChan <- struct{}{}
	<-c.continueChan
	return 0, errors.New("cancelled write")
}

type testNetConn struct {
	nc  net.Conn
	buf []byte

	deadlineerr error
	writeerr    error
	readerr     error
	closed      bool

	deadline      time.Time
	readDeadline  time.Time
	writeDeadline time.Time
}

func (tnc *testNetConn) Read(b []byte) (n int, err error) {
	if len(tnc.buf) > 0 {
		n := copy(b, tnc.buf)
		tnc.buf = tnc.buf[n:]
		return n, nil
	}
	if tnc.readerr != nil {
		return 0, tnc.readerr
	}
	if tnc.nc == nil {
		return 0, nil
	}
	return tnc.nc.Read(b)
}

func (tnc *testNetConn) Write(b []byte) (n int, err error) {
	if tnc.writeerr != nil {
		return 0, tnc.writeerr
	}
	if tnc.nc == nil {
		idx := len(tnc.buf)
		tnc.buf = append(tnc.buf, make([]byte, len(b))...)
		copy(tnc.buf[idx:], b)
		return len(b), nil
	}
	return tnc.nc.Write(b)
}

func (tnc *testNetConn) Close() error {
	tnc.closed = true
	if tnc.nc == nil {
		return nil
	}
	return tnc.nc.Close()
}

func (tnc *testNetConn) LocalAddr() net.Addr {
	if tnc.nc == nil {
		return nil
	}
	return tnc.nc.LocalAddr()
}

func (tnc *testNetConn) RemoteAddr() net.Addr {
	if tnc.nc == nil {
		return nil
	}
	return tnc.nc.RemoteAddr()
}

func (tnc *testNetConn) SetDeadline(t time.Time) error {
	tnc.deadline = t
	if tnc.deadlineerr != nil {
		return tnc.deadlineerr
	}
	if tnc.nc == nil {
		return nil
	}
	return tnc.nc.SetDeadline(t)
}

func (tnc *testNetConn) SetReadDeadline(t time.Time) error {
	tnc.readDeadline = t
	if tnc.deadlineerr != nil {
		return tnc.deadlineerr
	}
	if tnc.nc == nil {
		return nil
	}
	return tnc.nc.SetReadDeadline(t)
}

func (tnc *testNetConn) SetWriteDeadline(t time.Time) error {
	tnc.writeDeadline = t
	if tnc.deadlineerr != nil {
		return tnc.deadlineerr
	}
	if tnc.nc == nil {
		return nil
	}
	return tnc.nc.SetWriteDeadline(t)
}

// bootstrapConnection creates a listener that will listen for a single connection
// on the return address. The user provided run function will be called with the accepted
// connection. The user is responsible for closing the connection.
func bootstrapConnections(t *testing.T, num int, run func(net.Conn)) net.Addr {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Errorf("Could not set up a listener: %v", err)
		t.FailNow()
	}
	go func() {
		for i := 0; i < num; i++ {
			c, err := l.Accept()
			if err != nil {
				t.Errorf("Could not accept a connection: %v", err)
			}
			go run(c)
		}
		_ = l.Close()
	}()
	return l.Addr()
}

type netconn struct {
	net.Conn
	closed chan struct{}
	d      *dialer
}

func (nc *netconn) Close() error {
	nc.closed <- struct{}{}
	nc.d.connclosed(nc)
	return nc.Conn.Close()
}

type writeFailConn struct {
	net.Conn
}

func (wfc *writeFailConn) Write([]byte) (int, error) {
	return 0, errors.New("Write error")
}

func (wfc *writeFailConn) SetWriteDeadline(t time.Time) error {
	return nil
}

type dialer struct {
	Dialer
	opened        map[*netconn]struct{}
	closed        map[*netconn]struct{}
	closeCallBack func()
	sync.Mutex
}

func newdialer(d Dialer) *dialer {
	return &dialer{Dialer: d, opened: make(map[*netconn]struct{}), closed: make(map[*netconn]struct{})}
}

func (d *dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	d.Lock()
	defer d.Unlock()
	c, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	nc := &netconn{Conn: c, closed: make(chan struct{}, 1), d: d}
	d.opened[nc] = struct{}{}
	return nc, nil
}

func (d *dialer) connclosed(nc *netconn) {
	d.Lock()
	defer d.Unlock()
	d.closed[nc] = struct{}{}
	if d.closeCallBack != nil {
		d.closeCallBack()
	}
}

func (d *dialer) lenopened() int {
	d.Lock()
	defer d.Unlock()
	return len(d.opened)
}

func (d *dialer) lenclosed() int {
	d.Lock()
	defer d.Unlock()
	return len(d.closed)
}

type testCancellationListener struct {
	listener         *internal.CancellationListener
	numListen        int
	numStopListening int
	aborted          bool
}

// This function creates a new testCancellationListener. The aborted parameter specifies the value that should be
// returned by the StopListening method.
func newTestCancellationListener(aborted bool) *testCancellationListener {
	return &testCancellationListener{
		listener: internal.NewCancellationListener(),
		aborted:  aborted,
	}
}

func (t *testCancellationListener) Listen(ctx context.Context, abortFn func()) {
	t.numListen++
	t.listener.Listen(ctx, abortFn)
}

func (t *testCancellationListener) StopListening() bool {
	t.numStopListening++
	t.listener.StopListening()
	return t.aborted
}

func (t *testCancellationListener) assertMethodsCalled(testingT *testing.T, numListen int, numStopListening int) {
	assert.Equal(testingT, numListen, t.numListen, "expected Listen to be called %d times, got %d", numListen, t.numListen)
	assert.Equal(testingT, numStopListening, t.numStopListening, "expected StopListening to be called %d times, got %d",
		numListen, t.numListen)
}

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

func (h *hangingTLSConn) Handshake() error {
	time.Sleep(h.sleepTime)
	return h.Conn.Handshake()
}
