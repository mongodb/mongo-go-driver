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

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

var bvwPool = NewValueWriterPool()

var errInvalidValue = errors.New("cannot encode invalid element")

var sliceWriterPool = sync.Pool{
	New: func() interface{} {
		sw := make(SliceWriter, 0)
		return &sw
	},
}

func encodeElement(reg EncoderRegistry, dw DocumentWriter, e E) error {
	vw, err := dw.WriteDocumentElement(e.Key)
	if err != nil {
		return err
	}

	if e.Value == nil {
		return vw.WriteNull()
	}
	encoder, err := reg.LookupEncoder(reflect.TypeOf(e.Value))
	if err != nil {
		return err
	}

	err = encoder.EncodeValue(reg, vw, reflect.ValueOf(e.Value))
	if err != nil {
		return err
	}
	return nil
}

// registerDefaultEncoders will register the default encoder methods with the provided Registry.
func registerDefaultEncoders(rb *RegistryBuilder) {
	if rb == nil {
		panic(errors.New("argument to RegisterDefaultEncoders must not be nil"))
	}

	numEncoder := func() ValueEncoder { return &numCodec{} }
	rb.RegisterTypeEncoder(tByteSlice, func() ValueEncoder { return &byteSliceCodec{} }).
		RegisterTypeEncoder(tTime, func() ValueEncoder { return &timeCodec{} }).
		RegisterTypeEncoder(tEmpty, func() ValueEncoder { return &emptyInterfaceCodec{} }).
		RegisterTypeEncoder(tCoreArray, func() ValueEncoder { return &arrayCodec{} }).
		RegisterTypeEncoder(tOID, func() ValueEncoder { return ValueEncoderFunc(objectIDEncodeValue) }).
		RegisterTypeEncoder(tDecimal, func() ValueEncoder { return ValueEncoderFunc(decimal128EncodeValue) }).
		RegisterTypeEncoder(tJSONNumber, func() ValueEncoder { return ValueEncoderFunc(jsonNumberEncodeValue) }).
		RegisterTypeEncoder(tURL, func() ValueEncoder { return ValueEncoderFunc(urlEncodeValue) }).
		RegisterTypeEncoder(tJavaScript, func() ValueEncoder { return ValueEncoderFunc(javaScriptEncodeValue) }).
		RegisterTypeEncoder(tSymbol, func() ValueEncoder { return ValueEncoderFunc(symbolEncodeValue) }).
		RegisterTypeEncoder(tBinary, func() ValueEncoder { return ValueEncoderFunc(binaryEncodeValue) }).
		RegisterTypeEncoder(tUndefined, func() ValueEncoder { return ValueEncoderFunc(undefinedEncodeValue) }).
		RegisterTypeEncoder(tDateTime, func() ValueEncoder { return ValueEncoderFunc(dateTimeEncodeValue) }).
		RegisterTypeEncoder(tNull, func() ValueEncoder { return ValueEncoderFunc(nullEncodeValue) }).
		RegisterTypeEncoder(tRegex, func() ValueEncoder { return ValueEncoderFunc(regexEncodeValue) }).
		RegisterTypeEncoder(tDBPointer, func() ValueEncoder { return ValueEncoderFunc(dbPointerEncodeValue) }).
		RegisterTypeEncoder(tTimestamp, func() ValueEncoder { return ValueEncoderFunc(timestampEncodeValue) }).
		RegisterTypeEncoder(tMinKey, func() ValueEncoder { return ValueEncoderFunc(minKeyEncodeValue) }).
		RegisterTypeEncoder(tMaxKey, func() ValueEncoder { return ValueEncoderFunc(maxKeyEncodeValue) }).
		RegisterTypeEncoder(tCoreDocument, func() ValueEncoder { return ValueEncoderFunc(coreDocumentEncodeValue) }).
		RegisterTypeEncoder(tCodeWithScope, func() ValueEncoder { return ValueEncoderFunc(codeWithScopeEncodeValue) }).
		RegisterKindEncoder(reflect.Bool, func() ValueEncoder { return ValueEncoderFunc(booleanEncodeValue) }).
		RegisterKindEncoder(reflect.Int, numEncoder).
		RegisterKindEncoder(reflect.Int8, numEncoder).
		RegisterKindEncoder(reflect.Int16, numEncoder).
		RegisterKindEncoder(reflect.Int32, numEncoder).
		RegisterKindEncoder(reflect.Int64, numEncoder).
		RegisterKindEncoder(reflect.Uint, numEncoder).
		RegisterKindEncoder(reflect.Uint8, numEncoder).
		RegisterKindEncoder(reflect.Uint16, numEncoder).
		RegisterKindEncoder(reflect.Uint32, numEncoder).
		RegisterKindEncoder(reflect.Uint64, numEncoder).
		RegisterKindEncoder(reflect.Float32, numEncoder).
		RegisterKindEncoder(reflect.Float64, numEncoder).
		RegisterKindEncoder(reflect.Array, func() ValueEncoder { return ValueEncoderFunc(arrayEncodeValue) }).
		RegisterKindEncoder(reflect.Map, func() ValueEncoder { return &mapCodec{} }).
		RegisterKindEncoder(reflect.Slice, func() ValueEncoder { return &sliceCodec{} }).
		RegisterKindEncoder(reflect.String, func() ValueEncoder { return &stringCodec{} }).
		RegisterKindEncoder(reflect.Struct, func() ValueEncoder { return newStructCodec(DefaultStructTagParser) }).
		RegisterKindEncoder(reflect.Ptr, func() ValueEncoder { return &pointerCodec{} }).
		RegisterInterfaceEncoder(tValueMarshaler, func() ValueEncoder { return ValueEncoderFunc(valueMarshalerEncodeValue) }).
		RegisterInterfaceEncoder(tMarshaler, func() ValueEncoder { return ValueEncoderFunc(marshalerEncodeValue) }).
		RegisterInterfaceEncoder(tProxy, func() ValueEncoder { return ValueEncoderFunc(proxyEncodeValue) })
}

