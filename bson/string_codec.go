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

// stringCodec is the Codec used for string values.
type stringCodec struct{}

// Assert that stringCodec satisfies the typeDecoder interface, which allows it to be
// used by collection type decoders (e.g. map, slice, etc) to set individual values in a
// collection.
var _ typeDecoder = &stringCodec{}

// EncodeValue is the ValueEncoder for string types.
func (sc *stringCodec) EncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	if val.Kind() != reflect.String {
		return ValueEncoderError{
			Name:     "StringEncodeValue",
			Kinds:    []reflect.Kind{reflect.String},
			Received: val,
		}
	}

	return vw.WriteString(val.String())
}

func (sc *stringCodec) decodeType(dc DecodeContext, vr ValueReader, t reflect.Type) (reflect.Value, error) {
	if t.Kind() != reflect.String {
		return emptyValue, ValueDecoderError{
			Name:     "StringDecodeValue",
			Kinds:    []reflect.Kind{reflect.String},
			Received: reflect.Zero(t),
		}
	}

	var str string
	var err error
	switch vr.Type() {
	case TypeString:
		str, err = vr.ReadString()
		if err != nil {
			return emptyValue, err
		}
	case TypeObjectID:
		if dc.objectIDAsHexString {
			oid, err := vr.ReadObjectID()
			if err != nil {
				return emptyValue, err
			}
			str = oid.Hex()
		} else {
			const msg = "decoding an object ID into a string is not supported by default " +
				"(set Decoder.ObjectIDAsHexString to enable decoding as a hexadecimal string)"
			return emptyValue, errors.New(msg)
		}
	case TypeSymbol:
		str, err = vr.ReadSymbol()
		if err != nil {
			return emptyValue, err
		}
	case TypeBinary:
		data, subtype, err := vr.ReadBinary()
		if err != nil {
			return emptyValue, err
		}
		if subtype != TypeBinaryGeneric && subtype != TypeBinaryBinaryOld {
			return emptyValue, decodeBinaryError{subtype: subtype, typeName: "string"}
		}
		str = string(data)
	case TypeNull:
		if err = vr.ReadNull(); err != nil {
			return emptyValue, err
		}
	case TypeUndefined:
		if err = vr.ReadUndefined(); err != nil {
			return emptyValue, err
		}
	default:
		return emptyValue, fmt.Errorf("cannot decode %v into a string type", vr.Type())
	}

	return reflect.ValueOf(str), nil
}

// DecodeValue is the ValueDecoder for string types.
func (sc *stringCodec) DecodeValue(dctx DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Kind() != reflect.String {
		return ValueDecoderError{Name: "StringDecodeValue", Kinds: []reflect.Kind{reflect.String}, Received: val}
	}

	elem, err := sc.decodeType(dctx, vr, val.Type())
	if err != nil {
		return err
	}

	val.SetString(elem.String())
	return nil
}
