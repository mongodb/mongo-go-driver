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
	"regexp"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/csot"
	"go.mongodb.org/mongo-driver/v2/internal/driverutil"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo/address"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/operation"
)

func TestCMAPProse(t *testing.T) {
	t.Run("created and closed events", func(t *testing.T) {
		created := make(chan *event.PoolEvent, 10)
		closed := make(chan *event.PoolEvent, 10)
		clearEvents := func() {
			for len(created) > 0 {
				<-created
			}
			for len(closed) > 0 {
				<-closed
			}
		}
		monitor := &event.PoolMonitor{
			Event: func(evt *event.PoolEvent) {
				switch evt.Type {
				case event.ConnectionCreated:
					created <- evt
				case event.ConnectionClosed:
					closed <- evt
				}
			},
		}
		getConfig := func() poolConfig {
			return poolConfig{
				PoolMonitor: monitor,
			}
		}
		assertConnectionCounts := func(t *testing.T, p *pool, numCreated, numClosed int) {
			t.Helper()

			require.Eventuallyf(t,
				func() bool {
					return numCreated == len(created) && numClosed == len(closed)
				},
				1*time.Second,
				10*time.Millisecond,
				"expected %d creation events, got %d; expected %d closed events, got %d",
				numCreated,
				len(created),
				numClosed,
				len(closed))

			netCount := numCreated - numClosed
			assert.Equal(t, netCount, p.totalConnectionCount(), "expected %d total connections, got %d", netCount,
				p.totalConnectionCount())
		}

		t.Run("maintain", func(t *testing.T) {
			t.Run("connection error publishes events", func(t *testing.T) {
				// If a connection is created as part of minPoolSize maintenance and errors while connecting, checkOut()
				// should report that error and publish an event.
				// If maintain() creates a connection that encounters an error while connecting,
				// the pool should publish connection created and closed events.
				clearEvents()

				var dialer DialerFunc = func(context.Context, string, string) (net.Conn, error) {
					return &testNetConn{writeerr: errors.New("write error")}, nil
				}

				cfg := getConfig()
				cfg.MinPoolSize = 1
				connOpts := []ConnectionOption{
					WithDialer(func(Dialer) Dialer { return dialer }),
					WithHandshaker(func(Handshaker) Handshaker {
						return operation.NewHello()
					}),
				}
				pool := createTestPool(t, cfg, connOpts...)
				defer pool.close(context.Background())

				// Wait up to 3 seconds for the maintain() goroutine to run and for 1 connection
				// created and 1 connection closed events to be published.
				start := time.Now()
				for len(created) != 1 || len(closed) != 1 {
					if time.Since(start) > 3*time.Second {
						t.Errorf(
							"Expected 1 connection created and 1 connection closed events within 3 seconds. "+
								"Actual created events: %d, actual closed events: %d",
							len(created),
							len(closed))
					}
					time.Sleep(time.Millisecond)
				}
			})
		})
		t.Run("checkOut", func(t *testing.T) {
			t.Run("connection error publishes events", func(t *testing.T) {
				// If checkOut() creates a connection that encounters an error while connecting,
				// the pool should publish connection created and closed events and checkOut should
				// return the error.
				clearEvents()

				var dialer DialerFunc = func(context.Context, string, string) (net.Conn, error) {
					return &testNetConn{writeerr: errors.New("write error")}, nil
				}

				cfg := getConfig()
				connOpts := []ConnectionOption{
					WithDialer(func(Dialer) Dialer { return dialer }),
					WithHandshaker(func(Handshaker) Handshaker {
						return operation.NewHello()
					}),
				}
				pool := createTestPool(t, cfg, connOpts...)
				defer pool.close(context.Background())

				_, err := pool.checkOut(context.Background())
				assert.NotNil(t, err, "expected checkOut() error, got nil")

				assertConnectionCounts(t, pool, 1, 1)
			})
			t.Run("pool is empty", func(t *testing.T) {
				// If a checkOut() has to create a new connection and that connection encounters an
				// error while connecting, checkOut() should return that error and publish an event.
				clearEvents()

				var dialer DialerFunc = func(context.Context, string, string) (net.Conn, error) {
					return &testNetConn{writeerr: errors.New("write error")}, nil
				}

				connOpts := []ConnectionOption{
					WithDialer(func(Dialer) Dialer { return dialer }),
					WithHandshaker(func(Handshaker) Handshaker {
						return operation.NewHello()
					}),
				}
				pool := createTestPool(t, getConfig(), connOpts...)
				defer pool.close(context.Background())

				_, err := pool.checkOut(context.Background())
				assert.NotNil(t, err, "expected checkOut() error, got nil")
				assertConnectionCounts(t, pool, 1, 1)
			})
		})
		t.Run("checkIn", func(t *testing.T) {
			t.Run("errored connection", func(t *testing.T) {
				// If the connection being returned to the pool encountered a network error, it should be removed from
				// the pool and an event should be published.
				clearEvents()

				var dialer DialerFunc = func(context.Context, string, string) (net.Conn, error) {
					return &testNetConn{writeerr: errors.New("write error")}, nil
				}

				// We don't use the WithHandshaker option so the connection won't error during handshaking.
				connOpts := []ConnectionOption{
					WithDialer(func(Dialer) Dialer { return dialer }),
				}
				pool := createTestPool(t, getConfig(), connOpts...)
				defer pool.close(context.Background())

				conn, err := pool.checkOut(context.Background())
				assert.Nil(t, err, "checkOut() error: %v", err)

				// Force a network error by writing to the connection.
				err = conn.writeWireMessage(context.Background(), nil)
				assert.NotNil(t, err, "expected writeWireMessage error, got nil")

				err = pool.checkIn(conn)
				assert.Nil(t, err, "checkIn() error: %v", err)

				assertConnectionCounts(t, pool, 1, 1)
				evt := <-closed
				assert.Equal(t, event.ReasonError, evt.Reason, "expected reason %q, got %q",
					event.ReasonError, evt.Reason)
			})
		})
		t.Run("close", func(t *testing.T) {
			t.Run("connections returned gracefully", func(t *testing.T) {
				// If all connections are in the pool when close is called, they should be closed gracefully and
				// events should be published.
				clearEvents()

				numConns := 5
				var dialer DialerFunc = func(context.Context, string, string) (net.Conn, error) {
					return &testNetConn{}, nil
				}
				pool := createTestPool(t, getConfig(), WithDialer(func(Dialer) Dialer { return dialer }))
				defer pool.close(context.Background())

				conns := checkoutConnections(t, pool, numConns)
				assertConnectionCounts(t, pool, numConns, 0)

				// Return all connections to the pool and assert that none were closed by checkIn().
				for i, c := range conns {
					err := pool.checkIn(c)
					assert.Nil(t, err, "checkIn() error at index %d: %v", i, err)
				}
				assertConnectionCounts(t, pool, numConns, 0)

				// Close the pool and assert that a closed event is published for each connection.
				pool.close(context.Background())
				assertConnectionCounts(t, pool, numConns, numConns)

				for len(closed) > 0 {
					evt := <-closed
					assert.Equal(t, event.ReasonPoolClosed, evt.Reason, "expected reason %q, got %q",
						event.ReasonPoolClosed, evt.Reason)
				}
			})
			t.Run("connections closed forcefully", func(t *testing.T) {
				// If some connections are still checked out when close is called, they should be closed
				// forcefully and events should be published for them.
				clearEvents()

				numConns := 5
				var dialer DialerFunc = func(context.Context, string, string) (net.Conn, error) {
					return &testNetConn{}, nil
				}
				pool := createTestPool(t, getConfig(), WithDialer(func(Dialer) Dialer { return dialer }))

				conns := checkoutConnections(t, pool, numConns)
				assertConnectionCounts(t, pool, numConns, 0)

				// Only return 2 of the connection.
				for i := 0; i < 2; i++ {
					err := pool.checkIn(conns[i])
					assert.Nil(t, err, "checkIn() error at index %d: %v", i, err)
				}
				conns = conns[2:]
				assertConnectionCounts(t, pool, numConns, 0)

				// Close and assert that events are published for all connections.
				pool.close(context.Background())
				assertConnectionCounts(t, pool, numConns, numConns)

				// Return the remaining connections and assert that the closed event count does not increase because
				// these connections have already been closed.
				for i, c := range conns {
					err := pool.checkIn(c)
					assert.Nil(t, err, "checkIn() error at index %d: %v", i, err)
				}
				assertConnectionCounts(t, pool, numConns, numConns)

				// Ensure all closed events have the correct reason.
				for len(closed) > 0 {
					evt := <-closed
					assert.Equal(t, event.ReasonPoolClosed, evt.Reason, "expected reason %q, got %q",
						event.ReasonPoolClosed, evt.Reason)
				}

			})
		})
	})

	// Need to test the case where we attempt a non-blocking read to determine if
	// we should refresh the remaining time. In the case of the Go Driver, we do
	// this by attempt to "peek" at 1 byte with a deadline of 1ns.
	//t.Run("connection attempts peek but fails", func(t *testing.T) {
	//	// Create a client using a direct connection to the proxy.
	//	client, err := mongo.Connect(
	//})

	t.Run("connection attempts peek and succeeds", func(t *testing.T) {
		const requestID = int32(-1)
		timeout := 10 * time.Millisecond

		// Mock a TCP listener that will write a byte sequence > 5 (to avoid errors
		// due to size) to the TCP socket. Have the listener sleep for 2x the
		// timeout provided to the connection AFTER writing the byte sequence. This
		// wiill cause the connection to timeout while reading from the socket.
		addr := bootstrapConnections(t, 1, func(nc net.Conn) {
			defer func() {
				_ = nc.Close()
			}()

			_, err := nc.Write([]byte{12, 0, 0, 0, 0, 0, 0, 0, 1})
			require.NoError(t, err)
			time.Sleep(timeout * 2)

			// Write data that can be peeked at.
			_, err = nc.Write([]byte{12, 0, 0, 0, 0, 0, 0, 0, 1})
			require.NoError(t, err)

		})

		poolEventsByType := make(map[string][]event.PoolEvent)
		poolEventsByTypeMu := &sync.Mutex{}

		monitor := &event.PoolMonitor{
			Event: func(pe *event.PoolEvent) {
				poolEventsByTypeMu.Lock()
				poolEventsByType[pe.Type] = append(poolEventsByType[pe.Type], *pe)
				poolEventsByTypeMu.Unlock()
			},
		}

		p := newPool(
			poolConfig{
				Address:     address.Address(addr.String()),
				PoolMonitor: monitor,
			},
		)
		defer p.close(context.Background())
		err := p.ready()
		require.NoError(t, err)

		// Check out a connection and read from the socket, causing a timeout and
		// pinning the connection to a pending read state.
		conn, err := p.checkOut(context.Background())
		require.NoError(t, err)

		ctx, cancel := csot.WithTimeout(context.Background(), &timeout)
		defer cancel()

		ctx = driverutil.WithValueHasMaxTimeMS(ctx, true)
		ctx = driverutil.WithRequestID(ctx, requestID)

		_, err = conn.readWireMessage(ctx)
		regex := regexp.MustCompile(
			`^connection\(.*\[-\d+\]\) incomplete read of full message: context deadline exceeded: read tcp 127.0.0.1:.*->127.0.0.1:.*: i\/o timeout$`,
		)
		assert.True(t, regex.MatchString(err.Error()), "error %q does not match pattern %q", err, regex)

		// Check in the connection with a pending read state. The next time this
		// connection is checked out, it should attempt to read the pending
		// response.
		err = p.checkIn(conn)
		require.NoError(t, err)

		// Wait 3s to make sure there is no remaining time on the pending read
		// state.
		time.Sleep(3 * time.Second)

		// Check out the connection again. The remaining time should be exhausted
		// requiring us to "peek" at the connection to determine if we should
		// close as not alive.
		_, err = p.checkOut(context.Background())
		require.NoError(t, err)

		// There should be 1 ConnectionPendingResponseStarted event.
		started := poolEventsByType[event.ConnectionPendingResponseStarted]
		require.Len(t, started, 1)

		assert.Equal(t, addr.String(), started[0].Address)
		assert.Equal(t, conn.driverConnectionID, started[0].ConnectionID)
		assert.Equal(t, requestID, started[0].RequestID)

		// There should be 0 ConnectionPendingResponseFailed event.
		require.Len(t, poolEventsByType[event.ConnectionPendingResponseFailed], 0)

		// There should be 1 ConnectionPendingResponseSucceeded event.
		succeeded := poolEventsByType[event.ConnectionPendingResponseSucceeded]
		require.Len(t, succeeded, 1)

		assert.Equal(t, addr.String(), succeeded[0].Address)
		assert.Equal(t, conn.driverConnectionID, succeeded[0].ConnectionID)
		assert.Equal(t, requestID, succeeded[0].RequestID)
		assert.Greater(t, int(succeeded[0].Duration), 0)

		// The connection should not have been closed.
		require.Len(t, poolEventsByType[event.ConnectionClosed], 0)
	})
}

func createTestPool(t *testing.T, cfg poolConfig, opts ...ConnectionOption) *pool {
	t.Helper()

	pool := newPool(cfg, opts...)
	err := pool.ready()
	assert.Nil(t, err, "connect error: %v", err)
	return pool
}

func checkoutConnections(t *testing.T, p *pool, numConns int) []*connection {
	conns := make([]*connection, 0, numConns)

	for i := 0; i < numConns; i++ {
		conn, err := p.checkOut(context.Background())
		assert.Nil(t, err, "checkOut() error at index %d: %v", i, err)
		conns = append(conns, conn)
	}

	return conns
}
