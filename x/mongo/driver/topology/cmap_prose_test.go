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
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/x/mongo/driver/operation"
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

				assert.Equal(t, 1, len(created), "expected 1 opened events, got %d", len(created))
				assert.Equal(t, 1, len(closed), "expected 1 closed events, got %d", len(closed))
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

				// Close and assert that events are published for all conections.
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
