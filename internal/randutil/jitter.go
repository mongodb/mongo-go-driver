// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package randutil

import (
	"sync/atomic"
)

var presetRatio atomic.Pointer[float64]

var globalRand = NewLockedRand()

// JitterInt63n returns a random int64 in [0, n) by default. If a preset jitter ratio is set
// for testing, it returns a value based on the ratio instead of a random value.
func JitterInt63n(n int64) int64 {
	ratioPtr := presetRatio.Load()
	if ratioPtr == nil {
		return globalRand.Int63n(n)
	}
	val := int64(*ratioPtr * float64(n))
	if val < 0 {
		return 0
	}
	if val > n {
		return n
	}
	return int64(val)
}

// SetJitterForTesting sets a preset jitter ratio for testing and returns a restore function.
func SetJitterForTesting(ratio float64) func() {
	presetRatio.Store(&ratio)
	return func() { presetRatio.Store(nil) }
}
