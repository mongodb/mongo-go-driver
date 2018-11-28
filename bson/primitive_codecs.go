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
		RegisterEncoder(tDocument, bsoncodec.ValueEncoderLegacyFunc(pc.x.DocumentEncodeValue)).
		RegisterEncoder(tArray, bsoncodec.ValueEncoderLegacyFunc(pc.x.ArrayEncodeValue)).
		RegisterEncoder(tValue, bsoncodec.ValueEncoderLegacyFunc(pc.x.ValueEncodeValue)).
		RegisterEncoder(tJavaScript, bsoncodec.ValueEncoderLegacyFunc(pc.JavaScriptEncodeValue)).
		RegisterEncoder(tSymbol, bsoncodec.ValueEncoderLegacyFunc(pc.SymbolEncodeValue)).
		RegisterEncoder(tRawValue, bsoncodec.ValueEncoderLegacyFunc(pc.RawValueEncodeValue)).
		RegisterEncoder(tElementSlice, bsoncodec.ValueEncoderLegacyFunc(pc.x.ElementSliceEncodeValue)).
		RegisterEncoder(tBinary, bsoncodec.ValueEncoderLegacyFunc(pc.BinaryEncodeValue)).
		RegisterEncoder(tUndefined, bsoncodec.ValueEncoderLegacyFunc(pc.UndefinedEncodeValue)).
		RegisterEncoder(tDateTime, bsoncodec.ValueEncoderLegacyFunc(pc.DateTimeEncodeValue)).
		RegisterEncoder(tNull, bsoncodec.ValueEncoderLegacyFunc(pc.NullEncodeValue)).
		RegisterEncoder(tRegex, bsoncodec.ValueEncoderLegacyFunc(pc.RegexEncodeValue)).
		RegisterEncoder(tDBPointer, bsoncodec.ValueEncoderLegacyFunc(pc.DBPointerEncodeValue)).
		RegisterEncoder(tCodeWithScope, bsoncodec.ValueEncoderLegacyFunc(pc.CodeWithScopeEncodeValue)).
		RegisterEncoder(tTimestamp, bsoncodec.ValueEncoderLegacyFunc(pc.TimestampEncodeValue)).
		RegisterEncoder(tMinKey, bsoncodec.ValueEncoderLegacyFunc(pc.MinKeyEncodeValue)).
		RegisterEncoder(tMaxKey, bsoncodec.ValueEncoderLegacyFunc(pc.MaxKeyEncodeValue)).
		RegisterEncoder(tRaw, bsoncodec.ValueEncoderLegacyFunc(pc.RawEncodeValue)).
		RegisterEncoder(tD, bsoncodec.ValueEncoderLegacyFunc(pc.DEncodeValue)).
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

// JavaScriptEncodeValue is the ValueEncoderFunc for the primitive.JavaScript type.
func (PrimitiveCodecs) JavaScriptEncodeValue(ectx bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var js primitive.JavaScript
	switch t := i.(type) {
	case primitive.JavaScript:
		js = t
	case *primitive.JavaScript:
		if t == nil {
			return vw.WriteNull()
		}
		js = *t
	default:
		return bsoncodec.LegacyValueEncoderError{
			Name:     "JavaScriptEncodeValue",
			Types:    []interface{}{primitive.JavaScript(""), (*primitive.JavaScript)(nil)},
			Received: i,
		}
	}

	return vw.WriteJavascript(string(js))
}

// SymbolEncodeValue is the ValueEncoderFunc for the primitive.Symbol type.
func (PrimitiveCodecs) SymbolEncodeValue(ectx bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var symbol primitive.Symbol
	switch t := i.(type) {
	case primitive.Symbol:
		symbol = t
	case *primitive.Symbol:
		if t == nil {
			return vw.WriteNull()
		}
		symbol = *t
	default:
		return bsoncodec.LegacyValueEncoderError{
			Name:     "SymbolEncodeValue",
			Types:    []interface{}{primitive.Symbol(""), (*primitive.Symbol)(nil)},
			Received: i,
		}
	}

	return vw.WriteSymbol(string(symbol))
}

