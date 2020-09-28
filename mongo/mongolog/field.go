// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongolog

import (
	"fmt"
	"strconv"
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

// Int64 constructs a field that carries an int64
func Int64(key string, val int64) Field {
	return Field{Key: key, Type: Int64Type, Integer: val}
}

// String constructs a field that carries a string
func String(key string, val string) Field {
	return Field{Key: key, Type: StringType, String: val}
}

// Stringer constructs a field with the given key and the output of the value's
// String method
func Stringer(key string, val fmt.Stringer) Field {
	return Field{Key: key, Type: StringerType, Interface: val}
}

// getValueString returns the value stored in field f as a string
func (f Field) getValueString() string {
	switch f.Type {
	case BoolType:
		return strconv.FormatBool(f.Integer == 1)
	case Int64Type:
		return strconv.FormatInt(f.Integer, 10)
	case StringType:
		return f.String
	case StringerType:
		return f.Interface.(fmt.Stringer).String()
	default:
		panic(fmt.Sprintf("unknown field type: %v", f))
	}
}

// A FieldType indicates which member of the Field union struct should be used
// and how it should be serialized.
type FieldType uint8

const (
	_ FieldType = iota
	// BoolType indicates that the field carries a bool.
	BoolType
	// Int64Type indicates that the field carries an int64.
	Int64Type
	// StringType indicates that the field carries a string.
	StringType
	// StringerType indicates that the field carries a fmt.Stringer.
	StringerType
)
