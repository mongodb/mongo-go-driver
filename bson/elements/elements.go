// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package elements holds the logic to encode and decode the BSON element types
// from native Go to BSON binary and vice versa.
//
// These are low level helper methods, so they do not encode or decode BSON
// elements, only the specific types, e.g. these methods do not encode, decode,
// or identify a BSON element, so they won't read the identifier byte and they
// won't parse out the key string. There are encoder and decoder helper methods
// for the CString BSON element type, so this package can be used to parse
// keys.
package elements

import (
	"encoding/binary"
	"errors"
	"math"
	"unsafe"

	"github.com/10gen/mongo-go-driver/bson/decimal"
)

// ErrTooSmall indicates that slice provided to encode into is not large enough to fit the data.
var ErrTooSmall = errors.New("element: The provided slice is too small")

// These variables are used as namespaces for methods pertaining to encoding individual BSON types.
var (
	Double        double
	String        str
	Document      document
	Array         array
	Binary        bin
	ObjectID      objectid
	Boolean       boolean
	DateTime      datetime
	Regex         regex
	DBPointer     dbpointer
	JavaScript    javascript
	Symbol        symbol
	CodeWithScope codewithscope
	Int32         i32
	Timestamp     timestamp
	Int64         i64
	Decimal128    decimal128
	CString       cstring
	Byte          bsonbyte
)

type double struct{}
type str struct{}
type document struct{}
type array struct{}
type bin struct{}
type objectid struct{}
type boolean struct{}
type datetime struct{}
type regex struct{}
type dbpointer struct{}
type javascript struct{}
type symbol struct{}
type codewithscope struct{}
type i32 struct{}
type timestamp struct{}
type i64 struct{}
type decimal128 struct{}
type cstring struct{}
type bsonbyte struct{}

// Encodes a float64 into a BSON double element and serializes the bytes to the
// provided writer.
func (double) Encode(start uint, writer []byte, f float64) (int, error) {
	if len(writer) < int(start+8) {
		return 0, ErrTooSmall
	}

	bits := math.Float64bits(f)
	binary.LittleEndian.PutUint64(writer[start:start+8], bits)

	return 8, nil
}

func (double) Element(start uint, writer []byte, key string, f float64) (int, error) {
	var total int

	n, err := Byte.Encode(start, writer, '\x01')
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = CString.Encode(start, writer, key)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = Double.Encode(start, writer, f)
	total += n
	if err != nil {
		return total, err
	}

	return total, nil
}

func (str) Encode(start uint, writer []byte, s string) (int, error) {
	var total int

	written, err := Int32.Encode(start, writer, int32(len(s))+1)
	total += written
	if err != nil {
		return total, err
	}

	written, err = CString.Encode(start+uint(total), writer, s)
	total += written

	return total, nil
}

func (str) Element(start uint, writer []byte, key string, s string) (int, error) {
	var total int

	n, err := Byte.Encode(start, writer, '\x02')
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = CString.Encode(start, writer, key)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = String.Encode(start, writer, s)
	total += n
	if err != nil {
		return total, err
	}

	return total, nil
}

func (document) Encode(start uint, writer []byte, doc []byte) (int, error) {
	return encodeByteSlice(start, writer, doc)
}

func (document) Element(start uint, writer []byte, key string, doc []byte) (int, error) {
	var total int

	n, err := Byte.Encode(start, writer, '\x03')
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = CString.Encode(start, writer, key)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = Document.Encode(start, writer, doc)
	total += n
	if err != nil {
		return total, err
	}

	return total, nil
}

func (array) Encode(start uint, writer []byte, arr []byte) (int, error) {
	return Document.Encode(start, writer, arr)
}

func (array) Element(start uint, writer []byte, key string, arr []byte) (int, error) {
	var total int

	n, err := Byte.Encode(start, writer, '\x04')
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = CString.Encode(start, writer, key)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = Array.Encode(start, writer, arr)
	total += n
	if err != nil {
		return total, err
	}

	return total, nil
}

func (bin) Encode(start uint, writer []byte, b []byte, btype byte) (int, error) {
	if btype == 2 {
		return Binary.encodeSubtype2(start, writer, b)
	}

	var total int

	if len(writer) < int(start)+5+len(b) {
		return 0, ErrTooSmall
	}

	// write length
	n, err := Int32.Encode(start, writer, int32(len(b)))
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	writer[start] = btype
	start++
	total++

	total += copy(writer[start:], b)

	return total, nil
}

