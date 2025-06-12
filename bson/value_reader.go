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
	"sync"
)

var _ ValueReader = (*valueReader)(nil)

// ErrEOA is returned when the end of a BSON array is reached.
var ErrEOA = errors.New("end of array")

// ErrEOD is returned when the end of a BSON document is reached.
var ErrEOD = errors.New("end of document")

type vrState struct {
	mode  mode
	vType Type
	end   int64
}

var vrPool = sync.Pool{
	New: func() interface{} { return new(valueReader) },
}

// valueReader reads BSON values from an in-memory []byte.
type valueReader struct {
	d      []byte
	offset int64

	stack []vrState
	frame int64
}

// NewDocumentReader reads all of r into memory and returns a ValueReader.
func NewDocumentReader(r io.Reader) ValueReader {
	buf, err := io.ReadAll(r)
	if err != nil {
		panic("bson: read error: " + err.Error())
	}
	vr := vrPool.Get().(*valueReader)
	vr.d = buf
	vr.offset = 0
	vr.frame = 0
	if vr.stack == nil {
		vr.stack = make([]vrState, 1, 5)
	}
	vr.stack = vr.stack[:1]
	vr.stack[0] = vrState{mode: mTopLevel}
	return vr
}

// putDocumentReader returns this reader to the pool.
func putDocumentReader(vr *valueReader) {
	if vr == nil {
		return
	}
	vr.d = nil
	vrPool.Put(vr)
}

// getDocumentReader returns a TopLevel valueReader over the entire BSON stream in r.
func getDocumentReader(r io.Reader) *valueReader {
	return newDocumentReader(r)
}

// newDocumentReader reads the entire BSON blob into memory and
// returns a *valueReader in TopLevel mode.
func newDocumentReader(r io.Reader) *valueReader {
	buf, err := io.ReadAll(r)
	if err != nil {
		panic("bson: could not read document: " + err.Error())
	}
	vr := vrPool.Get().(*valueReader)
	// ensure we have at least one slot in the stack
	if cap(vr.stack) < 1 {
		vr.stack = make([]vrState, 1, 5)
	} else {
		vr.stack = vr.stack[:1]
	}
	vr.stack[0] = vrState{mode: mTopLevel, end: int64(len(buf))}
	vr.d = buf
	vr.offset = 0
	vr.frame = 0
	return vr
}

// newValueReader reads the single‐value BSON from r into a valueReader
// that starts in Value mode for type t.
func newValueReader(t Type, r io.Reader) ValueReader {
	// slurp the entire document/value into memory
	buf, err := io.ReadAll(r)
	if err != nil {
		panic("bson: could not read value: " + err.Error())
	}

	vr := vrPool.Get().(*valueReader)

	// make sure the stack slice has at least one element
	if cap(vr.stack) < 1 {
		vr.stack = make([]vrState, 1, 5)
	} else {
		vr.stack = vr.stack[:1]
	}

	// set up the initial frame in "value" mode, with end = len(buf)
	vr.stack[0] = vrState{
		mode:  mValue,
		vType: t,
		end:   int64(len(buf)),
	}

	vr.d = buf    // point at the in-memory bytes
	vr.offset = 0 // start at the beginning
	vr.frame = 0  // only one frame

	return vr
}

// ---------------- frame management ----------------

func (vr *valueReader) advanceFrame() {
	if vr.frame+1 >= int64(len(vr.stack)) {
		n := len(vr.stack)
		if n+1 >= cap(vr.stack) {
			buf := make([]vrState, 2*cap(vr.stack)+1)
			copy(buf, vr.stack)
			vr.stack = buf
		}
		vr.stack = vr.stack[:n+1]
	}
	vr.frame++
	vr.stack[vr.frame] = vrState{}
}

func (vr *valueReader) pop() {
	switch vr.stack[vr.frame].mode {
	case mElement, mValue:
		vr.frame--
	case mDocument, mArray, mCodeWithScope:
		vr.frame -= 2
	}
}

// ---------------- error helpers ----------------

func (vr *valueReader) invalidTransitionErr(dest mode, name string, modes []mode) error {
	te := TransitionError{
		name:        name,
		current:     vr.stack[vr.frame].mode,
		destination: dest,
		modes:       modes,
		action:      "read",
	}
	if vr.frame > 0 {
		te.parent = vr.stack[vr.frame-1].mode
	}
	return te
}

func (vr *valueReader) typeError(t Type) error {
	return fmt.Errorf("positioned on %s but attempted to read %s", vr.stack[vr.frame].vType, t)
}

