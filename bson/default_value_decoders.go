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
	"net/url"
	"reflect"
	"strconv"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

var (
	errCannotTruncate = errors.New("float64 can only be truncated to a lower precision type when truncation is enabled")
)

type decodeBinaryError struct {
	subtype  byte
	typeName string
}

func (d decodeBinaryError) Error() string {
	return fmt.Sprintf("only binary values with subtype 0x00 or 0x02 can be decoded into %s, but got subtype %v", d.typeName, d.subtype)
}

// registerDefaultDecoders will register the default decoder methods with the provided Registry.
//
// There is no support for decoding map[string]interface{} because there is no decoder for
// interface{}, so users must either register this decoder themselves or use the
// EmptyInterfaceDecoder available in the bson package.
func registerDefaultDecoders(rb *RegistryBuilder) {
	if rb == nil {
		panic(errors.New("argument to RegisterDefaultDecoders must not be nil"))
	}

	intDecoder := func() ValueDecoder { return &intCodec{} }
	floatDecoder := func() ValueDecoder { return &floatCodec{} }
	rb.RegisterTypeDecoder(tD, func() ValueDecoder { return ValueDecoderFunc(dDecodeValue) }).
		RegisterTypeDecoder(tBinary, func() ValueDecoder { return &decodeAdapter{binaryDecodeValue, binaryDecodeType} }).
		RegisterTypeDecoder(tUndefined, func() ValueDecoder { return &decodeAdapter{undefinedDecodeValue, undefinedDecodeType} }).
		RegisterTypeDecoder(tDateTime, func() ValueDecoder { return &decodeAdapter{dateTimeDecodeValue, dateTimeDecodeType} }).
		RegisterTypeDecoder(tNull, func() ValueDecoder { return &decodeAdapter{nullDecodeValue, nullDecodeType} }).
		RegisterTypeDecoder(tRegex, func() ValueDecoder { return &decodeAdapter{regexDecodeValue, regexDecodeType} }).
		RegisterTypeDecoder(tDBPointer, func() ValueDecoder { return &decodeAdapter{dbPointerDecodeValue, dbPointerDecodeType} }).
		RegisterTypeDecoder(tTimestamp, func() ValueDecoder { return &decodeAdapter{timestampDecodeValue, timestampDecodeType} }).
		RegisterTypeDecoder(tMinKey, func() ValueDecoder { return &decodeAdapter{minKeyDecodeValue, minKeyDecodeType} }).
		RegisterTypeDecoder(tMaxKey, func() ValueDecoder { return &decodeAdapter{maxKeyDecodeValue, maxKeyDecodeType} }).
		RegisterTypeDecoder(tJavaScript, func() ValueDecoder { return &decodeAdapter{javaScriptDecodeValue, javaScriptDecodeType} }).
		RegisterTypeDecoder(tSymbol, func() ValueDecoder { return &decodeAdapter{symbolDecodeValue, symbolDecodeType} }).
		RegisterTypeDecoder(tByteSlice, func() ValueDecoder { return &byteSliceCodec{} }).
		RegisterTypeDecoder(tTime, func() ValueDecoder { return &timeCodec{} }).
		RegisterTypeDecoder(tEmpty, func() ValueDecoder { return &emptyInterfaceCodec{} }).
		RegisterTypeDecoder(tCoreArray, func() ValueDecoder { return &arrayCodec{} }).
		RegisterTypeDecoder(tOID, func() ValueDecoder { return &decodeAdapter{objectIDDecodeValue, objectIDDecodeType} }).
		RegisterTypeDecoder(tDecimal, func() ValueDecoder { return &decodeAdapter{decimal128DecodeValue, decimal128DecodeType} }).
		RegisterTypeDecoder(tJSONNumber, func() ValueDecoder { return &decodeAdapter{jsonNumberDecodeValue, jsonNumberDecodeType} }).
		RegisterTypeDecoder(tURL, func() ValueDecoder { return &decodeAdapter{urlDecodeValue, urlDecodeType} }).
		RegisterTypeDecoder(tCoreDocument, func() ValueDecoder { return ValueDecoderFunc(coreDocumentDecodeValue) }).
		RegisterTypeDecoder(tCodeWithScope, func() ValueDecoder { return &decodeAdapter{codeWithScopeDecodeValue, codeWithScopeDecodeType} }).
		RegisterKindDecoder(reflect.Bool, func() ValueDecoder { return &decodeAdapter{booleanDecodeValue, booleanDecodeType} }).
		RegisterKindDecoder(reflect.Int, intDecoder).
		RegisterKindDecoder(reflect.Int8, intDecoder).
		RegisterKindDecoder(reflect.Int16, intDecoder).
		RegisterKindDecoder(reflect.Int32, intDecoder).
		RegisterKindDecoder(reflect.Int64, intDecoder).
		RegisterKindDecoder(reflect.Uint, intDecoder).
		RegisterKindDecoder(reflect.Uint8, intDecoder).
		RegisterKindDecoder(reflect.Uint16, intDecoder).
		RegisterKindDecoder(reflect.Uint32, intDecoder).
		RegisterKindDecoder(reflect.Uint64, intDecoder).
		RegisterKindDecoder(reflect.Float32, floatDecoder).
		RegisterKindDecoder(reflect.Float64, floatDecoder).
		RegisterKindDecoder(reflect.Array, func() ValueDecoder { return ValueDecoderFunc(arrayDecodeValue) }).
		RegisterKindDecoder(reflect.Map, func() ValueDecoder { return &mapCodec{} }).
		RegisterKindDecoder(reflect.Slice, func() ValueDecoder { return &sliceCodec{} }).
		RegisterKindDecoder(reflect.String, func() ValueDecoder { return &stringCodec{} }).
		RegisterKindDecoder(reflect.Struct, func() ValueDecoder { return newStructCodec(DefaultStructTagParser) }).
		RegisterKindDecoder(reflect.Ptr, func() ValueDecoder { return &pointerCodec{} }).
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
		RegisterInterfaceDecoder(tValueUnmarshaler, func() ValueDecoder { return ValueDecoderFunc(valueUnmarshalerDecodeValue) }).
		RegisterInterfaceDecoder(tUnmarshaler, func() ValueDecoder { return ValueDecoderFunc(unmarshalerDecodeValue) })
}

