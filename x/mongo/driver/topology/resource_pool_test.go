// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

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
	closeChan                         chan struct{}
}

func newExpiredCounter(total int) expiredCounter {
	return expiredCounter{
		total:     total,
		closeChan: make(chan struct{}, 1),
	}
}

func (ec *expiredCounter) expired(_ interface{}) bool {
	ec.expiredCalled++
	return ec.expiredCalled <= ec.total
}

func (ec *expiredCounter) close(_ interface{}) {
	ec.closeCalled++
	if ec.closeCalled == ec.total {
		ec.closeChan <- struct{}{}
	}
}

func initPool(startingSize, maxSize uint64, expFn expiredFunc, closeFn closeFunc, pruneInterval time.Duration) *resourcePool {
	rp := newResourcePool(maxSize, expFn, closeFn, pruneInterval)
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
			ec := newExpiredCounter(5)
			rp := initPool(5, 5, ec.expired, ec.close, time.Minute)
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
		t.Run("expired resource not returned", func(t *testing.T) {
			rp := initPool(0, 5, alwaysExpired, closeRsrc, time.Minute)
			ret := &rsrc{}
			if rp.Put(ret) {
				t.Fatal("return value mismatch; expected false, got true")
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
			ec := newExpiredCounter(3)
			rp := initPool(5, 5, ec.expired, ec.close, time.Minute)
			rp.Prune()
			if rp.len != 2 {
				t.Fatalf("length mismatch; expected 2, got %d", rp.len)
			}
			if ec.expiredCalled != 4 {
				t.Fatalf("count mismatch; expected ec.expired to be called 4 times, got %v", ec.expiredCalled)
			}
			if ec.closeCalled != 3 {
				t.Fatalf("count mismatch; expected ex.close to be called 3 times, got %v", ec.closeCalled)
			}
		})
	})
	t.Run("Background cleanup", func(t *testing.T) {
		t.Run("runs once every interval", func(t *testing.T) {
			ec := newExpiredCounter(5)
			_ = initPool(5, 5, ec.expired, ec.close, 100*time.Millisecond)
			select {
			case <-ec.closeChan:
			case <-time.After(5 * time.Second):
				t.Fatalf("value was not read on closeChan after 5 seconds")
			}
			if ec.expiredCalled != 5 {
				t.Fatalf("count mismatch; expected ec.expired to be called 5 times, got %v", ec.expiredCalled)
			}
			if ec.closeCalled != 5 {
				t.Fatalf("count mismatch; expected ec.close to be called 5 times, got %v", ec.closeCalled)
			}
		})
	})
}
