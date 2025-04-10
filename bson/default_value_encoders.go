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
	"time"

	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

var bvwPool = sync.Pool{
	New: func() interface{} {
		return new(valueWriter)
	},
}

var errInvalidValue = errors.New("cannot encode invalid element")

var sliceWriterPool = sync.Pool{
	New: func() interface{} {
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

	// Register the reflect-free default type encoders.
	reg.registerReflectFreeTypeEncoder(tByteSlice, byteSliceEncodeValueRF(false))
	reg.registerReflectFreeTypeEncoder(tTime, reflectFreeValueEncoderFunc(timeEncodeValueRF))
	reg.registerReflectFreeTypeEncoder(tEmpty, reflectFreeValueEncoderFunc(emptyInterfaceValueRF))
	reg.registerReflectFreeTypeEncoder(tCoreArray, reflectFreeValueEncoderFunc(coreArrayEncodeValueRF))
	reg.registerReflectFreeTypeEncoder(tNull, reflectFreeValueEncoderFunc(nullEncodeValueRF))
	reg.registerReflectFreeTypeEncoder(tOID, reflectFreeValueEncoderFunc(objectIDEncodeValueRF))
	reg.registerReflectFreeTypeEncoder(tDecimal, reflectFreeValueEncoderFunc(decimal128EncodeValueRF))
	reg.registerReflectFreeTypeEncoder(tJSONNumber, reflectFreeValueEncoderFunc(jsonNumberEncodeValueRF))
	reg.registerReflectFreeTypeEncoder(tURL, reflectFreeValueEncoderFunc(urlEncodeValueRF))
	reg.registerReflectFreeTypeEncoder(tJavaScript, reflectFreeValueEncoderFunc(javaScriptEncodeValueRF))
	reg.registerReflectFreeTypeEncoder(tSymbol, reflectFreeValueEncoderFunc(symbolEncodeValueRF))
	reg.registerReflectFreeTypeEncoder(tBinary, reflectFreeValueEncoderFunc(binaryEncodeValueRF))
	reg.registerReflectFreeTypeEncoder(tVector, reflectFreeValueEncoderFunc(vectorEncodeValueRF))
	reg.registerReflectFreeTypeEncoder(tUndefined, reflectFreeValueEncoderFunc(undefinedEncodeValueRF))
	reg.registerReflectFreeTypeEncoder(tDateTime, reflectFreeValueEncoderFunc(dateTimeEncodeValueRF))
	reg.registerReflectFreeTypeEncoder(tRegex, reflectFreeValueEncoderFunc(regexEncodeValueX))
	reg.registerReflectFreeTypeEncoder(tDBPointer, reflectFreeValueEncoderFunc(dbPointerEncodeValueX))
	reg.registerReflectFreeTypeEncoder(tTimestamp, reflectFreeValueEncoderFunc(timestampEncodeValueX))
	reg.registerReflectFreeTypeEncoder(tMinKey, reflectFreeValueEncoderFunc(minKeyEncodeValueX))
	reg.registerReflectFreeTypeEncoder(tMaxKey, reflectFreeValueEncoderFunc(maxKeyEncodeValueX))
	reg.registerReflectFreeTypeEncoder(tCoreDocument, reflectFreeValueEncoderFunc(coreDocumentEncodeValueX))
	reg.registerReflectFreeTypeEncoder(tCodeWithScope, reflectFreeValueEncoderFunc(codeWithScopeEncodeValueX))

	// Register the reflect-based default encoders. These are required since
	// removing them would break Registry.LookupEncoder. However, these will
	// never be used internally.
	//
	reg.RegisterTypeEncoder(tByteSlice, byteSliceEncodeValue(false))
	reg.RegisterTypeEncoder(tTime, defaultValueEncoderFunc(timeEncodeValue))
	reg.RegisterTypeEncoder(tEmpty, defaultValueEncoderFunc(emptyInterfaceValue)) // TODO: extend this to reflection free
	reg.RegisterTypeEncoder(tCoreArray, defaultValueEncoderFunc(coreArrayEncodeValue))
	reg.RegisterTypeEncoder(tOID, defaultValueEncoderFunc(objectIDEncodeValue))
	reg.RegisterTypeEncoder(tDecimal, defaultValueEncoderFunc(decimal128EncodeValue))
	reg.RegisterTypeEncoder(tJSONNumber, defaultValueEncoderFunc(jsonNumberEncodeValue))
	reg.RegisterTypeEncoder(tURL, defaultValueEncoderFunc(urlEncodeValue))
	reg.RegisterTypeEncoder(tJavaScript, defaultValueEncoderFunc(javaScriptEncodeValue))
	reg.RegisterTypeEncoder(tSymbol, defaultValueEncoderFunc(symbolEncodeValue))
	reg.RegisterTypeEncoder(tBinary, defaultValueEncoderFunc(binaryEncodeValue))
	reg.RegisterTypeEncoder(tVector, defaultValueEncoderFunc(vectorEncodeValue))
	reg.RegisterTypeEncoder(tUndefined, defaultValueEncoderFunc(undefinedEncodeValue))
	reg.RegisterTypeEncoder(tDateTime, defaultValueEncoderFunc(dateTimeEncodeValue))
	reg.RegisterTypeEncoder(tNull, defaultValueEncoderFunc(nullEncodeValue))
	reg.RegisterTypeEncoder(tRegex, defaultValueEncoderFunc(regexEncodeValue))
	reg.RegisterTypeEncoder(tDBPointer, defaultValueEncoderFunc(dbPointerEncodeValue))
	reg.RegisterTypeEncoder(tTimestamp, defaultValueEncoderFunc(timestampEncodeValue))
	reg.RegisterTypeEncoder(tMinKey, defaultValueEncoderFunc(minKeyEncodeValue))
	reg.RegisterTypeEncoder(tMaxKey, defaultValueEncoderFunc(maxKeyEncodeValue))
	reg.RegisterTypeEncoder(tCoreDocument, defaultValueEncoderFunc(coreDocumentEncodeValue))
	reg.RegisterTypeEncoder(tCodeWithScope, defaultValueEncoderFunc(codeWithScopeEncodeValue))

	// Register the kind-based default encoders. These must continue using
	// reflection since they account for custom types that cannot be anticipated.
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

	// Register the interface-based default encoders.
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

func floatEncodeValueRF(_ EncodeContext, vw ValueWriter, val any) error {
	if f32, ok := val.(float32); ok {
		return vw.WriteDouble(float64(f32))
	}

	if f64, ok := val.(float64); ok {
		return vw.WriteDouble(f64)
	}

	return ValueEncoderError{Name: "FloatEncodeValue", Kinds: []reflect.Kind{reflect.Float32, reflect.Float64}, Received: reflect.ValueOf(val)}
}

// floatEncodeValue is the ValueEncoderFunc for float types. this function is
// used to decode "types" and "kinds" and therefore cannot be a wrapper for
// reflection-free decoding in the default "type" case.
func floatEncodeValue(ec EncodeContext, vw ValueWriter, val reflect.Value) error {
	switch val.Kind() {
	case reflect.Float32, reflect.Float64:
		return vw.WriteDouble(val.Float())
	}

	return ValueEncoderError{Name: "FloatEncodeValue", Kinds: []reflect.Kind{reflect.Float32, reflect.Float64}, Received: val}
}

// objectIDEncodeValue is the ValueEncoderFunc for ObjectID.
func objectIDEncodeValueRF(_ EncodeContext, vw ValueWriter, val any) error {
	objID, ok := val.(ObjectID)
	if !ok {
		return ValueEncoderError{
			Name:     "ObjectIDEncodeValue",
			Types:    []reflect.Type{tOID},
			Received: reflect.ValueOf(val),
		}
	}

	return vw.WriteObjectID(objID)
}

// objectIDEncodeValue is the ValueEncoderFunc for ObjectID.
func objectIDEncodeValue(ec EncodeContext, vw ValueWriter, val reflect.Value) error {
	return objectIDEncodeValueRF(ec, vw, val.Interface())
}

func decimal128EncodeValueRF(_ EncodeContext, vw ValueWriter, val any) error {
	d128, ok := val.(Decimal128)
	if !ok {
		return ValueEncoderError{Name: "Decimal128EncodeValue", Types: []reflect.Type{tDecimal}, Received: reflect.ValueOf(val)}
	}

	return vw.WriteDecimal128(d128)
}

// decimal128EncodeValue is the ValueEncoderFunc for Decimal128.
func decimal128EncodeValue(ec EncodeContext, vw ValueWriter, val reflect.Value) error {
	return decimal128EncodeValueRF(ec, vw, val.Interface())
}

func jsonNumberEncodeValueRF(ec EncodeContext, vw ValueWriter, val any) error {
	jsnum, ok := val.(json.Number)
	if !ok {
		return ValueEncoderError{Name: "JSONNumberEncodeValue", Types: []reflect.Type{tJSONNumber}, Received: reflect.ValueOf(val)}
	}

	// Attempt int first, then float64
	if i64, err := jsnum.Int64(); err == nil {
		return intEncodeValue(ec, vw, reflect.ValueOf(i64))
	}

	f64, err := jsnum.Float64()
	if err != nil {
		return err
	}

	return floatEncodeValueRF(ec, vw, f64)
}

// jsonNumberEncodeValue is the ValueEncoderFunc for json.Number.
func jsonNumberEncodeValue(ec EncodeContext, vw ValueWriter, val reflect.Value) error {
	return jsonNumberEncodeValueRF(ec, vw, val.Interface())
}

func urlEncodeValueRF(_ EncodeContext, vw ValueWriter, val any) error {
	u, ok := val.(url.URL)
	if !ok {
		return ValueEncoderError{Name: "URLEncodeValue", Types: []reflect.Type{tURL}, Received: reflect.ValueOf(val)}
	}

	return vw.WriteString(u.String())
}

// urlEncodeValue is the ValueEncoderFunc for url.URL.
func urlEncodeValue(ec EncodeContext, vw ValueWriter, val reflect.Value) error {
	return urlEncodeValueRF(ec, vw, val.Interface())
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

func javaScriptEncodeValueRF(_ EncodeContext, vw ValueWriter, val any) error {
	jsString, ok := val.(JavaScript)
	if !ok {
		return ValueEncoderError{Name: "JavaScriptEncodeValue", Types: []reflect.Type{tJavaScript}, Received: reflect.ValueOf(val)}
	}

	return vw.WriteJavascript(string(jsString))
}

// javaScriptEncodeValue is the ValueEncoderFunc for the JavaScript type.
func javaScriptEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	return javaScriptEncodeValueRF(EncodeContext{}, vw, val.Interface())
}

func symbolEncodeValueRF(_ EncodeContext, vw ValueWriter, val any) error {
	symbol, ok := val.(Symbol)
	if !ok {
		return ValueEncoderError{Name: "SymbolEncodeValue", Types: []reflect.Type{tSymbol}, Received: reflect.ValueOf(val)}
	}

	return vw.WriteSymbol(string(symbol))
}

// symbolEncodeValue is the ValueEncoderFunc for the Symbol type.
func symbolEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	return symbolEncodeValueRF(EncodeContext{}, vw, val.Interface())
}

func binaryEncodeValueRF(_ EncodeContext, vw ValueWriter, val any) error {
	b, ok := val.(Binary)
	if !ok {
		return ValueEncoderError{Name: "BinaryEncodeValue", Types: []reflect.Type{tBinary}, Received: reflect.ValueOf(val)}
	}

	return vw.WriteBinaryWithSubtype(b.Data, b.Subtype)
}

// binaryEncodeValue is the ValueEncoderFunc for Binary.
func binaryEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	return binaryEncodeValueRF(EncodeContext{}, vw, val.Interface())
}

