// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

// Value represents a BSON value. It can be obtained as part of a bson.Element or created for use
// in a bson.Array with the bson.VC constructors.
type Value struct {
	// NOTE: For subdocuments, arrays, and code with scope, the data slice of
	// bytes may contain just the key, or the key and the code in the case of
	// code with scope. If this is the case, the start will be 0, the value will
	// be the length of the slice, and d will be non-nil.

	// start is the offset into the data slice of bytes where this element
	// begins.
	start uint32
	// offset is the offset into the data slice of bytes where this element's
	// value begins.
	offset uint32
	// data is a potentially shared slice of bytes that contains the actual
	// element. Most of the methods of this type directly index into this slice
	// of bytes.
	data []byte

	d *Document
}

// Offset returns the offset to the beginning of the value in the underlying data. When called on
// a value obtained from a Reader, it can be used to find the value manually within the Reader's
// bytes.
func (v *Value) Offset() uint32 {
	return v.offset
}

// Interface returns the Go value of this Value as an empty interface.
func (v *Value) Interface() interface{} {
	if v == nil {
		return nil
	}

	switch v.Type() {
	case TypeDouble:
		return v.Double()
	case TypeString:
		return v.StringValue()
	case TypeEmbeddedDocument:
		return v.ReaderDocument().String()
	case TypeArray:
		return v.MutableArray().String()
	case TypeBinary:
		_, data := v.Binary()
		return data
	case TypeUndefined:
		return nil
	case TypeObjectID:
		return v.ObjectID()
	case TypeBoolean:
		return v.Boolean()
	case TypeDateTime:
		return v.DateTime()
	case TypeNull:
		return nil
	case TypeRegex:
		p, o := v.Regex()
		return Regex{Pattern: p, Options: o}
	case TypeDBPointer:
		db, pointer := v.DBPointer()
		return DBPointer{DB: db, Pointer: pointer}
	case TypeJavaScript:
		return v.JavaScript()
	case TypeSymbol:
		return v.Symbol()
	case TypeCodeWithScope:
		code, scope := v.MutableJavaScriptWithScope()
		return CodeWithScope{Code: code, Scope: scope}
	case TypeInt32:
		return v.Int32()
	case TypeTimestamp:
		t, i := v.Timestamp()
		return Timestamp{T: t, I: i}
	case TypeInt64:
		return v.Int64()
	case TypeDecimal128:
		return v.Decimal128()
	case TypeMinKey:
		return nil
	case TypeMaxKey:
		return nil
	default:
		return nil
	}
}

