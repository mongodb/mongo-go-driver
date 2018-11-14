// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncodec

import (
	"encoding/json"
	"net/url"
	"reflect"
	"time"

	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

var ptBool = reflect.TypeOf((*bool)(nil))
var ptInt8 = reflect.TypeOf((*int8)(nil))
var ptInt16 = reflect.TypeOf((*int16)(nil))
var ptInt32 = reflect.TypeOf((*int32)(nil))
var ptInt64 = reflect.TypeOf((*int64)(nil))
var ptInt = reflect.TypeOf((*int)(nil))
var ptUint8 = reflect.TypeOf((*uint8)(nil))
var ptUint16 = reflect.TypeOf((*uint16)(nil))
var ptUint32 = reflect.TypeOf((*uint32)(nil))
var ptUint64 = reflect.TypeOf((*uint64)(nil))
var ptUint = reflect.TypeOf((*uint)(nil))
var ptFloat32 = reflect.TypeOf((*float32)(nil))
var ptFloat64 = reflect.TypeOf((*float64)(nil))
var ptString = reflect.TypeOf((*string)(nil))

var tBool = reflect.TypeOf(false)
var tDecimal = reflect.TypeOf(decimal.Decimal128{})
var tFloat32 = reflect.TypeOf(float32(0))
var tFloat64 = reflect.TypeOf(float64(0))
var tInt = reflect.TypeOf(int(0))
var tInt8 = reflect.TypeOf(int8(0))
var tInt16 = reflect.TypeOf(int16(0))
var tInt32 = reflect.TypeOf(int32(0))
var tInt64 = reflect.TypeOf(int64(0))
var tOID = reflect.TypeOf(objectid.ObjectID{})
var tString = reflect.TypeOf("")
var tTime = reflect.TypeOf(time.Time{})
var tUint = reflect.TypeOf(uint(0))
var tUint8 = reflect.TypeOf(uint8(0))
var tUint16 = reflect.TypeOf(uint16(0))
var tUint32 = reflect.TypeOf(uint32(0))
var tUint64 = reflect.TypeOf(uint64(0))

var tEmpty = reflect.TypeOf((*interface{})(nil)).Elem()
var tByteSlice = reflect.TypeOf([]byte(nil))
var tByte = reflect.TypeOf(byte(0x00))
var tURL = reflect.TypeOf(url.URL{})
var tJSONNumber = reflect.TypeOf(json.Number(""))

var tValueMarshaler = reflect.TypeOf((*ValueMarshaler)(nil)).Elem()
var tValueUnmarshaler = reflect.TypeOf((*ValueUnmarshaler)(nil)).Elem()
var tProxy = reflect.TypeOf((*Proxy)(nil)).Elem()
