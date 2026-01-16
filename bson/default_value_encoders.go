// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"encoding/json"
	"errors"
	"math"
	"net/url"
	"reflect"
	"sync"

	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

var bvwPool = sync.Pool{
	New: func() any {
		return new(valueWriter)
	},
}

var errInvalidValue = errors.New("cannot encode invalid element")

var sliceWriterPool = sync.Pool{
	New: func() any {
		sw := make(sliceWriter, 0)
		return &sw
	},
}

func encodeElement(ec EncodeContext, dw DocumentWriter, e E) error {
	vw, err := dw.WriteDocumentElement(e.Key)
	if err != nil {
		return err
	}

	if e.Value == nil {
		return vw.WriteNull()
	}
	encoder, err := ec.LookupEncoder(reflect.TypeOf(e.Value))
	if err != nil {
		return err
	}

	err = encoder.EncodeValue(ec, vw, reflect.ValueOf(e.Value))
	if err != nil {
		return err
	}
	return nil
}

// registerDefaultEncoders will register the encoder methods attached to DefaultValueEncoders with
// the provided RegistryBuilder.
func registerDefaultEncoders(reg *Registry) {
	mapEncoder := &mapCodec{}
	uintCodec := &uintCodec{}

	reg.RegisterTypeEncoder(tByteSlice, &byteSliceCodec{})
	reg.RegisterTypeEncoder(tTime, &timeCodec{})
	reg.RegisterTypeEncoder(tEmpty, &emptyInterfaceCodec{})
	reg.RegisterTypeEncoder(tCoreArray, &arrayCodec{})
	reg.RegisterTypeEncoder(tOID, ValueEncoderFunc(objectIDEncodeValue))
	reg.RegisterTypeEncoder(tDecimal, ValueEncoderFunc(decimal128EncodeValue))
	reg.RegisterTypeEncoder(tJSONNumber, ValueEncoderFunc(jsonNumberEncodeValue))
	reg.RegisterTypeEncoder(tURL, ValueEncoderFunc(urlEncodeValue))
	reg.RegisterTypeEncoder(tJavaScript, ValueEncoderFunc(javaScriptEncodeValue))
	reg.RegisterTypeEncoder(tSymbol, ValueEncoderFunc(symbolEncodeValue))
	reg.RegisterTypeEncoder(tBinary, ValueEncoderFunc(binaryEncodeValue))
	reg.RegisterTypeEncoder(tVector, ValueEncoderFunc(vectorEncodeValue))
	reg.RegisterTypeEncoder(tUndefined, ValueEncoderFunc(undefinedEncodeValue))
	reg.RegisterTypeEncoder(tDateTime, ValueEncoderFunc(dateTimeEncodeValue))
	reg.RegisterTypeEncoder(tNull, ValueEncoderFunc(nullEncodeValue))
	reg.RegisterTypeEncoder(tRegex, ValueEncoderFunc(regexEncodeValue))
	reg.RegisterTypeEncoder(tDBPointer, ValueEncoderFunc(dbPointerEncodeValue))
	reg.RegisterTypeEncoder(tTimestamp, ValueEncoderFunc(timestampEncodeValue))
	reg.RegisterTypeEncoder(tMinKey, ValueEncoderFunc(minKeyEncodeValue))
	reg.RegisterTypeEncoder(tMaxKey, ValueEncoderFunc(maxKeyEncodeValue))
	reg.RegisterTypeEncoder(tCoreDocument, ValueEncoderFunc(coreDocumentEncodeValue))
	reg.RegisterTypeEncoder(tCodeWithScope, ValueEncoderFunc(codeWithScopeEncodeValue))
	reg.RegisterKindEncoder(reflect.Bool, ValueEncoderFunc(booleanEncodeValue))
	reg.RegisterKindEncoder(reflect.Int, ValueEncoderFunc(intEncodeValue))
	reg.RegisterKindEncoder(reflect.Int8, ValueEncoderFunc(intEncodeValue))
	reg.RegisterKindEncoder(reflect.Int16, ValueEncoderFunc(intEncodeValue))
	reg.RegisterKindEncoder(reflect.Int32, ValueEncoderFunc(intEncodeValue))
	reg.RegisterKindEncoder(reflect.Int64, ValueEncoderFunc(intEncodeValue))
	reg.RegisterKindEncoder(reflect.Uint, uintCodec)
	reg.RegisterKindEncoder(reflect.Uint8, uintCodec)
	reg.RegisterKindEncoder(reflect.Uint16, uintCodec)
	reg.RegisterKindEncoder(reflect.Uint32, uintCodec)
	reg.RegisterKindEncoder(reflect.Uint64, uintCodec)
	reg.RegisterKindEncoder(reflect.Float32, ValueEncoderFunc(floatEncodeValue))
	reg.RegisterKindEncoder(reflect.Float64, ValueEncoderFunc(floatEncodeValue))
	reg.RegisterKindEncoder(reflect.Array, ValueEncoderFunc(arrayEncodeValue))
	reg.RegisterKindEncoder(reflect.Map, mapEncoder)
	reg.RegisterKindEncoder(reflect.Slice, &sliceCodec{})
	reg.RegisterKindEncoder(reflect.String, &stringCodec{})
	reg.RegisterKindEncoder(reflect.Struct, newStructCodec(mapEncoder))
	reg.RegisterKindEncoder(reflect.Ptr, &pointerCodec{})
	reg.RegisterInterfaceEncoder(tValueMarshaler, ValueEncoderFunc(valueMarshalerEncodeValue))
	reg.RegisterInterfaceEncoder(tMarshaler, ValueEncoderFunc(marshalerEncodeValue))
}

