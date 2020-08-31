// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongologger

import (
	"fmt"
	"math"
	"time"
)

var (
	_minTimeInt64 = time.Unix(0, math.MinInt64)
	_maxTimeInt64 = time.Unix(0, math.MaxInt64)
)

// A Field represents a key-value pair in the structured logs. Type is used to know
// where and how the variable is stored
type Field struct {
	Key       string
	Type      FieldType
	Integer   int64
	String    string
	Interface interface{}
}

// Bool constructs a field that carries a bool
func Bool(key string, val bool) Field {
	var ival int64
	if val {
		ival = 1
	}
	return Field{Key: key, Type: BoolType, Integer: ival}
}

// Float64 constructs a field that carries a float64
func Float64(key string, val float64) Field {
	return Field{Key: key, Type: Float64Type, Integer: int64(math.Float64bits(val))}
}

// Float32 constructs a field that carries a float32
func Float32(key string, val float32) Field {
	return Field{Key: key, Type: Float32Type, Integer: int64(math.Float32bits(val))}
}

// Int constructs a field that carries an int
func Int(key string, val int) Field {
	return Int64(key, int64(val))
}

// Int64 constructs a field that carries an int64
func Int64(key string, val int64) Field {
	return Field{Key: key, Type: Int64Type, Integer: val}
}

// Int32 constructs a field that carries an int32
func Int32(key string, val int32) Field {
	return Field{Key: key, Type: Int32Type, Integer: int64(val)}
}

// String constructs a field that carries a string
func String(key string, val string) Field {
	return Field{Key: key, Type: StringType, String: val}
}

// Uint constructs a field that carries a uint
func Uint(key string, val uint) Field {
	return Uint64(key, uint64(val))
}

// Uint64 constructs a field that carries a uint64
func Uint64(key string, val uint64) Field {
	return Field{Key: key, Type: Uint64Type, Integer: int64(val)}
}

// Uint32 constructs a field that carries a uint32
func Uint32(key string, val uint32) Field {
	return Field{Key: key, Type: Uint32Type, Integer: int64(val)}
}

// Uintptr constructs a field that carries a uintptr
func Uintptr(key string, val uintptr) Field {
	return Field{Key: key, Type: UintptrType, Integer: int64(val)}
}

// Reflect constructs a field that carries an arbitrary object
func Reflect(key string, val interface{}) Field {
	return Field{Key: key, Type: ReflectType, Interface: val}
}

// Stringer constructs a field with the given key and the output of the value's
// String method
func Stringer(key string, val fmt.Stringer) Field {
	return Field{Key: key, Type: StringerType, Interface: val}
}

// Time constructs a Field that carries a time.Time.
func Time(key string, val time.Time) Field {
	if val.Before(_minTimeInt64) || val.After(_maxTimeInt64) {
		return Field{Key: key, Type: TimeFullType, Interface: val}
	}
	return Field{Key: key, Type: TimeType, Integer: val.UnixNano(), Interface: val.Location()}
}

// Duration constructs a field that carries a time.Duration
func Duration(key string, val time.Duration) Field {
	return Field{Key: key, Type: DurationType, Integer: int64(val)}
}

// A FieldType indicates which member of the Field union struct should be used
// and how it should be serialized.
type FieldType uint8

const (
	_ FieldType = iota
	// BoolType indicates that the field carries a bool.
	BoolType
	// DurationType indicates that the field carries a time.Duration.
	DurationType
	// Float64Type indicates that the field carries a float64.
	Float64Type
	// Float32Type indicates that the field carries a float32.
	Float32Type
	// Int64Type indicates that the field carries an int64.
	Int64Type
	// Int32Type indicates that the field carries an int32.
	Int32Type
	// StringType indicates that the field carries a string.
	StringType
	// TimeType indicates that the field carries a time.Time that is
	// representable by a UnixNano() stored as an int64.
	TimeType
	// TimeFullType indicates that the field carries a time.Time stored as-is.
	TimeFullType
	// Uint64Type indicates that the field carries a uint64.
	Uint64Type
	// Uint32Type indicates that the field carries a uint32.
	Uint32Type
	// UintptrType indicates that the field carries a uintptr.
	UintptrType
	// ReflectType indicates that the field carries an interface{}, which should
	// be serialized using reflection.
	ReflectType
	// StringerType indicates that the field carries a fmt.Stringer.
	StringerType
)
