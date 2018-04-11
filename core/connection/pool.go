// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package connection

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/mongodb/mongo-go-driver/core/addr"
	"github.com/mongodb/mongo-go-driver/core/description"
	"golang.org/x/sync/semaphore"
)

// ErrPoolClosed is returned from an attempt to use a closed pool.
var ErrPoolClosed = errors.New("pool is closed")

// ErrSizeLargerThanCapacity is returned from an attempt to create a pool with a size
// larger than the capacity.
var ErrSizeLargerThanCapacity = errors.New("size is larger than capacity")

// Pool is used to pool Connections to a server.
type Pool interface {
	// Get must return a nil *description.Server if the returned connection is
	// not a newly dialed connection.
	Get(context.Context) (Connection, *description.Server, error)
	Close() error
	Drain() error
}

type pool struct {
	address    addr.Addr
	opts       []Option
	conns      chan *pooledConnection
	generation uint64
	sem        *semaphore.Weighted

	sync.Mutex
}

// NewPool creates a new pool that will hold size number of idle connections
// and will create a max of capacity connections. It will use the provided
// options.
func NewPool(address addr.Addr, size, capacity uint64, opts ...Option) (Pool, error) {
	if size > capacity {
		return nil, ErrSizeLargerThanCapacity
	}
	p := &pool{
		address:    address,
		conns:      make(chan *pooledConnection, size),
		generation: 0,
		sem:        semaphore.NewWeighted(int64(capacity)),
		opts:       opts,
	}
	return p, nil
}

func (p *pool) Drain() error {
	atomic.AddUint64(&p.generation, 1)
	return nil
}

func (p *pool) Close() error {
	p.Lock()
	conns := p.conns
	p.conns = nil
	p.Unlock()

	if conns == nil {
		return nil
	}

	close(conns)

	var err error

	for pc := range conns {
		err = pc.Close()
	}

	return err
}

func (p *pool) Get(ctx context.Context) (Connection, *description.Server, error) {
	p.Lock()
	conns := p.conns
	p.Unlock()

	if conns == nil {
		return nil, nil, ErrPoolClosed
	}

	return p.get(ctx, conns)
}

func (p *pool) get(ctx context.Context, conns chan *pooledConnection) (Connection, *description.Server, error) {
	g := atomic.LoadUint64(&p.generation)
	select {
	case c := <-conns:
		if c == nil {
			return nil, nil, ErrPoolClosed
		}
		if c.Expired() {
			go c.Connection.Close()
			return p.get(ctx, conns)
		}

		return c, nil, nil
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	default:
		err := p.sem.Acquire(ctx, 1)
		if err != nil {
			return nil, nil, err
		}
		c, desc, err := New(ctx, p.address, p.opts...)
		if err != nil {
			return nil, nil, err
		}

		return &pooledConnection{
			Connection: c,
			p:          p,
			generation: g,
		}, desc, nil
	}
}

func (p *pool) returnConnection(pc *pooledConnection) error {
	if pc.Expired() {
		return pc.Connection.Close()
	}

	p.Lock()
	defer p.Unlock()

	if p.conns == nil {
		pc.p.sem.Release(1)
		return pc.Connection.Close()
	}

	select {
	case p.conns <- pc:
		return nil
	default:
		pc.p.sem.Release(1)
		return pc.Connection.Close()
	}
}

func (p *pool) isExpired(generation uint64) bool {
	return generation < atomic.LoadUint64(&p.generation)
}

type pooledConnection struct {
	Connection
	p          *pool
	generation uint64
}

func (pc *pooledConnection) Close() error {
	return pc.p.returnConnection(pc)
}

func (pc *pooledConnection) Expired() bool {
	return pc.Connection.Expired() || pc.p.isExpired(pc.generation)
}
