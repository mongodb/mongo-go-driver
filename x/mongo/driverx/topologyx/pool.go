package topologyx

import (
	"context"
	"sync"
	"sync/atomic"

	"go.mongodb.org/mongo-driver/x/network/address"
	"go.mongodb.org/mongo-driver/x/network/description"
	"golang.org/x/sync/semaphore"
)

// ErrPoolClosed is returned from an attempt to use a closed pool.
var ErrPoolClosed = PoolError("pool is closed")

// ErrSizeLargerThanCapacity is returned from an attempt to create a pool with a size
// larger than the capacity.
var ErrSizeLargerThanCapacity = PoolError("size is larger than capacity")

// ErrPoolConnected is returned from an attempt to connect an already connected pool
var ErrPoolConnected = PoolError("pool is connected")

// ErrPoolDisconnected is returned from an attempt to disconnect an already disconnected
// or disconnecting pool.
var ErrPoolDisconnected = PoolError("pool is disconnected or disconnecting")

// ErrConnectionClosed is returned from an attempt to use an already closed connection.
var ErrConnectionClosed = ConnectionError{ConnectionID: "<closed>", message: "connection is closed"}

// // These constants represent the connection states of a pool.
// const (
// 	disconnected int32 = iota
// 	disconnecting
// 	connected
// )

// PoolError is an error returned from a Pool method.
type PoolError string

func (pe PoolError) Error() string { return string(pe) }

type pool struct {
	address    address.Address
	opts       []ConnectionOption
	conns      chan *connection
	generation uint64
	sem        *semaphore.Weighted
	connected  int32
	nextid     uint64
	capacity   uint64
	inflight   map[uint64]*connection

	sync.Mutex
}

// NewPool creates a new pool that will hold size number of idle connections
// and will create a max of capacity connections. It will use the provided
// options.
func NewPool(addr address.Address, size, capacity uint64, opts ...ConnectionOption) (*pool, error) {
	if size > capacity {
		return nil, ErrSizeLargerThanCapacity
	}
	p := &pool{
		address:    addr,
		conns:      make(chan *connection, size),
		generation: 0,
		sem:        semaphore.NewWeighted(int64(capacity)),
		connected:  disconnected,
		capacity:   capacity,
		inflight:   make(map[uint64]*connection),
		opts:       opts,
	}
	return p, nil
}

func (p *pool) Drain() error {
	atomic.AddUint64(&p.generation, 1)
	return nil
}

func (p *pool) Connect(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&p.connected, disconnected, connected) {
		return ErrPoolConnected
	}
	atomic.AddUint64(&p.generation, 1)
	return nil
}

func (p *pool) Disconnect(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&p.connected, connected, disconnecting) {
		return ErrPoolDisconnected
	}

	// We first clear out the idle connections, then we attempt to acquire the entire capacity
	// semaphore. If the context is either cancelled, the deadline expires, or there is a timeout
	// the semaphore acquire method will return an error. If that happens, we will aggressively
	// close the remaining open connections. If we were able to successfully acquire the semaphore,
	// then all of the in flight connections have been closed and we release the semaphore.
loop:
	for {
		select {
		case pc := <-p.conns:
			// This error would be overwritten by the semaphore
			_ = p.closeConnection(pc)
		default:
			break loop
		}
	}
	err := p.sem.Acquire(ctx, int64(p.capacity))
	if err != nil {
		p.Lock()
		// We copy the remaining connections to close into a slice, then
		// iterate the slice to do the closing. This allows us to use a single
		// function to actually clean up and close connections at the expense of
		// a double iteration in the worst case.
		toClose := make([]*connection, 0, len(p.inflight))
		for _, pc := range p.inflight {
			toClose = append(toClose, pc)
		}
		p.Unlock()
		for _, pc := range toClose {
			_ = pc.Close()
		}
	} else {
		p.sem.Release(int64(p.capacity))
	}
	atomic.StoreInt32(&p.connected, disconnected)
	return nil
}

func (p *pool) get(ctx context.Context) (*connection, *description.Server, error) {
	select {
	case c := <-p.conns:
		if c.Expired() {
			go p.closeConnection(c)
			return p.get(ctx)
		}

		return c, nil, nil
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	default:
		pc, err := newConnection(ctx, p, p.address, p.opts...)
		if err != nil {
			return nil, nil, err
		}

		p.Lock()
		if atomic.LoadInt32(&p.connected) != connected {
			p.Unlock()
			p.closeConnection(pc)
			return nil, nil, ErrPoolClosed
		}
		defer p.Unlock()
		p.inflight[pc.poolID] = pc
		return pc, &pc.desc, nil
	}
}

func (p *pool) closeConnection(pc *connection) error {
	if !atomic.CompareAndSwapInt32(&pc.closed, 0, 1) {
		return nil
	}
	p.Lock()
	delete(p.inflight, pc.poolID)
	p.Unlock()
	err := pc.nc.Close()
	if err != nil {
		return ConnectionError{
			ConnectionID: pc.id,
			Wrapped:      err,
			message:      "failed to close net.Conn",
		}
	}
	return nil
}

func (p *pool) returnConnection(pc *connection) error {
	if atomic.LoadInt32(&p.connected) != connected || pc.Expired() {
		return p.closeConnection(pc)
	}

	select {
	case p.conns <- pc:
		return nil
	default:
		pc.dead = true
		return p.closeConnection(pc)
	}
}

func (p *pool) isExpired(generation uint64) bool {
	return generation < atomic.LoadUint64(&p.generation)
}
