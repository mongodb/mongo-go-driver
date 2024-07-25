// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"fmt"
	"reflect"
)

var tRawValue = reflect.TypeOf(RawValue{})
var tRaw = reflect.TypeOf(Raw(nil))

// registerPrimitiveCodecs will register the encode and decode methods attached to PrimitiveCodecs
// with the provided RegistryBuilder. if rb is nil, a new empty RegistryBuilder will be created.
func registerPrimitiveCodecs(reg *Registry) {
	reg.RegisterTypeEncoder(tRawValue, ValueEncoderFunc(rawValueEncodeValue))
	reg.RegisterTypeEncoder(tRaw, ValueEncoderFunc(rawEncodeValue))
	reg.RegisterTypeDecoder(tRawValue, ValueDecoderFunc(rawValueDecodeValue))
	reg.RegisterTypeDecoder(tRaw, ValueDecoderFunc(rawDecodeValue))
}

// rawValueEncodeValue is the ValueEncoderFunc for RawValue.
//
// If the RawValue's Type is "invalid" and the RawValue's Value is not empty or
// nil, then this method will return an error.
func rawValueEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
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

// rawValueDecodeValue is the ValueDecoderFunc for RawValue.
func rawValueDecodeValue(_ DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tRawValue {
		return ValueDecoderError{Name: "RawValueDecodeValue", Types: []reflect.Type{tRawValue}, Received: val}
	}

	t, value, err := copyValueToBytes(vr)
	if err != nil {
		return err
	}

	val.Set(reflect.ValueOf(RawValue{Type: t, Value: value}))
	return nil
}

// rawEncodeValue is the ValueEncoderFunc for Reader.
func rawEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tRaw {
		return ValueEncoderError{Name: "RawEncodeValue", Types: []reflect.Type{tRaw}, Received: val}
	}

	rdr := val.Interface().(Raw)

	return copyDocumentFromBytes(vw, rdr)
}

// rawDecodeValue is the ValueDecoderFunc for Reader.
func rawDecodeValue(_ DecodeContext, vr ValueReader, val reflect.Value) error {
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