func (v *Value) validate(sizeOnly bool) (uint32, error) {
	if v.data == nil {
		return 0, ErrUninitializedElement
	}

	var total uint32

	switch v.data[v.start] {
	case '\x06', '\x0A', '\xFF', '\x7F':
	case '\x01':
		if int(v.offset+8) > len(v.data) {
			return total, NewErrTooSmall()
		}
		total += 8
	case '\x02', '\x0D', '\x0E':
		if int(v.offset+4) > len(v.data) {
			return total, NewErrTooSmall()
		}
		l := readi32(v.data[v.offset : v.offset+4])
		total += 4
		if int32(v.offset)+4+l > int32(len(v.data)) {
			return total, NewErrTooSmall()
		}
		// We check if the value that is the last element of the string is a
		// null terminator. We take the value offset, add 4 to account for the
		// length, add the length of the string, and subtract one since the size
		// isn't zero indexed.
		if !sizeOnly && v.data[v.offset+4+uint32(l)-1] != 0x00 {
			return total, ErrInvalidString
		}
		total += uint32(l)
	case '\x03':
		if v.d != nil {
			n, err := v.d.Validate()
			total += uint32(n)
			if err != nil {
				return total, err
			}
			break
		}

		if int(v.offset+4) > len(v.data) {
			return total, NewErrTooSmall()
		}
		l := readi32(v.data[v.offset : v.offset+4])
		total += 4
		if l < 5 {
			return total, ErrInvalidReadOnlyDocument
		}
		if int32(v.offset)+l > int32(len(v.data)) {
			return total, NewErrTooSmall()
		}
		if !sizeOnly {
			n, err := Reader(v.data[v.offset : v.offset+uint32(l)]).Validate()
			total += n - 4
			if err != nil {
				return total, err
			}
			break
		}
		total += uint32(l) - 4
	case '\x04':
		if v.d != nil {
			n, err := (&Array{v.d}).Validate()
			total += uint32(n)
			if err != nil {
				return total, err
			}
			break
		}

		if int(v.offset+4) > len(v.data) {
			return total, NewErrTooSmall()
		}
		l := readi32(v.data[v.offset : v.offset+4])
		total += 4
		if l < 5 {
			return total, ErrInvalidReadOnlyDocument
		}
		if int32(v.offset)+l > int32(len(v.data)) {
			return total, NewErrTooSmall()
		}
		if !sizeOnly {
			n, err := Reader(v.data[v.offset : v.offset+uint32(l)]).Validate()
			total += n - 4
			if err != nil {
				return total, err
			}
			break
		}
		total += uint32(l) - 4
	case '\x05':
		if int(v.offset+5) > len(v.data) {
			return total, NewErrTooSmall()
		}
		l := readi32(v.data[v.offset : v.offset+4])
		total += 5
		if v.data[v.offset+4] > '\x05' && v.data[v.offset+4] < '\x80' {
			return total, ErrInvalidBinarySubtype
		}
		if int32(v.offset)+5+l > int32(len(v.data)) {
			return total, NewErrTooSmall()
		}
		total += uint32(l)
	case '\x07':
		if int(v.offset+12) > len(v.data) {
			return total, NewErrTooSmall()
		}
		total += 12
	case '\x08':
		if int(v.offset+1) > len(v.data) {
			return total, NewErrTooSmall()
		}
		total++
		if v.data[v.offset] != '\x00' && v.data[v.offset] != '\x01' {
			return total, ErrInvalidBooleanType
		}
	case '\x09':
		if int(v.offset+8) > len(v.data) {
			return total, NewErrTooSmall()
		}
		total += 8
	case '\x0B':
		i := v.offset
		for ; int(i) < len(v.data) && v.data[i] != '\x00'; i++ {
			total++
		}
		if int(i) == len(v.data) || v.data[i] != '\x00' {
			return total, ErrInvalidString
		}
		i++
		total++
		for ; int(i) < len(v.data) && v.data[i] != '\x00'; i++ {
			total++
		}
		if int(i) == len(v.data) || v.data[i] != '\x00' {
			return total, ErrInvalidString
		}
		total++
	case '\x0C':
		if int(v.offset+4) > len(v.data) {
			return total, NewErrTooSmall()
		}
		l := readi32(v.data[v.offset : v.offset+4])
		total += 4
		if int32(v.offset)+4+l+12 > int32(len(v.data)) {
			return total, NewErrTooSmall()
		}
		total += uint32(l) + 12
	case '\x0F':
		if v.d != nil {
			// NOTE: For code with scope specifically, we write the length as
			// we are marshaling the element and the constructor doesn't know
			// the length of the document when it constructs the element.
			// Because of that we don't check the length here and just validate
			// the string and the document.
			if int(v.offset+8) > len(v.data) {
				return total, NewErrTooSmall()
			}
			total += 8
			sLength := readi32(v.data[v.offset+4 : v.offset+8])
			if int(sLength) > len(v.data)+8 {
				return total, NewErrTooSmall()
			}
			total += uint32(sLength)
			if !sizeOnly && v.data[v.offset+8+uint32(sLength)-1] != 0x00 {
				return total, ErrInvalidString
			}

			n, err := v.d.Validate()
			total += uint32(n)
			if err != nil {
				return total, err
			}
			break
		}
		if int(v.offset+4) > len(v.data) {
			return total, NewErrTooSmall()
		}
		l := readi32(v.data[v.offset : v.offset+4])
		total += 4
		if int32(v.offset)+l > int32(len(v.data)) {
			return total, NewErrTooSmall()
		}
		if !sizeOnly {
			sLength := readi32(v.data[v.offset+4 : v.offset+8])
			total += 4
			// If the length of the string is larger than the total length of the
			// field minus the int32 for length, 5 bytes for a minimum document
			// size, and an int32 for the string length the value is invalid.
			//
			// TODO(skriptble): We should actually validate that the string
			// doesn't consume any of the bytes used by the document.
			if sLength > l-13 {
				return total, ErrStringLargerThanContainer
			}
			// We check if the value that is the last element of the string is a
			// null terminator. We take the value offset, add 4 to account for the
			// length, add the length of the string, and subtract one since the size
			// isn't zero indexed.
			if v.data[v.offset+8+uint32(sLength)-1] != 0x00 {
				return total, ErrInvalidString
			}
			total += uint32(sLength)
			n, err := Reader(v.data[v.offset+8+uint32(sLength) : v.offset+uint32(l)]).Validate()
			total += n
			if err != nil {
				return total, err
			}
			break
		}
		total += uint32(l) - 4
	case '\x10':
		if int(v.offset+4) > len(v.data) {
			return total, NewErrTooSmall()
		}
		total += 4
	case '\x11', '\x12':
		if int(v.offset+8) > len(v.data) {
			return total, NewErrTooSmall()
		}
		total += 8
	case '\x13':
		if int(v.offset+16) > len(v.data) {
			return total, NewErrTooSmall()
		}
		total += 16

	default:
		return total, ErrInvalidElement
	}

	return total, nil
}

