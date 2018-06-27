package connection

import (
	"context"
	"errors"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mongodb/mongo-go-driver/core/address"
)

func TestPool(t *testing.T) {
	noerr := func(t *testing.T, err error) {
		t.Helper()
		if err != nil {
			t.Errorf("Unepexted error: %v", err)
			t.FailNow()
		}
	}
	t.Run("NewPool", func(t *testing.T) {
		t.Run("should be connected", func(t *testing.T) {
			P, err := NewPool(address.Address(""), 1, 2)
			p := P.(*pool)
			noerr(t, err)
			err = p.Connect(context.Background())
			noerr(t, err)
			if p.connected != connected {
				t.Errorf("Expected new pool to be connected. got %v; want %v", p.connected, connected)
			}
		})
		t.Run("size cannot be larger than capcity", func(t *testing.T) {
			_, err := NewPool(address.Address(""), 5, 1)
			if err != ErrSizeLargerThanCapacity {
				t.Errorf("Should receive error when size is larger than capacity. got %v; want %v", err, ErrSizeLargerThanCapacity)
			}
		})
	})
	t.Run("Disconnect", func(t *testing.T) {
		t.Run("cannot disconnect twice", func(t *testing.T) {
			p, err := NewPool(address.Address(""), 1, 2)
			noerr(t, err)
			err = p.Connect(context.Background())
			noerr(t, err)
			err = p.Disconnect(context.Background())
			noerr(t, err)
			err = p.Disconnect(context.Background())
			if err != ErrPoolDisconnected {
				t.Errorf("Should not be able to call disconnect twice. got %v; want %v", err, ErrPoolDisconnected)
			}
		})
		t.Run("closes idle connections", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 3, func(nc net.Conn) {
				<-cleanup
				nc.Close()
			})
			d := newdialer(&net.Dialer{})
			p, err := NewPool(address.Address(addr.String()), 3, 3, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect(context.Background())
			noerr(t, err)
			conns := [3]Connection{}
			for idx := range [3]struct{}{} {
				conns[idx], _, err = p.Get(context.Background())
				noerr(t, err)
			}
			for idx := range [3]struct{}{} {
				err = conns[idx].Close()
				noerr(t, err)
			}
			if d.lenopened() != 3 {
				t.Errorf("Should have opened 3 connections, but didn't. got %d; want %d", d.lenopened(), 3)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			err = p.Disconnect(ctx)
			noerr(t, err)
			if d.lenclosed() != 3 {
				t.Errorf("Should have closed 3 connections, but didn't. got %d; want %d", d.lenclosed(), 3)
			}
			close(cleanup)
			ok := p.(*pool).sem.TryAcquire(int64(p.(*pool).capacity))
			if !ok {
				t.Errorf("clean shutdown should acquire and release semaphore, but semaphore still held")
			} else {
				p.(*pool).sem.Release(int64(p.(*pool).capacity))
			}
		})
		t.Run("closes inflight connections when context expires", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 3, func(nc net.Conn) {
				<-cleanup
				nc.Close()
			})
			d := newdialer(&net.Dialer{})
			p, err := NewPool(address.Address(addr.String()), 3, 3, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect(context.Background())
			noerr(t, err)
			conns := [3]Connection{}
			for idx := range [3]struct{}{} {
				conns[idx], _, err = p.Get(context.Background())
				noerr(t, err)
			}
			for idx := range [2]struct{}{} {
				err = conns[idx].Close()
				noerr(t, err)
			}
			if d.lenopened() != 3 {
				t.Errorf("Should have opened 3 connections, but didn't. got %d; want %d", d.lenopened(), 3)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			cancel()
			err = p.Disconnect(ctx)
			noerr(t, err)
			if d.lenclosed() != 3 {
				t.Errorf("Should have closed 3 connections, but didn't. got %d; want %d", d.lenclosed(), 3)
			}
			close(cleanup)
			err = conns[2].Close()
			noerr(t, err)
			ok := p.(*pool).sem.TryAcquire(int64(p.(*pool).capacity))
			if !ok {
				t.Errorf("clean shutdown should acquire and release semaphore, but semaphore still held")
			} else {
				p.(*pool).sem.Release(int64(p.(*pool).capacity))
			}
		})
		t.Run("properly sets the connection state on return", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 3, func(nc net.Conn) {
				<-cleanup
				nc.Close()
			})
			d := newdialer(&net.Dialer{})
			p, err := NewPool(address.Address(addr.String()), 3, 3, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect(context.Background())
			noerr(t, err)
			c, _, err := p.Get(context.Background())
			noerr(t, err)
			err = c.Close()
			noerr(t, err)
			if d.lenopened() != 1 {
				t.Errorf("Should have opened 3 connections, but didn't. got %d; want %d", d.lenopened(), 1)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			err = p.Disconnect(ctx)
			noerr(t, err)
			if d.lenclosed() != 1 {
				t.Errorf("Should have closed 3 connections, but didn't. got %d; want %d", d.lenclosed(), 1)
			}
			close(cleanup)
			state := atomic.LoadInt32(&(p.(*pool)).connected)
			if state != disconnected {
				t.Errorf("Should have set the connection state on return. got %d; want %d", state, disconnected)
			}
		})
	})
	t.Run("Connect", func(t *testing.T) {
		t.Run("can reconnect a disconnected pool", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 3, func(nc net.Conn) {
				<-cleanup
				nc.Close()
			})
			d := newdialer(&net.Dialer{})
			p, err := NewPool(address.Address(addr.String()), 3, 3, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect(context.Background())
			noerr(t, err)
			c, _, err := p.Get(context.Background())
			noerr(t, err)
			gen := c.(*acquired).Connection.(*pooledConnection).generation
			if gen != 1 {
				t.Errorf("Connection should have a newer generation. got %d; want %d", gen, 1)
			}
			err = c.Close()
			noerr(t, err)
			if d.lenopened() != 1 {
				t.Errorf("Should have opened 3 connections, but didn't. got %d; want %d", d.lenopened(), 1)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			err = p.Disconnect(ctx)
			noerr(t, err)
			if d.lenclosed() != 1 {
				t.Errorf("Should have closed 3 connections, but didn't. got %d; want %d", d.lenclosed(), 1)
			}
			close(cleanup)
			state := atomic.LoadInt32(&(p.(*pool)).connected)
			if state != disconnected {
				t.Errorf("Should have set the connection state on return. got %d; want %d", state, disconnected)
			}
			err = p.Connect(context.Background())
			noerr(t, err)

			c, _, err = p.Get(context.Background())
			noerr(t, err)
			gen = atomic.LoadUint64(&(c.(*acquired).Connection.(*pooledConnection)).generation)
			if gen != 2 {
				t.Errorf("Connection should have a newer generation. got %d; want %d", gen, 2)
			}
			err = c.Close()
			noerr(t, err)
			if d.lenopened() != 2 {
				t.Errorf("Should have opened 3 connections, but didn't. got %d; want %d", d.lenopened(), 2)
			}
		})
		t.Run("cannot connect multiple times without disconnect", func(t *testing.T) {
			p, err := NewPool(address.Address(""), 3, 3)
			noerr(t, err)
			err = p.Connect(context.Background())
			noerr(t, err)
			err = p.Connect(context.Background())
			if err != ErrPoolConnected {
				t.Errorf("Shouldn't be able to connect to already connected pool. got %v; want %v", err, ErrPoolConnected)
			}
			err = p.Connect(context.Background())
			if err != ErrPoolConnected {
				t.Errorf("Shouldn't be able to connect to already connected pool. got %v; want %v", err, ErrPoolConnected)
			}
			err = p.Disconnect(context.Background())
			noerr(t, err)
			err = p.Connect(context.Background())
			if err != nil {
				t.Errorf("Should be able to connect to pool after disconnect. got %v; want <nil>", err)
			}
		})
		t.Run("can disconnect and reconnect multiple times", func(t *testing.T) {
			p, err := NewPool(address.Address(""), 3, 3)
			noerr(t, err)
			err = p.Connect(context.Background())
			noerr(t, err)
			err = p.Disconnect(context.Background())
			noerr(t, err)
			err = p.Connect(context.Background())
			if err != nil {
				t.Errorf("Should be able to connect to disconnected pool. got %v; want <nil>", err)
			}
			err = p.Disconnect(context.Background())
			noerr(t, err)
			err = p.Connect(context.Background())
			if err != nil {
				t.Errorf("Should be able to connect to disconnected pool. got %v; want <nil>", err)
			}
			err = p.Disconnect(context.Background())
			noerr(t, err)
			err = p.Connect(context.Background())
			if err != nil {
				t.Errorf("Should be able to connect to pool after disconnect. got %v; want <nil>", err)
			}
		})
	})
	t.Run("Get", func(t *testing.T) {
		t.Run("return context error when already cancelled", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 3, func(nc net.Conn) {
				<-cleanup
				nc.Close()
			})
			d := newdialer(&net.Dialer{})
			p, err := NewPool(address.Address(addr.String()), 3, 3, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect(context.Background())
			noerr(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			cancel()
			_, _, err = p.Get(ctx)
			if err != context.Canceled {
				t.Errorf("Should return context error when already cancelled. got %v; want %v", err, context.Canceled)
			}
			close(cleanup)
		})
		t.Run("return context error when attempting to acquire semaphore", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 3, func(nc net.Conn) {
				<-cleanup
				nc.Close()
			})
			d := newdialer(&net.Dialer{})
			p, err := NewPool(address.Address(addr.String()), 3, 3, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect(context.Background())
			noerr(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			ok := p.(*pool).sem.TryAcquire(3)
			if !ok {
				t.Errorf("Could not acquire the entire semaphore.")
			}
			_, _, err = p.Get(ctx)
			if err != context.DeadlineExceeded {
				t.Errorf("Should return context error when already canclled. got %v; want %v", err, context.DeadlineExceeded)
			}
			close(cleanup)
		})
		t.Run("return error when attempting to create new connection", func(t *testing.T) {
			want := errors.New("create new connection error")
			var dialer DialerFunc = func(context.Context, string, string) (net.Conn, error) { return nil, want }
			p, err := NewPool(address.Address(""), 1, 2, WithDialer(func(Dialer) Dialer { return dialer }))
			noerr(t, err)
			err = p.Connect(context.Background())
			noerr(t, err)
			_, _, got := p.Get(context.Background())
			if got != want {
				t.Errorf("Should return error from calling New. got %v; want %v", got, want)
			}
		})
		t.Run("adds connection to inflight pool", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 1, func(nc net.Conn) {
				<-cleanup
				nc.Close()
			})
			d := newdialer(&net.Dialer{})
			p, err := NewPool(address.Address(addr.String()), 3, 3, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect(context.Background())
			noerr(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			c, _, err := p.Get(ctx)
			noerr(t, err)
			inflight := len(p.(*pool).inflight)
			if inflight != 1 {
				t.Errorf("Incorrect number of inlight connections. got %d; want %d", inflight, 1)
			}
			err = c.Close()
			noerr(t, err)
			close(cleanup)
		})
		t.Run("closes expired connections", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 2, func(nc net.Conn) {
				<-cleanup
				nc.Close()
			})
			d := newdialer(&net.Dialer{})
			p, err := NewPool(
				address.Address(addr.String()), 3, 3,
				WithDialer(func(Dialer) Dialer { return d }),
				WithIdleTimeout(func(time.Duration) time.Duration { return 10 * time.Millisecond }),
			)
			noerr(t, err)
			err = p.Connect(context.Background())
			noerr(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			c, _, err := p.Get(ctx)
			noerr(t, err)
			if d.lenopened() != 1 {
				t.Errorf("Should have opened 1 connection, but didn't. got %d; want %d", d.lenopened(), 1)
			}
			err = c.Close()
			noerr(t, err)
			time.Sleep(15 * time.Millisecond)
			if d.lenclosed() != 0 {
				t.Errorf("Should have closed 0 connections, but didn't. got %d; want %d", d.lenopened(), 0)
			}
			c, _, err = p.Get(ctx)
			noerr(t, err)
			if d.lenopened() != 2 {
				t.Errorf("Should have opened 2 connections, but didn't. got %d; want %d", d.lenopened(), 2)
			}
			time.Sleep(10 * time.Millisecond)
			if d.lenclosed() != 1 {
				t.Errorf("Should have closed 1 connection, but didn't. got %d; want %d", d.lenopened(), 1)
			}
			close(cleanup)
		})
		t.Run("recycles connections", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 3, func(nc net.Conn) {
				<-cleanup
				nc.Close()
			})
			d := newdialer(&net.Dialer{})
			p, err := NewPool(address.Address(addr.String()), 3, 3, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect(context.Background())
			noerr(t, err)
			for range [3]struct{}{} {
				c, _, err := p.Get(context.Background())
				noerr(t, err)
				err = c.Close()
				noerr(t, err)
				if d.lenopened() != 1 {
					t.Errorf("Should have opened 1 connection, but didn't. got %d; want %d", d.lenopened(), 1)
				}
			}
			close(cleanup)
		})
		t.Run("cannot get from disconnected pool", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 3, func(nc net.Conn) {
				<-cleanup
				nc.Close()
			})
			d := newdialer(&net.Dialer{})
			p, err := NewPool(address.Address(addr.String()), 3, 3, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect(context.Background())
			noerr(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Microsecond)
			defer cancel()
			err = p.Disconnect(ctx)
			noerr(t, err)
			_, _, err = p.Get(context.Background())
			if err != ErrPoolClosed {
				t.Errorf("Should get error from disconnected pool. got %v; want %v", err, ErrPoolClosed)
			}
			close(cleanup)
		})
		t.Run("pool closes excess connections when returned", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 3, func(nc net.Conn) {
				<-cleanup
				nc.Close()
			})
			d := newdialer(&net.Dialer{})
			p, err := NewPool(address.Address(addr.String()), 1, 3, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect(context.Background())
			noerr(t, err)
			conns := [3]Connection{}
			for idx := range [3]struct{}{} {
				conns[idx], _, err = p.Get(context.Background())
				noerr(t, err)
			}
			for idx := range [3]struct{}{} {
				err = conns[idx].Close()
				noerr(t, err)
			}
			if d.lenopened() != 3 {
				t.Errorf("Should have opened 3 connections, but didn't. got %d; want %d", d.lenopened(), 3)
			}
			if d.lenclosed() != 2 {
				t.Errorf("Should have closed 2 connections, but didn't. got %d; want %d", d.lenopened(), 2)
			}
			close(cleanup)
		})
		t.Run("cannot get more than capacity connections", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 3, func(nc net.Conn) {
				<-cleanup
				nc.Close()
			})
			d := newdialer(&net.Dialer{})
			p, err := NewPool(address.Address(addr.String()), 1, 2, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect(context.Background())
			noerr(t, err)
			conns := [2]Connection{}
			for idx := range [2]struct{}{} {
				conns[idx], _, err = p.Get(context.Background())
				noerr(t, err)
			}
			if d.lenopened() != 2 {
				t.Errorf("Should have opened 2 connections, but didn't. got %d; want %d", d.lenopened(), 2)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()
			_, _, err = p.Get(ctx)
			if err != context.DeadlineExceeded {
				t.Errorf("Should not be able to get more than capacity connections. got %v; want %v", err, context.DeadlineExceeded)
			}
			err = conns[0].Close()
			noerr(t, err)
			ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()
			c, _, err := p.Get(ctx)
			noerr(t, err)
			err = c.Close()
			noerr(t, err)

			err = p.Drain()
			noerr(t, err)

			err = conns[1].Close()
			noerr(t, err)

			ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()
			c, _, err = p.Get(ctx)
			noerr(t, err)
			if d.lenopened() != 3 {
				t.Errorf("Should have opened 3 connections, but didn't. got %d; want %d", d.lenopened(), 3)
			}
			close(cleanup)
		})
		t.Run("Cannot starve connection request", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 3, func(nc net.Conn) {
				<-cleanup
				nc.Close()
			})
			d := newdialer(&net.Dialer{})
			p, err := NewPool(address.Address(addr.String()), 1, 1, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect(context.Background())
			noerr(t, err)
			conn, _, err := p.Get(context.Background())
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
				_, _, err := p.Get(ctx)
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
		t.Run("Does not leak permit from failure to dial connection", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 0, func(nc net.Conn) {
				<-cleanup
				nc.Close()
			})
			close(cleanup)
			want := errors.New("dialing error")
			p, err := NewPool(
				address.Address(addr.String()), 1, 2,
				WithDialer(
					func(Dialer) Dialer {
						return DialerFunc(func(ctx context.Context, network, address string) (net.Conn, error) {
							return nil, want
						})
					}),
			)
			noerr(t, err)
			err = p.Connect(context.Background())
			noerr(t, err)
			_, _, err = p.Get(context.Background())
			if err != want {
				t.Errorf("Expected dial failure but got: %v", err)
			}
			ok := p.(*pool).sem.TryAcquire(int64(p.(*pool).capacity))
			if !ok {
				t.Errorf("Dial failure should not leak semaphore permit")
			} else {
				p.(*pool).sem.Release(int64(p.(*pool).capacity))
			}
		})
		t.Run("Does not leak permit from cancelled context", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 1, func(nc net.Conn) {
				<-cleanup
				nc.Close()
			})
			close(cleanup)
			d := newdialer(&net.Dialer{})
			p, err := NewPool(address.Address(addr.String()), 1, 2, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect(context.Background())
			noerr(t, err)
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_, _, err = p.Get(ctx)
			if err != context.Canceled {
				t.Errorf("Expected context cancelled error. got %v; want %v", err, context.Canceled)
			}
			ok := p.(*pool).sem.TryAcquire(int64(p.(*pool).capacity))
			if !ok {
				t.Errorf("Canceled context should not leak semaphore permit")
			} else {
				p.(*pool).sem.Release(int64(p.(*pool).capacity))
			}
		})
		t.Run("Get does not acquire multiple permits", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 2, func(nc net.Conn) {
				<-cleanup
				nc.Close()
			})
			close(cleanup)
			d := newdialer(&net.Dialer{})
			p, err := NewPool(address.Address(addr.String()), 1, 2, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect(context.Background())
			noerr(t, err)
			c, _, err := p.Get(context.Background())
			noerr(t, err)
			err = c.Close()
			noerr(t, err)

			err = p.Drain()
			noerr(t, err)

			c, _, err = p.Get(context.Background())
			noerr(t, err)
			err = c.Close()
			noerr(t, err)
			ok := p.(*pool).sem.TryAcquire(int64(p.(*pool).capacity))
			if !ok {
				t.Errorf("Get should not acquire multiple permits (when expired conn in idle pool)")
			} else {
				p.(*pool).sem.Release(int64(p.(*pool).capacity))
			}
		})
	})
	t.Run("Connection", func(t *testing.T) {
		t.Run("Connection Close Does Not Error After Pool Is Disconnected", func(t *testing.T) {
			cleanup := make(chan struct{})
			defer close(cleanup)
			addr := bootstrapConnections(t, 3, func(nc net.Conn) {
				<-cleanup
				nc.Close()
			})
			d := newdialer(&net.Dialer{})
			p, err := NewPool(address.Address(addr.String()), 2, 4, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect(context.Background())
			noerr(t, err)
			c1, _, err := p.Get(context.Background())
			noerr(t, err)
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			err = p.Disconnect(ctx)
			noerr(t, err)
			err = c1.Close()
			if err != nil {
				t.Errorf("Connection Close should not error after Pool is Disconnected, but got error: %v", err)
			}
		})
		t.Run("Does not return to pool twice", func(t *testing.T) {
			cleanup := make(chan struct{})
			defer close(cleanup)
			addr := bootstrapConnections(t, 1, func(nc net.Conn) {
				<-cleanup
				nc.Close()
			})
			d := newdialer(&net.Dialer{})
			P, err := NewPool(address.Address(addr.String()), 2, 4, WithDialer(func(Dialer) Dialer { return d }))
			p := P.(*pool)
			noerr(t, err)
			err = p.Connect(context.Background())
			noerr(t, err)
			c1, _, err := p.Get(context.Background())
			noerr(t, err)
			if len(p.conns) != 0 {
				t.Errorf("Should be no connections in pool. got %d; want %d", len(p.conns), 0)
			}
			err = c1.Close()
			noerr(t, err)
			err = c1.Close()
			noerr(t, err)
			if len(p.conns) != 1 {
				t.Errorf("Should not return connection to pool twice. got %d; want %d", len(p.conns), 1)
			}
		})
	})
}
