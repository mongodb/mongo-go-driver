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
	"strings"
	"time"

	"github.com/mongodb/mongo-go-driver/bson/bsonrw"
	"github.com/mongodb/mongo-go-driver/bson/bsontype"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
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
		RegisterDecoder(reflect.PtrTo(tByteSlice), ValueDecoderFunc(dvd.ByteSliceDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tTime), ValueDecoderFunc(dvd.TimeDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tEmpty), ValueDecoderFunc(dvd.EmptyInterfaceDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tOID), ValueDecoderFunc(dvd.ObjectIDDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tDecimal), ValueDecoderFunc(dvd.Decimal128DecodeValue)).
		RegisterDecoder(reflect.PtrTo(tJSONNumber), ValueDecoderFunc(dvd.JSONNumberDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tURL), ValueDecoderFunc(dvd.URLDecodeValue)).
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
		RegisterDefaultDecoder(reflect.Array, ValueDecoderFunc(dvd.SliceDecodeValue)).
		RegisterDefaultDecoder(reflect.Map, ValueDecoderFunc(dvd.MapDecodeValue)).
		RegisterDefaultDecoder(reflect.Slice, ValueDecoderFunc(dvd.SliceDecodeValue)).
		RegisterDefaultDecoder(reflect.String, ValueDecoderFunc(dvd.StringDecodeValue)).
		RegisterDefaultDecoder(reflect.Struct, &StructCodec{cache: make(map[reflect.Type]*structDescription), parser: DefaultStructTagParser})
}

// getValue check if value is a pointer and return final value to
func getValue(val reflect.Value) reflect.Value {
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			val.Set(reflect.New(val.Type().Elem()))
		}
		return val.Elem()
	}
	return val
}

// getValueKind get value's kind or it's element's kind
func getValueKind(val reflect.Value) reflect.Kind {
	if val.Kind() == reflect.Ptr {
		return val.Type().Elem().Kind()
	}
	return val.Kind()
}

// checkTypeError check ValueReader is one of required bsontypes, an error is returned otherwise
func checkTypeError(vr bsonrw.ValueReader, requiredType ...bsontype.Type) error {
	currentType := vr.Type()
	for _, t := range requiredType {
		if currentType == t {
			return nil
		}
	}
	typeString := []string{}
	for _, t := range requiredType {
		var payload string
		switch t {
		case bsontype.Decimal128:
			payload = "decimal.Decimal128"
		default:
			payload = t.String()
		}
		typeString = append(typeString, payload)
	}
	return fmt.Errorf("cannot decode %v into a %s", vr.Type(), strings.Join(typeString, ", "))
}

// BooleanDecodeValue is the ValueDecoderFunc for bool types.
func (dvd DefaultValueDecoders) BooleanDecodeValue(dctx DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	var err error
	var b bool

	if i == nil {
		return ValueDecoderError{Name: "BooleanDecodeValue", Types: []interface{}{new(bool), new(*bool)}, Received: i, ReceivedNil: true}
	}

	switch target := i.(type) {
	case *bool:
		if target == nil { // We need to handle typed nil
			return ValueDecoderError{Name: "BooleanDecodeValue", Types: []interface{}{new(bool), new(*bool)}, Received: i, ReceivedNil: true}
		}
		if vr.Type() != bsontype.Boolean {
			return fmt.Errorf("cannot decode %v into a bool type", vr.Type())
		}
		*target, err = vr.ReadBoolean()
		return err
	case **bool:
		if target == nil { // We need to handle typed nil
			return ValueDecoderError{Name: "BooleanDecodeValue", Types: []interface{}{new(bool), new(*bool)}, Received: target, ReceivedNil: true}
		}
		switch vr.Type() {
		case bsontype.Boolean:
			if *target == nil {
				*target = new(bool)
			}
			**target, err = vr.ReadBoolean()
			return err
		case bsontype.Null:
			err = vr.ReadNull()
			*target = nil
			return err
		default:
			return fmt.Errorf("cannot decode %v into a bool type", vr.Type())
		}
	}

	val := reflect.ValueOf(i)
	if !val.IsValid() || val.Kind() != reflect.Ptr || !val.Elem().CanSet() {
		return ValueDecoderError{Name: "BooleanDecodeValue", Types: []interface{}{new(bool), new(*bool)}, Received: i, ReceivedNil: !val.IsValid()}
	}
	val = val.Elem()

	switch {
	case val.Kind() == reflect.Bool:
		if vr.Type() != bsontype.Boolean {
			return fmt.Errorf("cannot decode %v into a bool type", vr.Type())
		}
		b, err = vr.ReadBoolean()
		val.SetBool(b)
		return err
	case val.Kind() == reflect.Ptr && val.Elem().Kind() == reflect.Bool:
		switch vr.Type() {
		case bsontype.Boolean:
			val = getValue(val)
			b, err = vr.ReadBoolean()
			val.SetBool(b)
			return err
		case bsontype.Null:
			err = vr.ReadNull()
			val.Set(reflect.Zero(val.Type()))
			return err
		default:
			return fmt.Errorf("cannot decode %v into a bool type", vr.Type())
		}
	default:
		return ValueDecoderError{Name: "BooleanDecodeValue", Types: []interface{}{new(bool), new(*bool)}, Received: i}
	}
}

