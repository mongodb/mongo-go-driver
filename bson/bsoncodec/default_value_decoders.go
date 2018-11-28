// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncodec

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"reflect"
	"strconv"
	"time"

	"github.com/mongodb/mongo-go-driver/bson/bsonrw"
	"github.com/mongodb/mongo-go-driver/bson/bsontype"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
)

var defaultValueDecoders DefaultValueDecoders

// DefaultValueDecoders is a namespace type for the default ValueDecoders used
// when creating a registry.
type DefaultValueDecoders struct{}

// RegisterDefaultDecoders will register the decoder methods attached to DefaultValueDecoders with
// the provided RegistryBuilder.
//
// There is no support for decoding map[string]interface{} becuase there is no decoder for
// interface{}, so users must either register this decoder themselves or use the
// EmptyInterfaceDecoder avaialble in the bson package.
func (dvd DefaultValueDecoders) RegisterDefaultDecoders(rb *RegistryBuilder) {
	if rb == nil {
		panic(errors.New("argument to RegisterDefaultDecoders must not be nil"))
	}

	rb.
		RegisterDecoder(tByteSlice, ValueDecoderFunc(dvd.ByteSliceDecodeValue)).
		RegisterDecoder(tTime, ValueDecoderFunc(dvd.TimeDecodeValue)).
		RegisterDecoder(tEmpty, ValueDecoderLegacyFunc(dvd.EmptyInterfaceDecodeValue)).
		RegisterDecoder(tOID, ValueDecoderFunc(dvd.ObjectIDDecodeValue)).
		RegisterDecoder(tDecimal, ValueDecoderFunc(dvd.Decimal128DecodeValue)).
		RegisterDecoder(tJSONNumber, ValueDecoderFunc(dvd.JSONNumberDecodeValue)).
		RegisterDecoder(tURL, ValueDecoderFunc(dvd.URLDecodeValue)).
		RegisterDecoder(tValueUnmarshaler, ValueDecoderFunc(dvd.ValueUnmarshalerDecodeValue)).
		RegisterDefaultDecoder(reflect.Bool, ValueDecoderFunc(dvd.BooleanDecodeValue)).
		RegisterDefaultDecoder(reflect.Int, ValueDecoderFunc(dvd.IntDecodeValue)).
		RegisterDefaultDecoder(reflect.Int8, ValueDecoderFunc(dvd.IntDecodeValue)).
		RegisterDefaultDecoder(reflect.Int16, ValueDecoderFunc(dvd.IntDecodeValue)).
		RegisterDefaultDecoder(reflect.Int32, ValueDecoderFunc(dvd.IntDecodeValue)).
		RegisterDefaultDecoder(reflect.Int64, ValueDecoderFunc(dvd.IntDecodeValue)).
		RegisterDefaultDecoder(reflect.Uint, ValueDecoderFunc(dvd.UintDecodeValue)).
		RegisterDefaultDecoder(reflect.Uint8, ValueDecoderFunc(dvd.UintDecodeValue)).
		RegisterDefaultDecoder(reflect.Uint16, ValueDecoderFunc(dvd.UintDecodeValue)).
		RegisterDefaultDecoder(reflect.Uint32, ValueDecoderFunc(dvd.UintDecodeValue)).
		RegisterDefaultDecoder(reflect.Uint64, ValueDecoderFunc(dvd.UintDecodeValue)).
		RegisterDefaultDecoder(reflect.Float32, ValueDecoderFunc(dvd.FloatDecodeValue)).
		RegisterDefaultDecoder(reflect.Float64, ValueDecoderFunc(dvd.FloatDecodeValue)).
		RegisterDefaultDecoder(reflect.Array, ValueDecoderFunc(dvd.ArrayDecodeValue)).
		RegisterDefaultDecoder(reflect.Map, ValueDecoderFunc(dvd.MapDecodeValue)).
		RegisterDefaultDecoder(reflect.Slice, ValueDecoderFunc(dvd.SliceDecodeValue)).
		RegisterDefaultDecoder(reflect.String, ValueDecoderFunc(dvd.StringDecodeValue)).
		RegisterDefaultDecoder(reflect.Struct, &StructCodec{cache: make(map[reflect.Type]*structDescription), parser: DefaultStructTagParser}).
		RegisterDefaultDecoder(reflect.Ptr, NewPointerCodec())
}

