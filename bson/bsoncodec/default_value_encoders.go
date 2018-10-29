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
	"time"

	"github.com/mongodb/mongo-go-driver/bson/bsonrw"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

var defaultValueEncoders DefaultValueEncoders

// DefaultValueEncoders is a namespace type for the default ValueEncoders used
// when creating a registry.
type DefaultValueEncoders struct{}

// RegisterDefaultEncoders will register the encoder methods attached to DefaultValueEncoders with
// the provided RegistryBuilder.
func (dve DefaultValueEncoders) RegisterDefaultEncoders(rb *RegistryBuilder) {
	if rb == nil {
		panic(errors.New("argument to RegisterDefaultEncoders must not be nil"))
	}
	rb.
		RegisterEncoder(tByteSlice, ValueEncoderFunc(dve.ByteSliceEncodeValue)).
		RegisterEncoder(tTime, ValueEncoderFunc(dve.TimeEncodeValue)).
		RegisterEncoder(reflect.PtrTo(tEmpty), ValueEncoderFunc(dve.EmptyInterfaceEncodeValue)).
		RegisterEncoder(tOID, ValueEncoderFunc(dve.ObjectIDEncodeValue)).
		RegisterEncoder(tDecimal, ValueEncoderFunc(dve.Decimal128EncodeValue)).
		RegisterEncoder(tJSONNumber, ValueEncoderFunc(dve.JSONNumberEncodeValue)).
		RegisterEncoder(tURL, ValueEncoderFunc(dve.URLEncodeValue)).
		RegisterEncoder(tValueMarshaler, ValueEncoderFunc(dve.ValueMarshalerEncodeValue)).
		RegisterEncoder(tProxy, ValueEncoderFunc(dve.ProxyEncodeValue)).
		RegisterDefaultEncoder(reflect.Bool, ValueEncoderFunc(dve.BooleanEncodeValue)).
		RegisterDefaultEncoder(reflect.Int, ValueEncoderFunc(dve.IntEncodeValue)).
		RegisterDefaultEncoder(reflect.Int8, ValueEncoderFunc(dve.IntEncodeValue)).
		RegisterDefaultEncoder(reflect.Int16, ValueEncoderFunc(dve.IntEncodeValue)).
		RegisterDefaultEncoder(reflect.Int32, ValueEncoderFunc(dve.IntEncodeValue)).
		RegisterDefaultEncoder(reflect.Int64, ValueEncoderFunc(dve.IntEncodeValue)).
		RegisterDefaultEncoder(reflect.Uint, ValueEncoderFunc(dve.UintEncodeValue)).
		RegisterDefaultEncoder(reflect.Uint8, ValueEncoderFunc(dve.UintEncodeValue)).
		RegisterDefaultEncoder(reflect.Uint16, ValueEncoderFunc(dve.UintEncodeValue)).
		RegisterDefaultEncoder(reflect.Uint32, ValueEncoderFunc(dve.UintEncodeValue)).
		RegisterDefaultEncoder(reflect.Uint64, ValueEncoderFunc(dve.UintEncodeValue)).
		RegisterDefaultEncoder(reflect.Float32, ValueEncoderFunc(dve.FloatEncodeValue)).
		RegisterDefaultEncoder(reflect.Float64, ValueEncoderFunc(dve.FloatEncodeValue)).
		RegisterDefaultEncoder(reflect.Array, ValueEncoderFunc(dve.SliceEncodeValue)).
		RegisterDefaultEncoder(reflect.Map, ValueEncoderFunc(dve.MapEncodeValue)).
		RegisterDefaultEncoder(reflect.Slice, ValueEncoderFunc(dve.SliceEncodeValue)).
		RegisterDefaultEncoder(reflect.String, ValueEncoderFunc(dve.StringEncodeValue)).
		RegisterDefaultEncoder(reflect.Struct, &StructCodec{cache: make(map[reflect.Type]*structDescription), parser: DefaultStructTagParser})
}

// BooleanEncodeValue is the ValueEncoderFunc for bool types.
func (dve DefaultValueEncoders) BooleanEncodeValue(ectx EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	b, ok := i.(bool)
	if !ok {
		if reflect.TypeOf(i).Kind() != reflect.Bool {
			return ValueEncoderError{Name: "BooleanEncodeValue", Types: []interface{}{bool(true)}, Received: i}
		}

		b = reflect.ValueOf(i).Bool()
	}

	return vw.WriteBoolean(b)
}

