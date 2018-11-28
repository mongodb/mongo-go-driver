// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"github.com/mongodb/mongo-go-driver/bson/bsonrw"
	"github.com/mongodb/mongo-go-driver/bson/bsonrw/bsonrwtest"
	"github.com/mongodb/mongo-go-driver/bson/bsontype"
	"github.com/mongodb/mongo-go-driver/bson/primitive"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
	"github.com/mongodb/mongo-go-driver/x/bsonx/bsoncore"
)

func bytesFromDoc(doc bsonx.Doc) []byte {
	b, err := doc.MarshalBSON()
	if err != nil {
		panic(fmt.Errorf("Couldn't marshal BSON document: %v", err))
	}
	return b
}

func compareValues(v1, v2 bsonx.Val) bool    { return v1.Equal(v2) }
func compareElements(e1, e2 bsonx.Elem) bool { return e1.Equal(e2) }

func compareDecimal128(d1, d2 primitive.Decimal128) bool {
	d1H, d1L := d1.GetBytes()
	d2H, d2L := d2.GetBytes()

	if d1H != d2H {
		return false
	}

	if d1L != d2L {
		return false
	}

	return true
}

func compareErrors(err1, err2 error) bool {
	if err1 == nil && err2 == nil {
		return true
	}

	if err1 == nil || err2 == nil {
		return false
	}

	if err1.Error() != err2.Error() {
		return false
	}

	return true
}

