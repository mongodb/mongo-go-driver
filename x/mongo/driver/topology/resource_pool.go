// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// expiredFunc is the function type used for testing whether or not resources in a resourcePool have stale. It should
// return true if the resource has stale and can be removed from the pool.
type expiredFunc func(interface{}) bool

// closeFunc is the function type used to closeConnection resources in a resourcePool. The pool will always call this function
// asynchronously.
type closeFunc func(context.Context, interface{})

// initFunc is the function used to add a resource to the resource pool to maintain minimum size. It returns a new
// resource each time it is called.
type initFunc func() interface{}

type resourcePoolConfig struct {
	MinSize, MaxSize, StartSize uint64
	MaintainInterval            time.Duration
	ExpiredFn                   expiredFunc
	CloseFn                     closeFunc
	InitFn                      initFunc
}

// setup sets defaults in the rpc and checks that the given values are valid
func (rpc *resourcePoolConfig) setup() error {
	if rpc.MaxSize == 0 {
		rpc.MaxSize = math.MaxUint64
	}

	if (rpc.MinSize != 0 || rpc.StartSize != 0) && rpc.InitFn == nil {
		return fmt.Errorf("cannot have a floored resource pool without an InitFn")
	}
	if rpc.ExpiredFn == nil {
		return fmt.Errorf("an ExpiredFn is required to create a resource pool")
	}
	if rpc.CloseFn == nil {
		return fmt.Errorf("an CloseFn is required to create a resource pool")
	}
	if rpc.MinSize >= rpc.MaxSize {
		return fmt.Errorf("MinSize: %v must be less than MaxSize: %v", rpc.MinSize, rpc.MaxSize)
	}
	if rpc.StartSize > rpc.MaxSize {
		return fmt.Errorf("StartSize: %v is unatainable, is larger than MaxSize: %v", rpc.StartSize, rpc.MaxSize)
	}
	if rpc.MaintainInterval == time.Duration(int64(0)) {
		return fmt.Errorf("unable to have MaintainInterval time of %v", rpc.MaintainInterval)
	}
	return nil
}

// resourcePoolElement is a link list element
type resourcePoolElement struct {
	next, prev *resourcePoolElement
	value      interface{}
}

// resourcePool is a concurrent resource pool
type resourcePool struct {
	start, end                                *resourcePoolElement
	size, minSize, maxSize, releasedResources uint64
	expiredFn                                 expiredFunc
	closeFn                                   closeFunc
	initFn                                    initFunc
	maintainTimer                             *time.Timer
	maintainInterval                          time.Duration

	sync.Mutex
}

// NewResourcePool creates a new resourcePool instance that is capped to maxSize resources.
// If maxSize is 0, the pool size will be unbounded.
func newResourcePool(config resourcePoolConfig) (*resourcePool, error) {
	err := (&config).setup()
	if err != nil {
		return nil, err
	}
	rp := &resourcePool{
		maxSize:          config.MaxSize,
		minSize:          config.MinSize,
		expiredFn:        config.ExpiredFn,
		closeFn:          config.CloseFn,
		initFn:           config.InitFn,
		maintainInterval: config.MaintainInterval,
	}

	rp.Lock()
	for i := uint64(0); i < config.StartSize; i++ {
		rp.add(nil)
	}
	rp.maintainTimer = time.AfterFunc(rp.maintainInterval, rp.Maintain)
	rp.Unlock()

	rp.Maintain()
	return rp, nil
}

// add will add a new rpe to the pool, requires that the resource pool is locked
func (rp *resourcePool) add(e *resourcePoolElement) {
	if e == nil {
		e = &resourcePoolElement{
			value: rp.initFn(),
		}
	}

	e.next = rp.start
	if rp.start != nil {
		rp.start.prev = e
	}
	rp.start = e
	atomic.AddUint64(&rp.size, 1)
	if rp.end == nil {
		rp.end = e
	}
}

// Get returns the first un-stale resource from the pool. If no such resource can be found, nil is returned.
func (rp *resourcePool) Get() interface{} {
	rp.Lock()
	defer rp.Unlock()

	for ; rp.start != nil; rp.start = rp.start.next {
		atomicSubtract1Uint64(&rp.size)
		if !rp.expiredFn(rp.start.value) {
			resElem := rp.start
			rp.start = rp.start.next
			if rp.start == nil {
				rp.end = nil
			} else {
				rp.start.prev = nil
			}
			atomic.AddUint64(&rp.releasedResources, 1)
			return resElem.value
		}
		rp.closeFn(nil, rp.start.value)
	}
	return nil
}

// Return puts the resource back into the pool if it will not exceed the max size of the pool
func (rp *resourcePool) Return(v interface{}) bool {
	atomicSubtract1Uint64(&rp.releasedResources)

	if atomic.LoadUint64(&rp.size)+atomic.LoadUint64(&rp.releasedResources) >= rp.maxSize || rp.expiredFn(v) {
		rp.closeFn(context.Background(), v)
		return false
	}

	rp.Lock()
	defer rp.Unlock()
	rp.add(&resourcePoolElement{value: v})
	return true
}

// Track will make the resource pool track an externally created resource. May cause pool to exceed max size.
func (rp *resourcePool) Track(v interface{}) {
	atomic.AddUint64(&rp.releasedResources, 1)
}

// remove removes a rpe from the linked list. Requires that the pool be locked
func (rp *resourcePool) remove(e *resourcePoolElement) {
	if e == nil {
		return
	}

	if e.prev != nil {
		e.prev.next = e.next
	}
	if e.next != nil {
		e.next.prev = e.prev
	}
	if e == rp.start {
		rp.start = e.next
	}
	if e == rp.end {
		rp.end = e.prev
	}
	atomicSubtract1Uint64(&rp.size)
}

// Maintain puts the pool back into a state of having a correct number of resources if possible and removes all stale resources
func (rp *resourcePool) Maintain() {
	rp.Lock()
	defer rp.Unlock()
	for curr := rp.end; curr != nil; curr = curr.prev {
		if rp.expiredFn(curr.value) {
			rp.remove(curr)
			rp.closeFn(context.TODO(), curr.value)
		}
	}

	for atomic.LoadUint64(&rp.size)+atomic.LoadUint64(&rp.releasedResources) < rp.minSize {
		rp.add(nil)
	}
	for atomic.LoadUint64(&rp.size)+atomic.LoadUint64(&rp.releasedResources) > rp.maxSize {
		rp.remove(rp.end)
	}

	// reset the timer for the background cleanup routine
	if !rp.maintainTimer.Stop() {
		rp.maintainTimer = time.AfterFunc(rp.maintainInterval, rp.Maintain)
		return
	}
	rp.maintainTimer.Reset(rp.maintainInterval)
}

// Close clears the pool and stops the background maintenance of the pool
func (rp *resourcePool) Close() {
	rp.Clear()
	_ = rp.maintainTimer.Stop()
}

// Clear closes all resources in the pool
func (rp *resourcePool) Clear() {
	rp.Lock()
	defer rp.Unlock()
	for ; rp.start != nil; rp.start = rp.start.next {
		rp.closeFn(context.Background(), rp.start.value)
	}
	atomic.StoreUint64(&rp.size, 0)
	rp.end = nil
}
