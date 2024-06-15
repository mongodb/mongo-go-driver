// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

var _ ValueReader = (*valueReader)(nil)

// ErrEOA is the error returned when the end of a BSON array has been reached.
var ErrEOA = errors.New("end of array")

// ErrEOD is the error returned when the end of a BSON document has been reached.
var ErrEOD = errors.New("end of document")

type vrState struct {
	mode  mode
	vType Type
	end   int64
}

// valueReader is for reading BSON values.
type valueReader struct {
	offset int64
	d      []byte

	readerErr error
	r         io.Reader

	stack []vrState
	frame int64
}

// NewValueReader returns a ValueReader using b for the underlying BSON
// representation.
func NewValueReader(r io.Reader) ValueReader {
	return newValueReader(r)
}

// NewBSONValueReader returns a ValueReader that starts in the Value mode instead of in top
// level document mode. This enables the creation of a ValueReader for a single BSON value.
func NewBSONValueReader(t Type, r io.Reader) ValueReader {
	vr := newValueReader(r)
	if len(vr.stack) == 0 {
		vr.stack = make([]vrState, 1, 5)
	}
	vr.stack[0].mode = mValue
	vr.stack[0].vType = t
	return vr
}

func newValueReader(r io.Reader) *valueReader {
	stack := make([]vrState, 1, 5)
	stack[0] = vrState{
		mode: mTopLevel,
	}
	return &valueReader{
		r:     r,
		stack: stack,
	}
}

func newValueReaderFromSlice(buf []byte) *valueReader {
	vr := newValueReader(nil)
	vr.d = buf
	return vr
}

func (vr *valueReader) prepload(length int32) (int32, error) {
	const chunckSize = 512

	if vr.offset+int64(length) <= int64(len(vr.d)) {
		return length, nil
	}

	if vr.readerErr != nil {
		return 0, vr.readerErr
	}
	if vr.r == nil {
		vr.readerErr = io.EOF
		return 0, vr.readerErr
	}

	size := len(vr.d)
	var need int64 = chunckSize
	if l := int64(length) + vr.offset - int64(size); l > need {
		need = l
	}
	buf := make([]byte, need)
	n, err := vr.r.Read(buf)
	if err != nil {
		vr.readerErr = err
	}
	vr.d = append(vr.d, buf[0:n]...)
	if l := int64(n+size) - vr.offset; l < int64(length) {
		length = int32(l)
	}
	return length, err
}

func (vr *valueReader) indexByteAfter(offset int64, c byte) (int64, error) {
	const chunckSize = 512
	for idx := -1; idx < 0; {
		idx = bytes.IndexByte(vr.d[offset:], c)
		if idx < 0 {
			n, err := vr.prepload(chunckSize)
			if n == 0 && err != nil {
				return 0, err
			}
		} else {
			offset += int64(idx)
		}
	}
	return offset, nil
}

func (vr *valueReader) advanceFrame() {
	if vr.frame+1 >= int64(len(vr.stack)) { // We need to grow the stack
		length := len(vr.stack)
		if length+1 >= cap(vr.stack) {
			// double it
			buf := make([]vrState, 2*cap(vr.stack)+1)
			copy(buf, vr.stack)
			vr.stack = buf
		}
		vr.stack = vr.stack[:length+1]
	}
	vr.frame++

	// Clean the stack
	vr.stack[vr.frame].mode = 0
	vr.stack[vr.frame].vType = 0
	vr.stack[vr.frame].end = 0
}

func (vr *valueReader) pushDocument() error {
	vr.advanceFrame()

	vr.stack[vr.frame].mode = mDocument

	size, err := vr.readLength()
	if err != nil {
		return err
	}
	vr.stack[vr.frame].end = int64(size) + vr.offset - 4

	return nil
}

func (vr *valueReader) pushArray() error {
	vr.advanceFrame()

	vr.stack[vr.frame].mode = mArray

	size, err := vr.readLength()
	if err != nil {
		return err
	}
	vr.stack[vr.frame].end = int64(size) + vr.offset - 4

	return nil
}

func (vr *valueReader) pushElement(t Type) {
	vr.advanceFrame()

	vr.stack[vr.frame].mode = mElement
	vr.stack[vr.frame].vType = t
}

