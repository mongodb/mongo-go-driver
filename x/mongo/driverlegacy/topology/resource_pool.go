// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"container/list"
	"sync"
	"time"
)

// expiredFunc is the function type used for testing whether or not resources in a resourcePool have expired. It should
// return true if the resource has expired and can be removed from the pool.
type expiredFunc func(interface{}) bool

// closeFunc is the function type used to close resources in a resourcePool. The pool will always call this function
// asynchronously.
type closeFunc func(interface{})

// resourcePool is a concurrent resource pool that implements the behavior described in the sessions spec.
type resourcePool struct {
	deque        *list.List
	len, maxSize uint64
	expiredFn    expiredFunc
	closeFn      closeFunc
	cleanupTimer *time.Timer

	sync.Mutex
}

// NewResourcePool creates a new resourcePool instance that is capped to maxSize resources.
// If maxSize is 0, the pool size will be unbounded.
func NewResourcePool(maxSize uint64, expiredFn expiredFunc, closeFn closeFunc, cleanupInterval time.Duration) *resourcePool {
	rp := &resourcePool{
		deque:     list.New(),
		maxSize:   maxSize,
		expiredFn: expiredFn,
		closeFn:   closeFn,
	}
	rp.cleanupTimer = time.AfterFunc(cleanupInterval, rp.Prune)
	return rp
}

// Get returns the first un-expired resource from the pool. If no such resource can be found, nil is returned.
func (cp *resourcePool) Get() interface{} {
	cp.Lock()
	defer cp.Unlock()

	var next *list.Element
	for curr := cp.deque.Front(); curr != nil; curr = next {
		next = curr.Next()

		// remove the current resource and return it if it is valid
		cp.deque.Remove(curr)
		cp.len--
		if !cp.expiredFn(curr.Value) {
			// found un-expired resource
			return curr.Value
		}

		// asynchronously close expired resources
		go cp.closeFn(curr.Value)
	}

	// did not find a valid resource
	return nil
}

// Put clears expired resources from the pool and then returns resource v to the pool if there is room. It returns true
// if v was successfully added to the pool and false otherwise.
func (cp *resourcePool) Put(v interface{}) bool {
	cp.Lock()
	defer cp.Unlock()

	// close expired resources from the back of the pool
	cp.prune()
	if cp.len != 0 && cp.len == cp.maxSize {
		return false
	}
	cp.deque.PushFront(v)
	cp.len++
	return true
}

// Prune clears expired resources from the pool.
func (cp *resourcePool) Prune() {
	cp.Lock()
	defer cp.Unlock()
	cp.prune()
}

func (cp *resourcePool) prune() {
	// iterate over the list and stop at the first valid value
	var prev *list.Element
	for curr := cp.deque.Back(); curr != nil; curr = prev {
		prev = curr.Prev()
		if !cp.expiredFn(curr.Value) {
			// found unexpired resource
			break
		}

		// remove and asynchronously close expired resources
		cp.deque.Remove(curr)
		go cp.closeFn(curr.Value)
		cp.len--
	}

	// reset the timer for the background cleanup routine
	if !cp.cleanupTimer.Stop() {
		<-cp.cleanupTimer.C
	}
	cp.cleanupTimer.Reset(cleanupInterval)
}
