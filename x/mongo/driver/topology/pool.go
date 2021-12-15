// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo/address"
)

// ErrPoolConnected is returned from an attempt to connect an already connected pool
var ErrPoolConnected = PoolError("attempted to Connect to an already connected pool")

// ErrPoolDisconnected is returned from an attempt to Close an already disconnected
// or disconnecting pool.
var ErrPoolDisconnected = PoolError("attempted to check out a connection from closed connection pool")

// ErrConnectionClosed is returned from an attempt to use an already closed connection.
var ErrConnectionClosed = ConnectionError{ConnectionID: "<closed>", message: "connection is closed"}

// ErrWrongPool is return when a connection is returned to a pool it doesn't belong to.
var ErrWrongPool = PoolError("connection does not belong to this pool")

// PoolError is an error returned from a Pool method.
type PoolError string

func (pe PoolError) Error() string { return string(pe) }

// poolConfig contains all aspects of the pool that can be configured
type poolConfig struct {
	Address       address.Address
	MinPoolSize   uint64
	MaxPoolSize   uint64
	MaxConnecting uint64
	MaxIdleTime   time.Duration
	PoolMonitor   *event.PoolMonitor
}

type pool struct {
	// The following integer fields must be accessed using the atomic package
	// and should be at the beginning of the struct.
	// - atomic bug: https://pkg.go.dev/sync/atomic#pkg-note-BUG
	// - suggested layout: https://go101.org/article/memory-layout.html

	connected                    int64  // connected is the connected state of the connection pool.
	nextID                       uint64 // nextID is the next pool ID for a new connection.
	pinnedCursorConnections      uint64
	pinnedTransactionConnections uint64

	address       address.Address
	minSize       uint64
	maxSize       uint64
	maxConnecting uint64
	monitor       *event.PoolMonitor

	connOpts   []ConnectionOption
	generation *poolGenerationMap

	maintainInterval time.Duration      // maintainInterval is the maintain() loop interval.
	cancelBackground context.CancelFunc // cancelBackground is called to signal background goroutines to stop.
	backgroundDone   *sync.WaitGroup    // backgroundDone waits for all background goroutines to return.

	connsCond   *sync.Cond             // connsCond guards conns, newConnWait.
	conns       map[uint64]*connection // conns holds all currently open connections.
	newConnWait wantConnQueue          // newConnWait holds all wantConn requests for new connections.

	idleMu       sync.Mutex    // idleMu guards idleConns, idleConnWait
	idleConns    []*connection // idleConns holds all idle connections.
	idleConnWait wantConnQueue // idleConnWait holds all wantConn requests for idle connections.
}

// connectionPerished checks if a given connection is perished and should be removed from the pool.
func connectionPerished(conn *connection) (string, bool) {
	switch {
	case atomic.LoadInt64(&conn.pool.connected) != connected:
		return event.ReasonPoolClosed, true
	case conn.closed():
		// A connection would only be closed if it encountered a network error during an operation and closed itself.
		return event.ReasonConnectionErrored, true
	case conn.idleTimeoutExpired():
		return event.ReasonIdle, true
	case conn.pool.stale(conn):
		return event.ReasonStale, true
	}
	return "", false
}

// newPool creates a new pool. It will use the provided options when creating connections.
func newPool(config poolConfig, connOpts ...ConnectionOption) *pool {
	if config.MaxIdleTime != time.Duration(0) {
		connOpts = append(connOpts, WithIdleTimeout(func(_ time.Duration) time.Duration { return config.MaxIdleTime }))
	}

	var maxConnecting uint64 = 2
	if config.MaxConnecting > 0 {
		maxConnecting = config.MaxConnecting
	}

	pool := &pool{
		address:          config.Address,
		minSize:          config.MinPoolSize,
		maxSize:          config.MaxPoolSize,
		maxConnecting:    maxConnecting,
		monitor:          config.PoolMonitor,
		connOpts:         connOpts,
		generation:       newPoolGenerationMap(),
		connected:        disconnected,
		maintainInterval: 10 * time.Second,
		connsCond:        sync.NewCond(&sync.Mutex{}),
		conns:            make(map[uint64]*connection, config.MaxPoolSize),
		idleConns:        make([]*connection, 0, config.MaxPoolSize),
	}
	pool.connOpts = append(pool.connOpts, withGenerationNumberFn(func(_ generationNumberFn) generationNumberFn { return pool.getGenerationForNewConnection }))

	if pool.monitor != nil {
		pool.monitor.Event(&event.PoolEvent{
			Type: event.PoolCreated,
			PoolOptions: &event.MonitorPoolOptions{
				MaxPoolSize: config.MaxPoolSize,
				MinPoolSize: config.MinPoolSize,
			},
			Address: pool.address.String(),
		})
	}

	return pool
}

