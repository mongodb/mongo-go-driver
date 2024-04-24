// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mgocompat

import (
	"errors"
	"reflect"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsonoptions"
)

var (
	// ErrSetZero may be returned from a SetBSON method to have the value set to its respective zero value.
	ErrSetZero = errors.New("set to zero")

	tInt            = reflect.TypeOf(int(0))
	tTime           = reflect.TypeOf(time.Time{})
	tM              = reflect.TypeOf(bson.M{})
	tInterfaceSlice = reflect.TypeOf([]interface{}{})
	tByteSlice      = reflect.TypeOf([]byte{})
	tEmpty          = reflect.TypeOf((*interface{})(nil)).Elem()
	tGetter         = reflect.TypeOf((*Getter)(nil)).Elem()
	tSetter         = reflect.TypeOf((*Setter)(nil)).Elem()
)

// Registry is the mgo compatible bson.Registry. It contains the default and
// primitive codecs with mgo compatible options.
var Registry = newRegistry()

// RespectNilValuesRegistry is the bson.Registry compatible with mgo withSetRespectNilValues set to true.
var RespectNilValuesRegistry = newRespectNilValuesRegistry()

// newRegistry creates a new bson.Registry configured with the default encoders and decoders.
func newRegistry() *bson.Registry {
	reg := bson.NewRegistry()

	structcodec, _ := bson.NewStructCodec(bson.DefaultStructTagParser,
		bsonoptions.StructCodec().
			SetDecodeZeroStruct(true).
			SetEncodeOmitDefaultStruct(true).
			SetOverwriteDuplicatedInlinedFields(false).
			SetAllowUnexportedFields(true))
	emptyInterCodec := bson.NewEmptyInterfaceCodec(
		bsonoptions.EmptyInterfaceCodec().
			SetDecodeBinaryAsSlice(true))
	mapCodec := bson.NewMapCodec(
		bsonoptions.MapCodec().
			SetDecodeZerosMap(true).
			SetEncodeNilAsEmpty(true).
			SetEncodeKeysWithStringer(true))
	uintcodec := bson.NewUIntCodec(bsonoptions.UIntCodec().SetEncodeToMinSize(true))

	reg.RegisterTypeDecoder(tEmpty, emptyInterCodec)
	reg.RegisterKindDecoder(reflect.String, bson.NewStringCodec(bsonoptions.StringCodec().SetDecodeObjectIDAsHex(false)))
	reg.RegisterKindDecoder(reflect.Struct, structcodec)
	reg.RegisterKindDecoder(reflect.Map, mapCodec)
	reg.RegisterTypeEncoder(tByteSlice, bson.NewByteSliceCodec(bsonoptions.ByteSliceCodec().SetEncodeNilAsEmpty(true)))
	reg.RegisterKindEncoder(reflect.Struct, structcodec)
	reg.RegisterKindEncoder(reflect.Slice, bson.NewSliceCodec(bsonoptions.SliceCodec().SetEncodeNilAsEmpty(true)))
	reg.RegisterKindEncoder(reflect.Map, mapCodec)
	reg.RegisterKindEncoder(reflect.Uint, uintcodec)
	reg.RegisterKindEncoder(reflect.Uint8, uintcodec)
	reg.RegisterKindEncoder(reflect.Uint16, uintcodec)
	reg.RegisterKindEncoder(reflect.Uint32, uintcodec)
	reg.RegisterKindEncoder(reflect.Uint64, uintcodec)
	reg.RegisterTypeMapEntry(bson.TypeInt32, tInt)
	reg.RegisterTypeMapEntry(bson.TypeDateTime, tTime)
	reg.RegisterTypeMapEntry(bson.TypeArray, tInterfaceSlice)
	reg.RegisterTypeMapEntry(bson.Type(0), tM)
	reg.RegisterTypeMapEntry(bson.TypeEmbeddedDocument, tM)
	reg.RegisterInterfaceEncoder(tGetter, bson.ValueEncoderFunc(GetterEncodeValue))
	reg.RegisterInterfaceDecoder(tSetter, bson.ValueDecoderFunc(SetterDecodeValue))

	return reg
}

// newRespectNilValuesRegistry creates a new bson.Registry configured to behave like mgo/bson
// with RespectNilValues set to true.
func newRespectNilValuesRegistry() *bson.Registry {
	reg := newRegistry()

	structcodec, _ := bson.NewStructCodec(bson.DefaultStructTagParser,
		bsonoptions.StructCodec().
			SetDecodeZeroStruct(true).
			SetEncodeOmitDefaultStruct(true).
			SetOverwriteDuplicatedInlinedFields(false).
			SetAllowUnexportedFields(true))
	mapCodec := bson.NewMapCodec(
		bsonoptions.MapCodec().
			SetDecodeZerosMap(true).
			SetEncodeNilAsEmpty(false))

	reg.RegisterKindDecoder(reflect.Struct, structcodec)
	reg.RegisterKindDecoder(reflect.Map, mapCodec)
	reg.RegisterTypeEncoder(tByteSlice, bson.NewByteSliceCodec(bsonoptions.ByteSliceCodec().SetEncodeNilAsEmpty(false)))
	reg.RegisterKindEncoder(reflect.Struct, structcodec)
	reg.RegisterKindEncoder(reflect.Slice, bson.NewSliceCodec(bsonoptions.SliceCodec().SetEncodeNilAsEmpty(false)))
	reg.RegisterKindEncoder(reflect.Map, mapCodec)

	return reg
}
