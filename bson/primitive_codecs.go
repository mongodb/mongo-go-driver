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
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
	"github.com/mongodb/mongo-go-driver/bson/primitive"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
)

var primitiveCodecs PrimitiveCodecs

// PrimitiveCodecs is a namespace for all of the default bsoncodec.Codecs for the primitive types
// defined in this package.
type PrimitiveCodecs struct{}

// RegisterPrimitiveCodecs will register the encode and decode methods attached to PrimitiveCodecs
// with the provided RegistryBuilder. if rb is nil, a new empty RegistryBuilder will be created.
func (pc PrimitiveCodecs) RegisterPrimitiveCodecs(rb *bsoncodec.RegistryBuilder) {
	if rb == nil {
		panic(errors.New("argument to RegisterPrimitiveCodecs must not be nil"))
	}

	rb.
		RegisterEncoder(tDocument, bsoncodec.ValueEncoderFunc(pc.DocumentEncodeValue)).
		RegisterEncoder(tArray, bsoncodec.ValueEncoderFunc(pc.ArrayEncodeValue)).
		RegisterEncoder(tValue, bsoncodec.ValueEncoderFunc(pc.ValueEncodeValue)).
		RegisterEncoder(reflect.PtrTo(tRawValue), bsoncodec.ValueEncoderFunc(pc.RawValueEncodeValue)).
		RegisterEncoder(reflect.PtrTo(tElementSlice), bsoncodec.ValueEncoderFunc(pc.ElementSliceEncodeValue)).
		RegisterEncoder(reflect.PtrTo(tBinary), bsoncodec.ValueEncoderFunc(pc.BinaryEncodeValue)).
		RegisterEncoder(reflect.PtrTo(tUndefined), bsoncodec.ValueEncoderFunc(pc.UndefinedEncodeValue)).
		RegisterEncoder(reflect.PtrTo(tDateTime), bsoncodec.ValueEncoderFunc(pc.DateTimeEncodeValue)).
		RegisterEncoder(reflect.PtrTo(tNull), bsoncodec.ValueEncoderFunc(pc.NullEncodeValue)).
		RegisterEncoder(reflect.PtrTo(tRegex), bsoncodec.ValueEncoderFunc(pc.RegexEncodeValue)).
		RegisterEncoder(reflect.PtrTo(tDBPointer), bsoncodec.ValueEncoderFunc(pc.DBPointerEncodeValue)).
		RegisterEncoder(reflect.PtrTo(tCodeWithScope), bsoncodec.ValueEncoderFunc(pc.CodeWithScopeEncodeValue)).
		RegisterEncoder(reflect.PtrTo(tTimestamp), bsoncodec.ValueEncoderFunc(pc.TimestampEncodeValue)).
		RegisterEncoder(reflect.PtrTo(tMinKey), bsoncodec.ValueEncoderFunc(pc.MinKeyEncodeValue)).
		RegisterEncoder(reflect.PtrTo(tMaxKey), bsoncodec.ValueEncoderFunc(pc.MaxKeyEncodeValue)).
		RegisterEncoder(reflect.PtrTo(tRaw), bsoncodec.ValueEncoderFunc(pc.RawEncodeValue)).
		RegisterEncoder(reflect.PtrTo(tD), bsoncodec.ValueEncoderFunc(pc.DEncodeValue)).
		RegisterDecoder(tDocument, bsoncodec.ValueDecoderFunc(pc.DocumentDecodeValue)).
		RegisterDecoder(tArray, bsoncodec.ValueDecoderFunc(pc.ArrayDecodeValue)).
		RegisterDecoder(tValue, bsoncodec.ValueDecoderFunc(pc.ValueDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tRawValue), bsoncodec.ValueDecoderFunc(pc.RawValueDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tElementSlice), bsoncodec.ValueDecoderFunc(pc.ElementSliceDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tBinary), bsoncodec.ValueDecoderFunc(pc.BinaryDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tUndefined), bsoncodec.ValueDecoderFunc(pc.UndefinedDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tDateTime), bsoncodec.ValueDecoderFunc(pc.DateTimeDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tNull), bsoncodec.ValueDecoderFunc(pc.NullDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tRegex), bsoncodec.ValueDecoderFunc(pc.RegexDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tDBPointer), bsoncodec.ValueDecoderFunc(pc.DBPointerDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tCodeWithScope), bsoncodec.ValueDecoderFunc(pc.CodeWithScopeDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tTimestamp), bsoncodec.ValueDecoderFunc(pc.TimestampDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tMinKey), bsoncodec.ValueDecoderFunc(pc.MinKeyDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tMaxKey), bsoncodec.ValueDecoderFunc(pc.MaxKeyDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tRaw), bsoncodec.ValueDecoderFunc(pc.RawDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tEmpty), bsoncodec.ValueDecoderFunc(pc.EmptyInterfaceDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tD), bsoncodec.ValueDecoderFunc(pc.DDecodeValue))
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
		return bsoncodec.ValueEncoderError{
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
		return bsoncodec.ValueEncoderError{
			Name:     "SymbolEncodeValue",
			Types:    []interface{}{primitive.Symbol(""), (*primitive.Symbol)(nil)},
			Received: i,
		}
	}

	return vw.WriteJavascript(string(symbol))
}

