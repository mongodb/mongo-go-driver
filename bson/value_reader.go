// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"sync"
)

var _ ValueReader = &valueReader{}

// ErrEOA is the error returned when the end of a BSON array has been reached.
var ErrEOA = errors.New("end of array")

// ErrEOD is the error returned when the end of a BSON document has been reached.
var ErrEOD = errors.New("end of document")

type vrState struct {
	mode  mode
	vType Type
	end   int64
}

var bufioReaderPool = sync.Pool{
	New: func() interface{} {
		return bufio.NewReader(nil)
	},
}

var vrPool = sync.Pool{
	New: func() interface{} {
		return &valueReader{
			stack: make([]vrState, 1, 5),
		}
	},
}

// valueReader is for reading BSON values.
type valueReader struct {
	r      *bufio.Reader
	offset int64

	stack []vrState
	frame int64
}

func getDocumentReader(r io.Reader) *valueReader {
	vr := vrPool.Get().(*valueReader)

	vr.offset = 0
	vr.frame = 0

	vr.stack = vr.stack[:1]
	vr.stack[0] = vrState{mode: mTopLevel}

	br := bufioReaderPool.Get().(*bufio.Reader)
	br.Reset(r)
	vr.r = br

	return vr
}

func putDocumentReader(vr *valueReader) {
	if vr == nil {
		return
	}

	bufioReaderPool.Put(vr.r)
	vr.r = nil

	vrPool.Put(vr)
}

// NewDocumentReader returns a ValueReader using b for the underlying BSON
// representation.
func NewDocumentReader(r io.Reader) ValueReader {
	return newDocumentReader(r)
}

// newValueReader returns a ValueReader that starts in the Value mode instead of in top
// level document mode. This enables the creation of a ValueReader for a single BSON value.
func newValueReader(t Type, r io.Reader) ValueReader {
	vr := newDocumentReader(r)
	if len(vr.stack) == 0 {
		vr.stack = make([]vrState, 1, 5)
	}
	vr.stack[0].mode = mValue
	vr.stack[0].vType = t
	return vr
}

func newDocumentReader(r io.Reader) *valueReader {
	stack := make([]vrState, 1, 5)
	stack[0] = vrState{
		mode: mTopLevel,
	}
	return &valueReader{
		r:     bufio.NewReader(r),
		stack: stack,
	}
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

	length, err := vr.readLength()
	if err != nil {
		return err
	}
	vr.stack[vr.frame].end = int64(length) + vr.offset - 4

	return nil
}

func (vr *valueReader) pushArray() error {
	vr.advanceFrame()

	vr.stack[vr.frame].mode = mArray

	length, err := vr.readLength()
	if err != nil {
		return err
	}
	vr.stack[vr.frame].end = int64(length) + vr.offset - 4

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

	length, err := vr.readLength()
	if err != nil {
		return 0, err
	}
	vr.stack[vr.frame].end = int64(length) + vr.offset - 4

	return int64(length), nil
}

func (vr *valueReader) pop() error {
	var cnt int
	switch vr.stack[vr.frame].mode {
	case mElement, mValue:
		cnt = 1
	case mDocument, mArray, mCodeWithScope:
		cnt = 2 // we pop twice to jump over the vrElement: vrDocument -> vrElement -> vrDocument/TopLevel/etc...
	}
	for i := 0; i < cnt && vr.frame > 0; i++ {
		if vr.offset < vr.stack[vr.frame].end {
			_, err := vr.r.Discard(int(vr.stack[vr.frame].end - vr.offset))
			if err != nil {
				return err
			}
		}
		vr.frame--
	}
	if vr.frame == 0 {
		if vr.stack[0].end > vr.offset {
			vr.stack[0].end -= vr.offset
		} else {
			vr.stack[0].end = 0
		}
		vr.offset = 0
	}
	return nil
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

func (vr *valueReader) appendNextElement(dst []byte) ([]byte, error) {
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
		for n := 0; n < 2; n++ { // Read two C strings.
			str, err := vr.r.ReadBytes(0x00)
			if err != nil {
				return nil, err
			}
			dst = append(dst, str...)
			vr.offset += int64(len(str))
		}
		return dst, nil
	default:
		return nil, fmt.Errorf("attempted to read bytes of unknown BSON type %v", vr.stack[vr.frame].vType)
	}
	if err != nil {
		return nil, err
	}

	buf, err := vr.r.Peek(int(length))
	if err != nil {
		if err == bufio.ErrBufferFull {
			temp := make([]byte, length)
			if _, err = io.ReadFull(vr.r, temp); err != nil {
				return nil, err
			}
			dst = append(dst, temp...)
			vr.offset += int64(len(temp))
			return dst, nil
		}

		return nil, err
	}

	dst = append(dst, buf...)
	if _, err = vr.r.Discard(int(length)); err != nil {
		return nil, err
	}

	vr.offset += int64(length)
	return dst, nil
}

