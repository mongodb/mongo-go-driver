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

type byteSrc interface {
	io.ByteReader

	readExact(p []byte) (int, error)

	// Peek returns the next n bytes without advancing the cursor. It must return
	// exactly n bytes or n error if fewer are available.
	peek(n int) ([]byte, error)

	// discard advanced the cursor by n bytes, returning the actual number
	// discarded or an error if fewer were available.
	discard(n int) (int, error)

	// readSlice reads until (and including) the first occurrence of delim,
	// returning the entire slice [start...delimiter] and advancing the cursor.
	// past it. Returns an error if delim is not found.
	readSlice(delim byte) ([]byte, error)

	// pos returns the number of bytes consumed so far.
	pos() int64

	// regexLength returns the total byte length of a BSON regex value (two
	// C-strings including their terminating NULs) in buffered mode.
	regexLength() (int32, error)

	// streamable returns true if this source can be used in a streaming context.
	streamable() bool

	// reset resets the source to its initial state.
	reset()
}

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

var vrPool = sync.Pool{
	New: func() any {
		return &valueReader{
			stack: make([]vrState, 1, 5),
		}
	},
}

// valueReader is for reading BSON values.
type valueReader struct {
	src    byteSrc
	offset int64

	stack []vrState
	frame int64
}

func getBufferedDocumentReader(b []byte) *valueReader {
	return newBufferedDocumentReader(b)
}

func putBufferedDocumentReader(vr *valueReader) {
	if vr == nil {
		return
	}

	vr.src.reset()

	// Reset src and stack to avoid holding onto memory.
	vr.src = nil
	vr.frame = 0
	vr.stack = vr.stack[:0]

	vrPool.Put(vr)
}

// NewDocumentReader returns a ValueReader using b for the underlying BSON
// representation.
func NewDocumentReader(r io.Reader) ValueReader {
	stack := make([]vrState, 1, 5)
	stack[0] = vrState{
		mode: mTopLevel,
	}

	return &valueReader{
		src:   &streamingByteSrc{br: bufio.NewReader(r), offset: 0},
		stack: stack,
	}
}

// newBufferedValueReader returns a ValueReader that starts in the Value mode
// instead of in top level document mode. This enables the creation of a
// ValueReader for a single BSON value.
func newBufferedValueReader(t Type, b []byte) ValueReader {
	bVR := newBufferedDocumentReader(b)

	bVR.stack[0].vType = t
	bVR.stack[0].mode = mValue

	return bVR
}

