// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"reflect"
	"strconv"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

var (
	defaultValueDecoders DefaultValueDecoders
	errCannotTruncate    = errors.New("float64 can only be truncated to a lower precision type when truncation is enabled")
)

type decodeBinaryError struct {
	subtype  byte
	typeName string
}

func (d decodeBinaryError) Error() string {
	return fmt.Sprintf("only binary values with subtype 0x00 or 0x02 can be decoded into %s, but got subtype %v", d.typeName, d.subtype)
}

func newDefaultStructCodec() *StructCodec {
	codec, err := NewStructCodec(DefaultStructTagParser)
	if err != nil {
		// This function is called from the codec registration path, so errors can't be propagated. If there's an error
		// constructing the StructCodec, we panic to avoid losing it.
		panic(fmt.Errorf("error creating default StructCodec: %w", err))
	}
	return codec
}

// DefaultValueDecoders is a namespace type for the default ValueDecoders used
// when creating a registry.
//
// Deprecated: Use [go.mongodb.org/mongo-driver/bson.NewRegistry] to get a registry with all default
// value decoders registered.
type DefaultValueDecoders struct{}

// RegisterDefaultDecoders will register the decoder methods attached to DefaultValueDecoders with
// the provided RegistryBuilder.
//
// There is no support for decoding map[string]interface{} because there is no decoder for
// interface{}, so users must either register this decoder themselves or use the
// EmptyInterfaceDecoder available in the bson package.
//
// Deprecated: Use [go.mongodb.org/mongo-driver/bson.NewRegistry] to get a registry with all default
// value decoders registered.
func (dvd DefaultValueDecoders) RegisterDefaultDecoders(rb *RegistryBuilder) {
	if rb == nil {
		panic(errors.New("argument to RegisterDefaultDecoders must not be nil"))
	}

	intDecoder := decodeAdapter{dvd.IntDecodeValue, dvd.intDecodeType}
	floatDecoder := decodeAdapter{dvd.FloatDecodeValue, dvd.floatDecodeType}

	rb.
		RegisterTypeDecoder(tD, ValueDecoderFunc(dvd.DDecodeValue)).
		RegisterTypeDecoder(tBinary, decodeAdapter{dvd.BinaryDecodeValue, dvd.binaryDecodeType}).
		RegisterTypeDecoder(tUndefined, decodeAdapter{dvd.UndefinedDecodeValue, dvd.undefinedDecodeType}).
		RegisterTypeDecoder(tDateTime, decodeAdapter{dvd.DateTimeDecodeValue, dvd.dateTimeDecodeType}).
		RegisterTypeDecoder(tNull, decodeAdapter{dvd.NullDecodeValue, dvd.nullDecodeType}).
		RegisterTypeDecoder(tRegex, decodeAdapter{dvd.RegexDecodeValue, dvd.regexDecodeType}).
		RegisterTypeDecoder(tDBPointer, decodeAdapter{dvd.DBPointerDecodeValue, dvd.dBPointerDecodeType}).
		RegisterTypeDecoder(tTimestamp, decodeAdapter{dvd.TimestampDecodeValue, dvd.timestampDecodeType}).
		RegisterTypeDecoder(tMinKey, decodeAdapter{dvd.MinKeyDecodeValue, dvd.minKeyDecodeType}).
		RegisterTypeDecoder(tMaxKey, decodeAdapter{dvd.MaxKeyDecodeValue, dvd.maxKeyDecodeType}).
		RegisterTypeDecoder(tJavaScript, decodeAdapter{dvd.JavaScriptDecodeValue, dvd.javaScriptDecodeType}).
		RegisterTypeDecoder(tSymbol, decodeAdapter{dvd.SymbolDecodeValue, dvd.symbolDecodeType}).
		RegisterTypeDecoder(tByteSlice, defaultByteSliceCodec).
		RegisterTypeDecoder(tTime, defaultTimeCodec).
		RegisterTypeDecoder(tEmpty, defaultEmptyInterfaceCodec).
		RegisterTypeDecoder(tCoreArray, defaultArrayCodec).
		RegisterTypeDecoder(tOID, decodeAdapter{dvd.ObjectIDDecodeValue, dvd.objectIDDecodeType}).
		RegisterTypeDecoder(tDecimal, decodeAdapter{dvd.Decimal128DecodeValue, dvd.decimal128DecodeType}).
		RegisterTypeDecoder(tJSONNumber, decodeAdapter{dvd.JSONNumberDecodeValue, dvd.jsonNumberDecodeType}).
		RegisterTypeDecoder(tURL, decodeAdapter{dvd.URLDecodeValue, dvd.urlDecodeType}).
		RegisterTypeDecoder(tCoreDocument, ValueDecoderFunc(dvd.CoreDocumentDecodeValue)).
		RegisterTypeDecoder(tCodeWithScope, decodeAdapter{dvd.CodeWithScopeDecodeValue, dvd.codeWithScopeDecodeType}).
		RegisterDefaultDecoder(reflect.Bool, decodeAdapter{dvd.BooleanDecodeValue, dvd.booleanDecodeType}).
		RegisterDefaultDecoder(reflect.Int, intDecoder).
		RegisterDefaultDecoder(reflect.Int8, intDecoder).
		RegisterDefaultDecoder(reflect.Int16, intDecoder).
		RegisterDefaultDecoder(reflect.Int32, intDecoder).
		RegisterDefaultDecoder(reflect.Int64, intDecoder).
		RegisterDefaultDecoder(reflect.Uint, defaultUIntCodec).
		RegisterDefaultDecoder(reflect.Uint8, defaultUIntCodec).
		RegisterDefaultDecoder(reflect.Uint16, defaultUIntCodec).
		RegisterDefaultDecoder(reflect.Uint32, defaultUIntCodec).
		RegisterDefaultDecoder(reflect.Uint64, defaultUIntCodec).
		RegisterDefaultDecoder(reflect.Float32, floatDecoder).
		RegisterDefaultDecoder(reflect.Float64, floatDecoder).
		RegisterDefaultDecoder(reflect.Array, ValueDecoderFunc(dvd.ArrayDecodeValue)).
		RegisterDefaultDecoder(reflect.Map, defaultMapCodec).
		RegisterDefaultDecoder(reflect.Slice, defaultSliceCodec).
		RegisterDefaultDecoder(reflect.String, defaultStringCodec).
		RegisterDefaultDecoder(reflect.Struct, newDefaultStructCodec()).
		RegisterDefaultDecoder(reflect.Ptr, NewPointerCodec()).
		RegisterTypeMapEntry(TypeDouble, tFloat64).
		RegisterTypeMapEntry(TypeString, tString).
		RegisterTypeMapEntry(TypeArray, tA).
		RegisterTypeMapEntry(TypeBinary, tBinary).
		RegisterTypeMapEntry(TypeUndefined, tUndefined).
		RegisterTypeMapEntry(TypeObjectID, tOID).
		RegisterTypeMapEntry(TypeBoolean, tBool).
		RegisterTypeMapEntry(TypeDateTime, tDateTime).
		RegisterTypeMapEntry(TypeRegex, tRegex).
		RegisterTypeMapEntry(TypeDBPointer, tDBPointer).
		RegisterTypeMapEntry(TypeJavaScript, tJavaScript).
		RegisterTypeMapEntry(TypeSymbol, tSymbol).
		RegisterTypeMapEntry(TypeCodeWithScope, tCodeWithScope).
		RegisterTypeMapEntry(TypeInt32, tInt32).
		RegisterTypeMapEntry(TypeInt64, tInt64).
		RegisterTypeMapEntry(TypeTimestamp, tTimestamp).
		RegisterTypeMapEntry(TypeDecimal128, tDecimal).
		RegisterTypeMapEntry(TypeMinKey, tMinKey).
		RegisterTypeMapEntry(TypeMaxKey, tMaxKey).
		RegisterTypeMapEntry(Type(0), tD).
		RegisterTypeMapEntry(TypeEmbeddedDocument, tD).
		RegisterHookDecoder(tValueUnmarshaler, ValueDecoderFunc(dvd.ValueUnmarshalerDecodeValue)).
		RegisterHookDecoder(tUnmarshaler, ValueDecoderFunc(dvd.UnmarshalerDecodeValue))
}