// IntEncodeValue is the ValueEncoderFunc for int types.
func (dve DefaultValueEncoders) IntEncodeValue(ec EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	switch t := i.(type) {
	case int8:
		return vw.WriteInt32(int32(t))
	case int16:
		return vw.WriteInt32(int32(t))
	case int32:
		return vw.WriteInt32(t)
	case int64:
		if ec.MinSize && t <= math.MaxInt32 {
			return vw.WriteInt32(int32(t))
		}
		return vw.WriteInt64(t)
	case int:
		if ec.MinSize && t <= math.MaxInt32 {
			return vw.WriteInt32(int32(t))
		}
		return vw.WriteInt64(int64(t))
	}

	val := reflect.ValueOf(i)
	switch val.Type().Kind() {
	case reflect.Int8, reflect.Int16, reflect.Int32:
		return vw.WriteInt32(int32(val.Int()))
	case reflect.Int, reflect.Int64:
		i64 := val.Int()
		if ec.MinSize && i64 <= math.MaxInt32 {
			return vw.WriteInt32(int32(i64))
		}
		return vw.WriteInt64(i64)
	}

	return ValueEncoderError{
		Name:     "IntEncodeValue",
		Types:    []interface{}{int8(0), int16(0), int32(0), int64(0), int(0)},
		Received: i,
	}
}

// UintEncodeValue is the ValueEncoderFunc for uint types.
func (dve DefaultValueEncoders) UintEncodeValue(ec EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	switch t := i.(type) {
	case uint8:
		return vw.WriteInt32(int32(t))
	case uint16:
		return vw.WriteInt32(int32(t))
	case uint:
		if ec.MinSize && t <= math.MaxInt32 {
			return vw.WriteInt32(int32(t))
		}
		if uint64(t) > math.MaxInt64 {
			return fmt.Errorf("%d overflows int64", t)
		}
		return vw.WriteInt64(int64(t))
	case uint32:
		if ec.MinSize && t <= math.MaxInt32 {
			return vw.WriteInt32(int32(t))
		}
		return vw.WriteInt64(int64(t))
	case uint64:
		if ec.MinSize && t <= math.MaxInt32 {
			return vw.WriteInt32(int32(t))
		}
		if t > math.MaxInt64 {
			return fmt.Errorf("%d overflows int64", t)
		}
		return vw.WriteInt64(int64(t))
	}

	val := reflect.ValueOf(i)
	switch val.Type().Kind() {
	case reflect.Uint8, reflect.Uint16:
		return vw.WriteInt32(int32(val.Uint()))
	case reflect.Uint, reflect.Uint32, reflect.Uint64:
		u64 := val.Uint()
		if ec.MinSize && u64 <= math.MaxInt32 {
			return vw.WriteInt32(int32(u64))
		}
		if u64 > math.MaxInt64 {
			return fmt.Errorf("%d overflows int64", u64)
		}
		return vw.WriteInt64(int64(u64))
	}

	return ValueEncoderError{
		Name:     "UintEncodeValue",
		Types:    []interface{}{uint8(0), uint16(0), uint32(0), uint64(0), uint(0)},
		Received: i,
	}
}

// FloatEncodeValue is the ValueEncoderFunc for float types.
func (dve DefaultValueEncoders) FloatEncodeValue(ec EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	switch t := i.(type) {
	case float32:
		return vw.WriteDouble(float64(t))
	case float64:
		return vw.WriteDouble(t)
	}

	val := reflect.ValueOf(i)
	switch val.Type().Kind() {
	case reflect.Float32, reflect.Float64:
		return vw.WriteDouble(val.Float())
	}

	return ValueEncoderError{Name: "FloatEncodeValue", Types: []interface{}{float32(0), float64(0)}, Received: i}
}

