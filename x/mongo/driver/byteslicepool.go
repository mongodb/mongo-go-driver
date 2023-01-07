// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driver

import (
	"sync"
)

type byteslicePool struct {
	pool interface {
		Get() interface{}
		Put(interface{})
	}

	countmax int
	count    int
	mutex    *sync.Mutex
}

// newByteSlicePool creates a byte slices pool with a maximum number of items,
// which is specified by the parameter, "size".
func newByteSlicePool(size int) *byteslicePool {
	return &byteslicePool{
		pool: &sync.Pool{
			New: func() interface{} {
				// Start with 1kb buffers.
				b := make([]byte, 1024)
				// Return a pointer as the static analysis tool suggests.
				return &b
			},
		},
		countmax: size,
		mutex:    new(sync.Mutex),
	}
}

func (p *byteslicePool) Get() []byte {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.count < p.countmax {
		p.count++
		return (*p.pool.Get().(*[]byte))[:0]
	}
	return make([]byte, 0)
}

func (p *byteslicePool) Put(b []byte) {
	// Proper usage of a sync.Pool requires each entry to have approximately the same memory
	// cost. To obtain this property when the stored type contains a variably-sized buffer,
	// we add a hard limit on the maximum buffer to place back in the pool. We limit the
	// size to 16MiB because that's the maximum wire message size supported by MongoDB.
	//
	// Comment copied from https://cs.opensource.google/go/go/+/refs/tags/go1.19:src/fmt/print.go;l=147
	if c := cap(b); c <= 16*1024*1024 {
		p.mutex.Lock()
		defer p.mutex.Unlock()
		if p.count > 0 {
			p.pool.Put(&b)
			p.count--
		}
	}
}