func vectorEncodeValueRF(_ EncodeContext, vw ValueWriter, val any) error {
	v, ok := val.(Vector)
	if !ok {
		return ValueEncoderError{Name: "VectorEncodeValue", Types: []reflect.Type{tVector}, Received: reflect.ValueOf(val)}
	}

	b := v.Binary()
	return vw.WriteBinaryWithSubtype(b.Data, b.Subtype)
}

// vectorEncodeValue is the ValueEncoderFunc for Vector.
func vectorEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	return vectorEncodeValueRF(EncodeContext{}, vw, val.Interface())
}

func undefinedEncodeValueRF(_ EncodeContext, vw ValueWriter, val any) error {
	if _, ok := val.(Undefined); !ok {
		return ValueEncoderError{Name: "UndefinedEncodeValue", Types: []reflect.Type{tUndefined}, Received: reflect.ValueOf(val)}
	}

	return vw.WriteUndefined()
}

// undefinedEncodeValue is the ValueEncoderFunc for Undefined.
func undefinedEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	return undefinedEncodeValueRF(EncodeContext{}, vw, val.Interface())
}

func dateTimeEncodeValueRF(_ EncodeContext, vw ValueWriter, val any) error {
	dateTime, ok := val.(DateTime)
	if !ok {
		return ValueEncoderError{Name: "DateTimeEncodeValue", Types: []reflect.Type{tDateTime}, Received: reflect.ValueOf(val)}
	}

	return vw.WriteDateTime(int64(dateTime))
}