// booleanEncodeValue is the ValueEncoderFunc for bool types.
func booleanEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Kind() != reflect.Bool {
		return ValueEncoderError{Name: "BooleanEncodeValue", Kinds: []reflect.Kind{reflect.Bool}, Received: val}
	}
	return vw.WriteBoolean(val.Bool())
}

func fitsIn32Bits(i int64) bool {
	return math.MinInt32 <= i && i <= math.MaxInt32
}

// intEncodeValue is the ValueEncoderFunc for int types.
func intEncodeValue(ec EncodeContext, vw ValueWriter, val reflect.Value) error {
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
		if ec.minSize && fitsIn32Bits(i64) {
			return vw.WriteInt32(int32(i64))
		}
		return vw.WriteInt64(i64)
	}

	return ValueEncoderError{
		Name:     "IntEncodeValue",
		Kinds:    []reflect.Kind{reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int},
		Received: val,
	}
}

// floatEncodeValue is the ValueEncoderFunc for float types.
func floatEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	switch val.Kind() {
	case reflect.Float32, reflect.Float64:
		return vw.WriteDouble(val.Float())
	}

	return ValueEncoderError{Name: "FloatEncodeValue", Kinds: []reflect.Kind{reflect.Float32, reflect.Float64}, Received: val}
}

// objectIDEncodeValue is the ValueEncoderFunc for ObjectID.
func objectIDEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tOID {
		return ValueEncoderError{Name: "ObjectIDEncodeValue", Types: []reflect.Type{tOID}, Received: val}
	}
	return vw.WriteObjectID(val.Interface().(ObjectID))
}

// decimal128EncodeValue is the ValueEncoderFunc for Decimal128.
func decimal128EncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tDecimal {
		return ValueEncoderError{Name: "Decimal128EncodeValue", Types: []reflect.Type{tDecimal}, Received: val}
	}
	return vw.WriteDecimal128(val.Interface().(Decimal128))
}

// jsonNumberEncodeValue is the ValueEncoderFunc for json.Number.
func jsonNumberEncodeValue(ec EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tJSONNumber {
		return ValueEncoderError{Name: "JSONNumberEncodeValue", Types: []reflect.Type{tJSONNumber}, Received: val}
	}
	jsnum := val.Interface().(json.Number)

	// Attempt int first, then float64
	if i64, err := jsnum.Int64(); err == nil {
		return intEncodeValue(ec, vw, reflect.ValueOf(i64))
	}

	f64, err := jsnum.Float64()
	if err != nil {
		return err
	}

	return floatEncodeValue(ec, vw, reflect.ValueOf(f64))
}

// urlEncodeValue is the ValueEncoderFunc for url.URL.
func urlEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tURL {
		return ValueEncoderError{Name: "URLEncodeValue", Types: []reflect.Type{tURL}, Received: val}
	}
	u := val.Interface().(url.URL)
	return vw.WriteString(u.String())
}