// dDecodeValue is the ValueDecoderFunc for D instances.
func dDecodeValue(reg DecoderRegistry, vr ValueReader, val reflect.Value) error {
	if !val.IsValid() || !val.CanSet() || val.Type() != tD {
		return ValueDecoderError{Name: "DDecodeValue", Kinds: []reflect.Kind{reflect.Slice}, Received: val}
	}

	switch vrType := vr.Type(); vrType {
	case Type(0), TypeEmbeddedDocument:
		break
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

	decoder, err := reg.LookupDecoder(tEmpty)
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

		elem, err := decodeTypeOrValueWithInfo(decoder, reg, elemVr, tD)
		if err != nil {
			return err
		}

		elems = append(elems, E{Key: key, Value: elem.Interface()})
	}

	val.Set(reflect.ValueOf(elems))
	return nil
}

func booleanDecodeType(_ DecoderRegistry, vr ValueReader, t reflect.Type) (reflect.Value, error) {
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

// booleanDecodeValue is the ValueDecoderFunc for bool types.
func booleanDecodeValue(reg DecoderRegistry, vr ValueReader, val reflect.Value) error {
	if !val.IsValid() || !val.CanSet() || val.Kind() != reflect.Bool {
		return ValueDecoderError{Name: "BooleanDecodeValue", Kinds: []reflect.Kind{reflect.Bool}, Received: val}
	}

	elem, err := booleanDecodeType(reg, vr, val.Type())
	if err != nil {
		return err
	}

	val.SetBool(elem.Bool())
	return nil
}

func javaScriptDecodeType(_ DecoderRegistry, vr ValueReader, t reflect.Type) (reflect.Value, error) {
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

// javaScriptDecodeValue is the ValueDecoderFunc for the JavaScript type.
func javaScriptDecodeValue(reg DecoderRegistry, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tJavaScript {
		return ValueDecoderError{Name: "JavaScriptDecodeValue", Types: []reflect.Type{tJavaScript}, Received: val}
	}

	elem, err := javaScriptDecodeType(reg, vr, tJavaScript)
	if err != nil {
		return err
	}

	val.SetString(elem.String())
	return nil
}

func symbolDecodeType(_ DecoderRegistry, vr ValueReader, t reflect.Type) (reflect.Value, error) {
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

// symbolDecodeValue is the ValueDecoderFunc for the Symbol type.
func symbolDecodeValue(reg DecoderRegistry, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tSymbol {
		return ValueDecoderError{Name: "SymbolDecodeValue", Types: []reflect.Type{tSymbol}, Received: val}
	}

	elem, err := symbolDecodeType(reg, vr, tSymbol)
	if err != nil {
		return err
	}

	val.SetString(elem.String())
	return nil
}

func binaryDecodeType(_ DecoderRegistry, vr ValueReader, t reflect.Type) (reflect.Value, error) {
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

// binaryDecodeValue is the ValueDecoderFunc for Binary.
func binaryDecodeValue(reg DecoderRegistry, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tBinary {
		return ValueDecoderError{Name: "BinaryDecodeValue", Types: []reflect.Type{tBinary}, Received: val}
	}

	elem, err := binaryDecodeType(reg, vr, tBinary)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

func undefinedDecodeType(_ DecoderRegistry, vr ValueReader, t reflect.Type) (reflect.Value, error) {
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

// undefinedDecodeValue is the ValueDecoderFunc for Undefined.
func undefinedDecodeValue(reg DecoderRegistry, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tUndefined {
		return ValueDecoderError{Name: "UndefinedDecodeValue", Types: []reflect.Type{tUndefined}, Received: val}
	}

	elem, err := undefinedDecodeType(reg, vr, tUndefined)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

// Accept both 12-byte string and pretty-printed 24-byte hex string formats.
func objectIDDecodeType(_ DecoderRegistry, vr ValueReader, t reflect.Type) (reflect.Value, error) {
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

// objectIDDecodeValue is the ValueDecoderFunc for ObjectID.
func objectIDDecodeValue(reg DecoderRegistry, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tOID {
		return ValueDecoderError{Name: "ObjectIDDecodeValue", Types: []reflect.Type{tOID}, Received: val}
	}

	elem, err := objectIDDecodeType(reg, vr, tOID)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

func dateTimeDecodeType(_ DecoderRegistry, vr ValueReader, t reflect.Type) (reflect.Value, error) {
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

// dateTimeDecodeValue is the ValueDecoderFunc for DateTime.
func dateTimeDecodeValue(reg DecoderRegistry, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tDateTime {
		return ValueDecoderError{Name: "DateTimeDecodeValue", Types: []reflect.Type{tDateTime}, Received: val}
	}

	elem, err := dateTimeDecodeType(reg, vr, tDateTime)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

func nullDecodeType(_ DecoderRegistry, vr ValueReader, t reflect.Type) (reflect.Value, error) {
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

// nullDecodeValue is the ValueDecoderFunc for Null.
func nullDecodeValue(reg DecoderRegistry, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tNull {
		return ValueDecoderError{Name: "NullDecodeValue", Types: []reflect.Type{tNull}, Received: val}
	}

	elem, err := nullDecodeType(reg, vr, tNull)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

func regexDecodeType(_ DecoderRegistry, vr ValueReader, t reflect.Type) (reflect.Value, error) {
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

// regexDecodeValue is the ValueDecoderFunc for Regex.
func regexDecodeValue(reg DecoderRegistry, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tRegex {
		return ValueDecoderError{Name: "RegexDecodeValue", Types: []reflect.Type{tRegex}, Received: val}
	}

	elem, err := regexDecodeType(reg, vr, tRegex)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

func dbPointerDecodeType(_ DecoderRegistry, vr ValueReader, t reflect.Type) (reflect.Value, error) {
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

// dbPointerDecodeValue is the ValueDecoderFunc for DBPointer.
func dbPointerDecodeValue(reg DecoderRegistry, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tDBPointer {
		return ValueDecoderError{Name: "DBPointerDecodeValue", Types: []reflect.Type{tDBPointer}, Received: val}
	}

	elem, err := dbPointerDecodeType(reg, vr, tDBPointer)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

func timestampDecodeType(_ DecoderRegistry, vr ValueReader, reflectType reflect.Type) (reflect.Value, error) {
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

// timestampDecodeValue is the ValueDecoderFunc for Timestamp.
func timestampDecodeValue(reg DecoderRegistry, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tTimestamp {
		return ValueDecoderError{Name: "TimestampDecodeValue", Types: []reflect.Type{tTimestamp}, Received: val}
	}

	elem, err := timestampDecodeType(reg, vr, tTimestamp)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

func minKeyDecodeType(_ DecoderRegistry, vr ValueReader, t reflect.Type) (reflect.Value, error) {
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

// minKeyDecodeValue is the ValueDecoderFunc for MinKey.
func minKeyDecodeValue(reg DecoderRegistry, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tMinKey {
		return ValueDecoderError{Name: "MinKeyDecodeValue", Types: []reflect.Type{tMinKey}, Received: val}
	}

	elem, err := minKeyDecodeType(reg, vr, tMinKey)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

func maxKeyDecodeType(_ DecoderRegistry, vr ValueReader, t reflect.Type) (reflect.Value, error) {
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

// maxKeyDecodeValue is the ValueDecoderFunc for MaxKey.
func maxKeyDecodeValue(reg DecoderRegistry, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tMaxKey {
		return ValueDecoderError{Name: "MaxKeyDecodeValue", Types: []reflect.Type{tMaxKey}, Received: val}
	}

	elem, err := maxKeyDecodeType(reg, vr, tMaxKey)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

func decimal128DecodeType(_ DecoderRegistry, vr ValueReader, t reflect.Type) (reflect.Value, error) {
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

// decimal128DecodeValue is the ValueDecoderFunc for Decimal128.
func decimal128DecodeValue(reg DecoderRegistry, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tDecimal {
		return ValueDecoderError{Name: "Decimal128DecodeValue", Types: []reflect.Type{tDecimal}, Received: val}
	}

	elem, err := decimal128DecodeType(reg, vr, tDecimal)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

func jsonNumberDecodeType(_ DecoderRegistry, vr ValueReader, t reflect.Type) (reflect.Value, error) {
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

// jsonNumberDecodeValue is the ValueDecoderFunc for json.Number.
func jsonNumberDecodeValue(reg DecoderRegistry, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tJSONNumber {
		return ValueDecoderError{Name: "JSONNumberDecodeValue", Types: []reflect.Type{tJSONNumber}, Received: val}
	}

	elem, err := jsonNumberDecodeType(reg, vr, tJSONNumber)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

func urlDecodeType(_ DecoderRegistry, vr ValueReader, t reflect.Type) (reflect.Value, error) {
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

// urlDecodeValue is the ValueDecoderFunc for url.URL.
func urlDecodeValue(reg DecoderRegistry, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tURL {
		return ValueDecoderError{Name: "URLDecodeValue", Types: []reflect.Type{tURL}, Received: val}
	}

	elem, err := urlDecodeType(reg, vr, tURL)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

// arrayDecodeValue is the ValueDecoderFunc for array types.
func arrayDecodeValue(reg DecoderRegistry, vr ValueReader, val reflect.Value) error {
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

	var elemsFunc func(DecoderRegistry, ValueReader, reflect.Value) ([]reflect.Value, error)
	switch val.Type().Elem() {
	case tE:
		elemsFunc = decodeD
	default:
		elemsFunc = decodeDefault
	}

	elems, err := elemsFunc(reg, vr, val)
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

// valueUnmarshalerDecodeValue is the ValueDecoderFunc for ValueUnmarshaler implementations.
func valueUnmarshalerDecodeValue(_ DecoderRegistry, vr ValueReader, val reflect.Value) error {
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

// unmarshalerDecodeValue is the ValueDecoderFunc for Unmarshaler implementations.
func unmarshalerDecodeValue(_ DecoderRegistry, vr ValueReader, val reflect.Value) error {
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

// coreDocumentDecodeValue is the ValueDecoderFunc for bsoncore.Document.
func coreDocumentDecodeValue(_ DecoderRegistry, vr ValueReader, val reflect.Value) error {
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

func decodeDefault(reg DecoderRegistry, vr ValueReader, val reflect.Value) ([]reflect.Value, error) {
	elems := make([]reflect.Value, 0)

	ar, err := vr.ReadArray()
	if err != nil {
		return nil, err
	}

	eType := val.Type().Elem()

	decoder, err := reg.LookupDecoder(eType)
	if err != nil {
		return nil, err
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

		elem, err := decodeTypeOrValueWithInfo(decoder, reg, vr, eType)
		if err != nil {
			return nil, newDecodeError(strconv.Itoa(idx), err)
		}
		if elem.Type() != eType {
			elem = elem.Convert(eType)
		}
		elems = append(elems, elem)
		idx++
	}

	return elems, nil
}

func codeWithScopeDecodeType(reg DecoderRegistry, vr ValueReader, t reflect.Type) (reflect.Value, error) {
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
		code, dr, err := vr.ReadCodeWithScope()
		if err != nil {
			return emptyValue, err
		}

		scope := reflect.New(tD).Elem()
		elems, err := decodeElemsFromDocumentReader(reg, dr, tEmpty)
		if err != nil {
			return emptyValue, err
		}

		scope.Set(reflect.MakeSlice(tD, 0, len(elems)))
		scope.Set(reflect.Append(scope, elems...))

		cws = CodeWithScope{
			Code:  JavaScript(code),
			Scope: scope.Interface().(D),
		}
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

// codeWithScopeDecodeValue is the ValueDecoderFunc for CodeWithScope.
func codeWithScopeDecodeValue(reg DecoderRegistry, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tCodeWithScope {
		return ValueDecoderError{Name: "CodeWithScopeDecodeValue", Types: []reflect.Type{tCodeWithScope}, Received: val}
	}

	elem, err := codeWithScopeDecodeType(reg, vr, tCodeWithScope)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

func decodeD(reg DecoderRegistry, vr ValueReader, val reflect.Value) ([]reflect.Value, error) {
	switch vr.Type() {
	case Type(0), TypeEmbeddedDocument:
		break
	default:
		return nil, fmt.Errorf("cannot decode %v into a D", vr.Type())
	}

	dr, err := vr.ReadDocument()
	if err != nil {
		return nil, err
	}

	return decodeElemsFromDocumentReader(reg, dr, val.Type())
}

func decodeElemsFromDocumentReader(reg DecoderRegistry, dr DocumentReader, t reflect.Type) ([]reflect.Value, error) {
	decoder, err := reg.LookupDecoder(tEmpty)
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

		var val reflect.Value
		val, err = decodeTypeOrValueWithInfo(decoder, reg, vr, t)
		if err != nil {
			return nil, newDecodeError(key, err)
		}

		elems = append(elems, reflect.ValueOf(E{Key: key, Value: val.Interface()}))
	}

	return elems, nil
}
