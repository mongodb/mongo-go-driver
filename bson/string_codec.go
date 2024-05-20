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
type stringCodec struct {
	// decodeObjectIDAsHex specifies if object IDs should be decoded as their hex representation.
	// If false, a string made from the raw object ID bytes will be used. Defaults to true.
	decodeObjectIDAsHex bool
}

// EncodeValue is the ValueEncoder for string types.
func (sc *stringCodec) EncodeValue(_ EncoderRegistry, vw ValueWriter, val reflect.Value) error {
	if val.Kind() != reflect.String {
		return ValueEncoderError{
			Name:     "StringEncodeValue",
			Kinds:    []reflect.Kind{reflect.String},
			Received: val,
		}
	}

	return vw.WriteString(val.String())
}

func (sc *stringCodec) decodeType(_ DecoderRegistry, vr ValueReader, t reflect.Type) (reflect.Value, error) {
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
		oid, err := vr.ReadObjectID()
		if err != nil {
			return emptyValue, err
		}
		if !sc.decodeObjectIDAsHex {
			return emptyValue, errors.New("cannot decode ObjectID as string if DecodeObjectIDAsHex is not set")
		}
		str = oid.Hex()
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
func (sc *stringCodec) DecodeValue(reg DecoderRegistry, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Kind() != reflect.String {
		return ValueDecoderError{Name: "StringDecodeValue", Kinds: []reflect.Kind{reflect.String}, Received: val}
	}

	elem, err := sc.decodeType(reg, vr, val.Type())
	if err != nil {
		return err
	}

	val.SetString(elem.String())
	return nil
}