// JavaScriptDecodeValue is the ValueDecoderFunc for the primitive.JavaScript type.
func (PrimitiveCodecs) JavaScriptDecodeValue(dctx bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.JavaScript {
		return fmt.Errorf("cannot decode %v into a primitive.JavaScript", vr.Type())
	}

	js, err := vr.ReadJavascript()
	if err != nil {
		return err
	}

	if target, ok := i.(*primitive.JavaScript); ok && target != nil {
		*target = primitive.JavaScript(js)
		return nil
	}

	if target, ok := i.(**primitive.JavaScript); ok && target != nil {
		pjs := *target
		if pjs == nil {
			pjs = new(primitive.JavaScript)
		}
		*pjs = primitive.JavaScript(js)
		*target = pjs
		return nil
	}

	return bsoncodec.ValueDecoderError{
		Name:     "JavaScriptDecodeValue",
		Types:    []interface{}{(*primitive.JavaScript)(nil), (**primitive.JavaScript)(nil)},
		Received: i,
	}
}

// SymbolDecodeValue is the ValueDecoderFunc for the primitive.Symbol type.
func (PrimitiveCodecs) SymbolDecodeValue(dctx bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.Symbol {
		return fmt.Errorf("cannot decode %v into a primitive.Symbol", vr.Type())
	}

	symbol, err := vr.ReadSymbol()
	if err != nil {
		return err
	}

	if target, ok := i.(*primitive.Symbol); ok && target != nil {
		*target = primitive.Symbol(symbol)
		return nil
	}

	if target, ok := i.(**primitive.Symbol); ok && target != nil {
		psymbol := *target
		if psymbol == nil {
			psymbol = new(primitive.Symbol)
		}
		*psymbol = primitive.Symbol(symbol)
		*target = psymbol
		return nil
	}

	return bsoncodec.ValueDecoderError{Name: "SymbolDecodeValue", Types: []interface{}{(*primitive.Symbol)(nil), (**primitive.Symbol)(nil)}, Received: i}
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
		return bsoncodec.ValueEncoderError{
			Name:     "BinaryEncodeValue",
			Types:    []interface{}{primitive.Binary{}, (*primitive.Binary)(nil)},
			Received: i,
		}
	}

	return vw.WriteBinaryWithSubtype(b.Data, b.Subtype)
}

// BinaryDecodeValue is the ValueDecoderFunc for Binary.
func (PrimitiveCodecs) BinaryDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.Binary {
		return fmt.Errorf("cannot decode %v into a Binary", vr.Type())
	}

	data, subtype, err := vr.ReadBinary()
	if err != nil {
		return err
	}

	if target, ok := i.(*primitive.Binary); ok && target != nil {
		*target = primitive.Binary{Data: data, Subtype: subtype}
		return nil
	}

	if target, ok := i.(**primitive.Binary); ok && target != nil {
		pb := *target
		if pb == nil {
			pb = new(primitive.Binary)
		}
		*pb = primitive.Binary{Data: data, Subtype: subtype}
		*target = pb
		return nil
	}

	return bsoncodec.ValueDecoderError{Name: "BinaryDecodeValue", Types: []interface{}{(*primitive.Binary)(nil)}, Received: i}
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
		return bsoncodec.ValueEncoderError{
			Name:     "UndefinedEncodeValue",
			Types:    []interface{}{primitive.Undefined{}, (*primitive.Undefined)(nil)},
			Received: i,
		}
	}

	return vw.WriteUndefined()
}

// UndefinedDecodeValue is the ValueDecoderFunc for Undefined.
func (PrimitiveCodecs) UndefinedDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.Undefined {
		return fmt.Errorf("cannot decode %v into an Undefined", vr.Type())
	}

	target, ok := i.(*primitive.Undefined)
	if !ok || target == nil {
		return bsoncodec.ValueDecoderError{Name: "UndefinedDecodeValue", Types: []interface{}{(*primitive.Undefined)(nil)}, Received: i}
	}

	*target = primitive.Undefined{}
	return vr.ReadUndefined()
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
		return bsoncodec.ValueEncoderError{
			Name:     "DateTimeEncodeValue",
			Types:    []interface{}{primitive.DateTime(0), (*primitive.DateTime)(nil)},
			Received: i,
		}
	}

	return vw.WriteDateTime(int64(dt))
}

// DateTimeDecodeValue is the ValueDecoderFunc for DateTime.
func (PrimitiveCodecs) DateTimeDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.DateTime {
		return fmt.Errorf("cannot decode %v into a DateTime", vr.Type())
	}

	target, ok := i.(*primitive.DateTime)
	if !ok || target == nil {
		return bsoncodec.ValueDecoderError{Name: "DateTimeDecodeValue", Types: []interface{}{(*primitive.DateTime)(nil)}, Received: i}
	}

	dt, err := vr.ReadDateTime()
	if err != nil {
		return err
	}

	*target = primitive.DateTime(dt)
	return nil
}

