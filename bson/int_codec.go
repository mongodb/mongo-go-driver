// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"fmt"
	"math"
	"reflect"
)

// intCodec is the Codec used for uint values.
type intCodec struct {
	// encodeToMinSize causes EncodeValue to marshal Go uint values (excluding uint64) as the
	// minimum BSON int size (either 32-bit or 64-bit) that can represent the integer value.
	encodeToMinSize bool

	// truncate, if true, instructs decoders to to truncate the fractional part of BSON "double"
	// values when attempting to unmarshal them into a Go integer (int, int8, int16, int32, int64,
	// uint, uint8, uint16, uint32, or uint64) struct field. The truncation logic does not apply to
	// BSON "decimal128" values.
	truncate bool
}

// EncodeValue is the ValueEncoder for uint types.
func (ic *intCodec) EncodeValue(_ EncoderRegistry, vw ValueWriter, val reflect.Value) error {
	switch val.Kind() {
	case reflect.Int8, reflect.Int16, reflect.Int32:
		return vw.WriteInt32(int32(val.Int()))
	case reflect.Int:
		i64 := val.Int()
		if fitsIn32Bits(i64) {
			return vw.WriteInt32(int32(i64))
		}
		return vw.WriteInt64(i64)
	case reflect.Int64:
		i64 := val.Int()
		if ic.encodeToMinSize && fitsIn32Bits(i64) {
			return vw.WriteInt32(int32(i64))
		}
		return vw.WriteInt64(i64)

	case reflect.Uint8, reflect.Uint16:
		return vw.WriteInt32(int32(val.Uint()))
	case reflect.Uint, reflect.Uint32, reflect.Uint64:
		u64 := val.Uint()

		// If encodeToMinSize is true for a non-uint64 value we should write val as an int32
		useMinSize := ic.encodeToMinSize && val.Kind() != reflect.Uint64

		if u64 <= math.MaxInt32 && useMinSize {
			return vw.WriteInt32(int32(u64))
		}
		if u64 > math.MaxInt64 {
			return fmt.Errorf("%d overflows int64", u64)
		}
		return vw.WriteInt64(int64(u64))
	}

	return ValueEncoderError{
		Name: "IntEncodeValue",
		Kinds: []reflect.Kind{
			reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int,
			reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint,
		},
		Received: val,
	}
}

func (ic *intCodec) decodeType(_ DecoderRegistry, vr ValueReader, t reflect.Type) (reflect.Value, error) {
	var i64 int64
	switch vrType := vr.Type(); vrType {
	case TypeInt32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return emptyValue, err
		}
		i64 = int64(i32)
	case TypeInt64:
		var err error
		i64, err = vr.ReadInt64()
		if err != nil {
			return emptyValue, err
		}
	case TypeDouble:
		f64, err := vr.ReadDouble()
		if err != nil {
			return emptyValue, err
		}
		if !ic.truncate && math.Floor(f64) != f64 {
			return emptyValue, errCannotTruncate
		}
		if f64 > float64(math.MaxInt64) {
			return emptyValue, fmt.Errorf("%g overflows int64", f64)
		}
		i64 = int64(f64)
	case TypeBoolean:
		b, err := vr.ReadBoolean()
		if err != nil {
			return emptyValue, err
		}
		if b {
			i64 = 1
		}
	case TypeNull:
		if err := vr.ReadNull(); err != nil {
			return emptyValue, err
		}
	case TypeUndefined:
		if err := vr.ReadUndefined(); err != nil {
			return emptyValue, err
		}
	default:
		return emptyValue, fmt.Errorf("cannot decode %v into an integer type", vrType)
	}

	switch t.Kind() {
	case reflect.Int8:
		if i64 < math.MinInt8 || i64 > math.MaxInt8 {
			return emptyValue, fmt.Errorf("%d overflows int8", i64)
		}
		return reflect.ValueOf(int8(i64)), nil
	case reflect.Int16:
		if i64 < math.MinInt16 || i64 > math.MaxInt16 {
			return emptyValue, fmt.Errorf("%d overflows int16", i64)
		}
		return reflect.ValueOf(int16(i64)), nil
	case reflect.Int32:
		if i64 < math.MinInt32 || i64 > math.MaxInt32 {
			return emptyValue, fmt.Errorf("%d overflows int32", i64)
		}
		return reflect.ValueOf(int32(i64)), nil
	case reflect.Int64:
		return reflect.ValueOf(i64), nil
	case reflect.Int:
		if int64(int(i64)) != i64 { // Can we fit this inside of an int
			return emptyValue, fmt.Errorf("%d overflows int", i64)
		}
		return reflect.ValueOf(int(i64)), nil

	case reflect.Uint8:
		if i64 < 0 || i64 > math.MaxUint8 {
			return emptyValue, fmt.Errorf("%d overflows uint8", i64)
		}
		return reflect.ValueOf(uint8(i64)), nil
	case reflect.Uint16:
		if i64 < 0 || i64 > math.MaxUint16 {
			return emptyValue, fmt.Errorf("%d overflows uint16", i64)
		}
		return reflect.ValueOf(uint16(i64)), nil
	case reflect.Uint32:
		if i64 < 0 || i64 > math.MaxUint32 {
			return emptyValue, fmt.Errorf("%d overflows uint32", i64)
		}
		return reflect.ValueOf(uint32(i64)), nil
	case reflect.Uint64:
		if i64 < 0 {
			return emptyValue, fmt.Errorf("%d overflows uint64", i64)
		}
		return reflect.ValueOf(uint64(i64)), nil
	case reflect.Uint:
		if i64 < 0 || int64(uint(i64)) != i64 { // Can we fit this inside of an uint
			return emptyValue, fmt.Errorf("%d overflows uint", i64)
		}
		return reflect.ValueOf(uint(i64)), nil

	default:
		return emptyValue, ValueDecoderError{
			Name: "IntDecodeValue",
			Kinds: []reflect.Kind{
				reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int,
				reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint,
			},
			Received: reflect.Zero(t),
		}
	}
}

// DecodeValue is the ValueDecoder for uint types.
func (ic *intCodec) DecodeValue(reg DecoderRegistry, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() {
		return ValueDecoderError{
			Name:     "IntDecodeValue",
			Kinds:    []reflect.Kind{reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint},
			Received: val,
		}
	}

	elem, err := ic.decodeType(reg, vr, val.Type())
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}