// stale checks if a given connection's generation is below the generation of the pool
func (p *pool) stale(conn *connection) bool {
	return conn == nil || p.generation.stale(conn.desc.ServiceID, conn.generation)
}

// connect puts the pool into the connected state and starts the background connection creation and
// monitoring goroutines. connect must be called before connections can be checked out. An unused,
// connected pool must be disconnected or it will leak goroutines and will not be garbage collected.
func (p *pool) connect() error {
	if !atomic.CompareAndSwapInt64(&p.connected, disconnected, connecting) {
		return ErrPoolConnected
	}
	p.generation.connect()

	// Create a Context with cancellation that's used to signal the createConnections() and
	// maintain() background goroutines to stop. Also create a "backgroundDone" WaitGroup that is
	// used to wait for the background goroutines to return. Always create a new Context and
	// WaitGroup each time we start new set of background goroutines to prevent interaction between
	// current and previous sets of background goroutines.
	var ctx context.Context
	ctx, p.cancelBackground = context.WithCancel(context.Background())
	p.backgroundDone = &sync.WaitGroup{}

	for i := 0; i < int(p.maxConnecting); i++ {
		p.backgroundDone.Add(1)
		go p.createConnections(ctx, p.backgroundDone)
	}
	p.backgroundDone.Add(1)
	go p.maintain(ctx, p.backgroundDone)

	atomic.StoreInt64(&p.connected, connected)
	return nil
}

// disconnect disconnects the pool, closes all connections associated with the pool, and stops all
// background goroutines. All subsequent checkOut requests will return an error. An unused,
// connected pool must be disconnected or it will leak goroutines and will not be garbage collected.
func (p *pool) disconnect(ctx context.Context) error {
	if !atomic.CompareAndSwapInt64(&p.connected, connected, disconnecting) {
		return ErrPoolDisconnected
	}

	// Call cancelBackground() to exit the maintain() background goroutine and broadcast to the
	// connsCond to wake up all createConnections() goroutines. We must hold the connsCond lock here
	// because we're changing the condition by cancelling the "background goroutine" Context, even
	// tho cancelling the Context is also synchronized by a lock. Otherwise, we run into an
	// intermittent bug that prevents the createConnections() goroutines from exiting.
	p.connsCond.L.Lock()
	p.cancelBackground()
	p.connsCond.L.Unlock()
	p.connsCond.Broadcast()
	// Wait for all background goroutines to exit.
	p.backgroundDone.Wait()

	p.generation.disconnect()

	if ctx == nil {
		ctx = context.Background()
	}

	// If we have a deadline then we interpret it as a request to gracefully shutdown. We wait until
	// either all the connections have been checked back into the pool (i.e. total open connections
	// equals idle connections) or until the Context deadline is reached.
	if _, ok := ctx.Deadline(); ok {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

	graceful:
		for {
			if p.totalConnectionCount() == p.availableConnectionCount() {
				break graceful
			}

			select {
			case <-ticker.C:
			case <-ctx.Done():
				break graceful
			default:
			}
		}
	}

	// Empty the idle connections stack and try to deliver ErrPoolDisconnected to any waiting
	// wantConns from idleConnWait while holding the idleMu lock.
	p.idleMu.Lock()
	p.idleConns = p.idleConns[:0]
	for {
		w := p.idleConnWait.popFront()
		if w == nil {
			break
		}
		w.tryDeliver(nil, ErrPoolDisconnected)
	}
	p.idleMu.Unlock()

	// Collect all conns from the pool and try to deliver ErrPoolDisconnected to any waiting
	// wantConns from newConnWait while holding the connsCond lock. We can't call removeConnection
	// on the connections or cancel on the wantConns while holding any locks, so do that after we
	// release the lock.
	p.connsCond.L.Lock()
	conns := make([]*connection, 0, len(p.conns))
	for _, conn := range p.conns {
		conns = append(conns, conn)
	}
	for {
		w := p.newConnWait.popFront()
		if w == nil {
			break
		}
		w.tryDeliver(nil, ErrPoolDisconnected)
	}
	p.connsCond.L.Unlock()

	// Now that we're not holding any locks, remove all of the connections we collected from the
	// pool.
	for _, conn := range conns {
		_ = p.removeConnection(conn, event.ReasonPoolClosed)
		_ = p.closeConnection(conn) // We don't care about errors while closing the connection.
	}

	atomic.StoreInt64(&p.connected, disconnected)

	if p.monitor != nil {
		p.monitor.Event(&event.PoolEvent{
			Type:    event.PoolClosedEvent,
			Address: p.address.String(),
		})
	}

	return nil
}

