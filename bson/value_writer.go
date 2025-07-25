// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"sync"

	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

var _ ValueWriter = &valueWriter{}

var vwPool = sync.Pool{
	New: func() any {
		return new(valueWriter)
	},
}

func putValueWriter(vw *valueWriter) {
	if vw != nil {
		vw.w = nil // don't leak the writer
		vwPool.Put(vw)
	}
}

var documentWriterPool = sync.Pool{
	New: func() any {
		return newDocumentWriter(nil)
	},
}

func getDocumentWriter(w io.Writer) *valueWriter {
	vw := documentWriterPool.Get().(*valueWriter)

	vw.reset(vw.buf)
	vw.buf = vw.buf[:0]
	vw.w = w

	return vw
}

func putDocumentWriter(vw *valueWriter) {
	if vw != nil {
		vw.w = nil // don't leak the writer
		documentWriterPool.Put(vw)
	}
}

// This is here so that during testing we can change it and not require
// allocating a 4GB slice.
var maxSize = math.MaxInt32

type errMaxDocumentSizeExceeded struct {
	size int64
}

func (mdse errMaxDocumentSizeExceeded) Error() string {
	return fmt.Sprintf("document size (%d) is larger than the max int32", mdse.size)
}

type vwMode int

const (
	_ vwMode = iota
	vwTopLevel
	vwDocument
	vwArray
	vwValue
	vwElement
	vwCodeWithScope
)

func (vm vwMode) String() string {
	var str string

	switch vm {
	case vwTopLevel:
		str = "TopLevel"
	case vwDocument:
		str = "DocumentMode"
	case vwArray:
		str = "ArrayMode"
	case vwValue:
		str = "ValueMode"
	case vwElement:
		str = "ElementMode"
	case vwCodeWithScope:
		str = "CodeWithScopeMode"
	default:
		str = "UnknownMode"
	}

	return str
}

type vwState struct {
	mode   mode
	key    string
	arrkey int
	start  int32
}

type valueWriter struct {
	w   io.Writer
	buf []byte

	stack []vwState
	frame int64
}

func (vw *valueWriter) advanceFrame() {
	vw.frame++
	if vw.frame >= int64(len(vw.stack)) {
		vw.stack = append(vw.stack, vwState{})
	}
}

func (vw *valueWriter) push(m mode) {
	vw.advanceFrame()

	// Clean the stack
	vw.stack[vw.frame] = vwState{mode: m}

	switch m {
	case mDocument, mArray, mCodeWithScope:
		vw.reserveLength() // WARN: this is not needed
	}
}

func (vw *valueWriter) reserveLength() {
	vw.stack[vw.frame].start = int32(len(vw.buf))
	vw.buf = append(vw.buf, 0x00, 0x00, 0x00, 0x00)
}

func (vw *valueWriter) pop() {
	switch vw.stack[vw.frame].mode {
	case mElement, mValue:
		vw.frame--
	case mDocument, mArray, mCodeWithScope:
		vw.frame -= 2 // we pop twice to jump over the mElement: mDocument -> mElement -> mDocument/mTopLevel/etc...
	}
}

// NewDocumentWriter creates a ValueWriter that writes BSON to w.
//
// This ValueWriter will only write entire documents to the io.Writer and it
// will buffer the document as it is built.
func NewDocumentWriter(w io.Writer) ValueWriter {
	return newDocumentWriter(w)
}

func newDocumentWriter(w io.Writer) *valueWriter {
	vw := new(valueWriter)
	stack := make([]vwState, 1, 5)
	stack[0] = vwState{mode: mTopLevel}
	vw.w = w
	vw.stack = stack

	return vw
}

func newValueWriterFromSlice(buf []byte) *valueWriter {
	vw := new(valueWriter)
	stack := make([]vwState, 1, 5)
	stack[0] = vwState{mode: mTopLevel}
	vw.stack = stack
	vw.buf = buf

	return vw
}

func (vw *valueWriter) reset(buf []byte) {
	if vw.stack == nil {
		vw.stack = make([]vwState, 1, 5)
	}
	vw.stack = vw.stack[:1]
	vw.stack[0] = vwState{mode: mTopLevel}
	vw.buf = buf
	vw.frame = 0
	vw.w = nil
}

