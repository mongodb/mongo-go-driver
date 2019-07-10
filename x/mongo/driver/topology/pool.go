// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"context"
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

// strings for pool command monitoring reasons
const (
	IdleReason            = "idle"
	PoolClosedReason      = "poolClosed"
	StaleReason           = "stale"
	ConnectionErrorReason = "connectionError"
	TimeoutReason         = "timeout"
)

// strings for pool command monitoring types
const (
	ConnectionClosed   = "ConnectionClosed"
	PoolCreated        = "ConnectionPoolCreated"
	ConnectionCreated  = "ConnectionCreated"
	GetFailed          = "ConnectionCheckOutFailed"
	GetSucceeded       = "ConnectionCheckedOut"
	ConnectionReturned = "ConnectionCheckedIn"
	PoolCleared        = "ConnectionPoolCleared"
	PoolClosedEvent    = "ConnectionPoolClosed"
)

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
type PoolMonitor func(PoolEvent)

// poolConfig contains all aspects of the pool that can be configured
type poolConfig struct {
	Address                  address.Address
	MinPoolSize, MaxPoolSize uint64 // MaxPoolSize is not used because handling the max number of connections in the pool is handled in server. This is only used for command monitoring
	MaxIdleTime              time.Duration
	PoolMonitor              PoolMonitor
}

// checkOutResult is all the values that can be returned from a checkOut
type checkOutResult struct {
	c      *connection
	err    error
	reason string
}

// pool is a wrapper of resource pool that follows the CMAP spec for connection pools
type pool struct {
	address    address.Address
	opts       []ConnectionOption
	conns      *resourcePool // pool for non-checked out connections
	generation uint64        // must be accessed using atomic package, maintenance of maxSize requirement for the pool is handled in this pool not the resource pool
	monitor    PoolMonitor

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
		reason = PoolClosedReason
	}

	idle := c.expired()
	if !disconnected && idle {
		reason = IdleReason
	}

	stale := c.pool.stale(c)
	if !disconnected && !idle && stale {
		reason = StaleReason
	}

	res := disconnected || stale || idle
	if res && c.pool.monitor != nil {
		c.pool.monitor(PoolEvent{
			Type:         ConnectionClosed,
			Address:      c.pool.address.String(),
			ConnectionID: c.poolID,
			Reason:       reason,
		})
	}

	return res
}

// connectionCloseFunc closes a given connection. If ctx is nil, the closing will occur in the background
func connectionCloseFunc(v interface{}) {
	c, ok := v.(*connection)
	if !ok || v == nil {
		return
	}

	go func() { _ = c.pool.closeConnection(c) }()
}

// connectionInitFunc returns an init function for the resource pool that will make new connections for this pool
func (p *pool) connectionInitFunc() interface{} {
	res := p.makeNewConnection(context.Background())
	if res.err != nil {
		return nil
	}
	return res.c
}