func (p *pool) pinConnectionToCursor() {
	atomic.AddUint64(&p.pinnedCursorConnections, 1)
}

func (p *pool) unpinConnectionFromCursor() {
	// See https://golang.org/pkg/sync/atomic/#AddUint64 for an explanation of the ^uint64(0) syntax.
	atomic.AddUint64(&p.pinnedCursorConnections, ^uint64(0))
}

func (p *pool) pinConnectionToTransaction() {
	atomic.AddUint64(&p.pinnedTransactionConnections, 1)
}

func (p *pool) unpinConnectionFromTransaction() {
	// See https://golang.org/pkg/sync/atomic/#AddUint64 for an explanation of the ^uint64(0) syntax.
	atomic.AddUint64(&p.pinnedTransactionConnections, ^uint64(0))
}

// checkOut checks out a connection from the pool. If an idle connection is not available, the
// checkOut enters a queue waiting for either the next idle or new connection. If the pool is
// disconnected, checkOut returns an error.
// Based partially on https://cs.opensource.google/go/go/+/refs/tags/go1.16.6:src/net/http/transport.go;l=1324
func (p *pool) checkOut(ctx context.Context) (conn *connection, err error) {
	if atomic.LoadInt64(&p.connected) != connected {
		if p.monitor != nil {
			p.monitor.Event(&event.PoolEvent{
				Type:    event.GetFailed,
				Address: p.address.String(),
				Reason:  event.ReasonPoolClosed,
			})
		}
		return nil, ErrPoolDisconnected
	}

	if ctx == nil {
		ctx = context.Background()
	}

	// Create a wantConn, which we will use to request an existing idle or new connection. Always
	// cancel the wantConn if checkOut() returned an error to make sure any delivered connections
	// are returned to the pool (e.g. if a connection was delivered immediately after the Context
	// timed out).
	w := newWantConn()
	defer func() {
		if err != nil {
			w.cancel(p, err)
		}
	}()

	// Get in the queue for an idle connection. If getOrQueueForIdleConn returns true, it was able to
	// immediately deliver an idle connection to the wantConn, so we can return the connection or
	// error from the wantConn without waiting for "ready".
	if delivered := p.getOrQueueForIdleConn(w); delivered {
		if w.err != nil {
			if p.monitor != nil {
				p.monitor.Event(&event.PoolEvent{
					Type:    event.GetFailed,
					Address: p.address.String(),
					Reason:  event.ReasonConnectionErrored,
				})
			}
			return nil, w.err
		}

		if p.monitor != nil {
			p.monitor.Event(&event.PoolEvent{
				Type:         event.GetSucceeded,
				Address:      p.address.String(),
				ConnectionID: w.conn.poolID,
			})
		}
		return w.conn, nil
	}

	// If we didn't get an immediately available idle connection, also get in the queue for a new
	// connection while we're waiting for an idle connection.
	p.queueForNewConn(w)

	// Wait for either the wantConn to be ready or for the Context to time out.
	select {
	case <-w.ready:
		if w.err != nil {
			if p.monitor != nil {
				p.monitor.Event(&event.PoolEvent{
					Type:    event.GetFailed,
					Address: p.address.String(),
					Reason:  event.ReasonConnectionErrored,
				})
			}
			return nil, w.err
		}

		if p.monitor != nil {
			p.monitor.Event(&event.PoolEvent{
				Type:         event.GetSucceeded,
				Address:      p.address.String(),
				ConnectionID: w.conn.poolID,
			})
		}
		return w.conn, nil
	case <-ctx.Done():
		if p.monitor != nil {
			p.monitor.Event(&event.PoolEvent{
				Type:    event.GetFailed,
				Address: p.address.String(),
				Reason:  event.ReasonTimedOut,
			})
		}
		return nil, WaitQueueTimeoutError{
			Wrapped:                      ctx.Err(),
			PinnedCursorConnections:      atomic.LoadUint64(&p.pinnedCursorConnections),
			PinnedTransactionConnections: atomic.LoadUint64(&p.pinnedTransactionConnections),
			maxPoolSize:                  p.maxSize,
			totalConnectionCount:         p.totalConnectionCount(),
		}
	}
}

