// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package randutil

var presetRatio *float64

var globalRand = NewLockedRand()

// JitterInt63n returns, as an int64, a non-negative pseudo-random number in
// the half-open interval [0,n). It panics if n <= 0.
//
// If a static jitter ratio is set by calling SetJitterForTesting, JitterInt63n
// returns int64(n*ratio) in [0,n].
func JitterInt63n(n int64) int64 {
	if presetRatio == nil {
		return globalRand.Int63n(n)
	}
	val := int64(*presetRatio * float64(n))
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
	presetRatio = &ratio
	return func() { presetRatio = nil }
}