// IntDecodeValue is the ValueDecoderFunc for int types.
func (dvd DefaultValueDecoders) IntDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	var i64 int64
	var err error
	var isNull = false

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
	case bsontype.Null:
		err = vr.ReadNull()
		if err != nil {
			return err
		}
		isNull = true
	default:
		return fmt.Errorf("cannot decode %v into an integer type", vr.Type())
	}

	switch target := i.(type) {
	case *int8:
		if target == nil {
			return errors.New("IntDecodeValue can only be used to decode non-nil *int8")
		}
		if i64 < math.MinInt8 || i64 > math.MaxInt8 {
			return fmt.Errorf("%d overflows int8", i64)
		}
		*target = int8(i64)
		return nil
	case **int8:
		if target == nil {
			return errors.New("IntDecodeValue can only be used to decode non-nil **int8")
		}
		if isNull {
			*target = nil
		} else {
			if i64 < math.MinInt8 || i64 > math.MaxInt8 {
				return fmt.Errorf("%d overflows int8", i64)
			}
			i8 := int8(i64)
			*target = &i8
		}
		return nil
	case *int16:
		if target == nil {
			return errors.New("IntDecodeValue can only be used to decode non-nil *int16")
		}
		if i64 < math.MinInt16 || i64 > math.MaxInt16 {
			return fmt.Errorf("%d overflows int16", i64)
		}
		*target = int16(i64)
		return nil
	case **int16:
		if target == nil {
			return errors.New("IntDecodeValue can only be used to decode non-nil **int16")
		}
		if isNull {
			*target = nil
		} else {
			if i64 < math.MinInt16 || i64 > math.MaxInt16 {
				return fmt.Errorf("%d overflows int16", i64)
			}
			i16 := int16(i64)
			*target = &i16
		}
		return nil
	case *int32:
		if target == nil {
			return errors.New("IntDecodeValue can only be used to decode non-nil *int32")
		}
		if i64 < math.MinInt32 || i64 > math.MaxInt32 {
			return fmt.Errorf("%d overflows int32", i64)
		}
		*target = int32(i64)
		return nil
	case **int32:
		if target == nil {
			return errors.New("IntDecodeValue can only be used to decode non-nil **int32")
		}
		if isNull {
			*target = nil
		} else {
			if i64 < math.MinInt32 || i64 > math.MaxInt32 {
				return fmt.Errorf("%d overflows int32", i64)
			}
			i32 := int32(i64)
			*target = &i32
		}
		return nil
	case *int64:
		if target == nil {
			return errors.New("IntDecodeValue can only be used to decode non-nil *int64")
		}
		*target = int64(i64)
		return nil
	case **int64:
		if target == nil {
			return errors.New("IntDecodeValue can only be used to decode non-nil *int64")
		}
		if isNull {
			*target = nil
		} else {
			*target = &i64
		}
		return nil
	case *int:
		if target == nil {
			return errors.New("IntDecodeValue can only be used to decode non-nil *int")
		}
		if int64(int(i64)) != i64 { // Can we fit this inside of an int
			return fmt.Errorf("%d overflows int", i64)
		}
		*target = int(i64)
		return nil
	case **int:
		if target == nil {
			return errors.New("IntDecodeValue can only be used to decode non-nil **int")
		}
		if isNull {
			*target = nil
		} else {
			if int64(int(i64)) != i64 { // Can we fit this inside of an int
				return fmt.Errorf("%d overflows int", i64)
			}
			iint := int(i64)
			*target = &iint
		}
		return nil
	}

	val := reflect.ValueOf(i)
	if !val.IsValid() || val.Kind() != reflect.Ptr || !val.Elem().CanSet() {
		return fmt.Errorf("IntDecodeValue can only be used to decode settable (non-nil) values")
	}
	val = val.Elem()

	switch getValueKind(val) {
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
			Types:    []interface{}{(*int8)(nil), (*int16)(nil), (*int32)(nil), (*int64)(nil), (*int)(nil)},
			Received: i,
		}
	}
	if isNull && val.Kind() == reflect.Ptr {
		val.Set(reflect.Zero(val.Type()))
	} else {
		getValue(val).SetInt(i64)
	}
	return nil
}

