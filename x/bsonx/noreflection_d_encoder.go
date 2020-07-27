// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsonx

import (
	"fmt"
	"math"
	"reflect"
	"time"

	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	tPrimitiveD          = reflect.TypeOf(primitive.D{})
	tPrimitiveA          = reflect.TypeOf(primitive.A{})
	defaultValueEncoders = bsoncodec.DefaultValueEncoders{}
)

type reflectionFreeDEncoder struct{}

// ReflectionFreeDEncoder is a ValueEncoder for the primitive.D type that does not use reflection.
var ReflectionFreeDEncoder bsoncodec.ValueEncoder = &reflectionFreeDEncoder{}

func (r *reflectionFreeDEncoder) EncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tPrimitiveD {
		return bsoncodec.ValueEncoderError{Name: "DEncodeValue", Types: []reflect.Type{tPrimitiveD}, Received: val}
	}

	if val.IsNil() {
		return vw.WriteNull()
	}

	doc := val.Interface().(primitive.D)
	return r.encodeDocument(ec, vw, doc)
}

func (r *reflectionFreeDEncoder) encodeDocumentValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, v interface{}) error {
	switch val := v.(type) {
	case int:
		return r.encodeInt(ec, vw, val)
	case int8:
		return vw.WriteInt32(int32(val))
	case int16:
		return vw.WriteInt32(int32(val))
	case int32:
		return vw.WriteInt32(int32(val))
	case int64:
		return r.encodeInt64(ec, vw, val)
	case uint:
		return r.encodeUint64(ec, vw, uint64(val))
	case uint8:
		return vw.WriteInt32(int32(val))
	case uint16:
		return vw.WriteInt32(int32(val))
	case uint32:
		return r.encodeUint64(ec, vw, uint64(val))
	case uint64:
		return r.encodeUint64(ec, vw, val)
	case float32:
		return vw.WriteDouble(float64(val))
	case float64:
		return vw.WriteDouble(val)
	case []byte:
		return vw.WriteBinary(val)
	case primitive.Binary:
		return vw.WriteBinaryWithSubtype(val.Data, val.Subtype)
	case bool:
		return vw.WriteBoolean(val)
	case primitive.CodeWithScope:
		return defaultValueEncoders.CodeWithScopeEncodeValue(ec, vw, reflect.ValueOf(val))
	case primitive.DBPointer:
		return vw.WriteDBPointer(val.DB, val.Pointer)
	case primitive.DateTime:
		return vw.WriteDateTime(int64(val))
	case time.Time:
		dt := primitive.NewDateTimeFromTime(val)
		return vw.WriteDateTime(int64(dt))
	case primitive.Decimal128:
		return vw.WriteDecimal128(val)
	case primitive.JavaScript:
		return vw.WriteJavascript(string(val))
	case primitive.MinKey:
		return vw.WriteMinKey()
	case primitive.MaxKey:
		return vw.WriteMaxKey()
	case primitive.Null, nil:
		return vw.WriteNull()
	case primitive.ObjectID:
		return vw.WriteObjectID(val)
	case primitive.Regex:
		return vw.WriteRegex(val.Pattern, val.Options)
	case string:
		return vw.WriteString(val)
	case primitive.Symbol:
		return vw.WriteSymbol(string(val))
	case primitive.Timestamp:
		return vw.WriteTimestamp(val.T, val.I)
	case primitive.Undefined:
		return vw.WriteUndefined()
	case primitive.D:
		return r.encodeDocument(ec, vw, val)
	case primitive.A:
		return r.encodePrimitiveA(ec, vw, val)
	case []interface{}:
		return r.encodePrimitiveA(ec, vw, val)
	case []primitive.D:
		return r.encodeSliceD(ec, vw, val)
	case []int:
		return r.encodeSliceInt(ec, vw, val)
	case []int8:
		return r.encodeSliceInt8(ec, vw, val)
	case []int16:
		return r.encodeSliceInt16(ec, vw, val)
	case []int32:
		return r.encodeSliceInt32(ec, vw, val)
	case []int64:
		return r.encodeSliceInt64(ec, vw, val)
	case []uint:
		return r.encodeSliceUint(ec, vw, val)
	case []uint16:
		return r.encodeSliceUint16(ec, vw, val)
	case []uint32:
		return r.encodeSliceUint32(ec, vw, val)
	case []uint64:
		return r.encodeSliceUint64(ec, vw, val)
	case [][]byte:
		return r.encodeSliceByteSlice(ec, vw, val)
	case []primitive.Binary:
		return r.encodeSliceBinary(ec, vw, val)
	case []bool:
		return r.encodeSliceBoolean(ec, vw, val)
	case []primitive.CodeWithScope:
		return r.encodeSliceCWS(ec, vw, val)
	case []primitive.DBPointer:
		return r.encodeSliceDBPointer(ec, vw, val)
	case []primitive.DateTime:
		return r.encodeSliceDateTime(ec, vw, val)
	case []time.Time:
		return r.encodeSliceTimeTime(ec, vw, val)
	case []primitive.Decimal128:
		return r.encodeSliceDecimal128(ec, vw, val)
	case []float32:
		return r.encodeSliceFloat32(ec, vw, val)
	case []float64:
		return r.encodeSliceFloat64(ec, vw, val)
	case []primitive.JavaScript:
		return r.encodeSliceJavaScript(ec, vw, val)
	case []primitive.MinKey:
		return r.encodeSliceMinKey(ec, vw, val)
	case []primitive.MaxKey:
		return r.encodeSliceMaxKey(ec, vw, val)
	case []primitive.Null:
		return r.encodeSliceNull(ec, vw, val)
	case []primitive.ObjectID:
		return r.encodeSliceObjectID(ec, vw, val)
	case []primitive.Regex:
		return r.encodeSliceRegex(ec, vw, val)
	case []string:
		return r.encodeSliceString(ec, vw, val)
	case []primitive.Symbol:
		return r.encodeSliceSymbol(ec, vw, val)
	case []primitive.Timestamp:
		return r.encodeSliceTimestamp(ec, vw, val)
	case []primitive.Undefined:
		return r.encodeSliceUndefined(ec, vw, val)
	default:
		return fmt.Errorf("value of type %T not supported", v)
	}
}

