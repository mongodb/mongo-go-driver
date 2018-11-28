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

	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"github.com/mongodb/mongo-go-driver/bson/bsonrw"
	"github.com/mongodb/mongo-go-driver/bson/bsontype"
	"github.com/mongodb/mongo-go-driver/bson/primitive"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
)

var primitiveCodecs PrimitiveCodecs

// PrimitiveCodecs is a namespace for all of the default bsoncodec.Codecs for the primitive types
// defined in this package.
type PrimitiveCodecs struct {
	x bsonx.PrimitiveCodecs
}

// RegisterPrimitiveCodecs will register the encode and decode methods attached to PrimitiveCodecs
// with the provided RegistryBuilder. if rb is nil, a new empty RegistryBuilder will be created.
func (pc PrimitiveCodecs) RegisterPrimitiveCodecs(rb *bsoncodec.RegistryBuilder) {
	if rb == nil {
		panic(errors.New("argument to RegisterPrimitiveCodecs must not be nil"))
	}

	rb.
		RegisterEncoder(tDocument, bsoncodec.ValueEncoderFunc(pc.x.DocumentEncodeValue)).
		RegisterEncoder(tArray, bsoncodec.ValueEncoderFunc(pc.x.ArrayEncodeValue)).
		RegisterEncoder(tValue, bsoncodec.ValueEncoderFunc(pc.x.ValueEncodeValue)).
		RegisterEncoder(tRawValue, bsoncodec.ValueEncoderFunc(pc.RawValueEncodeValue)).
		RegisterEncoder(tElementSlice, bsoncodec.ValueEncoderFunc(pc.x.ElementSliceEncodeValue)).
		RegisterEncoder(tCodeWithScope, bsoncodec.ValueEncoderFunc(pc.CodeWithScopeEncodeValue)).
		RegisterEncoder(tRaw, bsoncodec.ValueEncoderFunc(pc.RawEncodeValue)).
		RegisterEncoder(tD, bsoncodec.ValueEncoderFunc(pc.DEncodeValue)).
		RegisterDecoder(tDocument, bsoncodec.ValueDecoderFunc(pc.x.DocumentDecodeValue)).
		RegisterDecoder(tArray, bsoncodec.ValueDecoderFunc(pc.x.ArrayDecodeValue)).
		RegisterDecoder(tValue, bsoncodec.ValueDecoderFunc(pc.x.ValueDecodeValue)).
		RegisterDecoder(tRawValue, bsoncodec.ValueDecoderFunc(pc.RawValueDecodeValue)).
		RegisterDecoder(tElementSlice, bsoncodec.ValueDecoderFunc(pc.x.ElementSliceDecodeValue)).
		RegisterDecoder(tCodeWithScope, bsoncodec.ValueDecoderFunc(pc.CodeWithScopeDecodeValue)).
		RegisterDecoder(tRaw, bsoncodec.ValueDecoderFunc(pc.RawDecodeValue)).
		RegisterDecoder(tD, bsoncodec.ValueDecoderFunc(pc.DDecodeValue)).
		RegisterTypeMapEntry(bsontype.EmbeddedDocument, tD).
		RegisterTypeMapEntry(bsontype.Array, tA)
}

// CodeWithScopeEncodeValue is the ValueEncoderFunc for CodeWithScope.
func (pc PrimitiveCodecs) CodeWithScopeEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tCodeWithScope {
		return bsoncodec.ValueEncoderError{Name: "CodeWithScopeEncodeValue", Types: []reflect.Type{tCodeWithScope}, Received: val}
	}

	cws := val.Interface().(primitive.CodeWithScope)

	dw, err := vw.WriteCodeWithScope(string(cws.Code))
	if err != nil {
		return err
	}

	doc, err := MarshalWithRegistry(ec.Registry, cws.Scope)
	if err != nil {
		return err
	}

	return pc.encodeRaw(ec, dw, doc)
}

// CodeWithScopeDecodeValue is the ValueDecoderFunc for CodeWithScope.
func (pc PrimitiveCodecs) CodeWithScopeDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tCodeWithScope {
		return bsoncodec.ValueDecoderError{Name: "CodeWithScopeDecodeValue", Types: []reflect.Type{tCodeWithScope}, Received: val}
	}

	if vr.Type() != bsontype.CodeWithScope {
		return fmt.Errorf("cannot decode %v into a CodeWithScope", vr.Type())
	}

	code, dr, err := vr.ReadCodeWithScope()
	if err != nil {
		return err
	}

	var scope bsonx.Doc
	err = pc.x.DecodeDocument(dc, dr, &scope)
	if err != nil {
		return err
	}

	val.Set(reflect.ValueOf(primitive.CodeWithScope{Code: primitive.JavaScript(code), Scope: scope}))
	return nil
}