// NullEncodeValue is the ValueEncoderFunc for Null.
func (PrimitiveCodecs) NullEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	switch i.(type) {
	case primitive.Null, *primitive.Null:
	default:
		return bsoncodec.ValueEncoderError{
			Name:     "NullEncodeValue",
			Types:    []interface{}{primitive.Null{}, (*primitive.Null)(nil)},
			Received: i,
		}
	}

	return vw.WriteNull()
}

// NullDecodeValue is the ValueDecoderFunc for Null.
func (PrimitiveCodecs) NullDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.Null {
		return fmt.Errorf("cannot decode %v into a Null", vr.Type())
	}

	target, ok := i.(*primitive.Null)
	if !ok || target == nil {
		return bsoncodec.ValueDecoderError{Name: "NullDecodeValue", Types: []interface{}{(*primitive.Null)(nil)}, Received: i}
	}

	*target = primitive.Null{}
	return vr.ReadNull()
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
		return bsoncodec.ValueEncoderError{
			Name:     "RegexEncodeValue",
			Types:    []interface{}{primitive.Regex{}, (*primitive.Regex)(nil)},
			Received: i,
		}
	}

	return vw.WriteRegex(regex.Pattern, regex.Options)
}

// RegexDecodeValue is the ValueDecoderFunc for Regex.
func (PrimitiveCodecs) RegexDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.Regex {
		return fmt.Errorf("cannot decode %v into a Regex", vr.Type())
	}

	target, ok := i.(*primitive.Regex)
	if !ok || target == nil {
		return bsoncodec.ValueDecoderError{Name: "RegexDecodeValue", Types: []interface{}{(*primitive.Regex)(nil)}, Received: i}
	}

	pattern, options, err := vr.ReadRegex()
	if err != nil {
		return err
	}

	*target = primitive.Regex{Pattern: pattern, Options: options}
	return nil
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
		return bsoncodec.ValueEncoderError{
			Name:     "DBPointerEncodeValue",
			Types:    []interface{}{primitive.DBPointer{}, (*primitive.DBPointer)(nil)},
			Received: i,
		}
	}

	return vw.WriteDBPointer(dbp.DB, dbp.Pointer)
}

// DBPointerDecodeValue is the ValueDecoderFunc for DBPointer.
func (PrimitiveCodecs) DBPointerDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.DBPointer {
		return fmt.Errorf("cannot decode %v into a DBPointer", vr.Type())
	}

	target, ok := i.(*primitive.DBPointer)
	if !ok || target == nil {
		return bsoncodec.ValueDecoderError{Name: "DBPointerDecodeValue", Types: []interface{}{(*primitive.DBPointer)(nil)}, Received: i}
	}

	ns, pointer, err := vr.ReadDBPointer()
	if err != nil {
		return err
	}

	*target = primitive.DBPointer{DB: ns, Pointer: pointer}
	return nil
}

// DocumentEncodeValue is the ValueEncoderFunc for *Document.
func (pc PrimitiveCodecs) DocumentEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	doc, ok := i.(bsonx.Doc)
	if !ok {
		return bsoncodec.ValueEncoderError{Name: "DocumentEncodeValue", Types: []interface{}{(bsonx.Doc)(nil), (*bsonx.Doc)(nil)}, Received: i}
	}

	if doc == nil {
		return vw.WriteNull()
	}

	dw, err := vw.WriteDocument()
	if err != nil {
		return err
	}

	return pc.encodeDocument(ec, dw, doc)
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
		return bsoncodec.ValueEncoderError{
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
func (pc PrimitiveCodecs) CodeWithScopeDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.CodeWithScope {
		return fmt.Errorf("cannot decode %v into a CodeWithScope", vr.Type())
	}

	target, ok := i.(*primitive.CodeWithScope)
	if !ok || target == nil {
		return bsoncodec.ValueDecoderError{
			Name:     "CodeWithScopeDecodeValue",
			Types:    []interface{}{(*primitive.CodeWithScope)(nil)},
			Received: i,
		}
	}

	code, dr, err := vr.ReadCodeWithScope()
	if err != nil {
		return err
	}

	var scope bsonx.Doc
	err = pc.decodeDocument(dc, dr, &scope)
	if err != nil {
		return err
	}

	*target = primitive.CodeWithScope{Code: primitive.JavaScript(code), Scope: scope}
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
		return bsoncodec.ValueEncoderError{
			Name:     "TimestampEncodeValue",
			Types:    []interface{}{primitive.Timestamp{}, (*primitive.Timestamp)(nil)},
			Received: i,
		}
	}

	return vw.WriteTimestamp(ts.T, ts.I)
}

// TimestampDecodeValue is the ValueDecoderFunc for Timestamp.
func (PrimitiveCodecs) TimestampDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.Timestamp {
		return fmt.Errorf("cannot decode %v into a Timestamp", vr.Type())
	}

	target, ok := i.(*primitive.Timestamp)
	if !ok || target == nil {
		return bsoncodec.ValueDecoderError{Name: "TimestampDecodeValue", Types: []interface{}{(*primitive.Timestamp)(nil)}, Received: i}
	}

	t, incr, err := vr.ReadTimestamp()
	if err != nil {
		return err
	}

	*target = primitive.Timestamp{T: t, I: incr}
	return nil
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
		return bsoncodec.ValueEncoderError{
			Name:     "MinKeyEncodeValue",
			Types:    []interface{}{primitive.MinKey{}, (*primitive.MinKey)(nil)},
			Received: i,
		}
	}

	return vw.WriteMinKey()
}