// arrayEncodeValue is the ValueEncoderFunc for array types.
func arrayEncodeValue(ec EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Kind() != reflect.Array {
		return ValueEncoderError{Name: "ArrayEncodeValue", Kinds: []reflect.Kind{reflect.Array}, Received: val}
	}

	// If we have a []E we want to treat it as a document instead of as an array.
	if val.Type().Elem() == tE {
		dw, err := vw.WriteDocument()
		if err != nil {
			return err
		}

		for idx := 0; idx < val.Len(); idx++ {
			e := val.Index(idx).Interface().(E)
			err = encodeElement(ec, dw, e)
			if err != nil {
				return err
			}
		}

		return dw.WriteDocumentEnd()
	}

	// If we have a []byte we want to treat it as a binary instead of as an array.
	if val.Type().Elem() == tByte {
		var byteSlice []byte
		for idx := 0; idx < val.Len(); idx++ {
			byteSlice = append(byteSlice, val.Index(idx).Interface().(byte))
		}
		return vw.WriteBinary(byteSlice)
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

func lookupElementEncoder(ec EncodeContext, origEncoder ValueEncoder, currVal reflect.Value) (ValueEncoder, reflect.Value, error) {
	if origEncoder != nil || (currVal.Kind() != reflect.Interface) {
		return origEncoder, currVal, nil
	}
	currVal = currVal.Elem()
	if !currVal.IsValid() {
		return nil, currVal, errInvalidValue
	}
	currEncoder, err := ec.LookupEncoder(currVal.Type())

	return currEncoder, currVal, err
}

// valueMarshalerEncodeValue is the ValueEncoderFunc for ValueMarshaler implementations.
func valueMarshalerEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	// Either val or a pointer to val must implement ValueMarshaler
	switch {
	case !val.IsValid():
		return ValueEncoderError{Name: "ValueMarshalerEncodeValue", Types: []reflect.Type{tValueMarshaler}, Received: val}
	case val.Type().Implements(tValueMarshaler):
		// If ValueMarshaler is implemented on a concrete type, make sure that val isn't a nil pointer
		if isImplementationNil(val, tValueMarshaler) {
			return vw.WriteNull()
		}
	case reflect.PtrTo(val.Type()).Implements(tValueMarshaler) && val.CanAddr():
		val = val.Addr()
	default:
		return ValueEncoderError{Name: "ValueMarshalerEncodeValue", Types: []reflect.Type{tValueMarshaler}, Received: val}
	}

	m, ok := val.Interface().(ValueMarshaler)
	if !ok {
		return vw.WriteNull()
	}
	t, data, err := m.MarshalBSONValue()
	if err != nil {
		return err
	}
	return copyValueFromBytes(vw, Type(t), data)
}

// marshalerEncodeValue is the ValueEncoderFunc for Marshaler implementations.
func marshalerEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	// Either val or a pointer to val must implement Marshaler
	switch {
	case !val.IsValid():
		return ValueEncoderError{Name: "MarshalerEncodeValue", Types: []reflect.Type{tMarshaler}, Received: val}
	case val.Type().Implements(tMarshaler):
		// If Marshaler is implemented on a concrete type, make sure that val isn't a nil pointer
		if isImplementationNil(val, tMarshaler) {
			return vw.WriteNull()
		}
	case reflect.PtrTo(val.Type()).Implements(tMarshaler) && val.CanAddr():
		val = val.Addr()
	default:
		return ValueEncoderError{Name: "MarshalerEncodeValue", Types: []reflect.Type{tMarshaler}, Received: val}
	}

	m, ok := val.Interface().(Marshaler)
	if !ok {
		return vw.WriteNull()
	}
	data, err := m.MarshalBSON()
	if err != nil {
		return err
	}
	return copyValueFromBytes(vw, TypeEmbeddedDocument, data)
}

// javaScriptEncodeValue is the ValueEncoderFunc for the JavaScript type.
func javaScriptEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tJavaScript {
		return ValueEncoderError{Name: "JavaScriptEncodeValue", Types: []reflect.Type{tJavaScript}, Received: val}
	}

	return vw.WriteJavascript(val.String())
}

// symbolEncodeValue is the ValueEncoderFunc for the Symbol type.
func symbolEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tSymbol {
		return ValueEncoderError{Name: "SymbolEncodeValue", Types: []reflect.Type{tSymbol}, Received: val}
	}

	return vw.WriteSymbol(val.String())
}

// binaryEncodeValue is the ValueEncoderFunc for Binary.
func binaryEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tBinary {
		return ValueEncoderError{Name: "BinaryEncodeValue", Types: []reflect.Type{tBinary}, Received: val}
	}
	b := val.Interface().(Binary)

	return vw.WriteBinaryWithSubtype(b.Data, b.Subtype)
}

// vectorEncodeValue is the ValueEncoderFunc for Vector.
func vectorEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	t := val.Type()
	if !val.IsValid() || t != tVector {
		return ValueEncoderError{
			Name:     "VectorEncodeValue",
			Types:    []reflect.Type{tVector},
			Received: val,
		}
	}
	v := val.Interface().(Vector)
	b := v.Binary()
	return vw.WriteBinaryWithSubtype(b.Data, b.Subtype)
}