func (vr *valueReader) invalidDocumentLengthError() error {
	s := vr.stack[vr.frame]
	return fmt.Errorf("document invalid: end=%d but offset=%d", s.end, vr.offset)
}

// ---------------- basic inspectors ----------------

func (vr *valueReader) Type() Type {
	return vr.stack[vr.frame].vType
}

func (vr *valueReader) ensureElementValue(t Type, dest mode, name string) error {
	m := vr.stack[vr.frame].mode
	if (m == mElement || m == mValue) && vr.stack[vr.frame].vType == t {
		return nil
	}
	return vr.invalidTransitionErr(dest, name, []mode{mElement, mValue})
}

func (vr *valueReader) peekLength() (int32, error) {
	if vr.offset+4 > int64(len(vr.d)) {
		return 0, io.EOF
	}
	return int32(binary.LittleEndian.Uint32(vr.d[vr.offset:])), nil
}

// ---------------- zero‐alloc raw readers ----------------

// readBytes returns a subslice [offset, offset+length) or EOF.
func (vr *valueReader) readBytes(length int32) ([]byte, error) {
	if length < 0 || vr.offset+int64(length) > int64(len(vr.d)) {
		return nil, io.EOF
	}
	start := vr.offset
	vr.offset += int64(length)
	return vr.d[start : start+int64(length)], nil
}

// appendBytes appends the next length bytes to dst.
func (vr *valueReader) appendBytes(dst []byte, length int32) ([]byte, error) {
	if length < 0 || vr.offset+int64(length) > int64(len(vr.d)) {
		return nil, io.EOF
	}
	start := vr.offset
	vr.offset += int64(length)
	return append(dst, vr.d[start:start+int64(length)]...), nil
}

// skipBytes just advances offset.
func (vr *valueReader) skipBytes(length int32) error {
	if length < 0 || vr.offset+int64(length) > int64(len(vr.d)) {
		return io.EOF
	}
	vr.offset += int64(length)
	return nil
}

// readByte returns one byte or EOF.
func (vr *valueReader) readByte() (byte, error) {
	if vr.offset >= int64(len(vr.d)) {
		return 0, io.EOF
	}
	b := vr.d[vr.offset]
	vr.offset++
	return b, nil
}

// skipCString finds the next NUL and skips past it.
func (vr *valueReader) skipCString() error {
	idx := bytes.IndexByte(vr.d[vr.offset:], 0x00)
	if idx < 0 {
		return io.EOF
	}
	vr.offset += int64(idx) + 1
	return nil
}

// readCString extracts bytes up to the next NUL.
func (vr *valueReader) readCString() (string, error) {
	idx := bytes.IndexByte(vr.d[vr.offset:], 0x00)
	if idx < 0 {
		return "", io.EOF
	}
	start := vr.offset
	vr.offset += int64(idx) + 1
	return string(vr.d[start : start+int64(idx)]), nil
}

// readString reads a length-prefixed string (incl. trailing NUL).
func (vr *valueReader) readString() (string, error) {
	length, err := vr.readLength()
	if err != nil {
		return "", err
	}
	if length <= 0 || vr.offset+int64(length) > int64(len(vr.d)) {
		return "", fmt.Errorf("invalid string length %d", length)
	}
	// must end with null
	if vr.d[vr.offset+int64(length)-1] != 0 {
		return "", fmt.Errorf("string missing trailing null")
	}
	start := vr.offset
	vr.offset += int64(length)
	return string(vr.d[start : start+int64(length)-1]), nil
}

func (vr *valueReader) readLength() (int32, error) {
	return vr.readi32()
}

func (vr *valueReader) readi32() (int32, error) {
	if vr.offset+4 > int64(len(vr.d)) {
		return 0, io.EOF
	}
	start := vr.offset
	vr.offset += 4
	return int32(binary.LittleEndian.Uint32(vr.d[start:])), nil
}

func (vr *valueReader) readu32() (uint32, error) {
	if vr.offset+4 > int64(len(vr.d)) {
		return 0, io.EOF
	}
	start := vr.offset
	vr.offset += 4
	return binary.LittleEndian.Uint32(vr.d[start:]), nil
}

func (vr *valueReader) readi64() (int64, error) {
	if vr.offset+8 > int64(len(vr.d)) {
		return 0, io.EOF
	}
	start := vr.offset
	vr.offset += 8
	return int64(binary.LittleEndian.Uint64(vr.d[start:])), nil
}