// MinKeyDecodeValue is the ValueDecoderFunc for MinKey.
func (PrimitiveCodecs) MinKeyDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.MinKey {
		return fmt.Errorf("cannot decode %v into a MinKey", vr.Type())
	}

	target, ok := i.(*primitive.MinKey)
	if !ok || target == nil {
		return bsoncodec.ValueDecoderError{Name: "MinKeyDecodeValue", Types: []interface{}{(*primitive.MinKey)(nil)}, Received: i}
	}

	*target = primitive.MinKey{}
	return vr.ReadMinKey()
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
		return bsoncodec.ValueEncoderError{
			Name:     "MaxKeyEncodeValue",
			Types:    []interface{}{primitive.MaxKey{}, (*primitive.MaxKey)(nil)},
			Received: i,
		}
	}

	return vw.WriteMaxKey()
}

// MaxKeyDecodeValue is the ValueDecoderFunc for MaxKey.
func (PrimitiveCodecs) MaxKeyDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.MaxKey {
		return fmt.Errorf("cannot decode %v into a MaxKey", vr.Type())
	}

	target, ok := i.(*primitive.MaxKey)
	if !ok || target == nil {
		return bsoncodec.ValueDecoderError{Name: "MaxKeyDecodeValue", Types: []interface{}{(*primitive.MaxKey)(nil)}, Received: i}
	}

	*target = primitive.MaxKey{}
	return vr.ReadMaxKey()
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
		return bsoncodec.ValueEncoderError{
			Name:     "RawValueEncodeValue",
			Types:    []interface{}{RawValue{}, (*RawValue)(nil)},
			Received: i,
		}
	}

	return bsonrw.Copier{}.CopyValueFromBytes(vw, rawvalue.Type, rawvalue.Value)
}

// RawValueDecodeValue is the ValueDecoderFunc for RawValue.
func (PrimitiveCodecs) RawValueDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	var target *RawValue
	fail := func() error {
		return bsoncodec.ValueDecoderError{
			Name:     "RawValueDecodeValue",
			Types:    []interface{}{(*RawValue)(nil), (**RawValue)(nil)},
			Received: i,
		}
	}
	switch t := i.(type) {
	case *RawValue:
		if t == nil {
			return fail()
		}
		target = t
	case **RawValue:
		if t == nil {
			return fail()
		}
		if *t == nil {
			*t = new(RawValue)
		}
		target = *t
	default:
		return fail()
	}

	t, val, err := bsonrw.Copier{}.CopyValueToBytes(vr)
	if err != nil {
		return err
	}

	target.Type, target.Value, target.r = t, val, dc.Registry
	return nil
}

// ValueEncodeValue is the ValueEncoderFunc for *Value.
func (pc PrimitiveCodecs) ValueEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var val Val
	switch tt := i.(type) {
	case Val:
		val = tt
	case *Val:
		if tt == nil {
			return vw.WriteNull()
		}
		val = *tt
	default:
		return bsoncodec.ValueEncoderError{
			Name:     "ValueEncodeValue",
			Types:    []interface{}{Val{}, (*Val)(nil)},
			Received: i,
		}
	}

	return pc.encodeValue(ec, vw, val)
}

// ValueDecodeValue is the ValueDecoderFunc for *Value.
func (pc PrimitiveCodecs) ValueDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	pval, ok := i.(*Val)
	if !ok {
		return bsoncodec.ValueDecoderError{Name: "ValueDecodeValue", Types: []interface{}{(*Val)(nil)}, Received: i}
	}

	if pval == nil {
		return errors.New("ValueDecodeValue can only be used to decode non-nil *Value")
	}

	return pc.valueDecodeValue(dc, vr, pval)
}

// RawEncodeValue is the ValueEncoderFunc for Reader.
func (PrimitiveCodecs) RawEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	rdr, ok := i.(Raw)
	if !ok {
		return bsoncodec.ValueEncoderError{
			Name:     "RawEncodeValue",
			Types:    []interface{}{Raw{}},
			Received: i,
		}
	}

	return bsonrw.Copier{}.CopyDocumentFromBytes(vw, rdr)
}

// RawDecodeValue is the ValueDecoderFunc for Reader.
func (PrimitiveCodecs) RawDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	rdr, ok := i.(*Raw)
	if !ok {
		return bsoncodec.ValueDecoderError{Name: "RawDecodeValue", Types: []interface{}{(*Raw)(nil)}, Received: i}
	}

	if rdr == nil {
		return errors.New("RawDecodeValue can only be used to decode non-nil *Reader")
	}

	if *rdr == nil {
		*rdr = make(Raw, 0)
	} else {
		*rdr = (*rdr)[:0]
	}

	var err error
	*rdr, err = bsonrw.Copier{}.AppendDocumentBytes(*rdr, vr)
	return err
}