func (vr *valueReader) pushValue(t Type) {
	vr.advanceFrame()

	vr.stack[vr.frame].mode = mValue
	vr.stack[vr.frame].vType = t
}

func (vr *valueReader) pushCodeWithScope() (int64, error) {
	vr.advanceFrame()

	vr.stack[vr.frame].mode = mCodeWithScope

	size, err := vr.readLength()
	if err != nil {
		return 0, err
	}
	vr.stack[vr.frame].end = int64(size) + vr.offset - 4

	return int64(size), nil
}

func (vr *valueReader) pop() {
	switch vr.stack[vr.frame].mode {
	case mElement, mValue:
		vr.frame--
	case mDocument, mArray, mCodeWithScope:
		vr.frame -= 2 // we pop twice to jump over the vrElement: vrDocument -> vrElement -> vrDocument/TopLevel/etc...
	}
	if vr.frame < 0 {
		vr.frame = 0
	}
	if vr.frame == 0 && (vr.stack[vr.frame].end <= vr.offset) {
		vr.d = vr.d[vr.stack[vr.frame].end:]
		vr.offset -= vr.stack[vr.frame].end
		vr.stack[vr.frame].end = 0
	}
}

func (vr *valueReader) invalidTransitionErr(destination mode, name string, modes []mode) error {
	te := TransitionError{
		name:        name,
		current:     vr.stack[vr.frame].mode,
		destination: destination,
		modes:       modes,
		action:      "read",
	}
	if vr.frame != 0 {
		te.parent = vr.stack[vr.frame-1].mode
	}
	return te
}

func (vr *valueReader) typeError(t Type) error {
	return fmt.Errorf("positioned on %s, but attempted to read %s", vr.stack[vr.frame].vType, t)
}

func (vr *valueReader) invalidDocumentLengthError() error {
	return fmt.Errorf("document is invalid, end byte is at %d, but null byte found at %d", vr.stack[vr.frame].end, vr.offset)
}

func (vr *valueReader) ensureElementValue(t Type, destination mode, callerName string) error {
	switch vr.stack[vr.frame].mode {
	case mElement, mValue:
		if vr.stack[vr.frame].vType != t {
			return vr.typeError(t)
		}
	default:
		return vr.invalidTransitionErr(destination, callerName, []mode{mElement, mValue})
	}

	return nil
}

func (vr *valueReader) Type() Type {
	return vr.stack[vr.frame].vType
}

func (vr *valueReader) nextElementLength() (int32, error) {
	var length int32
	var err error
	switch vr.stack[vr.frame].vType {
	case TypeArray, TypeEmbeddedDocument, TypeCodeWithScope:
		length, err = vr.peekLength()
	case TypeBinary:
		length, err = vr.peekLength()
		length += 4 + 1 // binary length + subtype byte
	case TypeBoolean:
		length = 1
	case TypeDBPointer:
		length, err = vr.peekLength()
		length += 4 + 12 // string length + ObjectID length
	case TypeDateTime, TypeDouble, TypeInt64, TypeTimestamp:
		length = 8
	case TypeDecimal128:
		length = 16
	case TypeInt32:
		length = 4
	case TypeJavaScript, TypeString, TypeSymbol:
		length, err = vr.peekLength()
		length += 4
	case TypeMaxKey, TypeMinKey, TypeNull, TypeUndefined:
		length = 0
	case TypeObjectID:
		length = 12
	case TypeRegex:
		offset := vr.offset
		for n := 0; n < 2; n++ { // Read two C strings.
			var err error
			offset, err = vr.indexByteAfter(offset, 0x00)
			if err != nil {
				return 0, err
			}
			offset++ // add 0x00
		}
		length = int32(offset - vr.offset)
	default:
		return 0, fmt.Errorf("attempted to read bytes of unknown BSON type %v", vr.stack[vr.frame].vType)
	}

	return length, err
}