// booleanEncodeValue is the ValueEncoderFunc for bool types.
func booleanEncodeValue(_ EncoderRegistry, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Kind() != reflect.Bool {
		return ValueEncoderError{Name: "BooleanEncodeValue", Kinds: []reflect.Kind{reflect.Bool}, Received: val}
	}
	return vw.WriteBoolean(val.Bool())
}

func fitsIn32Bits(i int64) bool {
	return math.MinInt32 <= i && i <= math.MaxInt32
}

// objectIDEncodeValue is the ValueEncoderFunc for ObjectID.
func objectIDEncodeValue(_ EncoderRegistry, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tOID {
		return ValueEncoderError{Name: "ObjectIDEncodeValue", Types: []reflect.Type{tOID}, Received: val}
	}
	return vw.WriteObjectID(val.Interface().(ObjectID))
}

// decimal128EncodeValue is the ValueEncoderFunc for Decimal128.
func decimal128EncodeValue(_ EncoderRegistry, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tDecimal {
		return ValueEncoderError{Name: "Decimal128EncodeValue", Types: []reflect.Type{tDecimal}, Received: val}
	}
	return vw.WriteDecimal128(val.Interface().(Decimal128))
}

// jsonNumberEncodeValue is the ValueEncoderFunc for json.Number.
func jsonNumberEncodeValue(reg EncoderRegistry, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tJSONNumber {
		return ValueEncoderError{Name: "JSONNumberEncodeValue", Types: []reflect.Type{tJSONNumber}, Received: val}
	}
	jsnum := val.Interface().(json.Number)

	// Attempt int first, then float64
	if i64, err := jsnum.Int64(); err == nil {
		encoder, err := reg.LookupEncoder(tInt64)
		if err != nil {
			return err
		}
		return encoder.EncodeValue(reg, vw, reflect.ValueOf(i64))
	}

	f64, err := jsnum.Float64()
	if err != nil {
		return err
	}

	return (&numCodec{}).EncodeValue(reg, vw, reflect.ValueOf(f64))
}

// urlEncodeValue is the ValueEncoderFunc for url.URL.
func urlEncodeValue(_ EncoderRegistry, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tURL {
		return ValueEncoderError{Name: "URLEncodeValue", Types: []reflect.Type{tURL}, Received: val}
	}
	u := val.Interface().(url.URL)
	return vw.WriteString(u.String())
}