// valueSize returns the size of the value in bytes.
func (v *Value) valueSize() (uint32, error) {
	return v.validate(true)
}

// Type returns the identifying element byte for this element.
// It panics if e is uninitialized.
func (v *Value) Type() Type {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	return Type(v.data[v.start])
}

// IsNumber returns true if the type of v is a numberic BSON type.
func (v *Value) IsNumber() bool {
	switch v.Type() {
	case TypeDouble, TypeInt32, TypeInt64, TypeDecimal128:
		return true
	default:
		return false
	}
}

// Double returns the float64 value for this element.
// It panics if e's BSON type is not double ('\x01') or if e is uninitialized.
func (v *Value) Double() float64 {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x01' {
		panic(ElementTypeError{"compact.Element.double", Type(v.data[v.start])})
	}
	return math.Float64frombits(v.getUint64())
}

// DoubleOK is the same as Double, but returns a boolean instead of panicking.
func (v *Value) DoubleOK() (float64, bool) {
	if v == nil || v.offset == 0 || v.data == nil || Type(v.data[v.start]) != TypeDouble {
		return 0, false
	}
	return v.Double(), true
}

// StringValue returns the string balue for this element.
// It panics if e's BSON type is not StringValue ('\x02') or if e is uninitialized.
//
// NOTE: This method is called StringValue to avoid it implementing the
// fmt.Stringer interface.
func (v *Value) StringValue() string {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x02' {
		panic(ElementTypeError{"compact.Element.String", Type(v.data[v.start])})
	}
	l := readi32(v.data[v.offset : v.offset+4])
	return string(v.data[v.offset+4 : int32(v.offset)+4+l-1])
}

// StringValueOK is the same as StringValue, but returns a boolean instead of
// panicking.
func (v *Value) StringValueOK() (string, bool) {
	if v == nil || v.offset == 0 || v.data == nil || Type(v.data[v.start]) != TypeString {
		return "", false
	}
	return v.StringValue(), true
}

