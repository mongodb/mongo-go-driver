// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"go.mongodb.org/mongo-driver/x/mongo/driver/address"
)

// ErrPoolConnected is returned from an attempt to Connect an already connected pool
var ErrPoolConnected = PoolError("attempted to Connect to an already connected pool")

// ErrPoolDisconnected is returned from an attempt to Close an already disconnected
// or disconnecting pool.
var ErrPoolDisconnected = PoolError("attempted to check out a connection from closed connection pool")

// ErrConnectionClosed is returned from an attempt to use an already closed connection.
var ErrConnectionClosed = ConnectionError{ConnectionID: "<closed>", message: "connection is closed"}

// ErrWrongPool is return when a connection is returned to a pool it doesn't belong to.
var ErrWrongPool = PoolError("connection does not belong to this pool")

// ErrWaitQueueTimeout is returned when the request to get a connection from the pool timesout when on the wait queue
var ErrWaitQueueTimeout = PoolError("timed out while checking out a connection from connection pool")

// ErrAllConnectionsReleasedTimeout is returned when a request to get a connection from the pool times out because all connections
// from the pool have been released and none are returned by the time the timeout is reached
var ErrAllConnectionsReleasedTimeout = PoolError("all connections from the pool have been released, timed out waiting for checkin")

// PoolError is an error returned from a Pool method.
type PoolError string

// maintainInterval is the interval at which the background routine to close stale connections will be run.
var maintainInterval = time.Minute

func (pe PoolError) Error() string { return string(pe) }

// MonitorPoolOptions contains pool options as formatted in pool events
type MonitorPoolOptions struct {
	MaxPoolSize        uint64 `json:"maxPoolSize"`
	MinPoolSize        uint64 `json:"minPoolSize"`
	WaitQueueTimeoutMS uint64 `json:"maxIdleTimeMS"`
}

// PoolEvent contains all information summarizing a pool event
type PoolEvent struct {
	Type         string              `json:"type"`
	Address      string              `json:"address"`
	ConnectionID uint64              `json:"connectionId"`
	PoolOptions  *MonitorPoolOptions `json:"options"`
	Reason       string              `json:"reason"`
}

// PoolMonitor is a function that allows the user to gain access to events occurring in the pool
type PoolMonitor = func(PoolEvent)

// poolConfig contains all aspects of the pool that can be configured
type poolConfig struct {
	Address     address.Address
	MaxPoolSize uint64
	MinPoolSize uint64
	MaxIdleTime time.Duration
	PoolMonitor PoolMonitor
}

// waitQueueItem is all the information that is needed to process an element of the wait queue
type waitQueueItem struct {
	op        func() interface{}
	result    chan interface{}
	canMoveOn chan interface{}
}

// checkOutResult is all the values that can be returned from a checkOut
type checkOutResult struct {
	c      *connection
	err    error
	reason string
}

// pool is a wrapper of resource pool that follows the CMAP spec for connection pools
type pool struct {
	address                                    address.Address
	opts                                       []ConnectionOption
	conns                                      *resourcePool // pool for non-checked out connections
	waitQueue                                  chan waitQueueItem
	generation, size, maxSize, connsCheckedOut uint64 // must be accessed using atomic package, maintenance of maxSize requirement for the pool is handled in this pool not the resource pool
	maxIdleTime                                *time.Duration
	monitor                                    PoolMonitor

	connected int32 // Must be accessed using the sync/atomic package.
	nextid    uint64
	opened    map[uint64]*connection // opened holds all of the currently open connections.
	sync.Mutex
}

// connectionExpiredFunc checks if a given connection is stale and should be removed from the resource pool
func connectionExpiredFunc(v interface{}) bool {
	if v == nil {
		return true
	}

	c, ok := v.(*connection)
	if !ok {
		return true
	}
	var reason string
	disconnected := atomic.LoadInt32(&c.pool.connected) != connected
	if disconnected {
		reason = "poolClosed"
	}

	c.pool.Lock()
	idle := c.lifetimeDeadline.Sub(time.Now()) < 0 || c.expired()
	if !disconnected && idle {
		reason = "idle"
	}
	c.pool.Unlock()
	stale := c.pool.stale(c)
	if !disconnected && !idle && stale {
		reason = "stale"
	}

	if (disconnected || stale || idle) && c.pool.monitor != nil {
		c.pool.monitor(PoolEvent{
			Type:         "ConnectionClosed",
			Address:      c.pool.address.String(),
			ConnectionID: c.poolID,
			Reason:       reason,
		})
	}

	return disconnected || stale || idle
}

