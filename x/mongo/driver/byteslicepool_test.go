// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driver

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/internal/assert"
)

func TestByteSlicePool(t *testing.T) {
	t.Run("allocation", func(t *testing.T) {
		var memoryPool = newByteSlicePool(1)
		b, err := memoryPool.Get(context.Background())
		assert.Nil(t, err, "allocation failed with error %v", err)
		assert.Equal(t, false, memoryPool.sem.TryAcquire(1), "semaphore was not acquired correctly")
		memoryPool.Put(b)
		assert.Equal(t, true, memoryPool.sem.TryAcquire(1), "semaphore was not released correctly")
	})
}
