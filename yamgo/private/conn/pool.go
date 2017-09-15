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
func NewPool(maxSize uint64, provider Provider) Pool {

	if maxSize == 0 {
		return &nonPool{provider}
	}

	return &idlePool{
		provider: provider,
		conns:    make(chan *poolConn, maxSize),
	}
}

// Pool holds connections such that they can be checked out
// and reused.
type Pool interface {
	// Clear drains the pool.
	Clear()
	// Close closes the pool, making it unusable.
	Close()
	// Get gets a connection from the pool. To return the connection
	// to the pool, close it.
	Get(context.Context) (Connection, error)
}

type nonPool struct {
	provider Provider
}

func (p *nonPool) Clear() {}

func (p *nonPool) Close() {}

func (p *nonPool) Get(ctx context.Context) (Connection, error) {
	return p.provider(ctx)
}

type idlePool struct {
	provider Provider

	connsLock sync.Mutex
	conns     chan *poolConn
	gen       uint32
}

func (p *idlePool) Clear() {
	atomic.AddUint32(&p.gen, 1)
}

func (p *idlePool) Close() {
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

func (p *idlePool) Get(ctx context.Context) (Connection, error) {
	p.connsLock.Lock()
	conns := p.conns
	p.connsLock.Unlock()

	if conns == nil {
		return nil, ErrPoolClosed
	}

	return p.getConn(ctx, conns)
}

func (p *idlePool) connExpired(gen uint32) bool {
	val := atomic.LoadUint32(&p.gen)
	return gen < val
}

func (p *idlePool) getConn(ctx context.Context, conns chan *poolConn) (Connection, error) {
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
		c, err := p.provider(ctx)
		if err != nil {
			return nil, err
		}

		return &poolConn{c, p, gen}, nil
	}
}

func (p *idlePool) returnConn(c *poolConn) error {
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
	p   *idlePool
	gen uint32
}

func (c *poolConn) Close() error {
	return c.p.returnConn(c)
}

func (c *poolConn) Expired() bool {
	return c.Connection.Expired() || c.p.connExpired(c.gen)
}