func TestDefaultValueEncoders(t *testing.T) {
	var pc PrimitiveCodecs
	var pcx bsonx.PrimitiveCodecs

	var pjs = new(primitive.JavaScript)
	*pjs = primitive.JavaScript("var hello = 'world';")
	var psymbol = new(primitive.Symbol)
	*psymbol = primitive.Symbol("foobarbaz")

	var wrong = func(string, string) string { return "wrong" }

	pdatetime := new(primitive.DateTime)
	*pdatetime = primitive.DateTime(1234567890)

	type subtest struct {
		name   string
		val    interface{}
		ectx   *bsoncodec.EncodeContext
		llvrw  *bsonrwtest.ValueReaderWriter
		invoke bsonrwtest.Invoked
		err    error
	}

	testCases := []struct {
		name     string
		ve       bsoncodec.ValueEncoder
		subtests []subtest
	}{
		{
			"RawValueEncodeValue",
			bsoncodec.ValueEncoderFunc(pc.RawValueEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueEncoderError{
						Name:     "RawValueEncodeValue",
						Types:    []reflect.Type{tRawValue},
						Received: reflect.ValueOf(wrong),
					},
				},
				{
					"RawValue/success",
					RawValue{Type: bsontype.Double, Value: bsoncore.AppendDouble(nil, 3.14159)},
					nil,
					nil,
					bsonrwtest.WriteDouble,
					nil,
				},
			},
		},
		{
			"ValueEncodeValue",
			bsoncodec.ValueEncoderFunc(pcx.ValueEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueEncoderError{Name: "ValueEncodeValue", Types: []reflect.Type{tValue}, Received: reflect.ValueOf(wrong)},
				},
				{"empty value", bsonx.Val{}, nil, nil, bsonrwtest.WriteNull, nil},
				{
					"success",
					bsonx.Null(),
					&bsoncodec.EncodeContext{Registry: DefaultRegistry},
					&bsonrwtest.ValueReaderWriter{},
					bsonrwtest.WriteNull,
					nil,
				},
			},
		},
		{
			"RawEncodeValue",
			bsoncodec.ValueEncoderFunc(pc.RawEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueEncoderError{Name: "RawEncodeValue", Types: []reflect.Type{tRaw}, Received: reflect.ValueOf(wrong)},
				},
				{
					"WriteDocument Error",
					Raw{},
					nil,
					&bsonrwtest.ValueReaderWriter{Err: errors.New("wd error"), ErrAfter: bsonrwtest.WriteDocument},
					bsonrwtest.WriteDocument,
					errors.New("wd error"),
				},
				{
					"Raw.Elements Error",
					Raw{0xFF, 0x00, 0x00, 0x00, 0x00},
					nil,
					&bsonrwtest.ValueReaderWriter{},
					bsonrwtest.WriteDocument,
					errors.New("length read exceeds number of bytes available. length=5 bytes=255"),
				},
				{
					"WriteDocumentElement Error",
					Raw(bytesFromDoc(bsonx.Doc{{"foo", bsonx.Null()}})),
					nil,
					&bsonrwtest.ValueReaderWriter{Err: errors.New("wde error"), ErrAfter: bsonrwtest.WriteDocumentElement},
					bsonrwtest.WriteDocumentElement,
					errors.New("wde error"),
				},
				{
					"encodeValue error",
					Raw(bytesFromDoc(bsonx.Doc{{"foo", bsonx.Null()}})),
					nil,
					&bsonrwtest.ValueReaderWriter{Err: errors.New("ev error"), ErrAfter: bsonrwtest.WriteNull},
					bsonrwtest.WriteNull,
					errors.New("ev error"),
				},
				{
					"iterator error",
					Raw{0x0C, 0x00, 0x00, 0x00, 0x01, 'f', 'o', 'o', 0x00, 0x01, 0x02, 0x03},
					nil,
					&bsonrwtest.ValueReaderWriter{},
					bsonrwtest.WriteDocumentElement,
					errors.New("not enough bytes available to read type. bytes=3 type=double"),
				},
			},
		},
		{
			"ElementSliceEncodeValue",
			bsoncodec.ValueEncoderFunc(pcx.ElementSliceEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueEncoderError{
						Name:     "ElementSliceEncodeValue",
						Types:    []reflect.Type{tElementSlice},
						Received: reflect.ValueOf(wrong),
					},
				},
			},
		},
		{
			"ArrayEncodeValue",
			bsoncodec.ValueEncoderFunc(pcx.ArrayEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueEncoderError{Name: "ArrayEncodeValue", Types: []reflect.Type{tArray}, Received: reflect.ValueOf(wrong)},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, subtest := range tc.subtests {
				t.Run(subtest.name, func(t *testing.T) {
					var ec bsoncodec.EncodeContext
					if subtest.ectx != nil {
						ec = *subtest.ectx
					}
					llvrw := new(bsonrwtest.ValueReaderWriter)
					if subtest.llvrw != nil {
						llvrw = subtest.llvrw
					}
					llvrw.T = t
					err := tc.ve.EncodeValue(ec, llvrw, reflect.ValueOf(subtest.val))
					if !compareErrors(err, subtest.err) {
						t.Errorf("Errors do not match. got %v; want %v", err, subtest.err)
					}
					invoked := llvrw.Invoked
					if !cmp.Equal(invoked, subtest.invoke) {
						t.Errorf("Incorrect method invoked. got %v; want %v", invoked, subtest.invoke)
					}
				})
			}
		})
	}

	t.Run("DocumentEncodeValue", func(t *testing.T) {
		t.Run("ValueEncoderError", func(t *testing.T) {
			val := reflect.ValueOf(bool(true))
			want := bsoncodec.ValueEncoderError{Name: "DocumentEncodeValue", Types: []reflect.Type{tDocument}, Received: val}
			got := (bsonx.PrimitiveCodecs{}).DocumentEncodeValue(bsoncodec.EncodeContext{}, nil, val)
			if !compareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
		t.Run("WriteDocument Error", func(t *testing.T) {
			want := errors.New("WriteDocument Error")
			llvrw := &bsonrwtest.ValueReaderWriter{
				T:        t,
				Err:      want,
				ErrAfter: bsonrwtest.WriteDocument,
			}
			got := (bsonx.PrimitiveCodecs{}).DocumentEncodeValue(bsoncodec.EncodeContext{}, llvrw, reflect.MakeSlice(tDocument, 0, 0))
			if !compareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
		t.Run("encodeDocument errors", func(t *testing.T) {
			ec := bsoncodec.EncodeContext{}
			err := errors.New("encodeDocument error")
			oid := primitive.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
			testCases := []struct {
				name  string
				ec    bsoncodec.EncodeContext
				llvrw *bsonrwtest.ValueReaderWriter
				doc   bsonx.Doc
				err   error
			}{
				{
					"WriteDocumentElement",
					ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: errors.New("wde error"), ErrAfter: bsonrwtest.WriteDocumentElement},
					bsonx.Doc{{"foo", bsonx.Null()}},
					errors.New("wde error"),
				},
				{
					"WriteDouble", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteDouble},
					bsonx.Doc{{"foo", bsonx.Double(3.14159)}}, err,
				},
				{
					"WriteString", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteString},
					bsonx.Doc{{"foo", bsonx.String("bar")}}, err,
				},
				{
					"WriteDocument (Lookup)", bsoncodec.EncodeContext{Registry: bsoncodec.NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{T: t},
					bsonx.Doc{{"foo", bsonx.Document(bsonx.Doc{{"bar", bsonx.Null()}})}},
					bsoncodec.ErrNoEncoder{Type: tDocument},
				},
				{
					"WriteArray (Lookup)", bsoncodec.EncodeContext{Registry: bsoncodec.NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{T: t},
					bsonx.Doc{{"foo", bsonx.Array(bsonx.Arr{bsonx.Null()})}},
					bsoncodec.ErrNoEncoder{Type: tArray},
				},
				{
					"WriteBinary", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteBinaryWithSubtype},
					bsonx.Doc{{"foo", bsonx.Binary(0xFF, []byte{0x01, 0x02, 0x03})}}, err,
				},
				{
					"WriteUndefined", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteUndefined},
					bsonx.Doc{{"foo", bsonx.Undefined()}}, err,
				},
				{
					"WriteObjectID", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteObjectID},
					bsonx.Doc{{"foo", bsonx.ObjectID(oid)}}, err,
				},
				{
					"WriteBoolean", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteBoolean},
					bsonx.Doc{{"foo", bsonx.Boolean(true)}}, err,
				},
				{
					"WriteDateTime", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteDateTime},
					bsonx.Doc{{"foo", bsonx.DateTime(1234567890)}}, err,
				},
				{
					"WriteNull", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteNull},
					bsonx.Doc{{"foo", bsonx.Null()}}, err,
				},
				{
					"WriteRegex", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteRegex},
					bsonx.Doc{{"foo", bsonx.Regex("bar", "baz")}}, err,
				},
				{
					"WriteDBPointer", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteDBPointer},
					bsonx.Doc{{"foo", bsonx.DBPointer("bar", oid)}}, err,
				},
				{
					"WriteJavascript", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteJavascript},
					bsonx.Doc{{"foo", bsonx.JavaScript("var hello = 'world';")}}, err,
				},
				{
					"WriteSymbol", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteSymbol},
					bsonx.Doc{{"foo", bsonx.Symbol("symbolbaz")}}, err,
				},
				{
					"WriteCodeWithScope (Lookup)", bsoncodec.EncodeContext{Registry: bsoncodec.NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteCodeWithScope},
					bsonx.Doc{{"foo", bsonx.CodeWithScope("var hello = 'world';", bsonx.Doc{}.Append("bar", bsonx.Null()))}},
					err,
				},
				{
					"WriteInt32", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteInt32},
					bsonx.Doc{{"foo", bsonx.Int32(12345)}}, err,
				},
				{
					"WriteInt64", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteInt64},
					bsonx.Doc{{"foo", bsonx.Int64(1234567890)}}, err,
				},
				{
					"WriteTimestamp", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteTimestamp},
					bsonx.Doc{{"foo", bsonx.Timestamp(10, 20)}}, err,
				},
				{
					"WriteDecimal128", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteDecimal128},
					bsonx.Doc{{"foo", bsonx.Decimal128(primitive.NewDecimal128(10, 20))}}, err,
				},
				{
					"WriteMinKey", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteMinKey},
					bsonx.Doc{{"foo", bsonx.MinKey()}}, err,
				},
				{
					"WriteMaxKey", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteMaxKey},
					bsonx.Doc{{"foo", bsonx.MaxKey()}}, err,
				},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					err := (bsonx.PrimitiveCodecs{}).DocumentEncodeValue(tc.ec, tc.llvrw, reflect.ValueOf(tc.doc))
					if !compareErrors(err, tc.err) {
						t.Errorf("Errors do not match. got %v; want %v", err, tc.err)
					}
				})
			}
		})

		t.Run("success", func(t *testing.T) {
			oid := primitive.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
			d128 := primitive.NewDecimal128(10, 20)
			want := bsonx.Doc{
				{"a", bsonx.Double(3.14159)}, {"b", bsonx.String("foo")},
				{"c", bsonx.Document(bsonx.Doc{{"aa", bsonx.Null()}})}, {"d", bsonx.Array(bsonx.Arr{bsonx.Null()})},
				{"e", bsonx.Binary(0xFF, []byte{0x01, 0x02, 0x03})}, {"f", bsonx.Undefined()},
				{"g", bsonx.ObjectID(oid)}, {"h", bsonx.Boolean(true)},
				{"i", bsonx.DateTime(1234567890)}, {"j", bsonx.Null()},
				{"k", bsonx.Regex("foo", "abr")},
				{"l", bsonx.DBPointer("foobar", oid)}, {"m", bsonx.JavaScript("var hello = 'world';")},
				{"n", bsonx.Symbol("bazqux")},
				{"o", bsonx.CodeWithScope("var hello = 'world';", bsonx.Doc{{"ab", bsonx.Null()}})},
				{"p", bsonx.Int32(12345)},
				{"q", bsonx.Timestamp(10, 20)}, {"r", bsonx.Int64(1234567890)}, {"s", bsonx.Decimal128(d128)}, {"t", bsonx.MinKey()}, {"u", bsonx.MaxKey()},
			}
			got := bsonx.Doc{}
			slc := make(bsonrw.SliceWriter, 0, 128)
			vw, err := bsonrw.NewBSONValueWriter(&slc)
			noerr(t, err)

			ec := bsoncodec.EncodeContext{Registry: DefaultRegistry}
			err = (bsonx.PrimitiveCodecs{}).DocumentEncodeValue(ec, vw, reflect.ValueOf(want))
			noerr(t, err)
			got, err = bsonx.ReadDoc(slc)
			noerr(t, err)
			if !got.Equal(want) {
				t.Error("Documents do not match")
				t.Errorf("\ngot :%v\nwant:%v", got, want)
			}
		})
	})

	t.Run("ArrayEncodeValue", func(t *testing.T) {
		t.Run("CodecEncodeError", func(t *testing.T) {
			val := reflect.ValueOf(bool(true))
			want := bsoncodec.ValueEncoderError{Name: "ArrayEncodeValue", Types: []reflect.Type{tArray}, Received: val}
			got := (bsonx.PrimitiveCodecs{}).ArrayEncodeValue(bsoncodec.EncodeContext{}, nil, val)
			if !compareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
		t.Run("WriteArray Error", func(t *testing.T) {
			want := errors.New("WriteArray Error")
			llvrw := &bsonrwtest.ValueReaderWriter{
				T:        t,
				Err:      want,
				ErrAfter: bsonrwtest.WriteArray,
			}
			got := (bsonx.PrimitiveCodecs{}).ArrayEncodeValue(bsoncodec.EncodeContext{}, llvrw, reflect.MakeSlice(tArray, 0, 0))
			if !compareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
		t.Run("encode array errors", func(t *testing.T) {
			ec := bsoncodec.EncodeContext{}
			err := errors.New("encode array error")
			oid := primitive.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
			testCases := []struct {
				name  string
				ec    bsoncodec.EncodeContext
				llvrw *bsonrwtest.ValueReaderWriter
				arr   bsonx.Arr
				err   error
			}{
				{
					"WriteDocumentElement",
					ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: errors.New("wde error"), ErrAfter: bsonrwtest.WriteArrayElement},
					bsonx.Arr{bsonx.Null()},
					errors.New("wde error"),
				},
				{
					"WriteDouble", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteDouble},
					bsonx.Arr{bsonx.Double(3.14159)}, err,
				},
				{
					"WriteString", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteString},
					bsonx.Arr{bsonx.String("bar")}, err,
				},
				{
					"WriteDocument (Lookup)", bsoncodec.EncodeContext{Registry: bsoncodec.NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{T: t},
					bsonx.Arr{bsonx.Document(bsonx.Doc{{"bar", bsonx.Null()}})},
					bsoncodec.ErrNoEncoder{Type: tDocument},
				},
				{
					"WriteArray (Lookup)", bsoncodec.EncodeContext{Registry: bsoncodec.NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{T: t},
					bsonx.Arr{bsonx.Array(bsonx.Arr{bsonx.Null()})},
					bsoncodec.ErrNoEncoder{Type: tArray},
				},
				{
					"WriteBinary", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteBinaryWithSubtype},
					bsonx.Arr{bsonx.Binary(0xFF, []byte{0x01, 0x02, 0x03})}, err,
				},
				{
					"WriteUndefined", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteUndefined},
					bsonx.Arr{bsonx.Undefined()}, err,
				},
				{
					"WriteObjectID", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteObjectID},
					bsonx.Arr{bsonx.ObjectID(oid)}, err,
				},
				{
					"WriteBoolean", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteBoolean},
					bsonx.Arr{bsonx.Boolean(true)}, err,
				},
				{
					"WriteDateTime", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteDateTime},
					bsonx.Arr{bsonx.DateTime(1234567890)}, err,
				},
				{
					"WriteNull", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteNull},
					bsonx.Arr{bsonx.Null()}, err,
				},
				{
					"WriteRegex", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteRegex},
					bsonx.Arr{bsonx.Regex("bar", "baz")}, err,
				},
				{
					"WriteDBPointer", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteDBPointer},
					bsonx.Arr{bsonx.DBPointer("bar", oid)}, err,
				},
				{
					"WriteJavascript", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteJavascript},
					bsonx.Arr{bsonx.JavaScript("var hello = 'world';")}, err,
				},
				{
					"WriteSymbol", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteSymbol},
					bsonx.Arr{bsonx.Symbol("symbolbaz")}, err,
				},
				{
					"WriteCodeWithScope (Lookup)", bsoncodec.EncodeContext{Registry: bsoncodec.NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteCodeWithScope},
					bsonx.Arr{bsonx.CodeWithScope("var hello = 'world';", bsonx.Doc{{"bar", bsonx.Null()}})},
					err,
				},
				{
					"WriteInt32", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteInt32},
					bsonx.Arr{bsonx.Int32(12345)}, err,
				},
				{
					"WriteInt64", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteInt64},
					bsonx.Arr{bsonx.Int64(1234567890)}, err,
				},
				{
					"WriteTimestamp", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteTimestamp},
					bsonx.Arr{bsonx.Timestamp(10, 20)}, err,
				},
				{
					"WriteDecimal128", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteDecimal128},
					bsonx.Arr{bsonx.Decimal128(primitive.NewDecimal128(10, 20))}, err,
				},
				{
					"WriteMinKey", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteMinKey},
					bsonx.Arr{bsonx.MinKey()}, err,
				},
				{
					"WriteMaxKey", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteMaxKey},
					bsonx.Arr{bsonx.MaxKey()}, err,
				},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					err := (bsonx.PrimitiveCodecs{}).ArrayEncodeValue(tc.ec, tc.llvrw, reflect.ValueOf(tc.arr))
					if !compareErrors(err, tc.err) {
						t.Errorf("Errors do not match. got %v; want %v", err, tc.err)
					}
				})
			}
		})

		t.Run("success", func(t *testing.T) {
			oid := primitive.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
			d128 := primitive.NewDecimal128(10, 20)
			want := bsonx.Arr{
				bsonx.Double(3.14159), bsonx.String("foo"), bsonx.Document(bsonx.Doc{{"aa", bsonx.Null()}}),
				bsonx.Array(bsonx.Arr{bsonx.Null()}),
				bsonx.Binary(0xFF, []byte{0x01, 0x02, 0x03}), bsonx.Undefined(),
				bsonx.ObjectID(oid), bsonx.Boolean(true), bsonx.DateTime(1234567890), bsonx.Null(), bsonx.Regex("foo", "abr"),
				bsonx.DBPointer("foobar", oid), bsonx.JavaScript("var hello = 'world';"), bsonx.Symbol("bazqux"),
				bsonx.CodeWithScope("var hello = 'world';", bsonx.Doc{{"ab", bsonx.Null()}}), bsonx.Int32(12345),
				bsonx.Timestamp(10, 20), bsonx.Int64(1234567890), bsonx.Decimal128(d128), bsonx.MinKey(), bsonx.MaxKey(),
			}

			ec := bsoncodec.EncodeContext{Registry: DefaultRegistry}

			slc := make(bsonrw.SliceWriter, 0, 128)
			vw, err := bsonrw.NewBSONValueWriter(&slc)
			noerr(t, err)

			dr, err := vw.WriteDocument()
			noerr(t, err)
			vr, err := dr.WriteDocumentElement("foo")
			noerr(t, err)

			err = (bsonx.PrimitiveCodecs{}).ArrayEncodeValue(ec, vr, reflect.ValueOf(want))
			noerr(t, err)

			err = dr.WriteDocumentEnd()
			noerr(t, err)

			val, err := bsoncore.Document(slc).LookupErr("foo")
			noerr(t, err)
			rgot := val.Array()
			doc, err := bsonx.ReadDoc(rgot)
			noerr(t, err)
			got := make(bsonx.Arr, 0)
			for _, elem := range doc {
				got = append(got, elem.Value)
			}
			if !got.Equal(want) {
				t.Error("Documents do not match")
				t.Errorf("\ngot :%v\nwant:%v", got, want)
			}
		})
	})

	t.Run("success path", func(t *testing.T) {
		oid := primitive.NewObjectID()
		oids := []primitive.ObjectID{primitive.NewObjectID(), primitive.NewObjectID(), primitive.NewObjectID()}
		var str = new(string)
		*str = "bar"
		now := time.Now().Truncate(time.Millisecond)
		murl, err := url.Parse("https://mongodb.com/random-url?hello=world")
		if err != nil {
			t.Errorf("Error parsing URL: %v", err)
			t.FailNow()
		}
		decimal128, err := primitive.ParseDecimal128("1.5e10")
		if err != nil {
			t.Errorf("Error parsing decimal128: %v", err)
			t.FailNow()
		}

		testCases := []struct {
			name  string
			value interface{}
			b     []byte
			err   error
		}{
			{
				"D to JavaScript",
				D{{"a", primitive.JavaScript(`function() { var hello = "world"; }`)}},
				docToBytes(bsonx.Doc{{"a", bsonx.JavaScript(`function() { var hello = "world"; }`)}}),
				nil,
			},
			{
				"D to Symbol",
				D{{"a", primitive.Symbol("foobarbaz")}},
				docToBytes(bsonx.Doc{{"a", bsonx.Symbol("foobarbaz")}}),
				nil,
			},
			{
				"struct{}",
				struct {
					A bool
					B int32
					C int64
					D uint16
					E uint64
					F float64
					G string
					H map[string]string
					I []byte
					K [2]string
					L struct {
						M string
					}
					O  bsonx.Doc
					P  Raw
					Q  primitive.ObjectID
					T  []struct{}
					Y  json.Number
					Z  time.Time
					AA json.Number
					AB *url.URL
					AC primitive.Decimal128
					AD *time.Time
					AE testValueMarshaler
					AF RawValue
					AG *RawValue
					AH D
					AI *D
					AJ *D
				}{
					A: true,
					B: 123,
					C: 456,
					D: 789,
					E: 101112,
					F: 3.14159,
					G: "Hello, world",
					H: map[string]string{"foo": "bar"},
					I: []byte{0x01, 0x02, 0x03},
					K: [2]string{"baz", "qux"},
					L: struct {
						M string
					}{
						M: "foobar",
					},
					O:  bsonx.Doc{{"countdown", bsonx.Int64(9876543210)}},
					P:  Raw{0x05, 0x00, 0x00, 0x00, 0x00},
					Q:  oid,
					T:  nil,
					Y:  json.Number("5"),
					Z:  now,
					AA: json.Number("10.1"),
					AB: murl,
					AC: decimal128,
					AD: &now,
					AE: testValueMarshaler{t: TypeString, buf: bsoncore.AppendString(nil, "hello, world")},
					AF: RawValue{Type: bsontype.String, Value: bsoncore.AppendString(nil, "hello, raw value")},
					AG: &RawValue{Type: bsontype.Double, Value: bsoncore.AppendDouble(nil, 3.14159)},
					AH: D{{"foo", "bar"}},
					AI: &D{{"pi", 3.14159}},
					AJ: nil,
				},
				docToBytes(bsonx.Doc{
					{"a", bsonx.Boolean(true)},
					{"b", bsonx.Int32(123)},
					{"c", bsonx.Int64(456)},
					{"d", bsonx.Int32(789)},
					{"e", bsonx.Int64(101112)},
					{"f", bsonx.Double(3.14159)},
					{"g", bsonx.String("Hello, world")},
					{"h", bsonx.Document(bsonx.Doc{{"foo", bsonx.String("bar")}})},
					{"i", bsonx.Binary(0x00, []byte{0x01, 0x02, 0x03})},
					{"k", bsonx.Array(bsonx.Arr{bsonx.String("baz"), bsonx.String("qux")})},
					{"l", bsonx.Document(bsonx.Doc{{"m", bsonx.String("foobar")}})},
					{"o", bsonx.Document(bsonx.Doc{{"countdown", bsonx.Int64(9876543210)}})},
					{"p", bsonx.Document(bsonx.Doc{})},
					{"q", bsonx.ObjectID(oid)},
					{"t", bsonx.Null()},
					{"y", bsonx.Int64(5)},
					{"z", bsonx.DateTime(now.UnixNano() / int64(time.Millisecond))},
					{"aa", bsonx.Double(10.1)},
					{"ab", bsonx.String(murl.String())},
					{"ac", bsonx.Decimal128(decimal128)},
					{"ad", bsonx.DateTime(now.UnixNano() / int64(time.Millisecond))},
					{"ae", bsonx.String("hello, world")},
					{"af", bsonx.String("hello, raw value")},
					{"ag", bsonx.Double(3.14159)},
					{"ah", bsonx.Document(bsonx.Doc{{"foo", bsonx.String("bar")}})},
					{"ai", bsonx.Document(bsonx.Doc{{"pi", bsonx.Double(3.14159)}})},
					{"aj", bsonx.Null()},
				}),
				nil,
			},
			{
				"struct{[]interface{}}",
				struct {
					A []bool
					B []int32
					C []int64
					D []uint16
					E []uint64
					F []float64
					G []string
					H []map[string]string
					I [][]byte
					K [1][2]string
					L []struct {
						M string
					}
					N  [][]string
					O  []bsonx.Elem
					P  []bsonx.Doc
					Q  []Raw
					R  []primitive.ObjectID
					T  []struct{}
					W  []map[string]struct{}
					X  []map[string]struct{}
					Y  []map[string]struct{}
					Z  []time.Time
					AA []json.Number
					AB []*url.URL
					AC []primitive.Decimal128
					AD []*time.Time
					AE []testValueMarshaler
					AF []D
					AG []*D
				}{
					A: []bool{true},
					B: []int32{123},
					C: []int64{456},
					D: []uint16{789},
					E: []uint64{101112},
					F: []float64{3.14159},
					G: []string{"Hello, world"},
					H: []map[string]string{{"foo": "bar"}},
					I: [][]byte{{0x01, 0x02, 0x03}},
					K: [1][2]string{{"baz", "qux"}},
					L: []struct {
						M string
					}{
						{
							M: "foobar",
						},
					},
					N:  [][]string{{"foo", "bar"}},
					O:  []bsonx.Elem{{"N", bsonx.Null()}},
					P:  []bsonx.Doc{{{"countdown", bsonx.Int64(9876543210)}}},
					Q:  []Raw{{0x05, 0x00, 0x00, 0x00, 0x00}},
					R:  oids,
					T:  nil,
					W:  nil,
					X:  []map[string]struct{}{},   // Should be empty BSON Array
					Y:  []map[string]struct{}{{}}, // Should be BSON array with one element, an empty BSON SubDocument
					Z:  []time.Time{now, now},
					AA: []json.Number{json.Number("5"), json.Number("10.1")},
					AB: []*url.URL{murl},
					AC: []primitive.Decimal128{decimal128},
					AD: []*time.Time{&now, &now},
					AE: []testValueMarshaler{
						{t: TypeString, buf: bsoncore.AppendString(nil, "hello")},
						{t: TypeString, buf: bsoncore.AppendString(nil, "world")},
					},
					AF: []D{{{"foo", "bar"}}, {{"hello", "world"}, {"number", 12345}}},
					AG: []*D{{{"pi", 3.14159}}, nil},
				},
				docToBytes(bsonx.Doc{
					{"a", bsonx.Array(bsonx.Arr{bsonx.Boolean(true)})},
					{"b", bsonx.Array(bsonx.Arr{bsonx.Int32(123)})},
					{"c", bsonx.Array(bsonx.Arr{bsonx.Int64(456)})},
					{"d", bsonx.Array(bsonx.Arr{bsonx.Int32(789)})},
					{"e", bsonx.Array(bsonx.Arr{bsonx.Int64(101112)})},
					{"f", bsonx.Array(bsonx.Arr{bsonx.Double(3.14159)})},
					{"g", bsonx.Array(bsonx.Arr{bsonx.String("Hello, world")})},
					{"h", bsonx.Array(bsonx.Arr{bsonx.Document(bsonx.Doc{{"foo", bsonx.String("bar")}})})},
					{"i", bsonx.Array(bsonx.Arr{bsonx.Binary(0x00, []byte{0x01, 0x02, 0x03})})},
					{"k", bsonx.Array(bsonx.Arr{bsonx.Array(bsonx.Arr{bsonx.String("baz"), bsonx.String("qux")})})},
					{"l", bsonx.Array(bsonx.Arr{bsonx.Document(bsonx.Doc{{"m", bsonx.String("foobar")}})})},
					{"n", bsonx.Array(bsonx.Arr{bsonx.Array(bsonx.Arr{bsonx.String("foo"), bsonx.String("bar")})})},
					{"o", bsonx.Document(bsonx.Doc{{"N", bsonx.Null()}})},
					{"p", bsonx.Array(bsonx.Arr{bsonx.Document(bsonx.Doc{{"countdown", bsonx.Int64(9876543210)}})})},
					{"q", bsonx.Array(bsonx.Arr{bsonx.Document(bsonx.Doc{})})},
					{"r", bsonx.Array(bsonx.Arr{bsonx.ObjectID(oids[0]), bsonx.ObjectID(oids[1]), bsonx.ObjectID(oids[2])})},
					{"t", bsonx.Null()},
					{"w", bsonx.Null()},
					{"x", bsonx.Array(bsonx.Arr{})},
					{"y", bsonx.Array(bsonx.Arr{bsonx.Document(bsonx.Doc{})})},
					{"z", bsonx.Array(bsonx.Arr{bsonx.DateTime(now.UnixNano() / int64(time.Millisecond)), bsonx.DateTime(now.UnixNano() / int64(time.Millisecond))})},
					{"aa", bsonx.Array(bsonx.Arr{bsonx.Int64(5), bsonx.Double(10.10)})},
					{"ab", bsonx.Array(bsonx.Arr{bsonx.String(murl.String())})},
					{"ac", bsonx.Array(bsonx.Arr{bsonx.Decimal128(decimal128)})},
					{"ad", bsonx.Array(bsonx.Arr{bsonx.DateTime(now.UnixNano() / int64(time.Millisecond)), bsonx.DateTime(now.UnixNano() / int64(time.Millisecond))})},
					{"ae", bsonx.Array(bsonx.Arr{bsonx.String("hello"), bsonx.String("world")})},
					{"af", bsonx.Array(bsonx.Arr{
						bsonx.Document(bsonx.Doc{{"foo", bsonx.String("bar")}}),
						bsonx.Document(bsonx.Doc{{"hello", bsonx.String("world")}, {"number", bsonx.Int64(12345)}})},
					)},
					{"ag", bsonx.Array(bsonx.Arr{bsonx.Document(bsonx.Doc{{"pi", bsonx.Double(3.14159)}}), bsonx.Null()})},
				}),
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				b := make(bsonrw.SliceWriter, 0, 512)
				vw, err := bsonrw.NewBSONValueWriter(&b)
				noerr(t, err)
				enc, err := NewEncoder(DefaultRegistry, vw)
				noerr(t, err)
				err = enc.Encode(tc.value)
				if err != tc.err {
					t.Errorf("Did not receive expected error. got %v; want %v", err, tc.err)
				}
				if diff := cmp.Diff([]byte(b), tc.b); diff != "" {
					t.Errorf("Bytes written differ: (-got +want)\n%s", diff)
					t.Errorf("Bytes\ngot: %v\nwant:%v\n", b, tc.b)
					t.Errorf("Readers\ngot: %v\nwant:%v\n", Raw(b), Raw(tc.b))
				}
			})
		}
	})
}