// ElementSliceEncodeValue is the ValueEncoderFunc for []*Element.
func (pc PrimitiveCodecs) ElementSliceEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var slce []Elem
	switch t := i.(type) {
	case []Elem:
		slce = t
	case *[]Elem:
		if t == nil {
			return vw.WriteNull()
		}
		slce = *t
	default:
		return bsoncodec.ValueEncoderError{
			Name:     "ElementSliceEncodeValue",
			Types:    []interface{}{[]Elem{}, (*[]Elem)(nil)},
			Received: i,
		}
	}

	return pc.DocumentEncodeValue(ec, vw, bsonx.Doc(slce))
}

// ElementSliceDecodeValue is the ValueDecoderFunc for []*Element.
func (pc PrimitiveCodecs) ElementSliceDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	dr, err := vr.ReadDocument()
	if err != nil {
		return err
	}
	elems := make([]Elem, 0)
	for {
		key, vr, err := dr.ReadElement()
		if err == bsonrw.ErrEOD {
			break
		}
		if err != nil {
			return err
		}

		var elem Elem
		err = pc.elementDecodeValue(dc, vr, key, &elem)
		if err != nil {
			return err
		}

		elems = append(elems, elem)
	}

	target, ok := i.(*[]Elem)
	if !ok || target == nil {
		return bsoncodec.ValueDecoderError{Name: "ElementSliceDecodeValue", Types: []interface{}{(*[]Elem)(nil)}, Received: i}
	}

	*target = elems
	return nil
}

// DocumentDecodeValue is the ValueDecoderFunc for *Document.
func (pc PrimitiveCodecs) DocumentDecodeValue(dctx bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	doc, ok := i.(*bsonx.Doc)
	if !ok {
		return bsoncodec.ValueDecoderError{Name: "DocumentDecodeValue", Types: []interface{}{(*bsonx.Doc)(nil)}, Received: i}
	}

	if doc == nil {
		return errors.New("DocumentDecodeValue can only be used to decode non-nil *Doc")
	}

	dr, err := vr.ReadDocument()
	if err != nil {
		return err
	}

	return pc.decodeDocument(dctx, dr, doc)
}

// ArrayEncodeValue is the ValueEncoderFunc for *Array.
func (pc PrimitiveCodecs) ArrayEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var arr Arr
	switch tt := i.(type) {
	case Arr:
		if tt == nil {
			return vw.WriteNull()
		}
		arr = tt
	case *Arr:
		if tt == nil {
			return vw.WriteNull()
		}
		arr = *tt
	default:
		return bsoncodec.ValueEncoderError{Name: "ArrayEncodeValue", Types: []interface{}{(Arr)(nil), (*Arr)(nil)}, Received: i}
	}

	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		dvw, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		err = pc.encodeValue(ec, dvw, val)

		if err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

// ArrayDecodeValue is the ValueDecoderFunc for *Array.
func (pc PrimitiveCodecs) ArrayDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	parr, ok := i.(*Arr)
	if !ok {
		return bsoncodec.ValueDecoderError{Name: "ArrayDecodeValue", Types: []interface{}{(*Arr)(nil)}, Received: i}
	}

	if parr == nil {
		return errors.New("ArrayDecodeValue can only be used to decode non-nil *Arr")
	}

	ar, err := vr.ReadArray()
	if err != nil {
		return err
	}

	if *parr == nil {
		*parr = make(Arr, 0)
	}
	*parr = (*parr)[:0]
	for {
		vr, err := ar.ReadValue()
		if err == bsonrw.ErrEOA {
			break
		}
		if err != nil {
			return err
		}

		var val Val
		err = pc.valueDecodeValue(dc, vr, &val)
		if err != nil {
			return err
		}

		*parr = append(*parr, val)
	}

	return nil
}