// BooleanDecodeValue is the ValueDecoderFunc for bool types.
func (dvd DefaultValueDecoders) BooleanDecodeValue(dctx DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if vr.Type() != bsontype.Boolean {
		return fmt.Errorf("cannot decode %v into a boolean", vr.Type())
	}
	if !val.IsValid() || !val.CanSet() || val.Kind() != reflect.Bool {
		return ValueDecoderError{Name: "BooleanDecodeValue", Kinds: []reflect.Kind{reflect.Bool}, Received: val}
	}

	b, err := vr.ReadBoolean()
	val.SetBool(b)
	return err
}

// IntDecodeValue is the ValueDecoderFunc for bool types.
func (dvd DefaultValueDecoders) IntDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	var i64 int64
	var err error
	switch vr.Type() {
	case bsontype.Int32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		i64 = int64(i32)
	case bsontype.Int64:
		i64, err = vr.ReadInt64()
		if err != nil {
			return err
		}
	case bsontype.Double:
		f64, err := vr.ReadDouble()
		if err != nil {
			return err
		}
		if !dc.Truncate && math.Floor(f64) != f64 {
			return errors.New("IntDecodeValue can only truncate float64 to an integer type when truncation is enabled")
		}
		if f64 > float64(math.MaxInt64) {
			return fmt.Errorf("%g overflows int64", f64)
		}
		i64 = int64(f64)
	default:
		return fmt.Errorf("cannot decode %v into an integer type", vr.Type())
	}

	if !val.CanSet() {
		return ValueDecoderError{
			Name:     "IntDecodeValue",
			Kinds:    []reflect.Kind{reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int},
			Received: val,
		}
	}

	switch val.Kind() {
	case reflect.Int8:
		if i64 < math.MinInt8 || i64 > math.MaxInt8 {
			return fmt.Errorf("%d overflows int8", i64)
		}
	case reflect.Int16:
		if i64 < math.MinInt16 || i64 > math.MaxInt16 {
			return fmt.Errorf("%d overflows int16", i64)
		}
	case reflect.Int32:
		if i64 < math.MinInt32 || i64 > math.MaxInt32 {
			return fmt.Errorf("%d overflows int32", i64)
		}
	case reflect.Int64:
	case reflect.Int:
		if int64(int(i64)) != i64 { // Can we fit this inside of an int
			return fmt.Errorf("%d overflows int", i64)
		}
	default:
		return ValueDecoderError{
			Name:     "IntDecodeValue",
			Kinds:    []reflect.Kind{reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int},
			Received: val,
		}
	}

	val.SetInt(i64)
	return nil
}

// UintDecodeValue is the ValueDecoderFunc for uint types.
func (dvd DefaultValueDecoders) UintDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	var i64 int64
	var err error
	switch vr.Type() {
	case bsontype.Int32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		i64 = int64(i32)
	case bsontype.Int64:
		i64, err = vr.ReadInt64()
		if err != nil {
			return err
		}
	case bsontype.Double:
		f64, err := vr.ReadDouble()
		if err != nil {
			return err
		}
		if !dc.Truncate && math.Floor(f64) != f64 {
			return errors.New("UintDecodeValue can only truncate float64 to an integer type when truncation is enabled")
		}
		if f64 > float64(math.MaxInt64) {
			return fmt.Errorf("%g overflows int64", f64)
		}
		i64 = int64(f64)
	default:
		return fmt.Errorf("cannot decode %v into an integer type", vr.Type())
	}

	if !val.CanSet() {
		return ValueDecoderError{
			Name:     "UintDecodeValue",
			Kinds:    []reflect.Kind{reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint},
			Received: val,
		}
	}

	switch val.Kind() {
	case reflect.Uint8:
		if i64 < 0 || i64 > math.MaxUint8 {
			return fmt.Errorf("%d overflows uint8", i64)
		}
	case reflect.Uint16:
		if i64 < 0 || i64 > math.MaxUint16 {
			return fmt.Errorf("%d overflows uint16", i64)
		}
	case reflect.Uint32:
		if i64 < 0 || i64 > math.MaxUint32 {
			return fmt.Errorf("%d overflows uint32", i64)
		}
	case reflect.Uint64:
		if i64 < 0 {
			return fmt.Errorf("%d overflows uint64", i64)
		}
	case reflect.Uint:
		if i64 < 0 || int64(uint(i64)) != i64 { // Can we fit this inside of an uint
			return fmt.Errorf("%d overflows uint", i64)
		}
	default:
		return ValueDecoderError{
			Name:     "UintDecodeValue",
			Kinds:    []reflect.Kind{reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint},
			Received: val,
		}
	}

	val.SetUint(uint64(i64))
	return nil
}

