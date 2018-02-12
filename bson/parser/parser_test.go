// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package parser

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/parser/ast"
)

func TestBSONParser(t *testing.T) {
	t.Run("read-int-32", func(t *testing.T) {
		var want int32 = 123
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint32(buf, uint32(want))
		rdr := bytes.NewReader(buf)
		p := &Parser{r: bufio.NewReader(rdr)}
		got, err := p.readInt32()
		if err != nil {
			t.Errorf("Unexpected error while reading int32: %s", err)
		}
		if got != want {
			t.Errorf("Unexpected result. got %d; want %d", got, want)
		}
		t.Run("read-error", func(t *testing.T) {
			want := errors.New("read-int-32-error")
			p := &Parser{r: bufio.NewReader(&errReader{err: want})}
			_, got := p.readInt32()
			if got != want {
				t.Errorf("Expected error. got %s; want %s", got, want)
			}
		})
	})

	t.Run("read-int-64", func(t *testing.T) {
		var want int64 = 123
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, uint64(want))
		rdr := bytes.NewReader(buf)
		p := &Parser{r: bufio.NewReader(rdr)}
		got, err := p.readInt64()
		if err != nil {
			t.Errorf("Unexpected error while reading int64: %s", err)
		}
		if got != want {
			t.Errorf("Unexpected result. got %d; want %d", got, want)
		}
		t.Run("read-error", func(t *testing.T) {
			want := errors.New("read-int-64-error")
			p := &Parser{r: bufio.NewReader(&errReader{err: want})}
			_, got := p.readInt64()
			if got != want {
				t.Errorf("Expected error. got %s; want %s", got, want)
			}
		})
	})

	t.Run("read-uint-64", func(t *testing.T) {
		var want uint64 = 123
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, want)
		rdr := bytes.NewReader(buf)
		p := &Parser{r: bufio.NewReader(rdr)}
		got, err := p.readUint64()
		if err != nil {
			t.Errorf("Unexpected error while reading uint64: %s", err)
		}
		if got != want {
			t.Errorf("Unexpected result. got %d; want %d", got, want)
		}
		t.Run("read-error", func(t *testing.T) {
			want := errors.New("read-uint-64-error")
			p := &Parser{r: bufio.NewReader(&errReader{err: want})}
			_, got := p.readInt64()
			if got != want {
				t.Errorf("Expected error. got %s; want %s", got, want)
			}
		})
	})
	t.Run("read-object-id", func(t *testing.T) {
		var want [12]byte
		_, err := rand.Read(want[:])
		if err != nil {
			t.Errorf("Error while creating random object id: %s", err)
		}
		rdr := bytes.NewReader(want[:])
		p := &Parser{r: bufio.NewReader(rdr)}
		got, err := p.readObjectID()
		if err != nil {
			t.Errorf("Unexpected error while reading objectID: %s", err)
		}
		if !bytes.Equal(got[:], want[:]) {
			t.Errorf("Unexpected result. got %d; want %d", got, want)
		}
		t.Run("read-error", func(t *testing.T) {
			want := errors.New("read-object-id-error")
			p := &Parser{r: bufio.NewReader(&errReader{err: want})}
			_, got := p.readObjectID()
			if got != want {
				t.Errorf("Expected error. got %s; want %s", got, want)
			}
		})
	})

	t.Run("read-bool", func(t *testing.T) {
		t.Run("false", func(t *testing.T) {
			want := false
			rdr := bytes.NewReader([]byte{'\x00'})
			p := &Parser{r: bufio.NewReader(rdr)}
			got, err := p.readBoolean()
			if err != nil {
				t.Errorf("Unexpected error while reading objectID: %s", err)
			}
			if got != want {
				t.Errorf("Unexpected result. got %t; want %t", got, want)
			}
		})
		t.Run("true", func(t *testing.T) {
			want := true
			rdr := bytes.NewReader([]byte{'\x01'})
			p := &Parser{r: bufio.NewReader(rdr)}
			got, err := p.readBoolean()
			if err != nil {
				t.Errorf("Unexpected error while reading objectID: %s", err)
			}
			if got != want {
				t.Errorf("Unexpected result. got %t; want %t", got, want)
			}
		})
		t.Run("corrupt-document", func(t *testing.T) {
			want := ErrCorruptDocument
			rdr := bytes.NewReader([]byte{'\x03'})
			p := &Parser{r: bufio.NewReader(rdr)}
			_, got := p.readBoolean()
			if got != want {
				t.Errorf("Unexpected result. got %s; want %s", got, want)
			}
		})
		t.Run("read-error", func(t *testing.T) {
			want := errors.New("read-bool-error")
			p := &Parser{r: bufio.NewReader(&errReader{err: want})}
			_, got := p.readBoolean()
			if got != want {
				t.Errorf("Expected error. got %s; want %s", got, want)
			}
		})
	})

	t.Run("parse-document", func(t *testing.T) {
		want := &ast.Document{
			Length: 5,
			EList:  []ast.Element{},
		}
		b := make([]byte, 5)
		binary.LittleEndian.PutUint32(b[:4], 5)
		b[4] = '\x00'
		r := bytes.NewReader(b)
		p := &Parser{r: bufio.NewReader(r)}
		got, err := p.ParseDocument()
		if err != nil {
			t.Errorf("Unexpected error while parsing document: %s", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Unexpected result. got %v; want %v", got, want)
		}

		t.Run("read-int-error", func(t *testing.T) {
			want := errors.New("parse-document-int-32-error")
			p := &Parser{r: bufio.NewReader(&errReader{err: want})}
			_, got := p.ParseDocument()
			if got != want {
				t.Errorf("Expected error. got %s; want %s", got, want)
			}
		})
		t.Run("parse-elist-error", func(t *testing.T) {
			want := errors.New("parse-document-parse-elist-error")
			b := make([]byte, 4)
			binary.LittleEndian.PutUint32(b, 5)
			p := &Parser{r: bufio.NewReader(&errReader{b: b, err: want})}
			_, got := p.ParseDocument()
			if got != want {
				t.Errorf("Expected error. got %s; want %s", got, want)
			}
		})
	})

	t.Run("parse-elist", func(t *testing.T) {
		t.Run("peek-error", func(t *testing.T) {
			want := errors.New("parse-elist-peek-error")
			p := &Parser{r: bufio.NewReader(&errReader{err: want})}
			_, got := p.ParseEList()
			if got != want {
				t.Errorf("Expected error. got %s; want %s", got, want)
			}
		})
		t.Run("peek-null", func(t *testing.T) {
			want := []ast.Element{}
			r := bytes.NewReader([]byte{'\x00'})
			p := &Parser{r: bufio.NewReader(r)}
			got, err := p.ParseEList()
			if err != nil {
				t.Errorf("Unexpected error: %s", err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("Unexpected result. got %v; want %v", got, want)
			}
		})
		t.Run("parse-element-error", func(t *testing.T) {
			want := errors.New("parse-element-error")
			b := []byte{'\x01'}
			p := &Parser{r: bufio.NewReader(&errReader{b: b, err: want})}
			_, got := p.ParseEList()
			if got != want {
				t.Errorf("Expected error. got %s; want %s", got, want)
			}
		})

		want := []ast.Element{&ast.NullElement{Name: &ast.ElementKeyName{Key: "foo"}}}
		r := bytes.NewReader([]byte{'\x0A', 'f', 'o', 'o', '\x00', '\x00'})
		p := &Parser{r: bufio.NewReader(r)}
		got, err := p.ParseEList()
		if err != nil {
			t.Errorf("Unexpected error while parsing elist: %s", err)
		}
		for _, elem := range got {
			ne, ok := elem.(*ast.NullElement)
			if !ok {
				t.Errorf("Unexpected result. got %T; want %T", ne, want[0])
			}
			if ne.Name.Key != want[0].(*ast.NullElement).Name.Key {
				t.Errorf("Unexpected result. got %v; want %v", ne.Name, want[0].(*ast.NullElement).Name)
			}
		}
	})

	t.Run("parse-element", parseElementTest)

	t.Run("parse-ename", func(t *testing.T) {
		t.Run("parse-cstring-error", func(t *testing.T) {
			want := errors.New("parse-cstring-error")
			p := &Parser{r: bufio.NewReader(&errReader{err: want})}
			_, got := p.ParseEName()
			if got != want {
				t.Errorf("Expected error. got %s; want %s", got, want)
			}
		})
		want := &ast.ElementKeyName{Key: "foo"}
		r := bytes.NewReader([]byte{'f', 'o', 'o', '\x00'})
		p := &Parser{r: bufio.NewReader(r)}
		got, err := p.ParseEName()
		if err != nil {
			t.Errorf("Unexpected error while parsing elist: %s", err)
		}
		if got.Key != want.Key {
			t.Errorf("Unexpected result. got %s; want %s", got.Key, want.Key)
		}
	})

	t.Run("parse-string", func(t *testing.T) {
		t.Run("read-int32-error", func(t *testing.T) {
			want := errors.New("read-int32-error")
			p := &Parser{r: bufio.NewReader(&errReader{err: want})}
			_, got := p.ParseString()
			if got != want {
				t.Errorf("Expected error. got %s; want %s", got, want)
			}
		})
		t.Run("readfull-error", func(t *testing.T) {
			b := make([]byte, 4)
			binary.LittleEndian.PutUint32(b, 10)
			want := errors.New("readfull-error")
			p := &Parser{r: bufio.NewReader(&errReader{b: b, err: want})}
			_, got := p.ParseString()
			if got != want {
				t.Errorf("Expected error. got %s; want %s", got, want)
			}
		})
		t.Run("readbyte-error", func(t *testing.T) {
			b := make([]byte, 7)
			binary.LittleEndian.PutUint32(b[:4], 3)
			b[4], b[5], b[6] = 'f', 'o', 'o'
			want := ErrCorruptDocument
			p := &Parser{r: bufio.NewReader(&errReader{b: b, err: want})}
			_, got := p.ParseString()
			if got != want {
				t.Errorf("Expected error. got %s; want %s", got, want)
			}
		})
		t.Run("corrupt-document", func(t *testing.T) {
			b := make([]byte, 8)
			binary.LittleEndian.PutUint32(b[:4], 3)
			b[4], b[5], b[6], b[7] = 'f', 'o', 'o', '\x01'
			want := ErrCorruptDocument
			p := &Parser{r: bufio.NewReader(&errReader{b: b, err: want})}
			_, got := p.ParseString()
			if got != want {
				t.Errorf("Expected error. got %s; want %s", got, want)
			}
		})
		b := make([]byte, 8)
		binary.LittleEndian.PutUint32(b[:4], 4)
		b[4], b[5], b[6], b[7] = 'f', 'o', 'o', '\x00'
		want := "foo"
		r := bytes.NewReader(b)
		p := &Parser{r: bufio.NewReader(r)}
		got, err := p.ParseString()
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if got != want {
			t.Errorf("Unexpected result. got %s; want %s", got, want)
		}
	})

	t.Run("parse-cstring", func(t *testing.T) {
		t.Run("read-bytes-error", func(t *testing.T) {
			want := errors.New("read-bytes-error")
			p := &Parser{r: bufio.NewReader(&errReader{err: want})}
			_, got := p.ParseCString()
			if got != want {
				t.Errorf("Expected error. got %s; want %s", got, want)
			}
		})
		b := []byte{'f', 'o', 'o', '\x00'}
		want := "foo"
		r := bytes.NewReader(b)
		p := &Parser{r: bufio.NewReader(r)}
		got, err := p.ParseCString()
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if got != want {
			t.Errorf("Unexpected result. got %s; want %s", got, want)
		}

	})

	t.Run("parse-binary", func(t *testing.T) {
		t.Run("read-int-32-error", func(t *testing.T) {
			want := errors.New("read-int-32-error")
			p := &Parser{r: bufio.NewReader(&errReader{err: want})}
			_, got := p.ParseBinary()
			if got != want {
				t.Errorf("Expected error. got %s; want %s", got, want)
			}
		})
		t.Run("parse-subtype-error", func(t *testing.T) {
			b := make([]byte, 4)
			binary.LittleEndian.PutUint32(b, 5)
			want := errors.New("parse-subtype-error")
			p := &Parser{r: bufio.NewReader(&errReader{b: b, err: want})}
			_, got := p.ParseBinary()
			if got != want {
				t.Errorf("Expected error. got %s; want %s", got, want)
			}
		})
		t.Run("read-full-error", func(t *testing.T) {
			b := make([]byte, 5)
			binary.LittleEndian.PutUint32(b[:4], 5)
			b[4] = '\x00'
			want := errors.New("read-full-error")
			p := &Parser{r: bufio.NewReader(&errReader{b: b, err: want})}
			_, got := p.ParseBinary()
			if got != want {
				t.Errorf("Expected error. got %s; want %s", got, want)
			}
		})
		t.Run("old-binary-corrupt-document-error", func(t *testing.T) {
			b := make([]byte, 7)
			binary.LittleEndian.PutUint32(b[:4], 2)
			b[4], b[5], b[6] = '\x02', 'f', 'o'
			want := ErrCorruptDocument
			p := &Parser{r: bufio.NewReader(&errReader{b: b, err: want})}
			_, got := p.ParseBinary()
			if got != want {
				t.Errorf("Expected error. got %s; want %s", got, want)
			}
		})
		t.Run("old-binary-success", func(t *testing.T) {
			b := make([]byte, 10)
			binary.LittleEndian.PutUint32(b[:4], 5)
			b[4] = '\x02'
			binary.LittleEndian.PutUint32(b[5:9], 1)
			b[9] = 'f'
			want := &ast.Binary{Subtype: ast.SubtypeBinaryOld, Data: []byte{'f'}}
			r := bytes.NewReader(b)
			p := &Parser{r: bufio.NewReader(r)}
			got, err := p.ParseBinary()
			if err != nil {
				t.Errorf("Unexpected error: %s", err)
			}
			if !reflect.DeepEqual(&got, &want) {
				t.Errorf("Unexpected result. got %v; want %v", got, want)
			}
		})
		b := make([]byte, 8)
		binary.LittleEndian.PutUint32(b[:4], 3)
		b[4], b[5], b[6], b[7] = '\x00', 'f', 'o', 'o'
		want := &ast.Binary{Subtype: ast.SubtypeGeneric, Data: []byte{'f', 'o', 'o'}}
		r := bytes.NewReader(b)
		p := &Parser{r: bufio.NewReader(r)}
		got, err := p.ParseBinary()
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if !reflect.DeepEqual(&got, &want) {
			t.Errorf("Unexpected result. got %v; want %v", got, want)
		}
	})

	t.Run("parse-subtype", func(t *testing.T) {
		t.Run("read-byte-error", func(t *testing.T) {
			want := errors.New("read-byte-error")
			p := &Parser{r: bufio.NewReader(&errReader{err: want})}
			_, got := p.ParseSubtype()
			if got != want {
				t.Errorf("Expected error. got %s; want %s", got, want)
			}
		})
		t.Run("unknown-subtype-error", func(t *testing.T) {
			b := []byte{'\x7F'}
			want := ErrUnknownSubtype
			r := bytes.NewReader(b)
			p := &Parser{r: bufio.NewReader(r)}
			_, got := p.ParseSubtype()
			if got != want {
				t.Errorf("Expected error. got %s; want %s", got, want)
			}
		})
		testCases := []struct {
			name string
			b    []byte
			want ast.BinarySubtype
		}{
			{"generic", []byte{'\x00'}, ast.SubtypeGeneric},
			{"function", []byte{'\x01'}, ast.SubtypeFunction},
			{"binary-old", []byte{'\x02'}, ast.SubtypeBinaryOld},
			{"UUID-old", []byte{'\x03'}, ast.SubtypeUUIDOld},
			{"UUID", []byte{'\x04'}, ast.SubtypeUUID},
			{"MD5", []byte{'\x05'}, ast.SubtypeMD5},
			{"user-defined", []byte{'\x80'}, ast.SubtypeUserDefined},
			{"user-defined-max", []byte{'\xFF'}, ast.SubtypeUserDefined},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				r := bytes.NewReader(tc.b)
				p := &Parser{r: bufio.NewReader(r)}
				got, err := p.ParseSubtype()
				if err != nil {
					t.Errorf("Unexpected error: %s", err)
				}
				if got != tc.want {
					t.Errorf("Unexpected result. got %v; want %v", got, tc.want)
				}
			})
		}
	})

	t.Run("parse-double", func(t *testing.T) {
		t.Run("binary-read-error", func(t *testing.T) {
			want := errors.New("binary-read-error")
			p := &Parser{r: bufio.NewReader(&errReader{err: want})}
			_, got := p.ParseDouble()
			if got != want {
				t.Errorf("Expected error. got %s; want %s", got, want)
			}
		})
		want := 3.14159
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, math.Float64bits(want))
		r := bytes.NewReader(b)
		p := &Parser{r: bufio.NewReader(r)}
		got, err := p.ParseDouble()
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if got != want {
			t.Errorf("Unexpected result. got %f; want %f", got, want)
		}
	})

	t.Run("parse-code-with-scope", func(t *testing.T) {
		t.Run("read-int-32-error", func(t *testing.T) {
			want := errors.New("read-int-32-error")
			p := &Parser{r: bufio.NewReader(&errReader{err: want})}
			_, got := p.ParseCodeWithScope()
			if got != want {
				t.Errorf("Expected error. got %s; want %s", got, want)
			}
		})
		t.Run("parse-string-error", func(t *testing.T) {
			b := make([]byte, 4)
			binary.LittleEndian.PutUint32(b, 4)
			want := errors.New("parse-string-error")
			p := &Parser{r: bufio.NewReader(&errReader{b: b, err: want})}
			_, got := p.ParseCodeWithScope()
			if got != want {
				t.Errorf("Expected error. got %s; want %s", got, want)
			}
		})
		t.Run("parse-document-error", func(t *testing.T) {
			b := make([]byte, 9)
			binary.LittleEndian.PutUint32(b[:4], 4)
			binary.LittleEndian.PutUint32(b[4:8], 0)
			b[8] = '\x00'
			want := errors.New("parse-document-error")
			p := &Parser{r: bufio.NewReader(&errReader{b: b, err: want})}
			_, got := p.ParseCodeWithScope()
			if got != want {
				t.Errorf("Expected error. got %s; want %s", got, want)
			}
		})
		want := ast.CodeWithScope{
			String:   "var a = 10;",
			Document: &ast.Document{Length: 5},
		}
		b := make([]byte, 8)
		binary.LittleEndian.PutUint32(b[:4], 0)
		binary.LittleEndian.PutUint32(b[4:8], uint32(len(want.String)+1))
		b = append(b, []byte(want.String)...)
		b = append(b, '\x00')
		doclen := make([]byte, 4)
		binary.LittleEndian.PutUint32(doclen, 5)
		b = append(b, doclen...)
		b = append(b, '\x00')
		r := bytes.NewReader(b)
		p := &Parser{r: bufio.NewReader(r)}
		got, err := p.ParseCodeWithScope()
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if got.String != want.String {
			t.Errorf("String contents do not match. got %s; want %s", got.String, want.String)
		}
		if len(got.Document.EList) != len(want.Document.EList) {
			t.Errorf("Number of elements in document does not match. got %d; want %d",
				len(got.Document.EList), len(want.Document.EList))
		}
		if got.Document.Length != want.Document.Length {
			t.Errorf("Length of documents does not match. got %d; want %d",
				got.Document.Length, want.Document.Length)
		}
	})
}