// newPool creates a new pool that will hold size number of idle connections. It will use the
// provided options when creating connections.
func newPool(config poolConfig, connOpts ...ConnectionOption) (*pool, error) {
	opts := connOpts
	if config.MaxIdleTime != time.Duration(0) {
		opts = append(opts, WithIdleTimeout(func(_ time.Duration) time.Duration { return config.MaxIdleTime }))
	}

	pool := &pool{
		address:   config.Address,
		monitor:   config.PoolMonitor,
		connected: disconnected,
		opened:    make(map[uint64]*connection),
		opts:      opts,
	}

	// we do not pass in config.MaxPoolSize because we manage the max size at this level rather than the resource pool level
	rpc := resourcePoolConfig{
		MinSize:          config.MinPoolSize,
		MaintainInterval: maintainInterval,
		ExpiredFn:        connectionExpiredFunc,
		CloseFn:          connectionCloseFunc,
		InitFn:           pool.connectionInitFunc,
	}

	if pool.monitor != nil {
		pool.monitor(PoolEvent{
			Type: PoolCreated,
			PoolOptions: &MonitorPoolOptions{
				MaxPoolSize:        config.MaxPoolSize,
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

// drain drains the pool by increasing the generation ID.
func (p *pool) drain() { atomic.AddUint64(&p.generation, 1) }

// stale checks if a given connection's generation is below the generation of the pool
func (p *pool) stale(c *connection) bool {
	return c == nil || c.generation < atomic.LoadUint64(&p.generation)
}

// Connect puts the pool into the connected state, allowing it to be used and will allow items to begin being processed from the wait queue
func (p *pool) Connect() error {
	if !atomic.CompareAndSwapInt32(&p.connected, disconnected, connected) {
		return ErrPoolConnected
	}
	p.conns.initialize()
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
		if p.monitor != nil {
			p.monitor(PoolEvent{
				Type:         ConnectionClosed,
				Address:      p.address.String(),
				ConnectionID: pc.poolID,
				Reason:       PoolClosedReason,
			})
		}
		_ = p.closeConnection(pc) // We don't care about errors while closing the connection.
	}
	atomic.StoreInt32(&p.connected, disconnected)

	if p.monitor != nil {
		p.monitor(PoolEvent{
			Type:    PoolClosedEvent,
			Address: p.address.String(),
		})
	}

	return err
}

// requires that p be locked
func (p *pool) makeNewConnection(ctx context.Context) checkOutResult {
	c, err := newConnection(ctx, p.address, p.opts...)
	if err != nil {
		return checkOutResult{
			c:      nil,
			err:    err,
			reason: ConnectionErrorReason,
		}
	}

	c.pool = p
	c.poolID = atomic.AddUint64(&p.nextid, 1)
	c.generation = p.generation

	if p.monitor != nil {
		p.monitor(PoolEvent{
			Type:         ConnectionCreated,
			Address:      p.address.String(),
			ConnectionID: c.poolID,
		})
	}

	if atomic.LoadInt32(&p.connected) != connected {
		_ = p.closeConnection(c) // The pool is disconnected or disconnecting, ignore the error from closing the connection.
		return checkOutResult{
			c:   nil,
			err: ErrPoolDisconnected,
		}
	}

	p.Lock()
	p.opened[c.poolID] = c
	p.Unlock()

	return checkOutResult{
		c:   c,
		err: nil,
	}

}

// Checkout returns a connection from the pool
func (p *pool) Get(ctx context.Context) (*connection, error) {

	if ctx == nil {
		ctx = context.Background()
	}

	if atomic.LoadInt32(&p.connected) != connected {
		if p.monitor != nil {
			p.monitor(PoolEvent{
				Type:    GetFailed,
				Address: p.address.String(),
				Reason:  PoolClosedReason,
			})
		}
		return nil, ErrPoolDisconnected
	}

	connVal := p.conns.Get()
	if c, ok := connVal.(*connection); ok && connVal != nil {
		if p.monitor != nil {
			p.monitor(PoolEvent{
				Type:         GetSucceeded,
				Address:      p.address.String(),
				ConnectionID: c.poolID,
			})
		}
		return c, nil
	}

	select {
	case <-ctx.Done():
		if p.monitor != nil {
			p.monitor(PoolEvent{
				Type:    GetFailed,
				Address: p.address.String(),
				Reason:  TimeoutReason,
			})
		}
		return nil, ctx.Err()
	default:
		res := p.makeNewConnection(ctx)
		if res.err != nil {
			if p.monitor != nil {
				p.monitor(PoolEvent{
					Type:    GetFailed,
					Address: p.address.String(),
					Reason:  res.reason,
				})
			}
			return nil, res.err
		}

		if p.monitor != nil {
			p.monitor(PoolEvent{
				Type:         GetSucceeded,
				Address:      p.address.String(),
				ConnectionID: res.c.poolID,
			})
		}
		return res.c, nil
	}
}

// closeConnection closes a connection, not the pool itself. This method will actually closeConnection the connection,
// making it unusable, to instead return the connection to the pool, use Return.
func (p *pool) closeConnection(c *connection) error {
	if c.pool != p {
		return ErrWrongPool
	}
	p.Lock()
	delete(p.opened, c.poolID)
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

// Return returns a connection to this pool. If the pool is connected, the connection is not
// stale, and there is space in the cache, the connection is returned to the cache.
func (p *pool) Return(c *connection) error {
	if p.monitor != nil {
		var cid uint64
		var addr string
		if c != nil {
			cid = c.poolID
			addr = c.addr.String()
		}
		p.monitor(PoolEvent{
			Type:         ConnectionReturned,
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

	_ = p.conns.Return(c)

	return nil
}

// Clear clears the pool by incrementing the generation and then maintaining the pool
func (p *pool) Clear() {
	if p.monitor != nil {
		p.monitor(PoolEvent{
			Type:    PoolCleared,
			Address: p.address.String(),
		})
	}

	p.drain()
	p.conns.Maintain()
}
