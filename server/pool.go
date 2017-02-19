package server

import (
	"context"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/internal"
)

type pool interface {
	Clear()
	Close()
	Get(context.Context) (conn.Connection, error)
}

type nonPool struct {
	factory func(ctx context.Context) (conn.Connection, error)
}

func (p *nonPool) Clear() {}

func (p *nonPool) Close() {}

func (p *nonPool) Get(ctx context.Context) (conn.Connection, error) {
	return p.factory(ctx)
}

func newLimitedPool(maxConns uint16, pool pool) *limitedPool {
	return &limitedPool{
		pool:    pool,
		permits: internal.NewSemaphore(uint64(maxConns)),
	}
}

type limitedPool struct {
	pool
	permits *internal.Semaphore
}

func (p *limitedPool) Get(ctx context.Context) (conn.Connection, error) {
	err := p.permits.Wait(ctx)
	if err != nil {
		return nil, err
	}

	c, err := p.pool.Get(ctx)
	if err != nil {
		p.permits.Release()
	}

	return &limitedPoolConn{c, p}, err
}

func (p *limitedPool) connClosed(c *limitedPoolConn) error {
	err := c.Connection.Close()
	p.permits.Release()
	return err
}

type limitedPoolConn struct {
	conn.Connection
	pool *limitedPool
}

func (c *limitedPoolConn) Close() error {
	return c.pool.connClosed(c)
}
