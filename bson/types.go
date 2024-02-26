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

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

// Type represents a BSON type.
type Type bsoncore.Type

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
	TypeDouble           = Type(bsoncore.TypeDouble)
	TypeString           = Type(bsoncore.TypeString)
	TypeEmbeddedDocument = Type(bsoncore.TypeEmbeddedDocument)
	TypeArray            = Type(bsoncore.TypeArray)
	TypeBinary           = Type(bsoncore.TypeBinary)
	TypeUndefined        = Type(bsoncore.TypeUndefined)
	TypeObjectID         = Type(bsoncore.TypeObjectID)
	TypeBoolean          = Type(bsoncore.TypeBoolean)
	TypeDateTime         = Type(bsoncore.TypeDateTime)
	TypeNull             = Type(bsoncore.TypeNull)
	TypeRegex            = Type(bsoncore.TypeRegex)
	TypeDBPointer        = Type(bsoncore.TypeDBPointer)
	TypeJavaScript       = Type(bsoncore.TypeJavaScript)
	TypeSymbol           = Type(bsoncore.TypeSymbol)
	TypeCodeWithScope    = Type(bsoncore.TypeCodeWithScope)
	TypeInt32            = Type(bsoncore.TypeInt32)
	TypeTimestamp        = Type(bsoncore.TypeTimestamp)
	TypeInt64            = Type(bsoncore.TypeInt64)
	TypeDecimal128       = Type(bsoncore.TypeDecimal128)
	TypeMaxKey           = Type(bsoncore.TypeMaxKey)
	TypeMinKey           = Type(bsoncore.TypeMinKey)
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
	TypeBinaryUserDefined byte = 0x80
)

var tBool = reflect.TypeOf(false)
var tFloat64 = reflect.TypeOf(float64(0))
var tInt32 = reflect.TypeOf(int32(0))
var tInt64 = reflect.TypeOf(int64(0))
var tString = reflect.TypeOf("")
var tTime = reflect.TypeOf(time.Time{})

var tEmpty = reflect.TypeOf((*interface{})(nil)).Elem()
var tByteSlice = reflect.TypeOf([]byte(nil))
var tByte = reflect.TypeOf(byte(0x00))
var tURL = reflect.TypeOf(url.URL{})
var tJSONNumber = reflect.TypeOf(json.Number(""))

var tValueMarshaler = reflect.TypeOf((*ValueMarshaler)(nil)).Elem()
var tValueUnmarshaler = reflect.TypeOf((*ValueUnmarshaler)(nil)).Elem()
var tMarshaler = reflect.TypeOf((*Marshaler)(nil)).Elem()
var tUnmarshaler = reflect.TypeOf((*Unmarshaler)(nil)).Elem()
var tProxy = reflect.TypeOf((*Proxy)(nil)).Elem()
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
var tMinKey = reflect.TypeOf(MinKey{})
var tMaxKey = reflect.TypeOf(MaxKey{})
var tD = reflect.TypeOf(D{})
var tA = reflect.TypeOf(A{})
var tE = reflect.TypeOf(E{})

var tCoreDocument = reflect.TypeOf(bsoncore.Document{})
var tCoreArray = reflect.TypeOf(bsoncore.Array{})