func (vw *valueWriter) invalidTransitionError(destination mode, name string, modes []mode) error {
	te := TransitionError{
		name:        name,
		current:     vw.stack[vw.frame].mode,
		destination: destination,
		modes:       modes,
		action:      "write",
	}
	if vw.frame != 0 {
		te.parent = vw.stack[vw.frame-1].mode
	}
	return te
}

func (vw *valueWriter) writeElementHeader(t Type, destination mode, callerName string, addmodes ...mode) error {
	frame := &vw.stack[vw.frame]
	switch frame.mode {
	case mElement:
		key := frame.key
		if !isValidCString(key) {
			return errors.New("BSON element key cannot contain null bytes")
		}
		vw.appendHeader(t, key)
	case mValue:
		vw.appendIntHeader(t, frame.arrkey)
	default:
		modes := []mode{mElement, mValue}
		if addmodes != nil {
			modes = append(modes, addmodes...)
		}
		return vw.invalidTransitionError(destination, callerName, modes)
	}

	return nil
}

func (vw *valueWriter) writeValueBytes(t Type, b []byte) error {
	if err := vw.writeElementHeader(t, mode(0), "WriteValueBytes"); err != nil {
		return err
	}
	vw.buf = append(vw.buf, b...)
	vw.pop()
	return nil
}

func (vw *valueWriter) WriteArray() (ArrayWriter, error) {
	if err := vw.writeElementHeader(TypeArray, mArray, "WriteArray"); err != nil {
		return nil, err
	}

	vw.push(mArray)

	return vw, nil
}

func (vw *valueWriter) WriteBinary(b []byte) error {
	return vw.WriteBinaryWithSubtype(b, 0x00)
}

func (vw *valueWriter) WriteBinaryWithSubtype(b []byte, btype byte) error {
	if err := vw.writeElementHeader(TypeBinary, mode(0), "WriteBinaryWithSubtype"); err != nil {
		return err
	}

	vw.buf = bsoncore.AppendBinary(vw.buf, btype, b)
	vw.pop()
	return nil
}

func (vw *valueWriter) WriteBoolean(b bool) error {
	if err := vw.writeElementHeader(TypeBoolean, mode(0), "WriteBoolean"); err != nil {
		return err
	}

	vw.buf = bsoncore.AppendBoolean(vw.buf, b)
	vw.pop()
	return nil
}

func (vw *valueWriter) WriteCodeWithScope(code string) (DocumentWriter, error) {
	if err := vw.writeElementHeader(TypeCodeWithScope, mCodeWithScope, "WriteCodeWithScope"); err != nil {
		return nil, err
	}

	// CodeWithScope is a different than other types because we need an extra
	// frame on the stack. In the EndDocument code, we write the document
	// length, pop, write the code with scope length, and pop. To simplify the
	// pop code, we push a spacer frame that we'll always jump over.
	vw.push(mCodeWithScope)
	vw.buf = bsoncore.AppendString(vw.buf, code)
	vw.push(mSpacer)
	vw.push(mDocument)

	return vw, nil
}

func (vw *valueWriter) WriteDBPointer(ns string, oid ObjectID) error {
	if err := vw.writeElementHeader(TypeDBPointer, mode(0), "WriteDBPointer"); err != nil {
		return err
	}

	vw.buf = bsoncore.AppendDBPointer(vw.buf, ns, oid)
	vw.pop()
	return nil
}

func (vw *valueWriter) WriteDateTime(dt int64) error {
	if err := vw.writeElementHeader(TypeDateTime, mode(0), "WriteDateTime"); err != nil {
		return err
	}

	vw.buf = bsoncore.AppendDateTime(vw.buf, dt)
	vw.pop()
	return nil
}

func (vw *valueWriter) WriteDecimal128(d128 Decimal128) error {
	if err := vw.writeElementHeader(TypeDecimal128, mode(0), "WriteDecimal128"); err != nil {
		return err
	}

	h, l := d128.GetBytes()
	vw.buf = bsoncore.AppendDecimal128(vw.buf, h, l)
	vw.pop()
	return nil
}

func (vw *valueWriter) WriteDouble(f float64) error {
	if err := vw.writeElementHeader(TypeDouble, mode(0), "WriteDouble"); err != nil {
		return err
	}

	vw.buf = bsoncore.AppendDouble(vw.buf, f)
	vw.pop()
	return nil
}