// closeConnection closes a connection.
func (p *pool) closeConnection(conn *connection) error {
	if conn.pool != p {
		return ErrWrongPool
	}

	if atomic.LoadInt64(&conn.connected) == connected {
		conn.closeConnectContext()
		_ = conn.wait() // Make sure that the connection has finished connecting
	}

	err := conn.close()
	if err != nil {
		return ConnectionError{ConnectionID: conn.id, Wrapped: err, message: "failed to close net.Conn"}
	}

	return nil
}

func (p *pool) getGenerationForNewConnection(serviceID *primitive.ObjectID) uint64 {
	return p.generation.addConnection(serviceID)
}

// removeConnection removes a connection from the pool and emits a "ConnectionClosed" event.
func (p *pool) removeConnection(conn *connection, reason string) error {
	if conn == nil {
		return nil
	}

	if conn.pool != p {
		return ErrWrongPool
	}

	p.connsCond.L.Lock()
	_, ok := p.conns[conn.poolID]
	if !ok {
		// If the connection has been removed from the pool already, exit without doing any
		// additional state changes.
		p.connsCond.L.Unlock()
		return nil
	}
	delete(p.conns, conn.poolID)
	// Signal the connsCond so any goroutines waiting for a new connection slot in the pool will
	// proceed.
	p.connsCond.Signal()
	p.connsCond.L.Unlock()

	// Only update the generation numbers map if the connection has retrieved its generation number.
	// Otherwise, we'd decrement the count for the generation even though it had never been
	// incremented.
	if conn.hasGenerationNumber() {
		p.generation.removeConnection(conn.desc.ServiceID)
	}

	if p.monitor != nil {
		p.monitor.Event(&event.PoolEvent{
			Type:         event.ConnectionClosed,
			Address:      p.address.String(),
			ConnectionID: conn.poolID,
			Reason:       reason,
		})
	}

	return nil
}

// checkIn returns an idle connection to the pool. If the connection is perished or the pool is
// disconnected, it is removed from the connection pool and closed.
func (p *pool) checkIn(conn *connection) error {
	if conn == nil {
		return nil
	}
	if conn.pool != p {
		return ErrWrongPool
	}

	if p.monitor != nil {
		p.monitor.Event(&event.PoolEvent{
			Type:         event.ConnectionReturned,
			ConnectionID: conn.poolID,
			Address:      conn.addr.String(),
		})
	}

	return p.checkInNoEvent(conn)
}

// checkInNoEvent returns a connection to the pool. It behaves identically to checkIn except it does
// not publish events. It is only intended for use by pool-internal functions.
func (p *pool) checkInNoEvent(conn *connection) error {
	if conn == nil {
		return nil
	}
	if conn.pool != p {
		return ErrWrongPool
	}

	if reason, perished := connectionPerished(conn); perished {
		_ = p.removeConnection(conn, reason)
		go func() {
			_ = p.closeConnection(conn)
		}()
		return nil
	}

	p.idleMu.Lock()
	defer p.idleMu.Unlock()

	for {
		w := p.idleConnWait.popFront()
		if w == nil {
			break
		}
		if w.tryDeliver(conn, nil) {
			return nil
		}
	}

	for _, idle := range p.idleConns {
		if idle == conn {
			return fmt.Errorf("duplicate idle conn %p in idle connections stack", conn)
		}
	}

	p.idleConns = append(p.idleConns, conn)
	return nil
}