func (vr *valueReader) readValueBytes(dst []byte) (Type, []byte, error) {
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
		dst, err := vr.appendNextElement(dst)
		if err != nil {
			return Type(0), dst, err
		}

		t := vr.stack[vr.frame].vType
		err = vr.pop()
		if err != nil {
			return Type(0), nil, err
		}
		return t, dst, nil
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

	_, err := vr.appendNextElement(nil)
	if err != nil {
		return err
	}

	return vr.pop()
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

	b = make([]byte, length)
	err = vr.read(b)
	if err != nil {
		return nil, 0, err
	}

	if err := vr.pop(); err != nil {
		return nil, 0, err
	}
	return b, btype, nil
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

	if err := vr.pop(); err != nil {
		return false, err
	}
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

		vr.stack[vr.frame].end = int64(length) + vr.offset - 4
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
	strBytes := make([]byte, strLength)
	err = vr.read(strBytes)
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

	err = vr.read(oid[:])
	if err != nil {
		return "", ObjectID{}, err
	}

	if err := vr.pop(); err != nil {
		return "", ObjectID{}, err
	}
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

	if err := vr.pop(); err != nil {
		return 0, err
	}
	return i, nil
}

func (vr *valueReader) ReadDecimal128() (Decimal128, error) {
	if err := vr.ensureElementValue(TypeDecimal128, 0, "ReadDecimal128"); err != nil {
		return Decimal128{}, err
	}

	var b [16]byte
	err := vr.read(b[:])
	if err != nil {
		return Decimal128{}, err
	}

	l := binary.LittleEndian.Uint64(b[0:8])
	h := binary.LittleEndian.Uint64(b[8:16])

	if err := vr.pop(); err != nil {
		return Decimal128{}, err
	}
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

	if err := vr.pop(); err != nil {
		return 0, err
	}
	return math.Float64frombits(u), nil
}

func (vr *valueReader) ReadInt32() (int32, error) {
	if err := vr.ensureElementValue(TypeInt32, 0, "ReadInt32"); err != nil {
		return 0, err
	}

	if err := vr.pop(); err != nil {
		return 0, err
	}
	return vr.readi32()
}

func (vr *valueReader) ReadInt64() (int64, error) {
	if err := vr.ensureElementValue(TypeInt64, 0, "ReadInt64"); err != nil {
		return 0, err
	}

	if err := vr.pop(); err != nil {
		return 0, err
	}
	return vr.readi64()
}

func (vr *valueReader) ReadJavascript() (string, error) {
	if err := vr.ensureElementValue(TypeJavaScript, 0, "ReadJavascript"); err != nil {
		return "", err
	}

	if err := vr.pop(); err != nil {
		return "", err
	}
	return vr.readString()
}

func (vr *valueReader) ReadMaxKey() error {
	if err := vr.ensureElementValue(TypeMaxKey, 0, "ReadMaxKey"); err != nil {
		return err
	}

	return vr.pop()
}

func (vr *valueReader) ReadMinKey() error {
	if err := vr.ensureElementValue(TypeMinKey, 0, "ReadMinKey"); err != nil {
		return err
	}

	return vr.pop()
}

func (vr *valueReader) ReadNull() error {
	if err := vr.ensureElementValue(TypeNull, 0, "ReadNull"); err != nil {
		return err
	}

	return vr.pop()
}