func TestDefaultValueDecoders(t *testing.T) {
	var pc PrimitiveCodecs
	var pcx bsonx.PrimitiveCodecs

	var pjs = new(primitive.JavaScript)
	*pjs = primitive.JavaScript("var hello = 'world';")
	var psymbol = new(primitive.Symbol)
	*psymbol = primitive.Symbol("foobarbaz")

	var wrong = func(string, string) string { return "wrong" }

	const cansetreflectiontest = "cansetreflectiontest"

	type subtest struct {
		name   string
		val    interface{}
		dctx   *bsoncodec.DecodeContext
		llvrw  *bsonrwtest.ValueReaderWriter
		invoke bsonrwtest.Invoked
		err    error
	}

	testCases := []struct {
		name     string
		vd       bsoncodec.ValueDecoder
		subtests []subtest
	}{
		{
			"RawValueDecodeValue",
			bsoncodec.ValueDecoderFunc(pc.RawValueDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{},
					bsonrwtest.Nothing,
					bsoncodec.ValueDecoderError{
						Name:     "RawValueDecodeValue",
						Types:    []reflect.Type{tRawValue},
						Received: reflect.ValueOf(wrong),
					},
				},
				{
					"ReadValue Error",
					RawValue{},
					nil,
					&bsonrwtest.ValueReaderWriter{
						BSONType: bsontype.Binary,
						Err:      errors.New("rb error"),
						ErrAfter: bsonrwtest.ReadBinary,
					},
					bsonrwtest.ReadBinary,
					errors.New("rb error"),
				},
				{
					"RawValue/success",
					RawValue{Type: bsontype.Binary, Value: bsoncore.AppendBinary(nil, 0xFF, []byte{0x01, 0x02, 0x03})},
					nil,
					&bsonrwtest.ValueReaderWriter{
						BSONType: bsontype.Binary,
						Return: bsoncore.Value{
							Type: bsontype.Binary,
							Data: bsoncore.AppendBinary(nil, 0xFF, []byte{0x01, 0x02, 0x03}),
						},
					},
					bsonrwtest.ReadBinary,
					nil,
				},
			},
		},
		{
			"ValueDecodeValue",
			bsoncodec.ValueDecoderFunc(pcx.ValueDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueDecoderError{
						Name:     "ValueDecodeValue",
						Types:    []reflect.Type{tValue},
						Received: reflect.ValueOf(wrong),
					},
				},
				{
					"invalid value",
					(*bsonx.Val)(nil),
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueDecoderError{
						Name:     "ValueDecodeValue",
						Types:    []reflect.Type{tValue},
						Received: reflect.ValueOf((*bsonx.Val)(nil)),
					},
				},
				{
					"success",
					bsonx.Double(3.14159),
					&bsoncodec.DecodeContext{Registry: NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Return: float64(3.14159)},
					bsonrwtest.ReadDouble,
					nil,
				},
			},
		},
		{
			"RawDecodeValue",
			bsoncodec.ValueDecoderFunc(pc.RawDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{},
					bsonrwtest.Nothing,
					bsoncodec.ValueDecoderError{Name: "RawDecodeValue", Types: []reflect.Type{tRaw}, Received: reflect.ValueOf(wrong)},
				},
				{
					"*Raw is nil",
					(*Raw)(nil),
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueDecoderError{
						Name:     "RawDecodeValue",
						Types:    []reflect.Type{tRaw},
						Received: reflect.ValueOf((*Raw)(nil)),
					},
				},
				{
					"Copy error",
					Raw{},
					nil,
					&bsonrwtest.ValueReaderWriter{Err: errors.New("copy error"), ErrAfter: bsonrwtest.ReadDocument},
					bsonrwtest.ReadDocument,
					errors.New("copy error"),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, rc := range tc.subtests {
				t.Run(rc.name, func(t *testing.T) {
					var dc bsoncodec.DecodeContext
					if rc.dctx != nil {
						dc = *rc.dctx
					}
					llvrw := new(bsonrwtest.ValueReaderWriter)
					if rc.llvrw != nil {
						llvrw = rc.llvrw
					}
					llvrw.T = t
					// var got interface{}
					if rc.val == cansetreflectiontest { // We're doing a CanSet reflection test
						err := tc.vd.DecodeValue(dc, llvrw, reflect.Value{})
						if !compareErrors(err, rc.err) {
							t.Errorf("Errors do not match. got %v; want %v", err, rc.err)
						}

						val := reflect.New(reflect.TypeOf(rc.val)).Elem()
						err = tc.vd.DecodeValue(dc, llvrw, val)
						if !compareErrors(err, rc.err) {
							t.Errorf("Errors do not match. got %v; want %v", err, rc.err)
						}
						return
					}
					// var unwrap bool
					// rtype := reflect.TypeOf(rc.val)
					// if rtype.Kind() == reflect.Ptr {
					// 	if reflect.ValueOf(rc.val).IsNil() {
					// 		got = rc.val
					// 	} else {
					// 		val := reflect.New(rtype).Elem()
					// 		elem := reflect.New(rtype.Elem())
					// 		val.Set(elem)
					// 		got = val.Addr().Interface()
					// 		unwrap = true
					// 	}
					// } else {
					// 	unwrap = true
					// 	got = reflect.New(reflect.TypeOf(rc.val)).Interface()
					// }
					var val reflect.Value
					if rtype := reflect.TypeOf(rc.val); rtype != nil {
						val = reflect.New(rtype).Elem()
					}
					want := rc.val
					defer func() {
						if err := recover(); err != nil {
							fmt.Println(t.Name())
							panic(err)
						}
					}()
					err := tc.vd.DecodeValue(dc, llvrw, val)
					if !compareErrors(err, rc.err) {
						t.Errorf("Errors do not match. got %v; want %v", err, rc.err)
					}
					invoked := llvrw.Invoked
					if !cmp.Equal(invoked, rc.invoke) {
						t.Errorf("Incorrect method invoked. got %v; want %v", invoked, rc.invoke)
					}
					// if unwrap {
					// 	got = reflect.ValueOf(got).Elem().Interface()
					// }
					var got interface{}
					if val.IsValid() && val.CanInterface() {
						got = val.Interface()
					}
					if rc.err == nil && !cmp.Equal(got, want, cmp.Comparer(compareValues)) {
						t.Errorf("Values do not match. got (%T)%v; want (%T)%v", got, got, want, want)
					}
				})
			}
		})
	}

	t.Run("DocumentDecodeValue", func(t *testing.T) {
		t.Run("CodecDecodeError", func(t *testing.T) {
			val := reflect.New(tBool).Elem()
			want := bsoncodec.ValueDecoderError{Name: "DocumentDecodeValue", Types: []reflect.Type{tDocument}, Received: val}
			got := pcx.DocumentDecodeValue(bsoncodec.DecodeContext{}, &bsonrwtest.ValueReaderWriter{BSONType: bsontype.EmbeddedDocument}, val)
			if !compareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
		t.Run("ReadDocument Error", func(t *testing.T) {
			want := errors.New("ReadDocument Error")
			llvrw := &bsonrwtest.ValueReaderWriter{
				T:        t,
				Err:      want,
				ErrAfter: bsonrwtest.ReadDocument,
				BSONType: bsontype.EmbeddedDocument,
			}
			got := pcx.DocumentDecodeValue(bsoncodec.DecodeContext{}, llvrw, reflect.New(reflect.TypeOf(bsonx.Doc{})).Elem())
			if !compareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
		t.Run("decodeDocument errors", func(t *testing.T) {
			dc := bsoncodec.DecodeContext{}
			err := errors.New("decodeDocument error")
			testCases := []struct {
				name  string
				dc    bsoncodec.DecodeContext
				llvrw *bsonrwtest.ValueReaderWriter
				err   error
			}{
				{
					"ReadElement",
					dc,
					&bsonrwtest.ValueReaderWriter{T: t, Err: errors.New("re error"), ErrAfter: bsonrwtest.ReadElement},
					errors.New("re error"),
				},
				{"ReadDouble", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadDouble, BSONType: bsontype.Double}, err},
				{"ReadString", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadString, BSONType: bsontype.String}, err},
				{"ReadBinary", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadBinary, BSONType: bsontype.Binary}, err},
				{"ReadUndefined", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadUndefined, BSONType: bsontype.Undefined}, err},
				{"ReadObjectID", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadObjectID, BSONType: bsontype.ObjectID}, err},
				{"ReadBoolean", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadBoolean, BSONType: bsontype.Boolean}, err},
				{"ReadDateTime", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadDateTime, BSONType: bsontype.DateTime}, err},
				{"ReadNull", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadNull, BSONType: bsontype.Null}, err},
				{"ReadRegex", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadRegex, BSONType: bsontype.Regex}, err},
				{"ReadDBPointer", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadDBPointer, BSONType: bsontype.DBPointer}, err},
				{"ReadJavascript", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadJavascript, BSONType: bsontype.JavaScript}, err},
				{"ReadSymbol", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadSymbol, BSONType: bsontype.Symbol}, err},
				{
					"ReadCodeWithScope (Lookup)", bsoncodec.DecodeContext{Registry: bsoncodec.NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadCodeWithScope, BSONType: bsontype.CodeWithScope},
					err,
				},
				{"ReadInt32", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadInt32, BSONType: bsontype.Int32}, err},
				{"ReadInt64", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadInt64, BSONType: bsontype.Int64}, err},
				{"ReadTimestamp", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadTimestamp, BSONType: bsontype.Timestamp}, err},
				{"ReadDecimal128", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadDecimal128, BSONType: bsontype.Decimal128}, err},
				{"ReadMinKey", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadMinKey, BSONType: bsontype.MinKey}, err},
				{"ReadMaxKey", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadMaxKey, BSONType: bsontype.MaxKey}, err},
				{"Invalid Type", dc, &bsonrwtest.ValueReaderWriter{T: t, BSONType: bsontype.Type(0)}, fmt.Errorf("Cannot read unknown BSON type %s", bsontype.Type(0))},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					err := pcx.DecodeDocument(tc.dc, tc.llvrw, new(bsonx.Doc))
					if !compareErrors(err, tc.err) {
						t.Errorf("Errors do not match. got %v; want %v", err, tc.err)
					}
				})
			}
		})

		t.Run("success", func(t *testing.T) {
			oid := primitive.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
			d128 := primitive.NewDecimal128(10, 20)
			want := bsonx.Doc{
				{"a", bsonx.Double(3.14159)}, {"b", bsonx.String("foo")},
				{"c", bsonx.Document(bsonx.Doc{{"aa", bsonx.Null()}})},
				{"d", bsonx.Array(bsonx.Arr{bsonx.Null()})},
				{"e", bsonx.Binary(0xFF, []byte{0x01, 0x02, 0x03})}, {"f", bsonx.Undefined()},
				{"g", bsonx.ObjectID(oid)}, {"h", bsonx.Boolean(true)},
				{"i", bsonx.DateTime(1234567890)}, {"j", bsonx.Null()}, {"k", bsonx.Regex("foo", "bar")},
				{"l", bsonx.DBPointer("foobar", oid)}, {"m", bsonx.JavaScript("var hello = 'world';")},
				{"n", bsonx.Symbol("bazqux")},
				{"o", bsonx.CodeWithScope("var hello = 'world';", bsonx.Doc{{"ab", bsonx.Null()}})},
				{"p", bsonx.Int32(12345)},
				{"q", bsonx.Timestamp(10, 20)}, {"r", bsonx.Int64(1234567890)},
				{"s", bsonx.Decimal128(d128)}, {"t", bsonx.MinKey()}, {"u", bsonx.MaxKey()},
			}
			got := reflect.New(reflect.TypeOf(bsonx.Doc{})).Elem()
			dc := bsoncodec.DecodeContext{Registry: NewRegistryBuilder().Build()}
			b, err := want.MarshalBSON()
			noerr(t, err)
			err = pcx.DocumentDecodeValue(dc, bsonrw.NewBSONDocumentReader(b), got)
			noerr(t, err)
			if !got.Interface().(bsonx.Doc).Equal(want) {
				t.Error("Documents do not match")
				t.Errorf("\ngot :%v\nwant:%v", got, want)
			}
		})
	})
	t.Run("ArrayDecodeValue", func(t *testing.T) {
		t.Run("CodecDecodeError", func(t *testing.T) {
			val := reflect.New(tBool).Elem()
			want := bsoncodec.ValueDecoderError{Name: "ArrayDecodeValue", Types: []reflect.Type{tArray}, Received: val}
			got := pcx.ArrayDecodeValue(bsoncodec.DecodeContext{}, &bsonrwtest.ValueReaderWriter{BSONType: bsontype.Array}, val)
			if !compareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
		t.Run("ReadArray Error", func(t *testing.T) {
			want := errors.New("ReadArray Error")
			llvrw := &bsonrwtest.ValueReaderWriter{
				T:        t,
				Err:      want,
				ErrAfter: bsonrwtest.ReadArray,
				BSONType: bsontype.Array,
			}
			got := pcx.ArrayDecodeValue(bsoncodec.DecodeContext{}, llvrw, reflect.New(tArray).Elem())
			if !compareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
		t.Run("decode array errors", func(t *testing.T) {
			dc := bsoncodec.DecodeContext{}
			err := errors.New("decode array error")
			testCases := []struct {
				name  string
				dc    bsoncodec.DecodeContext
				llvrw *bsonrwtest.ValueReaderWriter
				err   error
			}{
				{
					"ReadValue",
					dc,
					&bsonrwtest.ValueReaderWriter{T: t, Err: errors.New("re error"), ErrAfter: bsonrwtest.ReadValue},
					errors.New("re error"),
				},
				{"ReadDouble", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadDouble, BSONType: bsontype.Double}, err},
				{"ReadString", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadString, BSONType: bsontype.String}, err},
				{"ReadBinary", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadBinary, BSONType: bsontype.Binary}, err},
				{"ReadUndefined", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadUndefined, BSONType: bsontype.Undefined}, err},
				{"ReadObjectID", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadObjectID, BSONType: bsontype.ObjectID}, err},
				{"ReadBoolean", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadBoolean, BSONType: bsontype.Boolean}, err},
				{"ReadDateTime", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadDateTime, BSONType: bsontype.DateTime}, err},
				{"ReadNull", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadNull, BSONType: bsontype.Null}, err},
				{"ReadRegex", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadRegex, BSONType: bsontype.Regex}, err},
				{"ReadDBPointer", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadDBPointer, BSONType: bsontype.DBPointer}, err},
				{"ReadJavascript", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadJavascript, BSONType: bsontype.JavaScript}, err},
				{"ReadSymbol", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadSymbol, BSONType: bsontype.Symbol}, err},
				{
					"ReadCodeWithScope (Lookup)", bsoncodec.DecodeContext{Registry: bsoncodec.NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadCodeWithScope, BSONType: bsontype.CodeWithScope},
					err,
				},
				{"ReadInt32", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadInt32, BSONType: bsontype.Int32}, err},
				{"ReadInt64", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadInt64, BSONType: bsontype.Int64}, err},
				{"ReadTimestamp", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadTimestamp, BSONType: bsontype.Timestamp}, err},
				{"ReadDecimal128", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadDecimal128, BSONType: bsontype.Decimal128}, err},
				{"ReadMinKey", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadMinKey, BSONType: bsontype.MinKey}, err},
				{"ReadMaxKey", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadMaxKey, BSONType: bsontype.MaxKey}, err},
				{"Invalid Type", dc, &bsonrwtest.ValueReaderWriter{T: t, BSONType: bsontype.Type(0)}, fmt.Errorf("Cannot read unknown BSON type %s", bsontype.Type(0))},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					err := pcx.ArrayDecodeValue(tc.dc, tc.llvrw, reflect.New(tArray).Elem())
					if !compareErrors(err, tc.err) {
						t.Errorf("Errors do not match. got %v; want %v", err, tc.err)
					}
				})
			}
		})

		t.Run("success", func(t *testing.T) {
			oid := primitive.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
			d128 := primitive.NewDecimal128(10, 20)
			want := bsonx.Arr{
				bsonx.Double(3.14159), bsonx.String("foo"), bsonx.Document(bsonx.Doc{{"aa", bsonx.Null()}}),
				bsonx.Array(bsonx.Arr{bsonx.Null()}),
				bsonx.Binary(0xFF, []byte{0x01, 0x02, 0x03}), bsonx.Undefined(),
				bsonx.ObjectID(oid), bsonx.Boolean(true), bsonx.DateTime(1234567890), bsonx.Null(), bsonx.Regex("foo", "bar"),
				bsonx.DBPointer("foobar", oid), bsonx.JavaScript("var hello = 'world';"), bsonx.Symbol("bazqux"),
				bsonx.CodeWithScope("var hello = 'world';", bsonx.Doc{{"ab", bsonx.Null()}}), bsonx.Int32(12345),
				bsonx.Timestamp(10, 20), bsonx.Int64(1234567890), bsonx.Decimal128(d128), bsonx.MinKey(), bsonx.MaxKey(),
			}
			dc := bsoncodec.DecodeContext{Registry: NewRegistryBuilder().Build()}

			b, err := bsonx.Doc{{"", bsonx.Array(want)}}.MarshalBSON()
			noerr(t, err)
			dvr := bsonrw.NewBSONDocumentReader(b)
			dr, err := dvr.ReadDocument()
			noerr(t, err)
			_, vr, err := dr.ReadElement()
			noerr(t, err)

			val := reflect.New(tArray).Elem()
			err = pcx.ArrayDecodeValue(dc, vr, val)
			noerr(t, err)
			got := val.Interface().(bsonx.Arr)
			if !got.Equal(want) {
				t.Error("Documents do not match")
				t.Errorf("\ngot :%v\nwant:%v", got, want)
			}
		})
	})

	t.Run("success path", func(t *testing.T) {
		oid := primitive.NewObjectID()
		oids := []primitive.ObjectID{primitive.NewObjectID(), primitive.NewObjectID(), primitive.NewObjectID()}
		var str = new(string)
		*str = "bar"
		now := time.Now().Truncate(time.Millisecond)
		murl, err := url.Parse("https://mongodb.com/random-url?hello=world")
		if err != nil {
			t.Errorf("Error parsing URL: %v", err)
			t.FailNow()
		}
		decimal128, err := primitive.ParseDecimal128("1.5e10")
		if err != nil {
			t.Errorf("Error parsing decimal128: %v", err)
			t.FailNow()
		}

		testCases := []struct {
			name  string
			value interface{}
			b     []byte
			err   error
		}{
			{
				"map[string]int",
				map[string]int32{"foo": 1},
				[]byte{
					0x0E, 0x00, 0x00, 0x00,
					0x10, 'f', 'o', 'o', 0x00,
					0x01, 0x00, 0x00, 0x00,
					0x00,
				},
				nil,
			},
			{
				"map[string]primitive.ObjectID",
				map[string]primitive.ObjectID{"foo": oid},
				docToBytes(bsonx.Doc{{"foo", bsonx.ObjectID(oid)}}),
				nil,
			},
			{
				"map[string][]Element",
				map[string][]bsonx.Elem{"Z": {{"A", bsonx.Int32(1)}, {"B", bsonx.Int32(2)}, {"EC", bsonx.Int32(3)}}},
				docToBytes(bsonx.Doc{{"Z", bsonx.Document(bsonx.Doc{{"A", bsonx.Int32(1)}, {"B", bsonx.Int32(2)}, {"EC", bsonx.Int32(3)}})}}),
				nil,
			},
			{
				"map[string][]Value",
				map[string][]bsonx.Val{"Z": {bsonx.Int32(1), bsonx.Int32(2), bsonx.Int32(3)}},
				docToBytes(bsonx.Doc{{"Z", bsonx.Array(bsonx.Arr{bsonx.Int32(1), bsonx.Int32(2), bsonx.Int32(3)})}}),
				nil,
			},
			{
				"map[string]*Document",
				map[string]bsonx.Doc{"Z": {{"foo", bsonx.Null()}}},
				docToBytes(bsonx.Doc{{"Z", bsonx.Document(bsonx.Doc{{"foo", bsonx.Null()}})}}),
				nil,
			},
			{
				"map[string]Reader",
				map[string]Raw{"Z": {0x05, 0x00, 0x00, 0x00, 0x00}},
				docToBytes(bsonx.Doc{{"Z", bsonx.Document(rawToDoc(Raw{0x05, 0x00, 0x00, 0x00, 0x00}))}}),
				nil,
			},
			{
				"map[string][]int32",
				map[string][]int32{"Z": {1, 2, 3}},
				docToBytes(bsonx.Doc{{"Z", bsonx.Array(bsonx.Arr{bsonx.Int32(1), bsonx.Int32(2), bsonx.Int32(3)})}}),
				nil,
			},
			{
				"map[string][]primitive.ObjectID",
				map[string][]primitive.ObjectID{"Z": oids},
				docToBytes(bsonx.Doc{{"Z", bsonx.Array(bsonx.Arr{bsonx.ObjectID(oids[0]), bsonx.ObjectID(oids[1]), bsonx.ObjectID(oids[2])})}}),
				nil,
			},
			{
				"map[string][]json.Number(int64)",
				map[string][]json.Number{"Z": {json.Number("5"), json.Number("10")}},
				docToBytes(bsonx.Doc{{"Z", bsonx.Array(bsonx.Arr{bsonx.Int64(5), bsonx.Int64(10)})}}),
				nil,
			},
			{
				"map[string][]json.Number(float64)",
				map[string][]json.Number{"Z": {json.Number("5"), json.Number("10.1")}},
				docToBytes(bsonx.Doc{{"Z", bsonx.Array(bsonx.Arr{bsonx.Int64(5), bsonx.Double(10.1)})}}),
				nil,
			},
			{
				"map[string][]*url.URL",
				map[string][]*url.URL{"Z": {murl}},
				docToBytes(bsonx.Doc{{"Z", bsonx.Array(bsonx.Arr{bsonx.String(murl.String())})}}),
				nil,
			},
			{
				"map[string][]primitive.Decimal128",
				map[string][]primitive.Decimal128{"Z": {decimal128}},
				docToBytes(bsonx.Doc{{"Z", bsonx.Array(bsonx.Arr{bsonx.Decimal128(decimal128)})}}),
				nil,
			},
			{
				"-",
				struct {
					A string `bson:"-"`
				}{
					A: "",
				},
				docToBytes(bsonx.Doc{}),
				nil,
			},
			{
				"omitempty",
				struct {
					A string `bson:",omitempty"`
				}{
					A: "",
				},
				docToBytes(bsonx.Doc{}),
				nil,
			},
			{
				"omitempty, empty time",
				struct {
					A time.Time `bson:",omitempty"`
				}{
					A: time.Time{},
				},
				docToBytes(bsonx.Doc{}),
				nil,
			},
			{
				"no private fields",
				noPrivateFields{a: "should be empty"},
				docToBytes(bsonx.Doc{}),
				nil,
			},
			{
				"minsize",
				struct {
					A int64 `bson:",minsize"`
				}{
					A: 12345,
				},
				docToBytes(bsonx.Doc{{"a", bsonx.Int32(12345)}}),
				nil,
			},
			{
				"inline",
				struct {
					Foo struct {
						A int64 `bson:",minsize"`
					} `bson:",inline"`
				}{
					Foo: struct {
						A int64 `bson:",minsize"`
					}{
						A: 12345,
					},
				},
				docToBytes(bsonx.Doc{{"a", bsonx.Int32(12345)}}),
				nil,
			},
			{
				"inline map",
				struct {
					Foo map[string]string `bson:",inline"`
				}{
					Foo: map[string]string{"foo": "bar"},
				},
				docToBytes(bsonx.Doc{{"foo", bsonx.String("bar")}}),
				nil,
			},
			{
				"alternate name bson:name",
				struct {
					A string `bson:"foo"`
				}{
					A: "bar",
				},
				docToBytes(bsonx.Doc{{"foo", bsonx.String("bar")}}),
				nil,
			},
			{
				"alternate name",
				struct {
					A string `bson:"foo"`
				}{
					A: "bar",
				},
				docToBytes(bsonx.Doc{{"foo", bsonx.String("bar")}}),
				nil,
			},
			{
				"inline, omitempty",
				struct {
					A   string
					Foo zeroTest `bson:"omitempty,inline"`
				}{
					A:   "bar",
					Foo: zeroTest{true},
				},
				docToBytes(bsonx.Doc{{"a", bsonx.String("bar")}}),
				nil,
			},
			{
				"JavaScript to D",
				D{{"a", primitive.JavaScript(`function() { var hello = "world"; }`)}},
				docToBytes(bsonx.Doc{{"a", bsonx.JavaScript(`function() { var hello = "world"; }`)}}),
				nil,
			},
			{
				"Symbol to D",
				D{{"a", primitive.Symbol("foobarbaz")}},
				docToBytes(bsonx.Doc{{"a", bsonx.Symbol("foobarbaz")}}),
				nil,
			},
			{
				"struct{}",
				struct {
					A bool
					B int32
					C int64
					D uint16
					E uint64
					F float64
					G string
					H map[string]string
					I []byte
					K [2]string
					L struct {
						M string
					}
					O  bsonx.Doc
					P  Raw
					Q  primitive.ObjectID
					T  []struct{}
					Y  json.Number
					Z  time.Time
					AA json.Number
					AB *url.URL
					AC primitive.Decimal128
					AD *time.Time
					AE *testValueUnmarshaler
					AF RawValue
					AG *RawValue
					AH D
					AI *D
					AJ *D
				}{
					A: true,
					B: 123,
					C: 456,
					D: 789,
					E: 101112,
					F: 3.14159,
					G: "Hello, world",
					H: map[string]string{"foo": "bar"},
					I: []byte{0x01, 0x02, 0x03},
					K: [2]string{"baz", "qux"},
					L: struct {
						M string
					}{
						M: "foobar",
					},
					O:  bsonx.Doc{{"countdown", bsonx.Int64(9876543210)}},
					P:  Raw{0x05, 0x00, 0x00, 0x00, 0x00},
					Q:  oid,
					T:  nil,
					Y:  json.Number("5"),
					Z:  now,
					AA: json.Number("10.1"),
					AB: murl,
					AC: decimal128,
					AD: &now,
					AE: &testValueUnmarshaler{t: bsontype.String, val: bsoncore.AppendString(nil, "hello, world!")},
					AF: RawValue{Type: bsontype.Double, Value: bsoncore.AppendDouble(nil, 3.14159)},
					AG: &RawValue{Type: bsontype.Binary, Value: bsoncore.AppendBinary(nil, 0xFF, []byte{0x01, 0x02, 0x03})},
					AH: D{{"foo", "bar"}},
					AI: &D{{"pi", 3.14159}},
					AJ: nil,
				},
				docToBytes(bsonx.Doc{
					{"a", bsonx.Boolean(true)},
					{"b", bsonx.Int32(123)},
					{"c", bsonx.Int64(456)},
					{"d", bsonx.Int32(789)},
					{"e", bsonx.Int64(101112)},
					{"f", bsonx.Double(3.14159)},
					{"g", bsonx.String("Hello, world")},
					{"h", bsonx.Document(bsonx.Doc{{"foo", bsonx.String("bar")}})},
					{"i", bsonx.Binary(0x00, []byte{0x01, 0x02, 0x03})},
					{"k", bsonx.Array(bsonx.Arr{bsonx.String("baz"), bsonx.String("qux")})},
					{"l", bsonx.Document(bsonx.Doc{{"m", bsonx.String("foobar")}})},
					{"o", bsonx.Document(bsonx.Doc{{"countdown", bsonx.Int64(9876543210)}})},
					{"p", bsonx.Document(bsonx.Doc{})},
					{"q", bsonx.ObjectID(oid)},
					{"t", bsonx.Null()},
					{"y", bsonx.Int64(5)},
					{"z", bsonx.DateTime(now.UnixNano() / int64(time.Millisecond))},
					{"aa", bsonx.Double(10.1)},
					{"ab", bsonx.String(murl.String())},
					{"ac", bsonx.Decimal128(decimal128)},
					{"ad", bsonx.DateTime(now.UnixNano() / int64(time.Millisecond))},
					{"ae", bsonx.String("hello, world!")},
					{"af", bsonx.Double(3.14159)},
					{"ag", bsonx.Binary(0xFF, []byte{0x01, 0x02, 0x03})},
					{"ah", bsonx.Document(bsonx.Doc{{"foo", bsonx.String("bar")}})},
					{"ai", bsonx.Document(bsonx.Doc{{"pi", bsonx.Double(3.14159)}})},
					{"aj", bsonx.Null()},
				}),
				nil,
			},
			{
				"struct{[]interface{}}",
				struct {
					A []bool
					B []int32
					C []int64
					D []uint16
					E []uint64
					F []float64
					G []string
					H []map[string]string
					I [][]byte
					K [1][2]string
					L []struct {
						M string
					}
					N  [][]string
					O  []bsonx.Elem
					P  []bsonx.Doc
					Q  []Raw
					R  []primitive.ObjectID
					T  []struct{}
					W  []map[string]struct{}
					X  []map[string]struct{}
					Y  []map[string]struct{}
					Z  []time.Time
					AA []json.Number
					AB []*url.URL
					AC []primitive.Decimal128
					AD []*time.Time
					AE []*testValueUnmarshaler
					AF []D
					AG []*D
				}{
					A: []bool{true},
					B: []int32{123},
					C: []int64{456},
					D: []uint16{789},
					E: []uint64{101112},
					F: []float64{3.14159},
					G: []string{"Hello, world"},
					H: []map[string]string{{"foo": "bar"}},
					I: [][]byte{{0x01, 0x02, 0x03}},
					K: [1][2]string{{"baz", "qux"}},
					L: []struct {
						M string
					}{
						{
							M: "foobar",
						},
					},
					N:  [][]string{{"foo", "bar"}},
					O:  []bsonx.Elem{{"N", bsonx.Null()}},
					P:  []bsonx.Doc{{{"countdown", bsonx.Int64(9876543210)}}},
					Q:  []Raw{{0x05, 0x00, 0x00, 0x00, 0x00}},
					R:  oids,
					T:  nil,
					W:  nil,
					X:  []map[string]struct{}{},   // Should be empty BSON Array
					Y:  []map[string]struct{}{{}}, // Should be BSON array with one element, an empty BSON SubDocument
					Z:  []time.Time{now, now},
					AA: []json.Number{json.Number("5"), json.Number("10.1")},
					AB: []*url.URL{murl},
					AC: []primitive.Decimal128{decimal128},
					AD: []*time.Time{&now, &now},
					AE: []*testValueUnmarshaler{
						{t: bsontype.String, val: bsoncore.AppendString(nil, "hello")},
						{t: bsontype.String, val: bsoncore.AppendString(nil, "world")},
					},
					AF: []D{{{"foo", "bar"}}, {{"hello", "world"}, {"number", int64(12345)}}},
					AG: []*D{{{"pi", 3.14159}}, nil},
				},
				docToBytes(bsonx.Doc{
					{"a", bsonx.Array(bsonx.Arr{bsonx.Boolean(true)})},
					{"b", bsonx.Array(bsonx.Arr{bsonx.Int32(123)})},
					{"c", bsonx.Array(bsonx.Arr{bsonx.Int64(456)})},
					{"d", bsonx.Array(bsonx.Arr{bsonx.Int32(789)})},
					{"e", bsonx.Array(bsonx.Arr{bsonx.Int64(101112)})},
					{"f", bsonx.Array(bsonx.Arr{bsonx.Double(3.14159)})},
					{"g", bsonx.Array(bsonx.Arr{bsonx.String("Hello, world")})},
					{"h", bsonx.Array(bsonx.Arr{bsonx.Document(bsonx.Doc{{"foo", bsonx.String("bar")}})})},
					{"i", bsonx.Array(bsonx.Arr{bsonx.Binary(0x00, []byte{0x01, 0x02, 0x03})})},
					{"k", bsonx.Array(bsonx.Arr{bsonx.Array(bsonx.Arr{bsonx.String("baz"), bsonx.String("qux")})})},
					{"l", bsonx.Array(bsonx.Arr{bsonx.Document(bsonx.Doc{{"m", bsonx.String("foobar")}})})},
					{"n", bsonx.Array(bsonx.Arr{bsonx.Array(bsonx.Arr{bsonx.String("foo"), bsonx.String("bar")})})},
					{"o", bsonx.Document(bsonx.Doc{{"N", bsonx.Null()}})},
					{"p", bsonx.Array(bsonx.Arr{bsonx.Document(bsonx.Doc{{"countdown", bsonx.Int64(9876543210)}})})},
					{"q", bsonx.Array(bsonx.Arr{bsonx.Document(bsonx.Doc{})})},
					{"r", bsonx.Array(bsonx.Arr{bsonx.ObjectID(oids[0]), bsonx.ObjectID(oids[1]), bsonx.ObjectID(oids[2])})},
					{"t", bsonx.Null()},
					{"w", bsonx.Null()},
					{"x", bsonx.Array(bsonx.Arr{})},
					{"y", bsonx.Array(bsonx.Arr{bsonx.Document(bsonx.Doc{})})},
					{"z", bsonx.Array(bsonx.Arr{bsonx.DateTime(now.UnixNano() / int64(time.Millisecond)), bsonx.DateTime(now.UnixNano() / int64(time.Millisecond))})},
					{"aa", bsonx.Array(bsonx.Arr{bsonx.Int64(5), bsonx.Double(10.10)})},
					{"ab", bsonx.Array(bsonx.Arr{bsonx.String(murl.String())})},
					{"ac", bsonx.Array(bsonx.Arr{bsonx.Decimal128(decimal128)})},
					{"ad", bsonx.Array(bsonx.Arr{bsonx.DateTime(now.UnixNano() / int64(time.Millisecond)), bsonx.DateTime(now.UnixNano() / int64(time.Millisecond))})},
					{"ae", bsonx.Array(bsonx.Arr{bsonx.String("hello"), bsonx.String("world")})},
					{"af", bsonx.Array(bsonx.Arr{
						bsonx.Document(bsonx.Doc{{"foo", bsonx.String("bar")}}),
						bsonx.Document(bsonx.Doc{{"hello", bsonx.String("world")}, {"number", bsonx.Int64(12345)}}),
					})},
					{"ag", bsonx.Array(bsonx.Arr{bsonx.Document(bsonx.Doc{{"pi", bsonx.Double(3.14159)}}), bsonx.Null()})},
				}),
				nil,
			},
		}

		t.Run("Decode", func(t *testing.T) {
			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					vr := bsonrw.NewBSONDocumentReader(tc.b)
					dec, err := NewDecoder(DefaultRegistry, vr)
					noerr(t, err)
					gotVal := reflect.New(reflect.TypeOf(tc.value))
					err = dec.Decode(gotVal.Interface())
					noerr(t, err)
					got := gotVal.Elem().Interface()
					want := tc.value
					if diff := cmp.Diff(
						got, want,
						cmp.Comparer(compareElements),
						cmp.Comparer(compareDecimal128),
						cmp.Comparer(compareValues),
						cmp.Comparer(compareNoPrivateFields),
						cmp.Comparer(compareZeroTest),
					); diff != "" {
						t.Errorf("difference:\n%s", diff)
						t.Errorf("Values are not equal.\ngot: %#v\nwant:%#v", got, want)
					}
				})
			}
		})
	})
}

