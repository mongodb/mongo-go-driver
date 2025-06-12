// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

//
//import (
//	"bufio"
//	"bytes"
//	_ "embed"
//	"errors"
//	"fmt"
//	"io"
//	"math"
//	"testing"
//
//	"github.com/google/go-cmp/cmp"
//	"go.mongodb.org/mongo-driver/v2/internal/assert"
//	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
//)
//
////go:embed testdata/lorem.txt
//var lorem []byte
//
//var testcstring = append(lorem, []byte{0x00}...)
//
//func TestValueReader(t *testing.T) {
//	t.Run("ReadBinary", func(t *testing.T) {
//		testCases := []struct {
//			name  string
//			data  []byte
//			btype byte
//			b     []byte
//			err   error
//			vType Type
//		}{
//			{
//				"incorrect type",
//				[]byte{},
//				0,
//				nil,
//				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeBinary),
//				TypeEmbeddedDocument,
//			},
//			{
//				"length too short",
//				[]byte{},
//				0,
//				nil,
//				io.EOF,
//				TypeBinary,
//			},
//			{
//				"no byte available",
//				[]byte{0x00, 0x00, 0x00, 0x00},
//				0,
//				nil,
//				io.EOF,
//				TypeBinary,
//			},
//			{
//				"not enough bytes for binary",
//				[]byte{0x05, 0x00, 0x00, 0x00, 0x00},
//				0,
//				nil,
//				io.EOF,
//				TypeBinary,
//			},
//			{
//				"success",
//				[]byte{0x03, 0x00, 0x00, 0x00, 0xEA, 0x01, 0x02, 0x03},
//				0xEA,
//				[]byte{0x01, 0x02, 0x03},
//				nil,
//				TypeBinary,
//			},
//		}
//
//		for _, tc := range testCases {
//			t.Run(tc.name, func(t *testing.T) {
//				vr := &valueReader{
//					r: bufio.NewReader(bytes.NewReader(tc.data)),
//					stack: []vrState{
//						{mode: mTopLevel},
//						{
//							mode:  mElement,
//							vType: tc.vType,
//						},
//					},
//					frame: 1,
//				}
//
//				b, btype, err := vr.ReadBinary()
//				if !errequal(t, err, tc.err) {
//					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
//				}
//				if btype != tc.btype {
//					t.Errorf("Incorrect binary type returned. got %v; want %v", btype, tc.btype)
//				}
//				if !bytes.Equal(b, tc.b) {
//					t.Errorf("Binary data does not match. got %v; want %v", b, tc.b)
//				}
//			})
//		}
//	})
//	t.Run("ReadBoolean", func(t *testing.T) {
//		testCases := []struct {
//			name    string
//			data    []byte
//			boolean bool
//			err     error
//			vType   Type
//		}{
//			{
//				"incorrect type",
//				[]byte{},
//				false,
//				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeBoolean),
//				TypeEmbeddedDocument,
//			},
//			{
//				"no byte available",
//				[]byte{},
//				false,
//				io.EOF,
//				TypeBoolean,
//			},
//			{
//				"invalid byte for boolean",
//				[]byte{0x03},
//				false,
//				fmt.Errorf("invalid byte for boolean, %b", 0x03),
//				TypeBoolean,
//			},
//			{
//				"success",
//				[]byte{0x01},
//				true,
//				nil,
//				TypeBoolean,
//			},
//		}
//
//		for _, tc := range testCases {
//			t.Run(tc.name, func(t *testing.T) {
//				vr := &valueReader{
//					r: bufio.NewReader(bytes.NewReader(tc.data)),
//					stack: []vrState{
//						{mode: mTopLevel},
//						{
//							mode:  mElement,
//							vType: tc.vType,
//						},
//					},
//					frame: 1,
//				}
//
//				boolean, err := vr.ReadBoolean()
//				if !errequal(t, err, tc.err) {
//					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
//				}
//				if boolean != tc.boolean {
//					t.Errorf("Incorrect boolean returned. got %v; want %v", boolean, tc.boolean)
//				}
//			})
//		}
//	})
//	t.Run("ReadDocument", func(t *testing.T) {
//		t.Run("TopLevel", func(t *testing.T) {
//			doc := []byte{0x05, 0x00, 0x00, 0x00, 0x00}
//			vr := &valueReader{
//				stack: []vrState{{mode: mTopLevel}},
//				frame: 0,
//			}
//
//			// invalid length
//			vr.r = bufio.NewReader(bytes.NewReader([]byte{0x00, 0x00}))
//			_, err := vr.ReadDocument()
//			if !errors.Is(err, io.ErrUnexpectedEOF) {
//				t.Errorf("Expected io.ErrUnexpectedEOF with document length too small. got %v; want %v", err, io.EOF)
//			}
//			if vr.offset != 0 {
//				t.Errorf("Expected 0 offset. got %d", vr.offset)
//			}
//
//			vr.r = bufio.NewReader(bytes.NewReader(doc))
//			_, err = vr.ReadDocument()
//			noerr(t, err)
//			if vr.stack[vr.frame].end != 5 {
//				t.Errorf("Incorrect end for document. got %d; want %d", vr.stack[vr.frame].end, 5)
//			}
//		})
//		t.Run("EmbeddedDocument", func(t *testing.T) {
//			vr := &valueReader{
//				stack: []vrState{
//					{mode: mTopLevel},
//					{mode: mElement, vType: TypeBoolean},
//				},
//				frame: 1,
//			}
//
//			var wanterr = (&valueReader{stack: []vrState{{mode: mElement, vType: TypeBoolean}}}).typeError(TypeEmbeddedDocument)
//			_, err := vr.ReadDocument()
//			if err == nil || err.Error() != wanterr.Error() {
//				t.Errorf("Incorrect returned error. got %v; want %v", err, wanterr)
//			}
//
//			vr.stack[1].mode = mArray
//			wanterr = vr.invalidTransitionErr(mDocument, "ReadDocument", []mode{mTopLevel, mElement, mValue})
//			_, err = vr.ReadDocument()
//			if err == nil || err.Error() != wanterr.Error() {
//				t.Errorf("Incorrect returned error. got %v; want %v", err, wanterr)
//			}
//
//			vr.stack[1].mode, vr.stack[1].vType = mElement, TypeEmbeddedDocument
//			vr.offset = 4
//			vr.r = bufio.NewReader(bytes.NewReader([]byte{0x05, 0x00, 0x00, 0x00, 0x00, 0x00}))
//			_, err = vr.ReadDocument()
//			noerr(t, err)
//			if len(vr.stack) != 3 {
//				t.Errorf("Incorrect number of stack frames. got %d; want %d", len(vr.stack), 3)
//			}
//			if vr.stack[2].mode != mDocument {
//				t.Errorf("Incorrect mode set. got %v; want %v", vr.stack[2].mode, mDocument)
//			}
//			if vr.stack[2].end != 9 {
//				t.Errorf("End of embedded document is not correct. got %d; want %d", vr.stack[2].end, 9)
//			}
//			if vr.offset != 8 {
//				t.Errorf("Offset not incremented correctly. got %d; want %d", vr.offset, 8)
//			}
//
//			vr.frame--
//			_, err = vr.ReadDocument()
//			if !errors.Is(err, io.ErrUnexpectedEOF) {
//				t.Errorf("Should return error when attempting to read length with not enough bytes. got %v; want %v", err, io.EOF)
//			}
//		})
//	})
//	t.Run("ReadCodeWithScope", func(t *testing.T) {
//		codeWithScope := []byte{
//			0x11, 0x00, 0x00, 0x00, // total length
//			0x4, 0x00, 0x00, 0x00, // string length
//			'f', 'o', 'o', 0x00, // string
//			0x05, 0x00, 0x00, 0x00, 0x00, // document
//		}
//		mismatchCodeWithScope := []byte{
//			0x11, 0x00, 0x00, 0x00, // total length
//			0x4, 0x00, 0x00, 0x00, // string length
//			'f', 'o', 'o', 0x00, // string
//			0x07, 0x00, 0x00, 0x00, // document
//			0x0A, 0x00, // null element, empty key
//			0x00, // document end
//		}
//		invalidCodeWithScope := []byte{
//			0x7, 0x00, 0x00, 0x00, // total length
//			0x0, 0x00, 0x00, 0x00, // string length = 0
//			0x05, 0x00, 0x00, 0x00, 0x00, // document
//		}
//		testCases := []struct {
//			name  string
//			data  []byte
//			err   error
//			vType Type
//		}{
//			{
//				"incorrect type",
//				[]byte{},
//				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeCodeWithScope),
//				TypeEmbeddedDocument,
//			},
//			{
//				"total length not enough bytes",
//				[]byte{},
//				io.EOF,
//				TypeCodeWithScope,
//			},
//			{
//				"string length not enough bytes",
//				codeWithScope[:4],
//				io.EOF,
//				TypeCodeWithScope,
//			},
//			{
//				"not enough string bytes",
//				codeWithScope[:8],
//				io.EOF,
//				TypeCodeWithScope,
//			},
//			{
//				"document length not enough bytes",
//				codeWithScope[:12],
//				io.EOF,
//				TypeCodeWithScope,
//			},
//			{
//				"length mismatch",
//				mismatchCodeWithScope,
//				fmt.Errorf("length of CodeWithScope does not match lengths of components; total: %d; components: %d", 17, 19),
//				TypeCodeWithScope,
//			},
//			{
//				"invalid strLength",
//				invalidCodeWithScope,
//				fmt.Errorf("invalid string length: %d", 0),
//				TypeCodeWithScope,
//			},
//		}
//
//		for _, tc := range testCases {
//			t.Run(tc.name, func(t *testing.T) {
//				vr := &valueReader{
//					r: bufio.NewReader(bytes.NewReader(tc.data)),
//					stack: []vrState{
//						{mode: mTopLevel},
//						{
//							mode:  mElement,
//							vType: tc.vType,
//						},
//					},
//					frame: 1,
//				}
//
//				_, _, err := vr.ReadCodeWithScope()
//				if !errequal(t, err, tc.err) {
//					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
//				}
//			})
//		}
//
//		t.Run("success", func(t *testing.T) {
//			vr := &valueReader{
//				offset: 4,
//				r:      bufio.NewReader(bytes.NewReader(codeWithScope)),
//				stack: []vrState{
//					{mode: mTopLevel},
//					{mode: mElement, vType: TypeCodeWithScope},
//				},
//				frame: 1,
//			}
//
//			code, _, err := vr.ReadCodeWithScope()
//			noerr(t, err)
//			if code != "foo" {
//				t.Errorf("Code does not match. got %s; want %s", code, "foo")
//			}
//			if len(vr.stack) != 3 {
//				t.Errorf("Incorrect number of stack frames. got %d; want %d", len(vr.stack), 3)
//			}
//			if vr.stack[2].mode != mCodeWithScope {
//				t.Errorf("Incorrect mode set. got %v; want %v", vr.stack[2].mode, mDocument)
//			}
//			if vr.stack[2].end != 21 {
//				t.Errorf("End of scope is not correct. got %d; want %d", vr.stack[2].end, 21)
//			}
//			if vr.offset != 20 {
//				t.Errorf("Offset not incremented correctly. got %d; want %d", vr.offset, 20)
//			}
//		})
//	})
//	t.Run("ReadDBPointer", func(t *testing.T) {
//		testCases := []struct {
//			name  string
//			data  []byte
//			ns    string
//			oid   ObjectID
//			err   error
//			vType Type
//		}{
//			{
//				"incorrect type",
//				[]byte{},
//				"",
//				ObjectID{},
//				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeDBPointer),
//				TypeEmbeddedDocument,
//			},
//			{
//				"length too short",
//				[]byte{},
//				"",
//				ObjectID{},
//				io.EOF,
//				TypeDBPointer,
//			},
//			{
//				"not enough bytes for namespace",
//				[]byte{0x04, 0x00, 0x00, 0x00},
//				"",
//				ObjectID{},
//				io.EOF,
//				TypeDBPointer,
//			},
//			{
//				"not enough bytes for objectID",
//				[]byte{0x04, 0x00, 0x00, 0x00, 'f', 'o', 'o', 0x00},
//				"",
//				ObjectID{},
//				io.EOF,
//				TypeDBPointer,
//			},
//			{
//				"success",
//				[]byte{
//					0x04, 0x00, 0x00, 0x00, 'f', 'o', 'o', 0x00,
//					0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C,
//				},
//				"foo",
//				ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
//				nil,
//				TypeDBPointer,
//			},
//		}
//
//		for _, tc := range testCases {
//			t.Run(tc.name, func(t *testing.T) {
//				vr := &valueReader{
//					r: bufio.NewReader(bytes.NewReader(tc.data)),
//					stack: []vrState{
//						{mode: mTopLevel},
//						{
//							mode:  mElement,
//							vType: tc.vType,
//						},
//					},
//					frame: 1,
//				}
//
//				ns, oid, err := vr.ReadDBPointer()
//				if !errequal(t, err, tc.err) {
//					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
//				}
//				if ns != tc.ns {
//					t.Errorf("Incorrect namespace returned. got %v; want %v", ns, tc.ns)
//				}
//				if oid != tc.oid {
//					t.Errorf("ObjectIDs did not match. got %v; want %v", oid, tc.oid)
//				}
//			})
//		}
//	})
//	t.Run("ReadDateTime", func(t *testing.T) {
//		testCases := []struct {
//			name  string
//			data  []byte
//			dt    int64
//			err   error
//			vType Type
//		}{
//			{
//				"incorrect type",
//				[]byte{},
//				0,
//				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeDateTime),
//				TypeEmbeddedDocument,
//			},
//			{
//				"length too short",
//				[]byte{},
//				0,
//				io.EOF,
//				TypeDateTime,
//			},
//			{
//				"success",
//				[]byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
//				255,
//				nil,
//				TypeDateTime,
//			},
//		}
//
//		for _, tc := range testCases {
//			t.Run(tc.name, func(t *testing.T) {
//				vr := &valueReader{
//					r: bufio.NewReader(bytes.NewReader(tc.data)),
//					stack: []vrState{
//						{mode: mTopLevel},
//						{
//							mode:  mElement,
//							vType: tc.vType,
//						},
//					},
//					frame: 1,
//				}
//
//				dt, err := vr.ReadDateTime()
//				if !errequal(t, err, tc.err) {
//					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
//				}
//				if dt != tc.dt {
//					t.Errorf("Incorrect datetime returned. got %d; want %d", dt, tc.dt)
//				}
//			})
//		}
//	})
//	t.Run("ReadDecimal128", func(t *testing.T) {
//		testCases := []struct {
//			name  string
//			data  []byte
//			dc128 Decimal128
//			err   error
//			vType Type
//		}{
//			{
//				"incorrect type",
//				[]byte{},
//				Decimal128{},
//				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeDecimal128),
//				TypeEmbeddedDocument,
//			},
//			{
//				"length too short",
//				[]byte{},
//				Decimal128{},
//				io.EOF,
//				TypeDecimal128,
//			},
//			{
//				"success",
//				[]byte{
//					0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Low
//					0x00, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // High
//				},
//				NewDecimal128(65280, 255),
//				nil,
//				TypeDecimal128,
//			},
//		}
//
//		for _, tc := range testCases {
//			t.Run(tc.name, func(t *testing.T) {
//				vr := &valueReader{
//					r: bufio.NewReader(bytes.NewReader(tc.data)),
//					stack: []vrState{
//						{mode: mTopLevel},
//						{
//							mode:  mElement,
//							vType: tc.vType,
//						},
//					},
//					frame: 1,
//				}
//
//				dc128, err := vr.ReadDecimal128()
//				if !errequal(t, err, tc.err) {
//					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
//				}
//				gotHigh, gotLow := dc128.GetBytes()
//				wantHigh, wantLow := tc.dc128.GetBytes()
//				if gotHigh != wantHigh {
//					t.Errorf("Retuired high byte does not match. got %d; want %d", gotHigh, wantHigh)
//				}
//				if gotLow != wantLow {
//					t.Errorf("Returned low byte does not match. got %d; want %d", gotLow, wantLow)
//				}
//			})
//		}
//	})
//	t.Run("ReadDouble", func(t *testing.T) {
//		testCases := []struct {
//			name   string
//			data   []byte
//			double float64
//			err    error
//			vType  Type
//		}{
//			{
//				"incorrect type",
//				[]byte{},
//				0,
//				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeDouble),
//				TypeEmbeddedDocument,
//			},
//			{
//				"length too short",
//				[]byte{},
//				0,
//				io.EOF,
//				TypeDouble,
//			},
//			{
//				"success",
//				[]byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
//				math.Float64frombits(255),
//				nil,
//				TypeDouble,
//			},
//		}
//
//		for _, tc := range testCases {
//			t.Run(tc.name, func(t *testing.T) {
//				vr := &valueReader{
//					r: bufio.NewReader(bytes.NewReader(tc.data)),
//					stack: []vrState{
//						{mode: mTopLevel},
//						{
//							mode:  mElement,
//							vType: tc.vType,
//						},
//					},
//					frame: 1,
//				}
//
//				double, err := vr.ReadDouble()
//				if !errequal(t, err, tc.err) {
//					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
//				}
//				if double != tc.double {
//					t.Errorf("Incorrect double returned. got %f; want %f", double, tc.double)
//				}
//			})
//		}
//	})
//	t.Run("ReadInt32", func(t *testing.T) {
//		testCases := []struct {
//			name  string
//			data  []byte
//			i32   int32
//			err   error
//			vType Type
//		}{
//			{
//				"incorrect type",
//				[]byte{},
//				0,
//				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeInt32),
//				TypeEmbeddedDocument,
//			},
//			{
//				"length too short",
//				[]byte{},
//				0,
//				io.EOF,
//				TypeInt32,
//			},
//			{
//				"success",
//				[]byte{0xFF, 0x00, 0x00, 0x00},
//				255,
//				nil,
//				TypeInt32,
//			},
//		}
//
//		for _, tc := range testCases {
//			t.Run(tc.name, func(t *testing.T) {
//				vr := &valueReader{
//					r: bufio.NewReader(bytes.NewReader(tc.data)),
//					stack: []vrState{
//						{mode: mTopLevel},
//						{
//							mode:  mElement,
//							vType: tc.vType,
//						},
//					},
//					frame: 1,
//				}
//
//				i32, err := vr.ReadInt32()
//				if !errequal(t, err, tc.err) {
//					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
//				}
//				if i32 != tc.i32 {
//					t.Errorf("Incorrect int32 returned. got %d; want %d", i32, tc.i32)
//				}
//			})
//		}
//	})
//	t.Run("ReadInt64", func(t *testing.T) {
//		testCases := []struct {
//			name  string
//			data  []byte
//			i64   int64
//			err   error
//			vType Type
//		}{
//			{
//				"incorrect type",
//				[]byte{},
//				0,
//				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeInt64),
//				TypeEmbeddedDocument,
//			},
//			{
//				"length too short",
//				[]byte{},
//				0,
//				io.EOF,
//				TypeInt64,
//			},
//			{
//				"success",
//				[]byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
//				255,
//				nil,
//				TypeInt64,
//			},
//		}
//
//		for _, tc := range testCases {
//			t.Run(tc.name, func(t *testing.T) {
//				vr := &valueReader{
//					r: bufio.NewReader(bytes.NewReader(tc.data)),
//					stack: []vrState{
//						{mode: mTopLevel},
//						{
//							mode:  mElement,
//							vType: tc.vType,
//						},
//					},
//					frame: 1,
//				}
//
//				i64, err := vr.ReadInt64()
//				if !errequal(t, err, tc.err) {
//					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
//				}
//				if i64 != tc.i64 {
//					t.Errorf("Incorrect int64 returned. got %d; want %d", i64, tc.i64)
//				}
//			})
//		}
//	})
//	t.Run("ReadJavascript/ReadString/ReadSymbol", func(t *testing.T) {
//		testCases := []struct {
//			name  string
//			data  []byte
//			fn    func(*valueReader) (string, error)
//			css   string // code, string, symbol :P
//			err   error
//			vType Type
//		}{
//			{
//				"ReadJavascript/incorrect type",
//				[]byte{},
//				(*valueReader).ReadJavascript,
//				"",
//				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeJavaScript),
//				TypeEmbeddedDocument,
//			},
//			{
//				"ReadString/incorrect type",
//				[]byte{},
//				(*valueReader).ReadString,
//				"",
//				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeString),
//				TypeEmbeddedDocument,
//			},
//			{
//				"ReadSymbol/incorrect type",
//				[]byte{},
//				(*valueReader).ReadSymbol,
//				"",
//				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeSymbol),
//				TypeEmbeddedDocument,
//			},
//			{
//				"ReadJavascript/length too short",
//				[]byte{},
//				(*valueReader).ReadJavascript,
//				"",
//				io.EOF,
//				TypeJavaScript,
//			},
//			{
//				"ReadString/length too short",
//				[]byte{},
//				(*valueReader).ReadString,
//				"",
//				io.EOF,
//				TypeString,
//			},
//			{
//				"ReadString/long - length too short",
//				append([]byte{0x40, 0x27, 0x00, 0x00}, testcstring...),
//				(*valueReader).ReadString,
//				"",
//				io.ErrUnexpectedEOF,
//				TypeString,
//			},
//			{
//				"ReadSymbol/length too short",
//				[]byte{},
//				(*valueReader).ReadSymbol,
//				"",
//				io.EOF,
//				TypeSymbol,
//			},
//			{
//				"ReadJavascript/incorrect end byte",
//				[]byte{0x01, 0x00, 0x00, 0x00, 0x05},
//				(*valueReader).ReadJavascript,
//				"",
//				fmt.Errorf("string does not end with null byte, but with %v", 0x05),
//				TypeJavaScript,
//			},
//			{
//				"ReadString/incorrect end byte",
//				[]byte{0x01, 0x00, 0x00, 0x00, 0x05},
//				(*valueReader).ReadString,
//				"",
//				fmt.Errorf("string does not end with null byte, but with %v", 0x05),
//				TypeString,
//			},
//			{
//				"ReadString/long - incorrect end byte",
//				append([]byte{0x35, 0x27, 0x00, 0x00}, testcstring[:len(testcstring)-1]...),
//				(*valueReader).ReadString,
//				"",
//				fmt.Errorf("string does not end with null byte, but with %v", 0x20),
//				TypeString,
//			},
//			{
//				"ReadSymbol/incorrect end byte",
//				[]byte{0x01, 0x00, 0x00, 0x00, 0x05},
//				(*valueReader).ReadSymbol,
//				"",
//				fmt.Errorf("string does not end with null byte, but with %v", 0x05),
//				TypeSymbol,
//			},
//			{
//				"ReadJavascript/success",
//				[]byte{0x04, 0x00, 0x00, 0x00, 'f', 'o', 'o', 0x00},
//				(*valueReader).ReadJavascript,
//				"foo",
//				nil,
//				TypeJavaScript,
//			},
//			{
//				"ReadString/success",
//				[]byte{0x04, 0x00, 0x00, 0x00, 'f', 'o', 'o', 0x00},
//				(*valueReader).ReadString,
//				"foo",
//				nil,
//				TypeString,
//			},
//			{
//				"ReadString/long - success",
//				append([]byte{0x36, 0x27, 0x00, 0x00}, testcstring...),
//				(*valueReader).ReadString,
//				string(testcstring[:len(testcstring)-1]),
//				nil,
//				TypeString,
//			},
//			{
//				"ReadSymbol/success",
//				[]byte{0x04, 0x00, 0x00, 0x00, 'f', 'o', 'o', 0x00},
//				(*valueReader).ReadSymbol,
//				"foo",
//				nil,
//				TypeSymbol,
//			},
//		}
//
//		for _, tc := range testCases {
//			t.Run(tc.name, func(t *testing.T) {
//				vr := &valueReader{
//					r: bufio.NewReader(bytes.NewReader(tc.data)),
//					stack: []vrState{
//						{mode: mTopLevel},
//						{
//							mode:  mElement,
//							vType: tc.vType,
//						},
//					},
//					frame: 1,
//				}
//
//				css, err := tc.fn(vr)
//				if !errequal(t, err, tc.err) {
//					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
//				}
//				if css != tc.css {
//					t.Errorf("Incorrect (JavaScript,String,Symbol) returned. got %s; want %s", css, tc.css)
//				}
//			})
//		}
//	})
//	t.Run("ReadMaxKey/ReadMinKey/ReadNull/ReadUndefined", func(t *testing.T) {
//		testCases := []struct {
//			name  string
//			fn    func(*valueReader) error
//			err   error
//			vType Type
//		}{
//			{
//				"ReadMaxKey/incorrect type",
//				(*valueReader).ReadMaxKey,
//				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeMaxKey),
//				TypeEmbeddedDocument,
//			},
//			{
//				"ReadMaxKey/success",
//				(*valueReader).ReadMaxKey,
//				nil,
//				TypeMaxKey,
//			},
//			{
//				"ReadMinKey/incorrect type",
//				(*valueReader).ReadMinKey,
//				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeMinKey),
//				TypeEmbeddedDocument,
//			},
//			{
//				"ReadMinKey/success",
//				(*valueReader).ReadMinKey,
//				nil,
//				TypeMinKey,
//			},
//			{
//				"ReadNull/incorrect type",
//				(*valueReader).ReadNull,
//				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeNull),
//				TypeEmbeddedDocument,
//			},
//			{
//				"ReadNull/success",
//				(*valueReader).ReadNull,
//				nil,
//				TypeNull,
//			},
//			{
//				"ReadUndefined/incorrect type",
//				(*valueReader).ReadUndefined,
//				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeUndefined),
//				TypeEmbeddedDocument,
//			},
//			{
//				"ReadUndefined/success",
//				(*valueReader).ReadUndefined,
//				nil,
//				TypeUndefined,
//			},
//		}
//
//		for _, tc := range testCases {
//			t.Run(tc.name, func(t *testing.T) {
//				vr := &valueReader{
//					stack: []vrState{
//						{mode: mTopLevel},
//						{
//							mode:  mElement,
//							vType: tc.vType,
//						},
//					},
//					frame: 1,
//				}
//
//				err := tc.fn(vr)
//				if !errequal(t, err, tc.err) {
//					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
//				}
//			})
//		}
//	})
//	t.Run("ReadObjectID", func(t *testing.T) {
//		testCases := []struct {
//			name  string
//			data  []byte
//			oid   ObjectID
//			err   error
//			vType Type
//		}{
//			{
//				"incorrect type",
//				[]byte{},
//				ObjectID{},
//				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeObjectID),
//				TypeEmbeddedDocument,
//			},
//			{
//				"not enough bytes for objectID",
//				[]byte{},
//				ObjectID{},
//				io.EOF,
//				TypeObjectID,
//			},
//			{
//				"success",
//				[]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
//				ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
//				nil,
//				TypeObjectID,
//			},
//		}
//
//		for _, tc := range testCases {
//			t.Run(tc.name, func(t *testing.T) {
//				vr := &valueReader{
//					r: bufio.NewReader(bytes.NewReader(tc.data)),
//					stack: []vrState{
//						{mode: mTopLevel},
//						{
//							mode:  mElement,
//							vType: tc.vType,
//						},
//					},
//					frame: 1,
//				}
//
//				oid, err := vr.ReadObjectID()
//				if !errequal(t, err, tc.err) {
//					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
//				}
//				if oid != tc.oid {
//					t.Errorf("ObjectIDs did not match. got %v; want %v", oid, tc.oid)
//				}
//			})
//		}
//	})
//	t.Run("ReadRegex", func(t *testing.T) {
//		testCases := []struct {
//			name    string
//			data    []byte
//			pattern string
//			options string
//			err     error
//			vType   Type
//		}{
//			{
//				"incorrect type",
//				[]byte{},
//				"",
//				"",
//				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeRegex),
//				TypeEmbeddedDocument,
//			},
//			{
//				"length too short",
//				[]byte{},
//				"",
//				"",
//				io.EOF,
//				TypeRegex,
//			},
//			{
//				"not enough bytes for options",
//				[]byte{'f', 'o', 'o', 0x00},
//				"",
//				"",
//				io.EOF,
//				TypeRegex,
//			},
//			{
//				"success",
//				[]byte{'f', 'o', 'o', 0x00, 'b', 'a', 'r', 0x00},
//				"foo",
//				"bar",
//				nil,
//				TypeRegex,
//			},
//		}
//
//		for _, tc := range testCases {
//			t.Run(tc.name, func(t *testing.T) {
//				vr := &valueReader{
//					r: bufio.NewReader(bytes.NewReader(tc.data)),
//					stack: []vrState{
//						{mode: mTopLevel},
//						{
//							mode:  mElement,
//							vType: tc.vType,
//						},
//					},
//					frame: 1,
//				}
//
//				pattern, options, err := vr.ReadRegex()
//				if !errequal(t, err, tc.err) {
//					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
//				}
//				if pattern != tc.pattern {
//					t.Errorf("Incorrect pattern returned. got %s; want %s", pattern, tc.pattern)
//				}
//				if options != tc.options {
//					t.Errorf("Incorrect options returned. got %s; want %s", options, tc.options)
//				}
//			})
//		}
//	})
//	t.Run("ReadTimestamp", func(t *testing.T) {
//		testCases := []struct {
//			name  string
//			data  []byte
//			ts    uint32
//			incr  uint32
//			err   error
//			vType Type
//		}{
//			{
//				"incorrect type",
//				[]byte{},
//				0,
//				0,
//				(&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeTimestamp),
//				TypeEmbeddedDocument,
//			},
//			{
//				"not enough bytes for increment",
//				[]byte{},
//				0,
//				0,
//				io.EOF,
//				TypeTimestamp,
//			},
//			{
//				"not enough bytes for timestamp",
//				[]byte{0x01, 0x02, 0x03, 0x04},
//				0,
//				0,
//				io.EOF,
//				TypeTimestamp,
//			},
//			{
//				"success",
//				[]byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00},
//				256,
//				255,
//				nil,
//				TypeTimestamp,
//			},
//		}
//
//		for _, tc := range testCases {
//			t.Run(tc.name, func(t *testing.T) {
//				vr := &valueReader{
//					r: bufio.NewReader(bytes.NewReader(tc.data)),
//					stack: []vrState{
//						{mode: mTopLevel},
//						{
//							mode:  mElement,
//							vType: tc.vType,
//						},
//					},
//					frame: 1,
//				}
//
//				ts, incr, err := vr.ReadTimestamp()
//				if !errequal(t, err, tc.err) {
//					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
//				}
//				if ts != tc.ts {
//					t.Errorf("Incorrect timestamp returned. got %d; want %d", ts, tc.ts)
//				}
//				if incr != tc.incr {
//					t.Errorf("Incorrect increment returned. got %d; want %d", incr, tc.incr)
//				}
//			})
//		}
//	})
//
//	t.Run("ReadBytes & Skip", func(t *testing.T) {
//		index, docb := bsoncore.ReserveLength(nil)
//		docb = bsoncore.AppendNullElement(docb, "foobar")
//		docb = append(docb, 0x00)
//		docb = bsoncore.UpdateLength(docb, index, int32(len(docb)))
//		cwsbytes := bsoncore.AppendCodeWithScope(nil, "var hellow = world;", docb)
//		strbytes := []byte{0x04, 0x00, 0x00, 0x00, 'f', 'o', 'o', 0x00}
//		testCases := []struct {
//			name           string
//			t              Type
//			data           []byte
//			err            error
//			offset         int64
//			startingOffset int64
//		}{
//			{
//				"Array/invalid length",
//				TypeArray,
//				[]byte{0x01, 0x02, 0x03},
//				io.EOF, 0, 0,
//			},
//			{
//				"Array/not enough bytes",
//				TypeArray,
//				[]byte{0x0F, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03},
//				io.EOF, 0, 0,
//			},
//			{
//				"Array/success",
//				TypeArray,
//				[]byte{0x08, 0x00, 0x00, 0x00, 0x0A, '1', 0x00, 0x00},
//				nil, 8, 0,
//			},
//			{
//				"EmbeddedDocument/invalid length",
//				TypeEmbeddedDocument,
//				[]byte{0x01, 0x02, 0x03},
//				io.EOF, 0, 0,
//			},
//			{
//				"EmbeddedDocument/not enough bytes",
//				TypeEmbeddedDocument,
//				[]byte{0x0F, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03},
//				io.EOF, 0, 0,
//			},
//			{
//				"EmbeddedDocument/success",
//				TypeEmbeddedDocument,
//				[]byte{0x08, 0x00, 0x00, 0x00, 0x0A, 'A', 0x00, 0x00},
//				nil, 8, 0,
//			},
//			{
//				"CodeWithScope/invalid length",
//				TypeCodeWithScope,
//				[]byte{0x01, 0x02, 0x03},
//				io.EOF, 0, 0,
//			},
//			{
//				"CodeWithScope/not enough bytes",
//				TypeCodeWithScope,
//				[]byte{0x0F, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03},
//				io.EOF, 0, 0,
//			},
//			{
//				"CodeWithScope/success",
//				TypeCodeWithScope,
//				cwsbytes,
//				nil, 41, 0,
//			},
//			{
//				"Binary/invalid length",
//				TypeBinary,
//				[]byte{0x01, 0x02, 0x03},
//				io.EOF, 0, 0,
//			},
//			{
//				"Binary/not enough bytes",
//				TypeBinary,
//				[]byte{0x0F, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03},
//				io.EOF, 0, 0,
//			},
//			{
//				"Binary/success",
//				TypeBinary,
//				[]byte{0x03, 0x00, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03},
//				nil, 8, 0,
//			},
//			{
//				"Boolean/invalid length",
//				TypeBoolean,
//				[]byte{},
//				io.EOF, 0, 0,
//			},
//			{
//				"Boolean/success",
//				TypeBoolean,
//				[]byte{0x01},
//				nil, 1, 0,
//			},
//			{
//				"DBPointer/invalid length",
//				TypeDBPointer,
//				[]byte{0x01, 0x02, 0x03},
//				io.EOF, 0, 0,
//			},
//			{
//				"DBPointer/not enough bytes",
//				TypeDBPointer,
//				[]byte{0x0F, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03},
//				io.EOF, 0, 0,
//			},
//			{
//				"DBPointer/success",
//				TypeDBPointer,
//				[]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
//				nil, 17, 0,
//			},
//			{"DBPointer/not enough bytes", TypeDateTime, []byte{0x01, 0x02, 0x03, 0x04}, io.EOF, 0, 0},
//			{"DBPointer/success", TypeDateTime, []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, nil, 8, 0},
//			{"Double/not enough bytes", TypeDouble, []byte{0x01, 0x02, 0x03, 0x04}, io.EOF, 0, 0},
//			{"Double/success", TypeDouble, []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, nil, 8, 0},
//			{"Int64/not enough bytes", TypeInt64, []byte{0x01, 0x02, 0x03, 0x04}, io.EOF, 0, 0},
//			{"Int64/success", TypeInt64, []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, nil, 8, 0},
//			{"Timestamp/not enough bytes", TypeTimestamp, []byte{0x01, 0x02, 0x03, 0x04}, io.EOF, 0, 0},
//			{"Timestamp/success", TypeTimestamp, []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, nil, 8, 0},
//			{
//				"Decimal128/not enough bytes",
//				TypeDecimal128,
//				[]byte{0x01, 0x02, 0x03, 0x04},
//				io.EOF, 0, 0,
//			},
//			{
//				"Decimal128/success",
//				TypeDecimal128,
//				[]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
//				nil, 16, 0,
//			},
//			{"Int32/not enough bytes", TypeInt32, []byte{0x01, 0x02}, io.EOF, 0, 0},
//			{"Int32/success", TypeInt32, []byte{0x01, 0x02, 0x03, 0x04}, nil, 4, 0},
//			{"Javascript/invalid length", TypeJavaScript, strbytes[:2], io.EOF, 0, 0},
//			{"Javascript/not enough bytes", TypeJavaScript, strbytes[:5], io.EOF, 0, 0},
//			{"Javascript/success", TypeJavaScript, strbytes, nil, 8, 0},
//			{"String/invalid length", TypeString, strbytes[:2], io.EOF, 0, 0},
//			{"String/not enough bytes", TypeString, strbytes[:5], io.EOF, 0, 0},
//			{"String/success", TypeString, strbytes, nil, 8, 0},
//			{"Symbol/invalid length", TypeSymbol, strbytes[:2], io.EOF, 0, 0},
//			{"Symbol/not enough bytes", TypeSymbol, strbytes[:5], io.EOF, 0, 0},
//			{"Symbol/success", TypeSymbol, strbytes, nil, 8, 0},
//			{"MaxKey/success", TypeMaxKey, []byte{}, nil, 0, 0},
//			{"MinKey/success", TypeMinKey, []byte{}, nil, 0, 0},
//			{"Null/success", TypeNull, []byte{}, nil, 0, 0},
//			{"Undefined/success", TypeUndefined, []byte{}, nil, 0, 0},
//			{
//				"ObjectID/not enough bytes",
//				TypeObjectID,
//				[]byte{0x01, 0x02, 0x03, 0x04},
//				io.EOF, 0, 0,
//			},
//			{
//				"ObjectID/success",
//				TypeObjectID,
//				[]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
//				nil, 12, 0,
//			},
//			{
//				"Regex/not enough bytes (first string)",
//				TypeRegex,
//				[]byte{'f', 'o', 'o'},
//				io.EOF, 0, 0,
//			},
//			{
//				"Regex/not enough bytes (second string)",
//				TypeRegex,
//				[]byte{'f', 'o', 'o', 0x00, 'b', 'a', 'r'},
//				io.EOF, 0, 0,
//			},
//			{
//				"Regex/success",
//				TypeRegex,
//				[]byte{0x00, 0x00, 0x00, 'f', 'o', 'o', 0x00, 'i', 0x00},
//				nil, 9, 3,
//			},
//			{
//				"Unknown Type",
//				Type(0),
//				nil,
//				fmt.Errorf("attempted to read bytes of unknown BSON type %v", Type(0)), 0, 0,
//			},
//		}
//
//		for _, tc := range testCases {
//			t.Run(tc.name, func(t *testing.T) {
//				const startingEnd = 64
//				t.Run("Skip", func(t *testing.T) {
//					vr := &valueReader{
//						r: bufio.NewReader(bytes.NewReader(tc.data[tc.startingOffset:tc.offset])),
//						stack: []vrState{
//							{mode: mTopLevel, end: startingEnd},
//							{mode: mElement, vType: tc.t},
//						},
//						frame:  1,
//						offset: tc.startingOffset,
//					}
//
//					err := vr.Skip()
//					if !errequal(t, err, tc.err) {
//						t.Errorf("Did not receive expected error; got %v; want %v", err, tc.err)
//					}
//					if tc.err == nil {
//						offset := startingEnd - vr.stack[0].end
//						if offset != tc.offset {
//							t.Errorf("Offset not set at correct position; got %d; want %d", offset, tc.offset)
//						}
//					}
//				})
//				t.Run("ReadBytes", func(t *testing.T) {
//					vr := &valueReader{
//						r: bufio.NewReader(bytes.NewReader(tc.data[tc.startingOffset:tc.offset])),
//						stack: []vrState{
//							{mode: mTopLevel, end: startingEnd},
//							{mode: mElement, vType: tc.t},
//						},
//						frame:  1,
//						offset: tc.startingOffset,
//					}
//
//					_, got, err := vr.readValueBytes(nil)
//					if !errequal(t, err, tc.err) {
//						t.Errorf("Did not receive expected error; got %v; want %v", err, tc.err)
//					}
//					if tc.err == nil {
//						offset := startingEnd - vr.stack[0].end
//						if offset != tc.offset {
//							t.Errorf("Offset not set at correct position; got %d; want %d", vr.offset, tc.offset)
//						}
//					}
//					if tc.err == nil && !bytes.Equal(got, tc.data[tc.startingOffset:]) {
//						t.Errorf("Did not receive expected bytes. got %v; want %v", got, tc.data[tc.startingOffset:])
//					}
//				})
//			})
//		}
//		t.Run("ReadValueBytes/Top Level Doc", func(t *testing.T) {
//			testCases := []struct {
//				name     string
//				want     []byte
//				wantType Type
//				wantErr  error
//			}{
//				{
//					"success",
//					bsoncore.BuildDocument(nil, bsoncore.AppendDoubleElement(nil, "pi", 3.14159)),
//					Type(0),
//					nil,
//				},
//				{
//					"wrong length",
//					[]byte{0x01, 0x02, 0x03},
//					Type(0),
//					io.EOF,
//				},
//				{
//					"append bytes",
//					[]byte{0x01, 0x02, 0x03, 0x04},
//					Type(0),
//					io.ErrUnexpectedEOF,
//				},
//			}
//
//			for _, tc := range testCases {
//				tc := tc
//				t.Run(tc.name, func(t *testing.T) {
//					t.Parallel()
//					vr := &valueReader{
//						r: bufio.NewReader(bytes.NewReader(tc.want)),
//						stack: []vrState{
//							{mode: mTopLevel},
//						},
//						frame: 0,
//					}
//					gotType, got, gotErr := vr.readValueBytes(nil)
//					if !errors.Is(gotErr, tc.wantErr) {
//						t.Errorf("Did not receive expected error. got %v; want %v", gotErr, tc.wantErr)
//					}
//					if tc.wantErr == nil && gotType != tc.wantType {
//						t.Errorf("Did not receive expected type. got %v; want %v", gotType, tc.wantType)
//					}
//					if tc.wantErr == nil && !bytes.Equal(got, tc.want) {
//						t.Errorf("Did not receive expected bytes. got %v; want %v", got, tc.want)
//					}
//				})
//			}
//		})
//	})
//
//	t.Run("invalid transition", func(t *testing.T) {
//		t.Run("Skip", func(t *testing.T) {
//			vr := &valueReader{stack: []vrState{{mode: mTopLevel}}}
//			wanterr := (&valueReader{stack: []vrState{{mode: mTopLevel}}}).invalidTransitionErr(0, "Skip", []mode{mElement, mValue})
//			goterr := vr.Skip()
//			if !cmp.Equal(goterr, wanterr, cmp.Comparer(assert.CompareErrors)) {
//				t.Errorf("Expected correct invalid transition error. got %v; want %v", goterr, wanterr)
//			}
//		})
//	})
//	t.Run("ReadBytes", func(t *testing.T) {
//		vr := &valueReader{stack: []vrState{{mode: mTopLevel}, {mode: mDocument}}, frame: 1}
//		wanterr := (&valueReader{stack: []vrState{{mode: mTopLevel}, {mode: mDocument}}, frame: 1}).
//			invalidTransitionErr(0, "ReadValueBytes", []mode{mElement, mValue})
//		_, _, goterr := vr.readValueBytes(nil)
//		if !cmp.Equal(goterr, wanterr, cmp.Comparer(assert.CompareErrors)) {
//			t.Errorf("Expected correct invalid transition error. got %v; want %v", goterr, wanterr)
//		}
//	})
//}
//
//func errequal(t *testing.T, err1, err2 error) bool {
//	t.Helper()
//	if err1 == nil && err2 == nil { // If they are both nil, they are equal
//		return true
//	}
//	if err1 == nil || err2 == nil { // If only one is nil, they are not equal
//		return false
//	}
//
//	if errors.Is(err1, err2) { // They are the same error, they are equal
//		return true
//	}
//
//	if err1.Error() == err2.Error() { // They string formats match, they are equal
//		return true
//	}
//
//	return false
//}
