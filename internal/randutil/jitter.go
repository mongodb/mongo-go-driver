// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package randutil

var globalRand = NewLockedRand()

var jitterInt63n func(int64) int64 = globalRand.Int63n

// JitterInt63n returns, as an int64, a non-negative pseudo-random number in
// the half-open interval [0,n). It panics if n <= 0.
//
// If a test jitter function is set by calling SetJitterForTesting, JitterInt63n
// returns the value from the custom function.
func JitterInt63n(n int64) int64 {
	return jitterInt63n(n)
}

// SetJitterForTesting sets a custom jitter function for testing and returns a restore function.
func SetJitterForTesting(f func(int64) int64) func() {
	jitterInt63n = f
	return func() {
		jitterInt63n = globalRand.Int63n
	}
}
