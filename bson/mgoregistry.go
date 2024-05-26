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
	mapCodec := &mapCodec{
		decodeZerosMap:         true,
		encodeNilAsEmpty:       true,
		encodeKeysWithStringer: true,
	}
	newStructCodec := func(elemEncoder mapElementsEncoder) *structCodec {
		return &structCodec{
			elemEncoder:             elemEncoder,
			decodeZeroStruct:        true,
			encodeOmitDefaultStruct: true,
			allowUnexportedFields:   true,
		}
	}
	numcodecFac := func(*Registry) ValueEncoder { return &numCodec{encodeUintToMinSize: true} }

	return NewRegistryBuilder().
		RegisterTypeDecoder(tEmpty, func(*Registry) ValueDecoder { return &emptyInterfaceCodec{decodeBinaryAsSlice: true} }).
		RegisterKindDecoder(reflect.Struct, func(*Registry) ValueDecoder { return newStructCodec(nil) }).
		RegisterKindDecoder(reflect.Map, func(*Registry) ValueDecoder { return mapCodec }).
		RegisterTypeEncoder(tByteSlice, func(*Registry) ValueEncoder { return &byteSliceCodec{encodeNilAsEmpty: true} }).
		RegisterKindEncoder(reflect.Struct, func(reg *Registry) ValueEncoder {
			enc, _ := reg.lookupKindEncoder(reflect.Map)
			return newStructCodec(enc.(mapElementsEncoder))
		}).
		RegisterKindEncoder(reflect.Slice, func(*Registry) ValueEncoder { return &sliceCodec{encodeNilAsEmpty: true} }).
		RegisterKindEncoder(reflect.Map, func(*Registry) ValueEncoder { return mapCodec }).
		RegisterKindEncoder(reflect.Uint, numcodecFac).
		RegisterKindEncoder(reflect.Uint8, numcodecFac).
		RegisterKindEncoder(reflect.Uint16, numcodecFac).
		RegisterKindEncoder(reflect.Uint32, numcodecFac).
		RegisterKindEncoder(reflect.Uint64, numcodecFac).
		RegisterTypeMapEntry(TypeInt32, tInt).
		RegisterTypeMapEntry(TypeDateTime, tTime).
		RegisterTypeMapEntry(TypeArray, tInterfaceSlice).
		RegisterTypeMapEntry(Type(0), tM).
		RegisterTypeMapEntry(TypeEmbeddedDocument, tM).
		RegisterInterfaceEncoder(tGetter, func(*Registry) ValueEncoder { return ValueEncoderFunc(GetterEncodeValue) }).
		RegisterInterfaceDecoder(tSetter, func(*Registry) ValueDecoder { return ValueDecoderFunc(SetterDecodeValue) })
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
		RegisterKindDecoder(reflect.Map, func(*Registry) ValueDecoder { return mapCodec }).
		RegisterTypeEncoder(tByteSlice, func(*Registry) ValueEncoder { return &byteSliceCodec{encodeNilAsEmpty: false} }).
		RegisterKindEncoder(reflect.Slice, func(*Registry) ValueEncoder { return &sliceCodec{} }).
		RegisterKindEncoder(reflect.Map, func(*Registry) ValueEncoder { return mapCodec }).
		Build()
}