// FloatDecodeValue is the ValueDecoderFunc for float types.
func (dvd DefaultValueDecoders) FloatDecodeValue(ec DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	var f float64
	var err error
	switch vr.Type() {
	case bsontype.Int32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		f = float64(i32)
	case bsontype.Int64:
		i64, err := vr.ReadInt64()
		if err != nil {
			return err
		}
		f = float64(i64)
	case bsontype.Double:
		f, err = vr.ReadDouble()
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("cannot decode %v into a float32 or float64 type", vr.Type())
	}

	if !val.CanSet() {
		return ValueDecoderError{Name: "FloatDecodeValue", Kinds: []reflect.Kind{reflect.Float32, reflect.Float64}, Received: val}
	}

	switch val.Kind() {
	case reflect.Float32:
		if !ec.Truncate && float64(float32(f)) != f {
			return errors.New("FloatDecodeValue can only convert float64 to float32 when truncation is allowed")
		}
	case reflect.Float64:
	default:
		return ValueDecoderError{Name: "FloatDecodeValue", Kinds: []reflect.Kind{reflect.Float32, reflect.Float64}, Received: val}
	}

	val.SetFloat(f)
	return nil
}

// StringDecodeValue is the ValueDecoderFunc for string types.
func (dvd DefaultValueDecoders) StringDecodeValue(dctx DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	var str string
	var err error
	switch vr.Type() {
	// TODO(GODRIVER-577): Handle JavaScript and Symbol BSON types when allowed.
	case bsontype.String:
		str, err = vr.ReadString()
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("cannot decode %v into a string type", vr.Type())
	}
	if !val.CanSet() || val.Kind() != reflect.String {
		return ValueDecoderError{Name: "StringDecodeValue", Kinds: []reflect.Kind{reflect.String}, Received: val}
	}

	val.SetString(str)
	return nil
}

// ObjectIDDecodeValue is the ValueDecoderFunc for objectid.ObjectID.
func (dvd DefaultValueDecoders) ObjectIDDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if vr.Type() != bsontype.ObjectID {
		return fmt.Errorf("cannot decode %v into an ObjectID", vr.Type())
	}

	if !val.CanSet() || val.Type() != tOID {
		return ValueDecoderError{Name: "ObjectIDDecodeValue", Types: []reflect.Type{tOID}, Received: val}
	}
	oid, err := vr.ReadObjectID()
	val.Set(reflect.ValueOf(oid))
	return err
}

// Decimal128DecodeValue is the ValueDecoderFunc for decimal.Decimal128.
func (dvd DefaultValueDecoders) Decimal128DecodeValue(dctx DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if vr.Type() != bsontype.Decimal128 {
		return fmt.Errorf("cannot decode %v into a decimal.Decimal128", vr.Type())
	}

	if !val.CanSet() || val.Type() != tDecimal {
		return ValueDecoderError{Name: "Decimal128DecodeValue", Types: []reflect.Type{tDecimal}, Received: val}
	}
	d128, err := vr.ReadDecimal128()
	val.Set(reflect.ValueOf(d128))
	return err
}