func (bin) encodeSubtype2(start uint, writer []byte, b []byte) (int, error) {
	var total int

	if len(writer) < int(start)+9+len(b) {
		return 0, ErrTooSmall
	}

	// write length
	n, err := Int32.Encode(start, writer, int32(len(b))+4)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	writer[start] = 2
	start++
	total++

	n, err = Int32.Encode(start, writer, int32(len(b)))
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	total += copy(writer[start:], b)

	return total, nil
}

func (bin) Element(start uint, writer []byte, key string, b []byte, btype byte) (int, error) {
	var total int

	n, err := Byte.Encode(start, writer, '\x05')
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = CString.Encode(start, writer, key)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = Binary.Encode(start, writer, b, btype)
	total += n
	if err != nil {
		return total, err
	}

	return total, nil
}

func (objectid) Encode(start uint, writer []byte, oid [12]byte) (int, error) {
	return encodeByteSlice(start, writer, oid[:])
}

func (objectid) Element(start uint, writer []byte, key string, oid [12]byte) (int, error) {
	var total int

	n, err := Byte.Encode(start, writer, '\x07')
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = CString.Encode(start, writer, key)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = ObjectID.Encode(start, writer, oid)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	return total, nil
}

func (boolean) Encode(start uint, writer []byte, b bool) (int, error) {
	if len(writer) < int(start)+1 {
		return 0, ErrTooSmall
	}

	if b {
		writer[start] = 1
	} else {
		writer[start] = 0
	}

	return 1, nil
}

func (boolean) Element(start uint, writer []byte, key string, b bool) (int, error) {
	var total int

	n, err := Byte.Encode(start, writer, '\x08')
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = CString.Encode(start, writer, key)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = Boolean.Encode(start, writer, b)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	return total, nil
}

func (datetime) Encode(start uint, writer []byte, dt int64) (int, error) {
	return Int64.Encode(start, writer, dt)
}

func (datetime) Element(start uint, writer []byte, key string, dt int64) (int, error) {
	var total int

	n, err := Byte.Encode(start, writer, '\x09')
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = CString.Encode(start, writer, key)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = DateTime.Encode(start, writer, dt)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	return total, nil
}

func (regex) Encode(start uint, writer []byte, pattern, options string) (int, error) {
	var total int

	written, err := CString.Encode(start, writer, pattern)
	total += written
	if err != nil {
		return total, err
	}

	written, err = CString.Encode(start+uint(total), writer, options)
	total += written

	return total, err
}

func (regex) Element(start uint, writer []byte, key string, pattern, options string) (int, error) {
	var total int

	n, err := Byte.Encode(start, writer, '\x0B')
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = CString.Encode(start, writer, key)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = CString.Encode(start, writer, pattern)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = CString.Encode(start, writer, options)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	return total, nil
}

func (dbpointer) Encode(start uint, writer []byte, ns string, oid [12]byte) (int, error) {
	var total int

	written, err := String.Encode(start, writer, ns)
	total += written
	if err != nil {
		return total, err
	}

	written, err = ObjectID.Encode(start+uint(written), writer, oid)
	total += written

	return total, err
}

func (dbpointer) Element(start uint, writer []byte, key string, ns string, oid [12]byte) (int, error) {
	var total int

	n, err := Byte.Encode(start, writer, '\x0C')
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = CString.Encode(start, writer, key)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = DBPointer.Encode(start, writer, ns, oid)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	return total, nil

}

func (javascript) Encode(start uint, writer []byte, code string) (int, error) {
	return String.Encode(start, writer, code)
}

func (javascript) Element(start uint, writer []byte, key string, code string) (int, error) {
	var total int

	n, err := Byte.Encode(start, writer, '\x0D')
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = CString.Encode(start, writer, key)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = JavaScript.Encode(start, writer, code)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	return total, nil
}

func (symbol) Encode(start uint, writer []byte, symbol string) (int, error) {
	return String.Encode(start, writer, symbol)
}

func (symbol) Element(start uint, writer []byte, key string, symbol string) (int, error) {
	var total int

	n, err := Byte.Encode(start, writer, '\x0E')
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = CString.Encode(start, writer, key)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = Symbol.Encode(start, writer, symbol)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	return total, nil
}

func (codewithscope) Encode(start uint, writer []byte, code string, doc []byte) (int, error) {
	var total int

	// Length of CodeWithScope is 4 + 4 + len(code) + 1 + len(doc)
	n, err := Int32.Encode(start, writer, 9+int32(len(code))+int32(len(doc)))
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = String.Encode(start, writer, code)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = encodeByteSlice(start, writer, doc)
	total += n

	return total, err
}

