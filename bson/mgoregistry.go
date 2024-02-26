// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// MgoRegistry is a BSON registry compatible with globalsign/mgo's BSON, with some remaining
// differences. This file also provides MgoRespectNilValuesRegistry for compatibility with
// mgo's BSON with RespectNilValues set to true. A registry can be configured on a mongo.Client
// with the SetRegistry option. See the bsoncodec docs for more details on registries.
//
// Registry supports Getter and Setter equivalents by registering hooks. Note that if a value
// matches the hook for bsoncodec.Marshaler, bsoncodec.ValueMarshaler, or bsoncodec.Proxy, that
// hook will take priority over the Getter hook. The same is true for the hooks for
// bsoncodec.Unmarshaler and bsoncodec.ValueUnmarshaler and the Setter hook.
//
// The functional differences between Registry and globalsign/mgo's BSON library are:
//
// 1) Registry errors instead of silently skipping mismatched types when decoding.
//
// 2) Registry does not have special handling for marshaling array ops ("$in", "$nin", "$all").
//
// The driver uses different types than mgo's bson. The differences are:
//
//  1. The driver's bson.RawValue is equivalent to mgo's bson.Raw, but uses Value instead of Data
//     and uses Type, which wraps a byte, instead of bson.Raw's Kind, a byte.
//
//  2. The driver uses primitive.ObjectID, which is a [12]byte instead of mgo's
//     bson.ObjectId, a string. Due to this, the zero value marshals and unmarshals differently
//     for Extended JSON, with the driver marshaling as {"ID":"000000000000000000000000"} and
//     mgo as {"Id":""}. The driver can unmarshal {"ID":""} to a primitive.ObjectID.
//
//  3. The driver's primitive.Symbol is equivalent to mgo's bson.Symbol.
//
//  4. The driver uses primitive.Timestamp instead of mgo's bson.MongoTimestamp. While
//     MongoTimestamp is an int64, primitive.Timestamp stores the time and counter as two separate
//     uint32 values, T and I respectively.
//
//  5. The driver uses primitive.MinKey and primitive.MaxKey, which are struct{}, instead
//     of mgo's bson.MinKey and bson.MaxKey, which are int64.
//
//  6. The driver's primitive.Undefined is equivalent to mgo's bson.Undefined.
//
//  7. The driver's primitive.Binary is equivalent to mgo's bson.Binary, with variables named Subtype
//     and Data instead of Kind and Data.
//
//  8. The driver's primitive.Regex is equivalent to mgo's bson.RegEx.
//
//  9. The driver's primitive.JavaScript is equivalent to mgo's bson.JavaScript with no
//     scope and primitive.CodeWithScope is equivalent to mgo's bson.JavaScript with scope.
//
//  10. The driver's primitive.DBPointer is equivalent to mgo's bson.DBPointer, with variables
//     named DB and Pointer instead of Namespace and Id.
//
//  11. When implementing the Setter interface, ErrSetZero is equivalent to mgo's bson.ErrSetZero.

package bson

import (
	"errors"
	"reflect"

	"go.mongodb.org/mongo-driver/bson/bsonoptions"
)

var (
	// ErrSetZero may be returned from a SetBSON method to have the value set to its respective zero value.
	ErrSetZero = errors.New("set to zero")

	tInt            = reflect.TypeOf(int(0))
	tM              = reflect.TypeOf(M{})
	tInterfaceSlice = reflect.TypeOf([]interface{}{})
	tGetter         = reflect.TypeOf((*Getter)(nil)).Elem()
	tSetter         = reflect.TypeOf((*Setter)(nil)).Elem()
)

// MgoRegistry is the mgo compatible Registry. It contains the default and
// primitive codecs with mgo compatible options.
var MgoRegistry = NewMgoRegistryBuilder().Build()

// MgoRespectNilValuesRegistry is the Registry compatible with mgo withSetRespectNilValues set to true.
var MgoRespectNilValuesRegistry = NewMgoRespectNilValuesRegistryBuilder().Build()

