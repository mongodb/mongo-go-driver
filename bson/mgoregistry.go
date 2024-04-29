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

// NewMgoRegistry creates a new bson.Registry configured with the default encoders and decoders.
func NewMgoRegistry() *Registry {
	reg := NewRegistry()

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
	uintcodec := &uintCodec{encodeToMinSize: true}

	reg.RegisterTypeDecoder(tEmpty, &emptyInterfaceCodec{decodeBinaryAsSlice: true})
	reg.RegisterKindDecoder(reflect.String, &stringCodec{})
	reg.RegisterKindDecoder(reflect.Struct, structcodec)
	reg.RegisterKindDecoder(reflect.Map, mapCodec)
	reg.RegisterTypeEncoder(tByteSlice, &byteSliceCodec{encodeNilAsEmpty: true})
	reg.RegisterKindEncoder(reflect.Struct, structcodec)
	reg.RegisterKindEncoder(reflect.Slice, &sliceCodec{encodeNilAsEmpty: true})
	reg.RegisterKindEncoder(reflect.Map, mapCodec)
	reg.RegisterKindEncoder(reflect.Uint, uintcodec)
	reg.RegisterKindEncoder(reflect.Uint8, uintcodec)
	reg.RegisterKindEncoder(reflect.Uint16, uintcodec)
	reg.RegisterKindEncoder(reflect.Uint32, uintcodec)
	reg.RegisterKindEncoder(reflect.Uint64, uintcodec)
	reg.RegisterTypeMapEntry(TypeInt32, tInt)
	reg.RegisterTypeMapEntry(TypeDateTime, tTime)
	reg.RegisterTypeMapEntry(TypeArray, tInterfaceSlice)
	reg.RegisterTypeMapEntry(Type(0), tM)
	reg.RegisterTypeMapEntry(TypeEmbeddedDocument, tM)
	reg.RegisterInterfaceEncoder(tGetter, ValueEncoderFunc(GetterEncodeValue))
	reg.RegisterInterfaceDecoder(tSetter, ValueDecoderFunc(SetterDecodeValue))

	return reg
}

// NewRespectNilValuesMgoRegistry creates a new bson.Registry configured to behave like mgo/bson
// with RespectNilValues set to true.
func NewRespectNilValuesMgoRegistry() *Registry {
	reg := NewMgoRegistry()

	mapCodec := &mapCodec{
		decodeZerosMap: true,
	}

	reg.RegisterKindDecoder(reflect.Map, mapCodec)
	reg.RegisterTypeEncoder(tByteSlice, &byteSliceCodec{encodeNilAsEmpty: false})
	reg.RegisterKindEncoder(reflect.Slice, &sliceCodec{})
	reg.RegisterKindEncoder(reflect.Map, mapCodec)

	return reg
}