func (vr *valueReader) readu64() (uint64, error) {
	if vr.offset+8 > int64(len(vr.d)) {
		return 0, io.EOF
	}
	start := vr.offset
	vr.offset += 8
	return binary.LittleEndian.Uint64(vr.d[start:]), nil
}

// ---------------- high-level readers ----------------

func (vr *valueReader) ReadElement() (string, ValueReader, error) {
	switch vr.stack[vr.frame].mode {
	case mTopLevel, mDocument, mCodeWithScope:
	default:
		return "", nil, vr.invalidTransitionErr(mElement, "ReadElement",
			[]mode{mTopLevel, mDocument, mCodeWithScope})
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
	if vr.stack[vr.frame].mode != mArray {
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

func (vr *valueReader) Skip() error {
	_, _, err := vr.readValueBytes(nil)
	return err
}

func (vr *valueReader) ReadArray() (ArrayReader, error) {
	if err := vr.ensureElementValue(TypeArray, mArray, "ReadArray"); err != nil {
		return nil, err
	}
	vr.advanceFrame()
	size, err := vr.readLength()
	if err != nil {
		return nil, err
	}
	vr.stack[vr.frame].mode = mArray
	vr.stack[vr.frame].end = int64(size) + vr.offset - 4
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
	// copy so user doesn’t share underlying buffer
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
		return false, fmt.Errorf("invalid boolean byte %b", b)
	}
	vr.pop()
	return b == 1, nil
}

func (vr *valueReader) ReadDocument() (DocumentReader, error) {
	switch vr.stack[vr.frame].mode {
	case mTopLevel:
		size, err := vr.readLength()
		if err != nil {
			return nil, err
		}
		vr.stack[vr.frame].mode = mDocument
		vr.stack[vr.frame].end = int64(size) + vr.offset - 4
		return vr, nil
	case mElement, mValue:
		if vr.stack[vr.frame].vType != TypeEmbeddedDocument {
			return nil, vr.typeError(TypeEmbeddedDocument)
		}
	default:
		return nil, vr.invalidTransitionErr(mDocument, "ReadDocument", []mode{mTopLevel, mElement, mValue})
	}
	vr.advanceFrame()
	size, err := vr.readLength()
	if err != nil {
		return nil, err
	}
	vr.stack[vr.frame].mode = mDocument
	vr.stack[vr.frame].end = int64(size) + vr.offset - 4
	return vr, nil
}

func (vr *valueReader) ReadCodeWithScope() (string, DocumentReader, error) {
	if err := vr.ensureElementValue(TypeCodeWithScope, 0, "ReadCodeWithScope"); err != nil {
		return "", nil, err
	}
	totalLen, err := vr.readLength()
	if err != nil {
		return "", nil, err
	}
	strLen, err := vr.readLength()
	if err != nil {
		return "", nil, err
	}
	if strLen <= 0 {
		return "", nil, fmt.Errorf("invalid code length %d", strLen)
	}
	buf, err := vr.readBytes(strLen)
	if err != nil {
		return "", nil, err
	}
	code := string(buf[:len(buf)-1])
	vr.advanceFrame()
	size := int64(totalLen) - (4 + int64(strLen))
	vr.stack[vr.frame].mode = mCodeWithScope
	vr.stack[vr.frame].end = vr.offset + size - 4
	return code, vr, nil
}

func (vr *valueReader) ReadDBPointer() (string, ObjectID, error) {
	if err := vr.ensureElementValue(TypeDBPointer, 0, "ReadDBPointer"); err != nil {
		return "", ObjectID{}, err
	}
	ns, err := vr.readString()
	if err != nil {
		return "", ObjectID{}, err
	}
	oidBytes, err := vr.readBytes(12)
	if err != nil {
		return "", ObjectID{}, err
	}
	var oid ObjectID
	copy(oid[:], oidBytes)
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
	low := binary.LittleEndian.Uint64(b[0:8])
	high := binary.LittleEndian.Uint64(b[8:16])
	vr.pop()
	return NewDecimal128(high, low), nil
}

func (vr *valueReader) ReadDouble() (float64, error) {
	if err := vr.ensureElementValue(TypeDouble, 0, "ReadDouble"); err != nil {
		return 0, err
	}
	bits, err := vr.readu64()
	if err != nil {
		return 0, err
	}
	vr.pop()
	return math.Float64frombits(bits), nil
}

func (vr *valueReader) ReadInt32() (int32, error) {
	if err := vr.ensureElementValue(TypeInt32, 0, "ReadInt32"); err != nil {
		return 0, err
	}
	i, err := vr.readi32()
	if err != nil {
		return 0, err
	}
	vr.pop()
	return i, nil
}

func (vr *valueReader) ReadInt64() (int64, error) {
	if err := vr.ensureElementValue(TypeInt64, 0, "ReadInt64"); err != nil {
		return 0, err
	}
	i, err := vr.readi64()
	if err != nil {
		return 0, err
	}
	vr.pop()
	return i, nil
}

func (vr *valueReader) ReadJavascript() (string, error) {
	if err := vr.ensureElementValue(TypeJavaScript, 0, "ReadJavascript"); err != nil {
		return "", err
	}
	s, err := vr.readString()
	if err != nil {
		return "", err
	}
	vr.pop()
	return s, nil
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
	oidBytes, err := vr.readBytes(12)
	if err != nil {
		return ObjectID{}, err
	}
	var oid ObjectID
	copy(oid[:], oidBytes)
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
	s, err := vr.readString()
	if err != nil {
		return "", err
	}
	vr.pop()
	return s, nil
}

func (vr *valueReader) ReadSymbol() (string, error) {
	if err := vr.ensureElementValue(TypeSymbol, 0, "ReadSymbol"); err != nil {
		return "", err
	}
	s, err := vr.readString()
	if err != nil {
		return "", err
	}
	vr.pop()
	return s, nil
}

func (vr *valueReader) ReadTimestamp() (uint32, uint32, error) {
	if err := vr.ensureElementValue(TypeTimestamp, 0, "ReadTimestamp"); err != nil {
		return 0, 0, err
	}
	i, err := vr.readu32()
	if err != nil {
		return 0, 0, err
	}
	t, err := vr.readu32()
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

func (vr *valueReader) pushElement(t Type) {
	vr.advanceFrame()
	vr.stack[vr.frame].mode = mElement
	vr.stack[vr.frame].vType = t
}

// pushValue starts a new “value” frame of type t (used in arrays).
func (vr *valueReader) pushValue(t Type) {
	vr.advanceFrame()
	vr.stack[vr.frame].mode = mValue
	vr.stack[vr.frame].vType = t
}

// readValueBytes returns the raw bytes of the next value (or top‐level
// document) without allocating intermediary buffers, then pops the frame.
func (vr *valueReader) readValueBytes(dst []byte) (Type, []byte, error) {
	switch vr.stack[vr.frame].mode {
	case mTopLevel:
		// top‐level: read entire document
		length, err := vr.peekLength()
		if err != nil {
			return 0, nil, err
		}
		b, err := vr.readBytes(length)
		return Type(0), append(dst, b...), err

	case mElement, mValue:
		// element/value frame: read just the value portion
		t := vr.stack[vr.frame].vType

		// figure out how many bytes this type occupies
		// (use the same logic as appendNextElement but zero‐alloc)
		var length int32
		var err error
		switch t {
		case TypeString, TypeSymbol, TypeJavaScript:
			// length‐prefixed UTF-8
			length, err = vr.peekLength()
			length += 4
		case TypeBinary:
			// length + subtype byte
			length, err = vr.peekLength()
			length += 4 + 1
		case TypeInt32:
			length = 4
		case TypeInt64, TypeDateTime, TypeDouble, TypeTimestamp:
			length = 8
		case TypeDecimal128:
			length = 16
		case TypeObjectID:
			length = 12
		case TypeBoolean:
			length = 1
		case TypeMaxKey, TypeMinKey, TypeNull, TypeUndefined:
			length = 0
		case TypeArray, TypeEmbeddedDocument, TypeCodeWithScope:
			length, err = vr.peekLength()
		case TypeDBPointer:
			length, err = vr.peekLength()
			length += 4 + 12
		case TypeRegex:
			// find two C-strings
			// first: up to NUL
			idx := bytes.IndexByte(vr.d[vr.offset:], 0x00)
			if idx < 0 {
				return t, dst, io.EOF
			}
			// second: after that
			idx2 := bytes.IndexByte(vr.d[vr.offset+int64(idx)+1:], 0x00)
			if idx2 < 0 {
				return t, dst, io.EOF
			}
			length = int32(idx + 1 + idx2 + 1)
		default:
			return t, dst, fmt.Errorf("unknown BSON type %v", t)
		}
		if err != nil {
			return t, dst, err
		}

		b, err := vr.readBytes(length)
		vr.pop()
		return t, append(dst, b...), err

	default:
		return 0, nil, vr.invalidTransitionErr(0, "readValueBytes", []mode{mTopLevel, mElement, mValue})
	}
}
