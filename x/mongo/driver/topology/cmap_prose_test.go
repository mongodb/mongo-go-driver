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
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
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

			assert.Equal(t, numCreated, len(created), "expected %d creation events, got %d", numCreated, len(created))
			assert.Equal(t, numClosed, len(closed), "expected %d closed events, got %d", numClosed, len(closed))

			netCount := numCreated - numClosed
			assert.Equal(t, netCount, len(p.opened), "expected %d connections in opened map, got %d", netCount,
				len(p.opened))
		}

		t.Run("get", func(t *testing.T) {
			t.Run("errored connection exists in pool", func(t *testing.T) {
				// If a connection is created as part of minPoolSize maintenance and errors while connecting, get()
				// should report that error and publish an event.
				clearEvents()

				var dialer DialerFunc = func(context.Context, string, string) (net.Conn, error) {
					return &testNetConn{writeerr: errors.New("write error")}, nil
				}

				cfg := getConfig()
				cfg.MinPoolSize = 1
				connOpts := []ConnectionOption{
					WithDialer(func(Dialer) Dialer { return dialer }),
					WithHandshaker(func(Handshaker) Handshaker {
						return operation.NewIsMaster()
					}),
				}
				pool := createTestPool(t, cfg, connOpts...)

				_, err := pool.get(context.Background())
				assert.NotNil(t, err, "expected get() error, got nil")

				// If the connection doesn't finish connecting before resourcePool gives it back, the error will be
				// detected by pool.get and result in a created/closed count of 1. If it does finish connecting, the
				// error will be detected by resourcePool, which will return nil. Then, pool will try to create a new
				// connection, which will also error. This process will result in a created/closed count of 2.
				assert.True(t, len(created) == 1 || len(created) == 2, "expected 1 or 2 opened events, got %d", len(created))
				assert.True(t, len(closed) == 1 || len(closed) == 2, "expected 1 or 2 closed events, got %d", len(closed))
				netCount := len(created) - len(closed)
				assert.Equal(t, 0, netCount, "expected net connection count to be 0, got %d", netCount)
			})
			t.Run("pool is empty", func(t *testing.T) {
				// If a new connection is created during get(), get() should report that error and publish an event.
				clearEvents()

				var dialer DialerFunc = func(context.Context, string, string) (net.Conn, error) {
					return &testNetConn{writeerr: errors.New("write error")}, nil
				}

				connOpts := []ConnectionOption{
					WithDialer(func(Dialer) Dialer { return dialer }),
					WithHandshaker(func(Handshaker) Handshaker {
						return operation.NewIsMaster()
					}),
				}
				pool := createTestPool(t, getConfig(), connOpts...)

				_, err := pool.get(context.Background())
				assert.NotNil(t, err, "expected get() error, got nil")
				assertConnectionCounts(t, pool, 1, 1)
			})
		})
		t.Run("put", func(t *testing.T) {
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

				conn, err := pool.get(context.Background())
				assert.Nil(t, err, "get error: %v", err)

				// Force a network error by writing to the connection.
				err = conn.writeWireMessage(context.Background(), nil)
				assert.NotNil(t, err, "expected writeWireMessage error, got nil")

				err = pool.put(conn)
				assert.Nil(t, err, "put error: %v", err)

				assertConnectionCounts(t, pool, 1, 1)
				evt := <-closed
				assert.Equal(t, event.ReasonConnectionErrored, evt.Reason, "expected reason %q, got %q",
					event.ReasonConnectionErrored, evt.Reason)
			})
			t.Run("expired connection", func(t *testing.T) {
				// If the connection being returned to the pool is expired, it should be removed from the pool and an
				// event should be published.
				clearEvents()

				var dialer DialerFunc = func(context.Context, string, string) (net.Conn, error) {
					return &testNetConn{}, nil
				}

				// We don't use the WithHandshaker option so the connection won't error during handshaking.
				// WithIdleTimeout must be used because the connection.idleTimeoutExpired() function only checks the
				// deadline if the idleTimeout option is greater than 0.
				connOpts := []ConnectionOption{
					WithDialer(func(Dialer) Dialer { return dialer }),
					WithIdleTimeout(func(time.Duration) time.Duration { return 1 * time.Second }),
				}
				pool := createTestPool(t, getConfig(), connOpts...)

				conn, err := pool.get(context.Background())
				assert.Nil(t, err, "get error: %v", err)

				// Set the idleDeadline to a time in the past to simulate expiration.
				pastTime := time.Now().Add(-10 * time.Second)
				conn.idleDeadline.Store(pastTime)

				err = pool.put(conn)
				assert.Nil(t, err, "put error: %v", err)

				assertConnectionCounts(t, pool, 1, 1)
				evt := <-closed
				assert.Equal(t, event.ReasonIdle, evt.Reason, "expected reason %q, got %q",
					event.ReasonIdle, evt.Reason)
			})
		})
		t.Run("disconnect", func(t *testing.T) {
			t.Run("connections returned gracefully", func(t *testing.T) {
				// If all connections are in the pool when disconnect is called, they should be closed gracefully and
				// events should be published.
				clearEvents()

				numConns := 5
				var dialer DialerFunc = func(context.Context, string, string) (net.Conn, error) {
					return &testNetConn{}, nil
				}
				pool := createTestPool(t, getConfig(), WithDialer(func(Dialer) Dialer { return dialer }))

				conns := checkoutConnections(t, pool, numConns)
				assertConnectionCounts(t, pool, numConns, 0)

				// Return all connections to the pool and assert that none were closed by put().
				for i, c := range conns {
					err := pool.put(c)
					assert.Nil(t, err, "put error at index %d: %v", i, err)
				}
				assertConnectionCounts(t, pool, numConns, 0)

				// Disconnect the pool and assert that a closed event is published for each connection.
				err := pool.disconnect(context.Background())
				assert.Nil(t, err, "disconnect error: %v", err)
				assertConnectionCounts(t, pool, numConns, numConns)

				for len(closed) > 0 {
					evt := <-closed
					assert.Equal(t, event.ReasonPoolClosed, evt.Reason, "expected reason %q, got %q",
						event.ReasonPoolClosed, evt.Reason)
				}
			})
			t.Run("connections closed forcefully", func(t *testing.T) {
				// If some connections are still checked out when disconnect is called, they should be closed
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
					err := pool.put(conns[i])
					assert.Nil(t, err, "put error at index %d: %v", i, err)
				}
				conns = conns[2:]
				assertConnectionCounts(t, pool, numConns, 0)

				// Disconnect and assert that events are published for all conections.
				err := pool.disconnect(context.Background())
				assert.Nil(t, err, "disconnect error: %v", err)
				assertConnectionCounts(t, pool, numConns, numConns)

				// Return the remaining connections and assert that the closed event count does not increase because
				// these connections have already been closed.
				for i, c := range conns {
					err := pool.put(c)
					assert.Nil(t, err, "put error at index %d: %v", i, err)
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

	pool, err := newPool(cfg, opts...)
	assert.Nil(t, err, "newPool error: %v", err)
	err = pool.connect()
	assert.Nil(t, err, "connect error: %v", err)

	return pool
}

func checkoutConnections(t *testing.T, p *pool, numConns int) []*connection {
	conns := make([]*connection, 0, numConns)

	for i := 0; i < numConns; i++ {
		conn, err := p.get(context.Background())
		assert.Nil(t, err, "get error at index %d: %v", i, err)
		conns = append(conns, conn)
	}

	return conns
}