// ReaderDocument returns the BSON document the Value represents as a bson.Reader. It panics if the
// value is a BSON type other than document.
func (v *Value) ReaderDocument() Reader {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}

	if v.data[v.start] != '\x03' {
		panic(ElementTypeError{"compact.Element.Document", Type(v.data[v.start])})
	}

	var r Reader
	if v.d == nil {
		l := readi32(v.data[v.offset : v.offset+4])
		r = Reader(v.data[v.offset : v.offset+uint32(l)])
	} else {
		scope, err := v.d.MarshalBSON()
		if err != nil {
			panic(err)
		}

		r = Reader(scope)
	}

	return r
}

// ReaderDocumentOK is the same as ReaderDocument, except it returns a boolean
// instead of panicking.
func (v *Value) ReaderDocumentOK() (Reader, bool) {
	if v == nil || v.offset == 0 || v.data == nil || Type(v.data[v.start]) != TypeEmbeddedDocument {
		return nil, false
	}
	return v.ReaderDocument(), true
}

// MutableDocument returns the subdocument for this element.
func (v *Value) MutableDocument() *Document {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x03' {
		panic(ElementTypeError{"compact.Element.Document", Type(v.data[v.start])})
	}
	if v.d == nil {
		var err error
		l := int32(binary.LittleEndian.Uint32(v.data[v.offset : v.offset+4]))
		v.d, err = ReadDocument(v.data[v.offset : v.offset+uint32(l)])
		if err != nil {
			panic(err)
		}
	}
	return v.d
}

// MutableDocumentOK is the same as MutableDocument, except it returns a boolean
// instead of panicking.
func (v *Value) MutableDocumentOK() (*Document, bool) {
	if v == nil || v.offset == 0 || v.data == nil || Type(v.data[v.start]) != TypeEmbeddedDocument {
		return nil, false
	}
	return v.MutableDocument(), true
}

// ReaderArray returns the BSON document the Value represents as a bson.Reader. It panics if the
// value is a BSON type other than array.
func (v *Value) ReaderArray() Reader {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}

	if v.data[v.start] != '\x04' {
		panic(ElementTypeError{"compact.Element.Array", Type(v.data[v.start])})
	}

	var r Reader
	if v.d == nil {
		l := readi32(v.data[v.offset : v.offset+4])
		r = Reader(v.data[v.offset : v.offset+uint32(l)])
	} else {
		scope, err := v.d.MarshalBSON()
		if err != nil {
			panic(err)
		}

		r = Reader(scope)
	}

	return r
}

// ReaderArrayOK is the same as ReaderArray, except it returns a boolean instead
// of panicking.
func (v *Value) ReaderArrayOK() (Reader, bool) {
	if v == nil || v.offset == 0 || v.data == nil || Type(v.data[v.start]) != TypeArray {
		return nil, false
	}
	return v.ReaderArray(), true
}

// MutableArray returns the array for this element.
func (v *Value) MutableArray() *Array {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x04' {
		panic(ElementTypeError{"compact.Element.Array", Type(v.data[v.start])})
	}
	if v.d == nil {
		var err error
		l := int32(binary.LittleEndian.Uint32(v.data[v.offset : v.offset+4]))
		v.d, err = ReadDocument(v.data[v.offset : v.offset+uint32(l)])
		if err != nil {
			panic(err)
		}
	}
	return &Array{v.d}
}

// MutableArrayOK is the same as MutableArray, except it returns a boolean
// instead of panicking.
func (v *Value) MutableArrayOK() (*Array, bool) {
	if v == nil || v.offset == 0 || v.data == nil || Type(v.data[v.start]) != TypeArray {
		return nil, false
	}
	return v.MutableArray(), true
}

// Binary returns the BSON binary value the Value represents. It panics if the value is a BSON type
// other than binary.
func (v *Value) Binary() (subtype byte, data []byte) {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x05' {
		panic(ElementTypeError{"compact.Element.binary", Type(v.data[v.start])})
	}
	l := readi32(v.data[v.offset : v.offset+4])
	st := v.data[v.offset+4]
	b := make([]byte, l)
	copy(b, v.data[v.offset+5:int32(v.offset)+5+l])
	return st, b
}

