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

var tBool = reflect.TypeOf(false)
var tFloat64 = reflect.TypeOf(float64(0))
var tInt32 = reflect.TypeOf(int32(0))
var tInt64 = reflect.TypeOf(int64(0))
var tString = reflect.TypeOf("")
var tTime = reflect.TypeOf(time.Time{})

var tEmpty = reflect.TypeOf((*any)(nil)).Elem()
var tByteSlice = reflect.TypeOf([]byte(nil))
var tByte = reflect.TypeOf(byte(0x00))
var tURL = reflect.TypeOf(url.URL{})
var tJSONNumber = reflect.TypeOf(json.Number(""))

var tValueMarshaler = reflect.TypeOf((*ValueMarshaler)(nil)).Elem()
var tValueUnmarshaler = reflect.TypeOf((*ValueUnmarshaler)(nil)).Elem()
var tMarshaler = reflect.TypeOf((*Marshaler)(nil)).Elem()
var tUnmarshaler = reflect.TypeOf((*Unmarshaler)(nil)).Elem()
var tZeroer = reflect.TypeOf((*Zeroer)(nil)).Elem()

var tBinary = reflect.TypeOf(Binary{})
var tUndefined = reflect.TypeOf(Undefined{})
var tOID = reflect.TypeOf(ObjectID{})
var tDateTime = reflect.TypeOf(DateTime(0))
var tNull = reflect.TypeOf(Null{})
var tRegex = reflect.TypeOf(Regex{})
var tCodeWithScope = reflect.TypeOf(CodeWithScope{})
var tDBPointer = reflect.TypeOf(DBPointer{})
var tJavaScript = reflect.TypeOf(JavaScript(""))
var tSymbol = reflect.TypeOf(Symbol(""))
var tTimestamp = reflect.TypeOf(Timestamp{})
var tDecimal = reflect.TypeOf(Decimal128{})
var tVector = reflect.TypeOf(Vector{})
var tMinKey = reflect.TypeOf(MinKey{})
var tMaxKey = reflect.TypeOf(MaxKey{})
var tD = reflect.TypeOf(D{})
var tA = reflect.TypeOf(A{})
var tE = reflect.TypeOf(E{})

var tCoreDocument = reflect.TypeOf(bsoncore.Document{})
var tCoreArray = reflect.TypeOf(bsoncore.Array{})
