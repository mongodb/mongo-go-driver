package connection

import "context"

// Pool is used to pool Connections to a server.
type Pool interface {
	Get(context.Context) (Connection, error)
	Close() error
	Drain() error
}

// NewPool creates a new pool that will hold size number of idle connections
// and will create a max of capacity connections. It will use the provided
// options.
func NewPool(size, capacity uint64, opt ...Option) (Pool, error) { return nil, nil }
