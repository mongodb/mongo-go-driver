// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mathutil

import (
	"errors"
	"math"
	"reflect"
	"testing"
)

var (
	typeInt     = reflect.TypeOf(int(0))
	typeInt32   = reflect.TypeOf(int32(0))
	typeUint32  = reflect.TypeOf(uint32(0))
	typeUint64  = reflect.TypeOf(uint64(0))
	typeInt64   = reflect.TypeOf(int64(0))
	typeUint    = reflect.TypeOf(uint(0))
	maxInt      = int(^uint(0) >> 1)
	overflowU   = uint(maxInt) + 1
	overflowI32 = uint(math.MaxInt32) + 1
)

func TestSafeConvertNumeric(t *testing.T) {
	testCases := []struct {
		name      string
		target    reflect.Type
		input     any
		expectVal any
		expectErr error
	}{
		// int64 sources
		{name: "int64 to int32", target: typeInt32, input: int64(123), expectVal: int32(123)},
		{name: "int64 to int32 OF", target: typeInt32, input: int64(math.MaxInt32) + 1, expectErr: ErrOverflow},
		{name: "int64 to uint32", target: typeUint32, input: int64(789), expectVal: uint32(789)},
		{name: "int64 to uint32 OF", target: typeUint32, input: int64(math.MaxUint32) + 1, expectErr: ErrOverflow},
		{name: "int64 to uint32 UF", target: typeUint32, input: int64(-1), expectErr: ErrOverflow},
		{name: "int64 to uint64", target: typeUint64, input: int64(131415), expectVal: uint64(131415)},
		{name: "int64 to uint64 UF", target: typeUint64, input: int64(-1), expectErr: ErrUnderflow},

		// int sources
		{name: "int to int32", target: typeInt32, input: int(42), expectVal: int32(42)},
		{name: "int to int32 OF", target: typeInt32, input: int(math.MaxInt32) + 1, expectErr: ErrOverflow},
		{name: "int to uint32", target: typeUint32, input: int(101112), expectVal: uint32(101112)},
		{name: "int to uint32 OF", target: typeUint32, input: int(math.MaxUint32) + 1, expectErr: ErrOverflow},
		{name: "int to uint32 UF", target: typeUint32, input: int(-1), expectErr: ErrOverflow},
		{name: "int to uint64", target: typeUint64, input: int(161718), expectVal: uint64(161718)},
		{name: "int to uint64 UF", target: typeUint64, input: int(-1), expectErr: ErrUnderflow},
		{name: "int to uint", target: typeUint, input: int(202122), expectVal: uint(202122)},
		{name: "int to uint UF", target: typeUint, input: int(-1), expectErr: ErrOverflow},

		// uint sources
		{name: "uint to int", target: typeInt, input: uint(123), expectVal: int(123)},
		{name: "uint to int OF", target: typeInt, input: overflowU, expectErr: ErrOverflow},
		{name: "uint to int32", target: typeInt32, input: uint(321), expectVal: int32(321)},
		{name: "uint to int32 OF", target: typeInt32, input: overflowI32, expectErr: ErrOverflow},
		{name: "uint to int64", target: typeInt64, input: uint(654321), expectVal: int64(654321)},
		{name: "uint to uint", target: typeUint, input: uint(777), expectVal: uint(777)},
		{name: "uint to uint32", target: typeUint32, input: uint(888), expectVal: uint32(888)},
		{name: "uint to uint64", target: typeUint64, input: uint(999), expectVal: uint64(999)},

		// float64 sources
		{name: "float64 to uint64", target: typeUint64, input: float64(123), expectVal: uint64(123)},
		{name: "float64 to uint64 OF", target: typeUint64, input: math.Nextafter(float64(math.MaxUint64), math.Inf(1)), expectErr: ErrOverflow},
		{name: "float64 to uint64 UF", target: typeUint64, input: float64(-1), expectErr: ErrOverflow},
		{name: "float64 unsupported target", target: typeInt32, input: float64(1), expectErr: ErrUnsupported},

		// unsupported cases
		{name: "unsupported input type", target: typeInt32, input: "not-a-number", expectErr: ErrUnsupported},
		{name: "unsupported target type", target: typeInt64, input: int64(1), expectErr: ErrUnsupported},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var (
				got any
				err error
			)

			switch tc.target {
			case typeInt32:
				got, err = SafeConvertNumeric[int32](tc.input)
			case typeUint32:
				got, err = SafeConvertNumeric[uint32](tc.input)
			case typeUint64:
				got, err = SafeConvertNumeric[uint64](tc.input)
			case typeInt64:
				got, err = SafeConvertNumeric[int64](tc.input)
			case typeUint:
				got, err = SafeConvertNumeric[uint](tc.input)
			case typeInt:
				got, err = SafeConvertNumeric[int](tc.input)
			default:
				t.Fatalf("unexpected target type: %v", tc.target)
			}

			if tc.expectErr != nil {
				if !errors.Is(err, tc.expectErr) {
					t.Fatalf("expected error %v, got %v", tc.expectErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expectVal {
				t.Fatalf("expected %v, got %v", tc.expectVal, got)
			}
		})
	}
}

func BenchmarkSafeConvertNumeric(b *testing.B) {
	benchmarks := []struct {
		name   string
		target reflect.Type
		input  any
	}{
		// int64 sources
		{name: "int64 to int32", target: typeInt32, input: int64(123)},
		{name: "int64 to uint32", target: typeUint32, input: int64(789)},
		{name: "int64 to uint64", target: typeUint64, input: int64(131415)},

		// int sources
		{name: "int to int32", target: typeInt32, input: int(456)},
		{name: "int to uint32", target: typeUint32, input: int(101112)},
		{name: "int to uint64", target: typeUint64, input: int(161718)},
		{name: "int to uint", target: typeUint, input: int(202122)},

		// uint sources
		{name: "uint to int", target: typeInt, input: uint(123)},
		{name: "uint to int32", target: typeInt32, input: uint(321)},
		{name: "uint to uint32", target: typeUint32, input: uint(888)},
		{name: "uint to uint64", target: typeUint64, input: uint(999)},
		{name: "uint to uint", target: typeUint, input: uint(202122)},

		// float64 sources
		{name: "float64 to uint64", target: typeUint64, input: float64(123)},
	}

	for _, bm := range benchmarks {
		bm := bm
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			switch bm.target {
			case typeInt32:
				for i := 0; i < b.N; i++ {
					_, _ = SafeConvertNumeric[int32](bm.input)
				}
			case typeUint32:
				for i := 0; i < b.N; i++ {
					_, _ = SafeConvertNumeric[uint32](bm.input)
				}
			case typeUint64:
				for i := 0; i < b.N; i++ {
					_, _ = SafeConvertNumeric[uint64](bm.input)
				}
			case typeInt64:
				for i := 0; i < b.N; i++ {
					_, _ = SafeConvertNumeric[int64](bm.input)
				}
			case typeUint:
				for i := 0; i < b.N; i++ {
					_, _ = SafeConvertNumeric[uint](bm.input)
				}
			case typeInt:
				for i := 0; i < b.N; i++ {
					_, _ = SafeConvertNumeric[int](bm.input)
				}
			default:
				b.Fatalf("unexpected target type: %v", bm.target)
			}
		})
	}
}
