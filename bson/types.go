// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

// These constants uniquely refer to each BSON type.
const (
	TypeDouble           Type = 0x01
	TypeString           Type = 0x02
	TypeEmbeddedDocument Type = 0x03
	TypeArray            Type = 0x04
	TypeBinary           Type = 0x05
	TypeUndefined        Type = 0x06
	TypeObjectID         Type = 0x07
	TypeBoolean          Type = 0x08
	TypeDateTime         Type = 0x09
	TypeNull             Type = 0x0A
	TypeRegex            Type = 0x0B
	TypeDBPointer        Type = 0x0C
	TypeJavaScript       Type = 0x0D
	TypeSymbol           Type = 0x0E
	TypeCodeWithScope    Type = 0x0F
	TypeInt32            Type = 0x10
	TypeTimestamp        Type = 0x11
	TypeInt64            Type = 0x12
	TypeDecimal128       Type = 0x13
	TypeMinKey           Type = 0xFF
	TypeMaxKey           Type = 0x7F
)

// Type represents a BSON type.
type Type byte

// String returns the string representation of the BSON type's name.
func (bt Type) String() string {
	switch bt {
	case '\x01':
		return "double"
	case '\x02':
		return "string"
	case '\x03':
		return "embedded document"
	case '\x04':
		return "array"
	case '\x05':
		return "binary"
	case '\x06':
		return "undefined"
	case '\x07':
		return "objectID"
	case '\x08':
		return "boolean"
	case '\x09':
		return "UTC datetime"
	case '\x0A':
		return "null"
	case '\x0B':
		return "regex"
	case '\x0C':
		return "dbPointer"
	case '\x0D':
		return "javascript"
	case '\x0E':
		return "symbol"
	case '\x0F':
		return "code with scope"
	case '\x10':
		return "32-bit integer"
	case '\x11':
		return "timestamp"
	case '\x12':
		return "64-bit integer"
	case '\x13':
		return "128-bit decimal"
	case '\xFF':
		return "min key"
	case '\x7F':
		return "max key"
	default:
		return "invalid"
	}
}