// arrayEncodeValue is the ValueEncoderFunc for array types.
func arrayEncodeValue(reg EncoderRegistry, vw ValueWriter, val reflect.Value) error {
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
			err = encodeElement(reg, dw, e)
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
	encoder, err := reg.LookupEncoder(elemType)
	if err != nil && elemType.Kind() != reflect.Interface {
		return err
	}

	for idx := 0; idx < val.Len(); idx++ {
		currEncoder, currVal, lookupErr := lookupElementEncoder(reg, encoder, val.Index(idx))
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

		err = currEncoder.EncodeValue(reg, vw, currVal)
		if err != nil {
			return err
		}
	}
	return aw.WriteArrayEnd()
}

func lookupElementEncoder(reg EncoderRegistry, origEncoder ValueEncoder, currVal reflect.Value) (ValueEncoder, reflect.Value, error) {
	if origEncoder != nil || (currVal.Kind() != reflect.Interface) {
		return origEncoder, currVal, nil
	}
	currVal = currVal.Elem()
	if !currVal.IsValid() {
		return nil, currVal, errInvalidValue
	}
	currEncoder, err := reg.LookupEncoder(currVal.Type())

	return currEncoder, currVal, err
}

// valueMarshalerEncodeValue is the ValueEncoderFunc for ValueMarshaler implementations.
func valueMarshalerEncodeValue(_ EncoderRegistry, vw ValueWriter, val reflect.Value) error {
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
	return copyValueFromBytes(vw, t, data)
}