// NewMgoRegistryBuilder creates a new RegistryBuilder configured with the default encoders and
// decoders from the DefaultValueEncoders and DefaultValueDecoders types and the PrimitiveCodecs
// type in this package.
func NewMgoRegistryBuilder() *RegistryBuilder {
	rb := NewRegistryBuilder()
	DefaultValueEncoders{}.RegisterDefaultEncoders(rb)
	DefaultValueDecoders{}.RegisterDefaultDecoders(rb)
	PrimitiveCodecs{}.RegisterPrimitiveCodecs(rb)

	structcodec, _ := NewStructCodec(DefaultStructTagParser,
		bsonoptions.StructCodec().
			SetDecodeZeroStruct(true).
			SetEncodeOmitDefaultStruct(true).
			SetOverwriteDuplicatedInlinedFields(false).
			SetAllowUnexportedFields(true))
	emptyInterCodec := NewEmptyInterfaceCodec(
		bsonoptions.EmptyInterfaceCodec().
			SetDecodeBinaryAsSlice(true))
	mapCodec := NewMapCodec(
		bsonoptions.MapCodec().
			SetDecodeZerosMap(true).
			SetEncodeNilAsEmpty(true).
			SetEncodeKeysWithStringer(true))
	uintcodec := NewUIntCodec(bsonoptions.UIntCodec().SetEncodeToMinSize(true))

	rb.RegisterTypeDecoder(tEmpty, emptyInterCodec).
		RegisterDefaultDecoder(reflect.String, NewStringCodec(bsonoptions.StringCodec().SetDecodeObjectIDAsHex(false))).
		RegisterDefaultDecoder(reflect.Struct, structcodec).
		RegisterDefaultDecoder(reflect.Map, mapCodec).
		RegisterTypeEncoder(tByteSlice, NewByteSliceCodec(bsonoptions.ByteSliceCodec().SetEncodeNilAsEmpty(true))).
		RegisterDefaultEncoder(reflect.Struct, structcodec).
		RegisterDefaultEncoder(reflect.Slice, NewSliceCodec(bsonoptions.SliceCodec().SetEncodeNilAsEmpty(true))).
		RegisterDefaultEncoder(reflect.Map, mapCodec).
		RegisterDefaultEncoder(reflect.Uint, uintcodec).
		RegisterDefaultEncoder(reflect.Uint8, uintcodec).
		RegisterDefaultEncoder(reflect.Uint16, uintcodec).
		RegisterDefaultEncoder(reflect.Uint32, uintcodec).
		RegisterDefaultEncoder(reflect.Uint64, uintcodec).
		RegisterTypeMapEntry(TypeInt32, tInt).
		RegisterTypeMapEntry(TypeDateTime, tTime).
		RegisterTypeMapEntry(TypeArray, tInterfaceSlice).
		RegisterTypeMapEntry(Type(0), tM).
		RegisterTypeMapEntry(TypeEmbeddedDocument, tM).
		RegisterHookEncoder(tGetter, ValueEncoderFunc(GetterEncodeValue)).
		RegisterHookDecoder(tSetter, ValueDecoderFunc(SetterDecodeValue))

	return rb
}

// NewMgoRespectNilValuesRegistryBuilder creates a new RegistryBuilder configured to behave like mgo/bson
// with RespectNilValues set to true.
func NewMgoRespectNilValuesRegistryBuilder() *RegistryBuilder {
	rb := NewMgoRegistryBuilder()

	structcodec, _ := NewStructCodec(DefaultStructTagParser,
		bsonoptions.StructCodec().
			SetDecodeZeroStruct(true).
			SetEncodeOmitDefaultStruct(true).
			SetOverwriteDuplicatedInlinedFields(false).
			SetAllowUnexportedFields(true))
	mapCodec := NewMapCodec(
		bsonoptions.MapCodec().
			SetDecodeZerosMap(true).
			SetEncodeNilAsEmpty(false))

	rb.RegisterDefaultDecoder(reflect.Struct, structcodec).
		RegisterDefaultDecoder(reflect.Map, mapCodec).
		RegisterTypeEncoder(tByteSlice, NewByteSliceCodec(bsonoptions.ByteSliceCodec().SetEncodeNilAsEmpty(false))).
		RegisterDefaultEncoder(reflect.Struct, structcodec).
		RegisterDefaultEncoder(reflect.Slice, NewSliceCodec(bsonoptions.SliceCodec().SetEncodeNilAsEmpty(false))).
		RegisterDefaultEncoder(reflect.Map, mapCodec)

	return rb
}
