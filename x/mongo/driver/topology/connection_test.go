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
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/address"
	"go.mongodb.org/mongo-driver/x/mongo/driver/description"
)

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
				var want error = ConnectionError{Wrapped: err}
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
				var want error = ConnectionError{Wrapped: err}
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
							getDescription: func(context.Context, address.Address, driver.Connection) (description.Server, error) {
								return description.Server{}, handshakerError
							},
						}
					}),
					WithDialer(func(Dialer) Dialer {
						return DialerFunc(func(context.Context, string, string) (net.Conn, error) {
							return &net.TCPConn{}, nil
						})
					}),
					withErrorHandlingCallback(func(err error, _ uint64) {
						got = err
					}),
				)
				noerr(t, err)
				conn.connect(context.Background())

				var want error = ConnectionError{Wrapped: handshakerError}
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
							var testTLSConnectionSource tlsConnectionSourceFn = func(nc net.Conn, cfg *tls.Config) *tls.Conn {
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
				t.Run("error", func(t *testing.T) {
					err := errors.New("Write error")
					want := ConnectionError{ConnectionID: "foobar", Wrapped: err, message: "unable to write wire message to network"}
					tnc := &testNetConn{writeerr: err}
					conn := &connection{id: "foobar", nc: tnc, connected: connected}
					got := conn.writeWireMessage(context.Background(), []byte{})
					if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
						t.Errorf("errors do not match. got %v; want %v", got, want)
					}
					if !tnc.closed {
						t.Errorf("failed to closeConnection net.Conn after error writing bytes.")
					}
				})
				tnc := &testNetConn{}
				conn := &connection{id: "foobar", nc: tnc, connected: connected}
				want := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A}
				err := conn.writeWireMessage(context.Background(), want)
				noerr(t, err)
				got := tnc.buf
				if !cmp.Equal(got, want) {
					t.Errorf("writeWireMessage did not write the proper bytes. got %v; want %v", got, want)
				}
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
			t.Run("Read (size)", func(t *testing.T) {
				err := errors.New("Read error")
				want := ConnectionError{ConnectionID: "foobar", Wrapped: err, message: "incomplete read of message header"}
				tnc := &testNetConn{readerr: err}
				conn := &connection{id: "foobar", nc: tnc, connected: connected}
				_, got := conn.readWireMessage(context.Background(), []byte{})
				if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
					t.Errorf("errors do not match. got %v; want %v", got, want)
				}
				if !tnc.closed {
					t.Errorf("failed to closeConnection net.Conn after error writing bytes.")
				}
			})
			t.Run("Read (wire message)", func(t *testing.T) {
				err := errors.New("Read error")
				want := ConnectionError{ConnectionID: "foobar", Wrapped: err, message: "incomplete read of full message"}
				tnc := &testNetConn{readerr: err, buf: []byte{0x11, 0x00, 0x00, 0x00}}
				conn := &connection{id: "foobar", nc: tnc, connected: connected}
				_, got := conn.readWireMessage(context.Background(), []byte{})
				if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
					t.Errorf("errors do not match. got %v; want %v", got, want)
				}
				if !tnc.closed {
					t.Errorf("failed to closeConnection net.Conn after error writing bytes.")
				}
			})
			t.Run("Read (success)", func(t *testing.T) {
				want := []byte{0x0A, 0x00, 0x00, 0x00, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A}
				tnc := &testNetConn{buf: make([]byte, len(want))}
				copy(tnc.buf, want)
				conn := &connection{id: "foobar", nc: tnc, connected: connected}
				got, err := conn.readWireMessage(context.Background(), nil)
				noerr(t, err)
				if !cmp.Equal(got, want) {
					t.Errorf("did not read full wire message. got %v; want %v", got, want)
				}
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
	})
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
