package topology

import (
	"testing"
	"time"
)

type rsrc struct {
	closed bool
}

func closeRsrc(v interface{}) {
	v.(*rsrc).closed = true
}

func alwaysExpired(_ interface{}) bool {
	return true
}

func neverExpired(_ interface{}) bool {
	return false
}

// expiredCounter is used to implement an expiredFunc that will return true a fixed number of times.
type expiredCounter struct {
	total, expiredCalled, closeCalled int
}

func (ec *expiredCounter) expired(_ interface{}) bool {
	ec.expiredCalled++
	return ec.expiredCalled <= ec.total
}

func (ec *expiredCounter) close(_ interface{}) {
	ec.closeCalled++
}

func initPool(startingSize, maxSize uint64, expFn expiredFunc, closeFn closeFunc, cleanupInterval time.Duration) *resourcePool {
	rp := NewResourcePool(maxSize, expFn, closeFn, cleanupInterval)
	var i uint64
	for i = 0; i < startingSize; i++ {
		rp.deque.PushBack(&rsrc{})
		rp.len++
	}
	return rp
}

func TestResourcePool(t *testing.T) {
	t.Run("Get", func(t *testing.T) {
		t.Run("remove expired resources", func(t *testing.T) {
			rp := initPool(1, 1, alwaysExpired, closeRsrc, time.Minute)
			r := rp.deque.Front().Value.(*rsrc)

			if got := rp.Get(); got != nil {
				t.Fatalf("resource mismatch; expected nil, got %v", got)
			}
			if rp.len != 0 {
				t.Fatalf("length mismatch; expected 0, got %d", rp.len)
			}
			time.Sleep(10 * time.Millisecond) // r will be closed async
			if !r.closed {
				t.Fatalf("expected resource to be closed but was not")
			}
		})
		t.Run("recycle resources", func(t *testing.T) {
			rp := initPool(1, 1, neverExpired, closeRsrc, time.Minute)
			for i := 0; i < 5; i++ {
				got := rp.Get()
				if got == nil {
					t.Fatalf("resource mismatch; expected a resource but got nil")
				}
				if rp.len != 0 {
					t.Fatalf("length mismatch; expected 0, got %d", rp.len)
				}
				rp.Put(got)
				if rp.len != 1 {
					t.Fatalf("length mismatch; expected 1, got %d", rp.len)
				}
			}
		})
	})
	t.Run("Put", func(t *testing.T) {
		t.Run("prune expired resources", func(t *testing.T) {
			rp := initPool(5, 5, alwaysExpired, closeRsrc, time.Minute)
			ret := &rsrc{}
			if !rp.Put(ret) {
				t.Fatal("return value mismatch; expected true, got false")
			}
			if rp.len != 1 {
				t.Fatalf("length mismatch; expected 1, got %d", rp.len)
			}
			if headVal := rp.deque.Front().Value; headVal != ret {
				t.Fatalf("resource mismatch; expected %v at head of pool, got %v", ret, headVal)
			}
		})
		t.Run("max size not exceeded", func(t *testing.T) {
			rp := initPool(5, 5, neverExpired, closeRsrc, time.Minute)
			if rp.Put(&rsrc{}) {
				t.Fatalf("return value mismatch; expected false, got true")
			}
			if rp.len != 5 {
				t.Fatalf("length mismatch; expected 5, got %d", rp.len)
			}
		})
	})
	t.Run("Prune", func(t *testing.T) {
		t.Run("stop after first un-expired resource", func(t *testing.T) {
			ec := expiredCounter{total: 3}
			rp := initPool(5, 5, ec.expired, ec.close, time.Minute)
			rp.Prune()
			if rp.len != 2 {
				t.Fatalf("length mismatch; expected 2, got %d", rp.len)
			}
			if ec.expiredCalled != 4 {
				t.Fatalf("count mismatch; expected ec.expired to be called 4 times, got %v", ec.expiredCalled)
			}
			time.Sleep(10 * time.Millisecond) // rsrcs will be closed async
			if ec.closeCalled != 3 {
				t.Fatalf("count mismatch; expected ex.close to be called 3 times, got %v", ec.closeCalled)
			}
		})
	})
	t.Run("Background cleanup", func(t *testing.T) {
		t.Run("runs once every interval", func(t *testing.T) {
			ec := expiredCounter{total: 5}
			_ = initPool(5, 5, ec.expired, ec.close, 100*time.Millisecond)
			time.Sleep(time.Second)
			if ec.expiredCalled != 5 {
				t.Fatalf("count mismatch; expected ec.expired to be called 5 times, got %v", ec.expiredCalled)
			}
			if ec.closeCalled != 5 {
				t.Fatalf("count mismatch; expected ec.close to be called 5 times, got %v", ec.closeCalled)
			}
		})
	})
}
