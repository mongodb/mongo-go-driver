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
)

type ejvrState struct {
	mode  mode
	vType Type
	depth int
}

// extJSONValueReader is for reading extended JSON.
type extJSONValueReader struct {
	p *extJSONParser

	stack []ejvrState
	frame int
}

// NewExtJSONValueReader returns a ValueReader that reads Extended JSON values
// from r. If canonicalOnly is true, reading values from the ValueReader returns
// an error if the Extended JSON was not marshaled in canonical mode.
func NewExtJSONValueReader(r io.Reader, canonicalOnly bool) (ValueReader, error) {
	return newExtJSONValueReader(r, canonicalOnly)
}

func newExtJSONValueReader(r io.Reader, canonicalOnly bool) (*extJSONValueReader, error) {
	ejvr := new(extJSONValueReader)
	return ejvr.reset(r, canonicalOnly)
}

func (ejvr *extJSONValueReader) reset(r io.Reader, canonicalOnly bool) (*extJSONValueReader, error) {
	p := newExtJSONParser(r, canonicalOnly)
	typ, err := p.peekType()
	if err != nil {
		return nil, ErrInvalidJSON
	}

	var m mode
	switch typ {
	case TypeEmbeddedDocument:
		m = mTopLevel
	case TypeArray:
		m = mArray
	default:
		m = mValue
	}

	stack := make([]ejvrState, 1, 5)
	stack[0] = ejvrState{
		mode:  m,
		vType: typ,
	}
	return &extJSONValueReader{
		p:     p,
		stack: stack,
	}, nil
}

func (ejvr *extJSONValueReader) advanceFrame() {
	if ejvr.frame+1 >= len(ejvr.stack) { // We need to grow the stack
		length := len(ejvr.stack)
		if length+1 >= cap(ejvr.stack) {
			// double it
			buf := make([]ejvrState, 2*cap(ejvr.stack)+1)
			copy(buf, ejvr.stack)
			ejvr.stack = buf
		}
		ejvr.stack = ejvr.stack[:length+1]
	}
	ejvr.frame++

	// Clean the stack
	ejvr.stack[ejvr.frame].mode = 0
	ejvr.stack[ejvr.frame].vType = 0
	ejvr.stack[ejvr.frame].depth = 0
}

func (ejvr *extJSONValueReader) pushDocument() {
	ejvr.advanceFrame()

	ejvr.stack[ejvr.frame].mode = mDocument
	ejvr.stack[ejvr.frame].depth = ejvr.p.depth
}

func (ejvr *extJSONValueReader) pushCodeWithScope() {
	ejvr.advanceFrame()

	ejvr.stack[ejvr.frame].mode = mCodeWithScope
}

func (ejvr *extJSONValueReader) pushArray() {
	ejvr.advanceFrame()

	ejvr.stack[ejvr.frame].mode = mArray
}

func (ejvr *extJSONValueReader) push(m mode, t Type) {
	ejvr.advanceFrame()

	ejvr.stack[ejvr.frame].mode = m
	ejvr.stack[ejvr.frame].vType = t
}

func (ejvr *extJSONValueReader) pop() {
	switch ejvr.stack[ejvr.frame].mode {
	case mElement, mValue:
		ejvr.frame--
	case mDocument, mArray, mCodeWithScope:
		ejvr.frame -= 2 // we pop twice to jump over the vrElement: vrDocument -> vrElement -> vrDocument/TopLevel/etc...
	}
}

func (ejvr *extJSONValueReader) skipObject() {
	// read entire object until depth returns to 0 (last ending } or ] seen)
	depth := 1
	for depth > 0 {
		ejvr.p.advanceState()

		// If object is empty, raise depth and continue. When emptyObject is true, the
		// parser has already read both the opening and closing brackets of an empty
		// object ("{}"), so the next valid token will be part of the parent document,
		// not part of the nested document.
		//
		// If there is a comma, there are remaining fields, emptyObject must be set back
		// to false, and comma must be skipped with advanceState().
		if ejvr.p.emptyObject {
			if ejvr.p.s == jpsSawComma {
				ejvr.p.emptyObject = false
				ejvr.p.advanceState()
			}
			depth--
			continue
		}

		switch ejvr.p.s {
		case jpsSawBeginObject, jpsSawBeginArray:
			depth++
		case jpsSawEndObject, jpsSawEndArray:
			depth--
		}
	}
}