func (vw *valueWriter) WriteInt32(i32 int32) error {
	if err := vw.writeElementHeader(TypeInt32, mode(0), "WriteInt32"); err != nil {
		return err
	}

	vw.buf = bsoncore.AppendInt32(vw.buf, i32)
	vw.pop()
	return nil
}

func (vw *valueWriter) WriteInt64(i64 int64) error {
	if err := vw.writeElementHeader(TypeInt64, mode(0), "WriteInt64"); err != nil {
		return err
	}

	vw.buf = bsoncore.AppendInt64(vw.buf, i64)
	vw.pop()
	return nil
}

func (vw *valueWriter) WriteJavascript(code string) error {
	if err := vw.writeElementHeader(TypeJavaScript, mode(0), "WriteJavascript"); err != nil {
		return err
	}

	vw.buf = bsoncore.AppendJavaScript(vw.buf, code)
	vw.pop()
	return nil
}

func (vw *valueWriter) WriteMaxKey() error {
	if err := vw.writeElementHeader(TypeMaxKey, mode(0), "WriteMaxKey"); err != nil {
		return err
	}

	vw.pop()
	return nil
}

func (vw *valueWriter) WriteMinKey() error {
	if err := vw.writeElementHeader(TypeMinKey, mode(0), "WriteMinKey"); err != nil {
		return err
	}

	vw.pop()
	return nil
}

func (vw *valueWriter) WriteNull() error {
	if err := vw.writeElementHeader(TypeNull, mode(0), "WriteNull"); err != nil {
		return err
	}

	vw.pop()
	return nil
}

func (vw *valueWriter) WriteObjectID(oid ObjectID) error {
	if err := vw.writeElementHeader(TypeObjectID, mode(0), "WriteObjectID"); err != nil {
		return err
	}

	vw.buf = bsoncore.AppendObjectID(vw.buf, oid)
	vw.pop()
	return nil
}

func (vw *valueWriter) WriteRegex(pattern string, options string) error {
	if !isValidCString(pattern) || !isValidCString(options) {
		return errors.New("BSON regex values cannot contain null bytes")
	}
	if err := vw.writeElementHeader(TypeRegex, mode(0), "WriteRegex"); err != nil {
		return err
	}

	vw.buf = bsoncore.AppendRegex(vw.buf, pattern, sortStringAlphebeticAscending(options))
	vw.pop()
	return nil
}

func (vw *valueWriter) WriteString(s string) error {
	if err := vw.writeElementHeader(TypeString, mode(0), "WriteString"); err != nil {
		return err
	}

	vw.buf = bsoncore.AppendString(vw.buf, s)
	vw.pop()
	return nil
}

func (vw *valueWriter) WriteDocument() (DocumentWriter, error) {
	if vw.stack[vw.frame].mode == mTopLevel {
		vw.reserveLength()
		return vw, nil
	}
	if err := vw.writeElementHeader(TypeEmbeddedDocument, mDocument, "WriteDocument", mTopLevel); err != nil {
		return nil, err
	}

	vw.push(mDocument)
	return vw, nil
}

func (vw *valueWriter) WriteSymbol(symbol string) error {
	if err := vw.writeElementHeader(TypeSymbol, mode(0), "WriteSymbol"); err != nil {
		return err
	}

	vw.buf = bsoncore.AppendSymbol(vw.buf, symbol)
	vw.pop()
	return nil
}

func (vw *valueWriter) WriteTimestamp(t uint32, i uint32) error {
	if err := vw.writeElementHeader(TypeTimestamp, mode(0), "WriteTimestamp"); err != nil {
		return err
	}

	vw.buf = bsoncore.AppendTimestamp(vw.buf, t, i)
	vw.pop()
	return nil
}

func (vw *valueWriter) WriteUndefined() error {
	if err := vw.writeElementHeader(TypeUndefined, mode(0), "WriteUndefined"); err != nil {
		return err
	}

	vw.pop()
	return nil
}

func (vw *valueWriter) WriteDocumentElement(key string) (ValueWriter, error) {
	switch vw.stack[vw.frame].mode {
	case mTopLevel, mDocument:
	default:
		return nil, vw.invalidTransitionError(mElement, "WriteDocumentElement", []mode{mTopLevel, mDocument})
	}

	vw.push(mElement)
	vw.stack[vw.frame].key = key

	return vw, nil
}