func (vr *valueReader) ReadValueBytes(dst []byte) (Type, []byte, error) {
	switch vr.stack[vr.frame].mode {
	case mTopLevel:
		length, err := vr.peekLength()
		if err != nil {
			return Type(0), nil, err
		}
		dst, err = vr.appendBytes(dst, length)
		if err != nil {
			return Type(0), nil, err
		}
		return Type(0), dst, nil
	case mElement, mValue:
		length, err := vr.nextElementLength()
		if err != nil {
			return Type(0), dst, err
		}

		dst, err = vr.appendBytes(dst, length)
		t := vr.stack[vr.frame].vType
		vr.pop()
		return t, dst, err
	default:
		return Type(0), nil, vr.invalidTransitionErr(0, "ReadValueBytes", []mode{mElement, mValue})
	}
}

func (vr *valueReader) Skip() error {
	switch vr.stack[vr.frame].mode {
	case mElement, mValue:
	default:
		return vr.invalidTransitionErr(0, "Skip", []mode{mElement, mValue})
	}

	length, err := vr.nextElementLength()
	if err != nil {
		return err
	}

	err = vr.skipBytes(length)
	vr.pop()
	return err
}

func (vr *valueReader) ReadArray() (ArrayReader, error) {
	if err := vr.ensureElementValue(TypeArray, mArray, "ReadArray"); err != nil {
		return nil, err
	}

	err := vr.pushArray()
	if err != nil {
		return nil, err
	}

	return vr, nil
}

func (vr *valueReader) ReadBinary() (b []byte, btype byte, err error) {
	if err := vr.ensureElementValue(TypeBinary, 0, "ReadBinary"); err != nil {
		return nil, 0, err
	}

	length, err := vr.readLength()
	if err != nil {
		return nil, 0, err
	}

	btype, err = vr.readByte()
	if err != nil {
		return nil, 0, err
	}

	// Check length in case it is an old binary without a length.
	if btype == 0x02 && length > 4 {
		length, err = vr.readLength()
		if err != nil {
			return nil, 0, err
		}
	}

	b, err = vr.readBytes(length)
	if err != nil {
		return nil, 0, err
	}
	// Make a copy of the returned byte slice because it's just a subslice from the valueReader's
	// buffer and is not safe to return in the unmarshaled value.
	cp := make([]byte, len(b))
	copy(cp, b)

	vr.pop()
	return cp, btype, nil
}

func (vr *valueReader) ReadBoolean() (bool, error) {
	if err := vr.ensureElementValue(TypeBoolean, 0, "ReadBoolean"); err != nil {
		return false, err
	}

	b, err := vr.readByte()
	if err != nil {
		return false, err
	}

	if b > 1 {
		return false, fmt.Errorf("invalid byte for boolean, %b", b)
	}

	vr.pop()
	return b == 1, nil
}

func (vr *valueReader) ReadDocument() (DocumentReader, error) {
	switch vr.stack[vr.frame].mode {
	case mTopLevel:
		length, err := vr.readLength()
		if err != nil {
			return nil, err
		}
		if length <= 4 {
			return nil, fmt.Errorf("invalid string length: %d", length)
		}
		length -= 4

		n, err := vr.prepload(length)
		if n < length || err != nil {
			if err == nil {
				err = io.EOF
			}
			return nil, err
		}
		vr.stack[vr.frame].end = int64(length) + vr.offset
		return vr, nil
	case mElement, mValue:
		if vr.stack[vr.frame].vType != TypeEmbeddedDocument {
			return nil, vr.typeError(TypeEmbeddedDocument)
		}
	default:
		return nil, vr.invalidTransitionErr(mDocument, "ReadDocument", []mode{mTopLevel, mElement, mValue})
	}

	err := vr.pushDocument()
	if err != nil {
		return nil, err
	}

	return vr, nil
}

