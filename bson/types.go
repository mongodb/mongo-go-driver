// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"encoding/json"
	"net/url"
	"reflect"
	"time"

	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

// Type represents a BSON type.
type Type byte

// String returns the string representation of the BSON type's name.
func (bt Type) String() string {
	return bsoncore.Type(bt).String()
}

// IsValid will return true if the Type is valid.
func (bt Type) IsValid() bool {
	switch bt {
	case TypeDouble, TypeString, TypeEmbeddedDocument, TypeArray, TypeBinary,
		TypeUndefined, TypeObjectID, TypeBoolean, TypeDateTime, TypeNull, TypeRegex,
		TypeDBPointer, TypeJavaScript, TypeSymbol, TypeCodeWithScope, TypeInt32,
		TypeTimestamp, TypeInt64, TypeDecimal128, TypeMinKey, TypeMaxKey:
		return true
	default:
		return false
	}
}

// BSON element types as described in https://bsonspec.org/spec.html.
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
	TypeMaxKey           Type = 0x7F
	TypeMinKey           Type = 0xFF
)

// BSON binary element subtypes as described in https://bsonspec.org/spec.html.
const (
	TypeBinaryGeneric     byte = 0x00
	TypeBinaryFunction    byte = 0x01
	TypeBinaryBinaryOld   byte = 0x02
	TypeBinaryUUIDOld     byte = 0x03
	TypeBinaryUUID        byte = 0x04
	TypeBinaryMD5         byte = 0x05
	TypeBinaryEncrypted   byte = 0x06
	TypeBinaryColumn      byte = 0x07
	TypeBinarySensitive   byte = 0x08
	TypeBinaryVector      byte = 0x09
	TypeBinaryUserDefined byte = 0x80
)

var (
	tBool    = reflect.TypeOf(false)
	tFloat64 = reflect.TypeOf(float64(0))
	tInt32   = reflect.TypeOf(int32(0))
	tInt64   = reflect.TypeOf(int64(0))
	tString  = reflect.TypeOf("")
	tTime    = reflect.TypeOf(time.Time{})
)

var (
	tEmpty      = reflect.TypeOf((*any)(nil)).Elem()
	tByteSlice  = reflect.TypeOf([]byte(nil))
	tByte       = reflect.TypeOf(byte(0x00))
	tURL        = reflect.TypeOf(url.URL{})
	tJSONNumber = reflect.TypeOf(json.Number(""))
)

var (
	tValueMarshaler   = reflect.TypeOf((*ValueMarshaler)(nil)).Elem()
	tValueUnmarshaler = reflect.TypeOf((*ValueUnmarshaler)(nil)).Elem()
	tMarshaler        = reflect.TypeOf((*Marshaler)(nil)).Elem()
	tUnmarshaler      = reflect.TypeOf((*Unmarshaler)(nil)).Elem()
	tZeroer           = reflect.TypeOf((*Zeroer)(nil)).Elem()
)

var (
	tBinary        = reflect.TypeOf(Binary{})
	tUndefined     = reflect.TypeOf(Undefined{})
	tOID           = reflect.TypeOf(ObjectID{})
	tDateTime      = reflect.TypeOf(DateTime(0))
	tNull          = reflect.TypeOf(Null{})
	tRegex         = reflect.TypeOf(Regex{})
	tCodeWithScope = reflect.TypeOf(CodeWithScope{})
	tDBPointer     = reflect.TypeOf(DBPointer{})
	tJavaScript    = reflect.TypeOf(JavaScript(""))
	tSymbol        = reflect.TypeOf(Symbol(""))
	tTimestamp     = reflect.TypeOf(Timestamp{})
	tDecimal       = reflect.TypeOf(Decimal128{})
	tVector        = reflect.TypeOf(Vector{})
	tMinKey        = reflect.TypeOf(MinKey{})
	tMaxKey        = reflect.TypeOf(MaxKey{})
	tD             = reflect.TypeOf(D{})
	tA             = reflect.TypeOf(A{})
	tE             = reflect.TypeOf(E{})
)

var (
	tCoreDocument = reflect.TypeOf(bsoncore.Document{})
	tCoreArray    = reflect.TypeOf(bsoncore.Array{})
)