// ObjectID returns the BSON objectid value the Value represents. It panics if the value is a BSON
// type other than objectid.
func (v *Value) ObjectID() objectid.ObjectID {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x07' {
		panic(ElementTypeError{"compact.Element.ObejctID", Type(v.data[v.start])})
	}
	var arr [12]byte
	copy(arr[:], v.data[v.offset:v.offset+12])
	return arr
}

// ObjectIDOK is the same as ObjectID, except it returns a boolean instead of
// panicking.
func (v *Value) ObjectIDOK() (objectid.ObjectID, bool) {
	var empty objectid.ObjectID
	if v == nil || v.offset == 0 || v.data == nil || Type(v.data[v.start]) != TypeObjectID {
		return empty, false
	}
	return v.ObjectID(), true
}

// Boolean returns the boolean value the Value represents. It panics if the
// value is a BSON type other than boolean.
func (v *Value) Boolean() bool {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x08' {
		panic(ElementTypeError{"compact.Element.Boolean", Type(v.data[v.start])})
	}
	return v.data[v.offset] == '\x01'
}

// BooleanOK is the same as Boolean, except it returns a boolean instead of
// panicking.
func (v *Value) BooleanOK() (bool, bool) {
	if v == nil || v.offset == 0 || v.data == nil || Type(v.data[v.start]) != TypeBoolean {
		return false, false
	}
	return v.Boolean(), true
}

// DateTime returns the BSON datetime value the Value represents. It panics if the value is a BSON
// type other than datetime.
func (v *Value) DateTime() time.Time {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x09' {
		panic(ElementTypeError{"compact.Element.dateTime", Type(v.data[v.start])})
	}
	i := v.getUint64()
	return time.Unix(int64(i)/1000, int64(i)%1000*1000000)
}

// Regex returns the BSON regex value the Value represents. It panics if the value is a BSON
// type other than regex.
func (v *Value) Regex() (pattern, options string) {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x0B' {
		panic(ElementTypeError{"compact.Element.regex", Type(v.data[v.start])})
	}
	// TODO(skriptble): Use the elements package here.
	var pstart, pend, ostart, oend uint32
	i := v.offset
	pstart = i
	for ; v.data[i] != '\x00'; i++ {
	}
	pend = i
	i++
	ostart = i
	for ; v.data[i] != '\x00'; i++ {
	}
	oend = i

	return string(v.data[pstart:pend]), string(v.data[ostart:oend])
}

// DateTimeOK is the same as DateTime, except it returns a boolean instead of
// panicking.
func (v *Value) DateTimeOK() (time.Time, bool) {
	if v == nil || v.offset == 0 || v.data == nil || Type(v.data[v.start]) != TypeDateTime {
		return time.Time{}, false
	}
	return v.DateTime(), true
}

// DBPointer returns the BSON dbpointer value the Value represents. It panics if the value is a BSON
// type other than DBPointer.
func (v *Value) DBPointer() (string, objectid.ObjectID) {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x0C' {
		panic(ElementTypeError{"compact.Element.dbPointer", Type(v.data[v.start])})
	}
	l := readi32(v.data[v.offset : v.offset+4])
	var p [12]byte
	copy(p[:], v.data[v.offset+4+uint32(l):v.offset+4+uint32(l)+12])
	return string(v.data[v.offset+4 : int32(v.offset)+4+l-1]), p
}

// DBPointerOK is the same as DBPoitner, except that it returns a boolean
// instead of panicking.
func (v *Value) DBPointerOK() (string, objectid.ObjectID, bool) {
	var empty objectid.ObjectID
	if v == nil || v.offset == 0 || v.data == nil || Type(v.data[v.start]) != TypeDBPointer {
		return "", empty, false
	}
	s, o := v.DBPointer()
	return s, o, true
}

// JavaScript returns the BSON JavaScript code value the Value represents. It panics if the value is
// a BSON type other than JavaScript code.
func (v *Value) JavaScript() string {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x0D' {
		panic(ElementTypeError{"compact.Element.JavaScript", Type(v.data[v.start])})
	}
	l := readi32(v.data[v.offset : v.offset+4])
	return string(v.data[v.offset+4 : int32(v.offset)+4+l-1])
}