func newBufferedDocumentReader(b []byte) *valueReader {
	vr := vrPool.Get().(*valueReader)

	vr.src = &bufferedByteSrc{
		buf:    b,
		offset: 0,
	}

	// Reset parse state.
	vr.frame = 0
	if cap(vr.stack) < 1 {
		vr.stack = make([]vrState, 1, 5)
	} else {
		vr.stack = vr.stack[:1]
	}

	vr.stack[0] = vrState{
		mode: mTopLevel,
		end:  int64(len(b)),
	}

	return vr
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

func (vr *valueReader) pop() error {
	var cnt int
	switch vr.stack[vr.frame].mode {
	case mElement, mValue:
		cnt = 1
	case mDocument, mArray, mCodeWithScope:
		cnt = 2 // we pop twice to jump over the vrElement: vrDocument -> vrElement -> vrDocument/TopLevel/etc...
	}
	for i := 0; i < cnt && vr.frame > 0; i++ {
		if vr.src.pos() < vr.stack[vr.frame].end {
			_, err := vr.src.discard(int(vr.stack[vr.frame].end - vr.src.pos()))
			if err != nil {
				return err
			}
		}
		vr.frame--
	}

	if vr.src.streamable() {
		if vr.frame == 0 {
			if vr.stack[0].end > vr.src.pos() {
				vr.stack[0].end -= vr.src.pos()
			} else {
				vr.stack[0].end = 0
			}

			vr.src.reset()
		}
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

// peekNextValueSize returns the length of the next value in the stream without
// offsetting the reader position.
func peekNextValueSize(vr *valueReader) (int32, error) {
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
		length, err = vr.src.regexLength()
	default:
		return 0, fmt.Errorf("attempted to read bytes of unknown BSON type %v", vr.stack[vr.frame].vType)
	}

	return length, err
}

// readBytes tries to grab the next n bytes zero-allocation using peek+discard.
// If peek fails (e.g. bufio buffer full), it falls back to io.ReadFull.
func readBytes(src byteSrc, n int) ([]byte, error) {
	if src.streamable() {
		data := make([]byte, n)
		if _, err := src.readExact(data); err != nil {
			return nil, err
		}

		return data, nil
	}

	// Zero-allocation path.
	buf, err := src.peek(n)
	if err != nil {
		return nil, err
	}

	_, _ = src.discard(n) // Discard the bytes from the source.
	return buf, nil
}

// readBytesValueReader returns a subslice [offset, offset+length) or EOF.
func (vr *valueReader) readBytes(n int32) ([]byte, error) {
	if n < 0 {
		return nil, fmt.Errorf("invalid length: %d", n)
	}

	return readBytes(vr.src, int(n))
}

//nolint:unparam
func (vr *valueReader) readValueBytes(dst []byte) (Type, []byte, error) {
	switch vr.stack[vr.frame].mode {
	case mTopLevel:
		length, err := vr.peekLength()
		if err != nil {
			return 0, nil, err
		}
		b, err := vr.readBytes(length)
		return Type(0), append(dst, b...), err
	case mElement, mValue:
		t := vr.stack[vr.frame].vType

		length, err := peekNextValueSize(vr)
		if err != nil {
			return t, dst, err
		}

		b, err := vr.readBytes(length)

		if err := vr.pop(); err != nil {
			return Type(0), nil, err
		}

		return t, append(dst, b...), err

	default:
		return Type(0), nil, vr.invalidTransitionErr(0, "readValueBytes", []mode{mElement, mValue})
	}
}

func (vr *valueReader) Skip() error {
	switch vr.stack[vr.frame].mode {
	case mElement, mValue:
	default:
		return vr.invalidTransitionErr(0, "Skip", []mode{mElement, mValue})
	}

	length, err := peekNextValueSize(vr)
	if err != nil {
		return err
	}

	_, err = vr.src.discard(int(length))
	if err != nil {
		return err
	}

	return vr.pop()
}

// ReadArray returns an ArrayReader for the next BSON array in the valueReader
// source, advancing the reader position to the end of the array.
func (vr *valueReader) ReadArray() (ArrayReader, error) {
	if err := vr.ensureElementValue(TypeArray, mArray, "ReadArray"); err != nil {
		return nil, err
	}

	// Push a new frame for the array.
	vr.advanceFrame()

	// Read the 4-byte length.
	size, err := vr.readLength()
	if err != nil {
		return nil, err
	}

	// Compute the end position: current position + total size - length.
	vr.stack[vr.frame].mode = mArray
	vr.stack[vr.frame].end = vr.src.pos() + int64(size) - 4

	return vr, nil
}

// ReadBinary reads a BSON binary value, returning the byte slice and the
// type of the binary data (0x02 for old binary, 0x00 for new binary, etc.),
// advancing the reader position to the end of the binary value.
func (vr *valueReader) ReadBinary() ([]byte, byte, error) {
	if err := vr.ensureElementValue(TypeBinary, 0, "ReadBinary"); err != nil {
		return nil, 0, err
	}

	length, err := vr.readLength()
	if err != nil {
		return nil, 0, err
	}

	btype, err := vr.readByte()
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

	b, err := vr.readBytes(length)
	if err != nil {
		return nil, 0, err
	}

	// copy so user doesn’t share underlying buffer
	cp := make([]byte, len(b))
	copy(cp, b)

	if err := vr.pop(); err != nil {
		return nil, 0, err
	}

	return cp, btype, nil
}

// ReadBoolean reads a BSON boolean value, returning true or false, advancing
// the reader position to the end of the boolean value.
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

// ReadDocument reads a BSON embedded document, returning a DocumentReader,
// advancing the reader position to the end of the document.
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

		vr.stack[vr.frame].end = int64(length) + vr.src.pos() - 4
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
	vr.stack[vr.frame].end = int64(size) + vr.src.pos() - 4

	return vr, nil
}

// ReadCodeWithScope reads a BSON CodeWithScope value, returning the code as a
// string, advancing the reader position to the end of the CodeWithScope value.
func (vr *valueReader) ReadCodeWithScope() (string, DocumentReader, error) {
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
	buf, err := vr.readBytes(strLength)
	if err != nil {
		return "", nil, err
	}

	code := string(buf[:len(buf)-1])
	vr.advanceFrame()

	// Use readLength to ensure that we are not out of bounds.
	size, err := vr.readLength()
	if err != nil {
		return "", nil, err
	}

	vr.stack[vr.frame].mode = mCodeWithScope
	vr.stack[vr.frame].end = vr.src.pos() + int64(size) - 4

	// The total length should equal:
	// 4 (total length) + strLength + 4 (the length of str itself) + (document length)
	componentsLength := int64(4+strLength+4) + int64(size)
	if int64(totalLength) != componentsLength {
		return "", nil, fmt.Errorf(
			"length of CodeWithScope does not match lengths of components; total: %d; components: %d",
			totalLength, componentsLength,
		)
	}
	return code, vr, nil
}

// ReadDBPointer reads a BSON DBPointer value, returning the namespace, the
// object ID, and an error if any, advancing the reader position to the end of
// the DBPointer value.
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

	if err := vr.pop(); err != nil {
		return "", ObjectID{}, err
	}
	return ns, oid, nil
}