// JSONNumberDecodeValue is the ValueDecoderFunc for json.Number.
func (dvd DefaultValueDecoders) JSONNumberDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tJSONNumber {
		return ValueDecoderError{Name: "JSONNumberDecodeValue", Types: []reflect.Type{tJSONNumber}, Received: val}
	}

	switch vr.Type() {
	case bsontype.Double:
		f64, err := vr.ReadDouble()
		if err != nil {
			return err
		}
		val.Set(reflect.ValueOf(json.Number(strconv.FormatFloat(f64, 'g', -1, 64))))
	case bsontype.Int32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		val.Set(reflect.ValueOf(json.Number(strconv.FormatInt(int64(i32), 10))))
	case bsontype.Int64:
		i64, err := vr.ReadInt64()
		if err != nil {
			return err
		}
		val.Set(reflect.ValueOf(json.Number(strconv.FormatInt(i64, 10))))
	default:
		return fmt.Errorf("cannot decode %v into a json.Number", vr.Type())
	}

	return nil
}

// URLDecodeValue is the ValueDecoderFunc for url.URL.
func (dvd DefaultValueDecoders) URLDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if vr.Type() != bsontype.String {
		return fmt.Errorf("cannot decode %v into a *url.URL", vr.Type())
	}

	str, err := vr.ReadString()
	if err != nil {
		return err
	}

	u, err := url.Parse(str)
	if err != nil {
		return err
	}

	if !val.CanSet() || val.Type() != tURL {
		return ValueDecoderError{Name: "URLDecodeValue", Types: []reflect.Type{tURL}, Received: val}
	}

	val.Set(reflect.ValueOf(u).Elem())
	return nil
}

// TimeDecodeValue is the ValueDecoderFunc for time.Time.
func (dvd DefaultValueDecoders) TimeDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if vr.Type() != bsontype.DateTime {
		return fmt.Errorf("cannot decode %v into a time.Time", vr.Type())
	}

	dt, err := vr.ReadDateTime()
	if err != nil {
		return err
	}

	if !val.CanSet() || val.Type() != tTime {
		return ValueDecoderError{Name: "TimeDecodeValue", Types: []reflect.Type{tTime}, Received: val}
	}

	val.Set(reflect.ValueOf(time.Unix(dt/1000, dt%1000*1000000)))
	return nil
}

// ByteSliceDecodeValue is the ValueDecoderFunc for []byte.
func (dvd DefaultValueDecoders) ByteSliceDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if vr.Type() != bsontype.Binary {
		return fmt.Errorf("cannot decode %v into a []byte", vr.Type())
	}

	if !val.CanSet() || val.Type() != tByteSlice {
		return ValueDecoderError{Name: "ByteSliceDecodeValue", Types: []reflect.Type{tByteSlice}, Received: val}
	}

	data, subtype, err := vr.ReadBinary()
	if err != nil {
		return err
	}
	if subtype != 0x00 {
		return fmt.Errorf("ByteSliceDecodeValue can only be used to decode subtype 0x00 for %s, got %v", bsontype.Binary, subtype)
	}

	val.Set(reflect.ValueOf(data))
	return nil
}

// MapDecodeValue is the ValueDecoderFunc for map[string]* types.
func (dvd DefaultValueDecoders) MapDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Kind() != reflect.Map || val.Type().Key().Kind() != reflect.String {
		return ValueDecoderError{Name: "MapDecodeValue", Kinds: []reflect.Kind{reflect.Map}, Received: val}
	}

	dr, err := vr.ReadDocument()
	if err != nil {
		return err
	}

	if val.IsNil() {
		val.Set(reflect.MakeMap(val.Type()))
	}

	eType := val.Type().Elem()
	decoder, err := dc.LookupDecoder(eType)
	if err != nil {
		return err
	}

	for {
		key, vr, err := dr.ReadElement()
		if err == bsonrw.ErrEOD {
			break
		}
		if err != nil {
			return err
		}

		ptr := reflect.New(eType)

		err = decoder.DecodeValueLegacy(dc, vr, ptr.Interface())
		if err != nil {
			return err
		}

		val.SetMapIndex(reflect.ValueOf(key), ptr.Elem())
	}
	return nil
}