func (vr *valueReader) ReadCodeWithScope() (code string, dr DocumentReader, err error) {
	if err := vr.ensureElementValue(TypeCodeWithScope, 0, "ReadCodeWithScope"); err != nil {
		return "", nil, err
	}

	totalLength, err := vr.readLength()
	if err != nil {
		return "", nil, err
	}
	strLength, err := vr.readLength()
	if err != nil {
		return "", nil, err
	}
	if strLength <= 0 {
		return "", nil, fmt.Errorf("invalid string length: %d", strLength)
	}
	strBytes, err := vr.readBytes(strLength)
	if err != nil {
		return "", nil, err
	}
	code = string(strBytes[:len(strBytes)-1])

	size, err := vr.pushCodeWithScope()
	if err != nil {
		return "", nil, err
	}

	// The total length should equal:
	// 4 (total length) + strLength + 4 (the length of str itself) + (document length)
	componentsLength := int64(4+strLength+4) + size
	if int64(totalLength) != componentsLength {
		return "", nil, fmt.Errorf(
			"length of CodeWithScope does not match lengths of components; total: %d; components: %d",
			totalLength, componentsLength,
		)
	}
	return code, vr, nil
}

func (vr *valueReader) ReadDBPointer() (ns string, oid ObjectID, err error) {
	if err := vr.ensureElementValue(TypeDBPointer, 0, "ReadDBPointer"); err != nil {
		return "", oid, err
	}

	ns, err = vr.readString()
	if err != nil {
		return "", oid, err
	}

	oidbytes, err := vr.readBytes(12)
	if err != nil {
		return "", oid, err
	}

	copy(oid[:], oidbytes)

	vr.pop()
	return ns, oid, nil
}

func (vr *valueReader) ReadDateTime() (int64, error) {
	if err := vr.ensureElementValue(TypeDateTime, 0, "ReadDateTime"); err != nil {
		return 0, err
	}

	i, err := vr.readi64()
	if err != nil {
		return 0, err
	}

	vr.pop()
	return i, nil
}

func (vr *valueReader) ReadDecimal128() (Decimal128, error) {
	if err := vr.ensureElementValue(TypeDecimal128, 0, "ReadDecimal128"); err != nil {
		return Decimal128{}, err
	}

	b, err := vr.readBytes(16)
	if err != nil {
		return Decimal128{}, err
	}

	l := binary.LittleEndian.Uint64(b[0:8])
	h := binary.LittleEndian.Uint64(b[8:16])

	vr.pop()
	return NewDecimal128(h, l), nil
}

func (vr *valueReader) ReadDouble() (float64, error) {
	if err := vr.ensureElementValue(TypeDouble, 0, "ReadDouble"); err != nil {
		return 0, err
	}

	u, err := vr.readu64()
	if err != nil {
		return 0, err
	}

	vr.pop()
	return math.Float64frombits(u), nil
}

func (vr *valueReader) ReadInt32() (int32, error) {
	if err := vr.ensureElementValue(TypeInt32, 0, "ReadInt32"); err != nil {
		return 0, err
	}

	vr.pop()
	return vr.readi32()
}

func (vr *valueReader) ReadInt64() (int64, error) {
	if err := vr.ensureElementValue(TypeInt64, 0, "ReadInt64"); err != nil {
		return 0, err
	}

	vr.pop()
	return vr.readi64()
}

func (vr *valueReader) ReadJavascript() (code string, err error) {
	if err := vr.ensureElementValue(TypeJavaScript, 0, "ReadJavascript"); err != nil {
		return "", err
	}

	vr.pop()
	return vr.readString()
}

func (vr *valueReader) ReadMaxKey() error {
	if err := vr.ensureElementValue(TypeMaxKey, 0, "ReadMaxKey"); err != nil {
		return err
	}

	vr.pop()
	return nil
}

func (vr *valueReader) ReadMinKey() error {
	if err := vr.ensureElementValue(TypeMinKey, 0, "ReadMinKey"); err != nil {
		return err
	}

	vr.pop()
	return nil
}

func (vr *valueReader) ReadNull() error {
	if err := vr.ensureElementValue(TypeNull, 0, "ReadNull"); err != nil {
		return err
	}

	vr.pop()
	return nil
}

func (vr *valueReader) ReadObjectID() (ObjectID, error) {
	if err := vr.ensureElementValue(TypeObjectID, 0, "ReadObjectID"); err != nil {
		return ObjectID{}, err
	}

	oidbytes, err := vr.readBytes(12)
	if err != nil {
		return ObjectID{}, err
	}

	var oid ObjectID
	copy(oid[:], oidbytes)

	vr.pop()
	return oid, nil
}