// connectionCloseFunc closes a given connection. If ctx is nil, the closing will occur in the background
func connectionCloseFunc(ctx context.Context, v interface{}) {
	if v == nil {
		return
	}
	c := v.(*connection)

	if ctx == nil {
		go func() { _ = c.pool.closeConnection(c) }()
		return
	}

	var timeout <-chan time.Time
	timeout = make(chan time.Time, 0)
	if dl, ok := ctx.Deadline(); ok {
		timeout = time.After(dl.Sub(time.Now()))
	}

	closeChan := make(chan interface{}, 1)
	go func() {
		_ = c.pool.closeConnection(c)
		closeChan <- nil
	}()
	select {
	case <-closeChan:
	case <-timeout:
	case <-ctx.Done():
	}

}

// createInitFunc returns an init function for the resource pool that will make new connections for this pool
func (p *pool) createInitFunc() func() interface{} {
	return func() interface{} {
		c, err := newConnection(context.Background(), p.address, p.opts...)
		if err != nil {
			return nil
		}

		c.pool = p
		c.poolID = atomic.AddUint64(&p.nextid, 1)
		c.generation = atomic.LoadUint64(&p.generation)

		if p.monitor != nil {
			p.monitor(PoolEvent{
				Type:         "ConnectionCreated",
				Address:      p.address.String(),
				ConnectionID: c.poolID,
			})
		}

		p.Lock()
		p.opened[c.poolID] = c
		atomic.StoreUint64(&p.size, uint64(len(p.opened)))
		p.Unlock()

		if p.monitor != nil {
			p.monitor(PoolEvent{
				Type:         "ConnectionReady",
				ConnectionID: c.poolID,
			})
		}
		return c
	}
}

// validate sets defaults for the pool and verifies config for validity
func (pc *poolConfig) validate() error {
	if pc.MaxPoolSize == 0 {
		pc.MaxPoolSize = 100
	}
	if pc.MinPoolSize >= pc.MaxPoolSize {
		return fmt.Errorf("must have valid Min/MaxPoolSize combination: MinPoolSize: %v >= MaxPoolSize: %v", pc.MinPoolSize, pc.MaxPoolSize)
	}
	return nil
}

// newPool creates a new pool that will hold size number of idle connections. It will use the
// provided options when creating connections.
func newPool(config poolConfig, connOpts ...ConnectionOption) (*pool, error) {

	if err := (&config).validate(); err != nil {
		return nil, err
	}

	opts := connOpts
	if config.MaxIdleTime != time.Duration(0) {
		opts = append(opts, WithIdleTimeout(func(_ time.Duration) time.Duration { return config.MaxIdleTime }))
	}

	// TODO: change to maxPoolSize
	waitQueueLength := uint64(100)
	if config.MaxPoolSize > 50 {
		waitQueueLength = 2 * config.MaxPoolSize
	}

	pool := &pool{
		address:   config.Address,
		monitor:   config.PoolMonitor,
		maxSize:   config.MaxPoolSize,
		connected: disconnected,
		waitQueue: make(chan waitQueueItem, waitQueueLength),
		opened:    make(map[uint64]*connection),
		opts:      opts,
	}

	// we do not pass in config.MaxPoolSize because we manage the max size at this level rather than the resource pool level
	rpc := resourcePoolConfig{
		MinSize:          config.MinPoolSize,
		MaintainInterval: maintainInterval,
		ExpiredFn:        connectionExpiredFunc,
		CloseFn:          connectionCloseFunc,
		InitFn:           pool.createInitFunc(),
	}

	if pool.monitor != nil {
		pool.monitor(PoolEvent{
			Type: "ConnectionPoolCreated",
			PoolOptions: &MonitorPoolOptions{
				MaxPoolSize:        pool.maxSize,
				MinPoolSize:        rpc.MinSize,
				WaitQueueTimeoutMS: uint64(config.MaxIdleTime) / 1000000,
			},
			Address: pool.address.String(),
		})
	}

	rp, err := newResourcePool(rpc)
	if err != nil {
		return nil, err
	}
	pool.conns = rp

	return pool, nil
}

