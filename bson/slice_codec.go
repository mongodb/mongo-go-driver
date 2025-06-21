// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"reflect"
)

// sliceCodec is the Codec used for slice values.
type sliceCodec struct {
	// encodeNilAsEmpty causes EncodeValue to marshal nil Go slices as empty BSON arrays instead of
	// BSON null.
	encodeNilAsEmpty bool
}

// decodeVectorBinary handles decoding of BSON Vector binary (subtype 9) into slices.
// It returns errNotAVectorBinary if the binary data is not a Vector binary.
// The method supports decoding into []int8 and []float32 slices.
func (sc *sliceCodec) decodeVectorBinary(vr ValueReader, val reflect.Value) error {
	elemType := val.Type().Elem()

	if elemType != tInt8 && elemType != tFloat32 {
		return errNotAVectorBinary
	}

	data, subtype, err := vr.ReadBinary()
	if err != nil {
		return err
	}

	if subtype != TypeBinaryVector {
		return errNotAVectorBinary
	}

	switch elemType {
	case tInt8:
		int8Slice, err := decodeVectorInt8(data)
		if err != nil {
			return err
		}
		val.Set(reflect.ValueOf(int8Slice))
	case tFloat32:
		float32Slice, err := decodeVectorFloat32(data)
		if err != nil {
			return err
		}
		val.Set(reflect.ValueOf(float32Slice))
	}

	return nil
}

// EncodeValue is the ValueEncoder for slice types.
func (sc *sliceCodec) EncodeValue(ec EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Kind() != reflect.Slice {
		return ValueEncoderError{Name: "SliceEncodeValue", Kinds: []reflect.Kind{reflect.Slice}, Received: val}
	}

	if val.IsNil() && !sc.encodeNilAsEmpty && !ec.nilSliceAsEmpty {
		return vw.WriteNull()
	}

	// Treat []byte as binary data, but skip for []int8 since it's a different type.
	// Even though byte is an alias for uint8 which has the same underlying type as int8,
	// we want to maintain the semantic difference between []byte (binary data) and []int8 (array of integers).
	if val.Type().Elem() == tByte && val.Type() != reflect.TypeOf([]int8{}) {
		byteSlice := make([]byte, val.Len())
		reflect.Copy(reflect.ValueOf(byteSlice), val)
		return vw.WriteBinary(byteSlice)
	}

	// If we have a []E we want to treat it as a document instead of as an array.
	if val.Type() == tD || val.Type().ConvertibleTo(tD) {
		d := val.Convert(tD).Interface().(D)

		dw, err := vw.WriteDocument()
		if err != nil {
			return err
		}

		for _, e := range d {
			err = encodeElement(ec, dw, e)
			if err != nil {
				return err
			}
		}

		return dw.WriteDocumentEnd()
	}

	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	elemType := val.Type().Elem()
	encoder, err := ec.LookupEncoder(elemType)
	if err != nil && elemType.Kind() != reflect.Interface {
		return err
	}

	for idx := 0; idx < val.Len(); idx++ {
		currEncoder, currVal, lookupErr := lookupElementEncoder(ec, encoder, val.Index(idx))
		if lookupErr != nil && !errors.Is(lookupErr, errInvalidValue) {
			return lookupErr
		}

		vw, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if errors.Is(lookupErr, errInvalidValue) {
			err = vw.WriteNull()
			if err != nil {
				return err
			}
			continue
		}

		err = currEncoder.EncodeValue(ec, vw, currVal)
		if err != nil {
			return err
		}
	}
	return aw.WriteArrayEnd()
}

