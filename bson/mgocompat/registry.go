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
var Registry = NewRegistryBuilder().Build()

// RespectNilValuesRegistry is the bson.Registry compatible with mgo withSetRespectNilValues set to true.
var RespectNilValuesRegistry = NewRespectNilValuesRegistryBuilder().Build()

// NewRegistryBuilder creates a new bson.RegistryBuilder configured with the default encoders and
// decoders from the bson.DefaultValueEncoders and bson.DefaultValueDecoders types and the
// PrimitiveCodecs type in this package.
func NewRegistryBuilder() *bson.RegistryBuilder {
	rb := bson.NewRegistryBuilder()
	bson.DefaultValueEncoders{}.RegisterDefaultEncoders(rb)
	bson.DefaultValueDecoders{}.RegisterDefaultDecoders(rb)
	bson.PrimitiveCodecs{}.RegisterPrimitiveCodecs(rb)

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

	rb.RegisterTypeDecoder(tEmpty, emptyInterCodec).
		RegisterDefaultDecoder(reflect.String, bson.NewStringCodec(bsonoptions.StringCodec().SetDecodeObjectIDAsHex(false))).
		RegisterDefaultDecoder(reflect.Struct, structcodec).
		RegisterDefaultDecoder(reflect.Map, mapCodec).
		RegisterTypeEncoder(tByteSlice, bson.NewByteSliceCodec(bsonoptions.ByteSliceCodec().SetEncodeNilAsEmpty(true))).
		RegisterDefaultEncoder(reflect.Struct, structcodec).
		RegisterDefaultEncoder(reflect.Slice, bson.NewSliceCodec(bsonoptions.SliceCodec().SetEncodeNilAsEmpty(true))).
		RegisterDefaultEncoder(reflect.Map, mapCodec).
		RegisterDefaultEncoder(reflect.Uint, uintcodec).
		RegisterDefaultEncoder(reflect.Uint8, uintcodec).
		RegisterDefaultEncoder(reflect.Uint16, uintcodec).
		RegisterDefaultEncoder(reflect.Uint32, uintcodec).
		RegisterDefaultEncoder(reflect.Uint64, uintcodec).
		RegisterTypeMapEntry(bson.TypeInt32, tInt).
		RegisterTypeMapEntry(bson.TypeDateTime, tTime).
		RegisterTypeMapEntry(bson.TypeArray, tInterfaceSlice).
		RegisterTypeMapEntry(bson.Type(0), tM).
		RegisterTypeMapEntry(bson.TypeEmbeddedDocument, tM).
		RegisterHookEncoder(tGetter, bson.ValueEncoderFunc(GetterEncodeValue)).
		RegisterHookDecoder(tSetter, bson.ValueDecoderFunc(SetterDecodeValue))

	return rb
}

// NewRespectNilValuesRegistryBuilder creates a new bson.RegistryBuilder configured to behave like mgo/bson
// with RespectNilValues set to true.
func NewRespectNilValuesRegistryBuilder() *bson.RegistryBuilder {
	rb := NewRegistryBuilder()

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

	rb.RegisterDefaultDecoder(reflect.Struct, structcodec).
		RegisterDefaultDecoder(reflect.Map, mapCodec).
		RegisterTypeEncoder(tByteSlice, bson.NewByteSliceCodec(bsonoptions.ByteSliceCodec().SetEncodeNilAsEmpty(false))).
		RegisterDefaultEncoder(reflect.Struct, structcodec).
		RegisterDefaultEncoder(reflect.Slice, bson.NewSliceCodec(bsonoptions.SliceCodec().SetEncodeNilAsEmpty(false))).
		RegisterDefaultEncoder(reflect.Map, mapCodec)

	return rb
}
