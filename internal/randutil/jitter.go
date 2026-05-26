// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package randutil

import "time"

var globalRand = NewLockedRand()

// jitter is the function used by Jitter. Production code uses
// globalRand.Float64; tests may swap it via SetJitterForTesting.
var jitter func() float64 = globalRand.Float64

// JitterDuration returns the input duration weighted by a pseudo-random ratio
// in [0.0, 1.0).
//
// If a test jitter function is set by calling SetJitterForTesting,
// JitterDuration returns the value from that custom function.
func JitterDuration(d time.Duration) time.Duration {
	return time.Duration(float64(d) * jitter())
}

// SetJitterForTesting sets a custom jitter function for testing and returns
// a restore function. The function should return a ratio in [0.0, 1.0); values
// outside that range are not validated and will produce backoffs outside
// the spec-defined range.
func SetJitterForTesting(f func() float64) func() {
	jitter = f
	return func() {
		jitter = globalRand.Float64
	}
}