// DecodeValue is the ValueDecoder for slice types.
func (sc *sliceCodec) DecodeValue(dc DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Kind() != reflect.Slice {
		return ValueDecoderError{Name: "SliceDecodeValue", Kinds: []reflect.Kind{reflect.Slice}, Received: val}
	}

	switch vrType := vr.Type(); vrType {
	case TypeArray:
	case TypeNull:
		val.Set(reflect.Zero(val.Type()))
		return vr.ReadNull()
	case TypeUndefined:
		val.Set(reflect.Zero(val.Type()))
		return vr.ReadUndefined()
	case Type(0), TypeEmbeddedDocument:
		if val.Type().Elem() != tE {
			return fmt.Errorf("cannot decode document into %s", val.Type())
		}
	case TypeBinary:
		err := sc.decodeVectorBinary(vr, val)
		if err == nil {
			return nil
		}
		if err != errNotAVectorBinary {
			return err
		}

		if val.Type().Elem() != tByte {
			return fmt.Errorf("SliceDecodeValue can only decode a binary into a byte array, got %v", vrType)
		}
		data, subtype, err := vr.ReadBinary()
		if err != nil {
			return err
		}
		if subtype != TypeBinaryGeneric && subtype != TypeBinaryBinaryOld {
			return fmt.Errorf("SliceDecodeValue can only be used to decode subtype 0x00 or 0x02 for %s, got %v", TypeBinary, subtype)
		}

		if val.IsNil() {
			val.Set(reflect.MakeSlice(val.Type(), 0, len(data)))
		}
		val.SetLen(0)
		val.Set(reflect.AppendSlice(val, reflect.ValueOf(data)))
		return nil
	case TypeString:
		if sliceType := val.Type().Elem(); sliceType != tByte {
			return fmt.Errorf("SliceDecodeValue can only decode a string into a byte array, got %v", sliceType)
		}
		str, err := vr.ReadString()
		if err != nil {
			return err
		}
		byteStr := []byte(str)

		if val.IsNil() {
			val.Set(reflect.MakeSlice(val.Type(), 0, len(byteStr)))
		}
		val.SetLen(0)
		val.Set(reflect.AppendSlice(val, reflect.ValueOf(byteStr)))
		return nil
	default:
		return fmt.Errorf("cannot decode %v into a slice", vrType)
	}

	var elemsFunc func(DecodeContext, ValueReader, reflect.Value) ([]reflect.Value, error)
	switch val.Type().Elem() {
	case tE:
		elemsFunc = decodeD
	default:
		elemsFunc = decodeDefault
	}

	elems, err := elemsFunc(dc, vr, val)
	if err != nil {
		return err
	}

	if val.IsNil() {
		val.Set(reflect.MakeSlice(val.Type(), 0, len(elems)))
	}

	val.SetLen(0)
	val.Set(reflect.Append(val, elems...))

	return nil
}

// decodeVectorInt8 decodes a BSON Vector binary value (subtype 9) into a []int8 slice.
// The binary data should be in the format: [<vector type> <padding> <data>]
// For int8 vectors, the vector type is Int8Vector (0x03).
func decodeVectorInt8(data []byte) ([]int8, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("insufficient bytes to decode vector: expected at least 2 bytes")
	}

	vectorType := data[0]
	if vectorType != Int8Vector {
		return nil, fmt.Errorf("invalid vector type: expected int8 vector (0x%02x), got 0x%02x", Int8Vector, vectorType)
	}

	if padding := data[1]; padding != 0 {
		return nil, fmt.Errorf("invalid vector: padding byte must be 0")
	}

	values := make([]int8, 0, len(data)-2)
	for i := 2; i < len(data); i++ {
		values = append(values, int8(data[i]))
	}

	return values, nil
}

// decodeVectorFloat32 decodes a BSON Vector binary value (subtype 9) into a []float32 slice.
// The binary data should be in the format: [<vector type> <padding> <data>]
// For float32 vectors, the vector type is Float32Vector (0x27) and data must be a multiple of 4 bytes.
func decodeVectorFloat32(data []byte) ([]float32, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("insufficient bytes to decode vector: expected at least 2 bytes")
	}

	vectorType := data[0]
	if vectorType != Float32Vector {
		return nil, fmt.Errorf("invalid vector type: expected float32 vector (0x%02x), got 0x%02x", Float32Vector, vectorType)
	}

	if padding := data[1]; padding != 0 {
		return nil, fmt.Errorf("invalid vector: padding byte must be 0")
	}

	floatData := data[2:]
	if len(floatData)%4 != 0 {
		return nil, fmt.Errorf("invalid float32 vector: data length must be a multiple of 4")
	}

	values := make([]float32, 0, len(floatData)/4)
	for i := 0; i < len(floatData); i += 4 {
		if i+4 > len(floatData) {
			return nil, fmt.Errorf("invalid float32 vector: truncated data")
		}
		bits := binary.LittleEndian.Uint32(floatData[i : i+4])
		values = append(values, math.Float32frombits(bits))
	}

	return values, nil
}
