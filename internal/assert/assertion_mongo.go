// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// assertion_mongo.go contains MongoDB-specific extensions to the "assert"
// package.

package assert

import (
	"fmt"
	"reflect"
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