func parseElementTest(t *testing.T) {
	t.Run("read-byte-error", func(t *testing.T) {
		want := errors.New("read-byte-error")
		p := &Parser{r: bufio.NewReader(&errReader{err: want})}
		_, got := p.ParseElement()
		if got != want {
			t.Errorf("Expected error. got %s; want %s", got, want)
		}
	})
	t.Run("null-byte", func(t *testing.T) {
		b := []byte{'\x00'}
		r := bytes.NewReader(b)
		p := &Parser{r: bufio.NewReader(r)}
		got, gotErr := p.ParseElement()
		if gotErr != nil {
			t.Errorf("Unexpected error. got %s; want %v", gotErr, nil)
		}
		if got != nil {
			t.Errorf("Unexpected element. got %v; want %v", got, nil)
		}
	})
	t.Run("parse-ename-error", func(t *testing.T) {
		b := []byte{'\x01'}
		want := errors.New("parse-ename-error")
		p := &Parser{r: bufio.NewReader(&errReader{b: b, err: want})}
		_, got := p.ParseElement()
		if got != want {
			t.Errorf("Expected error. got %s; want %s", got, want)
		}
	})

	doubleBytes := func() []byte {
		key := "foobar"
		b := make([]byte, 1+6+1+8)
		b[0] = '\x01'
		copy(b[1:7], []byte(key))
		b[7] = '\x00'
		binary.LittleEndian.PutUint64(b[8:], math.Float64bits(3.14159))
		return b
	}
	stringBytes := func() []byte {
		key := "foobar"
		val := "bazqux"
		b := make([]byte, 1+len(key)+1+4+len(val)+1)
		b[0] = '\x02'
		copy(b[1:7], []byte(key))
		b[7] = '\x00'
		binary.LittleEndian.PutUint32(b[8:12], uint32(len(val)+1))
		copy(b[12:18], []byte(val))
		b[18] = '\x00'
		return b
	}
	documentBytes := func() []byte {
		key := "foobar"
		b := make([]byte, 1+len(key)+1+5)
		b[0] = '\x03'
		copy(b[1:7], []byte(key))
		b[7] = '\x00'
		binary.LittleEndian.PutUint32(b[8:12], 5)
		b[12] = '\x00'
		return b
	}
	arrayBytes := func() []byte {
		key := "foobar"
		b := make([]byte, 1+len(key)+1+5)
		b[0] = '\x04'
		copy(b[1:7], []byte(key))
		b[7] = '\x00'
		binary.LittleEndian.PutUint32(b[8:12], 5)
		b[12] = '\x00'
		return b
	}
	binaryBytes := func() []byte {
		key := "foobar"
		bin := []byte{'\x00', '\x01', '\x02'}
		b := make([]byte, 1+len(key)+1+4+1+len(bin))
		b[0] = '\x05'
		copy(b[1:7], []byte(key))
		b[7] = '\x00'
		binary.LittleEndian.PutUint32(b[8:12], uint32(len(bin)))
		b[12] = '\x00'
		copy(b[13:], bin)
		return b
	}
	undefinedBytes := func() []byte {
		key := "foobar"
		b := make([]byte, 1+len(key)+1)
		b[0] = '\x06'
		copy(b[1:7], []byte(key))
		b[7] = '\x00'
		return b
	}
	objectIDBytes := func() []byte {
		key := "foobar"
		id := [12]byte{
			'\x01', '\x02', '\x03', '\x04',
			'\x05', '\x06', '\x07', '\x08',
			'\x09', '\x10', '\x11', '\x12',
		}
		b := make([]byte, 1+len(key)+1+12)
		b[0] = '\x07'
		copy(b[1:7], []byte(key))
		b[7] = '\x00'
		copy(b[8:], id[:])
		return b
	}
	boolBytes := func() []byte {
		key := "foobar"
		boolean := '\x01'
		b := make([]byte, 1+len(key)+1+1)
		b[0] = '\x08'
		copy(b[1:7], []byte(key))
		b[7] = '\x00'
		b[8] = byte(boolean)
		return b
	}
	datetimeBytes := func() []byte {
		key := "foobar"
		datetime := 1234567890
		b := make([]byte, 1+len(key)+1+8)
		b[0] = '\x09'
		copy(b[1:7], []byte(key))
		b[7] = '\x00'
		binary.LittleEndian.PutUint64(b[8:], uint64(datetime))
		return b
	}
	nullBytes := func() []byte {
		key := "foobar"
		b := make([]byte, 1+len(key)+1)
		b[0] = '\x0A'
		copy(b[1:7], []byte(key))
		b[7] = '\x00'
		return b
	}
	regexBytes := func() []byte {
		key := "foobar"
		pattern := "hello\x00"
		options := "world\x00"
		b := make([]byte, 1+len(key)+1)
		b[0] = '\x0B'
		copy(b[1:7], []byte(key))
		b[7] = '\x00'
		b = append(b, pattern...)
		b = append(b, options...)
		return b
	}
	dbpointerBytes := func() []byte {
		key := "foobar"
		str := "hello"
		id := [12]byte{
			'\x01', '\x02', '\x03', '\x04',
			'\x05', '\x06', '\x07', '\x08',
			'\x09', '\x10', '\x11', '\x12',
		}
		b := make([]byte, 1+len(key)+1+4+len(str)+1+12)
		b[0] = '\x0C'
		copy(b[1:7], []byte(key))
		b[7] = '\x00'
		binary.LittleEndian.PutUint32(b[8:12], uint32(len(str)+1))
		copy(b[12:17], []byte(str))
		b[17] = '\x00'
		copy(b[18:], id[:])
		return b
	}
	javascriptBytes := func() []byte {
		key := "foobar"
		js := `var hello = "world";`
		b := make([]byte, 1+len(key)+1+4+len(js)+1)
		b[0] = '\x0D'
		copy(b[1:7], []byte(key))
		b[7] = '\x00'
		binary.LittleEndian.PutUint32(b[8:12], uint32(len(js)+1))
		copy(b[12:32], []byte(js))
		b[32] = byte('\x00')
		return b
	}
	symbolBytes := func() []byte {
		key := "foobar"
		js := `12345`
		b := make([]byte, 1+len(key)+1+4+len(js)+1)
		b[0] = '\x0E'
		copy(b[1:7], []byte(key))
		b[7] = '\x00'
		binary.LittleEndian.PutUint32(b[8:12], uint32(len(js)+1))
		copy(b[12:17], []byte(js))
		b[17] = byte('\x00')
		return b
	}
	codewithscopeBytes := func() []byte {
		key := "foobar"
		js := `var hello = "world";`
		b := make([]byte, 1+len(key)+1+4+4+len(js)+1+5)
		b[0] = '\x0F'
		copy(b[1:7], []byte(key))
		b[7] = '\x00'
		binary.LittleEndian.PutUint32(b[8:12], uint32(4+len(js)+1+5))
		binary.LittleEndian.PutUint32(b[12:16], uint32(len(js)+1))
		copy(b[16:36], []byte(js))
		b[36] = byte('\x00')
		binary.LittleEndian.PutUint32(b[37:41], 5)
		b[41] = byte('\x00')
		return b
	}
	int32Bytes := func() []byte {
		key := "foobar"
		b := make([]byte, 1+6+1+4)
		b[0] = '\x10'
		copy(b[1:7], []byte(key))
		b[7] = '\x00'
		binary.LittleEndian.PutUint32(b[8:], 12345)
		return b
	}
	timestampBytes := func() []byte {
		key := "foobar"
		b := make([]byte, 1+6+1+8)
		b[0] = '\x11'
		copy(b[1:7], []byte(key))
		b[7] = '\x00'
		binary.LittleEndian.PutUint64(b[8:], 123456)
		return b
	}
	int64Bytes := func() []byte {
		key := "foobar"
		b := make([]byte, 1+6+1+8)
		b[0] = '\x12'
		copy(b[1:7], []byte(key))
		b[7] = '\x00'
		binary.LittleEndian.PutUint64(b[8:], 1234567890)
		return b
	}
	decimalBytes := func() []byte {
		key := "foobar"
		b := make([]byte, 1+6+1+8+8)
		b[0] = '\x13'
		copy(b[1:7], []byte(key))
		b[7] = '\x00'
		binary.LittleEndian.PutUint64(b[8:16], 12345)
		binary.LittleEndian.PutUint64(b[16:], 0)
		return b
	}
	minKeyBytes := func() []byte {
		key := "foobar"
		b := make([]byte, 1+len(key)+1)
		b[0] = '\xFF'
		copy(b[1:7], []byte(key))
		b[7] = '\x00'
		return b
	}
	maxKeyBytes := func() []byte {
		key := "foobar"
		b := make([]byte, 1+len(key)+1)
		b[0] = '\x7F'
		copy(b[1:7], []byte(key))
		b[7] = '\x00'
		return b
	}

	testCases := []struct {
		name string
		want ast.Element
		b    []byte
	}{
		{"double", &ast.FloatElement{
			Name: &ast.ElementKeyName{Key: "foobar"}, Double: 3.14159},
			doubleBytes(),
		},
		{"string", &ast.StringElement{
			Name: &ast.ElementKeyName{Key: "foobar"}, String: "bazqux"},
			stringBytes(),
		},
		{"document", &ast.DocumentElement{
			Name:     &ast.ElementKeyName{Key: "foobar"},
			Document: &ast.Document{Length: 5, EList: []ast.Element{}}},
			documentBytes(),
		},
		{"array", &ast.ArrayElement{
			Name:  &ast.ElementKeyName{Key: "foobar"},
			Array: &ast.Document{Length: 5, EList: []ast.Element{}}},
			arrayBytes(),
		},
		{"binary", &ast.BinaryElement{
			Name: &ast.ElementKeyName{Key: "foobar"},
			Binary: &ast.Binary{
				Subtype: ast.SubtypeGeneric, Data: []byte{'\x00', '\x01', '\x02'},
			}},
			binaryBytes(),
		},
		{"undefined", &ast.UndefinedElement{
			Name: &ast.ElementKeyName{Key: "foobar"}},
			undefinedBytes(),
		},
		{"object-ID", &ast.ObjectIDElement{
			Name: &ast.ElementKeyName{Key: "foobar"},
			ID: [12]byte{
				'\x01', '\x02', '\x03', '\x04',
				'\x05', '\x06', '\x07', '\x08',
				'\x09', '\x10', '\x11', '\x12',
			}},
			objectIDBytes(),
		},
		{"boolean", &ast.BoolElement{
			Name: &ast.ElementKeyName{Key: "foobar"},
			Bool: true},
			boolBytes(),
		},
		{"date-time", &ast.DateTimeElement{
			Name:     &ast.ElementKeyName{Key: "foobar"},
			DateTime: 1234567890},
			datetimeBytes(),
		},
		{"null", &ast.NullElement{
			Name: &ast.ElementKeyName{Key: "foobar"}},
			nullBytes(),
		},
		{"regex", &ast.RegexElement{
			Name:         &ast.ElementKeyName{Key: "foobar"},
			RegexPattern: &ast.CString{String: "hello"},
			RegexOptions: &ast.CString{String: "world"}},
			regexBytes(),
		},
		{"db-pointer", &ast.DBPointerElement{
			Name:   &ast.ElementKeyName{Key: "foobar"},
			String: "hello",
			Pointer: [12]byte{
				'\x01', '\x02', '\x03', '\x04',
				'\x05', '\x06', '\x07', '\x08',
				'\x09', '\x10', '\x11', '\x12',
			}},
			dbpointerBytes(),
		},
		{"javascript", &ast.JavaScriptElement{
			Name:   &ast.ElementKeyName{Key: "foobar"},
			String: `var hello = "world";`},
			javascriptBytes(),
		},
		{"symbol", &ast.SymbolElement{
			Name:   &ast.ElementKeyName{Key: "foobar"},
			String: `12345`},
			symbolBytes(),
		},
		{"code-with-scope", &ast.CodeWithScopeElement{
			Name: &ast.ElementKeyName{Key: "foobar"},
			CodeWithScope: &ast.CodeWithScope{
				String: `var hello = "world";`,
				Document: &ast.Document{
					Length: 5,
					EList:  []ast.Element{},
				},
			}},
			codewithscopeBytes(),
		},
		{"int32", &ast.Int32Element{
			Name:  &ast.ElementKeyName{Key: "foobar"},
			Int32: 12345},
			int32Bytes(),
		},
		{"timestamp", &ast.TimestampElement{
			Name:      &ast.ElementKeyName{Key: "foobar"},
			Timestamp: 123456},
			timestampBytes(),
		},
		{"int64", &ast.Int64Element{
			Name:  &ast.ElementKeyName{Key: "foobar"},
			Int64: 1234567890},
			int64Bytes(),
		},
		{"decimal128", &ast.DecimalElement{
			Name:       &ast.ElementKeyName{Key: "foobar"},
			Decimal128: decimal.NewDecimal128(0, 12345)},
			decimalBytes(),
		},
		{"min-key", &ast.MinKeyElement{
			Name: &ast.ElementKeyName{Key: "foobar"}},
			minKeyBytes(),
		},
		{"max-key", &ast.MaxKeyElement{
			Name: &ast.ElementKeyName{Key: "foobar"}},
			maxKeyBytes(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := bytes.NewReader(tc.b)
			p := &Parser{r: bufio.NewReader(r)}
			got, err := p.ParseElement()
			if err != nil {
				t.Errorf("Unexpected error: %s", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Results don't match. got %#v; want %#v", got, tc.want)
			}
		})
	}
}

type errReader struct {
	b   []byte
	err error
}

func (er *errReader) Read(b []byte) (int, error) {
	if len(er.b) > 0 {
		total := copy(b, er.b)
		er.b = er.b[total:]
		return total, nil
	}
	return 0, er.err
}