func (r *reflectionFreeDEncoder) encodeInt(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val int) error {
	if fitsIn32Bits(int64(val)) {
		return vw.WriteInt32(int32(val))
	}
	return vw.WriteInt64(int64(val))
}

func (r *reflectionFreeDEncoder) encodeInt64(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val int64) error {
	if ec.MinSize && fitsIn32Bits(val) {
		return vw.WriteInt32(int32(val))
	}
	return vw.WriteInt64(int64(val))
}

func (r *reflectionFreeDEncoder) encodeUint64(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val uint64) error {
	if ec.MinSize && val <= math.MaxInt32 {
		return vw.WriteInt32(int32(val))
	}
	if val > math.MaxInt64 {
		return fmt.Errorf("%d overflows int64", val)
	}

	return vw.WriteInt64(int64(val))
}

func (r *reflectionFreeDEncoder) encodeDocument(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, doc primitive.D) error {
	dw, err := vw.WriteDocument()
	if err != nil {
		return err
	}

	for _, elem := range doc {
		docValWriter, err := dw.WriteDocumentElement(elem.Key)
		if err != nil {
			return err
		}

		if err := r.encodeDocumentValue(ec, docValWriter, elem.Value); err != nil {
			return err
		}
	}

	return dw.WriteDocumentEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceByteSlice(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr [][]byte) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := arrayValWriter.WriteBinary(val); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceBinary(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []primitive.Binary) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := arrayValWriter.WriteBinaryWithSubtype(val.Data, val.Subtype); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceBoolean(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []bool) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := arrayValWriter.WriteBoolean(val); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceCWS(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []primitive.CodeWithScope) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := defaultValueEncoders.CodeWithScopeEncodeValue(ec, arrayValWriter, reflect.ValueOf(val)); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceDBPointer(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []primitive.DBPointer) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := arrayValWriter.WriteDBPointer(val.DB, val.Pointer); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceDateTime(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []primitive.DateTime) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := arrayValWriter.WriteDateTime(int64(val)); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceTimeTime(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []time.Time) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		dt := primitive.NewDateTimeFromTime(val)
		if err := arrayValWriter.WriteDateTime(int64(dt)); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceDecimal128(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []primitive.Decimal128) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := arrayValWriter.WriteDecimal128(val); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceFloat32(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []float32) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := arrayValWriter.WriteDouble(float64(val)); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceFloat64(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []float64) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := arrayValWriter.WriteDouble(val); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceJavaScript(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []primitive.JavaScript) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := arrayValWriter.WriteJavascript(string(val)); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceMinKey(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []primitive.MinKey) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := arrayValWriter.WriteMinKey(); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceMaxKey(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []primitive.MaxKey) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := arrayValWriter.WriteMaxKey(); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceNull(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []primitive.Null) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := arrayValWriter.WriteNull(); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceObjectID(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []primitive.ObjectID) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := arrayValWriter.WriteObjectID(val); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceRegex(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []primitive.Regex) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := arrayValWriter.WriteRegex(val.Pattern, val.Options); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceString(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []string) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := arrayValWriter.WriteString(val); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceSymbol(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []primitive.Symbol) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := arrayValWriter.WriteSymbol(string(val)); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceTimestamp(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []primitive.Timestamp) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := arrayValWriter.WriteTimestamp(val.T, val.I); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceUndefined(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []primitive.Undefined) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := arrayValWriter.WriteUndefined(); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodePrimitiveA(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr primitive.A) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := r.encodeDocumentValue(ec, arrayValWriter, val); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceD(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []primitive.D) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := r.encodeDocument(ec, arrayValWriter, val); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceInt(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []int) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := r.encodeInt(ec, arrayValWriter, val); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceInt8(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []int8) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := arrayValWriter.WriteInt32(int32(val)); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceInt16(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []int16) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := arrayValWriter.WriteInt32(int32(val)); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceInt32(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []int32) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := arrayValWriter.WriteInt32(val); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceInt64(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []int64) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := r.encodeInt64(ec, arrayValWriter, val); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceUint(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []uint) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := r.encodeUint64(ec, arrayValWriter, uint64(val)); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceUint16(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []uint16) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := arrayValWriter.WriteInt32(int32(val)); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceUint32(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []uint32) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := r.encodeUint64(ec, arrayValWriter, uint64(val)); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func (r *reflectionFreeDEncoder) encodeSliceUint64(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, arr []uint64) error {
	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		arrayValWriter, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		if err := r.encodeUint64(ec, arrayValWriter, val); err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func fitsIn32Bits(i int64) bool {
	return math.MinInt32 <= i && i <= math.MaxInt32
}