// EmptyInterfaceDecodeValue is the ValueDecoderFunc for interface{}.
func (PrimitiveCodecs) EmptyInterfaceDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	target, ok := i.(*interface{})
	if !ok || target == nil {
		return fmt.Errorf("EmptyInterfaceDecodeValue can only be used to decode non-nil *interface{} values, provided type if %T", i)
	}

	// fn is a function we call to assign val back to the target, we do this so
	// we can keep down on the repeated code in this method. In all of the
	// implementations this is a closure, so we don't need to provide the
	// target as a parameter.
	var fn func()
	var val interface{}
	var rtype reflect.Type

	switch vr.Type() {
	case bsontype.Double:
		val = new(float64)
		rtype = tFloat64
		fn = func() { *target = *(val.(*float64)) }
	case bsontype.String:
		val = new(string)
		rtype = tString
		fn = func() { *target = *(val.(*string)) }
	case bsontype.EmbeddedDocument:
		val = new(bsonx.Doc)
		rtype = tDocument
		fn = func() { *target = *val.(*bsonx.Doc) }
	case bsontype.Array:
		val = new(Arr)
		rtype = tArray
		fn = func() { *target = *val.(*Arr) }
	case bsontype.Binary:
		val = new(primitive.Binary)
		rtype = tBinary
		fn = func() { *target = *(val.(*primitive.Binary)) }
	case bsontype.Undefined:
		val = new(primitive.Undefined)
		rtype = tUndefined
		fn = func() { *target = *(val.(*primitive.Undefined)) }
	case bsontype.ObjectID:
		val = new(objectid.ObjectID)
		rtype = tOID
		fn = func() { *target = *(val.(*objectid.ObjectID)) }
	case bsontype.Boolean:
		val = new(bool)
		rtype = tBool
		fn = func() { *target = *(val.(*bool)) }
	case bsontype.DateTime:
		val = new(primitive.DateTime)
		rtype = tDateTime
		fn = func() { *target = *(val.(*primitive.DateTime)) }
	case bsontype.Null:
		val = new(primitive.Null)
		rtype = tNull
		fn = func() { *target = *(val.(*primitive.Null)) }
	case bsontype.Regex:
		val = new(primitive.Regex)
		rtype = tRegex
		fn = func() { *target = *(val.(*primitive.Regex)) }
	case bsontype.DBPointer:
		val = new(primitive.DBPointer)
		rtype = tDBPointer
		fn = func() { *target = *(val.(*primitive.DBPointer)) }
	case bsontype.JavaScript:
		val = new(primitive.JavaScript)
		rtype = tJavaScript
		fn = func() { *target = *(val.(*primitive.JavaScript)) }
	case bsontype.Symbol:
		val = new(primitive.Symbol)
		rtype = tSymbol
		fn = func() { *target = *(val.(*primitive.Symbol)) }
	case bsontype.CodeWithScope:
		val = new(primitive.CodeWithScope)
		rtype = tCodeWithScope
		fn = func() { *target = *(val.(*primitive.CodeWithScope)) }
	case bsontype.Int32:
		val = new(int32)
		rtype = tInt32
		fn = func() { *target = *(val.(*int32)) }
	case bsontype.Int64:
		val = new(int64)
		rtype = tInt64
		fn = func() { *target = *(val.(*int64)) }
	case bsontype.Timestamp:
		val = new(primitive.Timestamp)
		rtype = tTimestamp
		fn = func() { *target = *(val.(*primitive.Timestamp)) }
	case bsontype.Decimal128:
		val = new(decimal.Decimal128)
		rtype = tDecimal
		fn = func() { *target = *(val.(*decimal.Decimal128)) }
	case bsontype.MinKey:
		val = new(primitive.MinKey)
		rtype = tMinKey
		fn = func() { *target = *(val.(*primitive.MinKey)) }
	case bsontype.MaxKey:
		val = new(primitive.MaxKey)
		rtype = tMaxKey
		fn = func() { *target = *(val.(*primitive.MaxKey)) }
	default:
		return fmt.Errorf("Type %s is not a valid BSON type and has no default Go type to decode into", vr.Type())
	}

	decoder, err := dc.LookupDecoder(rtype)
	if err != nil {
		return err
	}
	err = decoder.DecodeValue(dc, vr, val)
	if err != nil {
		return err
	}

	fn()
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
		return bsoncodec.ValueEncoderError{Name: "DEncodeValue", Types: []interface{}{D{}, (*D)(nil)}, Received: i}
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

		err = encoder.EncodeValue(ec, vw, e.Value)
		if err != nil {
			return err
		}
	}

	return dw.WriteDocumentEnd()
}

// DDecodeValue is the ValueDecoderFunc for *D and **D.
func (pc PrimitiveCodecs) DDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	var target *D
	fail := func() error {
		return bsoncodec.ValueDecoderError{Name: "DDecodeValue", Types: []interface{}{(*D)(nil), (**D)(nil)}, Received: i}
	}
	switch tt := i.(type) {
	case *D:
		if tt == nil {
			return fail()
		}
		target = tt
	case **D:
		if tt == nil {
			return fail()
		}
		if vr.Type() == bsontype.Null {
			err := vr.ReadNull()
			*tt = nil
			return err
		}
		if *tt == nil {
			*tt = new(D)
		}
		target = *tt
	default:
		return fail()
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

	*target = (*target)[:0]

	for {
		key, vr, err := dr.ReadElement()
		if err == bsonrw.ErrEOD {
			break
		}
		if err != nil {
			return err
		}

		var val interface{}
		err = pc.EmptyInterfaceDecodeValue(dc, vr, &val)
		if err != nil {
			return err
		}

		*target = append(*target, E{Key: key, Value: val})
	}

	return nil
}

