package conn

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

// ErrPoolClosed is an error that occurs when
// attempting to use a pool that is closed.
var ErrPoolClosed = errors.New("pool is closed")

// NewPool creates a new connection pool.
func NewPool(maxSize uint16, factory func(context.Context) (Connection, error)) *Pool {
	return &Pool{
		factory: factory,
		conns:   make(chan *poolConn, maxSize),
	}
}

// Pool holds connections such that they can be checked out
// and reused.
type Pool struct {
	factory func(context.Context) (Connection, error)

	connsLock sync.Mutex
	conns     chan *poolConn
	gen       uint32
}

// Clear clears the pool. This does not happen immediately,
// but rather occurs as connections are checked out and
// checked in.
func (p *Pool) Clear() {
	atomic.AddUint32(&p.gen, 1)
}

// Close closes the pool, making it unusable. It closes
// all connections in the pool.
func (p *Pool) Close() {
	p.connsLock.Lock()
	conns := p.conns
	p.conns = nil
	p.connsLock.Unlock()

	if conns == nil {
		return
	}

	close(conns)
	for c := range conns {
		c.Close()
	}
}

// Get gets a connection from the pool. To return the connection
// to the pool, close it.
func (p *Pool) Get(ctx context.Context) (Connection, error) {
	p.connsLock.Lock()
	conns := p.conns
	p.connsLock.Unlock()

	if conns == nil {
		return nil, ErrPoolClosed
	}

	return p.getConn(ctx, conns)
}

func (p *Pool) connExpired(gen uint32) bool {
	val := atomic.LoadUint32(&p.gen)
	return gen < val
}

func (p *Pool) getConn(ctx context.Context, conns chan *poolConn) (Connection, error) {
	gen := atomic.LoadUint32(&p.gen)
	select {
	case c := <-conns:
		if c == nil {
			return nil, ErrPoolClosed
		}

		if c.Expired() {
			c.Connection.Close()
			return p.getConn(ctx, conns)
		}

		return c, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		c, err := p.factory(ctx)
		if err != nil {
			return nil, err
		}

		return &poolConn{c, p, gen}, nil
	}
}

func (p *Pool) returnConn(c *poolConn) error {
	if c.Expired() {
		return c.Connection.Close()
	}

	p.connsLock.Lock()
	defer p.connsLock.Unlock()

	if p.conns == nil {
		return c.Connection.Close()
	}

	select {
	case p.conns <- c:
		return nil
	default:
		// pool is full
		return c.Connection.Close()
	}
}

type poolConn struct {
	Connection
	p   *Pool
	gen uint32
}

func (c *poolConn) Close() error {
	return c.p.returnConn(c)
}

func (c *poolConn) Expired() bool {
	return c.Connection.Expired() || c.p.connExpired(c.gen)
}