// ReadDateTime reads a BSON DateTime value, advancing the reader position to
// the end of the DateTime value.
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

// ReadDecimal128 reads a BSON Decimal128 value, advancing the reader
// to the end of the Decimal128 value.
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

	if err := vr.pop(); err != nil {
		return Decimal128{}, err
	}
	return NewDecimal128(h, l), nil
}

// ReadDouble reads a BSON double value, advancing the reader position to
// to the end of the double value.
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

// ReadInt32 reads a BSON int32 value, advancing the reader position to the end
// of the int32 value.
func (vr *valueReader) ReadInt32() (int32, error) {
	if err := vr.ensureElementValue(TypeInt32, 0, "ReadInt32"); err != nil {
		return 0, err
	}
	i, err := vr.readi32()
	if err != nil {
		return 0, err
	}

	if err := vr.pop(); err != nil {
		return 0, err
	}
	return i, nil
}

// ReadInt64 reads a BSON int64 value, advancing the reader position to the end
// of the int64 value.
func (vr *valueReader) ReadInt64() (int64, error) {
	if err := vr.ensureElementValue(TypeInt64, 0, "ReadInt64"); err != nil {
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

// ReadJavascript reads a BSON JavaScript value, advancing the reader
// to the end of the JavaScript value.
func (vr *valueReader) ReadJavascript() (string, error) {
	if err := vr.ensureElementValue(TypeJavaScript, 0, "ReadJavascript"); err != nil {
		return "", err
	}
	s, err := vr.readString()
	if err != nil {
		return "", err
	}

	if err := vr.pop(); err != nil {
		return "", err
	}
	return s, nil
}

// ReadMaxKey reads a BSON MaxKey value, advancing the reader position to the
// end of the MaxKey value.
func (vr *valueReader) ReadMaxKey() error {
	if err := vr.ensureElementValue(TypeMaxKey, 0, "ReadMaxKey"); err != nil {
		return err
	}

	return vr.pop()
}

// ReadMinKey reads a BSON MinKey value, advancing the reader position to the
// end of the MinKey value.
func (vr *valueReader) ReadMinKey() error {
	if err := vr.ensureElementValue(TypeMinKey, 0, "ReadMinKey"); err != nil {
		return err
	}

	return vr.pop()
}

// REadNull reads a BSON Null value, advancing the reader position to the
// end of the Null value.
func (vr *valueReader) ReadNull() error {
	if err := vr.ensureElementValue(TypeNull, 0, "ReadNull"); err != nil {
		return err
	}

	return vr.pop()
}

// ReadObjectID reads a BSON ObjectID value, advancing the reader to the end of
// the ObjectID value.
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

	if err := vr.pop(); err != nil {
		return ObjectID{}, err
	}
	return oid, nil
}

// ReadRegex reads a BSON Regex value, advancing the reader position to the
// regex value.
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

// ReadString reads a BSON String value, advancing the reader position to the
// end of the String value.
func (vr *valueReader) ReadString() (string, error) {
	if err := vr.ensureElementValue(TypeString, 0, "ReadString"); err != nil {
		return "", err
	}
	s, err := vr.readString()
	if err != nil {
		return "", err
	}

	if err := vr.pop(); err != nil {
		return "", err
	}
	return s, nil
}

// ReadSymbol reads a BSON Symbol value, advancing the reader position to the
// end of the Symbol value.
func (vr *valueReader) ReadSymbol() (string, error) {
	if err := vr.ensureElementValue(TypeSymbol, 0, "ReadSymbol"); err != nil {
		return "", err
	}
	s, err := vr.readString()
	if err != nil {
		return "", err
	}
	if err := vr.pop(); err != nil {
		return "", err
	}
	return s, nil
}

// ReadTimestamp reads a BSON Timestamp value, advancing the reader to the end
// of the Timestamp value.
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

	if err := vr.pop(); err != nil {
		return 0, 0, err
	}
	return t, i, nil
}

