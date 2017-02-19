package internal

import "context"

// NewSemaphore creates a new semaphore.
func NewSemaphore(slots uint64) *Semaphore {
	ch := make(chan struct{}, slots)
	for i := uint64(0); i < slots; i++ {
		ch <- struct{}{}
	}

	return &Semaphore{
		permits: ch,
	}
}

// Semaphore is a syncronization primitive that controls access
// to a common resource.
type Semaphore struct {
	permits chan struct{}
}

// Len gets the number of permits available.
func (s *Semaphore) Len() uint64 {
	return uint64(len(s.permits))
}

// Wait waits until a resource is available or until the context
// is done.
func (s *Semaphore) Wait(ctx context.Context) error {
	select {
	case <-s.permits:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Release releases a resource back into the pool.
func (s *Semaphore) Release() {
	select {
	case s.permits <- struct{}{}:
	default:
		panic("internal.Semaphore.Release: attempt to release more resources than are available")
	}
}
