// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"errors"
	"fmt"
	"reflect"
)

var tRawValue = reflect.TypeOf(RawValue{})
var tRaw = reflect.TypeOf(Raw(nil))

// PrimitiveCodecs is a namespace for all of the default Codecs for the primitive types
// defined in this package.
//
// Deprecated: Use bson.NewRegistry to get a registry with all primitive encoders and decoders
// registered.
type PrimitiveCodecs struct{}

// RegisterPrimitiveCodecs will register the encode and decode methods attached to PrimitiveCodecs
// with the provided RegistryBuilder. if rb is nil, a new empty RegistryBuilder will be created.
//
// Deprecated: Use bson.NewRegistry to get a registry with all primitive encoders and decoders
// registered.
func (pc PrimitiveCodecs) RegisterPrimitiveCodecs(rb *RegistryBuilder) {
	if rb == nil {
		panic(errors.New("argument to RegisterPrimitiveCodecs must not be nil"))
	}

	rb.
		RegisterTypeEncoder(tRawValue, ValueEncoderFunc(pc.RawValueEncodeValue)).
		RegisterTypeEncoder(tRaw, ValueEncoderFunc(pc.RawEncodeValue)).
		RegisterTypeDecoder(tRawValue, ValueDecoderFunc(pc.RawValueDecodeValue)).
		RegisterTypeDecoder(tRaw, ValueDecoderFunc(pc.RawDecodeValue))
}

// RawValueEncodeValue is the ValueEncoderFunc for RawValue.
//
// If the RawValue's Type is "invalid" and the RawValue's Value is not empty or
// nil, then this method will return an error.
//
// Deprecated: Use bson.NewRegistry to get a registry with all primitive
// encoders and decoders registered.
func (PrimitiveCodecs) RawValueEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tRawValue {
		return ValueEncoderError{
			Name:     "RawValueEncodeValue",
			Types:    []reflect.Type{tRawValue},
			Received: val,
		}
	}

	rawvalue := val.Interface().(RawValue)

	if !rawvalue.Type.IsValid() {
		return fmt.Errorf("the RawValue Type specifies an invalid BSON type: %#x", byte(rawvalue.Type))
	}

	return copyValueFromBytes(vw, rawvalue.Type, rawvalue.Value)
}

// RawValueDecodeValue is the ValueDecoderFunc for RawValue.
//
// Deprecated: Use bson.NewRegistry to get a registry with all primitive encoders and decoders
// registered.
func (PrimitiveCodecs) RawValueDecodeValue(_ DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tRawValue {
		return ValueDecoderError{Name: "RawValueDecodeValue", Types: []reflect.Type{tRawValue}, Received: val}
	}

	t, value, err := CopyValueToBytes(vr)
	if err != nil {
		return err
	}

	val.Set(reflect.ValueOf(RawValue{Type: t, Value: value}))
	return nil
}

// RawEncodeValue is the ValueEncoderFunc for Reader.
//
// Deprecated: Use bson.NewRegistry to get a registry with all primitive encoders and decoders
// registered.
func (PrimitiveCodecs) RawEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tRaw {
		return ValueEncoderError{Name: "RawEncodeValue", Types: []reflect.Type{tRaw}, Received: val}
	}

	rdr := val.Interface().(Raw)

	return copyDocumentFromBytes(vw, rdr)
}

// RawDecodeValue is the ValueDecoderFunc for Reader.
//
// Deprecated: Use bson.NewRegistry to get a registry with all primitive encoders and decoders
// registered.
func (PrimitiveCodecs) RawDecodeValue(_ DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tRaw {
		return ValueDecoderError{Name: "RawDecodeValue", Types: []reflect.Type{tRaw}, Received: val}
	}

	if val.IsNil() {
		val.Set(reflect.MakeSlice(val.Type(), 0, 0))
	}

	val.SetLen(0)

	rdr, err := appendDocumentBytes(val.Interface().(Raw), vr)
	val.Set(reflect.ValueOf(rdr))
	return err
}
