// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driver

import (
	"testing"

	"go.mongodb.org/mongo-driver/internal/assert"
)

type dummypool struct {
	getcnt int
	putcnt int
}

func (p *dummypool) Get() interface{} {
	p.getcnt++
	b := make([]byte, 42)
	return &b
}

func (p *dummypool) Put(_ interface{}) {
	p.putcnt++
}

func TestByteSlicePool(t *testing.T) {
	t.Run("allocation", func(t *testing.T) {
		var memoryPool = newByteSlicePool(1)
		p := &dummypool{}
		memoryPool.pool = p
		b1 := memoryPool.Get()
		assert.Equal(t, 1, p.getcnt, "slice was not allocated correctly")
		b2 := memoryPool.Get()
		assert.Equal(t, 1, p.getcnt, "slice was not allocated correctly")
		memoryPool.Put(b2)
		assert.Equal(t, 1, p.putcnt, "slice was not returned correctly")
		memoryPool.Put(b1)
		assert.Equal(t, 1, p.putcnt, "slice was not returned correctly")
	})
}