// UintDecodeValue is the ValueDecoderFunc for uint types.
func (dvd DefaultValueDecoders) UintDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	var i64 int64
	var err error
	var isNull = false

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
	case bsontype.Null:
		err = vr.ReadNull()
		if err != nil {
			return err
		}
		isNull = true
	default:
		return fmt.Errorf("cannot decode %v into an integer type", vr.Type())
	}

	switch target := i.(type) {
	case *uint8:
		if target == nil {
			return errors.New("UintDecodeValue can only be used to decode non-nil *uint8")
		}
		if i64 < 0 || i64 > math.MaxUint8 {
			return fmt.Errorf("%d overflows uint8", i64)
		}
		*target = uint8(i64)
		return nil
	case **uint8:
		if target == nil {
			return errors.New("UintDecodeValue can only be used to decode non-nil **uint8")
		}
		if i64 < 0 || i64 > math.MaxUint8 {
			return fmt.Errorf("%d overflows uint8", i64)
		}
		if isNull {
			*target = nil
		} else {
			u8 := uint8(i64)
			*target = &u8
		}
		return nil
	case *uint16:
		if target == nil {
			return errors.New("UintDecodeValue can only be used to decode non-nil *uint16")
		}
		if i64 < 0 || i64 > math.MaxUint16 {
			return fmt.Errorf("%d overflows uint16", i64)
		}
		*target = uint16(i64)
		return nil
	case **uint16:
		if target == nil {
			return errors.New("UintDecodeValue can only be used to decode non-nil **uint16")
		}
		if i64 < 0 || i64 > math.MaxUint16 {
			return fmt.Errorf("%d overflows uint16", i64)
		}
		if isNull {
			*target = nil
		} else {
			u16 := uint16(i64)
			*target = &u16
		}
		return nil
	case *uint32:
		if target == nil {
			return errors.New("UintDecodeValue can only be used to decode non-nil *uint32")
		}
		if i64 < 0 || i64 > math.MaxUint32 {
			return fmt.Errorf("%d overflows uint32", i64)
		}
		*target = uint32(i64)
		return nil
	case **uint32:
		if target == nil {
			return errors.New("UintDecodeValue can only be used to decode non-nil **uint32")
		}
		if i64 < 0 || i64 > math.MaxUint32 {
			return fmt.Errorf("%d overflows uint32", i64)
		}
		if isNull {
			*target = nil
		} else {
			u32 := uint32(i64)
			*target = &u32
		}
		return nil
	case *uint64:
		if target == nil {
			return errors.New("UintDecodeValue can only be used to decode non-nil *uint64")
		}
		if i64 < 0 {
			return fmt.Errorf("%d overflows uint64", i64)
		}
		*target = uint64(i64)
		return nil
	case **uint64:
		if target == nil {
			return errors.New("UintDecodeValue can only be used to decode non-nil *uint64")
		}
		if i64 < 0 {
			return fmt.Errorf("%d overflows uint64", i64)
		}
		if isNull {
			*target = nil
		} else {
			u64 := uint64(i64)
			*target = &u64
		}
		return nil
	case *uint:
		if target == nil {
			return errors.New("UintDecodeValue can only be used to decode non-nil *uint")
		}
		if i64 < 0 || int64(uint(i64)) != i64 { // Can we fit this inside of an uint
			return fmt.Errorf("%d overflows uint", i64)
		}
		*target = uint(i64)
		return nil
	case **uint:
		if target == nil {
			return errors.New("UintDecodeValue can only be used to decode non-nil **uint")
		}
		if i64 < 0 || int64(uint(i64)) != i64 { // Can we fit this inside of an uint
			return fmt.Errorf("%d overflows uint", i64)
		}
		if isNull {
			*target = nil
		} else {
			uuint := uint(i64)
			*target = &uuint
		}
		return nil
	}

	val := reflect.ValueOf(i)
	if !val.IsValid() || val.Kind() != reflect.Ptr || !val.Elem().CanSet() {
		return errors.New("UintDecodeValue can only be used to decode settable (non-nil) values")
	}
	val = val.Elem()

	switch getValueKind(val) {
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
			Types:    []interface{}{(*uint8)(nil), (*uint16)(nil), (*uint32)(nil), (*uint64)(nil), (*uint)(nil)},
			Received: i,
		}
	}
	if isNull && val.Kind() == reflect.Ptr {
		val.Set(reflect.Zero(val.Type()))
	} else {
		getValue(val).SetUint(uint64(i64))
	}
	return nil
}