// clear clears the pool by incrementing the generation
func (p *pool) clear(serviceID *primitive.ObjectID) {
	if p.monitor != nil {
		p.monitor.Event(&event.PoolEvent{
			Type:      event.PoolCleared,
			Address:   p.address.String(),
			ServiceID: serviceID,
		})
	}
	p.generation.clear(serviceID)
}

// getOrQueueForIdleConn attempts to deliver an idle connection to the given wantConn. If there is
// an idle connection in the idle connections stack, it pops an idle connection, delivers it to the
// wantConn, and returns true. If there are no idle connections in the idle connections stack, it
// adds the wantConn to the idleConnWait queue and returns false.
func (p *pool) getOrQueueForIdleConn(w *wantConn) bool {
	p.idleMu.Lock()
	defer p.idleMu.Unlock()

	// Try to deliver an idle connection from the idleConns stack first.
	for len(p.idleConns) > 0 {
		conn := p.idleConns[len(p.idleConns)-1]
		p.idleConns = p.idleConns[:len(p.idleConns)-1]

		if conn == nil {
			continue
		}

		if reason, perished := connectionPerished(conn); perished {
			_ = conn.pool.removeConnection(conn, reason)
			go func() {
				_ = conn.pool.closeConnection(conn)
			}()
			continue
		}

		if !w.tryDeliver(conn, nil) {
			// If we couldn't deliver the conn to w, put it back in the idleConns stack.
			p.idleConns = append(p.idleConns, conn)
		}

		// If we got here, we tried to deliver an idle conn to w. No matter if tryDeliver() returned
		// true or false, w is no longer waiting and doesn't need to be added to any wait queues, so
		// return delivered = true.
		return true
	}

	p.idleConnWait.cleanFront()
	p.idleConnWait.pushBack(w)
	return false
}

func (p *pool) queueForNewConn(w *wantConn) {
	p.connsCond.L.Lock()
	defer p.connsCond.L.Unlock()

	p.newConnWait.cleanFront()
	p.newConnWait.pushBack(w)
	p.connsCond.Signal()
}

func (p *pool) totalConnectionCount() int {
	p.connsCond.L.Lock()
	defer p.connsCond.L.Unlock()

	return len(p.conns)
}

func (p *pool) availableConnectionCount() int {
	p.idleMu.Lock()
	defer p.idleMu.Unlock()

	return len(p.idleConns)
}

// createConnections creates connections for wantConn requests on the newConnWait queue.
func (p *pool) createConnections(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	// condition returns true if the createConnections() loop should continue and false if it should
	// wait. Note that the condition also listens for Context cancellation, which also causes the
	// loop to continue, allowing for a subsequent check to return from createConnections().
	condition := func() bool {
		checkOutWaiting := p.newConnWait.len() > 0
		poolHasSpace := p.maxSize == 0 || uint64(len(p.conns)) < p.maxSize
		cancelled := ctx.Err() != nil
		return (checkOutWaiting && poolHasSpace) || cancelled
	}

	// wait waits for there to be an available wantConn and for the pool to have space for a new
	// connection. When the condition becomes true, it creates a new connection and returns the
	// waiting wantConn and new connection. If the Context is cancelled or there are any
	// errors, wait returns with "ok = false".
	wait := func() (*wantConn, *connection, bool) {
		p.connsCond.L.Lock()
		defer p.connsCond.L.Unlock()

		for !condition() {
			p.connsCond.Wait()
		}

		if ctx.Err() != nil {
			return nil, nil, false
		}

		p.newConnWait.cleanFront()
		w := p.newConnWait.popFront()
		if w == nil {
			return nil, nil, false
		}

		conn, err := newConnection(p.address, p.connOpts...)
		if err != nil {
			w.tryDeliver(nil, err)
			return nil, nil, false
		}

		conn.pool = p
		conn.poolID = atomic.AddUint64(&p.nextID, 1)
		p.conns[conn.poolID] = conn

		return w, conn, true
	}

	for ctx.Err() == nil {
		w, conn, ok := wait()
		if !ok {
			continue
		}

		if p.monitor != nil {
			p.monitor.Event(&event.PoolEvent{
				Type:         event.ConnectionCreated,
				Address:      p.address.String(),
				ConnectionID: conn.poolID,
			})
		}

		conn.connect(context.Background())
		err := conn.wait()
		if err != nil {
			_ = p.removeConnection(conn, event.ReasonConnectionErrored)
			_ = p.closeConnection(conn)
			w.tryDeliver(nil, err)
			continue
		}

		if p.monitor != nil {
			p.monitor.Event(&event.PoolEvent{
				Type:         event.ConnectionReady,
				Address:      p.address.String(),
				ConnectionID: conn.poolID,
			})
		}

		if w.tryDeliver(conn, nil) {
			continue
		}

		_ = p.checkInNoEvent(conn)
	}
}

