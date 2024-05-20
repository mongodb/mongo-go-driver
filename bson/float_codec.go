// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"fmt"
	"reflect"
)

type floatCodec struct {
	// truncate, if true, instructs decoders to to truncate the fractional part of BSON "double"
	// values when attempting to unmarshal them into a Go float struct field. The truncation logic
	// does not apply to BSON "decimal128" values.
	truncate bool
}

// floatEncodeValue is the ValueEncoderFunc for float types.
func (fc *floatCodec) EncodeValue(_ EncoderRegistry, vw ValueWriter, val reflect.Value) error {
	switch val.Kind() {
	case reflect.Float32, reflect.Float64:
		return vw.WriteDouble(val.Float())
	}

	return ValueEncoderError{Name: "FloatEncodeValue", Kinds: []reflect.Kind{reflect.Float32, reflect.Float64}, Received: val}
}

func (fc *floatCodec) floatDecodeType(_ DecoderRegistry, vr ValueReader, t reflect.Type) (reflect.Value, error) {
	var f float64
	var err error
	switch vrType := vr.Type(); vrType {
	case TypeInt32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return emptyValue, err
		}
		f = float64(i32)
	case TypeInt64:
		i64, err := vr.ReadInt64()
		if err != nil {
			return emptyValue, err
		}
		f = float64(i64)
	case TypeDouble:
		f, err = vr.ReadDouble()
		if err != nil {
			return emptyValue, err
		}
	case TypeBoolean:
		b, err := vr.ReadBoolean()
		if err != nil {
			return emptyValue, err
		}
		if b {
			f = 1
		}
	case TypeNull:
		if err = vr.ReadNull(); err != nil {
			return emptyValue, err
		}
	case TypeUndefined:
		if err = vr.ReadUndefined(); err != nil {
			return emptyValue, err
		}
	default:
		return emptyValue, fmt.Errorf("cannot decode %v into a float32 or float64 type", vrType)
	}

	switch t.Kind() {
	case reflect.Float32:
		if !fc.truncate && float64(float32(f)) != f {
			return emptyValue, errCannotTruncate
		}

		return reflect.ValueOf(float32(f)), nil
	case reflect.Float64:
		return reflect.ValueOf(f), nil
	default:
		return emptyValue, ValueDecoderError{
			Name:     "FloatDecodeValue",
			Kinds:    []reflect.Kind{reflect.Float32, reflect.Float64},
			Received: reflect.Zero(t),
		}
	}
}

// DecodeValue is the ValueDecoder for float types.
func (fc *floatCodec) DecodeValue(reg DecoderRegistry, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() {
		return ValueDecoderError{
			Name:     "FloatDecodeValue",
			Kinds:    []reflect.Kind{reflect.Float32, reflect.Float64},
			Received: val,
		}
	}

	elem, err := fc.floatDecodeType(reg, vr, val.Type())
	if err != nil {
		return err
	}

	val.SetFloat(elem.Float())
	return nil
}
