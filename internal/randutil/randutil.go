// Package randutil provides common random number utilities.
package randutil

import (
	"math/rand"
	"sync"
)

// A LockedRand wraps a "math/rand".Rand and is safe to use from multiple goroutines.
type LockedRand struct {
	mu sync.Mutex
	r  *rand.Rand
}

// NewLockedRand returns a new LockedRand that uses random values from src to generate other random
// values. It is safe to use from multiple goroutines.
func NewLockedRand(src rand.Source) *LockedRand {
	return &LockedRand{
		// Ignore gosec warning "Use of weak random number generator (math/rand instead of
		// crypto/rand)". We intentionally use a pseudo-random number generator.
		/* #nosec G404 */
		r: rand.New(src),
	}
}

// Read generates len(p) random bytes and writes them into p. It always returns len(p) and a nil
// error.
func (lr *LockedRand) Read(p []byte) (int, error) {
	lr.mu.Lock()
	n, err := lr.r.Read(p)
	lr.mu.Unlock()
	return n, err
}

// Intn returns, as an int, a non-negative pseudo-random number in the half-open interval [0,n). It
// panics if n <= 0.
func (lr *LockedRand) Intn(n int) int {
	lr.mu.Lock()
	x := lr.r.Intn(n)
	lr.mu.Unlock()
	return x
}

// Shuffle pseudo-randomizes the order of elements. n is the number of elements. Shuffle panics if
// n < 0. swap swaps the elements with indexes i and j.
//
// Note that Shuffle locks the LockedRand, so shuffling large collections may adversely affect other
// concurrent calls. If many concurrent Shuffle and random value calls are required, consider using
// the global "math/rand".Shuffle instead because it uses much more granular locking.
func (lr *LockedRand) Shuffle(n int, swap func(i, j int)) {
	lr.mu.Lock()
	lr.r.Shuffle(n, swap)
	lr.mu.Unlock()
}