// BinaryEncodeValue is the ValueEncoderFunc for Binary.
func (PrimitiveCodecs) BinaryEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var b primitive.Binary
	switch t := i.(type) {
	case primitive.Binary:
		b = t
	case *primitive.Binary:
		if t == nil {
			return vw.WriteNull()
		}
		b = *t
	default:
		return bsoncodec.LegacyValueEncoderError{
			Name:     "BinaryEncodeValue",
			Types:    []interface{}{primitive.Binary{}, (*primitive.Binary)(nil)},
			Received: i,
		}
	}

	return vw.WriteBinaryWithSubtype(b.Data, b.Subtype)
}

// UndefinedEncodeValue is the ValueEncoderFunc for Undefined.
func (PrimitiveCodecs) UndefinedEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	switch tt := i.(type) {
	case primitive.Undefined:
	case *primitive.Undefined:
		if tt == nil {
			return vw.WriteNull()
		}
	default:
		return bsoncodec.LegacyValueEncoderError{
			Name:     "UndefinedEncodeValue",
			Types:    []interface{}{primitive.Undefined{}, (*primitive.Undefined)(nil)},
			Received: i,
		}
	}

	return vw.WriteUndefined()
}

// DateTimeEncodeValue is the ValueEncoderFunc for DateTime.
func (PrimitiveCodecs) DateTimeEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var dt primitive.DateTime
	switch t := i.(type) {
	case primitive.DateTime:
		dt = t
	case *primitive.DateTime:
		if t == nil {
			return vw.WriteNull()
		}
		dt = *t
	default:
		return bsoncodec.LegacyValueEncoderError{
			Name:     "DateTimeEncodeValue",
			Types:    []interface{}{primitive.DateTime(0), (*primitive.DateTime)(nil)},
			Received: i,
		}
	}

	return vw.WriteDateTime(int64(dt))
}

// NullEncodeValue is the ValueEncoderFunc for Null.
func (PrimitiveCodecs) NullEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	switch i.(type) {
	case primitive.Null, *primitive.Null:
	default:
		return bsoncodec.LegacyValueEncoderError{
			Name:     "NullEncodeValue",
			Types:    []interface{}{primitive.Null{}, (*primitive.Null)(nil)},
			Received: i,
		}
	}

	return vw.WriteNull()
}

// RegexEncodeValue is the ValueEncoderFunc for Regex.
func (PrimitiveCodecs) RegexEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var regex primitive.Regex
	switch t := i.(type) {
	case primitive.Regex:
		regex = t
	case *primitive.Regex:
		if t == nil {
			return vw.WriteNull()
		}
		regex = *t
	default:
		return bsoncodec.LegacyValueEncoderError{
			Name:     "RegexEncodeValue",
			Types:    []interface{}{primitive.Regex{}, (*primitive.Regex)(nil)},
			Received: i,
		}
	}

	return vw.WriteRegex(regex.Pattern, regex.Options)
}

// DBPointerEncodeValue is the ValueEncoderFunc for DBPointer.
func (PrimitiveCodecs) DBPointerEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var dbp primitive.DBPointer
	switch t := i.(type) {
	case primitive.DBPointer:
		dbp = t
	case *primitive.DBPointer:
		if t == nil {
			return vw.WriteNull()
		}
		dbp = *t
	default:
		return bsoncodec.LegacyValueEncoderError{
			Name:     "DBPointerEncodeValue",
			Types:    []interface{}{primitive.DBPointer{}, (*primitive.DBPointer)(nil)},
			Received: i,
		}
	}

	return vw.WriteDBPointer(dbp.DB, dbp.Pointer)
}