// StringEncodeValue is the ValueEncoderFunc for string types.
func (dve DefaultValueEncoders) StringEncodeValue(ectx EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	switch t := i.(type) {
	// TODO(GODRIVER-577): Encode strings to either JavaScript or Symbol.
	case string:
		return vw.WriteString(t)
	}

	val := reflect.ValueOf(i)
	if val.Type().Kind() != reflect.String {
		return ValueEncoderError{
			Name:     "StringEncodeValue",
			Types:    []interface{}{string("")},
			Received: i,
		}
	}

	return vw.WriteString(val.String())
}

// ObjectIDEncodeValue is the ValueEncoderFunc for objectid.ObjectID.
func (dve DefaultValueEncoders) ObjectIDEncodeValue(ec EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var oid objectid.ObjectID
	switch t := i.(type) {
	case objectid.ObjectID:
		oid = t
	case *objectid.ObjectID:
		if t == nil {
			return vw.WriteNull()
		}
		oid = *t
	default:
		return ValueEncoderError{
			Name:     "ObjectIDEncodeValue",
			Types:    []interface{}{objectid.ObjectID{}, (*objectid.ObjectID)(nil)},
			Received: i,
		}
	}

	return vw.WriteObjectID(oid)
}

// Decimal128EncodeValue is the ValueEncoderFunc for decimal.Decimal128.
func (dve DefaultValueEncoders) Decimal128EncodeValue(ec EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var d128 decimal.Decimal128
	switch t := i.(type) {
	case decimal.Decimal128:
		d128 = t
	case *decimal.Decimal128:
		if t == nil {
			return vw.WriteNull()
		}
		d128 = *t
	default:
		return ValueEncoderError{
			Name:     "Decimal128EncodeValue",
			Types:    []interface{}{decimal.Decimal128{}, (*decimal.Decimal128)(nil)},
			Received: i,
		}
	}

	return vw.WriteDecimal128(d128)
}

// JSONNumberEncodeValue is the ValueEncoderFunc for json.Number.
func (dve DefaultValueEncoders) JSONNumberEncodeValue(ec EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var jsnum json.Number
	switch t := i.(type) {
	case json.Number:
		jsnum = t
	case *json.Number:
		if t == nil {
			return vw.WriteNull()
		}
		jsnum = *t
	default:
		return ValueEncoderError{
			Name:     "JSONNumberEncodeValue",
			Types:    []interface{}{json.Number(""), (*json.Number)(nil)},
			Received: i,
		}
	}

	// Attempt int first, then float64
	if i64, err := jsnum.Int64(); err == nil {
		return dve.IntEncodeValue(ec, vw, i64)
	}

	f64, err := jsnum.Float64()
	if err != nil {
		return err
	}

	return dve.FloatEncodeValue(ec, vw, f64)
}

// URLEncodeValue is the ValueEncoderFunc for url.URL.
func (dve DefaultValueEncoders) URLEncodeValue(ec EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var u *url.URL
	switch t := i.(type) {
	case url.URL:
		u = &t
	case *url.URL:
		if t == nil {
			return vw.WriteNull()
		}
		u = t
	default:
		return ValueEncoderError{
			Name:     "URLEncodeValue",
			Types:    []interface{}{url.URL{}, (*url.URL)(nil)},
			Received: i,
		}
	}

	return vw.WriteString(u.String())
}

// TimeEncodeValue is the ValueEncoderFunc for time.TIme.
func (dve DefaultValueEncoders) TimeEncodeValue(ec EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var tt time.Time
	switch t := i.(type) {
	case time.Time:
		tt = t
	case *time.Time:
		if t == nil {
			return vw.WriteNull()
		}
		tt = *t
	default:
		return ValueEncoderError{
			Name:     "TimeEncodeValue",
			Types:    []interface{}{time.Time{}, (*time.Time)(nil)},
			Received: i,
		}
	}

	return vw.WriteDateTime(tt.Unix()*1000 + int64(tt.Nanosecond()/1e6))
}

// ByteSliceEncodeValue is the ValueEncoderFunc for []byte.
func (dve DefaultValueEncoders) ByteSliceEncodeValue(ec EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var slcb []byte
	switch t := i.(type) {
	case []byte:
		slcb = t
	case *[]byte:
		if t == nil {
			return vw.WriteNull()
		}
		slcb = *t
	default:
		return ValueEncoderError{
			Name:     "ByteSliceEncodeValue",
			Types:    []interface{}{[]byte{}, (*[]byte)(nil)},
			Received: i,
		}
	}

	return vw.WriteBinary(slcb)
}