func (codewithscope) Element(start uint, writer []byte, key string, code string, scope []byte) (int, error) {
	var total int

	n, err := Byte.Encode(start, writer, '\x0F')
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = CString.Encode(start, writer, key)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = CodeWithScope.Encode(start, writer, code, scope)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	return total, nil
}

func (i32) Encode(start uint, writer []byte, i int32) (int, error) {
	if len(writer) < int(start)+4 {
		return 0, ErrTooSmall
	}

	binary.LittleEndian.PutUint32(writer[start:start+4], signed32ToUnsigned(i))

	return 4, nil

}

func (i32) Element(start uint, writer []byte, key string, i int32) (int, error) {
	var total int

	n, err := Byte.Encode(start, writer, '\x10')
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = CString.Encode(start, writer, key)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = Int32.Encode(start, writer, i)
	total += n
	if err != nil {
		return total, err
	}

	return total, nil
}

func (timestamp) Encode(start uint, writer []byte, t uint32, i uint32) (int, error) {
	var total int

	n, err := encodeUint32(start, writer, i)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = encodeUint32(start, writer, t)
	start += uint(n)
	total += n

	return total, err
}

func (timestamp) Element(start uint, writer []byte, key string, t uint32, i uint32) (int, error) {
	var total int

	n, err := Byte.Encode(start, writer, '\x11')
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = CString.Encode(start, writer, key)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = Timestamp.Encode(start, writer, t, i)
	total += n
	if err != nil {
		return total, err
	}

	return total, nil
}

func (i64) Encode(start uint, writer []byte, i int64) (int, error) {
	u := signed64ToUnsigned(i)

	return encodeUint64(start, writer, u)
}

func (i64) Element(start uint, writer []byte, key string, i int64) (int, error) {
	var total int

	n, err := Byte.Encode(start, writer, '\x12')
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = CString.Encode(start, writer, key)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = Int64.Encode(start, writer, i)
	total += n
	if err != nil {
		return total, err
	}

	return total, nil
}

func (decimal128) Encode(start uint, writer []byte, d decimal.Decimal128) (int, error) {
	var total int
	high, low := d.GetBytes()

	written, err := encodeUint64(start, writer, low)
	total += written
	if err != nil {
		return total, err
	}

	written, err = encodeUint64(start+uint(total), writer, high)
	total += written

	return total, err
}

func (decimal128) Element(start uint, writer []byte, key string, d decimal.Decimal128) (int, error) {
	var total int

	n, err := Byte.Encode(start, writer, '\x13')
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = CString.Encode(start, writer, key)
	start += uint(n)
	total += n
	if err != nil {
		return total, err
	}

	n, err = Decimal128.Encode(start, writer, d)
	total += n
	if err != nil {
		return total, err
	}

	return total, nil
}

func (cstring) Encode(start uint, writer []byte, str string) (int, error) {
	if len(writer) < int(start+1)+len(str) {
		return 0, ErrTooSmall
	}

	end := int(start) + len(str)
	written := copy(writer[start:end], str)
	writer[end] = '\x00'

	return written + 1, nil
}

func (bsonbyte) Encode(start uint, writer []byte, t byte) (int, error) {
	if len(writer) < int(start+1) {
		return 0, ErrTooSmall
	}

	writer[start] = t

	return 1, nil
}

func encodeByteSlice(start uint, writer []byte, b []byte) (int, error) {
	if len(writer) < int(start)+len(b) {
		return 0, ErrTooSmall
	}

	total := copy(writer[start:], b)

	return total, nil
}

func encodeUint32(start uint, writer []byte, u uint32) (int, error) {
	if len(writer) < int(start+4) {
		return 0, ErrTooSmall
	}

	binary.LittleEndian.PutUint32(writer[start:], u)

	return 4, nil

}

func encodeUint64(start uint, writer []byte, u uint64) (int, error) {
	if len(writer) < int(start+8) {
		return 0, ErrTooSmall
	}

	binary.LittleEndian.PutUint64(writer[start:], u)

	return 8, nil

}

func signed32ToUnsigned(i int32) uint32 {
	return *(*uint32)(unsafe.Pointer(&i))
}

func signed64ToUnsigned(i int64) uint64 {
	return *(*uint64)(unsafe.Pointer(&i))
}