// SliceDecodeValue is the ValueDecoderFunc for array types.
func (dvd DefaultValueDecoders) ArrayDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if !val.IsValid() || val.Kind() != reflect.Array {
		return ValueDecoderError{Name: "ArrayDecodeValue", Kinds: []reflect.Kind{reflect.Array}, Received: val}
	}

	if vr.Type() != bsontype.Array {
		return fmt.Errorf("cannot decode %v into an array", vr.Type())
	}

	ar, err := vr.ReadArray()
	if err != nil {
		return err
	}

	eType := val.Type().Elem()

	elems, err := dvd.decodeDefault(dc, ar, eType)
	if err != nil {
		return err
	}

	if len(elems) > val.Len() {
		return fmt.Errorf("more elements returned in array than can fit inside %s", val.Type())
	}

	for idx, elem := range elems {
		val.Index(idx).Set(elem)
	}

	return nil
}

// SliceDecodeValue is the ValueDecoderFunc for slice types.
func (dvd DefaultValueDecoders) SliceDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Kind() != reflect.Slice {
		return ValueDecoderError{Name: "SliceDecodeValue", Kinds: []reflect.Kind{reflect.Slice}, Received: val}
	}

	switch vr.Type() {
	case bsontype.Array:
	case bsontype.Null:
		val.Set(reflect.Zero(val.Type()))
		return vr.ReadNull()
	default:
		return fmt.Errorf("cannot decode %v into a slice", vr.Type())
	}

	ar, err := vr.ReadArray()
	if err != nil {
		return err
	}

	eType := val.Type().Elem()

	elems, err := dvd.decodeDefault(dc, ar, eType)
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

// ValueUnmarshalerDecodeValue is the ValueDecoderFunc for ValueUnmarshaler implementations.
func (dvd DefaultValueDecoders) ValueUnmarshalerDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
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
		val = val.Addr() // If they type doesn't implement the interface, a pointer to it must.
	}

	t, src, err := bsonrw.Copier{}.CopyValueToBytes(vr)
	if err != nil {
		return err
	}

	fn := val.Convert(tValueUnmarshaler).MethodByName("UnmarshalBSONValue")
	errVal := fn.Call([]reflect.Value{reflect.ValueOf(t), reflect.ValueOf(src)})[0]
	if !errVal.IsNil() {
		return errVal.Interface().(error)
	}
	return nil
}

// EmptyInterfaceDecodeValue is the ValueDecoderFunc for interface{}.
func (dvd DefaultValueDecoders) EmptyInterfaceDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
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
	case bsontype.Boolean:
		val = new(bool)
		rtype = tBool
		fn = func() { *target = *(val.(*bool)) }
	case bsontype.Int32:
		val = new(int32)
		rtype = tInt32
		fn = func() { *target = *(val.(*int32)) }
	case bsontype.Int64:
		val = new(int64)
		rtype = tInt64
		fn = func() { *target = *(val.(*int64)) }
	case bsontype.Decimal128:
		val = new(decimal.Decimal128)
		rtype = tDecimal
		fn = func() { *target = *(val.(*decimal.Decimal128)) }
	default:
		return fmt.Errorf("Type %s is not a valid BSON type and has no default Go type to decode into", vr.Type())
	}

	decoder, err := dc.LookupDecoder(rtype)
	if err != nil {
		return err
	}
	err = decoder.DecodeValueLegacy(dc, vr, val)
	if err != nil {
		return err
	}

	fn()
	return nil
}

func (dvd DefaultValueDecoders) decodeDefault(dc DecodeContext, ar bsonrw.ArrayReader, eType reflect.Type) ([]reflect.Value, error) {
	elems := make([]reflect.Value, 0)

	decoder, err := dc.LookupDecoder(eType)
	if err != nil {
		return nil, err
	}

	for {
		vr, err := ar.ReadValue()
		if err == bsonrw.ErrEOA {
			break
		}
		if err != nil {
			return nil, err
		}

		ptr := reflect.New(eType)

		err = decoder.DecodeValueLegacy(dc, vr, ptr.Interface())
		if err != nil {
			return nil, err
		}
		elems = append(elems, ptr.Elem())
	}

	return elems, nil
}