// FloatDecodeValue is the ValueDecoderFunc for float types.
func (dvd DefaultValueDecoders) FloatDecodeValue(ec DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	var f float64
	var err error
	var isNull = false

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
	case bsontype.Null:
		err = vr.ReadNull()
		if err != nil {
			return err
		}
		isNull = true
	default:
		return fmt.Errorf("cannot decode %v into a float32 or float64 type", vr.Type())
	}

	switch target := i.(type) {
	case *float32:
		if target == nil {
			return errors.New("FloatDecodeValue can only be used to decode non-nil *float32")
		}
		if !ec.Truncate && float64(float32(f)) != f {
			return errors.New("FloatDecodeValue can only convert float64 to float32 when truncation is allowed")
		}
		*target = float32(f)
		return nil
	case **float32:
		if target == nil {
			return errors.New("FloatDecodeValue can only be used to decode non-nil **float32")
		}
		if !ec.Truncate && float64(float32(f)) != f {
			return errors.New("FloatDecodeValue can only convert float64 to float32 when truncation is allowed")
		}
		if isNull {
			*target = nil
		} else {
			tt := float32(f)
			*target = &tt
		}
		return nil
	case *float64:
		if target == nil {
			return errors.New("FloatDecodeValue can only be used to decode non-nil *float64")
		}
		*target = f
		return nil
	case **float64:
		if target == nil {
			return errors.New("FloatDecodeValue can only be used to decode non-nil *float64")
		}
		if isNull {
			*target = nil
		} else {
			*target = &f
		}
		return nil
	}

	val := reflect.ValueOf(i)
	if !val.IsValid() || val.Kind() != reflect.Ptr || !val.Elem().CanSet() {
		return errors.New("FloatDecodeValue can only be used to decode settable (non-nil) values")
	}
	val = val.Elem()

	switch getValueKind(val) {
	case reflect.Float32:
		if !ec.Truncate && float64(float32(f)) != f {
			return errors.New("FloatDecodeValue can only convert float64 to float32 when truncation is allowed")
		}
	case reflect.Float64:
	default:
		return ValueDecoderError{Name: "FloatDecodeValue", Types: []interface{}{(*float32)(nil), (*float64)(nil)}, Received: i}
	}

	if val.Kind() == reflect.Ptr && isNull {
		val.Set(reflect.Zero(val.Type()))
	} else {
		getValue(val).SetFloat(f)
	}

	return nil
}