// runWaitQueue is designed to be run in a background go routine that will progress through the wait queue
func (p *pool) runWaitQueue() {
	for atomic.LoadInt32(&p.connected) == connected {
		item := <-p.waitQueue
		item.result <- item.op()
		<-item.canMoveOn
	}
}

// drain drains the pool by increasing the generation ID.
func (p *pool) drain() { atomic.AddUint64(&p.generation, 1) }

// stale checks if a given connection's generation is below the generation of the pool
func (p *pool) stale(c *connection) bool {
	return c != nil && c.generation < atomic.LoadUint64(&p.generation)
}

// Connect puts the pool into the connected state, allowing it to be used and will allow items to begin being processed from the wait queue
func (p *pool) Connect() error {
	if !atomic.CompareAndSwapInt32(&p.connected, disconnected, connected) {
		return ErrPoolConnected
	}
	go p.runWaitQueue()
	return nil
}

// Disconnect disconnects the pool and closes all connections including those both in and out of the pool
func (p *pool) Disconnect(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&p.connected, connected, disconnecting) {
		return ErrPoolDisconnected
	}

	if ctx == nil {
		ctx = context.Background()
	}

	p.conns.Close()

	atomic.AddUint64(&p.generation, 1)

	var err error
	if dl, ok := ctx.Deadline(); ok {
		// If we have a deadline then we interpret it as a request to gracefully shutdown. We wait
		// until either all the connections have landed back in the pool (and have been closed) or
		// until the timer is done.
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		timer := time.NewTimer(time.Now().Sub(dl))
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
			case <-ctx.Done():
			case <-ticker.C: // Can we replace this with an actual signal channel? We will know when p.inflight hits zero from the close method.
				p.Lock()
				if len(p.opened) > 0 {
					p.Unlock()
					continue
				}
				p.Unlock()
			}
			break
		}
	}

	// We copy the remaining connections into a slice, then iterate it to close them. This allows us
	// to use a single function to actually clean up and close connections at the expense of a
	// double iteration in the worse case.
	p.Lock()
	toClose := make([]*connection, 0, len(p.opened))
	for _, pc := range p.opened {
		toClose = append(toClose, pc)
	}
	p.Unlock()
	for _, pc := range toClose {
		_ = p.closeConnection(pc) // We don't care about errors while closing the connection.
	}
	atomic.StoreInt32(&p.connected, disconnected)
	return err
}

// Close disconnects the pool closes all connections in the pool. All connections outside of the pool will be closed when they are checked in
func (p *pool) Close(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&p.connected, connected, disconnecting) {
		return ErrPoolDisconnected
	}

	if ctx == nil {
		ctx = context.Background()
	}

	for {
		connVal := p.conns.Get()
		if connVal == nil {
			break
		}

		if p.monitor != nil {
			p.monitor(PoolEvent{
				Type:         "ConnectionClosed",
				Address:      p.address.String(),
				ConnectionID: connVal.(*connection).poolID,
				Reason:       "poolClosed",
			})
		}

		_ = p.closeConnection(connVal.(*connection))
	}

	atomic.AddUint64(&p.generation, 1)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timer := time.After(time.Duration(math.MaxInt32))
	if dl, ok := ctx.Deadline(); ok {
		timer = time.After(dl.Sub(time.Now()))
	}

	// If we have a deadline then we interpret it as a request to gracefully shutdown. We wait
	// until either all the connections have landed back in the pool (and have been closed) or
	// until the timer is done. If we do not have a deadline, we interpret it as an unlimited deadline
	for {
		select {
		case <-timer:
		case <-ctx.Done():
		case <-ticker.C: // Can we replace this with an actual signal channel? We will know when p.inflight hits zero from the close method.
			p.Lock()
			if len(p.opened) > 0 {
				p.Unlock()
				continue
			}
			p.Unlock()
		}
		break
	}

	atomic.StoreInt32(&p.connected, disconnected)

	if p.monitor != nil {
		p.monitor(PoolEvent{
			Type:    "ConnectionPoolClosed",
			Address: p.address.String(),
		})
	}

	return nil
}

