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

	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

// BSON element types as described in https://bsonspec.org/spec.html.
const (
	TypeDouble           = bsontype.Double
	TypeString           = bsontype.String
	TypeEmbeddedDocument = bsontype.EmbeddedDocument
	TypeArray            = bsontype.Array
	TypeBinary           = bsontype.Binary
	TypeUndefined        = bsontype.Undefined
	TypeObjectID         = bsontype.ObjectID
	TypeBoolean          = bsontype.Boolean
	TypeDateTime         = bsontype.DateTime
	TypeNull             = bsontype.Null
	TypeRegex            = bsontype.Regex
	TypeDBPointer        = bsontype.DBPointer
	TypeJavaScript       = bsontype.JavaScript
	TypeSymbol           = bsontype.Symbol
	TypeCodeWithScope    = bsontype.CodeWithScope
	TypeInt32            = bsontype.Int32
	TypeTimestamp        = bsontype.Timestamp
	TypeInt64            = bsontype.Int64
	TypeDecimal128       = bsontype.Decimal128
	TypeMinKey           = bsontype.MinKey
	TypeMaxKey           = bsontype.MaxKey
)

// BSON binary element subtypes as described in https://bsonspec.org/spec.html.
const (
	TypeBinaryGeneric     = bsontype.BinaryGeneric
	TypeBinaryFunction    = bsontype.BinaryFunction
	TypeBinaryBinaryOld   = bsontype.BinaryBinaryOld
	TypeBinaryUUIDOld     = bsontype.BinaryUUIDOld
	TypeBinaryUUID        = bsontype.BinaryUUID
	TypeBinaryMD5         = bsontype.BinaryMD5
	TypeBinaryEncrypted   = bsontype.BinaryEncrypted
	TypeBinaryColumn      = bsontype.BinaryColumn
	TypeBinarySensitive   = bsontype.BinarySensitive
	TypeBinaryUserDefined = bsontype.BinaryUserDefined
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