// StringDecodeValue is the ValueDecoderFunc for string types.
func (dvd DefaultValueDecoders) StringDecodeValue(dctx DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	var str string
	var err error
	var isNull = false

	switch vr.Type() {
	// TODO(GODRIVER-577): Handle JavaScript and Symbol BSON types when allowed.
	case bsontype.String:
		str, err = vr.ReadString()
		if err != nil {
			return err
		}
	case bsontype.Null:
		err = vr.ReadNull()
		if err != nil {
			return err
		}
		isNull = true
	default:
		return fmt.Errorf("cannot decode %v into a string type", vr.Type())
	}

	switch t := i.(type) {
	case *string:
		if t == nil {
			return errors.New("StringDecodeValue can only be used to decode non-nil *string")
		}
		*t = str
		return nil
	case **string:
		if t == nil {
			return errors.New("StringDecodeValue can only be used to decode non-nil **string")
		}
		if isNull {
			*t = nil
		} else {
			tt := &str
			*t = tt
		}
		return nil
	}

	val := reflect.ValueOf(i)
	if !val.IsValid() || val.Kind() != reflect.Ptr || !val.Elem().CanSet() {
		return errors.New("StringDecodeValue can only be used to decode settable (non-nil) values")
	}
	val = val.Elem()

	if getValueKind(val) != reflect.String {
		return ValueDecoderError{
			Name:     "StringDecodeValue",
			Types:    []interface{}{(*string)(nil), (**string)(nil)},
			Received: i,
		}
	}
	if val.Kind() == reflect.Ptr && isNull {
		val.Set(reflect.Zero(val.Type()))
	} else {
		getValue(val).SetString(str)
	}
	return nil
}

// ObjectIDDecodeValue is the ValueDecoderFunc for objectid.ObjectID.
func (dvd DefaultValueDecoders) ObjectIDDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	var isNull = false
	var oid objectid.ObjectID
	var err error
	var set func()

	switch target := i.(type) {
	case *objectid.ObjectID:
		if target == nil {
			return ValueDecoderError{Name: "ObjectIDDecodeValue", Types: []interface{}{(*objectid.ObjectID)(nil)}, Received: i}
		}
		set = func() {
			*target = oid
		}
	case **objectid.ObjectID:
		if target == nil {
			return ValueDecoderError{Name: "ObjectIDDecodeValue", Types: []interface{}{(**objectid.ObjectID)(nil)}, Received: i}
		}
		set = func() {
			if isNull {
				*target = nil
			} else {
				*target = &oid
			}
		}
	default:
		return ValueDecoderError{Name: "ObjectIDDecodeValue", Types: []interface{}{(*objectid.ObjectID)(nil)}, Received: i}
	}

	switch vr.Type() {
	case bsontype.ObjectID:
		oid, err = vr.ReadObjectID()
		if err != nil {
			return err
		}
	case bsontype.Null:
		err = vr.ReadNull()
		if err != nil {
			return err
		}
		isNull = true
	default:
		return checkTypeError(vr, bsontype.ObjectID, bsontype.Null)
	}
	set()
	return err
}

// Decimal128DecodeValue is the ValueDecoderFunc for decimal.Decimal128.
func (dvd DefaultValueDecoders) Decimal128DecodeValue(dctx DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	var d128 decimal.Decimal128
	var err error
	var isNull = false
	var set func()

	if i == nil {
		return ValueDecoderError{Name: "Decimal128DecodeValue", Types: []interface{}{(*decimal.Decimal128)(nil), (**decimal.Decimal128)(nil)}, Received: i}
	}
	switch target := i.(type) {
	case *decimal.Decimal128:
		set = func() {
			*target = d128
		}
	case **decimal.Decimal128:
		set = func() {
			if isNull {
				*target = nil
			} else {
				*target = &d128
			}
		}
	default:
		return ValueDecoderError{Name: "Decimal128DecodeValue", Types: []interface{}{(*decimal.Decimal128)(nil), (**decimal.Decimal128)(nil)}, Received: i}
	}

	switch vr.Type() {
	case bsontype.Decimal128:
		d128, err = vr.ReadDecimal128()
		if err != nil {
			return err
		}
	case bsontype.Null:
		err = vr.ReadNull()
		if err != nil {
			return err
		}
		isNull = true
	default:
		return checkTypeError(vr, bsontype.Null, bsontype.Decimal128)
	}
	set()
	return nil
}