type testValueMarshaler struct {
	t   bsontype.Type
	buf []byte
	err error
}

func (tvm testValueMarshaler) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return tvm.t, tvm.buf, tvm.err
}

type testValueUnmarshaler struct {
	t   bsontype.Type
	val []byte
	err error
}

func (tvu *testValueUnmarshaler) UnmarshalBSONValue(t bsontype.Type, val []byte) error {
	tvu.t, tvu.val = t, val
	return tvu.err
}
func (tvu testValueUnmarshaler) Equal(tvu2 testValueUnmarshaler) bool {
	return tvu.t == tvu2.t && bytes.Equal(tvu.val, tvu2.val)
}

type noPrivateFields struct {
	a string
}

func compareNoPrivateFields(npf1, npf2 noPrivateFields) bool {
	return npf1.a != npf2.a // We don't want these to be equal
}

type zeroTest struct {
	reportZero bool
}

func (z zeroTest) IsZero() bool { return z.reportZero }

func compareZeroTest(_, _ zeroTest) bool { return true }

type nonZeroer struct {
	value bool
}

type llCodec struct {
	t         *testing.T
	decodeval interface{}
	encodeval interface{}
	err       error
}

func (llc *llCodec) EncodeValue(_ bsoncodec.EncodeContext, _ bsonrw.ValueWriter, i interface{}) error {
	if llc.err != nil {
		return llc.err
	}

	llc.encodeval = i
	return nil
}