func (vr *valueReader) ReadRegex() (string, string, error) {
	if err := vr.ensureElementValue(TypeRegex, 0, "ReadRegex"); err != nil {
		return "", "", err
	}

	pattern, err := vr.readCString()
	if err != nil {
		return "", "", err
	}

	options, err := vr.readCString()
	if err != nil {
		return "", "", err
	}

	vr.pop()
	return pattern, options, nil
}

func (vr *valueReader) ReadString() (string, error) {
	if err := vr.ensureElementValue(TypeString, 0, "ReadString"); err != nil {
		return "", err
	}

	vr.pop()
	return vr.readString()
}

func (vr *valueReader) ReadSymbol() (symbol string, err error) {
	if err := vr.ensureElementValue(TypeSymbol, 0, "ReadSymbol"); err != nil {
		return "", err
	}

	vr.pop()
	return vr.readString()
}

func (vr *valueReader) ReadTimestamp() (t uint32, i uint32, err error) {
	if err := vr.ensureElementValue(TypeTimestamp, 0, "ReadTimestamp"); err != nil {
		return 0, 0, err
	}

	i, err = vr.readu32()
	if err != nil {
		return 0, 0, err
	}

	t, err = vr.readu32()
	if err != nil {
		return 0, 0, err
	}

	vr.pop()
	return t, i, nil
}

func (vr *valueReader) ReadUndefined() error {
	if err := vr.ensureElementValue(TypeUndefined, 0, "ReadUndefined"); err != nil {
		return err
	}

	vr.pop()
	return nil
}

func (vr *valueReader) ReadElement() (string, ValueReader, error) {
	switch vr.stack[vr.frame].mode {
	case mTopLevel, mDocument, mCodeWithScope:
	default:
		return "", nil, vr.invalidTransitionErr(mElement, "ReadElement", []mode{mTopLevel, mDocument, mCodeWithScope})
	}

	t, err := vr.readByte()
	if err != nil {
		return "", nil, err
	}

	if t == 0 {
		if vr.offset != vr.stack[vr.frame].end {
			return "", nil, vr.invalidDocumentLengthError()
		}

		vr.pop()
		return "", nil, ErrEOD
	}

	name, err := vr.readCString()
	if err != nil {
		return "", nil, err
	}

	vr.pushElement(Type(t))
	return name, vr, nil
}

func (vr *valueReader) ReadValue() (ValueReader, error) {
	switch vr.stack[vr.frame].mode {
	case mArray:
	default:
		return nil, vr.invalidTransitionErr(mValue, "ReadValue", []mode{mArray})
	}

	t, err := vr.readByte()
	if err != nil {
		return nil, err
	}

	if t == 0 {
		if vr.offset != vr.stack[vr.frame].end {
			return nil, vr.invalidDocumentLengthError()
		}

		vr.pop()
		return nil, ErrEOA
	}

	if err := vr.skipCString(); err != nil {
		return nil, err
	}

	vr.pushValue(Type(t))
	return vr, nil
}

// readBytes reads length bytes from the valueReader starting at the current offset. Note that the
// returned byte slice is a subslice from the valueReader buffer and must be converted or copied
// before returning in an unmarshaled value.
func (vr *valueReader) readBytes(length int32) ([]byte, error) {
	if length < 0 {
		return nil, fmt.Errorf("invalid length: %d", length)
	}

	n, err := vr.prepload(length)
	if n < length || err != nil {
		if err == nil {
			err = io.EOF
		}
		return nil, err
	}

	start := vr.offset
	vr.offset += int64(length)

	return vr.d[start : start+int64(length)], nil
}

func (vr *valueReader) appendBytes(dst []byte, length int32) ([]byte, error) {
	n, err := vr.prepload(length)
	if n < length || err != nil {
		if err == nil {
			err = io.EOF
		}
		return nil, err
	}
	start := vr.offset
	vr.offset += int64(length)
	return append(dst, vr.d[start:start+int64(length)]...), nil
}

func (vr *valueReader) skipBytes(length int32) error {
	if length < 0 {
		return fmt.Errorf("invalid length: %d", length)
	}

	n, err := vr.prepload(length)
	if n < length || err != nil {
		if err == nil {
			err = io.EOF
		}
		return err
	}
	vr.offset += int64(length)
	return nil
}