// encodeDocument is a separate function that we use because CodeWithScope
// returns us a DocumentWriter and we need to do the same logic that we would do
// for a document but cannot use a Codec.
func (pc PrimitiveCodecs) encodeDocument(ec bsoncodec.EncodeContext, dw bsonrw.DocumentWriter, doc bsonx.Doc) error {
	for _, elem := range doc {
		dvw, err := dw.WriteDocumentElement(elem.Key)
		if err != nil {
			return err
		}

		err = pc.encodeValue(ec, dvw, elem.Value)

		if err != nil {
			return err
		}
	}

	return dw.WriteDocumentEnd()
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

func (pc PrimitiveCodecs) elementDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, key string, elem *Elem) error {
	var val Val
	switch vr.Type() {
	case bsontype.Double:
		f64, err := vr.ReadDouble()
		if err != nil {
			return err
		}
		val = Double(f64)
	case bsontype.String:
		str, err := vr.ReadString()
		if err != nil {
			return err
		}
		val = String(str)
	case bsontype.EmbeddedDocument:
		var embeddedDoc bsonx.Doc
		err := pc.DocumentDecodeValue(dc, vr, &embeddedDoc)
		if err != nil {
			return err
		}
		val = Document(embeddedDoc)
	case bsontype.Array:
		var arr Arr
		err := pc.ArrayDecodeValue(dc, vr, &arr)
		if err != nil {
			return err
		}
		val = Array(arr)
	case bsontype.Binary:
		data, subtype, err := vr.ReadBinary()
		if err != nil {
			return err
		}
		val = Binary(subtype, data)
	case bsontype.Undefined:
		err := vr.ReadUndefined()
		if err != nil {
			return err
		}
		val = Undefined()
	case bsontype.ObjectID:
		oid, err := vr.ReadObjectID()
		if err != nil {
			return err
		}
		val = ObjectID(oid)
	case bsontype.Boolean:
		b, err := vr.ReadBoolean()
		if err != nil {
			return err
		}
		val = Boolean(b)
	case bsontype.DateTime:
		dt, err := vr.ReadDateTime()
		if err != nil {
			return err
		}
		val = DateTime(dt)
	case bsontype.Null:
		err := vr.ReadNull()
		if err != nil {
			return err
		}
		val = Null()
	case bsontype.Regex:
		pattern, options, err := vr.ReadRegex()
		if err != nil {
			return err
		}
		val = Regex(pattern, options)
	case bsontype.DBPointer:
		ns, pointer, err := vr.ReadDBPointer()
		if err != nil {
			return err
		}
		val = DBPointer(ns, pointer)
	case bsontype.JavaScript:
		js, err := vr.ReadJavascript()
		if err != nil {
			return err
		}
		val = JavaScript(js)
	case bsontype.Symbol:
		symbol, err := vr.ReadSymbol()
		if err != nil {
			return err
		}
		val = Symbol(symbol)
	case bsontype.CodeWithScope:
		code, scope, err := vr.ReadCodeWithScope()
		if err != nil {
			return err
		}
		var doc bsonx.Doc
		err = pc.decodeDocument(dc, scope, &doc)
		if err != nil {
			return err
		}
		val = CodeWithScope(code, doc)
	case bsontype.Int32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		val = Int32(i32)
	case bsontype.Timestamp:
		t, i, err := vr.ReadTimestamp()
		if err != nil {
			return err
		}
		val = Timestamp(t, i)
	case bsontype.Int64:
		i64, err := vr.ReadInt64()
		if err != nil {
			return err
		}
		val = Int64(i64)
	case bsontype.Decimal128:
		d128, err := vr.ReadDecimal128()
		if err != nil {
			return err
		}
		val = Decimal128(d128)
	case bsontype.MinKey:
		err := vr.ReadMinKey()
		if err != nil {
			return err
		}
		val = MinKey()
	case bsontype.MaxKey:
		err := vr.ReadMaxKey()
		if err != nil {
			return err
		}
		val = MaxKey()
	default:
		return fmt.Errorf("Cannot read unknown BSON type %s", vr.Type())
	}

	*elem = Elem{Key: key, Value: val}
	return nil
}

func (pc PrimitiveCodecs) decodeDocument(dctx bsoncodec.DecodeContext, dr bsonrw.DocumentReader, pdoc *bsonx.Doc) error {
	if *pdoc == nil {
		*pdoc = make(bsonx.Doc, 0)
	}
	*pdoc = (*pdoc)[:0]
	for {
		key, vr, err := dr.ReadElement()
		if err == bsonrw.ErrEOD {
			break
		}
		if err != nil {
			return err
		}

		var elem Elem
		err = pc.elementDecodeValue(dctx, vr, key, &elem)
		if err != nil {
			return err
		}

		*pdoc = append(*pdoc, elem)
	}
	return nil
}