// ReadUndefined reads a BSON Undefined value, advancing the reader position
// to the end of the Undefined value.
func (vr *valueReader) ReadUndefined() error {
	if err := vr.ensureElementValue(TypeUndefined, 0, "ReadUndefined"); err != nil {
		return err
	}

	return vr.pop()
}

// ReadElement reads the next element in the BSON document, advancing the
// reader position to the end of the element.
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
		if vr.src.pos() != vr.stack[vr.frame].end {
			return "", nil, vr.invalidDocumentLengthError()
		}

		_ = vr.pop() // Ignore the error because the call here never reads from the underlying reader.
		return "", nil, ErrEOD
	}

	name, err := vr.readCString()
	if err != nil {
		return "", nil, err
	}

	vr.advanceFrame()

	vr.stack[vr.frame].mode = mElement
	vr.stack[vr.frame].vType = Type(t)
	return name, vr, nil
}

// ReadValue reads the next value in the BSON array, advancing the to the end of
// the value.
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
		if vr.src.pos() != vr.stack[vr.frame].end {
			return nil, vr.invalidDocumentLengthError()
		}

		_ = vr.pop() // Ignore the error because the call here never reads from the underlying reader.
		return nil, ErrEOA
	}

	_, err = vr.src.readSlice(0x00)
	if err != nil {
		return nil, err
	}

	vr.advanceFrame()

	vr.stack[vr.frame].mode = mValue
	vr.stack[vr.frame].vType = Type(t)
	return vr, nil
}

func (vr *valueReader) readByte() (byte, error) {
	b, err := vr.src.ReadByte()
	if err != nil {
		return 0x0, err
	}
	return b, nil
}

func (vr *valueReader) readCString() (string, error) {
	data, err := vr.src.readSlice(0x00)
	if err != nil {
		return "", err
	}
	return string(data[:len(data)-1]), nil
}

func (vr *valueReader) readString() (string, error) {
	length, err := vr.readLength()
	if err != nil {
		return "", err
	}

	if length <= 0 {
		return "", fmt.Errorf("invalid string length: %d", length)
	}

	raw, err := readBytes(vr.src, int(length))
	if err != nil {
		return "", err
	}

	// Check that the last byte is the NUL terminator.
	if raw[len(raw)-1] != 0x00 {
		return "", fmt.Errorf("string does not end with null byte, but with %v", raw[len(raw)-1])
	}

	// Convert and strip the trailing NUL.
	return string(raw[:len(raw)-1]), nil
}

func (vr *valueReader) peekLength() (int32, error) {
	buf, err := vr.src.peek(4)
	if err != nil {
		return 0, err
	}
	return int32(binary.LittleEndian.Uint32(buf)), nil
}

func (vr *valueReader) readLength() (int32, error) {
	return vr.readi32()
}

func (vr *valueReader) readi32() (int32, error) {
	raw, err := readBytes(vr.src, 4)
	if err != nil {
		return 0, err
	}

	return int32(binary.LittleEndian.Uint32(raw)), nil
}

func (vr *valueReader) readu32() (uint32, error) {
	raw, err := readBytes(vr.src, 4)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint32(raw), nil
}

func (vr *valueReader) readi64() (int64, error) {
	raw, err := readBytes(vr.src, 8)
	if err != nil {
		return 0, err
	}

	return int64(binary.LittleEndian.Uint64(raw)), nil
}

func (vr *valueReader) readu64() (uint64, error) {
	raw, err := readBytes(vr.src, 8)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint64(raw), nil
}
