package topology

import (
	"context"
	"sync/atomic"
)

func (p *pool) spawnConnection(ctx context.Context, w *wantConn, conn *connection) {
	// TODO
}

// hasSpace checks if the pool has space for a new connection.
func (p *pool) hasSpace() bool {
	return p.maxSize == 0 || uint64(len(p.conns)) < p.maxSize
}

// checkOutWaiting checks if there are any waiting connections that need to be
// checked out.
func (p *pool) checkOutWaiting() bool {
	return p.newConnWait.len() > 0
}

// waitForNewConn blocks until there's both work and room in the pool (or the
// context is canceled) then pops exactly one wantconn and creates+registes its
// connection.
func (p *pool) waitForNewConn(ctx context.Context) (*wantConn, *connection, bool) {
	p.createConnectionsCond.L.Lock()
	defer p.createConnectionsCond.L.Unlock()

	for !(p.checkOutWaiting() && p.hasSpace()) && ctx.Err() == nil {
		p.createConnectionsCond.Wait()
	}

	if ctx.Err() != nil {
		return nil, nil, false
	}

	p.newConnWait.cleanFront()
	w := p.newConnWait.popFront()
	if w == nil {
		return nil, nil, false
	}

	conn := newConnection(p.address, p.connOpts...)
	conn.pool = p
	conn.driverConnectionID = atomic.AddInt64(&p.nextID, 1)
	p.conns[conn.driverConnectionID] = conn

	return w, conn, true
}

// spawnConnectionIfNeeded takes on waiting waitConn (if any) and starts its
// connection creation subject to the semaphore limit.
func (p *pool) spawnConnectionIfNeeded(ctx context.Context) {
	// Block until we're allowed to start another connection.
	p.connectionSem <- struct{}{}

	// Wait on pool space & context.
	w, conn, ok := p.waitForNewConn(ctx)
	if !ok {
		<-p.connectionSem // Release slot on failure.

		return
	}

	// Check out connection in background as non-blocking.
	go p.spawnConnection(ctx, w, conn)
}