// JavaScriptOK is the same as Javascript, excepti that it returns a boolean
// instead of panicking.
func (v *Value) JavaScriptOK() (string, bool) {
	if v == nil || v.offset == 0 || v.data == nil || Type(v.data[v.start]) != TypeJavaScript {
		return "", false
	}
	return v.JavaScript(), true
}

// Symbol returns the BSON symbol value the Value represents. It panics if the value is a BSON
// type other than symbol.
func (v *Value) Symbol() string {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x0E' {
		panic(ElementTypeError{"compact.Element.symbol", Type(v.data[v.start])})
	}
	l := readi32(v.data[v.offset : v.offset+4])
	return string(v.data[v.offset+4 : int32(v.offset)+4+l-1])
}

// ReaderJavaScriptWithScope returns the BSON JavaScript code with scope the Value represents, with
// the scope being returned as a bson.Reader. It panics if the value is a BSON type other than
// JavaScript code with scope.
func (v *Value) ReaderJavaScriptWithScope() (string, Reader) {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}

	if v.data[v.start] != '\x0F' {
		panic(ElementTypeError{"compact.Element.JavaScriptWithScope", Type(v.data[v.start])})
	}

	sLength := readi32(v.data[v.offset+4 : v.offset+8])
	// If the length of the string is larger than the total length of the
	// field minus the int32 for length, 5 bytes for a minimum document
	// size, and an int32 for the string length the value is invalid.
	str := string(v.data[v.offset+8 : v.offset+8+uint32(sLength)-1])

	var r Reader
	if v.d == nil {
		l := readi32(v.data[v.offset : v.offset+4])
		r = Reader(v.data[v.offset+8+uint32(sLength) : v.offset+uint32(l)])
	} else {
		scope, err := v.d.MarshalBSON()
		if err != nil {
			panic(err)
		}

		r = Reader(scope)
	}

	return str, r
}

// ReaderJavaScriptWithScopeOK is the same as ReaderJavaScriptWithScope,
// except that it returns a boolean instead of panicking.
func (v *Value) ReaderJavaScriptWithScopeOK() (string, Reader, bool) {
	if v == nil || v.offset == 0 || v.data == nil || Type(v.data[v.start]) != TypeCodeWithScope {
		return "", nil, false
	}
	s, r := v.ReaderJavaScriptWithScope()
	return s, r, true
}

// MutableJavaScriptWithScope returns the javascript code and the scope document for
// this element.
func (v *Value) MutableJavaScriptWithScope() (code string, d *Document) {
	if v == nil || v.offset == 0 {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x0F' {
		panic(ElementTypeError{"compact.Element.JavaScriptWithScope", Type(v.data[v.start])})
	}
	// TODO(skriptble): This is wrong and could cause a panic.
	l := int32(binary.LittleEndian.Uint32(v.data[v.offset : v.offset+4]))
	// TODO(skriptble): This is wrong and could cause a panic.
	sLength := int32(binary.LittleEndian.Uint32(v.data[v.offset+4 : v.offset+8]))
	// If the length of the string is larger than the total length of the
	// field minus the int32 for length, 5 bytes for a minimum document
	// size, and an int32 for the string length the value is invalid.
	str := string(v.data[v.offset+4 : v.offset+4+uint32(sLength)])
	if v.d == nil {
		var err error
		v.d, err = ReadDocument(v.data[v.offset+4+uint32(sLength) : v.offset+uint32(l)])
		if err != nil {
			panic(err)
		}
	}
	return str, v.d
}

// MutableJavaScriptWithScopeOK is the same as MutableJavascriptWithScope,
// except that it returns a boolean instead of panicking.
func (v *Value) MutableJavaScriptWithScopeOK() (string, *Document, bool) {
	if v == nil || v.offset == 0 || v.data == nil || Type(v.data[v.start]) != TypeCodeWithScope {
		return "", nil, false
	}
	s, d := v.MutableJavaScriptWithScope()
	return s, d, true
}