// DDecodeValue is the ValueDecoderFunc for D instances.
//
// Deprecated: Use [go.mongodb.org/mongo-driver/bson.NewRegistry] to get a registry with all default
// value decoders registered.
func (dvd DefaultValueDecoders) DDecodeValue(dc DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.IsValid() || !val.CanSet() || val.Type() != tD {
		return ValueDecoderError{Name: "DDecodeValue", Kinds: []reflect.Kind{reflect.Slice}, Received: val}
	}

	switch vrType := vr.Type(); vrType {
	case Type(0), TypeEmbeddedDocument:
		dc.Ancestor = tD
	case TypeNull:
		val.Set(reflect.Zero(val.Type()))
		return vr.ReadNull()
	default:
		return fmt.Errorf("cannot decode %v into a D", vrType)
	}

	dr, err := vr.ReadDocument()
	if err != nil {
		return err
	}

	decoder, err := dc.LookupDecoder(tEmpty)
	if err != nil {
		return err
	}

	// Use the elements in the provided value if it's non nil. Otherwise, allocate a new D instance.
	var elems D
	if !val.IsNil() {
		val.SetLen(0)
		elems = val.Interface().(D)
	} else {
		elems = make(D, 0)
	}

	for {
		key, elemVr, err := dr.ReadElement()
		if errors.Is(err, ErrEOD) {
			break
		} else if err != nil {
			return err
		}

		// Pass false for convert because we don't need to call reflect.Value.Convert for tEmpty.
		var v interface{}
		err = decoder.DecodeValue(dc, elemVr, reflect.ValueOf(&v).Elem())
		if err != nil {
			return err
		}

		elems = append(elems, E{Key: key, Value: v})
	}

	val.Set(reflect.ValueOf(elems))
	return nil
}

func (dvd DefaultValueDecoders) booleanDecodeType(_ DecodeContext, vr ValueReader, t reflect.Type) (reflect.Value, error) {
	if t.Kind() != reflect.Bool {
		return emptyValue, ValueDecoderError{
			Name:     "BooleanDecodeValue",
			Kinds:    []reflect.Kind{reflect.Bool},
			Received: reflect.Zero(t),
		}
	}

	var b bool
	var err error
	switch vrType := vr.Type(); vrType {
	case TypeInt32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return emptyValue, err
		}
		b = (i32 != 0)
	case TypeInt64:
		i64, err := vr.ReadInt64()
		if err != nil {
			return emptyValue, err
		}
		b = (i64 != 0)
	case TypeDouble:
		f64, err := vr.ReadDouble()
		if err != nil {
			return emptyValue, err
		}
		b = (f64 != 0)
	case TypeBoolean:
		b, err = vr.ReadBoolean()
	case TypeNull:
		err = vr.ReadNull()
	case TypeUndefined:
		err = vr.ReadUndefined()
	default:
		return emptyValue, fmt.Errorf("cannot decode %v into a boolean", vrType)
	}
	if err != nil {
		return emptyValue, err
	}

	return reflect.ValueOf(b), nil
}

// BooleanDecodeValue is the ValueDecoderFunc for bool types.
//
// Deprecated: Use [go.mongodb.org/mongo-driver/bson.NewRegistry] to get a registry with all default
// value decoders registered.
func (dvd DefaultValueDecoders) BooleanDecodeValue(dctx DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.IsValid() || !val.CanSet() || val.Kind() != reflect.Bool {
		return ValueDecoderError{Name: "BooleanDecodeValue", Kinds: []reflect.Kind{reflect.Bool}, Received: val}
	}

	elem, err := dvd.booleanDecodeType(dctx, vr, val.Type())
	if err != nil {
		return err
	}

	val.SetBool(elem.Bool())
	return nil
}