// marshalerEncodeValue is the ValueEncoderFunc for Marshaler implementations.
func marshalerEncodeValue(_ EncoderRegistry, vw ValueWriter, val reflect.Value) error {
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

// proxyEncodeValue is the ValueEncoderFunc for Proxy implementations.
func proxyEncodeValue(reg EncoderRegistry, vw ValueWriter, val reflect.Value) error {
	// Either val or a pointer to val must implement Proxy
	switch {
	case !val.IsValid():
		return ValueEncoderError{Name: "ProxyEncodeValue", Types: []reflect.Type{tProxy}, Received: val}
	case val.Type().Implements(tProxy):
		// If Proxy is implemented on a concrete type, make sure that val isn't a nil pointer
		if isImplementationNil(val, tProxy) {
			return vw.WriteNull()
		}
	case reflect.PtrTo(val.Type()).Implements(tProxy) && val.CanAddr():
		val = val.Addr()
	default:
		return ValueEncoderError{Name: "ProxyEncodeValue", Types: []reflect.Type{tProxy}, Received: val}
	}

	m, ok := val.Interface().(Proxy)
	if !ok {
		return vw.WriteNull()
	}
	v, err := m.ProxyBSON()
	if err != nil {
		return err
	}
	if v == nil {
		encoder, err := reg.LookupEncoder(nil)
		if err != nil {
			return err
		}
		return encoder.EncodeValue(reg, vw, reflect.ValueOf(nil))
	}
	vv := reflect.ValueOf(v)
	switch vv.Kind() {
	case reflect.Ptr, reflect.Interface:
		vv = vv.Elem()
	}
	encoder, err := reg.LookupEncoder(vv.Type())
	if err != nil {
		return err
	}
	return encoder.EncodeValue(reg, vw, vv)
}

// javaScriptEncodeValue is the ValueEncoderFunc for the JavaScript type.
func javaScriptEncodeValue(_ EncoderRegistry, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tJavaScript {
		return ValueEncoderError{Name: "JavaScriptEncodeValue", Types: []reflect.Type{tJavaScript}, Received: val}
	}

	return vw.WriteJavascript(val.String())
}

// symbolEncodeValue is the ValueEncoderFunc for the Symbol type.
func symbolEncodeValue(_ EncoderRegistry, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tSymbol {
		return ValueEncoderError{Name: "SymbolEncodeValue", Types: []reflect.Type{tSymbol}, Received: val}
	}

	return vw.WriteSymbol(val.String())
}

// binaryEncodeValue is the ValueEncoderFunc for Binary.
func binaryEncodeValue(_ EncoderRegistry, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tBinary {
		return ValueEncoderError{Name: "BinaryEncodeValue", Types: []reflect.Type{tBinary}, Received: val}
	}
	b := val.Interface().(Binary)

	return vw.WriteBinaryWithSubtype(b.Data, b.Subtype)
}

// undefinedEncodeValue is the ValueEncoderFunc for Undefined.
func undefinedEncodeValue(_ EncoderRegistry, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tUndefined {
		return ValueEncoderError{Name: "UndefinedEncodeValue", Types: []reflect.Type{tUndefined}, Received: val}
	}

	return vw.WriteUndefined()
}

// dateTimeEncodeValue is the ValueEncoderFunc for DateTime.
func dateTimeEncodeValue(_ EncoderRegistry, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tDateTime {
		return ValueEncoderError{Name: "DateTimeEncodeValue", Types: []reflect.Type{tDateTime}, Received: val}
	}

	return vw.WriteDateTime(val.Int())
}

// nullEncodeValue is the ValueEncoderFunc for Null.
func nullEncodeValue(_ EncoderRegistry, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tNull {
		return ValueEncoderError{Name: "NullEncodeValue", Types: []reflect.Type{tNull}, Received: val}
	}

	return vw.WriteNull()
}

// regexEncodeValue is the ValueEncoderFunc for Regex.
func regexEncodeValue(_ EncoderRegistry, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tRegex {
		return ValueEncoderError{Name: "RegexEncodeValue", Types: []reflect.Type{tRegex}, Received: val}
	}

	regex := val.Interface().(Regex)

	return vw.WriteRegex(regex.Pattern, regex.Options)
}

// dbPointerEncodeValue is the ValueEncoderFunc for DBPointer.
func dbPointerEncodeValue(_ EncoderRegistry, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tDBPointer {
		return ValueEncoderError{Name: "DBPointerEncodeValue", Types: []reflect.Type{tDBPointer}, Received: val}
	}

	dbp := val.Interface().(DBPointer)

	return vw.WriteDBPointer(dbp.DB, dbp.Pointer)
}

// timestampEncodeValue is the ValueEncoderFunc for Timestamp.
func timestampEncodeValue(_ EncoderRegistry, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tTimestamp {
		return ValueEncoderError{Name: "TimestampEncodeValue", Types: []reflect.Type{tTimestamp}, Received: val}
	}

	ts := val.Interface().(Timestamp)

	return vw.WriteTimestamp(ts.T, ts.I)
}

// minKeyEncodeValue is the ValueEncoderFunc for MinKey.
func minKeyEncodeValue(_ EncoderRegistry, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tMinKey {
		return ValueEncoderError{Name: "MinKeyEncodeValue", Types: []reflect.Type{tMinKey}, Received: val}
	}

	return vw.WriteMinKey()
}

// maxKeyEncodeValue is the ValueEncoderFunc for MaxKey.
func maxKeyEncodeValue(_ EncoderRegistry, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tMaxKey {
		return ValueEncoderError{Name: "MaxKeyEncodeValue", Types: []reflect.Type{tMaxKey}, Received: val}
	}

	return vw.WriteMaxKey()
}

// coreDocumentEncodeValue is the ValueEncoderFunc for bsoncore.Document.
func coreDocumentEncodeValue(_ EncoderRegistry, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tCoreDocument {
		return ValueEncoderError{Name: "CoreDocumentEncodeValue", Types: []reflect.Type{tCoreDocument}, Received: val}
	}

	cdoc := val.Interface().(bsoncore.Document)

	return copyDocumentFromBytes(vw, cdoc)
}

// codeWithScopeEncodeValue is the ValueEncoderFunc for CodeWithScope.
func codeWithScopeEncodeValue(reg EncoderRegistry, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tCodeWithScope {
		return ValueEncoderError{Name: "CodeWithScopeEncodeValue", Types: []reflect.Type{tCodeWithScope}, Received: val}
	}

	cws := val.Interface().(CodeWithScope)

	dw, err := vw.WriteCodeWithScope(string(cws.Code))
	if err != nil {
		return err
	}

	sw := sliceWriterPool.Get().(*SliceWriter)
	defer sliceWriterPool.Put(sw)
	*sw = (*sw)[:0]

	scopeVW := bvwPool.Get(sw)
	defer bvwPool.Put(scopeVW)

	encoder, err := reg.LookupEncoder(reflect.TypeOf(cws.Scope))
	if err != nil {
		return err
	}

	err = encoder.EncodeValue(reg, scopeVW, reflect.ValueOf(cws.Scope))
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
