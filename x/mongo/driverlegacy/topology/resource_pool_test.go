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
	total, called int
}

func (ec *expiredCounter) expired(_ interface{}) bool {
	ec.called++
	return ec.called <= ec.total
}

func initPool(startingSize, maxSize uint64, expFn expiredFunc) *resourcePool {
	rp := NewResourcePool(maxSize, expFn, closeRsrc)
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
			rp := initPool(1, 1, alwaysExpired)
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
			rp := initPool(1, 1, neverExpired)
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
			rp := initPool(5, 5, alwaysExpired)
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
			rp := initPool(5, 5, neverExpired)
			if rp.Put(&rsrc{}) {
				t.Fatalf("return value mismatch; expected false, got true")
			}
			if rp.len != 5 {
				t.Fatalf("length mismatch; expected 5, got %d", rp.len)
			}
		})
	})
	t.Run("Prune", func(t *testing.T) {
		t.Run("stpo after first un-expired resource", func(t *testing.T) {
			ec := expiredCounter{total: 3}
			rp := initPool(5, 5, ec.expired)
			rp.Prune()
			if rp.len != 2 {
				t.Fatalf("length mismatch; expected 2, got %d", rp.len)
			}
			if ec.called != 4 {
				t.Fatalf("count mismatch; expected ec.called to be 4, got %v", ec.called)
			}
		})
	})
}