func (p *pool) maintain(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(p.maintainInterval)
	defer ticker.Stop()

	// remove removes the *wantConn at index i from the slice and returns the new slice. The order
	// of the slice is not maintained.
	remove := func(arr []*wantConn, i int) []*wantConn {
		end := len(arr) - 1
		arr[i], arr[end] = arr[end], arr[i]
		return arr[:end]
	}

	// removeNotWaiting removes any wantConns that are no longer waiting from given slice of
	// wantConns. That allows maintain() to use the size of its wantConns slice as an indication of
	// how many new connection requests are outstanding and subtract that from the number of
	// connections to ask for when maintaining minPoolSize.
	removeNotWaiting := func(arr []*wantConn) []*wantConn {
		for i := len(arr) - 1; i >= 0; i-- {
			w := arr[i]
			if !w.waiting() {
				arr = remove(arr, i)
			}
		}

		return arr
	}

	wantConns := make([]*wantConn, 0, p.minSize)
	defer func() {
		for _, w := range wantConns {
			w.tryDeliver(nil, ErrPoolDisconnected)
		}
	}()

	for {
		p.removePerishedConns()

		// Remove any wantConns that are no longer waiting.
		wantConns = removeNotWaiting(wantConns)

		// Figure out how many more wantConns we need to satisfy minPoolSize. Assume that the
		// outstanding wantConns (i.e. the ones that weren't removed from the slice) will all return
		// connections when they're ready, so only add wantConns to make up the difference. Limit
		// the number of connections requested to max 10 at a time to prevent overshooting
		// minPoolSize in case other checkOut() calls are requesting new connections, too.
		total := p.totalConnectionCount()
		n := int(p.minSize) - total - len(wantConns)
		if n > 10 {
			n = 10
		}

		for i := 0; i < n; i++ {
			w := newWantConn()
			p.queueForNewConn(w)
			wantConns = append(wantConns, w)

			// Start a goroutine for each new wantConn, waiting for it to be ready.
			go func() {
				<-w.ready
				if w.conn != nil {
					_ = p.checkInNoEvent(w.conn)
				}
			}()
		}

		// Wait for the next tick at the bottom of the loop so that maintain() runs once immediately
		// after connect() is called. Exit the loop if the Context is cancelled.
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}

func (p *pool) removePerishedConns() {
	p.idleMu.Lock()
	defer p.idleMu.Unlock()

	for i := range p.idleConns {
		conn := p.idleConns[i]
		if conn == nil {
			continue
		}

		if reason, perished := connectionPerished(conn); perished {
			p.idleConns[i] = nil

			_ = p.removeConnection(conn, reason)
			go func() {
				_ = p.closeConnection(conn)
			}()
		}
	}

	p.idleConns = compact(p.idleConns)
}

// compact removes any nil pointers from the slice and keeps the non-nil pointers, retaining the
// order of the non-nil pointers.
func compact(arr []*connection) []*connection {
	offset := 0
	for i := range arr {
		if arr[i] == nil {
			continue
		}
		arr[offset] = arr[i]
		offset++
	}
	return arr[:offset]
}

// A wantConn records state about a wanted connection (that is, an active call to checkOut).
// The conn may be gotten by creating a new connection or by finding an idle connection, or a
// cancellation may make the conn no longer wanted. These three options are racing against each
// other and use wantConn to coordinate and agree about the winning outcome.
// Based on https://cs.opensource.google/go/go/+/refs/tags/go1.16.6:src/net/http/transport.go;l=1174-1240
type wantConn struct {
	ready chan struct{}

	mu   sync.Mutex // Guards conn, err
	conn *connection
	err  error
}

func newWantConn() *wantConn {
	return &wantConn{
		ready: make(chan struct{}, 1),
	}
}

// waiting reports whether w is still waiting for an answer (connection or error).
func (w *wantConn) waiting() bool {
	select {
	case <-w.ready:
		return false
	default:
		return true
	}
}

// tryDeliver attempts to deliver conn, err to w and reports whether it succeeded.
func (w *wantConn) tryDeliver(conn *connection, err error) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.conn != nil || w.err != nil {
		return false
	}

	w.conn = conn
	w.err = err
	if w.conn == nil && w.err == nil {
		panic("x/mongo/driver/topology: internal error: misuse of tryDeliver")
	}
	close(w.ready)
	return true
}