// Int32 returns the int32 the Value represents. It panics if the value is a BSON type other than
// int32.
func (v *Value) Int32() int32 {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x10' {
		panic(ElementTypeError{"compact.Element.int32", Type(v.data[v.start])})
	}
	return readi32(v.data[v.offset : v.offset+4])
}

// Int32OK is the same as Int32, except that it returns a boolean instead of
// panicking.
func (v *Value) Int32OK() (int32, bool) {
	if v == nil || v.offset == 0 || v.data == nil || Type(v.data[v.start]) != TypeInt32 {
		return 0, false
	}
	return v.Int32(), true
}

// Timestamp returns the BSON timestamp value the Value represents. It panics if the value is a
// BSON type other than timestamp.
func (v *Value) Timestamp() (uint32, uint32) {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x11' {
		panic(ElementTypeError{"compact.Element.timestamp", Type(v.data[v.start])})
	}
	return binary.LittleEndian.Uint32(v.data[v.offset+4 : v.offset+8]), binary.LittleEndian.Uint32(v.data[v.offset : v.offset+4])
}

// TimestampOK is the same as Timestamp, except that it returns a boolean
// instead of panicking.
func (v *Value) TimestampOK() (uint32, uint32, bool) {
	if v == nil || v.offset == 0 || v.data == nil || Type(v.data[v.start]) != TypeTimestamp {
		return 0, 0, false
	}
	t, i := v.Timestamp()
	return t, i, true
}

// Int64 returns the int64 the Value represents. It panics if the value is a BSON type other than
// int64.
func (v *Value) Int64() int64 {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x12' {
		panic(ElementTypeError{"compact.Element.int64Type", Type(v.data[v.start])})
	}
	return int64(v.getUint64())
}

func (v *Value) getUint64() uint64 {
	return binary.LittleEndian.Uint64(v.data[v.offset : v.offset+8])
}

// Int64OK is the same as Int64, except that it returns a boolean instead of
// panicking.
func (v *Value) Int64OK() (int64, bool) {
	if v == nil || v.offset == 0 || v.data == nil || Type(v.data[v.start]) != TypeInt64 {
		return 0, false
	}
	return v.Int64(), true
}

// Decimal128 returns the decimal the Value represents. It panics if the value is a BSON type other than
// decimal.
func (v *Value) Decimal128() decimal.Decimal128 {
	if v == nil || v.offset == 0 || v.data == nil {
		panic(ErrUninitializedElement)
	}
	if v.data[v.start] != '\x13' {
		panic(ElementTypeError{"compact.Element.Decimal128", Type(v.data[v.start])})
	}
	l := binary.LittleEndian.Uint64(v.data[v.offset : v.offset+8])
	h := binary.LittleEndian.Uint64(v.data[v.offset+8 : v.offset+16])
	return decimal.NewDecimal128(h, l)
}

// Decimal128OK is the same as Decimal128, except that it returns a boolean
// instead of panicking.
func (v *Value) Decimal128OK() (decimal.Decimal128, bool) {
	if v == nil || v.offset == 0 || v.data == nil || Type(v.data[v.start]) != TypeDecimal128 {
		return decimal.NewDecimal128(0, 0), false
	}
	return v.Decimal128(), true
}

func (v *Value) asString() (string, error) {
	var str string
	var err error
	switch v.Type() {
	case TypeString:
		str = v.StringValue()
	case TypeDouble:
		str = fmt.Sprintf("%f", v.Double())
	case TypeInt32:
		str = fmt.Sprintf("%d", v.Int32())
	case TypeInt64:
		str = fmt.Sprintf("%d", v.Int64())
	case TypeBoolean:
		str = fmt.Sprintf("%t", v.Boolean())
	case TypeNull:
		str = "null"
	default:
		err = fmt.Errorf("cannot Stringify %s yet", v.Type())
	}
	return str, err
}