// MapEncodeValue is the ValueEncoderFunc for map[string]* types.
func (dve DefaultValueEncoders) MapEncodeValue(ec EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	val := reflect.ValueOf(i)
	if val.Kind() != reflect.Map || val.Type().Key().Kind() != reflect.String {
		return errors.New("MapEncodeValue can only encode maps with string keys")
	}

	dw, err := vw.WriteDocument()
	if err != nil {
		return err
	}

	return dve.mapEncodeValue(ec, dw, val, nil)
}

// mapEncodeValue handles encoding of the values of a map. The collisionFn returns
// true if the provided key exists, this is mainly used for inline maps in the
// struct codec.
func (dve DefaultValueEncoders) mapEncodeValue(ec EncodeContext, dw bsonrw.DocumentWriter, val reflect.Value, collisionFn func(string) bool) error {

	encoder, err := ec.LookupEncoder(val.Type().Elem())
	if err != nil {
		return err
	}

	keys := val.MapKeys()
	for _, key := range keys {
		if collisionFn != nil && collisionFn(key.String()) {
			return fmt.Errorf("Key %s of inlined map conflicts with a struct field name", key)
		}
		vw, err := dw.WriteDocumentElement(key.String())
		if err != nil {
			return err
		}

		err = encoder.EncodeValue(ec, vw, val.MapIndex(key).Interface())
		if err != nil {
			return err
		}
	}

	return dw.WriteDocumentEnd()
}

// SliceEncodeValue is the ValueEncoderFunc for []* types.
func (dve DefaultValueEncoders) SliceEncodeValue(ec EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	val := reflect.ValueOf(i)
	switch val.Kind() {
	case reflect.Array:
	case reflect.Slice:
		if val.IsNil() { // When nil, special case to null
			return vw.WriteNull()
		}
	default:
		return errors.New("SliceEncodeValue can only encode arrays and slices")
	}

	length := val.Len()

	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	// We do this outside of the loop because an array or a slice can only have
	// one element type. If it's the empty interface, we'll use the empty
	// interface codec.
	var encoder ValueEncoder
	switch val.Type().Elem() {
	// case tElement:
	// 	encoder = ValueEncoderFunc(dve.elementEncodeValue)
	default:
		encoder, err = ec.LookupEncoder(val.Type().Elem())
		if err != nil {
			return err
		}
	}
	for idx := 0; idx < length; idx++ {
		vw, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		err = encoder.EncodeValue(ec, vw, val.Index(idx).Interface())
		if err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

// EmptyInterfaceEncodeValue is the ValueEncoderFunc for interface{}.
func (dve DefaultValueEncoders) EmptyInterfaceEncodeValue(ec EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	encoder, err := ec.LookupEncoder(reflect.TypeOf(i))
	if err != nil {
		return err
	}

	return encoder.EncodeValue(ec, vw, i)
}

// ValueMarshalerEncodeValue is the ValueEncoderFunc for ValueMarshaler implementations.
func (dve DefaultValueEncoders) ValueMarshalerEncodeValue(ec EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	vm, ok := i.(ValueMarshaler)
	if !ok {
		return ValueEncoderError{
			Name:     "ValueMarshalerEncodeValue",
			Types:    []interface{}{(ValueMarshaler)(nil)},
			Received: i,
		}
	}

	t, val, err := vm.MarshalBSONValue()
	if err != nil {
		return err
	}
	return bsonrw.Copier{}.CopyValueFromBytes(vw, t, val)
}

// ProxyEncodeValue is the ValueEncoderFunc for Proxy implementations.
func (dve DefaultValueEncoders) ProxyEncodeValue(ec EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	proxy, ok := i.(Proxy)
	if !ok {
		return ValueEncoderError{
			Name:     "ProxyEncodeValue",
			Types:    []interface{}{(Proxy)(nil)},
			Received: i,
		}
	}

	val, err := proxy.ProxyBSON()
	if err != nil {
		return err
	}
	encoder, err := ec.LookupEncoder(reflect.TypeOf(val))
	if err != nil {
		return err
	}
	return encoder.EncodeValue(ec, vw, val)
}
