// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

func TestCopier(t *testing.T) {
	t.Run("CopyDocument", func(t *testing.T) {
		t.Run("ReadDocument Error", func(t *testing.T) {
			want := errors.New("ReadDocumentError")
			src := &TestValueReaderWriter{t: t, err: want, errAfter: llvrwReadDocument}
			got := copyDocument(nil, src)
			if !assert.CompareErrors(got, want) {
				t.Errorf("Did not receive correct error. got %v; want %v", got, want)
			}
		})
		t.Run("WriteDocument Error", func(t *testing.T) {
			want := errors.New("WriteDocumentError")
			src := &TestValueReaderWriter{}
			dst := &TestValueReaderWriter{t: t, err: want, errAfter: llvrwWriteDocument}
			got := copyDocument(dst, src)
			if !assert.CompareErrors(got, want) {
				t.Errorf("Did not receive correct error. got %v; want %v", got, want)
			}
		})
		t.Run("success", func(t *testing.T) {
			idx, doc := bsoncore.AppendDocumentStart(nil)
			doc = bsoncore.AppendStringElement(doc, "Hello", "world")
			doc, err := bsoncore.AppendDocumentEnd(doc, idx)
			noerr(t, err)
			src := newBufferedDocumentReader(doc)
			dst := newValueWriterFromSlice(make([]byte, 0))
			want := doc
			err = copyDocument(dst, src)
			noerr(t, err)
			got := dst.buf
			if !bytes.Equal(got, want) {
				t.Errorf("Bytes are not equal. got %v; want %v", got, want)
			}
		})
	})
	t.Run("copyArray", func(t *testing.T) {
		t.Run("ReadArray Error", func(t *testing.T) {
			want := errors.New("ReadArrayError")
			src := &TestValueReaderWriter{t: t, err: want, errAfter: llvrwReadArray}
			got := copyArray(nil, src)
			if !assert.CompareErrors(got, want) {
				t.Errorf("Did not receive correct error. got %v; want %v", got, want)
			}
		})
		t.Run("WriteArray Error", func(t *testing.T) {
			want := errors.New("WriteArrayError")
			src := &TestValueReaderWriter{}
			dst := &TestValueReaderWriter{t: t, err: want, errAfter: llvrwWriteArray}
			got := copyArray(dst, src)
			if !assert.CompareErrors(got, want) {
				t.Errorf("Did not receive correct error. got %v; want %v", got, want)
			}
		})
		t.Run("success", func(t *testing.T) {
			idx, doc := bsoncore.AppendDocumentStart(nil)
			aidx, doc := bsoncore.AppendArrayElementStart(doc, "foo")
			doc = bsoncore.AppendStringElement(doc, "0", "Hello, world!")
			doc, err := bsoncore.AppendArrayEnd(doc, aidx)
			noerr(t, err)
			doc, err = bsoncore.AppendDocumentEnd(doc, idx)
			noerr(t, err)
			src := newBufferedDocumentReader(doc)

			_, err = src.ReadDocument()
			noerr(t, err)
			_, _, err = src.ReadElement()
			noerr(t, err)

			dst := newValueWriterFromSlice(make([]byte, 0))
			_, err = dst.WriteDocument()
			noerr(t, err)
			_, err = dst.WriteDocumentElement("foo")
			noerr(t, err)
			want := doc

			err = copyArray(dst, src)
			noerr(t, err)

			err = dst.WriteDocumentEnd()
			noerr(t, err)

			got := dst.buf
			if !bytes.Equal(got, want) {
				t.Errorf("Bytes are not equal. got %v; want %v", got, want)
			}
		})
	})
	t.Run("CopyValue", func(t *testing.T) {
		testCases := []struct {
			name string
			dst  *TestValueReaderWriter
			src  *TestValueReaderWriter
			err  error
		}{
			{
				"Double/src/error",
				&TestValueReaderWriter{},
				&TestValueReaderWriter{bsontype: TypeDouble, err: errors.New("1"), errAfter: llvrwReadDouble},
				errors.New("1"),
			},
			{
				"Double/dst/error",
				&TestValueReaderWriter{bsontype: TypeDouble, err: errors.New("2"), errAfter: llvrwWriteDouble},
				&TestValueReaderWriter{bsontype: TypeDouble, readval: float64(3.14159)},
				errors.New("2"),
			},
			{
				"String/src/error",
				&TestValueReaderWriter{},
				&TestValueReaderWriter{bsontype: TypeString, err: errors.New("1"), errAfter: llvrwReadString},
				errors.New("1"),
			},
			{
				"String/dst/error",
				&TestValueReaderWriter{bsontype: TypeString, err: errors.New("2"), errAfter: llvrwWriteString},
				&TestValueReaderWriter{bsontype: TypeString, readval: "hello, world"},
				errors.New("2"),
			},
			{
				"Document/src/error",
				&TestValueReaderWriter{},
				&TestValueReaderWriter{bsontype: TypeEmbeddedDocument, err: errors.New("1"), errAfter: llvrwReadDocument},
				errors.New("1"),
			},
			{
				"Array/dst/error",
				&TestValueReaderWriter{},
				&TestValueReaderWriter{bsontype: TypeArray, err: errors.New("2"), errAfter: llvrwReadArray},
				errors.New("2"),
			},
			{
				"Binary/src/error",
				&TestValueReaderWriter{},
				&TestValueReaderWriter{bsontype: TypeBinary, err: errors.New("1"), errAfter: llvrwReadBinary},
				errors.New("1"),
			},
			{
				"Binary/dst/error",
				&TestValueReaderWriter{bsontype: TypeBinary, err: errors.New("2"), errAfter: llvrwWriteBinaryWithSubtype},
				&TestValueReaderWriter{
					bsontype: TypeBinary,
					readval: bsoncore.Value{
						Type: bsoncore.TypeBinary,
						Data: []byte{0x03, 0x00, 0x00, 0x00, 0xFF, 0x01, 0x02, 0x03},
					},
				},
				errors.New("2"),
			},
			{
				"Undefined/src/error",
				&TestValueReaderWriter{},
				&TestValueReaderWriter{bsontype: TypeUndefined, err: errors.New("1"), errAfter: llvrwReadUndefined},
				errors.New("1"),
			},
			{
				"Undefined/dst/error",
				&TestValueReaderWriter{bsontype: TypeUndefined, err: errors.New("2"), errAfter: llvrwWriteUndefined},
				&TestValueReaderWriter{bsontype: TypeUndefined},
				errors.New("2"),
			},
			{
				"ObjectID/src/error",
				&TestValueReaderWriter{},
				&TestValueReaderWriter{bsontype: TypeObjectID, err: errors.New("1"), errAfter: llvrwReadObjectID},
				errors.New("1"),
			},
			{
				"ObjectID/dst/error",
				&TestValueReaderWriter{bsontype: TypeObjectID, err: errors.New("2"), errAfter: llvrwWriteObjectID},
				&TestValueReaderWriter{bsontype: TypeObjectID, readval: ObjectID{0x01, 0x02, 0x03}},
				errors.New("2"),
			},
			{
				"Boolean/src/error",
				&TestValueReaderWriter{},
				&TestValueReaderWriter{bsontype: TypeBoolean, err: errors.New("1"), errAfter: llvrwReadBoolean},
				errors.New("1"),
			},
			{
				"Boolean/dst/error",
				&TestValueReaderWriter{bsontype: TypeBoolean, err: errors.New("2"), errAfter: llvrwWriteBoolean},
				&TestValueReaderWriter{bsontype: TypeBoolean, readval: bool(true)},
				errors.New("2"),
			},
			{
				"DateTime/src/error",
				&TestValueReaderWriter{},
				&TestValueReaderWriter{bsontype: TypeDateTime, err: errors.New("1"), errAfter: llvrwReadDateTime},
				errors.New("1"),
			},
			{
				"DateTime/dst/error",
				&TestValueReaderWriter{bsontype: TypeDateTime, err: errors.New("2"), errAfter: llvrwWriteDateTime},
				&TestValueReaderWriter{bsontype: TypeDateTime, readval: int64(1234567890)},
				errors.New("2"),
			},
			{
				"Null/src/error",
				&TestValueReaderWriter{},
				&TestValueReaderWriter{bsontype: TypeNull, err: errors.New("1"), errAfter: llvrwReadNull},
				errors.New("1"),
			},
			{
				"Null/dst/error",
				&TestValueReaderWriter{bsontype: TypeNull, err: errors.New("2"), errAfter: llvrwWriteNull},
				&TestValueReaderWriter{bsontype: TypeNull},
				errors.New("2"),
			},
			{
				"Regex/src/error",
				&TestValueReaderWriter{},
				&TestValueReaderWriter{bsontype: TypeRegex, err: errors.New("1"), errAfter: llvrwReadRegex},
				errors.New("1"),
			},
			{
				"Regex/dst/error",
				&TestValueReaderWriter{bsontype: TypeRegex, err: errors.New("2"), errAfter: llvrwWriteRegex},
				&TestValueReaderWriter{
					bsontype: TypeRegex,
					readval: bsoncore.Value{
						Type: bsoncore.TypeRegex,
						Data: bsoncore.AppendRegex(nil, "hello", "world"),
					},
				},
				errors.New("2"),
			},
			{
				"DBPointer/src/error",
				&TestValueReaderWriter{},
				&TestValueReaderWriter{bsontype: TypeDBPointer, err: errors.New("1"), errAfter: llvrwReadDBPointer},
				errors.New("1"),
			},
			{
				"DBPointer/dst/error",
				&TestValueReaderWriter{bsontype: TypeDBPointer, err: errors.New("2"), errAfter: llvrwWriteDBPointer},
				&TestValueReaderWriter{
					bsontype: TypeDBPointer,
					readval: bsoncore.Value{
						Type: bsoncore.TypeDBPointer,
						Data: bsoncore.AppendDBPointer(nil, "foo", ObjectID{0x01, 0x02, 0x03}),
					},
				},
				errors.New("2"),
			},
			{
				"Javascript/src/error",
				&TestValueReaderWriter{},
				&TestValueReaderWriter{bsontype: TypeJavaScript, err: errors.New("1"), errAfter: llvrwReadJavascript},
				errors.New("1"),
			},
			{
				"Javascript/dst/error",
				&TestValueReaderWriter{bsontype: TypeJavaScript, err: errors.New("2"), errAfter: llvrwWriteJavascript},
				&TestValueReaderWriter{bsontype: TypeJavaScript, readval: "hello, world"},
				errors.New("2"),
			},
			{
				"Symbol/src/error",
				&TestValueReaderWriter{},
				&TestValueReaderWriter{bsontype: TypeSymbol, err: errors.New("1"), errAfter: llvrwReadSymbol},
				errors.New("1"),
			},
			{
				"Symbol/dst/error",
				&TestValueReaderWriter{bsontype: TypeSymbol, err: errors.New("2"), errAfter: llvrwWriteSymbol},
				&TestValueReaderWriter{
					bsontype: TypeSymbol,
					readval: bsoncore.Value{
						Type: bsoncore.TypeSymbol,
						Data: bsoncore.AppendSymbol(nil, "hello, world"),
					},
				},
				errors.New("2"),
			},
			{
				"CodeWithScope/src/error",
				&TestValueReaderWriter{},
				&TestValueReaderWriter{bsontype: TypeCodeWithScope, err: errors.New("1"), errAfter: llvrwReadCodeWithScope},
				errors.New("1"),
			},
			{
				"CodeWithScope/dst/error",
				&TestValueReaderWriter{bsontype: TypeCodeWithScope, err: errors.New("2"), errAfter: llvrwWriteCodeWithScope},
				&TestValueReaderWriter{bsontype: TypeCodeWithScope},
				errors.New("2"),
			},
			{
				"CodeWithScope/dst/copyDocumentCore error",
				&TestValueReaderWriter{err: errors.New("3"), errAfter: llvrwWriteDocumentElement},
				&TestValueReaderWriter{bsontype: TypeCodeWithScope},
				errors.New("3"),
			},
			{
				"Int32/src/error",
				&TestValueReaderWriter{},
				&TestValueReaderWriter{bsontype: TypeInt32, err: errors.New("1"), errAfter: llvrwReadInt32},
				errors.New("1"),
			},
			{
				"Int32/dst/error",
				&TestValueReaderWriter{bsontype: TypeInt32, err: errors.New("2"), errAfter: llvrwWriteInt32},
				&TestValueReaderWriter{bsontype: TypeInt32, readval: int32(12345)},
				errors.New("2"),
			},
			{
				"Timestamp/src/error",
				&TestValueReaderWriter{},
				&TestValueReaderWriter{bsontype: TypeTimestamp, err: errors.New("1"), errAfter: llvrwReadTimestamp},
				errors.New("1"),
			},
			{
				"Timestamp/dst/error",
				&TestValueReaderWriter{bsontype: TypeTimestamp, err: errors.New("2"), errAfter: llvrwWriteTimestamp},
				&TestValueReaderWriter{
					bsontype: TypeTimestamp,
					readval: bsoncore.Value{
						Type: bsoncore.TypeTimestamp,
						Data: bsoncore.AppendTimestamp(nil, 12345, 67890),
					},
				},
				errors.New("2"),
			},
			{
				"Int64/src/error",
				&TestValueReaderWriter{},
				&TestValueReaderWriter{bsontype: TypeInt64, err: errors.New("1"), errAfter: llvrwReadInt64},
				errors.New("1"),
			},
			{
				"Int64/dst/error",
				&TestValueReaderWriter{bsontype: TypeInt64, err: errors.New("2"), errAfter: llvrwWriteInt64},
				&TestValueReaderWriter{bsontype: TypeInt64, readval: int64(1234567890)},
				errors.New("2"),
			},
			{
				"Decimal128/src/error",
				&TestValueReaderWriter{},
				&TestValueReaderWriter{bsontype: TypeDecimal128, err: errors.New("1"), errAfter: llvrwReadDecimal128},
				errors.New("1"),
			},
			{
				"Decimal128/dst/error",
				&TestValueReaderWriter{bsontype: TypeDecimal128, err: errors.New("2"), errAfter: llvrwWriteDecimal128},
				&TestValueReaderWriter{bsontype: TypeDecimal128, readval: NewDecimal128(12345, 67890)},
				errors.New("2"),
			},
			{
				"MinKey/src/error",
				&TestValueReaderWriter{},
				&TestValueReaderWriter{bsontype: TypeMinKey, err: errors.New("1"), errAfter: llvrwReadMinKey},
				errors.New("1"),
			},
			{
				"MinKey/dst/error",
				&TestValueReaderWriter{bsontype: TypeMinKey, err: errors.New("2"), errAfter: llvrwWriteMinKey},
				&TestValueReaderWriter{bsontype: TypeMinKey},
				errors.New("2"),
			},
			{
				"MaxKey/src/error",
				&TestValueReaderWriter{},
				&TestValueReaderWriter{bsontype: TypeMaxKey, err: errors.New("1"), errAfter: llvrwReadMaxKey},
				errors.New("1"),
			},
			{
				"MaxKey/dst/error",
				&TestValueReaderWriter{bsontype: TypeMaxKey, err: errors.New("2"), errAfter: llvrwWriteMaxKey},
				&TestValueReaderWriter{bsontype: TypeMaxKey},
				errors.New("2"),
			},
			{
				"Unknown BSON type error",
				&TestValueReaderWriter{},
				&TestValueReaderWriter{},
				fmt.Errorf("cannot copy unknown BSON type %s", Type(0)),
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tc.dst.t, tc.src.t = t, t
				err := copyValue(tc.dst, tc.src)
				if !assert.CompareErrors(err, tc.err) {
					t.Errorf("Did not receive expected error. got %v; want %v", err, tc.err)
				}
			})
		}
	})
	t.Run("CopyValueFromBytes", func(t *testing.T) {
		t.Run("BytesWriter", func(t *testing.T) {
			vw := newValueWriterFromSlice(make([]byte, 0))
			_, err := vw.WriteDocument()
			noerr(t, err)
			_, err = vw.WriteDocumentElement("foo")
			noerr(t, err)
			err = copyValueFromBytes(vw, TypeString, bsoncore.AppendString(nil, "bar"))
			noerr(t, err)
			err = vw.WriteDocumentEnd()
			noerr(t, err)
			var idx int32
			want, err := bsoncore.AppendDocumentEnd(
				bsoncore.AppendStringElement(
					bsoncore.AppendDocumentStartInline(nil, &idx),
					"foo", "bar",
				),
				idx,
			)
			noerr(t, err)
			got := vw.buf
			if !bytes.Equal(got, want) {
				t.Errorf("Bytes are not equal. got %v; want %v", got, want)
			}
		})
		t.Run("Non BytesWriter", func(t *testing.T) {
			llvrw := &TestValueReaderWriter{t: t}
			err := copyValueFromBytes(llvrw, TypeString, bsoncore.AppendString(nil, "bar"))
			noerr(t, err)
			got, want := llvrw.invoked, llvrwWriteString
			if got != want {
				t.Errorf("Incorrect method invoked on llvrw. got %v; want %v", got, want)
			}
		})
	})
	t.Run("CopyValueToBytes", func(t *testing.T) {
		t.Run("BytesReader", func(t *testing.T) {
			var idx int32
			b, err := bsoncore.AppendDocumentEnd(
				bsoncore.AppendStringElement(
					bsoncore.AppendDocumentStartInline(nil, &idx),
					"hello", "world",
				),
				idx,
			)
			noerr(t, err)
			vr := newBufferedDocumentReader(b)
			_, err = vr.ReadDocument()
			noerr(t, err)
			_, _, err = vr.ReadElement()
			noerr(t, err)
			btype, got, err := copyValueToBytes(vr)
			noerr(t, err)
			want := bsoncore.AppendString(nil, "world")
			if btype != TypeString {
				t.Errorf("Incorrect type returned. got %v; want %v", btype, TypeString)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("Bytes do not match. got %v; want %v", got, want)
			}
		})
		t.Run("Non BytesReader", func(t *testing.T) {
			llvrw := &TestValueReaderWriter{t: t, bsontype: TypeString, readval: "Hello, world!"}
			btype, got, err := copyValueToBytes(llvrw)
			noerr(t, err)
			want := bsoncore.AppendString(nil, "Hello, world!")
			if btype != TypeString {
				t.Errorf("Incorrect type returned. got %v; want %v", btype, TypeString)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("Bytes do not match. got %v; want %v", got, want)
			}
		})
	})
	t.Run("AppendValueBytes", func(t *testing.T) {
		t.Run("BytesReader", func(t *testing.T) {
			var idx int32
			b, err := bsoncore.AppendDocumentEnd(
				bsoncore.AppendStringElement(
					bsoncore.AppendDocumentStartInline(nil, &idx),
					"hello", "world",
				),
				idx,
			)
			noerr(t, err)
			vr := newBufferedDocumentReader(b)
			_, err = vr.ReadDocument()
			noerr(t, err)
			_, _, err = vr.ReadElement()
			noerr(t, err)
			btype, got, err := copyValueToBytes(vr)
			noerr(t, err)
			want := bsoncore.AppendString(nil, "world")
			if btype != TypeString {
				t.Errorf("Incorrect type returned. got %v; want %v", btype, TypeString)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("Bytes do not match. got %v; want %v", got, want)
			}
		})
		t.Run("Non BytesReader", func(t *testing.T) {
			llvrw := &TestValueReaderWriter{t: t, bsontype: TypeString, readval: "Hello, world!"}
			btype, got, err := copyValueToBytes(llvrw)
			noerr(t, err)
			want := bsoncore.AppendString(nil, "Hello, world!")
			if btype != TypeString {
				t.Errorf("Incorrect type returned. got %v; want %v", btype, TypeString)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("Bytes do not match. got %v; want %v", got, want)
			}
		})
		t.Run("CopyValue error", func(t *testing.T) {
			want := errors.New("CopyValue error")
			llvrw := &TestValueReaderWriter{t: t, bsontype: TypeString, err: want, errAfter: llvrwReadString}
			_, _, got := copyValueToBytes(llvrw)
			if !assert.CompareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
	})
}