// dateTimeEncodeValue is the ValueEncoderFunc for DateTime.
func dateTimeEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	return dateTimeEncodeValueRF(EncodeContext{}, vw, val.Interface())
}

func nullEncodeValueRF(_ EncodeContext, vw ValueWriter, val any) error {
	if _, ok := val.(Null); !ok {
		return ValueEncoderError{Name: "NullEncodeValue", Types: []reflect.Type{tNull}, Received: reflect.ValueOf(val)}
	}

	return vw.WriteNull()
}

// nullEncodeValue is the ValueEncoderFunc for Null.
func nullEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	return nullEncodeValueRF(EncodeContext{}, vw, val.Interface())
}

// regexEncodeValue is the ValueEncoderFunc for Regex.
func regexEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tRegex {
		return ValueEncoderError{Name: "RegexEncodeValue", Types: []reflect.Type{tRegex}, Received: val}
	}

	regex := val.Interface().(Regex)

	return vw.WriteRegex(regex.Pattern, regex.Options)
}

func regexEncodeValueX(_ EncodeContext, vw ValueWriter, val any) error {
	regex, ok := val.(Regex)
	if !ok {
		return ValueEncoderError{Name: "RegexEncodeValue", Types: []reflect.Type{tRegex}, Received: reflect.ValueOf(val)}
	}

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

func dbPointerEncodeValueX(_ EncodeContext, vw ValueWriter, val any) error {
	dbp, ok := val.(DBPointer)
	if !ok {
		return ValueEncoderError{Name: "DBPointerEncodeValue", Types: []reflect.Type{tDBPointer}, Received: reflect.ValueOf(val)}
	}

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

func timestampEncodeValueX(_ EncodeContext, vw ValueWriter, val any) error {
	ts, ok := val.(Timestamp)
	if !ok {
		return ValueEncoderError{Name: "TimestampEncodeValue", Types: []reflect.Type{tTimestamp}, Received: reflect.ValueOf(val)}
	}

	return vw.WriteTimestamp(ts.T, ts.I)
}

// minKeyEncodeValue is the ValueEncoderFunc for MinKey.
func minKeyEncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tMinKey {
		return ValueEncoderError{Name: "MinKeyEncodeValue", Types: []reflect.Type{tMinKey}, Received: val}
	}

	return vw.WriteMinKey()
}

func minKeyEncodeValueX(_ EncodeContext, vw ValueWriter, val any) error {
	if _, ok := val.(MinKey); !ok {
		return ValueEncoderError{Name: "MinKeyEncodeValue", Types: []reflect.Type{tMinKey}, Received: reflect.ValueOf(val)}
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

func maxKeyEncodeValueX(_ EncodeContext, vw ValueWriter, val any) error {
	if _, ok := val.(MaxKey); !ok {
		return ValueEncoderError{Name: "MaxKeyEncodeValue", Types: []reflect.Type{tMaxKey}, Received: reflect.ValueOf(val)}
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

func coreDocumentEncodeValueX(_ EncodeContext, vw ValueWriter, val any) error {
	cdoc, ok := val.(bsoncore.Document)
	if !ok {
		return ValueEncoderError{Name: "CoreDocumentEncodeValue", Types: []reflect.Type{tCoreDocument}, Received: reflect.ValueOf(val)}
	}

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

func codeWithScopeEncodeValueX(ec EncodeContext, vw ValueWriter, val any) error {
	cws, ok := val.(CodeWithScope)
	if !ok {
		return ValueEncoderError{Name: "CodeWithScopeEncodeValue", Types: []reflect.Type{tCodeWithScope}, Received: reflect.ValueOf(val)}
	}

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

func byteSliceEncodeValueRF(encodeNilAsEmpty bool) reflectFreeValueEncoderFunc {
	return reflectFreeValueEncoderFunc(func(ec EncodeContext, vw ValueWriter, val any) error {
		byteSlice, ok := val.([]byte)
		if !ok {
			return ValueEncoderError{
				Name:     "ByteSliceEncodeValue",
				Types:    []reflect.Type{tByteSlice},
				Received: reflect.ValueOf(val),
			}
		}

		if byteSlice == nil && !encodeNilAsEmpty && !ec.nilByteSliceAsEmpty {
			return vw.WriteNull()
		}

		return vw.WriteBinary(byteSlice)
	})
}

func byteSliceEncodeValue(encodeNilAsEmpty bool) defaultValueEncoderFunc {
	return defaultValueEncoderFunc(func(ec EncodeContext, vw ValueWriter, val reflect.Value) error {
		return byteSliceEncodeValueRF(encodeNilAsEmpty)(ec, vw, val.Interface())
	})
}

func timeEncodeValueRF(ec EncodeContext, vw ValueWriter, val any) error {
	tt, ok := val.(time.Time)
	if !ok {
		return ValueEncoderError{Name: "TimeEncodeValue", Types: []reflect.Type{tTime}, Received: reflect.ValueOf(val)}
	}

	dt := NewDateTimeFromTime(tt)
	return vw.WriteDateTime(int64(dt))
}

func timeEncodeValue(ec EncodeContext, vw ValueWriter, val reflect.Value) error {
	return timeEncodeValueRF(ec, vw, val.Interface())
}

func coreArrayEncodeValueRF(ec EncodeContext, vw ValueWriter, val any) error {
	arr, ok := val.(bsoncore.Array)
	if !ok {
		return ValueEncoderError{Name: "CoreArrayEncodeValue", Types: []reflect.Type{tCoreArray}, Received: reflect.ValueOf(val)}
	}

	return copyArrayFromBytes(vw, arr)
}

func coreArrayEncodeValue(ec EncodeContext, vw ValueWriter, val reflect.Value) error {
	return coreArrayEncodeValueRF(ec, vw, val.Interface())
}

func emptyInterfaceValueRF(ec EncodeContext, vw ValueWriter, val any) error {
	if val == nil {
		return vw.WriteNull()
	}
	encoder, err := ec.LookupEncoder(reflect.TypeOf(val))
	if err != nil {
		return err
	}

	return encoder.EncodeValue(ec, vw, reflect.ValueOf(val))
}

func emptyInterfaceValue(ec EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tEmpty {
		return ValueEncoderError{Name: "EmptyInterfaceEncodeValue", Types: []reflect.Type{tEmpty}, Received: val}
	}

	if val.IsNil() {
		return vw.WriteNull()
	}

	encoder, err := ec.LookupEncoder(val.Elem().Type())
	if err != nil {
		return err
	}

	return encoder.EncodeValue(ec, vw, val.Elem())
}
