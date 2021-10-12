package topology

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo/address"
	"go.mongodb.org/mongo-driver/x/mongo/driver/operation"
)

func TestPool(t *testing.T) {
	t.Run("newPool", func(t *testing.T) {
		t.Parallel()

		t.Run("should be connected", func(t *testing.T) {
			t.Parallel()

			p, err := newPool(poolConfig{})
			noerr(t, err)

			err = p.connect()
			noerr(t, err)

			assert.Equal(t, connected, p.connected, "expected new pool to be connected")

			err = p.disconnect(context.Background())
			noerr(t, err)
		})
	})
	t.Run("closeConnection", func(t *testing.T) {
		t.Parallel()

		t.Run("can't close connection from different pool", func(t *testing.T) {
			t.Parallel()

			p1, err := newPool(poolConfig{})
			noerr(t, err)

			err = p1.connect()
			noerr(t, err)

			p2, err := newPool(poolConfig{})
			noerr(t, err)

			err = p2.connect()
			noerr(t, err)

			err = p2.closeConnection(&connection{pool: p1})
			assert.Equal(t, ErrWrongPool, err, "expected ErrWrongPool error")

			err = p1.disconnect(context.Background())
			noerr(t, err)
			err = p2.disconnect(context.Background())
			noerr(t, err)
		})
	})
	t.Run("disconnect", func(t *testing.T) {
		t.Parallel()

		t.Run("cannot disconnect multiple times without connect", func(t *testing.T) {
			t.Parallel()

			p, err := newPool(poolConfig{})
			noerr(t, err)

			err = p.connect()
			noerr(t, err)

			err = p.disconnect(context.Background())
			noerr(t, err)

			for i := 0; i < 5; i++ {
				err = p.disconnect(context.Background())
				assert.Equal(
					t,
					ErrPoolDisconnected,
					err,
					"disconnecting an already disconnected pool should return ErrPoolDisconnected")
			}
		})
		t.Run("closes idle connections", func(t *testing.T) {
			t.Parallel()

			disconnected := make(chan struct{})
			var wg sync.WaitGroup
			wg.Add(3)
			addr := bootstrapConnections(t, 3, func(nc net.Conn) {
				<-disconnected
				_, err := nc.Read([]byte{0})
				assert.Error(t, err, "expected net.Conn to be closed after disconnect")
				wg.Done()
			})

			d := newdialer(&net.Dialer{})
			p, err := newPool(poolConfig{
				Address: address.Address(addr.String()),
			}, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)

			err = p.connect()
			noerr(t, err)

			conns := make([]*connection, 3)
			for i := range conns {
				conns[i], err = p.checkOut(context.Background())
				noerr(t, err)
			}
			for i := range conns {
				err = p.checkIn(conns[i])
				noerr(t, err)
			}
			assert.Equal(t, 3, d.lenopened(), "should have opened 3 connections")
			assert.Equal(t, 3, p.availableConnectionCount(), "should have 3 available connections")
			assert.Equal(t, 3, p.totalConnectionCount(), "should have 3 total connections")

			err = p.disconnect(context.Background())
			noerr(t, err)
			assertConnectionsClosed(t, d, 3)
			assert.Equal(t, 0, p.availableConnectionCount(), "should have 0 available connections")
			assert.Equal(t, 0, p.availableConnectionCount(), "should have 0 total connections")

			close(disconnected)
			wg.Wait()
		})
		t.Run("closes all open connections", func(t *testing.T) {
			t.Parallel()

			disconnected := make(chan struct{})
			var wg sync.WaitGroup
			wg.Add(3)
			addr := bootstrapConnections(t, 3, func(nc net.Conn) {
				<-disconnected
				_, err := nc.Read([]byte{0})
				assert.Error(t, err, "expected net.Conn to be closed after disconnect")
				wg.Done()
			})

			d := newdialer(&net.Dialer{})
			p, err := newPool(poolConfig{
				Address: address.Address(addr.String()),
			}, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)

			err = p.connect()
			noerr(t, err)

			conns := make([]*connection, 3)
			for i := range conns {
				conns[i], err = p.checkOut(context.Background())
				noerr(t, err)
			}
			for i := 0; i < 2; i++ {
				err = p.checkIn(conns[i])
				noerr(t, err)
			}
			assert.Equal(t, 3, d.lenopened(), "should have opened 3 connections")
			assert.Equal(t, 2, p.availableConnectionCount(), "should have 2 available connections")
			assert.Equal(t, 3, p.totalConnectionCount(), "should have 3 total connections")

			err = p.disconnect(context.Background())
			noerr(t, err)
			assertConnectionsClosed(t, d, 3)
			assert.Equal(t, 0, p.availableConnectionCount(), "should have 0 available connections")
			assert.Equal(t, 0, p.totalConnectionCount(), "should have 0 total connections")

			close(disconnected)
			wg.Wait()
		})
		t.Run("no race if connections are also connecting", func(t *testing.T) {
			t.Parallel()

			addr := bootstrapConnections(t, 3, func(nc net.Conn) {})
			p, err := newPool(poolConfig{
				Address: address.Address(addr.String()),
			})
			noerr(t, err)

			err = p.connect()
			noerr(t, err)

			_, err = p.checkOut(context.Background())
			noerr(t, err)

			disconnected := make(chan struct{})
			started := make(chan struct{})
			go func() {
				close(started)

				for {
					select {
					case <-disconnected:
						return
					default:
						c, _ := p.checkOut(context.Background())
						_ = p.checkIn(c)
						time.Sleep(time.Millisecond)
					}
				}
			}()

			// Wait for the background goroutine to start running before trying to disconnect the
			// connection pool.
			<-started
			_, err = p.checkOut(context.Background())
			noerr(t, err)

			err = p.disconnect(context.Background())
			noerr(t, err)

			close(disconnected)
		})
		t.Run("shuts down gracefully if Context has a deadline", func(t *testing.T) {
			t.Parallel()

			addr := bootstrapConnections(t, 3, func(nc net.Conn) {})
			p, err := newPool(poolConfig{
				Address: address.Address(addr.String()),
			})
			noerr(t, err)

			err = p.connect()
			noerr(t, err)

			// Check out 2 connections from the pool and add them to a conns slice.
			conns := make([]*connection, 2)
			for i := 0; i < 2; i++ {
				c, err := p.checkOut(context.Background())
				noerr(t, err)

				conns[i] = c
			}

			// Check out a 3rd connection from the pool and immediately check it back in so there is
			// a mixture of in-use and idle connections.
			c, err := p.checkOut(context.Background())
			noerr(t, err)

			err = p.checkIn(c)
			noerr(t, err)

			// Start a goroutine that waits for the pool to start disconnecting, then checks in the
			// 2 in-use connections. Assert that both connections are still connected during
			// graceful shutdown before they are checked in.
			go func() {
				for atomic.LoadInt64(&p.connected) == connected {
					time.Sleep(time.Millisecond)
				}
				for _, c := range conns {
					assert.Equal(t, connected, c.connected, "expected conn to still be connected")

					err := p.checkIn(c)
					noerr(t, err)
				}
			}()

			// Disconnect the pool with a 1-hour graceful shutdown timeout. Expect that the call to
			// disconnect() returns when all of the connections are checked in. If disconnect()
			// doesn't return when all of the connections are checked in, the test will time out.
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
			defer cancel()
			err = p.disconnect(ctx)
			noerr(t, err)
		})
		t.Run("closing a Connection does not cause an error after pool is disconnected", func(t *testing.T) {
			t.Parallel()

			addr := bootstrapConnections(t, 3, func(nc net.Conn) {})
			p, err := newPool(poolConfig{
				Address: address.Address(addr.String()),
			})
			noerr(t, err)

			err = p.connect()
			noerr(t, err)

			c, err := p.checkOut(context.Background())
			noerr(t, err)

			err = p.disconnect(context.Background())
			noerr(t, err)

			c1 := &Connection{connection: c}
			err = c1.Close()
			noerr(t, err)
		})
	})
	t.Run("connect", func(t *testing.T) {
		t.Parallel()

		t.Run("can reconnect a disconnected pool", func(t *testing.T) {
			t.Parallel()

			addr := bootstrapConnections(t, 6, func(nc net.Conn) {})

			p, err := newPool(poolConfig{
				Address: address.Address(addr.String()),
			})
			noerr(t, err)

			err = p.connect()
			noerr(t, err)

			for i := 0; i < 3; i++ {
				_, err := p.checkOut(context.Background())
				noerr(t, err)
			}
			assert.Equal(t, 0, p.availableConnectionCount(), "should have 0 available connections")
			assert.Equal(t, 3, p.totalConnectionCount(), "should have 3 total connections")

			err = p.disconnect(context.Background())
			noerr(t, err)
			assert.Equal(t, 0, p.availableConnectionCount(), "should have 0 available connections")
			assert.Equal(t, 0, p.totalConnectionCount(), "should have 0 total connections")

			err = p.connect()
			noerr(t, err)

			for i := 0; i < 3; i++ {
				_, err := p.checkOut(context.Background())
				noerr(t, err)
			}
			assert.Equal(t, 0, p.availableConnectionCount(), "should have 0 available connections")
			assert.Equal(t, 3, p.totalConnectionCount(), "should have 3 total connections")

			err = p.disconnect(context.Background())
			noerr(t, err)
		})
		t.Run("cannot connect multiple times without disconnect", func(t *testing.T) {
			t.Parallel()

			p, err := newPool(poolConfig{})
			noerr(t, err)

			err = p.connect()
			noerr(t, err)

			for i := 0; i < 5; i++ {
				err = p.connect()
				assert.Equal(
					t,
					ErrPoolConnected,
					err,
					"connecting an already connected pool should return ErrPoolConnected")
			}

			err = p.disconnect(context.Background())
			noerr(t, err)
		})
		t.Run("can disconnect and reconnect multiple times", func(t *testing.T) {
			t.Parallel()

			p, err := newPool(poolConfig{})
			noerr(t, err)

			for i := 0; i < 100; i++ {
				err = p.connect()
				noerr(t, err)

				err = p.disconnect(context.Background())
				noerr(t, err)
			}
		})
	})
	t.Run("checkOut", func(t *testing.T) {
		t.Parallel()

		t.Run("return error when attempting to create new connection", func(t *testing.T) {
			t.Parallel()

			dialErr := errors.New("create new connection error")
			p, err := newPool(poolConfig{}, WithDialer(func(Dialer) Dialer {
				return DialerFunc(func(context.Context, string, string) (net.Conn, error) {
					return nil, dialErr
				})
			}))
			noerr(t, err)

			err = p.connect()
			noerr(t, err)

			_, err = p.checkOut(context.Background())
			var want error = ConnectionError{Wrapped: dialErr, init: true}
			assert.Equal(t, want, err, "should return error from calling checkOut()")
			assert.Equal(t, 0, p.totalConnectionCount(), "pool should have 0 total connections")

			err = p.disconnect(context.Background())
			noerr(t, err)
		})
		t.Run("closes perished connections", func(t *testing.T) {
			t.Parallel()

			addr := bootstrapConnections(t, 2, func(nc net.Conn) {})
			d := newdialer(&net.Dialer{})
			p, err := newPool(
				poolConfig{
					Address:     address.Address(addr.String()),
					MaxIdleTime: time.Millisecond,
				},
				WithDialer(func(Dialer) Dialer { return d }),
			)
			noerr(t, err)

			err = p.connect()
			noerr(t, err)

			// Check out a connection and assert that the idle timeout is properly set then check it
			// back into the pool.
			c1, err := p.checkOut(context.Background())
			noerr(t, err)
			assert.Equal(t, 1, d.lenopened(), "should have opened 1 connection")
			assert.Equal(t, 1, p.totalConnectionCount(), "pool should have 1 total connection")
			assert.Equal(t, time.Millisecond, c1.idleTimeout, "connection should have a 1ms idle timeout")

			err = p.checkIn(c1)
			noerr(t, err)

			// Sleep for more than the 1ms idle timeout and then try to check out a connection.
			// Expect that the previously checked-out connection is closed because it's idle and a
			// new connection is created.
			time.Sleep(50 * time.Millisecond)
			c2, err := p.checkOut(context.Background())
			noerr(t, err)
			assert.NotEqual(t, c1, c2, "expected a new connection on 2nd check out after idle timeout expires")
			assert.Equal(t, 2, d.lenopened(), "should have opened 2 connections")
			assert.Equal(t, 1, p.totalConnectionCount(), "pool should have 1 total connection")

			err = p.disconnect(context.Background())
			noerr(t, err)
		})
		t.Run("recycles connections", func(t *testing.T) {
			t.Parallel()

			addr := bootstrapConnections(t, 1, func(nc net.Conn) {})
			d := newdialer(&net.Dialer{})
			p, err := newPool(poolConfig{
				Address: address.Address(addr.String()),
			}, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)

			err = p.connect()
			noerr(t, err)

			for i := 0; i < 100; i++ {
				c, err := p.checkOut(context.Background())
				noerr(t, err)

				err = p.checkIn(c)
				noerr(t, err)
			}
			assert.Equal(t, 1, d.lenopened(), "should have opened 1 connection")
		})
		t.Run("cannot checkOut from disconnected pool", func(t *testing.T) {
			t.Parallel()

			addr := bootstrapConnections(t, 3, func(nc net.Conn) {})
			p, err := newPool(poolConfig{
				Address: address.Address(addr.String()),
			})
			noerr(t, err)

			err = p.connect()
			noerr(t, err)

			err = p.disconnect(context.Background())
			noerr(t, err)

			_, err = p.checkOut(context.Background())
			assert.Equal(
				t,
				ErrPoolDisconnected,
				err,
				"expected an error from checkOut() from a disconnected pool")
		})
		t.Run("handshaker i/o fails", func(t *testing.T) {
			t.Parallel()

			p, err := newPool(
				poolConfig{},
				WithHandshaker(func(Handshaker) Handshaker {
					return operation.NewHello()
				}),
				WithDialer(func(Dialer) Dialer {
					return DialerFunc(func(context.Context, string, string) (net.Conn, error) {
						return &writeFailConn{&net.TCPConn{}}, nil
					})
				}),
			)
			noerr(t, err)

			err = p.connect()
			noerr(t, err)

			_, err = p.checkOut(context.Background())
			assert.IsType(t, ConnectionError{}, err, "expected a ConnectionError")
			if err, ok := err.(ConnectionError); ok {
				assert.Contains(
					t,
					err.Unwrap().Error(),
					"unable to write wire message to network: Write error")
			}
			assert.Equal(t, 0, p.totalConnectionCount(), "pool should have 0 total connections")

			err = p.disconnect(context.Background())
			noerr(t, err)
		})
		t.Run("wait queue timeout error", func(t *testing.T) {
			t.Parallel()

			addr := bootstrapConnections(t, 1, func(nc net.Conn) {})
			p, err := newPool(poolConfig{
				Address:     address.Address(addr.String()),
				MaxPoolSize: 1,
			})
			noerr(t, err)

			err = p.connect()
			noerr(t, err)

			// check out first connection.
			_, err = p.checkOut(context.Background())
			noerr(t, err)

			// Set a short timeout and check out again.
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			_, err = p.checkOut(ctx)
			assert.NotNil(t, err, "expected a WaitQueueTimeout error")

			// Assert that error received is WaitQueueTimeoutError with context deadline exceeded.
			assert.IsType(t, WaitQueueTimeoutError{}, err, "expected a WaitQueueTimeoutError")
			if err, ok := err.(WaitQueueTimeoutError); ok {
				assert.Equal(t, context.DeadlineExceeded, err.Unwrap(), "expected wrapped error to be a context.Timeout")
			}

			err = p.disconnect(context.Background())
			noerr(t, err)
		})
	})
	t.Run("checkIn", func(t *testing.T) {
		t.Parallel()

		t.Run("does not return to pool twice", func(t *testing.T) {
			t.Parallel()

			addr := bootstrapConnections(t, 1, func(nc net.Conn) {})
			p, err := newPool(poolConfig{
				Address: address.Address(addr.String()),
			})
			noerr(t, err)

			err = p.connect()
			noerr(t, err)

			c, err := p.checkOut(context.Background())
			noerr(t, err)
			assert.Equal(t, 0, p.availableConnectionCount(), "should be no idle connections in pool")
			assert.Equal(t, 1, p.totalConnectionCount(), "should be 1 total connection in pool")

			err = p.checkIn(c)
			noerr(t, err)

			err = p.checkIn(c)
			assert.NotNil(t, err, "expected an error trying to return the same conn to the pool twice")

			assert.Equal(t, 1, p.availableConnectionCount(), "should have returned 1 idle connection to the pool")
			assert.Equal(t, 1, p.totalConnectionCount(), "should have 1 total connection in pool")

			err = p.disconnect(context.Background())
			noerr(t, err)
		})
	})
	t.Run("maintain", func(t *testing.T) {
		t.Parallel()

		t.Run("creates MinPoolSize connections shortly after calling connect()", func(t *testing.T) {
			t.Parallel()

			addr := bootstrapConnections(t, 3, func(nc net.Conn) {})
			d := newdialer(&net.Dialer{})
			p, err := newPool(poolConfig{
				Address:     address.Address(addr.String()),
				MinPoolSize: 3,
			}, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)

			err = p.connect()
			noerr(t, err)

			assertConnectionsOpened(t, d, 3)
			assert.Equal(t, 3, p.availableConnectionCount(), "should be 3 idle connections in pool")
			assert.Equal(t, 3, p.totalConnectionCount(), "should be 3 total connection in pool")

			err = p.disconnect(context.Background())
			noerr(t, err)
		})
		t.Run("removes perished connections", func(t *testing.T) {
			t.Parallel()

			addr := bootstrapConnections(t, 5, func(nc net.Conn) {})
			d := newdialer(&net.Dialer{})
			p, err := newPool(poolConfig{
				Address: address.Address(addr.String()),
			}, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)

			// Set the pool's maintain interval to 10ms so that it allows the test to run quickly.
			p.maintainInterval = 10 * time.Millisecond
			err = p.connect()
			noerr(t, err)

			// Check out and check in 3 connections. Assert that there are 3 total and 3 idle
			// connections in the pool.
			conns := make([]*connection, 3)
			for i := range conns {
				conns[i], err = p.checkOut(context.Background())
				noerr(t, err)
			}
			for _, c := range conns {
				err = p.checkIn(c)
				noerr(t, err)
			}
			assert.Equal(t, 3, d.lenopened(), "should have opened 3 connections")
			assert.Equal(t, 3, p.availableConnectionCount(), "should be 3 idle connections in pool")
			assert.Equal(t, 3, p.totalConnectionCount(), "should be 3 total connection in pool")

			// Manually make two of the connections in the idle connections stack perished due to
			// passing the connection's idle deadline. Assert that maintain() closes the two
			// perished connections and removes them from the pool.
			p.idleMu.Lock()
			for i := 0; i < 2; i++ {
				p.idleConns[i].idleTimeout = time.Millisecond
				p.idleConns[i].idleDeadline.Store(time.Now().Add(-1 * time.Hour))
			}
			p.idleMu.Unlock()
			assertConnectionsClosed(t, d, 2)
			assert.Equal(t, 1, p.availableConnectionCount(), "should be 1 idle connections in pool")
			assert.Equal(t, 1, p.totalConnectionCount(), "should be 1 total connection in pool")

			err = p.disconnect(context.Background())
			noerr(t, err)
		})
		t.Run("removes perished connections and replaces them to maintain MinPoolSize", func(t *testing.T) {
			t.Parallel()

			addr := bootstrapConnections(t, 5, func(nc net.Conn) {})
			d := newdialer(&net.Dialer{})
			p, err := newPool(poolConfig{
				Address:     address.Address(addr.String()),
				MinPoolSize: 3,
			}, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)

			// Set the pool's maintain interval to 10ms so that it allows the test to run quickly.
			p.maintainInterval = 10 * time.Millisecond
			err = p.connect()
			noerr(t, err)
			assertConnectionsOpened(t, d, 3)
			assert.Equal(t, 3, p.availableConnectionCount(), "should be 3 idle connections in pool")
			assert.Equal(t, 3, p.totalConnectionCount(), "should be 3 total connection in pool")

			p.idleMu.Lock()
			for i := 0; i < 2; i++ {
				p.idleConns[i].idleTimeout = time.Millisecond
				p.idleConns[i].idleDeadline.Store(time.Now().Add(-1 * time.Hour))
			}
			p.idleMu.Unlock()
			assertConnectionsClosed(t, d, 2)
			assertConnectionsOpened(t, d, 5)
			assert.Equal(t, 3, p.availableConnectionCount(), "should be 3 idle connections in pool")
			assert.Equal(t, 3, p.totalConnectionCount(), "should be 3 total connection in pool")

			err = p.disconnect(context.Background())
			noerr(t, err)
		})
	})
}

func assertConnectionsClosed(t *testing.T, dialer *dialer, count int) {
	t.Helper()

	start := time.Now()
	for {
		if dialer.lenclosed() == count {
			return
		}
		if time.Since(start) > 3*time.Second {
			t.Errorf(
				"Waited for 3 seconds for %d connections to be closed, but got %d",
				count,
				dialer.lenclosed())
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func assertConnectionsOpened(t *testing.T, dialer *dialer, count int) {
	t.Helper()

	start := time.Now()
	for {
		if dialer.lenopened() == count {
			return
		}
		if time.Since(start) > 3*time.Second {
			t.Errorf(
				"Waited for 3 seconds for %d connections to be opened, but got %d",
				count,
				dialer.lenopened())
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}