// JSONNumberDecodeValue is the ValueDecoderFunc for json.Number.
func (dvd DefaultValueDecoders) JSONNumberDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	var decodeError = ValueDecoderError{Name: "JSONNumberDecodeValue", Types: []interface{}{(*json.Number)(nil), (**json.Number)(nil)}, Received: i}
	if i == nil {
		return decodeError
	}

	switch vr.Type() {
	case bsontype.Double:
		f64, err := vr.ReadDouble()
		if err != nil {
			return err
		}
		switch target := i.(type) {
		case *json.Number:
			*target = json.Number(strconv.FormatFloat(f64, 'g', -1, 64))
		case **json.Number:
			targetNumber := json.Number(strconv.FormatFloat(f64, 'g', -1, 64))
			*target = &targetNumber
		default:
			return decodeError
		}
	case bsontype.Int32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		switch target := i.(type) {
		case *json.Number:
			*target = json.Number(strconv.FormatInt(int64(i32), 10))
		case **json.Number:
			targetNumber := json.Number(strconv.FormatInt(int64(i32), 10))
			*target = &targetNumber
		default:
			return decodeError
		}
	case bsontype.Int64:
		i64, err := vr.ReadInt64()
		if err != nil {
			return err
		}
		switch target := i.(type) {
		case *json.Number:
			*target = json.Number(strconv.FormatInt(i64, 10))
		case **json.Number:
			targetNumber := json.Number(strconv.FormatInt(i64, 10))
			*target = &targetNumber
		default:
			return decodeError
		}
	case bsontype.Null:
		if target, ok := i.(**json.Number); ok {
			err := vr.ReadNull()
			if err != nil {
				return err
			}
			*target = nil
		}
	default:
		switch i.(type) {
		case **json.Number, *json.Number:
			return fmt.Errorf("cannot decode %v into a json.Number", vr.Type())
		default:
			return decodeError
		}
	}

	return nil
}

// URLDecodeValue is the ValueDecoderFunc for url.URL.
func (dvd DefaultValueDecoders) URLDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
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

	err = ValueDecoderError{Name: "URLDecodeValue", Types: []interface{}{(*url.URL)(nil), (**url.URL)(nil)}, Received: i}

	// It's valid to use either a *url.URL or a url.URL
	switch target := i.(type) {
	case *url.URL:
		if target == nil {
			return err
		}
		*target = *u
	case **url.URL:
		if target == nil {
			return err
		}
		*target = u
	default:
		return err
	}
	return nil
}

// TimeDecodeValue is the ValueDecoderFunc for time.Time.
func (dvd DefaultValueDecoders) TimeDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.DateTime {
		return fmt.Errorf("cannot decode %v into a time.Time", vr.Type())
	}

	dt, err := vr.ReadDateTime()
	if err != nil {
		return err
	}

	if target, ok := i.(*time.Time); ok && target != nil {
		*target = time.Unix(dt/1000, dt%1000*1000000)
		return nil
	}

	if target, ok := i.(**time.Time); ok && target != nil {
		tt := *target
		if tt == nil {
			tt = new(time.Time)
		}
		*tt = time.Unix(dt/1000, dt%1000*1000000)
		*target = tt
		return nil
	}

	return ValueDecoderError{
		Name:     "TimeDecodeValue",
		Types:    []interface{}{(*time.Time)(nil), (**time.Time)(nil)},
		Received: i,
	}
}

// ByteSliceDecodeValue is the ValueDecoderFunc for []byte.
func (dvd DefaultValueDecoders) ByteSliceDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.Binary {
		return fmt.Errorf("cannot decode %v into a *[]byte", vr.Type())
	}

	target, ok := i.(*[]byte)
	if !ok || target == nil {
		return ValueDecoderError{Name: "ByteSliceDecodeValue", Types: []interface{}{(*[]byte)(nil)}, Received: i}
	}

	data, subtype, err := vr.ReadBinary()
	if err != nil {
		return err
	}
	if subtype != 0x00 {
		return fmt.Errorf("ByteSliceDecodeValue can only be used to decode subtype 0x00 for %s, got %v", bsontype.Binary, subtype)
	}

	*target = data
	return nil
}

