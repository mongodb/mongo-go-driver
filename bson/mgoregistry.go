// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"errors"
	"reflect"
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

func newMgoRegistryBuilder() *RegistryBuilder {
	structcodec := &structCodec{
		parser:                  DefaultStructTagParser,
		decodeZeroStruct:        true,
		encodeOmitDefaultStruct: true,
		allowUnexportedFields:   true,
	}
	mapCodec := &mapCodec{
		decodeZerosMap:         true,
		encodeNilAsEmpty:       true,
		encodeKeysWithStringer: true,
	}
	intcodec := func() ValueEncoder { return &intCodec{encodeUintToMinSize: true} }

	return NewRegistryBuilder().
		RegisterTypeDecoder(tEmpty, func() ValueDecoder { return &emptyInterfaceCodec{decodeBinaryAsSlice: true} }).
		RegisterKindDecoder(reflect.String, func() ValueDecoder { return &stringCodec{} }).
		RegisterKindDecoder(reflect.Struct, func() ValueDecoder { return structcodec }).
		RegisterKindDecoder(reflect.Map, func() ValueDecoder { return mapCodec }).
		RegisterTypeEncoder(tByteSlice, func() ValueEncoder { return &byteSliceCodec{encodeNilAsEmpty: true} }).
		RegisterKindEncoder(reflect.Struct, func() ValueEncoder { return structcodec }).
		RegisterKindEncoder(reflect.Slice, func() ValueEncoder { return &sliceCodec{encodeNilAsEmpty: true} }).
		RegisterKindEncoder(reflect.Map, func() ValueEncoder { return mapCodec }).
		RegisterKindEncoder(reflect.Uint, intcodec).
		RegisterKindEncoder(reflect.Uint8, intcodec).
		RegisterKindEncoder(reflect.Uint16, intcodec).
		RegisterKindEncoder(reflect.Uint32, intcodec).
		RegisterKindEncoder(reflect.Uint64, intcodec).
		RegisterTypeMapEntry(TypeInt32, tInt).
		RegisterTypeMapEntry(TypeDateTime, tTime).
		RegisterTypeMapEntry(TypeArray, tInterfaceSlice).
		RegisterTypeMapEntry(Type(0), tM).
		RegisterTypeMapEntry(TypeEmbeddedDocument, tM).
		RegisterInterfaceEncoder(tGetter, func() ValueEncoder { return ValueEncoderFunc(GetterEncodeValue) }).
		RegisterInterfaceDecoder(tSetter, func() ValueDecoder { return ValueDecoderFunc(SetterDecodeValue) })
}

// NewMgoRegistry creates a new bson.Registry configured with the default encoders and decoders.
func NewMgoRegistry() *Registry {
	return newMgoRegistryBuilder().Build()
}

// NewRespectNilValuesMgoRegistry creates a new bson.Registry configured to behave like mgo/bson
// with RespectNilValues set to true.
func NewRespectNilValuesMgoRegistry() *Registry {
	mapCodec := &mapCodec{
		decodeZerosMap: true,
	}

	return newMgoRegistryBuilder().
		RegisterKindDecoder(reflect.Map, func() ValueDecoder { return mapCodec }).
		RegisterTypeEncoder(tByteSlice, func() ValueEncoder { return &byteSliceCodec{encodeNilAsEmpty: false} }).
		RegisterKindEncoder(reflect.Slice, func() ValueEncoder { return &sliceCodec{} }).
		RegisterKindEncoder(reflect.Map, func() ValueEncoder { return mapCodec }).
		Build()
}
