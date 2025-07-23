// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bufio"
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

//go:embed testdata/lorem.txt
var lorem []byte

var testcstring = append(lorem, []byte{0x00}...)

func TestValueReader_ReadBinary(t *testing.T) {
	testCases := []struct {
		name  string
		data  []byte
		btype byte
		b     []byte
		err   error
		vType Type
	}{
		{
			name:  "incorrect type",
			data:  []byte{},
			btype: 0,
			b:     nil,
			err:   (&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeBinary),
			vType: TypeEmbeddedDocument,
		},
		{
			name:  "length too short",
			data:  []byte{},
			btype: 0,
			b:     nil,
			err:   io.EOF,
			vType: TypeBinary,
		},
		{
			name:  "no byte available",
			data:  []byte{0x00, 0x00, 0x00, 0x00},
			btype: 0,
			b:     nil,
			err:   io.EOF,
			vType: TypeBinary,
		},
		{
			name:  "not enough bytes for binary",
			data:  []byte{0x05, 0x00, 0x00, 0x00, 0x00},
			btype: 0,
			b:     nil,
			err:   io.EOF,
			vType: TypeBinary,
		},
		{
			name:  "success",
			data:  []byte{0x03, 0x00, 0x00, 0x00, 0xEA, 0x01, 0x02, 0x03},
			btype: 0xEA,
			b:     []byte{0x01, 0x02, 0x03},
			err:   nil,
			vType: TypeBinary,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("buffered", func(t *testing.T) {
				vr := &valueReader{
					src: &bufferedByteSrc{buf: tc.data},
					stack: []vrState{
						{mode: mTopLevel},
						{
							mode:  mElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}

				b, btype, err := vr.ReadBinary()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if btype != tc.btype {
					t.Errorf("Incorrect binary type returned. got %v; want %v", btype, tc.btype)
				}
				if !bytes.Equal(b, tc.b) {
					t.Errorf("Binary data does not match. got %v; want %v", b, tc.b)
				}
			})

			t.Run("streaming", func(t *testing.T) {
				vr := &valueReader{
					src: &streamingValueReader{br: bufio.NewReader(bytes.NewReader(tc.data))},
					stack: []vrState{
						{mode: mTopLevel},
						{
							mode:  mElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}

				b, btype, err := vr.ReadBinary()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if btype != tc.btype {
					t.Errorf("Incorrect binary type returned. got %v; want %v", btype, tc.btype)
				}
				if !bytes.Equal(b, tc.b) {
					t.Errorf("Binary data does not match. got %v; want %v", b, tc.b)
				}
			})
		})
	}
}

func TestValueReader_ReadBoolean(t *testing.T) {
	testCases := []struct {
		name    string
		data    []byte
		boolean bool
		err     error
		vType   Type
	}{
		{
			name:    "incorrect type",
			data:    []byte{},
			boolean: false,
			err:     (&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeBoolean),
			vType:   TypeEmbeddedDocument,
		},
		{
			name:    "no byte available",
			data:    []byte{},
			boolean: false,
			err:     io.EOF,
			vType:   TypeBoolean,
		},
		{
			name:    "invalid byte for boolean",
			data:    []byte{0x03},
			boolean: false,
			err:     fmt.Errorf("invalid byte for boolean, %b", 0x03),
			vType:   TypeBoolean,
		},
		{
			name:    "success",
			data:    []byte{0x01},
			boolean: true,
			err:     nil,
			vType:   TypeBoolean,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("buffered", func(t *testing.T) {
				vr := &valueReader{
					src: &bufferedByteSrc{buf: tc.data},
					stack: []vrState{
						{mode: mTopLevel},
						{
							mode:  mElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}

				boolean, err := vr.ReadBoolean()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if boolean != tc.boolean {
					t.Errorf("Incorrect boolean returned. got %v; want %v", boolean, tc.boolean)
				}
			})

			t.Run("streaming", func(t *testing.T) {
				vr := &valueReader{
					src: &streamingValueReader{br: bufio.NewReader(bytes.NewReader(tc.data))},
					stack: []vrState{
						{mode: mTopLevel},
						{
							mode:  mElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}

				boolean, err := vr.ReadBoolean()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if boolean != tc.boolean {
					t.Errorf("Incorrect boolean returned. got %v; want %v", boolean, tc.boolean)
				}
			})
		})
	}
}

func TestValueReader_ReadDocument_TopLevel_InvalidLength(t *testing.T) {
	t.Run("buffered", func(t *testing.T) {
		vr := &valueReader{
			src:   &bufferedByteSrc{buf: []byte{0x00, 0x00}},
			stack: []vrState{{mode: mTopLevel}},
			frame: 0,
		}

		_, err := vr.ReadDocument()
		if !errors.Is(err, io.EOF) {
			t.Errorf("Expected io.ErrUnexpectedEOF with document length too small. got %v; want %v", err, io.EOF)
		}
		if vr.src.pos() != 0 {
			t.Errorf("Expected 0 offset. got %d", vr.src.pos())
		}
	})

	t.Run("streaming", func(t *testing.T) {
		vr := &valueReader{
			src:   &streamingValueReader{br: bufio.NewReader(bytes.NewReader([]byte{0x00, 0x00}))},
			stack: []vrState{{mode: mTopLevel}},
			frame: 0,
		}

		_, err := vr.ReadDocument()
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Errorf("Expected io.ErrUnexpectedEOF with document length too small. got %v; want %v", err, io.EOF)
		}
		if vr.src.pos() != 0 {
			t.Errorf("Expected 0 offset. got %d", vr.src.pos())
		}
	})
}

func TestValueReader_ReadDocument_TopLevel_ValidDocumentWithIncorrectEnd(t *testing.T) {
	t.Run("buffered", func(t *testing.T) {
		vr := &valueReader{
			src:   &bufferedByteSrc{buf: []byte{0x05, 0x00, 0x00, 0x00, 0x00}},
			stack: []vrState{{mode: mTopLevel}},
			frame: 0,
		}

		_, err := vr.ReadDocument()
		noerr(t, err)
		if vr.stack[vr.frame].end != 5 {
			t.Errorf("Incorrect end for document. got %d; want %d", vr.stack[vr.frame].end, 5)
		}
	})

	t.Run("streaming", func(t *testing.T) {

		vr := &valueReader{
			src:   &streamingValueReader{br: bufio.NewReader(bytes.NewReader([]byte{0x05, 0x00, 0x00, 0x00, 0x00}))},
			stack: []vrState{{mode: mTopLevel}},
			frame: 0,
		}

		_, err := vr.ReadDocument()
		noerr(t, err)
		if vr.stack[vr.frame].end != 5 {
			t.Errorf("Incorrect end for document. got %d; want %d", vr.stack[vr.frame].end, 5)
		}
	})
}

func TestValueReader_ReadDocument_EmbeddedDocument(t *testing.T) {
	t.Run("buffered", func(t *testing.T) {
		vr := &valueReader{
			src: &bufferedByteSrc{buf: []byte{0x0a, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00}},
			stack: []vrState{
				{mode: mTopLevel},
				{mode: mElement, vType: TypeBoolean},
			},
			frame: 1,
		}

		var wanterr = (&valueReader{stack: []vrState{{mode: mElement, vType: TypeBoolean}}}).typeError(TypeEmbeddedDocument)
		_, err := vr.ReadDocument()
		if err == nil || err.Error() != wanterr.Error() {
			t.Errorf("Incorrect returned error. got %v; want %v", err, wanterr)
		}

		vr.stack[1].mode = mArray
		wanterr = vr.invalidTransitionErr(mDocument, "ReadDocument", []mode{mTopLevel, mElement, mValue})
		_, err = vr.ReadDocument()
		if err == nil || err.Error() != wanterr.Error() {
			t.Errorf("Incorrect returned error. got %v; want %v", err, wanterr)
		}

		vr.stack[1].mode, vr.stack[1].vType = mElement, TypeEmbeddedDocument
		_, _ = vr.src.discard(4)

		_, err = vr.ReadDocument()
		noerr(t, err)
		if len(vr.stack) != 3 {
			t.Errorf("Incorrect number of stack frames. got %d; want %d", len(vr.stack), 3)
		}
		if vr.stack[2].mode != mDocument {
			t.Errorf("Incorrect mode set. got %v; want %v", vr.stack[2].mode, mDocument)
		}
		if vr.stack[2].end != 9 {
			t.Errorf("End of embedded document is not correct. got %d; want %d", vr.stack[2].end, 9)
		}
		if vr.src.pos() != 8 {
			t.Errorf("Offset not incremented correctly. got %d; want %d", vr.src.pos(), 8)
		}

		vr.frame--
		_, err = vr.ReadDocument()

		if !errors.Is(err, io.EOF) {
			t.Errorf("Should return error when attempting to read length with not enough bytes. got %v; want %v", err, io.EOF)
		}
	})

	t.Run("streaming", func(t *testing.T) {
		vr := &valueReader{
			src: &streamingValueReader{
				br: bufio.NewReader(bytes.NewReader([]byte{0x0a, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00})),
			},
			stack: []vrState{
				{mode: mTopLevel},
				{mode: mElement, vType: TypeBoolean},
			},
			frame: 1,
		}

		var wanterr = (&valueReader{stack: []vrState{{mode: mElement, vType: TypeBoolean}}}).typeError(TypeEmbeddedDocument)
		_, err := vr.ReadDocument()
		if err == nil || err.Error() != wanterr.Error() {
			t.Errorf("Incorrect returned error. got %v; want %v", err, wanterr)
		}

		vr.stack[1].mode = mArray
		wanterr = vr.invalidTransitionErr(mDocument, "ReadDocument", []mode{mTopLevel, mElement, mValue})
		_, err = vr.ReadDocument()
		if err == nil || err.Error() != wanterr.Error() {
			t.Errorf("Incorrect returned error. got %v; want %v", err, wanterr)
		}

		vr.stack[1].mode, vr.stack[1].vType = mElement, TypeEmbeddedDocument
		_, _ = vr.src.discard(4)

		_, err = vr.ReadDocument()
		noerr(t, err)
		if len(vr.stack) != 3 {
			t.Errorf("Incorrect number of stack frames. got %d; want %d", len(vr.stack), 3)
		}
		if vr.stack[2].mode != mDocument {
			t.Errorf("Incorrect mode set. got %v; want %v", vr.stack[2].mode, mDocument)
		}
		if vr.stack[2].end != 9 {
			t.Errorf("End of embedded document is not correct. got %d; want %d", vr.stack[2].end, 9)
		}
		if vr.src.pos() != 8 {
			t.Errorf("Offset not incremented correctly. got %d; want %d", vr.src.pos(), 8)
		}

		vr.frame--
		_, err = vr.ReadDocument()

		if !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Errorf("Should return error when attempting to read length with not enough bytes. got %v; want %v", err, io.EOF)
		}
	})
}

func TestValueReader_ReadCodeWithScope(t *testing.T) {
	codeWithScope := []byte{
		0x11, 0x00, 0x00, 0x00, // total length
		0x4, 0x00, 0x00, 0x00, // string length
		'f', 'o', 'o', 0x00, // string
		0x05, 0x00, 0x00, 0x00, 0x00, // document
	}
	mismatchCodeWithScope := []byte{
		0x11, 0x00, 0x00, 0x00, // total length
		0x4, 0x00, 0x00, 0x00, // string length
		'f', 'o', 'o', 0x00, // string
		0x07, 0x00, 0x00, 0x00, // document
		0x0A, 0x00, // null element, empty key
		0x00, // document end
	}
	invalidCodeWithScope := []byte{
		0x7, 0x00, 0x00, 0x00, // total length
		0x0, 0x00, 0x00, 0x00, // string length = 0
		0x05, 0x00, 0x00, 0x00, 0x00, // document
	}

	testCases := []struct {
		name  string
		data  []byte
		err   error
		vType Type
	}{
		{
			name:  "incorrect type",
			data:  []byte{},
			err:   (&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeCodeWithScope),
			vType: TypeEmbeddedDocument,
		},
		{
			name:  "total length not enough bytes",
			data:  []byte{},
			err:   io.EOF,
			vType: TypeCodeWithScope,
		},
		{
			name:  "string length not enough bytes",
			data:  codeWithScope[:4],
			err:   io.EOF,
			vType: TypeCodeWithScope,
		},
		{
			name:  "not enough string bytes",
			data:  codeWithScope[:8],
			err:   io.EOF,
			vType: TypeCodeWithScope,
		},
		{
			name:  "document length not enough bytes",
			data:  codeWithScope[:12],
			err:   io.EOF,
			vType: TypeCodeWithScope,
		},
		{
			name:  "length mismatch",
			data:  mismatchCodeWithScope,
			err:   fmt.Errorf("length of CodeWithScope does not match lengths of components; total: %d; components: %d", 17, 19),
			vType: TypeCodeWithScope,
		},
		{
			name:  "invalid strLength",
			data:  invalidCodeWithScope,
			err:   fmt.Errorf("invalid string length: %d", 0),
			vType: TypeCodeWithScope,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("buffered", func(t *testing.T) {
				vr := &valueReader{
					src: &bufferedByteSrc{buf: tc.data},
					stack: []vrState{
						{mode: mTopLevel},
						{
							mode:  mElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}

				_, _, err := vr.ReadCodeWithScope()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
			})
		})

		t.Run("streaming", func(t *testing.T) {
			vr := &valueReader{
				src: &streamingValueReader{br: bufio.NewReader(bytes.NewReader(tc.data))},
				stack: []vrState{
					{mode: mTopLevel},
					{
						mode:  mElement,
						vType: tc.vType,
					},
				},
				frame: 1,
			}

			_, _, err := vr.ReadCodeWithScope()
			if !errequal(t, err, tc.err) {
				t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
			}
		})
	}

	t.Run("buffered success", func(t *testing.T) {
		doc := []byte{0x00, 0x00, 0x00, 0x00}
		doc = append(doc, codeWithScope...)
		doc = append(doc, 0x00)
		vr := &valueReader{
			src: &bufferedByteSrc{buf: doc},
			stack: []vrState{
				{mode: mTopLevel},
				{mode: mElement, vType: TypeCodeWithScope},
			},
			frame: 1,
		}
		_, _ = vr.src.discard(4) // discard the document length

		code, _, err := vr.ReadCodeWithScope()
		noerr(t, err)
		if code != "foo" {
			t.Errorf("Code does not match. got %s; want %s", code, "foo")
		}
		if len(vr.stack) != 3 {
			t.Errorf("Incorrect number of stack frames. got %d; want %d", len(vr.stack), 3)
		}
		if vr.stack[2].mode != mCodeWithScope {
			t.Errorf("Incorrect mode set. got %v; want %v", vr.stack[2].mode, mDocument)
		}
		if vr.stack[2].end != 21 {
			t.Errorf("End of scope is not correct. got %d; want %d", vr.stack[2].end, 21)
		}
		if vr.src.pos() != 20 {
			t.Errorf("Offset not incremented correctly. got %d; want %d", vr.src.pos(), 20)
		}
	})

	t.Run("streaming success", func(t *testing.T) {
		doc := []byte{0x00, 0x00, 0x00, 0x00}
		doc = append(doc, codeWithScope...)
		doc = append(doc, 0x00)
		vr := &valueReader{
			src: &streamingValueReader{br: bufio.NewReader(bytes.NewReader(doc))},
			stack: []vrState{
				{mode: mTopLevel},
				{mode: mElement, vType: TypeCodeWithScope},
			},
			frame: 1,
		}
		_, _ = vr.src.discard(4) // discard the document length

		code, _, err := vr.ReadCodeWithScope()
		noerr(t, err)
		if code != "foo" {
			t.Errorf("Code does not match. got %s; want %s", code, "foo")
		}
		if len(vr.stack) != 3 {
			t.Errorf("Incorrect number of stack frames. got %d; want %d", len(vr.stack), 3)
		}
		if vr.stack[2].mode != mCodeWithScope {
			t.Errorf("Incorrect mode set. got %v; want %v", vr.stack[2].mode, mDocument)
		}
		if vr.stack[2].end != 21 {
			t.Errorf("End of scope is not correct. got %d; want %d", vr.stack[2].end, 21)
		}
		if vr.src.pos() != 20 {
			t.Errorf("Offset not incremented correctly. got %d; want %d", vr.src.pos(), 20)
		}
	})
}

func TestValueReader_ReadDBPointer(t *testing.T) {
	testCases := []struct {
		name  string
		data  []byte
		ns    string
		oid   ObjectID
		err   error
		vType Type
	}{
		{
			name:  "incorrect type",
			data:  []byte{},
			ns:    "",
			oid:   ObjectID{},
			err:   (&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeDBPointer),
			vType: TypeEmbeddedDocument,
		},
		{
			name:  "length too short",
			data:  []byte{},
			ns:    "",
			oid:   ObjectID{},
			err:   io.EOF,
			vType: TypeDBPointer,
		},
		{
			name:  "not enough bytes for namespace",
			data:  []byte{0x04, 0x00, 0x00, 0x00},
			ns:    "",
			oid:   ObjectID{},
			err:   io.EOF,
			vType: TypeDBPointer,
		},
		{
			name:  "not enough bytes for objectID",
			data:  []byte{0x04, 0x00, 0x00, 0x00, 'f', 'o', 'o', 0x00},
			ns:    "",
			oid:   ObjectID{},
			err:   io.EOF,
			vType: TypeDBPointer,
		},
		{
			name: "success",
			data: []byte{
				0x04, 0x00, 0x00, 0x00, 'f', 'o', 'o', 0x00,
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C,
			},
			ns:    "foo",
			oid:   ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
			err:   nil,
			vType: TypeDBPointer,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("buffered", func(t *testing.T) {
				vr := &valueReader{
					src: &bufferedByteSrc{buf: tc.data},
					stack: []vrState{
						{mode: mTopLevel},
						{
							mode:  mElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}

				ns, oid, err := vr.ReadDBPointer()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if ns != tc.ns {
					t.Errorf("Incorrect namespace returned. got %v; want %v", ns, tc.ns)
				}
				if oid != tc.oid {
					t.Errorf("ObjectIDs did not match. got %v; want %v", oid, tc.oid)
				}
			})

			t.Run("streaming", func(t *testing.T) {
				vr := &valueReader{
					src: &streamingValueReader{br: bufio.NewReader(bytes.NewReader(tc.data))},
					stack: []vrState{
						{mode: mTopLevel},
						{
							mode:  mElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}

				ns, oid, err := vr.ReadDBPointer()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if ns != tc.ns {
					t.Errorf("Incorrect namespace returned. got %v; want %v", ns, tc.ns)
				}
				if oid != tc.oid {
					t.Errorf("ObjectIDs did not match. got %v; want %v", oid, tc.oid)
				}
			})
		})
	}
}

func TestValueReader_ReadDateTime(t *testing.T) {
	testCases := []struct {
		name  string
		data  []byte
		dt    int64
		err   error
		vType Type
	}{
		{
			name:  "incorrect type",
			data:  []byte{},
			dt:    0,
			err:   (&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeDateTime),
			vType: TypeEmbeddedDocument,
		},
		{
			name:  "length too short",
			data:  []byte{},
			dt:    0,
			err:   io.EOF,
			vType: TypeDateTime,
		},
		{
			name:  "success",
			data:  []byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			dt:    255,
			err:   nil,
			vType: TypeDateTime,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("buffered", func(t *testing.T) {
				vr := &valueReader{
					src: &bufferedByteSrc{buf: tc.data},
					stack: []vrState{
						{mode: mTopLevel},
						{
							mode:  mElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}

				dt, err := vr.ReadDateTime()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if dt != tc.dt {
					t.Errorf("Incorrect datetime returned. got %d; want %d", dt, tc.dt)
				}
			})

			t.Run("streaming", func(t *testing.T) {
				vr := &valueReader{
					src: &streamingValueReader{br: bufio.NewReader(bytes.NewReader(tc.data))},
					stack: []vrState{
						{mode: mTopLevel},
						{
							mode:  mElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}

				dt, err := vr.ReadDateTime()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if dt != tc.dt {
					t.Errorf("Incorrect datetime returned. got %d; want %d", dt, tc.dt)
				}
			})
		})
	}
}

func TestValueReader_ReadDecimal128(t *testing.T) {
	testCases := []struct {
		name  string
		data  []byte
		dc128 Decimal128
		err   error
		vType Type
	}{
		{
			name:  "incorrect type",
			data:  []byte{},
			dc128: Decimal128{},
			err:   (&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeDecimal128),
			vType: TypeEmbeddedDocument,
		},
		{
			name:  "length too short",
			data:  []byte{},
			dc128: Decimal128{},
			err:   io.EOF,
			vType: TypeDecimal128,
		},
		{
			name: "success",
			data: []byte{
				0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Low
				0x00, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // High
			},
			dc128: NewDecimal128(65280, 255),
			err:   nil,
			vType: TypeDecimal128,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("buffered", func(t *testing.T) {
				vr := &valueReader{
					src: &bufferedByteSrc{buf: tc.data},
					stack: []vrState{
						{mode: mTopLevel},
						{
							mode:  mElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}

				dc128, err := vr.ReadDecimal128()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				gotHigh, gotLow := dc128.GetBytes()
				wantHigh, wantLow := tc.dc128.GetBytes()
				if gotHigh != wantHigh {
					t.Errorf("Returned high byte does not match. got %d; want %d", gotHigh, wantHigh)
				}
				if gotLow != wantLow {
					t.Errorf("Returned low byte does not match. got %d; want %d", gotLow, wantLow)
				}
			})

			t.Run("streaming", func(t *testing.T) {
				vr := &valueReader{
					src: &streamingValueReader{br: bufio.NewReader(bytes.NewReader(tc.data))},
					stack: []vrState{
						{mode: mTopLevel},
						{
							mode:  mElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}

				dc128, err := vr.ReadDecimal128()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				gotHigh, gotLow := dc128.GetBytes()
				wantHigh, wantLow := tc.dc128.GetBytes()
				if gotHigh != wantHigh {
					t.Errorf("Returned high byte does not match. got %d; want %d", gotHigh, wantHigh)
				}
				if gotLow != wantLow {
					t.Errorf("Returned low byte does not match. got %d; want %d", gotLow, wantLow)
				}
			})
		})
	}
}

func TestValueReader_ReadDouble(t *testing.T) {
	testCases := []struct {
		name   string
		data   []byte
		double float64
		err    error
		vType  Type
	}{
		{
			name:   "incorrect type",
			data:   []byte{},
			double: 0,
			err:    (&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeDouble),
			vType:  TypeEmbeddedDocument,
		},
		{
			name:   "length too short",
			data:   []byte{},
			double: 0,
			err:    io.EOF,
			vType:  TypeDouble,
		},
		{
			name:   "success",
			data:   []byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			double: math.Float64frombits(255),
			err:    nil,
			vType:  TypeDouble,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("buffered", func(t *testing.T) {
				vr := &valueReader{
					src: &bufferedByteSrc{buf: tc.data},
					stack: []vrState{
						{mode: mTopLevel},
						{
							mode:  mElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}

				double, err := vr.ReadDouble()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if double != tc.double {
					t.Errorf("Incorrect double returned. got %f; want %f", double, tc.double)
				}
			})

			t.Run("streaming", func(t *testing.T) {
				vr := &valueReader{
					src: &streamingValueReader{br: bufio.NewReader(bytes.NewReader(tc.data))},
					stack: []vrState{
						{mode: mTopLevel},
						{
							mode:  mElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}

				double, err := vr.ReadDouble()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if double != tc.double {
					t.Errorf("Incorrect double returned. got %f; want %f", double, tc.double)
				}
			})
		})
	}
}

func TestValueReader_ReadInt32(t *testing.T) {
	testCases := []struct {
		name  string
		data  []byte
		i32   int32
		err   error
		vType Type
	}{
		{
			name:  "incorrect type",
			data:  []byte{},
			i32:   0,
			err:   (&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeInt32),
			vType: TypeEmbeddedDocument,
		},
		{
			name:  "length too short",
			data:  []byte{},
			i32:   0,
			err:   io.EOF,
			vType: TypeInt32,
		},
		{
			name:  "success",
			data:  []byte{0xFF, 0x00, 0x00, 0x00},
			i32:   255,
			err:   nil,
			vType: TypeInt32,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("buffered", func(t *testing.T) {
				vr := &valueReader{
					src: &bufferedByteSrc{buf: tc.data},
					stack: []vrState{
						{mode: mTopLevel},
						{
							mode:  mElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}

				i32, err := vr.ReadInt32()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if i32 != tc.i32 {
					t.Errorf("Incorrect int32 returned. got %d; want %d", i32, tc.i32)
				}
			})

			t.Run("streaming", func(t *testing.T) {
				vr := &valueReader{
					src: &streamingValueReader{br: bufio.NewReader(bytes.NewReader(tc.data))},
					stack: []vrState{
						{mode: mTopLevel},
						{
							mode:  mElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}

				i32, err := vr.ReadInt32()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if i32 != tc.i32 {
					t.Errorf("Incorrect int32 returned. got %d; want %d", i32, tc.i32)
				}
			})
		})
	}
}

func TestValueReader_ReadInt64(t *testing.T) {
	testCases := []struct {
		name  string
		data  []byte
		i64   int64
		err   error
		vType Type
	}{
		{
			name:  "incorrect type",
			data:  []byte{},
			i64:   0,
			err:   (&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeInt64),
			vType: TypeEmbeddedDocument,
		},
		{
			name:  "length too short",
			data:  []byte{},
			i64:   0,
			err:   io.EOF,
			vType: TypeInt64,
		},
		{
			name:  "success",
			data:  []byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			i64:   255,
			err:   nil,
			vType: TypeInt64,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("buffered", func(t *testing.T) {
				vr := &valueReader{
					src: &bufferedByteSrc{buf: tc.data},
					stack: []vrState{
						{mode: mTopLevel},
						{
							mode:  mElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}

				i64, err := vr.ReadInt64()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if i64 != tc.i64 {
					t.Errorf("Incorrect int64 returned. got %d; want %d", i64, tc.i64)
				}
			})

			t.Run("streaming", func(t *testing.T) {
				vr := &valueReader{
					src: &streamingValueReader{br: bufio.NewReader(bytes.NewReader(tc.data))},
					stack: []vrState{
						{mode: mTopLevel},
						{
							mode:  mElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}

				i64, err := vr.ReadInt64()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if i64 != tc.i64 {
					t.Errorf("Incorrect int64 returned. got %d; want %d", i64, tc.i64)
				}
			})
		})
	}
}

func TestValueReader_ReadJavascript_ReadString_ReadSymbol(t *testing.T) {
	testCases := []struct {
		name          string
		data          []byte
		fn            func(*valueReader) (string, error)
		css           string // code, string, symbol
		streamingErr  error
		bufferedError error
		vType         Type
	}{
		{
			name:          "ReadJavascript/incorrect type",
			data:          []byte{},
			fn:            (*valueReader).ReadJavascript,
			css:           "",
			streamingErr:  (&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeJavaScript),
			bufferedError: (&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeJavaScript),
			vType:         TypeEmbeddedDocument,
		},
		{
			name:          "ReadString/incorrect type",
			data:          []byte{},
			fn:            (*valueReader).ReadString,
			css:           "",
			streamingErr:  (&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeString),
			bufferedError: (&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeString),
			vType:         TypeEmbeddedDocument,
		},
		{
			name:          "ReadSymbol/incorrect type",
			data:          []byte{},
			fn:            (*valueReader).ReadSymbol,
			css:           "",
			streamingErr:  (&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeSymbol),
			bufferedError: (&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeSymbol),
			vType:         TypeEmbeddedDocument,
		},
		{
			name:          "ReadJavascript/length too short",
			data:          []byte{},
			fn:            (*valueReader).ReadJavascript,
			css:           "",
			streamingErr:  io.EOF,
			bufferedError: io.EOF,
			vType:         TypeJavaScript,
		},
		{
			name:          "ReadString/length too short",
			data:          []byte{},
			fn:            (*valueReader).ReadString,
			css:           "",
			streamingErr:  io.EOF,
			bufferedError: io.EOF,
			vType:         TypeString,
		},
		{
			name:          "ReadString/long - length too short",
			data:          append([]byte{0x40, 0x27, 0x00, 0x00}, testcstring...),
			fn:            (*valueReader).ReadString,
			css:           "",
			streamingErr:  io.ErrUnexpectedEOF,
			bufferedError: io.EOF,
			vType:         TypeString,
		},
		{
			name:          "ReadSymbol/length too short",
			data:          []byte{},
			fn:            (*valueReader).ReadSymbol,
			css:           "",
			streamingErr:  io.EOF,
			bufferedError: io.EOF,
			vType:         TypeSymbol,
		},
		{
			name:          "ReadJavascript/incorrect end byte",
			data:          []byte{0x01, 0x00, 0x00, 0x00, 0x05},
			fn:            (*valueReader).ReadJavascript,
			css:           "",
			streamingErr:  fmt.Errorf("string does not end with null byte, but with %v", 0x05),
			bufferedError: fmt.Errorf("string does not end with null byte, but with %v", 0x05),
			vType:         TypeJavaScript,
		},
		{
			name:          "ReadString/incorrect end byte",
			data:          []byte{0x01, 0x00, 0x00, 0x00, 0x05},
			fn:            (*valueReader).ReadString,
			css:           "",
			streamingErr:  fmt.Errorf("string does not end with null byte, but with %v", 0x05),
			bufferedError: fmt.Errorf("string does not end with null byte, but with %v", 0x05),
			vType:         TypeString,
		},
		{
			name:          "ReadString/long - incorrect end byte",
			data:          append([]byte{0x35, 0x27, 0x00, 0x00}, testcstring[:len(testcstring)-1]...),
			fn:            (*valueReader).ReadString,
			css:           "",
			streamingErr:  fmt.Errorf("string does not end with null byte, but with %v", 0x20),
			bufferedError: fmt.Errorf("string does not end with null byte, but with %v", 0x20),
			vType:         TypeString,
		},
		{
			name:          "ReadSymbol/incorrect end byte",
			data:          []byte{0x01, 0x00, 0x00, 0x00, 0x05},
			fn:            (*valueReader).ReadSymbol,
			css:           "",
			streamingErr:  fmt.Errorf("string does not end with null byte, but with %v", 0x05),
			bufferedError: fmt.Errorf("string does not end with null byte, but with %v", 0x05),
			vType:         TypeSymbol,
		},
		{
			name:          "ReadJavascript/success",
			data:          []byte{0x04, 0x00, 0x00, 0x00, 'f', 'o', 'o', 0x00},
			fn:            (*valueReader).ReadJavascript,
			css:           "foo",
			streamingErr:  nil,
			bufferedError: nil,
			vType:         TypeJavaScript,
		},
		{
			name:          "ReadString/success",
			data:          []byte{0x04, 0x00, 0x00, 0x00, 'f', 'o', 'o', 0x00},
			fn:            (*valueReader).ReadString,
			css:           "foo",
			streamingErr:  nil,
			bufferedError: nil,
			vType:         TypeString,
		},
		{
			name:          "ReadString/long - success",
			data:          append([]byte{0x36, 0x27, 0x00, 0x00}, testcstring...),
			fn:            (*valueReader).ReadString,
			css:           string(testcstring[:len(testcstring)-1]),
			streamingErr:  nil,
			bufferedError: nil,
			vType:         TypeString,
		},
		{
			name:          "ReadSymbol/success",
			data:          []byte{0x04, 0x00, 0x00, 0x00, 'f', 'o', 'o', 0x00},
			fn:            (*valueReader).ReadSymbol,
			css:           "foo",
			streamingErr:  nil,
			bufferedError: nil,
			vType:         TypeSymbol,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("buffered", func(t *testing.T) {
				vr := &valueReader{
					src: &bufferedByteSrc{buf: tc.data},
					stack: []vrState{
						{mode: mTopLevel},
						{
							mode:  mElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}
				wantErr := tc.bufferedError

				css, err := tc.fn(vr)
				if !errequal(t, err, wantErr) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, wantErr)
				}
				if css != tc.css {
					t.Errorf("Incorrect (JavaScript,String,Symbol) returned. got %s; want %s", css, tc.css)
				}
			})

			t.Run("streaming", func(t *testing.T) {
				vr := &valueReader{
					src: &streamingValueReader{br: bufio.NewReader(bytes.NewReader(tc.data))},
					stack: []vrState{
						{mode: mTopLevel},
						{
							mode:  mElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}
				wantErr := tc.streamingErr

				css, err := tc.fn(vr)
				if !errequal(t, err, wantErr) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, wantErr)
				}
				if css != tc.css {
					t.Errorf("Incorrect (JavaScript,String,Symbol) returned. got %s; want %s", css, tc.css)
				}
			})
		})
	}
}

func TestValueReader_ReadMaxKey_ReadMinKey_ReadNull_ReadUndefined(t *testing.T) {
	testCases := []struct {
		name         string
		fn           func(*valueReader) error
		streamingErr error
		bufferedErr  error
		vType        Type
	}{
		{
			name:         "ReadMaxKey/incorrect type",
			fn:           (*valueReader).ReadMaxKey,
			streamingErr: (&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeMaxKey),
			bufferedErr:  (&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeMaxKey),
			vType:        TypeEmbeddedDocument,
		},
		{
			name:         "ReadMaxKey/success",
			fn:           (*valueReader).ReadMaxKey,
			streamingErr: nil,
			bufferedErr:  nil,
			vType:        TypeMaxKey,
		},
		{
			name:         "ReadMinKey/incorrect type",
			fn:           (*valueReader).ReadMinKey,
			streamingErr: (&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeMinKey),
			bufferedErr:  (&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeMinKey),
			vType:        TypeEmbeddedDocument,
		},
		{
			name:         "ReadMinKey/success",
			fn:           (*valueReader).ReadMinKey,
			streamingErr: nil,
			bufferedErr:  nil,
			vType:        TypeMinKey,
		},
		{
			name:         "ReadNull/incorrect type",
			fn:           (*valueReader).ReadNull,
			streamingErr: (&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeNull),
			bufferedErr:  (&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeNull),
			vType:        TypeEmbeddedDocument,
		},
		{
			name:         "ReadNull/success",
			fn:           (*valueReader).ReadNull,
			streamingErr: nil,
			bufferedErr:  nil,
			vType:        TypeNull,
		},
		{
			name:         "ReadUndefined/incorrect type",
			fn:           (*valueReader).ReadUndefined,
			streamingErr: (&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeUndefined),
			bufferedErr:  (&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeUndefined),
			vType:        TypeEmbeddedDocument,
		},
		{
			name:         "ReadUndefined/success",
			fn:           (*valueReader).ReadUndefined,
			streamingErr: nil,
			bufferedErr:  nil,
			vType:        TypeUndefined,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("buffered", func(t *testing.T) {
				vr := &valueReader{
					src: &bufferedByteSrc{buf: []byte{}},
					stack: []vrState{
						{mode: mTopLevel},
						{
							mode:  mElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}
				wantErr := tc.bufferedErr

				err := tc.fn(vr)
				if !errequal(t, err, wantErr) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, wantErr)
				}
			})

			t.Run("streaming", func(t *testing.T) {
				vr := &valueReader{
					src: &streamingValueReader{br: bufio.NewReader(bytes.NewReader([]byte{}))},
					stack: []vrState{
						{mode: mTopLevel},
						{
							mode:  mElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}
				wantErr := tc.streamingErr

				err := tc.fn(vr)
				if !errequal(t, err, wantErr) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, wantErr)
				}
			})
		})
	}
}

func TestValueReader_ReadObjectID(t *testing.T) {
	testCases := []struct {
		name  string
		data  []byte
		oid   ObjectID
		err   error
		vType Type
	}{
		{
			name:  "incorrect type",
			data:  []byte{},
			oid:   ObjectID{},
			err:   (&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeObjectID),
			vType: TypeEmbeddedDocument,
		},
		{
			name:  "not enough bytes for objectID",
			data:  []byte{},
			oid:   ObjectID{},
			err:   io.EOF,
			vType: TypeObjectID,
		},
		{
			name:  "success",
			data:  []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
			oid:   ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
			err:   nil,
			vType: TypeObjectID,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("buffered", func(t *testing.T) {
				vr := &valueReader{
					src: &bufferedByteSrc{buf: tc.data},
					stack: []vrState{
						{mode: mTopLevel},
						{
							mode:  mElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}

				oid, err := vr.ReadObjectID()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if oid != tc.oid {
					t.Errorf("ObjectIDs did not match. got %v; want %v", oid, tc.oid)
				}
			})

			t.Run("streaming", func(t *testing.T) {
				vr := &valueReader{
					src: &streamingValueReader{br: bufio.NewReader(bytes.NewReader(tc.data))},
					stack: []vrState{
						{mode: mTopLevel},
						{
							mode:  mElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}

				oid, err := vr.ReadObjectID()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if oid != tc.oid {
					t.Errorf("ObjectIDs did not match. got %v; want %v", oid, tc.oid)
				}
			})
		})
	}
}

func TestValueReader_ReadRegex(t *testing.T) {
	testCases := []struct {
		name    string
		data    []byte
		pattern string
		options string
		err     error
		vType   Type
	}{
		{
			name:    "incorrect type",
			data:    []byte{},
			pattern: "",
			options: "",
			err:     (&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeRegex),
			vType:   TypeEmbeddedDocument,
		},
		{
			name:    "length too short",
			data:    []byte{},
			pattern: "",
			options: "",
			err:     io.EOF,
			vType:   TypeRegex,
		},
		{
			name:    "not enough bytes for options",
			data:    []byte{'f', 'o', 'o', 0x00},
			pattern: "",
			options: "",
			err:     io.EOF,
			vType:   TypeRegex,
		},
		{
			name:    "success",
			data:    []byte{'f', 'o', 'o', 0x00, 'b', 'a', 'r', 0x00},
			pattern: "foo",
			options: "bar",
			err:     nil,
			vType:   TypeRegex,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("buffered", func(t *testing.T) {
				vr := &valueReader{
					src: &bufferedByteSrc{buf: tc.data},
					stack: []vrState{
						{mode: mTopLevel},
						{
							mode:  mElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}

				pattern, options, err := vr.ReadRegex()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if pattern != tc.pattern {
					t.Errorf("Incorrect pattern returned. got %s; want %s", pattern, tc.pattern)
				}
				if options != tc.options {
					t.Errorf("Incorrect options returned. got %s; want %s", options, tc.options)
				}
			})

			t.Run("streaming", func(t *testing.T) {
				vr := &valueReader{
					src: &streamingValueReader{br: bufio.NewReader(bytes.NewReader(tc.data))},
					stack: []vrState{
						{mode: mTopLevel},
						{
							mode:  mElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}

				pattern, options, err := vr.ReadRegex()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if pattern != tc.pattern {
					t.Errorf("Incorrect pattern returned. got %s; want %s", pattern, tc.pattern)
				}
				if options != tc.options {
					t.Errorf("Incorrect options returned. got %s; want %s", options, tc.options)
				}
			})
		})
	}
}

func TestValueReader_ReadTimestamp(t *testing.T) {
	testCases := []struct {
		name  string
		data  []byte
		ts    uint32
		incr  uint32
		err   error
		vType Type
	}{
		{
			name:  "incorrect type",
			data:  []byte{},
			ts:    0,
			incr:  0,
			err:   (&valueReader{stack: []vrState{{vType: TypeEmbeddedDocument}}, frame: 0}).typeError(TypeTimestamp),
			vType: TypeEmbeddedDocument,
		},
		{
			name:  "not enough bytes for increment",
			data:  []byte{},
			ts:    0,
			incr:  0,
			err:   io.EOF,
			vType: TypeTimestamp,
		},
		{
			name:  "not enough bytes for timestamp",
			data:  []byte{0x01, 0x02, 0x03, 0x04},
			ts:    0,
			incr:  0,
			err:   io.EOF,
			vType: TypeTimestamp,
		},
		{
			name:  "success",
			data:  []byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00},
			ts:    256,
			incr:  255,
			err:   nil,
			vType: TypeTimestamp,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("buffered", func(t *testing.T) {
				vr := &valueReader{
					src: &bufferedByteSrc{buf: tc.data},
					stack: []vrState{
						{mode: mTopLevel},
						{
							mode:  mElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}

				ts, incr, err := vr.ReadTimestamp()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if ts != tc.ts {
					t.Errorf("Incorrect timestamp returned. got %d; want %d", ts, tc.ts)
				}
				if incr != tc.incr {
					t.Errorf("Incorrect increment returned. got %d; want %d", incr, tc.incr)
				}
			})

			t.Run("streaming", func(t *testing.T) {
				vr := &valueReader{
					src: &streamingValueReader{br: bufio.NewReader(bytes.NewReader(tc.data))},
					stack: []vrState{
						{mode: mTopLevel},
						{
							mode:  mElement,
							vType: tc.vType,
						},
					},
					frame: 1,
				}

				ts, incr, err := vr.ReadTimestamp()
				if !errequal(t, err, tc.err) {
					t.Errorf("Returned errors do not match. got %v; want %v", err, tc.err)
				}
				if ts != tc.ts {
					t.Errorf("Incorrect timestamp returned. got %d; want %d", ts, tc.ts)
				}
				if incr != tc.incr {
					t.Errorf("Incorrect increment returned. got %d; want %d", incr, tc.incr)
				}
			})
		})
	}
}

func TestValueReader_ReadBytes_Skip_Buffered(t *testing.T) {
	index, docb := bsoncore.ReserveLength(nil)
	docb = bsoncore.AppendNullElement(docb, "foobar")
	docb = append(docb, 0x00)
	docb = bsoncore.UpdateLength(docb, index, int32(len(docb)))
	cwsbytes := bsoncore.AppendCodeWithScope(nil, "var hellow = world;", docb)
	strbytes := []byte{0x04, 0x00, 0x00, 0x00, 'f', 'o', 'o', 0x00}

	testCases := []struct {
		name           string
		t              Type
		data           []byte
		err            error
		offset         int64
		startingOffset int64
	}{
		{
			name: "Array/invalid length",
			t:    TypeArray,
			data: []byte{0x01, 0x02, 0x03},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Array/not enough bytes",
			t:    TypeArray,
			data: []byte{0x0F, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Array/success",
			t:    TypeArray,
			data: []byte{0x08, 0x00, 0x00, 0x00, 0x0A, '1', 0x00, 0x00},
			err:  nil, offset: 8, startingOffset: 0,
		},
		{
			name: "EmbeddedDocument/invalid length",
			t:    TypeEmbeddedDocument,
			data: []byte{0x01, 0x02, 0x03},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "EmbeddedDocument/not enough bytes",
			t:    TypeEmbeddedDocument,
			data: []byte{0x0F, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "EmbeddedDocument/success",
			t:    TypeEmbeddedDocument,
			data: []byte{0x08, 0x00, 0x00, 0x00, 0x0A, 'A', 0x00, 0x00},
			err:  nil, offset: 8, startingOffset: 0,
		},
		{
			name: "CodeWithScope/invalid length",
			t:    TypeCodeWithScope,
			data: []byte{0x01, 0x02, 0x03},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "CodeWithScope/not enough bytes",
			t:    TypeCodeWithScope,
			data: []byte{0x0F, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "CodeWithScope/success",
			t:    TypeCodeWithScope,
			data: cwsbytes,
			err:  nil, offset: 41, startingOffset: 0,
		},
		{
			name: "Binary/invalid length",
			t:    TypeBinary,
			data: []byte{0x01, 0x02, 0x03},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Binary/not enough bytes",
			t:    TypeBinary,
			data: []byte{0x0F, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Binary/success",
			t:    TypeBinary,
			data: []byte{0x03, 0x00, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03},
			err:  nil, offset: 8, startingOffset: 0,
		},
		{
			name: "Boolean/invalid length",
			t:    TypeBoolean,
			data: []byte{},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Boolean/success",
			t:    TypeBoolean,
			data: []byte{0x01},
			err:  nil, offset: 1, startingOffset: 0,
		},
		{
			name: "DBPointer/invalid length",
			t:    TypeDBPointer,
			data: []byte{0x01, 0x02, 0x03},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "DBPointer/not enough bytes",
			t:    TypeDBPointer,
			data: []byte{0x0F, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "DBPointer/success",
			t:    TypeDBPointer,
			data: []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
			err:  nil, offset: 17, startingOffset: 0,
		},
		{
			name: "DBPointer/not enough bytes",
			t:    TypeDateTime,
			data: []byte{0x01, 0x02, 0x03, 0x04},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "DBPointer/success",
			t:    TypeDateTime,
			data: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			err:  nil, offset: 8, startingOffset: 0,
		},
		{
			name: "Double/not enough bytes",
			t:    TypeDouble,
			data: []byte{0x01, 0x02, 0x03, 0x04},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Double/success",
			t:    TypeDouble,
			data: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			err:  nil, offset: 8, startingOffset: 0,
		},
		{
			name: "Int64/not enough bytes",
			t:    TypeInt64,
			data: []byte{0x01, 0x02, 0x03, 0x04},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Int64/success",
			t:    TypeInt64,
			data: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			err:  nil, offset: 8, startingOffset: 0,
		},
		{
			name: "Timestamp/not enough bytes",
			t:    TypeTimestamp,
			data: []byte{0x01, 0x02, 0x03, 0x04},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Timestamp/success",
			t:    TypeTimestamp,
			data: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			err:  nil, offset: 8, startingOffset: 0,
		},
		{
			name: "Decimal128/not enough bytes",
			t:    TypeDecimal128,
			data: []byte{0x01, 0x02, 0x03, 0x04},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Decimal128/success",
			t:    TypeDecimal128,
			data: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
			err:  nil, offset: 16, startingOffset: 0,
		},
		{
			name: "Int32/not enough bytes",
			t:    TypeInt32,
			data: []byte{0x01, 0x02},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Int32/success",
			t:    TypeInt32,
			data: []byte{0x01, 0x02, 0x03, 0x04},
			err:  nil, offset: 4, startingOffset: 0,
		},
		{
			name: "Javascript/invalid length",
			t:    TypeJavaScript,
			data: strbytes[:2],
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Javascript/not enough bytes",
			t:    TypeJavaScript,
			data: strbytes[:5],
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Javascript/success",
			t:    TypeJavaScript,
			data: strbytes,
			err:  nil, offset: 8, startingOffset: 0,
		},
		{
			name: "String/invalid length",
			t:    TypeString,
			data: strbytes[:2],
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "String/not enough bytes",
			t:    TypeString,
			data: strbytes[:5],
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "String/success",
			t:    TypeString,
			data: strbytes,
			err:  nil, offset: 8, startingOffset: 0,
		},
		{
			name: "Symbol/invalid length",
			t:    TypeSymbol,
			data: strbytes[:2],
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Symbol/not enough bytes",
			t:    TypeSymbol,
			data: strbytes[:5],
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Symbol/success",
			t:    TypeSymbol,
			data: strbytes,
			err:  nil, offset: 8, startingOffset: 0,
		},
		{
			name: "MaxKey/success",
			t:    TypeMaxKey,
			data: []byte{},
			err:  nil, offset: 0, startingOffset: 0,
		},
		{
			name: "MinKey/success",
			t:    TypeMinKey,
			data: []byte{},
			err:  nil, offset: 0, startingOffset: 0,
		},
		{
			name: "Null/success",
			t:    TypeNull,
			data: []byte{},
			err:  nil, offset: 0, startingOffset: 0,
		},
		{
			name: "Undefined/success",
			t:    TypeUndefined,
			data: []byte{},
			err:  nil, offset: 0, startingOffset: 0,
		},
		{
			name: "ObjectID/not enough bytes",
			t:    TypeObjectID,
			data: []byte{0x01, 0x02, 0x03, 0x04},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "ObjectID/success",
			t:    TypeObjectID,
			data: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
			err:  nil, offset: 12, startingOffset: 0,
		},
		{
			name: "Regex/not enough bytes (first string)",
			t:    TypeRegex,
			data: []byte{'f', 'o', 'o'},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Regex/not enough bytes (second string)",
			t:    TypeRegex,
			data: []byte{'f', 'o', 'o', 0x00, 'b', 'a', 'r'},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Regex/success",
			t:    TypeRegex,
			data: []byte{0x00, 0x00, 0x00, 'f', 'o', 'o', 0x00, 'i', 0x00},
			err:  nil, offset: 9, startingOffset: 3,
		},
		{
			name:   "Unknown Type",
			t:      Type(0),
			data:   nil,
			err:    fmt.Errorf("attempted to read bytes of unknown BSON type %v", Type(0)),
			offset: 0, startingOffset: 0,
		},
	}

	for _, tc := range testCases {
		tc := tc
		const startingEnd = 64

		t.Run(tc.name, func(t *testing.T) {
			t.Run("Skip", func(t *testing.T) {
				vr := &valueReader{
					src: &bufferedByteSrc{buf: tc.data, offset: tc.startingOffset},
					stack: []vrState{
						{mode: mTopLevel, end: startingEnd},
						{mode: mElement, vType: tc.t},
					},
					frame: 1,
				}

				err := vr.Skip()
				if !errequal(t, err, tc.err) {
					t.Errorf("Did not receive expected error; got %v; want %v", err, tc.err)
				}
				if tc.err == nil && vr.src.pos() != tc.offset {
					t.Errorf("Offset not set at correct position; got %d; want %d", vr.src.pos(), tc.offset)
				}
			})
			t.Run("ReadBytes", func(t *testing.T) {
				vr := &valueReader{
					src: &bufferedByteSrc{buf: tc.data, offset: tc.startingOffset},
					stack: []vrState{
						{mode: mTopLevel, end: startingEnd},
						{mode: mElement, vType: tc.t},
					},
					frame: 1,
				}

				_, got, err := vr.readValueBytes(nil)
				if !errequal(t, err, tc.err) {
					t.Errorf("Did not receive expected error; got %v; want %v", err, tc.err)
				}
				if tc.err == nil && vr.src.pos() != tc.offset {
					t.Errorf("Offset not set at correct position; got %d; want %d", vr.src.pos(), tc.offset)
				}
				if tc.err == nil && !bytes.Equal(got, tc.data[tc.startingOffset:]) {
					t.Errorf("Did not receive expected bytes. got %v; want %v", got, tc.data[tc.startingOffset:])
				}
			})
		})
	}

	t.Run("ReadValueBytes/Top Level Doc", func(t *testing.T) {
		testCases := []struct {
			name     string
			want     []byte
			wantType Type
			wantErr  error
		}{
			{
				"success",
				bsoncore.BuildDocument(nil, bsoncore.AppendDoubleElement(nil, "pi", 3.14159)),
				Type(0),
				nil,
			},
			{
				"wrong length",
				[]byte{0x01, 0x02, 0x03},
				Type(0),
				io.EOF,
			},
			{
				"append bytes",
				[]byte{0x01, 0x02, 0x03, 0x04},
				Type(0),
				io.EOF,
			},
		}

		for _, tc := range testCases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				vr := &valueReader{
					src: &bufferedByteSrc{buf: tc.want},
					stack: []vrState{
						{mode: mTopLevel},
					},
					frame: 0,
				}
				gotType, got, gotErr := vr.readValueBytes(nil)
				if !errors.Is(gotErr, tc.wantErr) {
					t.Errorf("Did not receive expected error. got %v; want %v", gotErr, tc.wantErr)
				}
				if tc.wantErr == nil && gotType != tc.wantType {
					t.Errorf("Did not receive expected type. got %v; want %v", gotType, tc.wantType)
				}
				if tc.wantErr == nil && !bytes.Equal(got, tc.want) {
					t.Errorf("Did not receive expected bytes. got %v; want %v", got, tc.want)
				}
			})
		}
	})
}

func TestValueReader_ReadBytes_Skip_Streaming(t *testing.T) {
	index, docb := bsoncore.ReserveLength(nil)
	docb = bsoncore.AppendNullElement(docb, "foobar")
	docb = append(docb, 0x00)
	docb = bsoncore.UpdateLength(docb, index, int32(len(docb)))
	cwsbytes := bsoncore.AppendCodeWithScope(nil, "var hellow = world;", docb)
	strbytes := []byte{0x04, 0x00, 0x00, 0x00, 'f', 'o', 'o', 0x00}

	testCases := []struct {
		name           string
		t              Type
		data           []byte
		err            error
		offset         int64
		startingOffset int64
	}{
		{
			name: "Array/invalid length",
			t:    TypeArray,
			data: []byte{0x01, 0x02, 0x03},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Array/not enough bytes",
			t:    TypeArray,
			data: []byte{0x0F, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Array/success",
			t:    TypeArray,
			data: []byte{0x08, 0x00, 0x00, 0x00, 0x0A, '1', 0x00, 0x00},
			err:  nil, offset: 8, startingOffset: 0,
		},
		{
			name: "EmbeddedDocument/invalid length",
			t:    TypeEmbeddedDocument,
			data: []byte{0x01, 0x02, 0x03},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "EmbeddedDocument/not enough bytes",
			t:    TypeEmbeddedDocument,
			data: []byte{0x0F, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "EmbeddedDocument/success",
			t:    TypeEmbeddedDocument,
			data: []byte{0x08, 0x00, 0x00, 0x00, 0x0A, 'A', 0x00, 0x00},
			err:  nil, offset: 8, startingOffset: 0,
		},
		{
			name: "CodeWithScope/invalid length",
			t:    TypeCodeWithScope,
			data: []byte{0x01, 0x02, 0x03},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "CodeWithScope/not enough bytes",
			t:    TypeCodeWithScope,
			data: []byte{0x0F, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "CodeWithScope/success",
			t:    TypeCodeWithScope,
			data: cwsbytes,
			err:  nil, offset: 41, startingOffset: 0,
		},
		{
			name: "Binary/invalid length",
			t:    TypeBinary,
			data: []byte{0x01, 0x02, 0x03},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Binary/not enough bytes",
			t:    TypeBinary,
			data: []byte{0x0F, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Binary/success",
			t:    TypeBinary,
			data: []byte{0x03, 0x00, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03},
			err:  nil, offset: 8, startingOffset: 0,
		},
		{
			name: "Boolean/invalid length",
			t:    TypeBoolean,
			data: []byte{},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Boolean/success",
			t:    TypeBoolean,
			data: []byte{0x01},
			err:  nil, offset: 1, startingOffset: 0,
		},
		{
			name: "DBPointer/invalid length",
			t:    TypeDBPointer,
			data: []byte{0x01, 0x02, 0x03},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "DBPointer/not enough bytes",
			t:    TypeDBPointer,
			data: []byte{0x0F, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "DBPointer/success",
			t:    TypeDBPointer,
			data: []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
			err:  nil, offset: 17, startingOffset: 0,
		},
		{
			name: "DBPointer/not enough bytes",
			t:    TypeDateTime,
			data: []byte{0x01, 0x02, 0x03, 0x04},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "DBPointer/success",
			t:    TypeDateTime,
			data: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			err:  nil, offset: 8, startingOffset: 0,
		},
		{
			name: "Double/not enough bytes",
			t:    TypeDouble,
			data: []byte{0x01, 0x02, 0x03, 0x04},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Double/success",
			t:    TypeDouble,
			data: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			err:  nil, offset: 8, startingOffset: 0,
		},
		{
			name: "Int64/not enough bytes",
			t:    TypeInt64,
			data: []byte{0x01, 0x02, 0x03, 0x04},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Int64/success",
			t:    TypeInt64,
			data: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			err:  nil, offset: 8, startingOffset: 0,
		},
		{
			name: "Timestamp/not enough bytes",
			t:    TypeTimestamp,
			data: []byte{0x01, 0x02, 0x03, 0x04},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Timestamp/success",
			t:    TypeTimestamp,
			data: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			err:  nil, offset: 8, startingOffset: 0,
		},
		{
			name: "Decimal128/not enough bytes",
			t:    TypeDecimal128,
			data: []byte{0x01, 0x02, 0x03, 0x04},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Decimal128/success",
			t:    TypeDecimal128,
			data: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
			err:  nil, offset: 16, startingOffset: 0,
		},
		{
			name: "Int32/not enough bytes",
			t:    TypeInt32,
			data: []byte{0x01, 0x02},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Int32/success",
			t:    TypeInt32,
			data: []byte{0x01, 0x02, 0x03, 0x04},
			err:  nil, offset: 4, startingOffset: 0,
		},
		{
			name: "Javascript/invalid length",
			t:    TypeJavaScript,
			data: strbytes[:2],
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Javascript/not enough bytes",
			t:    TypeJavaScript,
			data: strbytes[:5],
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Javascript/success",
			t:    TypeJavaScript,
			data: strbytes,
			err:  nil, offset: 8, startingOffset: 0,
		},
		{
			name: "String/invalid length",
			t:    TypeString,
			data: strbytes[:2],
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "String/not enough bytes",
			t:    TypeString,
			data: strbytes[:5],
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "String/success",
			t:    TypeString,
			data: strbytes,
			err:  nil, offset: 8, startingOffset: 0,
		},
		{
			name: "Symbol/invalid length",
			t:    TypeSymbol,
			data: strbytes[:2],
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Symbol/not enough bytes",
			t:    TypeSymbol,
			data: strbytes[:5],
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Symbol/success",
			t:    TypeSymbol,
			data: strbytes,
			err:  nil, offset: 8, startingOffset: 0,
		},
		{
			name: "MaxKey/success",
			t:    TypeMaxKey,
			data: []byte{},
			err:  nil, offset: 0, startingOffset: 0,
		},
		{
			name: "MinKey/success",
			t:    TypeMinKey,
			data: []byte{},
			err:  nil, offset: 0, startingOffset: 0,
		},
		{
			name: "Null/success",
			t:    TypeNull,
			data: []byte{},
			err:  nil, offset: 0, startingOffset: 0,
		},
		{
			name: "Undefined/success",
			t:    TypeUndefined,
			data: []byte{},
			err:  nil, offset: 0, startingOffset: 0,
		},
		{
			name: "ObjectID/not enough bytes",
			t:    TypeObjectID,
			data: []byte{0x01, 0x02, 0x03, 0x04},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "ObjectID/success",
			t:    TypeObjectID,
			data: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
			err:  nil, offset: 12, startingOffset: 0,
		},
		{
			name: "Regex/not enough bytes (first string)",
			t:    TypeRegex,
			data: []byte{'f', 'o', 'o'},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Regex/not enough bytes (second string)",
			t:    TypeRegex,
			data: []byte{'f', 'o', 'o', 0x00, 'b', 'a', 'r'},
			err:  io.EOF, offset: 0, startingOffset: 0,
		},
		{
			name: "Regex/success",
			t:    TypeRegex,
			data: []byte{0x00, 0x00, 0x00, 'f', 'o', 'o', 0x00, 'i', 0x00},
			err:  nil, offset: 9, startingOffset: 3,
		},
		{
			name:   "Unknown Type",
			t:      Type(0),
			data:   nil,
			err:    fmt.Errorf("attempted to read bytes of unknown BSON type %v", Type(0)),
			offset: 0, startingOffset: 0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			const startingEnd = 64
			t.Run("Skip", func(t *testing.T) {
				vr := &valueReader{
					src: &streamingValueReader{
						br:     bufio.NewReader(bytes.NewReader(tc.data[tc.startingOffset:tc.offset])),
						offset: tc.startingOffset,
					},
					stack: []vrState{
						{mode: mTopLevel, end: startingEnd},
						{mode: mElement, vType: tc.t},
					},
					frame: 1,
				}

				err := vr.Skip()
				if !errequal(t, err, tc.err) {
					t.Errorf("Did not receive expected error; got %v; want %v", err, tc.err)
				}
				if tc.err == nil {
					offset := startingEnd - vr.stack[0].end
					if offset != tc.offset {
						t.Errorf("Offset not set at correct position; got %d; want %d", offset, tc.offset)
					}
				}
			})
			t.Run("ReadBytes", func(t *testing.T) {
				vr := &valueReader{
					src: &streamingValueReader{
						br:     bufio.NewReader(bytes.NewReader(tc.data[tc.startingOffset:tc.offset])),
						offset: tc.startingOffset,
					},
					stack: []vrState{
						{mode: mTopLevel, end: startingEnd},
						{mode: mElement, vType: tc.t},
					},
					frame: 1,
				}

				_, got, err := vr.readValueBytes(nil)
				if !errequal(t, err, tc.err) {
					t.Errorf("Did not receive expected error; got %v; want %v", err, tc.err)
				}
				if tc.err == nil {
					offset := startingEnd - vr.stack[0].end
					if offset != tc.offset {
						t.Errorf("Offset not set at correct position; got %d; want %d", vr.offset, tc.offset)
					}
				}
				if tc.err == nil && !bytes.Equal(got, tc.data[tc.startingOffset:]) {
					t.Errorf("Did not receive expected bytes. got %v; want %v", got, tc.data[tc.startingOffset:])
				}
			})
		})
	}

	t.Run("ReadValueBytes/Top Level Doc", func(t *testing.T) {
		testCases := []struct {
			name     string
			want     []byte
			wantType Type
			wantErr  error
		}{
			{
				"success",
				bsoncore.BuildDocument(nil, bsoncore.AppendDoubleElement(nil, "pi", 3.14159)),
				Type(0),
				nil,
			},
			{
				"wrong length",
				[]byte{0x01, 0x02, 0x03},
				Type(0),
				io.EOF,
			},
			{
				"append bytes",
				[]byte{0x01, 0x02, 0x03, 0x04},
				Type(0),
				io.ErrUnexpectedEOF,
			},
		}

		for _, tc := range testCases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				vr := &valueReader{
					src: &streamingValueReader{
						br: bufio.NewReader(bytes.NewReader(tc.want)),
					},
					stack: []vrState{
						{mode: mTopLevel},
					},
					frame: 0,
				}
				gotType, got, gotErr := vr.readValueBytes(nil)
				if !errors.Is(gotErr, tc.wantErr) {
					t.Errorf("Did not receive expected error. got %v; want %v", gotErr, tc.wantErr)
				}
				if tc.wantErr == nil && gotType != tc.wantType {
					t.Errorf("Did not receive expected type. got %v; want %v", gotType, tc.wantType)
				}
				if tc.wantErr == nil && !bytes.Equal(got, tc.want) {
					t.Errorf("Did not receive expected bytes. got %v; want %v", got, tc.want)
				}
			})
		}
	})
}

func TestValueReader_InvalidTransition(t *testing.T) {
	t.Run("Skip", func(t *testing.T) {
		vr := &valueReader{stack: []vrState{{mode: mTopLevel}}}
		wanterr := (&valueReader{stack: []vrState{{mode: mTopLevel}}}).invalidTransitionErr(0, "Skip", []mode{mElement, mValue})
		goterr := vr.Skip()
		if !cmp.Equal(goterr, wanterr, cmp.Comparer(assert.CompareErrors)) {
			t.Errorf("Expected correct invalid transition error. got %v; want %v", goterr, wanterr)
		}
	})

	t.Run("ReadBytes", func(t *testing.T) {
		vr := &valueReader{stack: []vrState{{mode: mTopLevel}, {mode: mDocument}}, frame: 1}
		wanterr := (&valueReader{stack: []vrState{{mode: mTopLevel}, {mode: mDocument}}, frame: 1}).
			invalidTransitionErr(0, "readValueBytes", []mode{mElement, mValue})
		_, _, goterr := vr.readValueBytes(nil)
		if !cmp.Equal(goterr, wanterr, cmp.Comparer(assert.CompareErrors)) {
			t.Errorf("Expected correct invalid transition error. got %v; want %v", goterr, wanterr)
		}
	})
}

func errequal(t *testing.T, err1, err2 error) bool {
	t.Helper()
	if err1 == nil && err2 == nil { // If they are both nil, they are equal
		return true
	}
	if err1 == nil || err2 == nil { // If only one is nil, they are not equal
		return false
	}

	if errors.Is(err1, err2) { // They are the same error, they are equal
		return true
	}

	if err1.Error() == err2.Error() { // They string formats match, they are equal
		return true
	}

	return false
}
