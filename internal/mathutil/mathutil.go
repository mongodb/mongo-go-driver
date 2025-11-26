// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mathutil

import (
	"errors"
	"fmt"
	"math"
)

var (
	ErrOverflow    = errors.New("numeric overflow")
	ErrUnderflow   = errors.New("numeric underflow")
	ErrUnsupported = errors.New("unsupported numeric type")
)

func overflowError(from any, to any) error {
	return fmt.Errorf("%w: %v (%T) to %v (%T)", ErrOverflow, from, from, to, to)
}

func underflowError(from any, to any) error {
	return fmt.Errorf("%w: %v (%T) to %v (%T)", ErrUnderflow, from, from, to, to)
}

func unsupportedError(from any, to any) error {
	return fmt.Errorf("%w: %v (%T) to %v (%T)", ErrUnsupported, from, from, to, to)
}

type Numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 |
		~uint64 | ~float32 | ~float64
}

func i64ToT[T Numeric](i64 int64) (T, error) {
	var zero T

	switch any(zero).(type) {
	case int:
		if i64 > int64(math.MaxInt) || i64 < int64(math.MinInt) {
			return zero, overflowError(i64, zero)
		}
	case int32:
		if i64 > int64(math.MaxInt32) || i64 < int64(math.MinInt32) {
			return zero, overflowError(i64, zero)
		}
	case uint32:
		if i64 < 0 || i64 > int64(math.MaxUint32) {
			return zero, overflowError(i64, zero)
		}
	case uint64:
		if i64 < 0 {
			return zero, underflowError(i64, zero)
		}
	default:
		return zero, unsupportedError(i64, zero)
	}

	return T(i64), nil
}

func iToT[T Numeric](i int) (T, error) {
	var zero T

	switch any(zero).(type) {
	case int32:
		if i > int(math.MaxInt32) || i < int(math.MinInt32) {
			return zero, overflowError(i, zero)
		}
	case int64:
		return T(i), nil
	case uint:
		if i < 0 {
			return zero, overflowError(i, zero)
		}
	case uint32:
		if i < 0 || i > int(math.MaxUint32) {
			return zero, overflowError(i, zero)
		}
	case uint64:
		if i < 0 {
			return zero, underflowError(i, zero)
		}
	default:
		return zero, unsupportedError(i, zero)
	}

	return T(i), nil
}

func u64ToT[T Numeric](u64 uint64) (T, error) {
	var zero T
	maxUint := ^uint(0)

	switch any(zero).(type) {
	case int:
		if u64 > uint64(math.MaxInt) {
			return zero, overflowError(u64, zero)
		}
	case int32:
		if u64 > uint64(math.MaxInt32) {
			return zero, overflowError(u64, zero)
		}
	case int64:
		if u64 > uint64(math.MaxInt64) {
			return zero, overflowError(u64, zero)
		}
	case uint:
		if u64 > uint64(maxUint) {
			return zero, overflowError(u64, zero)
		}
	case uint32:
		if u64 > uint64(math.MaxUint32) {
			return zero, overflowError(u64, zero)
		}
	case uint64:
	default:
		return zero, unsupportedError(u64, zero)
	}

	return T(u64), nil
}

func uToT[T Numeric](u uint) (T, error) {
	return u64ToT[T](uint64(u))
}

func f64ToT[T Numeric](f64 float64) (T, error) {
	var zero T

	switch any(zero).(type) {
	case uint64:
		if f64 < 0 || f64 > float64(math.MaxUint64) {
			return zero, overflowError(f64, zero)
		}
	default:
		return zero, unsupportedError(f64, zero)
	}

	return T(f64), nil
}

func SafeConvertNumeric[T Numeric](number any) (T, error) {
	switch v := number.(type) {
	case int:
		return iToT[T](v)
	case int64:
		return i64ToT[T](v)
	case uint:
		return uToT[T](v)
	case uint64:
		return u64ToT[T](v)
	case float64:
		return f64ToT[T](v)
	}

	return *new(T), unsupportedError(number, *new(T))
}