func (vw *valueWriter) WriteDocumentEnd() error {
	switch vw.stack[vw.frame].mode {
	case mTopLevel, mDocument:
	default:
		return fmt.Errorf("incorrect mode to end document: %s", vw.stack[vw.frame].mode)
	}

	vw.buf = append(vw.buf, 0x00)

	err := vw.writeLength()
	if err != nil {
		return err
	}

	if vw.stack[vw.frame].mode == mTopLevel {
		if err = vw.Flush(); err != nil {
			return err
		}
	}

	vw.pop()

	if vw.stack[vw.frame].mode == mCodeWithScope {
		// We ignore the error here because of the guarantee of writeLength.
		// See the docs for writeLength for more info.
		_ = vw.writeLength()
		vw.pop()
	}
	return nil
}

func (vw *valueWriter) Flush() error {
	if vw.w == nil {
		return nil
	}

	if _, err := vw.w.Write(vw.buf); err != nil {
		return err
	}
	// reset buffer
	vw.buf = vw.buf[:0]
	return nil
}

func (vw *valueWriter) WriteArrayElement() (ValueWriter, error) {
	if vw.stack[vw.frame].mode != mArray {
		return nil, vw.invalidTransitionError(mValue, "WriteArrayElement", []mode{mArray})
	}

	arrkey := vw.stack[vw.frame].arrkey
	vw.stack[vw.frame].arrkey++

	vw.push(mValue)
	vw.stack[vw.frame].arrkey = arrkey

	return vw, nil
}

func (vw *valueWriter) WriteArrayEnd() error {
	if vw.stack[vw.frame].mode != mArray {
		return fmt.Errorf("incorrect mode to end array: %s", vw.stack[vw.frame].mode)
	}

	vw.buf = append(vw.buf, 0x00)

	err := vw.writeLength()
	if err != nil {
		return err
	}

	vw.pop()
	return nil
}

// NOTE: We assume that if we call writeLength more than once the same function
// within the same function without altering the vw.buf that this method will
// not return an error. If this changes ensure that the following methods are
// updated:
//
// - WriteDocumentEnd
func (vw *valueWriter) writeLength() error {
	length := len(vw.buf)
	if length > maxSize {
		return errMaxDocumentSizeExceeded{size: int64(len(vw.buf))}
	}
	frame := &vw.stack[vw.frame]
	length -= int(frame.start)
	start := frame.start

	_ = vw.buf[start+3] // BCE
	vw.buf[start+0] = byte(length)
	vw.buf[start+1] = byte(length >> 8)
	vw.buf[start+2] = byte(length >> 16)
	vw.buf[start+3] = byte(length >> 24)
	return nil
}

func isValidCString(cs string) bool {
	// Disallow the zero byte in a cstring because the zero byte is used as the
	// terminating character.
	//
	// It's safe to check bytes instead of runes because all multibyte UTF-8
	// code points start with (binary) 11xxxxxx or 10xxxxxx, so 00000000 (i.e.
	// 0) will never be part of a multibyte UTF-8 code point. This logic is the
	// same as the "r < utf8.RuneSelf" case in strings.IndexRune but can be
	// inlined.
	//
	// https://cs.opensource.google/go/go/+/refs/tags/go1.21.1:src/strings/strings.go;l=127
	return strings.IndexByte(cs, 0) == -1
}

// appendHeader is the same as bsoncore.AppendHeader but does not check if the
// key is a valid C string since the caller has already checked for that.
//
// The caller of this function must check if key is a valid C string.
func (vw *valueWriter) appendHeader(t Type, key string) {
	vw.buf = bsoncore.AppendType(vw.buf, bsoncore.Type(t))
	vw.buf = append(vw.buf, key...)
	vw.buf = append(vw.buf, 0x00)
}

func (vw *valueWriter) appendIntHeader(t Type, key int) {
	vw.buf = bsoncore.AppendType(vw.buf, bsoncore.Type(t))
	vw.buf = strconv.AppendInt(vw.buf, int64(key), 10)
	vw.buf = append(vw.buf, 0x00)
}