// encodeValue does not validation, and the callers must perform validation on val before calling
// this method.
func (pc PrimitiveCodecs) encodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val Val) error {
	var err error
	switch val.Type() {
	case bsontype.Double:
		err = vw.WriteDouble(val.Double())
	case bsontype.String:
		err = vw.WriteString(val.StringValue())
	case bsontype.EmbeddedDocument:
		var encoder bsoncodec.ValueEncoder
		encoder, err = ec.LookupEncoder(tDocument)
		if err != nil {
			break
		}
		err = encoder.EncodeValue(ec, vw, val.Document())
	case bsontype.Array:
		var encoder bsoncodec.ValueEncoder
		encoder, err = ec.LookupEncoder(tArray)
		if err != nil {
			break
		}
		err = encoder.EncodeValue(ec, vw, val.Array())
	case bsontype.Binary:
		// TODO: FIX THIS (╯°□°）╯︵ ┻━┻
		subtype, data := val.Binary()
		err = vw.WriteBinaryWithSubtype(data, subtype)
	case bsontype.Undefined:
		err = vw.WriteUndefined()
	case bsontype.ObjectID:
		err = vw.WriteObjectID(val.ObjectID())
	case bsontype.Boolean:
		err = vw.WriteBoolean(val.Boolean())
	case bsontype.DateTime:
		err = vw.WriteDateTime(val.DateTime())
	case bsontype.Null:
		err = vw.WriteNull()
	case bsontype.Regex:
		err = vw.WriteRegex(val.Regex())
	case bsontype.DBPointer:
		err = vw.WriteDBPointer(val.DBPointer())
	case bsontype.JavaScript:
		err = vw.WriteJavascript(val.JavaScript())
	case bsontype.Symbol:
		err = vw.WriteSymbol(val.Symbol())
	case bsontype.CodeWithScope:
		code, scope := val.CodeWithScope()

		var cwsw bsonrw.DocumentWriter
		cwsw, err = vw.WriteCodeWithScope(code)
		if err != nil {
			break
		}

		err = pc.encodeDocument(ec, cwsw, scope)
	case bsontype.Int32:
		err = vw.WriteInt32(val.Int32())
	case bsontype.Timestamp:
		err = vw.WriteTimestamp(val.Timestamp())
	case bsontype.Int64:
		err = vw.WriteInt64(val.Int64())
	case bsontype.Decimal128:
		err = vw.WriteDecimal128(val.Decimal128())
	case bsontype.MinKey:
		err = vw.WriteMinKey()
	case bsontype.MaxKey:
		err = vw.WriteMaxKey()
	default:
		err = fmt.Errorf("%T is not a valid BSON type to encode", val.Type())
	}

	return err
}

func (pc PrimitiveCodecs) valueDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, val *Val) error {
	switch vr.Type() {
	case bsontype.Double:
		f64, err := vr.ReadDouble()
		if err != nil {
			return err
		}
		*val = Double(f64)
	case bsontype.String:
		str, err := vr.ReadString()
		if err != nil {
			return err
		}
		*val = String(str)
	case bsontype.EmbeddedDocument:
		var embeddedDoc bsonx.Doc
		err := pc.DocumentDecodeValue(dc, vr, &embeddedDoc)
		if err != nil {
			return err
		}
		*val = Document(embeddedDoc)
	case bsontype.Array:
		var arr Arr
		err := pc.ArrayDecodeValue(dc, vr, &arr)
		if err != nil {
			return err
		}
		*val = Array(arr)
	case bsontype.Binary:
		data, subtype, err := vr.ReadBinary()
		if err != nil {
			return err
		}
		*val = Binary(subtype, data)
	case bsontype.Undefined:
		err := vr.ReadUndefined()
		if err != nil {
			return err
		}
		*val = Undefined()
	case bsontype.ObjectID:
		oid, err := vr.ReadObjectID()
		if err != nil {
			return err
		}
		*val = ObjectID(oid)
	case bsontype.Boolean:
		b, err := vr.ReadBoolean()
		if err != nil {
			return err
		}
		*val = Boolean(b)
	case bsontype.DateTime:
		dt, err := vr.ReadDateTime()
		if err != nil {
			return err
		}
		*val = DateTime(dt)
	case bsontype.Null:
		err := vr.ReadNull()
		if err != nil {
			return err
		}
		*val = Null()
	case bsontype.Regex:
		pattern, options, err := vr.ReadRegex()
		if err != nil {
			return err
		}
		*val = Regex(pattern, options)
	case bsontype.DBPointer:
		ns, pointer, err := vr.ReadDBPointer()
		if err != nil {
			return err
		}
		*val = DBPointer(ns, pointer)
	case bsontype.JavaScript:
		js, err := vr.ReadJavascript()
		if err != nil {
			return err
		}
		*val = JavaScript(js)
	case bsontype.Symbol:
		symbol, err := vr.ReadSymbol()
		if err != nil {
			return err
		}
		*val = Symbol(symbol)
	case bsontype.CodeWithScope:
		code, scope, err := vr.ReadCodeWithScope()
		if err != nil {
			return err
		}
		var scopeDoc bsonx.Doc
		err = pc.decodeDocument(dc, scope, &scopeDoc)
		if err != nil {
			return err
		}
		*val = CodeWithScope(code, scopeDoc)
	case bsontype.Int32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		*val = Int32(i32)
	case bsontype.Timestamp:
		t, i, err := vr.ReadTimestamp()
		if err != nil {
			return err
		}
		*val = Timestamp(t, i)
	case bsontype.Int64:
		i64, err := vr.ReadInt64()
		if err != nil {
			return err
		}
		*val = Int64(i64)
	case bsontype.Decimal128:
		d128, err := vr.ReadDecimal128()
		if err != nil {
			return err
		}
		*val = Decimal128(d128)
	case bsontype.MinKey:
		err := vr.ReadMinKey()
		if err != nil {
			return err
		}
		*val = MinKey()
	case bsontype.MaxKey:
		err := vr.ReadMaxKey()
		if err != nil {
			return err
		}
		*val = MaxKey()
	default:
		return fmt.Errorf("Cannot read unknown BSON type %s", vr.Type())
	}

	return nil
}