func (v *Value) setString(str string) {
	size := 2 + 4 + len(str) + 1
	b := make([]byte, size)
	b[0], b[1] = byte(TypeString), 0x00
	copy(b[2:], str)
	b[size-1] = 0x00

	v.start = 0
	v.offset = 2
	v.data = b
}

func (v *Value) setDouble(f float64) {
	header := v.data[v.start:v.offset]
	if v.start != 0 || len(v.data) < len(header)+8 {
		b := make([]byte, len(header)+8)
		copy(b, header)
		v.offset = v.offset - v.start
		v.start = 0
		v.data = b
	}
	v.data[v.start] = byte(TypeDouble)
	bits := math.Float64bits(f)
	binary.LittleEndian.PutUint64(v.data[v.offset:v.offset+8], bits)
}

func (v *Value) setInt32(i int32) {
	header := v.data[v.start:v.offset]
	if v.start != 0 || len(v.data) < len(header)+4 {
		b := make([]byte, len(header)+4)
		copy(b, header)
		v.offset = v.offset - v.start
		v.start = 0
		v.data = b
	}
	v.data[v.start] = byte(TypeInt32)
	binary.LittleEndian.PutUint32(v.data[v.offset:v.offset+4], uint32(i))
}

func (v *Value) setInt64(i int64) {
	header := v.data[v.start:v.offset]
	if v.start != 0 || len(v.data) < len(header)+8 {
		b := make([]byte, len(header)+8)
		copy(b, header)
		v.offset = v.offset - v.start
		v.start = 0
		v.data = b
	}
	v.data[v.start] = byte(TypeInt64)
	binary.LittleEndian.PutUint64(v.data[v.offset:v.offset+8], uint64(i))
}

func (v *Value) addNumber(v2 *Value) {
	// TODO: decimal128
	switch v.Type() {
	case TypeDouble:
		switch v2.Type() {
		case TypeDouble:
			v.setDouble(v.Double() + v2.Double())
		case TypeInt32:
			v.setDouble(v.Double() + float64(v2.Int32()))
		case TypeInt64:
			v.setDouble(v.Double() + float64(v2.Int64()))
		}
	case TypeInt32:
		switch v2.Type() {
		case TypeDouble:
			v.setDouble(float64(v.Int32()) + v2.Double())
		case TypeInt32:
			v.setInt32(v.Int32() + v2.Int32())
		case TypeInt64:
			v.setInt64(int64(v.Int32()) + v2.Int64())
		}
	case TypeInt64:
		switch v2.Type() {
		case TypeDouble:
			v.setDouble(float64(v.Int64()) + v.Double())
		case TypeInt32:
			v.setInt64(v.Int64() + int64(v2.Int32()))
		case TypeInt64:
			v.setInt64(v.Int64() + v2.Int64())
		}
	}
}

// Add will add this Value to another. This is currently only implemented for
// strings and numbers. If either value is a string, the other type is coerced
// into a string and added to the other.
func (v *Value) Add(v2 *Value) error {
	if v.Type() == TypeString || v2.Type() == TypeString {
		str1, err := v.asString()
		if err != nil {
			return err
		}
		str2, err := v2.asString()
		if err != nil {
			return err
		}
		v.setString(str1 + str2)
		return nil
	}

	if v.IsNumber() && v2.IsNumber() {
		v.addNumber(v2)
		return nil
	}

	return fmt.Errorf("cannot Add values of types %s and %s yet", v.Type(), v2.Type())
}

func (v *Value) equal(v2 *Value) bool {
	if v == nil && v2 == nil {
		return true
	}

	if v == nil || v2 == nil {
		return false
	}

	if v.start != v2.start {
		return false
	}

	if v.offset != v2.offset {
		return false
	}

	if v.d != nil && !v.d.Equal(v2.d) {
		return false
	}

	return bytes.Equal(v.data, v2.data)
}