// RawValueEncodeValue is the ValueEncoderFunc for RawValue.
func (PrimitiveCodecs) RawValueEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tRawValue {
		return bsoncodec.ValueEncoderError{Name: "RawValueEncodeValue", Types: []reflect.Type{tRawValue}, Received: val}
	}

	rawvalue := val.Interface().(RawValue)

	return bsonrw.Copier{}.CopyValueFromBytes(vw, rawvalue.Type, rawvalue.Value)
}

// RawValueDecodeValue is the ValueDecoderFunc for RawValue.
func (PrimitiveCodecs) RawValueDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tRawValue {
		return bsoncodec.ValueDecoderError{Name: "RawValueDecodeValue", Types: []reflect.Type{tRawValue}, Received: val}
	}

	t, value, err := bsonrw.Copier{}.CopyValueToBytes(vr)
	if err != nil {
		return err
	}

	val.Set(reflect.ValueOf(RawValue{Type: t, Value: value}))
	return nil
}

// RawEncodeValue is the ValueEncoderFunc for Reader.
func (PrimitiveCodecs) RawEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tRaw {
		return bsoncodec.ValueEncoderError{Name: "RawEncodeValue", Types: []reflect.Type{tRaw}, Received: val}
	}

	rdr := val.Interface().(Raw)

	return bsonrw.Copier{}.CopyDocumentFromBytes(vw, rdr)
}

// RawDecodeValue is the ValueDecoderFunc for Reader.
func (PrimitiveCodecs) RawDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tRaw {
		return bsoncodec.ValueDecoderError{Name: "RawDecodeValue", Types: []reflect.Type{tRaw}, Received: val}
	}

	if val.IsNil() {
		val.Set(reflect.MakeSlice(val.Type(), 0, 0))
	}

	val.SetLen(0)

	rdr, err := bsonrw.Copier{}.AppendDocumentBytes(val.Interface().(Raw), vr)
	val.Set(reflect.ValueOf(rdr))
	return err
}

// DEncodeValue is the ValueEncoderFunc for D and *D.
func (pc PrimitiveCodecs) DEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tD {
		return bsoncodec.ValueEncoderError{Name: "DEncodeValue", Types: []reflect.Type{tD}, Received: val}
	}

	d := val.Interface().(D)

	dw, err := vw.WriteDocument()
	if err != nil {
		return err
	}

	for _, e := range d {
		vw, err := dw.WriteDocumentElement(e.Key)
		if err != nil {
			return err
		}

		encoder, err := ec.LookupEncoder(reflect.TypeOf(e.Value))
		if err != nil {
			return err
		}

		err = encoder.EncodeValue(ec, vw, reflect.ValueOf(e.Value))
		if err != nil {
			return err
		}
	}

	return dw.WriteDocumentEnd()
}

// DDecodeValue is the ValueDecoderFunc for *D and **D.
func (pc PrimitiveCodecs) DDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tD {
		return bsoncodec.ValueDecoderError{Name: "DDecodeValue", Types: []reflect.Type{tD}, Received: val}
	}

	switch vr.Type() {
	case bsontype.Type(0), bsontype.EmbeddedDocument:
	default:
		return fmt.Errorf("cannot decode %v into a D", vr.Type())
	}

	dr, err := vr.ReadDocument()
	if err != nil {
		return err
	}

	if val.IsNil() {
		val.Set(reflect.MakeSlice(val.Type(), 0, 0))
	}

	val.SetLen(0)

	decoder, err := dc.LookupDecoder(tEmpty)
	if err != nil {
		return err
	}

	elems := make([]reflect.Value, 0)
	for {
		key, vr, err := dr.ReadElement()
		if err == bsonrw.ErrEOD {
			break
		}
		if err != nil {
			return err
		}

		val := reflect.New(tEmpty).Elem()
		err = decoder.DecodeValue(dc, vr, val)
		if err != nil {
			return err
		}

		elems = append(elems, reflect.ValueOf(E{Key: key, Value: val.Interface()}))
	}

	val.Set(reflect.Append(val, elems...))
	return nil
}

func (pc PrimitiveCodecs) encodeRaw(ec bsoncodec.EncodeContext, dw bsonrw.DocumentWriter, raw Raw) error {
	var copier bsonrw.Copier
	elems, err := raw.Elements()
	if err != nil {
		return err
	}
	for _, elem := range elems {
		dvw, err := dw.WriteDocumentElement(elem.Key())
		if err != nil {
			return err
		}

		val := elem.Value()
		err = copier.CopyValueFromBytes(dvw, val.Type, val.Value)
		if err != nil {
			return err
		}
	}

	return dw.WriteDocumentEnd()
}