func (llc *llCodec) DecodeValue(_ bsoncodec.DecodeContext, _ bsonrw.ValueReader, val reflect.Value) error {
	if llc.err != nil {
		return llc.err
	}

	switch val.Type() {
	case tDocument:
		decodeval, ok := llc.decodeval.(bsonx.Doc)
		if !ok {
			llc.t.Errorf("decodeval must be a *Document if the i is a *Document. decodeval %T", llc.decodeval)
			return nil
		}

		doc := val.Interface().(bsonx.Doc)
		doc = doc[:0]
		doc = append(doc, decodeval...)
		return nil
	case tArray:
		decodeval, ok := llc.decodeval.(bsonx.Arr)
		if !ok {
			llc.t.Errorf("decodeval must be a *Array if the i is a *Array. decodeval %T", llc.decodeval)
			return nil
		}

		arr := val.Interface().(bsonx.Arr)
		arr = arr[:0]
		arr = append(arr, decodeval...)
		return nil
	}

	if !reflect.TypeOf(llc.decodeval).AssignableTo(val.Type()) {
		llc.t.Errorf("decodeval must be assignable to val provided to DecodeValue, but is not. decodeval %T; val %T", llc.decodeval, val)
		return nil
	}

	val.Set(reflect.ValueOf(llc.decodeval))
	return nil
}

func rawToDoc(raw Raw) bsonx.Doc {
	doc, err := bsonx.ReadDoc(raw)
	if err != nil {
		panic(err)
	}
	return doc
}