// makeWaitQueueFunc makes a function to be put on a wait queue given a timeout and a context
func (p *pool) makeWaitQueueFunc(ctx context.Context, timeout <-chan time.Time) func() interface{} {
	return func() interface{} {
	Loop:
		for { // must wait until there are not maxSize connections checked out
			select {
			case <-timeout:
				return ErrWaitQueueTimeout
			case <-ctx.Done():
				return ctx.Err()
			default:
				if atomic.LoadUint64(&p.connsCheckedOut) == atomic.LoadUint64(&p.maxSize) {
					continue
				}
				break Loop
			}
		}

		if res := p.conns.Get(); res != nil {
			return res.(*connection)
		}
		return nil
	}
}

// Checkout returns a connection from the pool
func (p *pool) CheckOut(ctx context.Context) (*connection, error) {
	if p.monitor != nil {
		p.monitor(PoolEvent{
			Type:    "ConnectionCheckOutStarted",
			Address: p.address.String(),
		})
	}

	if ctx == nil {
		ctx = context.Background()
	}

	timeout := time.After(time.Duration(math.MaxInt32))
	if dl, ok := ctx.Deadline(); ok {
		timeout = time.After(dl.Sub(time.Now()))
	}

	if atomic.LoadInt32(&p.connected) != connected {
		if p.monitor != nil {
			p.monitor(PoolEvent{
				Type:    "ConnectionCheckOutFailed",
				Address: p.address.String(),
				Reason:  "poolClosed",
			})
		}
		return nil, ErrPoolDisconnected
	}
	resultChan := make(chan interface{}, 1)
	doneChan := make(chan interface{}, 1)

	var connVal *connection
	connFunc := p.makeWaitQueueFunc(ctx, timeout)

	select {
	case p.waitQueue <- waitQueueItem{connFunc, resultChan, doneChan}:
		defer func() {
			doneChan <- nil
		}()
		select {
		case temp := <-resultChan:
			switch temp.(type) {
			case *connection:
				connVal = temp.(*connection)
			case error:
				return nil, temp.(error)
			default:
				connVal = nil
			}
		case <-timeout:
			if p.monitor != nil {
				p.monitor(PoolEvent{
					Type:    "ConnectionCheckOutFailed",
					Address: p.address.String(),
					Reason:  "timeout",
				})
			}
			return nil, ErrWaitQueueTimeout
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	case <-timeout:
		if p.monitor != nil {
			p.monitor(PoolEvent{
				Type:    "ConnectionCheckOutFailed",
				Address: p.address.String(),
				Reason:  "timeout",
			})
		}
		return nil, ErrWaitQueueTimeout
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	if connVal != nil {
		if p.monitor != nil {
			p.monitor(PoolEvent{
				Type:         "ConnectionCheckedOut",
				Address:      p.address.String(),
				ConnectionID: connVal.poolID,
			})
		}
		atomic.AddUint64(&p.connsCheckedOut, 1)
		return connVal, nil
	}

	// couldn't find an unexpired connection. create a new one if that wont exceed the pool size
	madeNewConnChan := make(chan checkOutResult, 1)
	go func() {
	Loop:
		for { // wait for the size of the pool to be not the max size so that a new connection can be added
			select {
			case <-timeout:
				madeNewConnChan <- checkOutResult{nil, ErrAllConnectionsReleasedTimeout, "timeout"}
				return
			case <-ctx.Done():
				madeNewConnChan <- checkOutResult{nil, ctx.Err(), ""}
				return
			default:
				if atomic.LoadUint64(&p.size) != p.maxSize {
					break Loop
				}
			}
		}
		c, err := newConnection(context.Background(), p.address, p.opts...)
		madeNewConnChan <- checkOutResult{c, err, "connectionError"}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timeout:
		if p.monitor != nil {
			p.monitor(PoolEvent{
				Type:    "ConnectionCheckOutFailed",
				Address: p.address.String(),
				Reason:  "timeout",
			})
		}
		return nil, ErrAllConnectionsReleasedTimeout
	case result := <-madeNewConnChan:
		c, err := result.c, result.err
		if err != nil {
			if p.monitor != nil {
				p.monitor(PoolEvent{
					Type:    "ConnectionCheckOutFailed",
					Address: p.address.String(),
					Reason:  result.reason,
				})
			}
			return nil, err
		}

		c.pool = p
		c.poolID = atomic.AddUint64(&p.nextid, 1)
		c.generation = p.generation

		if p.monitor != nil {
			p.monitor(PoolEvent{
				Type:         "ConnectionCreated",
				Address:      p.address.String(),
				ConnectionID: c.poolID,
			})
		}

		if atomic.LoadInt32(&p.connected) != connected {
			_ = p.closeConnection(c) // The pool is disconnected or disconnecting, ignore the error from closing the connection.
			return nil, ErrPoolDisconnected
		}

		p.Lock()
		p.opened[c.poolID] = c
		atomic.StoreUint64(&p.size, uint64(len(p.opened)))
		p.Unlock()

		p.conns.Track(c)

		if p.monitor != nil {
			p.monitor(PoolEvent{
				Type:         "ConnectionReady",
				ConnectionID: c.poolID,
			})
			p.monitor(PoolEvent{
				Type:         "ConnectionCheckedOut",
				ConnectionID: c.poolID,
				Address:      c.addr.String(),
			})
		}
		atomic.AddUint64(&p.connsCheckedOut, 1)
		return c, err
	}
}

// closeConnection closes a connection, not the pool itself. This method will actually closeConnection the connection,
// making it unusable, to instead return the connection to the pool, use CheckIn.
func (p *pool) closeConnection(c *connection) error {
	if c.pool != p {
		return ErrWrongPool
	}
	p.Lock()
	delete(p.opened, c.poolID)
	atomic.StoreUint64(&p.size, uint64(len(p.opened)))
	p.Unlock()

	if !atomic.CompareAndSwapInt32(&c.connected, connected, disconnected) {
		return nil // We're closing an already closed connection
	}
	err := c.nc.Close()
	if err != nil {
		return ConnectionError{ConnectionID: c.id, Wrapped: err, message: "failed to closeConnection net.Conn"}
	}
	return nil
}

// CheckIn returns a connection to this pool. If the pool is connected, the connection is not
// stale, and there is space in the cache, the connection is returned to the cache.
func (p *pool) CheckIn(c *connection) error {
	if p.monitor != nil {
		cid := uint64(0)
		addr := ""
		if c != nil {
			cid = c.poolID
			addr = c.addr.String()
		}
		p.monitor(PoolEvent{
			Type:         "ConnectionCheckedIn",
			ConnectionID: cid,
			Address:      addr,
		})
	}

	if c == nil {
		return nil
	}

	if c.pool != p {
		return ErrWrongPool
	}

	defer atomicSubtract1Uint64(&p.connsCheckedOut)
	_ = p.conns.Return(c)

	return nil
}

// Clear clears the pool by incrementing the generation and then maintaining the pool
func (p *pool) Clear() {
	if p.monitor != nil {
		p.monitor(PoolEvent{
			Type:    "ConnectionPoolCleared",
			Address: p.address.String(),
		})
	}

	p.drain()
	p.conns.Maintain()
}

func atomicSubtract1Uint64(p *uint64) {
	if p == nil || atomic.LoadUint64(p) == 0 {
		return
	}

	for {
		expected := atomic.LoadUint64(p)
		if atomic.CompareAndSwapUint64(p, expected, expected-1) {
			return
		}
	}
}
