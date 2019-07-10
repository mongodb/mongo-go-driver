package topology

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/x/mongo/driver/address"
	"net"
	"runtime"

	//"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPool(t *testing.T) {
	t.Run("newPool", func(t *testing.T) {
		t.Run("should be connected", func(t *testing.T) {
			pc := poolConfig{
				Address:     address.Address(""),
				MaxPoolSize: 2,
			}
			p, err := newPool(pc)
			noerr(t, err)
			err = p.Connect()
			noerr(t, err)
			if p.connected != connected {
				t.Errorf("Expected new pool to be connected. got %v; want %v", p.connected, connected)
			}
		})
	})
	t.Run("closeConnection", func(t *testing.T) {
		t.Run("can't CheckIn connection from different pool", func(t *testing.T) {
			pc1 := poolConfig{
				Address:     address.Address(""),
				MaxPoolSize: 2,
			}
			p1, err := newPool(pc1)
			noerr(t, err)
			err = p1.Connect()
			noerr(t, err)

			pc2 := poolConfig{
				Address:     address.Address(""),
				MaxPoolSize: 2,
			}
			p2, err := newPool(pc2)
			noerr(t, err)
			err = p2.Connect()
			noerr(t, err)

			c1 := &connection{pool: p1}
			want := ErrWrongPool
			got := p2.closeConnection(c1)
			if got != want {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
	})
	t.Run("Disconnect", func(t *testing.T) {
		t.Run("cannot close twice", func(t *testing.T) {
			pc := poolConfig{
				Address:     address.Address(""),
				MaxPoolSize: 2,
			}
			p, err := newPool(pc)
			noerr(t, err)
			err = p.Connect()
			noerr(t, err)
			err = p.Disconnect(context.Background())
			noerr(t, err)
			err = p.Disconnect(context.Background())
			if err != ErrPoolDisconnected {
				t.Errorf("Should not be able to call Disconnect twice. got %v; want %v", err, ErrPoolDisconnected)
			}
		})
		t.Run("closes idle connections", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 3, func(nc net.Conn) {
				<-cleanup
				_ = nc.Close()
			})
			d := newdialer(&net.Dialer{})
			pc := poolConfig{
				Address:     address.Address(addr.String()),
				MaxPoolSize: 3,
				MaxIdleTime: 100 * time.Millisecond,
			}
			p, err := newPool(pc, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)

			err = p.Connect()
			noerr(t, err)
			conns := [3]*connection{}
			for idx := range [3]struct{}{} {
				conns[idx], err = p.CheckOut(context.Background())
				noerr(t, err)
			}
			for idx := range [3]struct{}{} {
				err = p.CheckIn(conns[idx])
				noerr(t, err)
			}
			if d.lenopened() != 3 {
				t.Errorf("Should have opened 3 connections, but didn't. got %d; want %d", d.lenopened(), 3)
			}
			err = p.Disconnect(context.Background())
			time.Sleep(time.Second)

			noerr(t, err)
			if d.lenclosed() != 3 {
				t.Errorf("Should have closed 3 connections, but didn't. got %d; want %d", d.lenclosed(), 3)
			}
			close(cleanup)
		})
		t.Run("closes all connections currently in pool and closes all remaining connections", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 3, func(nc net.Conn) {
				<-cleanup
				_ = nc.Close()
			})
			d := newdialer(&net.Dialer{})
			pc := poolConfig{
				Address:     address.Address(addr.String()),
				MaxPoolSize: 3,
			}
			p, err := newPool(pc, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect()
			noerr(t, err)
			conns := [3]*connection{}
			for idx := range [3]struct{}{} {
				conns[idx], err = p.CheckOut(context.Background())
				noerr(t, err)
			}
			for idx := range [2]struct{}{} {
				err = p.CheckIn(conns[idx])
				noerr(t, err)
			}
			if d.lenopened() != 3 {
				t.Errorf("Should have opened 3 connections, but didn't. got %d; want %d", d.lenopened(), 3)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Microsecond)
			defer cancel()
			err = p.Disconnect(ctx)
			noerr(t, err)
			if d.lenclosed() != 3 {
				t.Errorf("Should have closed 3 connections, but didn't. got %d; want %d", d.lenclosed(), 3)
			}
			close(cleanup)
		})
		t.Run("properly sets the connection state on return", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 3, func(nc net.Conn) {
				<-cleanup
				_ = nc.Close()
			})
			d := newdialer(&net.Dialer{})
			pc := poolConfig{
				Address:     address.Address(addr.String()),
				MaxPoolSize: 3,
				MinPoolSize: 0,
			}
			p, err := newPool(pc, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect()
			noerr(t, err)
			c, err := p.CheckOut(context.Background())
			noerr(t, err)
			err = p.closeConnection(c)
			noerr(t, err)
			if d.lenopened() != 1 {
				t.Errorf("Should have opened 1 connections, but didn't. got %d; want %d", d.lenopened(), 1)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Microsecond)
			defer cancel()
			err = p.Disconnect(ctx)
			noerr(t, err)
			if d.lenclosed() != 1 {
				t.Errorf("Should have closed 1 connections, but didn't. got %d; want %d", d.lenclosed(), 1)
			}
			close(cleanup)
			state := atomic.LoadInt32(&p.connected)
			if state != disconnected {
				t.Errorf("Should have set the connection state on return. got %d; want %d", state, disconnected)
			}
		})
	})
	t.Run("Close", func(t *testing.T) {
		t.Run("cannot close twice", func(t *testing.T) {
			pc := poolConfig{
				Address:     address.Address(""),
				MaxPoolSize: 2,
			}
			p, err := newPool(pc)
			noerr(t, err)
			err = p.Connect()
			noerr(t, err)
			err = p.Close(context.Background())
			noerr(t, err)
			err = p.Close(context.Background())
			if err != ErrPoolDisconnected {
				t.Errorf("Should not be able to call close twice. got %v; want %v", err, ErrPoolDisconnected)
			}
		})
		t.Run("closes idle connections", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 3, func(nc net.Conn) {
				<-cleanup
				_ = nc.Close()
			})
			d := newdialer(&net.Dialer{})
			pc := poolConfig{
				Address:     address.Address(addr.String()),
				MaxPoolSize: 3,
				MaxIdleTime: 100 * time.Millisecond,
			}
			p, err := newPool(pc, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)

			err = p.Connect()
			noerr(t, err)
			conns := [3]*connection{}
			for idx := range [3]struct{}{} {
				conns[idx], err = p.CheckOut(context.Background())
				noerr(t, err)
			}
			for idx := range [3]struct{}{} {
				err = p.CheckIn(conns[idx])
				noerr(t, err)
			}
			if d.lenopened() != 3 {
				t.Errorf("Should have opened 3 connections, but didn't. got %d; want %d", d.lenopened(), 3)
			}
			err = p.Close(context.Background())
			time.Sleep(time.Second)

			noerr(t, err)
			if d.lenclosed() != 3 {
				t.Errorf("Should have closed 3 connections, but didn't. got %d; want %d", d.lenclosed(), 3)
			}
			close(cleanup)
		})
		t.Run("closes all connections currently in pool and closes all remaining connections when checked in", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 3, func(nc net.Conn) {
				<-cleanup
				_ = nc.Close()
			})
			d := newdialer(&net.Dialer{})
			pc := poolConfig{
				Address:     address.Address(addr.String()),
				MaxPoolSize: 3,
			}
			p, err := newPool(pc, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect()
			noerr(t, err)
			conns := [3]*connection{}
			for idx := range [3]struct{}{} {
				conns[idx], err = p.CheckOut(context.Background())
				noerr(t, err)
			}
			for idx := range [2]struct{}{} {
				err = p.CheckIn(conns[idx])
				noerr(t, err)
			}
			if d.lenopened() != 3 {
				t.Errorf("Should have opened 3 connections, but didn't. got %d; want %d", d.lenopened(), 3)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Microsecond)
			cancel()
			err = p.Close(ctx)
			noerr(t, err)
			time.Sleep(time.Second)
			if d.lenclosed() != 2 {
				t.Errorf("Should have closed 2 connections, but didn't. got %d; want %d", d.lenclosed(), 2)
			}
			err = p.CheckIn(conns[2])
			noerr(t, err)
			if d.lenclosed() != 3 {
				t.Errorf("Should have closed 3 connections, but didn't. got %d; want %d", d.lenclosed(), 3)
			}

			close(cleanup)
		})
		t.Run("properly sets the connection state on return", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 3, func(nc net.Conn) {
				<-cleanup
				_ = nc.Close()
			})
			d := newdialer(&net.Dialer{})
			pc := poolConfig{
				Address:     address.Address(addr.String()),
				MaxPoolSize: 3,
				MinPoolSize: 0,
			}
			p, err := newPool(pc, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect()
			noerr(t, err)
			c, err := p.CheckOut(context.Background())
			noerr(t, err)
			err = p.closeConnection(c)
			noerr(t, err)
			if d.lenopened() != 1 {
				t.Errorf("Should have opened 1 connections, but didn't. got %d; want %d", d.lenopened(), 1)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Microsecond)
			defer cancel()
			err = p.Close(ctx)
			noerr(t, err)
			if d.lenclosed() != 1 {
				t.Errorf("Should have closed 1 connections, but didn't. got %d; want %d", d.lenclosed(), 1)
			}
			close(cleanup)
			state := atomic.LoadInt32(&p.connected)
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
				_ = nc.Close()
			})
			d := newdialer(&net.Dialer{})
			pc := poolConfig{
				Address:     address.Address(addr.String()),
				MaxPoolSize: 3,
			}
			p, err := newPool(pc, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect()
			noerr(t, err)
			c, err := p.CheckOut(context.Background())
			noerr(t, err)
			gen := c.generation
			if gen != 0 {
				t.Errorf("Connection should have a newer generation. got %d; want %d", gen, 0)
			}
			err = p.CheckIn(c)
			noerr(t, err)
			if d.lenopened() != 1 {
				t.Errorf("Should have opened 1 connections, but didn't. got %d; want %d", d.lenopened(), 1)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			err = p.Close(ctx)
			noerr(t, err)
			if d.lenclosed() != 1 {
				t.Errorf("Should have closed 1 connections, but didn't. got %d; want %d", d.lenclosed(), 1)
			}
			close(cleanup)
			state := atomic.LoadInt32(&p.connected)
			if state != disconnected {
				t.Errorf("Should have set the connection state on return. got %d; want %d", state, disconnected)
			}
			err = p.Connect()
			noerr(t, err)

			c, err = p.CheckOut(context.Background())
			noerr(t, err)
			gen = atomic.LoadUint64(&c.generation)
			if gen != 1 {
				t.Errorf("Connection should have a newer generation. got %d; want %d", gen, 1)
			}
			err = p.CheckIn(c)
			noerr(t, err)
			if d.lenopened() != 2 {
				t.Errorf("Should have opened 3 connections, but didn't. got %d; want %d", d.lenopened(), 2)
			}
		})
		t.Run("cannot Connect multiple times without Close", func(t *testing.T) {
			pc := poolConfig{
				Address:     "",
				MaxPoolSize: 3,
			}
			p, err := newPool(pc)
			noerr(t, err)
			err = p.Connect()
			noerr(t, err)
			err = p.Connect()
			if err != ErrPoolConnected {
				t.Errorf("Shouldn't be able to Connect to already connected pool. got %v; want %v", err, ErrPoolConnected)
			}
			err = p.Connect()
			if err != ErrPoolConnected {
				t.Errorf("Shouldn't be able to Connect to already connected pool. got %v; want %v", err, ErrPoolConnected)
			}
			err = p.Close(context.Background())
			noerr(t, err)
			err = p.Connect()
			if err != nil {
				t.Errorf("Should be able to Connect to pool after Close. got %v; want <nil>", err)
			}
		})
		t.Run("can Close and reconnect multiple times", func(t *testing.T) {
			pc := poolConfig{
				Address:     address.Address(""),
				MaxPoolSize: 3,
			}
			p, err := newPool(pc)
			noerr(t, err)
			err = p.Connect()
			noerr(t, err)
			err = p.Close(context.Background())
			noerr(t, err)
			err = p.Connect()
			if err != nil {
				t.Errorf("Should be able to Connect to disconnected pool. got %v; want <nil>", err)
			}
			err = p.Close(context.Background())
			noerr(t, err)
			err = p.Connect()
			if err != nil {
				t.Errorf("Should be able to Connect to disconnected pool. got %v; want <nil>", err)
			}
			err = p.Close(context.Background())
			noerr(t, err)
			err = p.Connect()
			if err != nil {
				t.Errorf("Should be able to Connect to pool after Close. got %v; want <nil>", err)
			}
		})
	})
	t.Run("Get", func(t *testing.T) {
		t.Run("return context error when already cancelled", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 3, func(nc net.Conn) {
				<-cleanup
				_ = nc.Close()
			})
			d := newdialer(&net.Dialer{})
			pc := poolConfig{
				Address:     address.Address(addr.String()),
				MaxPoolSize: 3,
			}
			p, err := newPool(pc, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect()
			noerr(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			cancel()
			_, err = p.CheckOut(ctx)
			if err != context.Canceled {
				t.Errorf("Should return context error when already cancelled. got %v; want %v", err, context.Canceled)
			}
			close(cleanup)
		})
		t.Run("return error when attempting to create new connection", func(t *testing.T) {
			wanterr := errors.New("create new connection error")
			var want error = ConnectionError{Wrapped: wanterr, init: true}
			var dialer DialerFunc = func(context.Context, string, string) (net.Conn, error) { return nil, wanterr }
			pc := poolConfig{
				Address:     address.Address(""),
				MaxPoolSize: 2,
			}
			p, err := newPool(pc, WithDialer(func(Dialer) Dialer { return dialer }))
			noerr(t, err)
			err = p.Connect()
			noerr(t, err)
			_, got := p.CheckOut(context.Background())
			if got != want {
				t.Errorf("Should return error from calling New. got %v; want %v", got, want)
			}
		})
		t.Run("adds connection to inflight pool", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 1, func(nc net.Conn) {
				<-cleanup
				_ = nc.Close()
			})
			d := newdialer(&net.Dialer{})
			pc := poolConfig{
				Address:     address.Address(addr.String()),
				MaxPoolSize: 3,
			}
			p, err := newPool(pc, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect()
			noerr(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			c, err := p.CheckOut(ctx)
			noerr(t, err)
			inflight := len(p.opened)
			if inflight != 1 {
				t.Errorf("Incorrect number of inlight connections. got %d; want %d", inflight, 1)
			}
			err = p.closeConnection(c)
			noerr(t, err)
			close(cleanup)
		})
		t.Run("closes stale connections", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 2, func(nc net.Conn) {
				<-cleanup
				_ = nc.Close()
			})
			d := newdialer(&net.Dialer{})
			pc := poolConfig{
				Address:     address.Address(addr.String()),
				MaxPoolSize: 3,
			}
			p, err := newPool(
				pc,
				WithDialer(func(Dialer) Dialer { return d }),
				WithIdleTimeout(func(time.Duration) time.Duration { return 10 * time.Millisecond }),
			)
			noerr(t, err)
			err = p.Connect()
			noerr(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			c, err := p.CheckOut(ctx)
			noerr(t, err)
			if d.lenopened() != 1 {
				t.Errorf("Should have opened 1 connection, but didn't. got %d; want %d", d.lenopened(), 1)
			}
			time.Sleep(15 * time.Millisecond)
			err = p.CheckIn(c)
			noerr(t, err)
			if d.lenclosed() != 1 {
				t.Errorf("Should have closed 1 connections, but didn't. got %d; want %d", d.lenclosed(), 1)
			}
			c, err = p.CheckOut(ctx)
			noerr(t, err)
			if d.lenopened() != 2 {
				t.Errorf("Should have opened 2 connections, but didn't. got %d; want %d", d.lenopened(), 2)
			}
			time.Sleep(10 * time.Millisecond)
			if d.lenclosed() != 1 {
				t.Errorf("Should have closed 1 connection, but didn't. got %d; want %d", d.lenclosed(), 1)
			}
			close(cleanup)
		})
		t.Run("recycles connections", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 3, func(nc net.Conn) {
				<-cleanup
				_ = nc.Close()
			})
			d := newdialer(&net.Dialer{})
			pc := poolConfig{
				Address:     address.Address(addr.String()),
				MaxPoolSize: 3,
			}
			p, err := newPool(pc, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect()
			noerr(t, err)
			for range [3]struct{}{} {
				c, err := p.CheckOut(context.Background())
				noerr(t, err)
				err = p.CheckIn(c)
				noerr(t, err)
				if d.lenopened() != 1 {
					t.Errorf("Should have opened 1 connection, but didn't. got %d; want %d", d.lenopened(), 1)
				}
			}
			close(cleanup)
		})
		t.Run("cannot CheckOut from disconnected pool", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 3, func(nc net.Conn) {
				<-cleanup
				_ = nc.Close()
			})
			d := newdialer(&net.Dialer{})
			pc := poolConfig{
				Address:     address.Address(addr.String()),
				MaxPoolSize: 3,
			}
			p, err := newPool(pc, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect()
			noerr(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Microsecond)
			defer cancel()
			err = p.Close(ctx)
			noerr(t, err)
			_, err = p.CheckOut(context.Background())
			if err != ErrPoolDisconnected {
				t.Errorf("Should CheckOut error from disconnected pool. got %v; want %v", err, ErrPoolDisconnected)
			}
			close(cleanup)
		})
		t.Run("pool closes excess connections when returned", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 3, func(nc net.Conn) {
				<-cleanup
				_ = nc.Close()
			})
			d := newdialer(&net.Dialer{})
			pc := poolConfig{
				Address:     address.Address(addr.String()),
				MaxPoolSize: 3,
			}
			p, err := newPool(pc, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect()
			noerr(t, err)
			conns := [3]*connection{}
			for idx := range [3]struct{}{} {
				conns[idx], err = p.CheckOut(context.Background())
				noerr(t, err)
			}
			err = p.Close(context.Background())
			noerr(t, err)
			for idx := range [3]struct{}{} {
				err = p.CheckIn(conns[idx])
				noerr(t, err)
			}
			if d.lenopened() != 3 {
				t.Errorf("Should have opened 3 connections, but didn't. got %d; want %d", d.lenopened(), 3)
			}
			if d.lenclosed() != 3 {
				t.Errorf("Should have closed 3 connections, but didn't. got %d; want %d", d.lenclosed(), 3)
			}
			close(cleanup)
		})
		t.Run("Cannot starve connection request", func(t *testing.T) {
			cleanup := make(chan struct{})
			addr := bootstrapConnections(t, 3, func(nc net.Conn) {
				<-cleanup
				_ = nc.Close()
			})
			d := newdialer(&net.Dialer{})
			pc := poolConfig{
				Address:     address.Address(addr.String()),
				MaxPoolSize: 1,
			}
			p, err := newPool(pc, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect()
			noerr(t, err)
			conn, err := p.CheckOut(context.Background())
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
				_, err := p.CheckOut(ctx)
				if err != nil {
					t.Errorf("Should not be able to starve connection request, but got error: %v", err)
				}
				wg.Done()
			}()
			<-ch
			runtime.Gosched()
			err = p.CheckIn(conn)
			noerr(t, err)
			wg.Wait()
			close(cleanup)
		})
	})
	t.Run("Connection", func(t *testing.T) {
		t.Run("Connection Close Does Not Error After Pool Is Disconnected", func(t *testing.T) {
			cleanup := make(chan struct{})
			defer close(cleanup)
			addr := bootstrapConnections(t, 3, func(nc net.Conn) {
				<-cleanup
				_ = nc.Close()
			})
			d := newdialer(&net.Dialer{})
			pc := poolConfig{
				Address:     address.Address(addr.String()),
				MaxPoolSize: 4,
			}
			p, err := newPool(pc, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect()
			noerr(t, err)
			c, err := p.CheckOut(context.Background())
			noerr(t, err)
			c1 := &Connection{connection: c}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			err = p.Close(ctx)
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
				_ = nc.Close()
			})
			d := newdialer(&net.Dialer{})
			pc := poolConfig{
				Address:     address.Address(addr.String()),
				MaxPoolSize: 4,
			}
			p, err := newPool(pc, WithDialer(func(Dialer) Dialer { return d }))
			noerr(t, err)
			err = p.Connect()
			noerr(t, err)
			c, err := p.CheckOut(context.Background())
			c1 := &Connection{connection: c}
			noerr(t, err)
			if p.conns.size != 0 {
				t.Errorf("Should be no connections in pool. got %d; want %d", p.conns.size, 0)
			}
			err = c1.Close()
			noerr(t, err)
			err = c1.Close()
			noerr(t, err)
			if p.conns.size != 1 {
				t.Errorf("Should not return connection to pool twice. got %d; want %d", p.conns.size, 1)
			}
		})
	})
}