func (ejvr *extJSONValueReader) invalidTransitionErr(destination mode, name string, modes []mode) error {
	te := TransitionError{
		name:        name,
		current:     ejvr.stack[ejvr.frame].mode,
		destination: destination,
		modes:       modes,
		action:      "read",
	}
	if ejvr.frame != 0 {
		te.parent = ejvr.stack[ejvr.frame-1].mode
	}
	return te
}

func (ejvr *extJSONValueReader) typeError(t Type) error {
	return fmt.Errorf("positioned on %s, but attempted to read %s", ejvr.stack[ejvr.frame].vType, t)
}

func (ejvr *extJSONValueReader) ensureElementValue(t Type, destination mode, callerName string, addModes ...mode) error {
	switch ejvr.stack[ejvr.frame].mode {
	case mElement, mValue:
		if ejvr.stack[ejvr.frame].vType != t {
			return ejvr.typeError(t)
		}
	default:
		modes := []mode{mElement, mValue}
		if addModes != nil {
			modes = append(modes, addModes...)
		}
		return ejvr.invalidTransitionErr(destination, callerName, modes)
	}

	return nil
}

func (ejvr *extJSONValueReader) Type() Type {
	return ejvr.stack[ejvr.frame].vType
}

func (ejvr *extJSONValueReader) Skip() error {
	switch ejvr.stack[ejvr.frame].mode {
	case mElement, mValue:
	default:
		return ejvr.invalidTransitionErr(0, "Skip", []mode{mElement, mValue})
	}

	defer ejvr.pop()

	t := ejvr.stack[ejvr.frame].vType
	switch t {
	case TypeArray, TypeEmbeddedDocument, TypeCodeWithScope:
		// read entire array, doc or CodeWithScope
		ejvr.skipObject()
	default:
		_, err := ejvr.p.readValue(t)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ejvr *extJSONValueReader) ReadArray() (ArrayReader, error) {
	switch ejvr.stack[ejvr.frame].mode {
	case mTopLevel: // allow reading array from top level
	case mArray:
		return ejvr, nil
	default:
		if err := ejvr.ensureElementValue(TypeArray, mArray, "ReadArray", mTopLevel, mArray); err != nil {
			return nil, err
		}
	}

	ejvr.pushArray()

	return ejvr, nil
}

func (ejvr *extJSONValueReader) ReadBinary() (b []byte, btype byte, err error) {
	if err := ejvr.ensureElementValue(TypeBinary, 0, "ReadBinary"); err != nil {
		return nil, 0, err
	}

	v, err := ejvr.p.readValue(TypeBinary)
	if err != nil {
		return nil, 0, err
	}

	b, btype, err = v.parseBinary()

	ejvr.pop()
	return b, btype, err
}

func (ejvr *extJSONValueReader) ReadBoolean() (bool, error) {
	if err := ejvr.ensureElementValue(TypeBoolean, 0, "ReadBoolean"); err != nil {
		return false, err
	}

	v, err := ejvr.p.readValue(TypeBoolean)
	if err != nil {
		return false, err
	}

	if v.t != TypeBoolean {
		return false, fmt.Errorf("expected type bool, but got type %s", v.t)
	}

	ejvr.pop()
	return v.v.(bool), nil
}

func (ejvr *extJSONValueReader) ReadDocument() (DocumentReader, error) {
	switch ejvr.stack[ejvr.frame].mode {
	case mTopLevel:
		return ejvr, nil
	case mElement, mValue:
		if ejvr.stack[ejvr.frame].vType != TypeEmbeddedDocument {
			return nil, ejvr.typeError(TypeEmbeddedDocument)
		}

		ejvr.pushDocument()
		return ejvr, nil
	default:
		return nil, ejvr.invalidTransitionErr(mDocument, "ReadDocument", []mode{mTopLevel, mElement, mValue})
	}
}

func (ejvr *extJSONValueReader) ReadCodeWithScope() (code string, dr DocumentReader, err error) {
	if err = ejvr.ensureElementValue(TypeCodeWithScope, 0, "ReadCodeWithScope"); err != nil {
		return "", nil, err
	}

	v, err := ejvr.p.readValue(TypeCodeWithScope)
	if err != nil {
		return "", nil, err
	}

	code, err = v.parseJavascript()

	ejvr.pushCodeWithScope()
	return code, ejvr, err
}

func (ejvr *extJSONValueReader) ReadDBPointer() (ns string, oid ObjectID, err error) {
	if err = ejvr.ensureElementValue(TypeDBPointer, 0, "ReadDBPointer"); err != nil {
		return "", NilObjectID, err
	}

	v, err := ejvr.p.readValue(TypeDBPointer)
	if err != nil {
		return "", NilObjectID, err
	}

	ns, oid, err = v.parseDBPointer()

	ejvr.pop()
	return ns, oid, err
}

func (ejvr *extJSONValueReader) ReadDateTime() (int64, error) {
	if err := ejvr.ensureElementValue(TypeDateTime, 0, "ReadDateTime"); err != nil {
		return 0, err
	}

	v, err := ejvr.p.readValue(TypeDateTime)
	if err != nil {
		return 0, err
	}

	d, err := v.parseDateTime()

	ejvr.pop()
	return d, err
}

func (ejvr *extJSONValueReader) ReadDecimal128() (Decimal128, error) {
	if err := ejvr.ensureElementValue(TypeDecimal128, 0, "ReadDecimal128"); err != nil {
		return Decimal128{}, err
	}

	v, err := ejvr.p.readValue(TypeDecimal128)
	if err != nil {
		return Decimal128{}, err
	}

	d, err := v.parseDecimal128()

	ejvr.pop()
	return d, err
}

func (ejvr *extJSONValueReader) ReadDouble() (float64, error) {
	if err := ejvr.ensureElementValue(TypeDouble, 0, "ReadDouble"); err != nil {
		return 0, err
	}

	v, err := ejvr.p.readValue(TypeDouble)
	if err != nil {
		return 0, err
	}

	d, err := v.parseDouble()

	ejvr.pop()
	return d, err
}

func (ejvr *extJSONValueReader) ReadInt32() (int32, error) {
	if err := ejvr.ensureElementValue(TypeInt32, 0, "ReadInt32"); err != nil {
		return 0, err
	}

	v, err := ejvr.p.readValue(TypeInt32)
	if err != nil {
		return 0, err
	}

	i, err := v.parseInt32()

	ejvr.pop()
	return i, err
}

func (ejvr *extJSONValueReader) ReadInt64() (int64, error) {
	if err := ejvr.ensureElementValue(TypeInt64, 0, "ReadInt64"); err != nil {
		return 0, err
	}

	v, err := ejvr.p.readValue(TypeInt64)
	if err != nil {
		return 0, err
	}

	i, err := v.parseInt64()

	ejvr.pop()
	return i, err
}

func (ejvr *extJSONValueReader) ReadJavascript() (code string, err error) {
	if err = ejvr.ensureElementValue(TypeJavaScript, 0, "ReadJavascript"); err != nil {
		return "", err
	}

	v, err := ejvr.p.readValue(TypeJavaScript)
	if err != nil {
		return "", err
	}

	code, err = v.parseJavascript()

	ejvr.pop()
	return code, err
}

func (ejvr *extJSONValueReader) ReadMaxKey() error {
	if err := ejvr.ensureElementValue(TypeMaxKey, 0, "ReadMaxKey"); err != nil {
		return err
	}

	v, err := ejvr.p.readValue(TypeMaxKey)
	if err != nil {
		return err
	}

	err = v.parseMinMaxKey("max")

	ejvr.pop()
	return err
}

func (ejvr *extJSONValueReader) ReadMinKey() error {
	if err := ejvr.ensureElementValue(TypeMinKey, 0, "ReadMinKey"); err != nil {
		return err
	}

	v, err := ejvr.p.readValue(TypeMinKey)
	if err != nil {
		return err
	}

	err = v.parseMinMaxKey("min")

	ejvr.pop()
	return err
}

func (ejvr *extJSONValueReader) ReadNull() error {
	if err := ejvr.ensureElementValue(TypeNull, 0, "ReadNull"); err != nil {
		return err
	}

	v, err := ejvr.p.readValue(TypeNull)
	if err != nil {
		return err
	}

	if v.t != TypeNull {
		return fmt.Errorf("expected type null but got type %s", v.t)
	}

	ejvr.pop()
	return nil
}

func (ejvr *extJSONValueReader) ReadObjectID() (ObjectID, error) {
	if err := ejvr.ensureElementValue(TypeObjectID, 0, "ReadObjectID"); err != nil {
		return ObjectID{}, err
	}

	v, err := ejvr.p.readValue(TypeObjectID)
	if err != nil {
		return ObjectID{}, err
	}

	oid, err := v.parseObjectID()

	ejvr.pop()
	return oid, err
}

func (ejvr *extJSONValueReader) ReadRegex() (pattern string, options string, err error) {
	if err = ejvr.ensureElementValue(TypeRegex, 0, "ReadRegex"); err != nil {
		return "", "", err
	}

	v, err := ejvr.p.readValue(TypeRegex)
	if err != nil {
		return "", "", err
	}

	pattern, options, err = v.parseRegex()

	ejvr.pop()
	return pattern, options, err
}

func (ejvr *extJSONValueReader) ReadString() (string, error) {
	if err := ejvr.ensureElementValue(TypeString, 0, "ReadString"); err != nil {
		return "", err
	}

	v, err := ejvr.p.readValue(TypeString)
	if err != nil {
		return "", err
	}

	if v.t != TypeString {
		return "", fmt.Errorf("expected type string but got type %s", v.t)
	}

	ejvr.pop()
	return v.v.(string), nil
}

func (ejvr *extJSONValueReader) ReadSymbol() (symbol string, err error) {
	if err = ejvr.ensureElementValue(TypeSymbol, 0, "ReadSymbol"); err != nil {
		return "", err
	}

	v, err := ejvr.p.readValue(TypeSymbol)
	if err != nil {
		return "", err
	}

	symbol, err = v.parseSymbol()

	ejvr.pop()
	return symbol, err
}

func (ejvr *extJSONValueReader) ReadTimestamp() (t uint32, i uint32, err error) {
	if err = ejvr.ensureElementValue(TypeTimestamp, 0, "ReadTimestamp"); err != nil {
		return 0, 0, err
	}

	v, err := ejvr.p.readValue(TypeTimestamp)
	if err != nil {
		return 0, 0, err
	}

	t, i, err = v.parseTimestamp()

	ejvr.pop()
	return t, i, err
}

func (ejvr *extJSONValueReader) ReadUndefined() error {
	if err := ejvr.ensureElementValue(TypeUndefined, 0, "ReadUndefined"); err != nil {
		return err
	}

	v, err := ejvr.p.readValue(TypeUndefined)
	if err != nil {
		return err
	}

	err = v.parseUndefined()

	ejvr.pop()
	return err
}

func (ejvr *extJSONValueReader) ReadElement() (string, ValueReader, error) {
	switch ejvr.stack[ejvr.frame].mode {
	case mTopLevel, mDocument, mCodeWithScope:
	default:
		return "", nil, ejvr.invalidTransitionErr(mElement, "ReadElement", []mode{mTopLevel, mDocument, mCodeWithScope})
	}

	name, t, err := ejvr.p.readKey()
	if err != nil {
		if errors.Is(err, ErrEOD) {
			if ejvr.stack[ejvr.frame].mode == mCodeWithScope {
				_, err := ejvr.p.peekType()
				if err != nil {
					return "", nil, err
				}
			}

			ejvr.pop()
		}

		return "", nil, err
	}

	ejvr.push(mElement, t)
	return name, ejvr, nil
}

func (ejvr *extJSONValueReader) ReadValue() (ValueReader, error) {
	switch ejvr.stack[ejvr.frame].mode {
	case mArray:
	default:
		return nil, ejvr.invalidTransitionErr(mValue, "ReadValue", []mode{mArray})
	}

	t, err := ejvr.p.peekType()
	if err != nil {
		if errors.Is(err, ErrEOA) {
			ejvr.pop()
		}

		return nil, err
	}

	ejvr.push(mValue, t)
	return ejvr, nil
}