// CodeWithScopeEncodeValue is the ValueEncoderFunc for CodeWithScope.
func (pc PrimitiveCodecs) CodeWithScopeEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var cws primitive.CodeWithScope
	switch t := i.(type) {
	case primitive.CodeWithScope:
		cws = t
	case *primitive.CodeWithScope:
		if t == nil {
			return vw.WriteNull()
		}
		cws = *t
	default:
		return bsoncodec.LegacyValueEncoderError{
			Name:     "CodeWithScopeEncodeValue",
			Types:    []interface{}{primitive.CodeWithScope{}, (*primitive.CodeWithScope)(nil)},
			Received: i,
		}
	}

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

// TimestampEncodeValue is the ValueEncoderFunc for Timestamp.
func (PrimitiveCodecs) TimestampEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var ts primitive.Timestamp
	switch t := i.(type) {
	case primitive.Timestamp:
		ts = t
	case *primitive.Timestamp:
		if t == nil {
			return vw.WriteNull()
		}
		ts = *t
	default:
		return bsoncodec.LegacyValueEncoderError{
			Name:     "TimestampEncodeValue",
			Types:    []interface{}{primitive.Timestamp{}, (*primitive.Timestamp)(nil)},
			Received: i,
		}
	}

	return vw.WriteTimestamp(ts.T, ts.I)
}

// MinKeyEncodeValue is the ValueEncoderFunc for MinKey.
func (PrimitiveCodecs) MinKeyEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	switch tt := i.(type) {
	case primitive.MinKey:
	case *primitive.MinKey:
		if tt == nil {
			return vw.WriteNull()
		}
	default:
		return bsoncodec.LegacyValueEncoderError{
			Name:     "MinKeyEncodeValue",
			Types:    []interface{}{primitive.MinKey{}, (*primitive.MinKey)(nil)},
			Received: i,
		}
	}

	return vw.WriteMinKey()
}

// MaxKeyEncodeValue is the ValueEncoderFunc for MaxKey.
func (PrimitiveCodecs) MaxKeyEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	switch tt := i.(type) {
	case primitive.MaxKey:
	case *primitive.MaxKey:
		if tt == nil {
			return vw.WriteNull()
		}
	default:
		return bsoncodec.LegacyValueEncoderError{
			Name:     "MaxKeyEncodeValue",
			Types:    []interface{}{primitive.MaxKey{}, (*primitive.MaxKey)(nil)},
			Received: i,
		}
	}

	return vw.WriteMaxKey()
}

// RawValueEncodeValue is the ValueEncoderFunc for RawValue.
func (PrimitiveCodecs) RawValueEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var rawvalue RawValue
	switch t := i.(type) {
	case RawValue:
		rawvalue = t
	case *RawValue:
		if t == nil {
			return vw.WriteNull()
		}
		rawvalue = *t
	default:
		return bsoncodec.LegacyValueEncoderError{
			Name:     "RawValueEncodeValue",
			Types:    []interface{}{RawValue{}, (*RawValue)(nil)},
			Received: i,
		}
	}

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
func (PrimitiveCodecs) RawEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	rdr, ok := i.(Raw)
	if !ok {
		return bsoncodec.LegacyValueEncoderError{
			Name:     "RawEncodeValue",
			Types:    []interface{}{Raw{}},
			Received: i,
		}
	}

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

// EmptyInterfaceDecodeValue is the ValueDecoderFunc for interface{}.
func (PrimitiveCodecs) EmptyInterfaceDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	return nil
}

// DEncodeValue is the ValueEncoderFunc for D and *D.
func (pc PrimitiveCodecs) DEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var d D
	switch tt := i.(type) {
	case D:
		d = tt
	case *D:
		if tt == nil {
			return vw.WriteNull()
		}
		d = *tt
	default:
		return bsoncodec.LegacyValueEncoderError{Name: "DEncodeValue", Types: []interface{}{D{}, (*D)(nil)}, Received: i}
	}

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

		err = encoder.EncodeValueLegacy(ec, vw, e.Value)
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