func (vr *valueReader) ReadObjectID() (ObjectID, error) {
	if err := vr.ensureElementValue(TypeObjectID, 0, "ReadObjectID"); err != nil {
		return ObjectID{}, err
	}

	var oid ObjectID
	err := vr.read(oid[:])
	if err != nil {
		return ObjectID{}, err
	}

	if err := vr.pop(); err != nil {
		return ObjectID{}, err
	}
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

	if err := vr.pop(); err != nil {
		return "", "", err
	}
	return pattern, options, nil
}

func (vr *valueReader) ReadString() (string, error) {
	if err := vr.ensureElementValue(TypeString, 0, "ReadString"); err != nil {
		return "", err
	}

	if err := vr.pop(); err != nil {
		return "", err
	}
	return vr.readString()
}

func (vr *valueReader) ReadSymbol() (string, error) {
	if err := vr.ensureElementValue(TypeSymbol, 0, "ReadSymbol"); err != nil {
		return "", err
	}

	if err := vr.pop(); err != nil {
		return "", err
	}
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

	if err := vr.pop(); err != nil {
		return 0, 0, err
	}
	return t, i, nil
}

func (vr *valueReader) ReadUndefined() error {
	if err := vr.ensureElementValue(TypeUndefined, 0, "ReadUndefined"); err != nil {
		return err
	}

	return vr.pop()
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

		_ = vr.pop() // Ignore the error because the call here never reads from the underlying reader.
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

		_ = vr.pop() // Ignore the error because the call here never reads from the underlying reader.
		return nil, ErrEOA
	}

	if _, err := vr.readCString(); err != nil {
		return nil, err
	}

	vr.pushValue(Type(t))
	return vr, nil
}

func (vr *valueReader) read(p []byte) error {
	n, err := io.ReadFull(vr.r, p)
	if err != nil {
		return err
	}
	vr.offset += int64(n)
	return nil
}

func (vr *valueReader) appendBytes(dst []byte, length int32) ([]byte, error) {
	buf := make([]byte, length)
	err := vr.read(buf)
	if err != nil {
		return nil, err
	}
	return append(dst, buf...), nil
}

func (vr *valueReader) readByte() (byte, error) {
	b, err := vr.r.ReadByte()
	if err != nil {
		return 0x0, err
	}
	vr.offset++
	return b, nil
}

func (vr *valueReader) readCString() (string, error) {
	str, err := vr.r.ReadString(0x00)
	if err != nil {
		return "", err
	}
	l := len(str)
	vr.offset += int64(l)
	return str[:l-1], nil
}

func (vr *valueReader) readString() (string, error) {
	length, err := vr.readLength()
	if err != nil {
		return "", err
	}
	if length <= 0 {
		return "", fmt.Errorf("invalid string length: %d", length)
	}

	buf := make([]byte, length)
	err = vr.read(buf)
	if err != nil {
		return "", err
	}

	if buf[length-1] != 0x00 {
		return "", fmt.Errorf("string does not end with null byte, but with %v", buf[length-1])
	}

	return string(buf[:length-1]), nil
}

func (vr *valueReader) peekLength() (int32, error) {
	buf, err := vr.r.Peek(4)
	if err != nil {
		return 0, err
	}

	return int32(binary.LittleEndian.Uint32(buf)), nil
}

func (vr *valueReader) readLength() (int32, error) {
	l, err := vr.readi32()
	if err != nil {
		return 0, err
	}
	if l < 0 {
		return 0, fmt.Errorf("invalid negative length: %d", l)
	}
	return l, nil
}

func (vr *valueReader) readi32() (int32, error) {
	var buf [4]byte
	err := vr.read(buf[:])
	if err != nil {
		return 0, err
	}

	return int32(binary.LittleEndian.Uint32(buf[:])), nil
}

func (vr *valueReader) readu32() (uint32, error) {
	var buf [4]byte
	err := vr.read(buf[:])
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint32(buf[:]), nil
}

func (vr *valueReader) readi64() (int64, error) {
	var buf [8]byte
	err := vr.read(buf[:])
	if err != nil {
		return 0, err
	}

	return int64(binary.LittleEndian.Uint64(buf[:])), nil
}

func (vr *valueReader) readu64() (uint64, error) {
	var buf [8]byte
	err := vr.read(buf[:])
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint64(buf[:]), nil
}