// MapDecodeValue is the ValueDecoderFunc for map[string]* types.
func (dvd DefaultValueDecoders) MapDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	val := reflect.ValueOf(i)
	if !val.IsValid() || val.Kind() != reflect.Ptr || val.IsNil() {
		return fmt.Errorf("MapDecodeValue can only be used to decode non-nil pointers to map values, got %T", i)
	}

	if val.Elem().Kind() != reflect.Map || val.Elem().Type().Key().Kind() != reflect.String || !val.Elem().CanSet() {
		return errors.New("MapDecodeValue can only decode settable maps with string keys")
	}

	dr, err := vr.ReadDocument()
	if err != nil {
		return err
	}

	if val.Elem().IsNil() {
		val.Elem().Set(reflect.MakeMap(val.Elem().Type()))
	}

	mVal := val.Elem()

	eType := mVal.Type().Elem()
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

		err = decoder.DecodeValue(dc, vr, ptr.Interface())
		if err != nil {
			return err
		}

		mVal.SetMapIndex(reflect.ValueOf(key), ptr.Elem())
	}
	return err
}

// SliceDecodeValue is the ValueDecoderFunc for []* types.
func (dvd DefaultValueDecoders) SliceDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	val := reflect.ValueOf(i)
	if !val.IsValid() || val.Kind() != reflect.Ptr || val.IsNil() {
		return fmt.Errorf("SliceDecodeValue can only be used to decode non-nil pointers to slice or array values, got %T", i)
	}

	switch val.Elem().Kind() {
	case reflect.Slice, reflect.Array:
		if !val.Elem().CanSet() {
			return errors.New("SliceDecodeValue can only decode settable slice and array values")
		}
	default:
		return fmt.Errorf("SliceDecodeValue can only decode settable slice and array values, got %T", i)
	}

	switch vr.Type() {
	case bsontype.Array:
	case bsontype.Null:
		if val.Elem().Kind() != reflect.Slice {
			return fmt.Errorf("cannot decode %v into an array", vr.Type())
		}
		null := reflect.Zero(val.Elem().Type())
		val.Elem().Set(null)
		return vr.ReadNull()
	default:
		return fmt.Errorf("cannot decode %v into a slice", vr.Type())
	}

	eType := val.Type().Elem().Elem()

	ar, err := vr.ReadArray()
	if err != nil {
		return err
	}

	elems, err := dvd.decodeDefault(dc, ar, eType)
	if err != nil {
		return err
	}

	switch val.Elem().Kind() {
	case reflect.Slice:
		slc := reflect.MakeSlice(val.Elem().Type(), len(elems), len(elems))

		for idx, elem := range elems {
			slc.Index(idx).Set(elem)
		}

		val.Elem().Set(slc)
	case reflect.Array:
		if len(elems) > val.Elem().Len() {
			return fmt.Errorf("more elements returned in array than can fit inside %s", val.Elem().Type())
		}

		for idx, elem := range elems {
			val.Elem().Index(idx).Set(elem)
		}
	}

	return nil
}

// ValueUnmarshalerDecodeValue is the ValueDecoderFunc for ValueUnmarshaler implementations.
func (dvd DefaultValueDecoders) ValueUnmarshalerDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	val := reflect.ValueOf(i)
	var valueUnmarshaler ValueUnmarshaler
	if val.Kind() == reflect.Ptr && val.IsNil() {
		return fmt.Errorf("ValueUnmarshalerDecodeValue can only unmarshal into non-nil ValueUnmarshaler values, got %T", i)
	}
	if val.Type().Implements(tValueUnmarshaler) {
		valueUnmarshaler = val.Interface().(ValueUnmarshaler)
	} else if val.Type().Kind() == reflect.Ptr && val.Elem().Type().Implements(tValueUnmarshaler) {
		if val.Elem().Kind() == reflect.Ptr && val.Elem().IsNil() {
			val.Elem().Set(reflect.New(val.Type().Elem().Elem()))
		}
		valueUnmarshaler = val.Elem().Interface().(ValueUnmarshaler)
	} else {
		return fmt.Errorf("ValueUnmarshalerDecodeValue can only handle types or pointers to types that are a ValueUnmarshaler, got %T", i)
	}

	t, src, err := bsonrw.Copier{}.CopyValueToBytes(vr)
	if err != nil {
		return err
	}

	return valueUnmarshaler.UnmarshalBSONValue(t, src)
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
	err = decoder.DecodeValue(dc, vr, val)
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

		err = decoder.DecodeValue(dc, vr, ptr.Interface())
		if err != nil {
			return nil, err
		}
		elems = append(elems, ptr.Elem())
	}

	return elems, nil
}