func (vr *valueReader) readByte() (byte, error) {
	n, err := vr.prepload(1)
	if n < 1 || err != nil {
		if err == nil {
			err = io.EOF
		}
		return 0x0, err
	}
	vr.offset++
	return vr.d[vr.offset-1], nil
}

func (vr *valueReader) skipCString() error {
	offset, err := vr.indexByteAfter(vr.offset, 0x00)
	if err != nil {
		return err
	}
	vr.offset = offset + 1
	return nil
}

func (vr *valueReader) readCString() (string, error) {
	offset, err := vr.indexByteAfter(vr.offset, 0x00)
	if err != nil {
		return "", err
	}
	start := vr.offset
	vr.offset = offset + 1
	return string(vr.d[start : vr.offset-1]), nil
}

func (vr *valueReader) readString() (string, error) {
	length, err := vr.readLength()
	if err != nil {
		return "", err
	}
	if length <= 0 {
		return "", fmt.Errorf("invalid string length: %d", length)
	}

	n, err := vr.prepload(length)
	if n < length || err != nil {
		if err == nil {
			err = io.EOF
		}
		return "", err
	}

	if vr.d[vr.offset+int64(length)-1] != 0x00 {
		return "", fmt.Errorf("string does not end with null byte, but with %v", vr.d[vr.offset+int64(length)-1])
	}

	start := vr.offset
	vr.offset += int64(length)
	return string(vr.d[start : start+int64(length)-1]), nil
}

func (vr *valueReader) peekLength() (int32, error) {
	n, err := vr.prepload(4)
	if n < 4 || err != nil {
		if err == nil {
			err = io.EOF
		}
		return 0, err
	}

	idx := vr.offset
	return (int32(vr.d[idx]) | int32(vr.d[idx+1])<<8 | int32(vr.d[idx+2])<<16 | int32(vr.d[idx+3])<<24), nil
}

func (vr *valueReader) readLength() (int32, error) { return vr.readi32() }

func (vr *valueReader) readi32() (int32, error) {
	n, err := vr.prepload(4)
	if n < 4 || err != nil {
		if err == nil {
			err = io.EOF
		}
		return 0, err
	}

	idx := vr.offset
	vr.offset += 4
	return (int32(vr.d[idx]) | int32(vr.d[idx+1])<<8 | int32(vr.d[idx+2])<<16 | int32(vr.d[idx+3])<<24), nil
}

func (vr *valueReader) readu32() (uint32, error) {
	n, err := vr.prepload(4)
	if n < 4 || err != nil {
		if err == nil {
			err = io.EOF
		}
		return 0, err
	}

	idx := vr.offset
	vr.offset += 4
	return (uint32(vr.d[idx]) | uint32(vr.d[idx+1])<<8 | uint32(vr.d[idx+2])<<16 | uint32(vr.d[idx+3])<<24), nil
}

func (vr *valueReader) readi64() (int64, error) {
	n, err := vr.prepload(8)
	if n < 8 || err != nil {
		if err == nil {
			err = io.EOF
		}
		return 0, err
	}

	idx := vr.offset
	vr.offset += 8
	return int64(vr.d[idx]) | int64(vr.d[idx+1])<<8 | int64(vr.d[idx+2])<<16 | int64(vr.d[idx+3])<<24 |
		int64(vr.d[idx+4])<<32 | int64(vr.d[idx+5])<<40 | int64(vr.d[idx+6])<<48 | int64(vr.d[idx+7])<<56, nil
}

func (vr *valueReader) readu64() (uint64, error) {
	n, err := vr.prepload(8)
	if n < 8 || err != nil {
		if err == nil {
			err = io.EOF
		}
		return 0, err
	}

	idx := vr.offset
	vr.offset += 8
	return uint64(vr.d[idx]) | uint64(vr.d[idx+1])<<8 | uint64(vr.d[idx+2])<<16 | uint64(vr.d[idx+3])<<24 |
		uint64(vr.d[idx+4])<<32 | uint64(vr.d[idx+5])<<40 | uint64(vr.d[idx+6])<<48 | uint64(vr.d[idx+7])<<56, nil
}