func (DefaultValueDecoders) intDecodeType(dc DecodeContext, vr ValueReader, t reflect.Type) (reflect.Value, error) {
	var i64 int64
	var err error
	switch vrType := vr.Type(); vrType {
	case TypeInt32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return emptyValue, err
		}
		i64 = int64(i32)
	case TypeInt64:
		i64, err = vr.ReadInt64()
		if err != nil {
			return emptyValue, err
		}
	case TypeDouble:
		f64, err := vr.ReadDouble()
		if err != nil {
			return emptyValue, err
		}
		if !dc.Truncate && math.Floor(f64) != f64 {
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
		if err = vr.ReadNull(); err != nil {
			return emptyValue, err
		}
	case TypeUndefined:
		if err = vr.ReadUndefined(); err != nil {
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
	default:
		return emptyValue, ValueDecoderError{
			Name:     "IntDecodeValue",
			Kinds:    []reflect.Kind{reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int},
			Received: reflect.Zero(t),
		}
	}
}

// IntDecodeValue is the ValueDecoderFunc for int types.
//
// Deprecated: Use [go.mongodb.org/mongo-driver/bson.NewRegistry] to get a registry with all default
// value decoders registered.
func (dvd DefaultValueDecoders) IntDecodeValue(dc DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() {
		return ValueDecoderError{
			Name:     "IntDecodeValue",
			Kinds:    []reflect.Kind{reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int},
			Received: val,
		}
	}

	elem, err := dvd.intDecodeType(dc, vr, val.Type())
	if err != nil {
		return err
	}

	val.SetInt(elem.Int())
	return nil
}

func (dvd DefaultValueDecoders) floatDecodeType(dc DecodeContext, vr ValueReader, t reflect.Type) (reflect.Value, error) {
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
		if !dc.Truncate && float64(float32(f)) != f {
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

// FloatDecodeValue is the ValueDecoderFunc for float types.
//
// Deprecated: Use [go.mongodb.org/mongo-driver/bson.NewRegistry] to get a registry with all default
// value decoders registered.
func (dvd DefaultValueDecoders) FloatDecodeValue(ec DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() {
		return ValueDecoderError{
			Name:     "FloatDecodeValue",
			Kinds:    []reflect.Kind{reflect.Float32, reflect.Float64},
			Received: val,
		}
	}

	elem, err := dvd.floatDecodeType(ec, vr, val.Type())
	if err != nil {
		return err
	}

	val.SetFloat(elem.Float())
	return nil
}

func (DefaultValueDecoders) javaScriptDecodeType(_ DecodeContext, vr ValueReader, t reflect.Type) (reflect.Value, error) {
	if t != tJavaScript {
		return emptyValue, ValueDecoderError{
			Name:     "JavaScriptDecodeValue",
			Types:    []reflect.Type{tJavaScript},
			Received: reflect.Zero(t),
		}
	}

	var js string
	var err error
	switch vrType := vr.Type(); vrType {
	case TypeJavaScript:
		js, err = vr.ReadJavascript()
	case TypeNull:
		err = vr.ReadNull()
	case TypeUndefined:
		err = vr.ReadUndefined()
	default:
		return emptyValue, fmt.Errorf("cannot decode %v into a JavaScript", vrType)
	}
	if err != nil {
		return emptyValue, err
	}

	return reflect.ValueOf(JavaScript(js)), nil
}

// JavaScriptDecodeValue is the ValueDecoderFunc for the JavaScript type.
//
// Deprecated: Use [go.mongodb.org/mongo-driver/bson.NewRegistry] to get a registry with all default
// value decoders registered.
func (dvd DefaultValueDecoders) JavaScriptDecodeValue(dctx DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tJavaScript {
		return ValueDecoderError{Name: "JavaScriptDecodeValue", Types: []reflect.Type{tJavaScript}, Received: val}
	}

	elem, err := dvd.javaScriptDecodeType(dctx, vr, tJavaScript)
	if err != nil {
		return err
	}

	val.SetString(elem.String())
	return nil
}

func (DefaultValueDecoders) symbolDecodeType(_ DecodeContext, vr ValueReader, t reflect.Type) (reflect.Value, error) {
	if t != tSymbol {
		return emptyValue, ValueDecoderError{
			Name:     "SymbolDecodeValue",
			Types:    []reflect.Type{tSymbol},
			Received: reflect.Zero(t),
		}
	}

	var symbol string
	var err error
	switch vrType := vr.Type(); vrType {
	case TypeString:
		symbol, err = vr.ReadString()
	case TypeSymbol:
		symbol, err = vr.ReadSymbol()
	case TypeBinary:
		data, subtype, err := vr.ReadBinary()
		if err != nil {
			return emptyValue, err
		}

		if subtype != TypeBinaryGeneric && subtype != TypeBinaryBinaryOld {
			return emptyValue, decodeBinaryError{subtype: subtype, typeName: "Symbol"}
		}
		symbol = string(data)
	case TypeNull:
		err = vr.ReadNull()
	case TypeUndefined:
		err = vr.ReadUndefined()
	default:
		return emptyValue, fmt.Errorf("cannot decode %v into a Symbol", vrType)
	}
	if err != nil {
		return emptyValue, err
	}

	return reflect.ValueOf(Symbol(symbol)), nil
}

// SymbolDecodeValue is the ValueDecoderFunc for the Symbol type.
//
// Deprecated: Use [go.mongodb.org/mongo-driver/bson.NewRegistry] to get a registry with all default
// value decoders registered.
func (dvd DefaultValueDecoders) SymbolDecodeValue(dctx DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tSymbol {
		return ValueDecoderError{Name: "SymbolDecodeValue", Types: []reflect.Type{tSymbol}, Received: val}
	}

	elem, err := dvd.symbolDecodeType(dctx, vr, tSymbol)
	if err != nil {
		return err
	}

	val.SetString(elem.String())
	return nil
}

func (DefaultValueDecoders) binaryDecodeType(_ DecodeContext, vr ValueReader, t reflect.Type) (reflect.Value, error) {
	if t != tBinary {
		return emptyValue, ValueDecoderError{
			Name:     "BinaryDecodeValue",
			Types:    []reflect.Type{tBinary},
			Received: reflect.Zero(t),
		}
	}

	var data []byte
	var subtype byte
	var err error
	switch vrType := vr.Type(); vrType {
	case TypeBinary:
		data, subtype, err = vr.ReadBinary()
	case TypeNull:
		err = vr.ReadNull()
	case TypeUndefined:
		err = vr.ReadUndefined()
	default:
		return emptyValue, fmt.Errorf("cannot decode %v into a Binary", vrType)
	}
	if err != nil {
		return emptyValue, err
	}

	return reflect.ValueOf(Binary{Subtype: subtype, Data: data}), nil
}

// BinaryDecodeValue is the ValueDecoderFunc for Binary.
//
// Deprecated: Use [go.mongodb.org/mongo-driver/bson.NewRegistry] to get a registry with all default
// value decoders registered.
func (dvd DefaultValueDecoders) BinaryDecodeValue(dc DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tBinary {
		return ValueDecoderError{Name: "BinaryDecodeValue", Types: []reflect.Type{tBinary}, Received: val}
	}

	elem, err := dvd.binaryDecodeType(dc, vr, tBinary)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

func (DefaultValueDecoders) undefinedDecodeType(_ DecodeContext, vr ValueReader, t reflect.Type) (reflect.Value, error) {
	if t != tUndefined {
		return emptyValue, ValueDecoderError{
			Name:     "UndefinedDecodeValue",
			Types:    []reflect.Type{tUndefined},
			Received: reflect.Zero(t),
		}
	}

	var err error
	switch vrType := vr.Type(); vrType {
	case TypeUndefined:
		err = vr.ReadUndefined()
	case TypeNull:
		err = vr.ReadNull()
	default:
		return emptyValue, fmt.Errorf("cannot decode %v into an Undefined", vr.Type())
	}
	if err != nil {
		return emptyValue, err
	}

	return reflect.ValueOf(Undefined{}), nil
}

// UndefinedDecodeValue is the ValueDecoderFunc for Undefined.
//
// Deprecated: Use [go.mongodb.org/mongo-driver/bson.NewRegistry] to get a registry with all default
// value decoders registered.
func (dvd DefaultValueDecoders) UndefinedDecodeValue(dc DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tUndefined {
		return ValueDecoderError{Name: "UndefinedDecodeValue", Types: []reflect.Type{tUndefined}, Received: val}
	}

	elem, err := dvd.undefinedDecodeType(dc, vr, tUndefined)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

// Accept both 12-byte string and pretty-printed 24-byte hex string formats.
func (dvd DefaultValueDecoders) objectIDDecodeType(_ DecodeContext, vr ValueReader, t reflect.Type) (reflect.Value, error) {
	if t != tOID {
		return emptyValue, ValueDecoderError{
			Name:     "ObjectIDDecodeValue",
			Types:    []reflect.Type{tOID},
			Received: reflect.Zero(t),
		}
	}

	var oid ObjectID
	var err error
	switch vrType := vr.Type(); vrType {
	case TypeObjectID:
		oid, err = vr.ReadObjectID()
		if err != nil {
			return emptyValue, err
		}
	case TypeString:
		str, err := vr.ReadString()
		if err != nil {
			return emptyValue, err
		}
		if oid, err = ObjectIDFromHex(str); err == nil {
			break
		}
		if len(str) != 12 {
			return emptyValue, fmt.Errorf("an ObjectID string must be exactly 12 bytes long (got %v)", len(str))
		}
		byteArr := []byte(str)
		copy(oid[:], byteArr)
	case TypeNull:
		if err = vr.ReadNull(); err != nil {
			return emptyValue, err
		}
	case TypeUndefined:
		if err = vr.ReadUndefined(); err != nil {
			return emptyValue, err
		}
	default:
		return emptyValue, fmt.Errorf("cannot decode %v into an ObjectID", vrType)
	}

	return reflect.ValueOf(oid), nil
}

// ObjectIDDecodeValue is the ValueDecoderFunc for ObjectID.
//
// Deprecated: Use [go.mongodb.org/mongo-driver/bson.NewRegistry] to get a registry with all default
// value decoders registered.
func (dvd DefaultValueDecoders) ObjectIDDecodeValue(dc DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tOID {
		return ValueDecoderError{Name: "ObjectIDDecodeValue", Types: []reflect.Type{tOID}, Received: val}
	}

	elem, err := dvd.objectIDDecodeType(dc, vr, tOID)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

func (DefaultValueDecoders) dateTimeDecodeType(_ DecodeContext, vr ValueReader, t reflect.Type) (reflect.Value, error) {
	if t != tDateTime {
		return emptyValue, ValueDecoderError{
			Name:     "DateTimeDecodeValue",
			Types:    []reflect.Type{tDateTime},
			Received: reflect.Zero(t),
		}
	}

	var dt int64
	var err error
	switch vrType := vr.Type(); vrType {
	case TypeDateTime:
		dt, err = vr.ReadDateTime()
	case TypeNull:
		err = vr.ReadNull()
	case TypeUndefined:
		err = vr.ReadUndefined()
	default:
		return emptyValue, fmt.Errorf("cannot decode %v into a DateTime", vrType)
	}
	if err != nil {
		return emptyValue, err
	}

	return reflect.ValueOf(DateTime(dt)), nil
}

// DateTimeDecodeValue is the ValueDecoderFunc for DateTime.
//
// Deprecated: Use [go.mongodb.org/mongo-driver/bson.NewRegistry] to get a registry with all default
// value decoders registered.
func (dvd DefaultValueDecoders) DateTimeDecodeValue(dc DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tDateTime {
		return ValueDecoderError{Name: "DateTimeDecodeValue", Types: []reflect.Type{tDateTime}, Received: val}
	}

	elem, err := dvd.dateTimeDecodeType(dc, vr, tDateTime)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

func (DefaultValueDecoders) nullDecodeType(_ DecodeContext, vr ValueReader, t reflect.Type) (reflect.Value, error) {
	if t != tNull {
		return emptyValue, ValueDecoderError{
			Name:     "NullDecodeValue",
			Types:    []reflect.Type{tNull},
			Received: reflect.Zero(t),
		}
	}

	var err error
	switch vrType := vr.Type(); vrType {
	case TypeUndefined:
		err = vr.ReadUndefined()
	case TypeNull:
		err = vr.ReadNull()
	default:
		return emptyValue, fmt.Errorf("cannot decode %v into a Null", vr.Type())
	}
	if err != nil {
		return emptyValue, err
	}

	return reflect.ValueOf(Null{}), nil
}

// NullDecodeValue is the ValueDecoderFunc for Null.
//
// Deprecated: Use [go.mongodb.org/mongo-driver/bson.NewRegistry] to get a registry with all default
// value decoders registered.
func (dvd DefaultValueDecoders) NullDecodeValue(dc DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tNull {
		return ValueDecoderError{Name: "NullDecodeValue", Types: []reflect.Type{tNull}, Received: val}
	}

	elem, err := dvd.nullDecodeType(dc, vr, tNull)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

func (DefaultValueDecoders) regexDecodeType(_ DecodeContext, vr ValueReader, t reflect.Type) (reflect.Value, error) {
	if t != tRegex {
		return emptyValue, ValueDecoderError{
			Name:     "RegexDecodeValue",
			Types:    []reflect.Type{tRegex},
			Received: reflect.Zero(t),
		}
	}

	var pattern, options string
	var err error
	switch vrType := vr.Type(); vrType {
	case TypeRegex:
		pattern, options, err = vr.ReadRegex()
	case TypeNull:
		err = vr.ReadNull()
	case TypeUndefined:
		err = vr.ReadUndefined()
	default:
		return emptyValue, fmt.Errorf("cannot decode %v into a Regex", vrType)
	}
	if err != nil {
		return emptyValue, err
	}

	return reflect.ValueOf(Regex{Pattern: pattern, Options: options}), nil
}

// RegexDecodeValue is the ValueDecoderFunc for Regex.
//
// Deprecated: Use [go.mongodb.org/mongo-driver/bson.NewRegistry] to get a registry with all default
// value decoders registered.
func (dvd DefaultValueDecoders) RegexDecodeValue(dc DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tRegex {
		return ValueDecoderError{Name: "RegexDecodeValue", Types: []reflect.Type{tRegex}, Received: val}
	}

	elem, err := dvd.regexDecodeType(dc, vr, tRegex)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

func (DefaultValueDecoders) dBPointerDecodeType(_ DecodeContext, vr ValueReader, t reflect.Type) (reflect.Value, error) {
	if t != tDBPointer {
		return emptyValue, ValueDecoderError{
			Name:     "DBPointerDecodeValue",
			Types:    []reflect.Type{tDBPointer},
			Received: reflect.Zero(t),
		}
	}

	var ns string
	var pointer ObjectID
	var err error
	switch vrType := vr.Type(); vrType {
	case TypeDBPointer:
		ns, pointer, err = vr.ReadDBPointer()
	case TypeNull:
		err = vr.ReadNull()
	case TypeUndefined:
		err = vr.ReadUndefined()
	default:
		return emptyValue, fmt.Errorf("cannot decode %v into a DBPointer", vrType)
	}
	if err != nil {
		return emptyValue, err
	}

	return reflect.ValueOf(DBPointer{DB: ns, Pointer: pointer}), nil
}

// DBPointerDecodeValue is the ValueDecoderFunc for DBPointer.
//
// Deprecated: Use [go.mongodb.org/mongo-driver/bson.NewRegistry] to get a registry with all default
// value decoders registered.
func (dvd DefaultValueDecoders) DBPointerDecodeValue(dc DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tDBPointer {
		return ValueDecoderError{Name: "DBPointerDecodeValue", Types: []reflect.Type{tDBPointer}, Received: val}
	}

	elem, err := dvd.dBPointerDecodeType(dc, vr, tDBPointer)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

func (DefaultValueDecoders) timestampDecodeType(_ DecodeContext, vr ValueReader, reflectType reflect.Type) (reflect.Value, error) {
	if reflectType != tTimestamp {
		return emptyValue, ValueDecoderError{
			Name:     "TimestampDecodeValue",
			Types:    []reflect.Type{tTimestamp},
			Received: reflect.Zero(reflectType),
		}
	}

	var t, incr uint32
	var err error
	switch vrType := vr.Type(); vrType {
	case TypeTimestamp:
		t, incr, err = vr.ReadTimestamp()
	case TypeNull:
		err = vr.ReadNull()
	case TypeUndefined:
		err = vr.ReadUndefined()
	default:
		return emptyValue, fmt.Errorf("cannot decode %v into a Timestamp", vrType)
	}
	if err != nil {
		return emptyValue, err
	}

	return reflect.ValueOf(Timestamp{T: t, I: incr}), nil
}

// TimestampDecodeValue is the ValueDecoderFunc for Timestamp.
//
// Deprecated: Use [go.mongodb.org/mongo-driver/bson.NewRegistry] to get a registry with all default
// value decoders registered.
func (dvd DefaultValueDecoders) TimestampDecodeValue(dc DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tTimestamp {
		return ValueDecoderError{Name: "TimestampDecodeValue", Types: []reflect.Type{tTimestamp}, Received: val}
	}

	elem, err := dvd.timestampDecodeType(dc, vr, tTimestamp)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

func (DefaultValueDecoders) minKeyDecodeType(_ DecodeContext, vr ValueReader, t reflect.Type) (reflect.Value, error) {
	if t != tMinKey {
		return emptyValue, ValueDecoderError{
			Name:     "MinKeyDecodeValue",
			Types:    []reflect.Type{tMinKey},
			Received: reflect.Zero(t),
		}
	}

	var err error
	switch vrType := vr.Type(); vrType {
	case TypeMinKey:
		err = vr.ReadMinKey()
	case TypeNull:
		err = vr.ReadNull()
	case TypeUndefined:
		err = vr.ReadUndefined()
	default:
		return emptyValue, fmt.Errorf("cannot decode %v into a MinKey", vr.Type())
	}
	if err != nil {
		return emptyValue, err
	}

	return reflect.ValueOf(MinKey{}), nil
}

// MinKeyDecodeValue is the ValueDecoderFunc for MinKey.
//
// Deprecated: Use [go.mongodb.org/mongo-driver/bson.NewRegistry] to get a registry with all default
// value decoders registered.
func (dvd DefaultValueDecoders) MinKeyDecodeValue(dc DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tMinKey {
		return ValueDecoderError{Name: "MinKeyDecodeValue", Types: []reflect.Type{tMinKey}, Received: val}
	}

	elem, err := dvd.minKeyDecodeType(dc, vr, tMinKey)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

func (DefaultValueDecoders) maxKeyDecodeType(_ DecodeContext, vr ValueReader, t reflect.Type) (reflect.Value, error) {
	if t != tMaxKey {
		return emptyValue, ValueDecoderError{
			Name:     "MaxKeyDecodeValue",
			Types:    []reflect.Type{tMaxKey},
			Received: reflect.Zero(t),
		}
	}

	var err error
	switch vrType := vr.Type(); vrType {
	case TypeMaxKey:
		err = vr.ReadMaxKey()
	case TypeNull:
		err = vr.ReadNull()
	case TypeUndefined:
		err = vr.ReadUndefined()
	default:
		return emptyValue, fmt.Errorf("cannot decode %v into a MaxKey", vr.Type())
	}
	if err != nil {
		return emptyValue, err
	}

	return reflect.ValueOf(MaxKey{}), nil
}

// MaxKeyDecodeValue is the ValueDecoderFunc for MaxKey.
//
// Deprecated: Use [go.mongodb.org/mongo-driver/bson.NewRegistry] to get a registry with all default
// value decoders registered.
func (dvd DefaultValueDecoders) MaxKeyDecodeValue(dc DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tMaxKey {
		return ValueDecoderError{Name: "MaxKeyDecodeValue", Types: []reflect.Type{tMaxKey}, Received: val}
	}

	elem, err := dvd.maxKeyDecodeType(dc, vr, tMaxKey)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

func (dvd DefaultValueDecoders) decimal128DecodeType(_ DecodeContext, vr ValueReader, t reflect.Type) (reflect.Value, error) {
	if t != tDecimal {
		return emptyValue, ValueDecoderError{
			Name:     "Decimal128DecodeValue",
			Types:    []reflect.Type{tDecimal},
			Received: reflect.Zero(t),
		}
	}

	var d128 Decimal128
	var err error
	switch vrType := vr.Type(); vrType {
	case TypeDecimal128:
		d128, err = vr.ReadDecimal128()
	case TypeNull:
		err = vr.ReadNull()
	case TypeUndefined:
		err = vr.ReadUndefined()
	default:
		return emptyValue, fmt.Errorf("cannot decode %v into a Decimal128", vr.Type())
	}
	if err != nil {
		return emptyValue, err
	}

	return reflect.ValueOf(d128), nil
}

// Decimal128DecodeValue is the ValueDecoderFunc for Decimal128.
//
// Deprecated: Use [go.mongodb.org/mongo-driver/bson.NewRegistry] to get a registry with all default
// value decoders registered.
func (dvd DefaultValueDecoders) Decimal128DecodeValue(dctx DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tDecimal {
		return ValueDecoderError{Name: "Decimal128DecodeValue", Types: []reflect.Type{tDecimal}, Received: val}
	}

	elem, err := dvd.decimal128DecodeType(dctx, vr, tDecimal)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

func (dvd DefaultValueDecoders) jsonNumberDecodeType(_ DecodeContext, vr ValueReader, t reflect.Type) (reflect.Value, error) {
	if t != tJSONNumber {
		return emptyValue, ValueDecoderError{
			Name:     "JSONNumberDecodeValue",
			Types:    []reflect.Type{tJSONNumber},
			Received: reflect.Zero(t),
		}
	}

	var jsonNum json.Number
	var err error
	switch vrType := vr.Type(); vrType {
	case TypeDouble:
		f64, err := vr.ReadDouble()
		if err != nil {
			return emptyValue, err
		}
		jsonNum = json.Number(strconv.FormatFloat(f64, 'f', -1, 64))
	case TypeInt32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return emptyValue, err
		}
		jsonNum = json.Number(strconv.FormatInt(int64(i32), 10))
	case TypeInt64:
		i64, err := vr.ReadInt64()
		if err != nil {
			return emptyValue, err
		}
		jsonNum = json.Number(strconv.FormatInt(i64, 10))
	case TypeNull:
		err = vr.ReadNull()
	case TypeUndefined:
		err = vr.ReadUndefined()
	default:
		return emptyValue, fmt.Errorf("cannot decode %v into a json.Number", vrType)
	}
	if err != nil {
		return emptyValue, err
	}

	return reflect.ValueOf(jsonNum), nil
}

// JSONNumberDecodeValue is the ValueDecoderFunc for json.Number.
//
// Deprecated: Use [go.mongodb.org/mongo-driver/bson.NewRegistry] to get a registry with all default
// value decoders registered.
func (dvd DefaultValueDecoders) JSONNumberDecodeValue(dc DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tJSONNumber {
		return ValueDecoderError{Name: "JSONNumberDecodeValue", Types: []reflect.Type{tJSONNumber}, Received: val}
	}

	elem, err := dvd.jsonNumberDecodeType(dc, vr, tJSONNumber)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

func (dvd DefaultValueDecoders) urlDecodeType(_ DecodeContext, vr ValueReader, t reflect.Type) (reflect.Value, error) {
	if t != tURL {
		return emptyValue, ValueDecoderError{
			Name:     "URLDecodeValue",
			Types:    []reflect.Type{tURL},
			Received: reflect.Zero(t),
		}
	}

	urlPtr := &url.URL{}
	var err error
	switch vrType := vr.Type(); vrType {
	case TypeString:
		var str string // Declare str here to avoid shadowing err during the ReadString call.
		str, err = vr.ReadString()
		if err != nil {
			return emptyValue, err
		}

		urlPtr, err = url.Parse(str)
	case TypeNull:
		err = vr.ReadNull()
	case TypeUndefined:
		err = vr.ReadUndefined()
	default:
		return emptyValue, fmt.Errorf("cannot decode %v into a *url.URL", vrType)
	}
	if err != nil {
		return emptyValue, err
	}

	return reflect.ValueOf(urlPtr).Elem(), nil
}

// URLDecodeValue is the ValueDecoderFunc for url.URL.
//
// Deprecated: Use [go.mongodb.org/mongo-driver/bson.NewRegistry] to get a registry with all default
// value decoders registered.
func (dvd DefaultValueDecoders) URLDecodeValue(dc DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tURL {
		return ValueDecoderError{Name: "URLDecodeValue", Types: []reflect.Type{tURL}, Received: val}
	}

	elem, err := dvd.urlDecodeType(dc, vr, tURL)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

// ArrayDecodeValue is the ValueDecoderFunc for array types.
//
// Deprecated: Use [go.mongodb.org/mongo-driver/bson.NewRegistry] to get a registry with all default
// value decoders registered.
func (dvd DefaultValueDecoders) ArrayDecodeValue(dc DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.IsValid() || val.Kind() != reflect.Array {
		return ValueDecoderError{Name: "ArrayDecodeValue", Kinds: []reflect.Kind{reflect.Array}, Received: val}
	}

	switch vrType := vr.Type(); vrType {
	case TypeArray:
	case Type(0), TypeEmbeddedDocument:
		if val.Type().Elem() != tE {
			return fmt.Errorf("cannot decode document into %s", val.Type())
		}
	case TypeBinary:
		if val.Type().Elem() != tByte {
			return fmt.Errorf("ArrayDecodeValue can only be used to decode binary into a byte array, got %v", vrType)
		}
		data, subtype, err := vr.ReadBinary()
		if err != nil {
			return err
		}
		if subtype != TypeBinaryGeneric && subtype != TypeBinaryBinaryOld {
			return fmt.Errorf("ArrayDecodeValue can only be used to decode subtype 0x00 or 0x02 for %s, got %v", TypeBinary, subtype)
		}

		if len(data) > val.Len() {
			return fmt.Errorf("more elements returned in array than can fit inside %s", val.Type())
		}

		for idx, elem := range data {
			val.Index(idx).Set(reflect.ValueOf(elem))
		}
		return nil
	case TypeNull:
		val.Set(reflect.Zero(val.Type()))
		return vr.ReadNull()
	case TypeUndefined:
		val.Set(reflect.Zero(val.Type()))
		return vr.ReadUndefined()
	default:
		return fmt.Errorf("cannot decode %v into an array", vrType)
	}

	var elemsFunc func(DecodeContext, ValueReader, reflect.Value) ([]reflect.Value, error)
	switch val.Type().Elem() {
	case tE:
		elemsFunc = dvd.decodeD
	default:
		elemsFunc = dvd.decodeDefault
	}

	elems, err := elemsFunc(dc, vr, val)
	if err != nil {
		return err
	}

	if len(elems) > val.Len() {
		return fmt.Errorf("more elements returned in array than can fit inside %s, got %v elements", val.Type(), len(elems))
	}

	for idx, elem := range elems {
		val.Index(idx).Set(elem)
	}

	return nil
}

// ValueUnmarshalerDecodeValue is the ValueDecoderFunc for ValueUnmarshaler implementations.
//
// Deprecated: Use [go.mongodb.org/mongo-driver/bson.NewRegistry] to get a registry with all default
// value decoders registered.
func (dvd DefaultValueDecoders) ValueUnmarshalerDecodeValue(_ DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.IsValid() || (!val.Type().Implements(tValueUnmarshaler) && !reflect.PtrTo(val.Type()).Implements(tValueUnmarshaler)) {
		return ValueDecoderError{Name: "ValueUnmarshalerDecodeValue", Types: []reflect.Type{tValueUnmarshaler}, Received: val}
	}

	if val.Kind() == reflect.Ptr && val.IsNil() {
		if !val.CanSet() {
			return ValueDecoderError{Name: "ValueUnmarshalerDecodeValue", Types: []reflect.Type{tValueUnmarshaler}, Received: val}
		}
		val.Set(reflect.New(val.Type().Elem()))
	}

	if !val.Type().Implements(tValueUnmarshaler) {
		if !val.CanAddr() {
			return ValueDecoderError{Name: "ValueUnmarshalerDecodeValue", Types: []reflect.Type{tValueUnmarshaler}, Received: val}
		}
		val = val.Addr() // If the type doesn't implement the interface, a pointer to it must.
	}

	t, src, err := CopyValueToBytes(vr)
	if err != nil {
		return err
	}

	m, ok := val.Interface().(ValueUnmarshaler)
	if !ok {
		// NB: this error should be unreachable due to the above checks
		return ValueDecoderError{Name: "ValueUnmarshalerDecodeValue", Types: []reflect.Type{tValueUnmarshaler}, Received: val}
	}
	return m.UnmarshalBSONValue(t, src)
}

// UnmarshalerDecodeValue is the ValueDecoderFunc for Unmarshaler implementations.
//
// Deprecated: Use [go.mongodb.org/mongo-driver/bson.NewRegistry] to get a registry with all default
// value decoders registered.
func (dvd DefaultValueDecoders) UnmarshalerDecodeValue(_ DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.IsValid() || (!val.Type().Implements(tUnmarshaler) && !reflect.PtrTo(val.Type()).Implements(tUnmarshaler)) {
		return ValueDecoderError{Name: "UnmarshalerDecodeValue", Types: []reflect.Type{tUnmarshaler}, Received: val}
	}

	if val.Kind() == reflect.Ptr && val.IsNil() {
		if !val.CanSet() {
			return ValueDecoderError{Name: "UnmarshalerDecodeValue", Types: []reflect.Type{tUnmarshaler}, Received: val}
		}
		val.Set(reflect.New(val.Type().Elem()))
	}

	_, src, err := CopyValueToBytes(vr)
	if err != nil {
		return err
	}

	// If the target Go value is a pointer and the BSON field value is empty, set the value to the
	// zero value of the pointer (nil) and don't call UnmarshalBSON. UnmarshalBSON has no way to
	// change the pointer value from within the function (only the value at the pointer address),
	// so it can't set the pointer to "nil" itself. Since the most common Go value for an empty BSON
	// field value is "nil", we set "nil" here and don't call UnmarshalBSON. This behavior matches
	// the behavior of the Go "encoding/json" unmarshaler when the target Go value is a pointer and
	// the JSON field value is "null".
	if val.Kind() == reflect.Ptr && len(src) == 0 {
		val.Set(reflect.Zero(val.Type()))
		return nil
	}

	if !val.Type().Implements(tUnmarshaler) {
		if !val.CanAddr() {
			return ValueDecoderError{Name: "UnmarshalerDecodeValue", Types: []reflect.Type{tUnmarshaler}, Received: val}
		}
		val = val.Addr() // If the type doesn't implement the interface, a pointer to it must.
	}

	m, ok := val.Interface().(Unmarshaler)
	if !ok {
		// NB: this error should be unreachable due to the above checks
		return ValueDecoderError{Name: "UnmarshalerDecodeValue", Types: []reflect.Type{tUnmarshaler}, Received: val}
	}
	return m.UnmarshalBSON(src)
}

// CoreDocumentDecodeValue is the ValueDecoderFunc for bsoncore.Document.
//
// Deprecated: Use [go.mongodb.org/mongo-driver/bson.NewRegistry] to get a registry with all default
// value decoders registered.
func (DefaultValueDecoders) CoreDocumentDecodeValue(_ DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tCoreDocument {
		return ValueDecoderError{Name: "CoreDocumentDecodeValue", Types: []reflect.Type{tCoreDocument}, Received: val}
	}

	if val.IsNil() {
		val.Set(reflect.MakeSlice(val.Type(), 0, 0))
	}

	val.SetLen(0)

	cdoc, err := appendDocumentBytes(val.Interface().(bsoncore.Document), vr)
	val.Set(reflect.ValueOf(cdoc))
	return err
}

func (dvd DefaultValueDecoders) decodeDefault(dc DecodeContext, vr ValueReader, val reflect.Value) ([]reflect.Value, error) {
	elems := make([]reflect.Value, 0)

	ar, err := vr.ReadArray()
	if err != nil {
		return nil, err
	}

	eType := val.Type().Elem()

	var vDecoder ValueDecoder
	if !(eType.Kind() == reflect.Interface && val.Len() > 0) {
		vDecoder, err = dc.LookupDecoder(eType)
		if err != nil {
			return nil, err
		}
	}

	idx := 0
	for {
		vr, err := ar.ReadValue()
		if errors.Is(err, ErrEOA) {
			break
		}
		if err != nil {
			return nil, err
		}

		var elem reflect.Value
		if vDecoder == nil {
			elem = val.Index(idx).Elem()
			if elem.Kind() != reflect.Ptr || elem.IsNil() {
				valueDecoder, err := dc.LookupDecoder(elem.Type())
				if err != nil {
					return nil, err
				}
				err = valueDecoder.DecodeValue(dc, vr, elem)
				if err != nil {
					return nil, newDecodeError(strconv.Itoa(idx), err)
				}
			} else if vr.Type() == TypeNull {
				elem = reflect.Zero(val.Index(idx).Type())
				if err = vr.ReadNull(); err != nil {
					return nil, err
				}
			} else {
				e := elem.Elem()
				valueDecoder, err := dc.LookupDecoder(e.Type())
				if err != nil {
					return nil, err
				}
				err = valueDecoder.DecodeValue(dc, vr, e)
				if err != nil {
					return nil, newDecodeError(strconv.Itoa(idx), err)
				}
			}
		} else {
			elem, err = decodeTypeOrValueWithInfo(vDecoder, dc, vr, eType)
			if err != nil {
				return nil, newDecodeError(strconv.Itoa(idx), err)
			}
		}

		elems = append(elems, elem)
		idx++
	}

	return elems, nil
}

func (dvd DefaultValueDecoders) readCodeWithScope(dc DecodeContext, vr ValueReader) (CodeWithScope, error) {
	var cws CodeWithScope

	code, dr, err := vr.ReadCodeWithScope()
	if err != nil {
		return cws, err
	}

	scope := reflect.New(tD).Elem()
	elems, err := dvd.decodeElemsFromDocumentReader(dc, dr)
	if err != nil {
		return cws, err
	}

	scope.Set(reflect.MakeSlice(tD, 0, len(elems)))
	scope.Set(reflect.Append(scope, elems...))

	cws = CodeWithScope{
		Code:  JavaScript(code),
		Scope: scope.Interface().(D),
	}
	return cws, nil
}

func (dvd DefaultValueDecoders) codeWithScopeDecodeType(dc DecodeContext, vr ValueReader, t reflect.Type) (reflect.Value, error) {
	if t != tCodeWithScope {
		return emptyValue, ValueDecoderError{
			Name:     "CodeWithScopeDecodeValue",
			Types:    []reflect.Type{tCodeWithScope},
			Received: reflect.Zero(t),
		}
	}

	var cws CodeWithScope
	var err error
	switch vrType := vr.Type(); vrType {
	case TypeCodeWithScope:
		cws, err = dvd.readCodeWithScope(dc, vr)
	case TypeNull:
		err = vr.ReadNull()
	case TypeUndefined:
		err = vr.ReadUndefined()
	default:
		return emptyValue, fmt.Errorf("cannot decode %v into a CodeWithScope", vrType)
	}
	if err != nil {
		return emptyValue, err
	}

	return reflect.ValueOf(cws), nil
}

// CodeWithScopeDecodeValue is the ValueDecoderFunc for CodeWithScope.
//
// Deprecated: Use [go.mongodb.org/mongo-driver/bson.NewRegistry] to get a registry with all default
// value decoders registered.
func (dvd DefaultValueDecoders) CodeWithScopeDecodeValue(dc DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tCodeWithScope {
		return ValueDecoderError{Name: "CodeWithScopeDecodeValue", Types: []reflect.Type{tCodeWithScope}, Received: val}
	}

	elem, err := dvd.codeWithScopeDecodeType(dc, vr, tCodeWithScope)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

func (dvd DefaultValueDecoders) decodeD(dc DecodeContext, vr ValueReader, _ reflect.Value) ([]reflect.Value, error) {
	switch vr.Type() {
	case Type(0), TypeEmbeddedDocument:
	default:
		return nil, fmt.Errorf("cannot decode %v into a D", vr.Type())
	}

	dr, err := vr.ReadDocument()
	if err != nil {
		return nil, err
	}

	return dvd.decodeElemsFromDocumentReader(dc, dr)
}

func (DefaultValueDecoders) decodeElemsFromDocumentReader(dc DecodeContext, dr DocumentReader) ([]reflect.Value, error) {
	decoder, err := dc.LookupDecoder(tEmpty)
	if err != nil {
		return nil, err
	}

	elems := make([]reflect.Value, 0)
	for {
		key, vr, err := dr.ReadElement()
		if errors.Is(err, ErrEOD) {
			break
		}
		if err != nil {
			return nil, err
		}

		val := reflect.New(tEmpty).Elem()
		err = decoder.DecodeValue(dc, vr, val)
		if err != nil {
			return nil, newDecodeError(key, err)
		}

		elems = append(elems, reflect.ValueOf(E{Key: key, Value: val.Interface()}))
	}

	return elems, nil
}