// undefinedEncodeValue is the ValueEncoderFunc for Undefined.
func undefinedEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tUndefined {
		return ValueEncoderError{Name: "UndefinedEncodeValue", Types: []reflect.Type{tUndefined}, Received: val}
	}

	return vw.WriteUndefined()
}

// dateTimeEncodeValue is the ValueEncoderFunc for DateTime.
func dateTimeEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tDateTime {
		return ValueEncoderError{Name: "DateTimeEncodeValue", Types: []reflect.Type{tDateTime}, Received: val}
	}

	return vw.WriteDateTime(val.Int())
}

// nullEncodeValue is the ValueEncoderFunc for Null.
func nullEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tNull {
		return ValueEncoderError{Name: "NullEncodeValue", Types: []reflect.Type{tNull}, Received: val}
	}

	return vw.WriteNull()
}

// regexEncodeValue is the ValueEncoderFunc for Regex.
func regexEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tRegex {
		return ValueEncoderError{Name: "RegexEncodeValue", Types: []reflect.Type{tRegex}, Received: val}
	}

	regex := val.Interface().(Regex)

	return vw.WriteRegex(regex.Pattern, regex.Options)
}

// dbPointerEncodeValue is the ValueEncoderFunc for DBPointer.
func dbPointerEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tDBPointer {
		return ValueEncoderError{Name: "DBPointerEncodeValue", Types: []reflect.Type{tDBPointer}, Received: val}
	}

	dbp := val.Interface().(DBPointer)

	return vw.WriteDBPointer(dbp.DB, dbp.Pointer)
}

// timestampEncodeValue is the ValueEncoderFunc for Timestamp.
func timestampEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tTimestamp {
		return ValueEncoderError{Name: "TimestampEncodeValue", Types: []reflect.Type{tTimestamp}, Received: val}
	}

	ts := val.Interface().(Timestamp)

	return vw.WriteTimestamp(ts.T, ts.I)
}

// minKeyEncodeValue is the ValueEncoderFunc for MinKey.
func minKeyEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tMinKey {
		return ValueEncoderError{Name: "MinKeyEncodeValue", Types: []reflect.Type{tMinKey}, Received: val}
	}

	return vw.WriteMinKey()
}

// maxKeyEncodeValue is the ValueEncoderFunc for MaxKey.
func maxKeyEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tMaxKey {
		return ValueEncoderError{Name: "MaxKeyEncodeValue", Types: []reflect.Type{tMaxKey}, Received: val}
	}

	return vw.WriteMaxKey()
}

// coreDocumentEncodeValue is the ValueEncoderFunc for bsoncore.Document.
func coreDocumentEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tCoreDocument {
		return ValueEncoderError{Name: "CoreDocumentEncodeValue", Types: []reflect.Type{tCoreDocument}, Received: val}
	}

	cdoc := val.Interface().(bsoncore.Document)

	return copyDocumentFromBytes(vw, cdoc)
}

// codeWithScopeEncodeValue is the ValueEncoderFunc for CodeWithScope.
func codeWithScopeEncodeValue(ec EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tCodeWithScope {
		return ValueEncoderError{Name: "CodeWithScopeEncodeValue", Types: []reflect.Type{tCodeWithScope}, Received: val}
	}

	cws := val.Interface().(CodeWithScope)

	dw, err := vw.WriteCodeWithScope(string(cws.Code))
	if err != nil {
		return err
	}

	sw := sliceWriterPool.Get().(*sliceWriter)
	defer sliceWriterPool.Put(sw)
	*sw = (*sw)[:0]

	scopeVW := bvwPool.Get().(*valueWriter)
	scopeVW.reset(scopeVW.buf[:0])
	scopeVW.w = sw
	defer bvwPool.Put(scopeVW)

	encoder, err := ec.LookupEncoder(reflect.TypeOf(cws.Scope))
	if err != nil {
		return err
	}

	err = encoder.EncodeValue(ec, scopeVW, reflect.ValueOf(cws.Scope))
	if err != nil {
		return err
	}

	err = copyBytesToDocumentWriter(dw, *sw)
	if err != nil {
		return err
	}
	return dw.WriteDocumentEnd()
}

// isImplementationNil returns if val is a nil pointer and inter is implemented on a concrete type
func isImplementationNil(val reflect.Value, inter reflect.Type) bool {
	vt := val.Type()
	for vt.Kind() == reflect.Ptr {
		vt = vt.Elem()
	}
	return vt.Implements(inter) && val.Kind() == reflect.Ptr && val.IsNil()
}
