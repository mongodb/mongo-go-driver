// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// assertion_mongo.go contains MongoDB-specific extensions to the "assert"
// package.

package assert

import (
	"context"
	"fmt"
	"reflect"
	"time"
	"unsafe"
)

// DifferentAddressRanges asserts that two byte slices reference distinct memory
// address ranges, meaning they reference different underlying byte arrays.
func DifferentAddressRanges(t TestingT, a, b []byte) (ok bool) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}

	if len(a) == 0 || len(b) == 0 {
		return true
	}

	// Find the start and end memory addresses for the underlying byte array for
	// each input byte slice.
	sliceAddrRange := func(b []byte) (uintptr, uintptr) {
		sh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
		return sh.Data, sh.Data + uintptr(sh.Cap-1)
	}
	aStart, aEnd := sliceAddrRange(a)
	bStart, bEnd := sliceAddrRange(b)

	// If "b" starts after "a" ends or "a" starts after "b" ends, there is no
	// overlap.
	if bStart > aEnd || aStart > bEnd {
		return true
	}

	// Otherwise, calculate the overlap start and end and print the memory
	// overlap error message.
	min := func(a, b uintptr) uintptr {
		if a < b {
			return a
		}
		return b
	}
	max := func(a, b uintptr) uintptr {
		if a > b {
			return a
		}
		return b
	}
	overlapLow := max(aStart, bStart)
	overlapHigh := min(aEnd, bEnd)

	t.Errorf("Byte slices point to the same underlying byte array:\n"+
		"\ta addresses:\t%d ... %d\n"+
		"\tb addresses:\t%d ... %d\n"+
		"\toverlap:\t%d ... %d",
		aStart, aEnd,
		bStart, bEnd,
		overlapLow, overlapHigh)

	return false
}

// EqualBSON asserts that the expected and actual BSON binary values are equal.
// If the values are not equal, it prints both the binary and Extended JSON diff
// of the BSON values. The provided BSON value types must implement the
// fmt.Stringer interface.
func EqualBSON(t TestingT, expected, actual interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}

	return Equal(t,
		expected,
		actual,
		`expected and actual BSON values do not match
As Extended JSON:
Expected: %s
Actual  : %s`,
		expected.(fmt.Stringer).String(),
		actual.(fmt.Stringer).String())
}

// Soon runs the provided callback and fails the passed-in test if the callback
// does not complete within timeout. The provided callback should respect the
// passed-in context and cease execution when it has expired.
//
// Deprecated: This function will be removed with GODRIVER-2667, use
// assert.Eventually instead.
func Soon(t TestingT, callback func(ctx context.Context), timeout time.Duration) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}

	// Create context to manually cancel callback after Soon assertion.
	callbackCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	fullCallback := func() {
		callback(callbackCtx)
		done <- struct{}{}
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	go fullCallback()

	select {
	case <-done:
		return
	case <-timer.C:
		t.Errorf("timed out in %s waiting for callback", timeout)
	}
}