// cancel marks w as no longer wanting a result (for example, due to cancellation). If a connection
// has been delivered already, cancel returns it with p.checkInNoEvent(). Note that the caller must
// not hold any locks on the pool while calling cancel.
func (w *wantConn) cancel(p *pool, err error) {
	if err == nil {
		panic("x/mongo/driver/topology: internal error: misuse of cancel")
	}

	w.mu.Lock()
	if w.conn == nil && w.err == nil {
		close(w.ready) // catch misbehavior in future delivery
	}
	conn := w.conn
	w.conn = nil
	w.err = err
	w.mu.Unlock()

	if conn != nil {
		_ = p.checkInNoEvent(conn)
	}
}

// A wantConnQueue is a queue of wantConns.
// Based on https://cs.opensource.google/go/go/+/refs/tags/go1.16.6:src/net/http/transport.go;l=1242-1306
type wantConnQueue struct {
	// This is a queue, not a deque.
	// It is split into two stages - head[headPos:] and tail.
	// popFront is trivial (headPos++) on the first stage, and
	// pushBack is trivial (append) on the second stage.
	// If the first stage is empty, popFront can swap the
	// first and second stages to remedy the situation.
	//
	// This two-stage split is analogous to the use of two lists
	// in Okasaki's purely functional queue but without the
	// overhead of reversing the list when swapping stages.
	head    []*wantConn
	headPos int
	tail    []*wantConn
}

// len returns the number of items in the queue.
func (q *wantConnQueue) len() int {
	return len(q.head) - q.headPos + len(q.tail)
}

// pushBack adds w to the back of the queue.
func (q *wantConnQueue) pushBack(w *wantConn) {
	q.tail = append(q.tail, w)
}

// popFront removes and returns the wantConn at the front of the queue.
func (q *wantConnQueue) popFront() *wantConn {
	if q.headPos >= len(q.head) {
		if len(q.tail) == 0 {
			return nil
		}
		// Pick up tail as new head, clear tail.
		q.head, q.headPos, q.tail = q.tail, 0, q.head[:0]
	}
	w := q.head[q.headPos]
	q.head[q.headPos] = nil
	q.headPos++
	return w
}

// peekFront returns the wantConn at the front of the queue without removing it.
func (q *wantConnQueue) peekFront() *wantConn {
	if q.headPos < len(q.head) {
		return q.head[q.headPos]
	}
	if len(q.tail) > 0 {
		return q.tail[0]
	}
	return nil
}

// cleanFront pops any wantConns that are no longer waiting from the head of the
// queue, reporting whether any were popped.
func (q *wantConnQueue) cleanFront() (cleaned bool) {
	for {
		w := q.peekFront()
		if w == nil || w.waiting() {
			return cleaned
		}
		q.popFront()
		cleaned = true
	}
}
