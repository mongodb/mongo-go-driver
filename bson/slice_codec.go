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

// sliceCodec is the Codec used for slice values.
type sliceCodec struct {
	// encodeNilAsEmpty causes EncodeValue to marshal nil Go slices as empty BSON arrays instead of
	// BSON null.
	encodeNilAsEmpty bool
}

// EncodeValue is the ValueEncoder for slice types.
func (sc *sliceCodec) EncodeValue(ec EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Kind() != reflect.Slice {
		return ValueEncoderError{Name: "SliceEncodeValue", Kinds: []reflect.Kind{reflect.Slice}, Received: val}
	}

	if val.IsNil() && !sc.encodeNilAsEmpty && !ec.nilSliceAsEmpty {
		return vw.WriteNull()
	}

	// If we have a []byte we want to treat it as a binary instead of as an array.
	if val.Type().Elem() == tByte {
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
