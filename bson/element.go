// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/go-stack/stack"
	"github.com/mongodb/mongo-go-driver/bson/elements"
)

const validateMaxDepthDefault = 2048

// ErrTooSmall indicates that a slice provided to write into is not large enough to fit the data.
type ErrTooSmall struct {
	Stack stack.CallStack
}

// NewErrTooSmall creates a new ErrTooSmall with the given message and the current stack.
func NewErrTooSmall() ErrTooSmall {
	return ErrTooSmall{Stack: stack.Trace().TrimRuntime()}
}

// Error implements the error interface.
func (e ErrTooSmall) Error() string {
	return "too small"
}

// ErrorStack returns a string representing the stack at the point where the error occurred.
func (e ErrTooSmall) ErrorStack() string {
	s := bytes.NewBufferString("too small: [")

	for i, call := range e.Stack {
		if i != 0 {
			s.WriteString(", ")
		}

		// go vet doesn't like %k even though it's part of stack's API, so we move the format
		// string so it doesn't complain. (We also can't make it a constant, or go vet still
		// complains.)
		callFormat := "%k.%n %v"

		s.WriteString(fmt.Sprintf(callFormat, call, call, call))
	}

	s.WriteRune(']')

	return s.String()
}

// Equals checks that err2 also is an ErrTooSmall.
func (e ErrTooSmall) Equals(err2 error) bool {
	switch err2.(type) {
	case ErrTooSmall:
		return true
	default:
		return false
	}
}

// ErrUninitializedElement is returned whenever any method is invoked on an uninitialized Element.
var ErrUninitializedElement = errors.New("bson/ast/compact: Method call on uninitialized Element")

// ErrInvalidWriter indicates that a type that can't be written into was passed to a writer method.
var ErrInvalidWriter = errors.New("bson: invalid writer provided")

// ErrInvalidString indicates that a BSON string value had an incorrect length.
var ErrInvalidString = errors.New("invalid string value")

// ErrInvalidBinarySubtype indicates that a BSON binary value had an undefined subtype.
var ErrInvalidBinarySubtype = errors.New("invalid BSON binary Subtype")

// ErrInvalidBooleanType indicates that a BSON boolean value had an incorrect byte.
var ErrInvalidBooleanType = errors.New("invalid value for BSON Boolean Type")

// ErrStringLargerThanContainer indicates that the code portion of a BSON JavaScript code with scope
// value is larger than the specified length of the entire value.
var ErrStringLargerThanContainer = errors.New("string size is larger than the JavaScript code with scope container")

// ErrInvalidElement indicates that a bson.Element had invalid underlying BSON.
var ErrInvalidElement = errors.New("invalid Element")

// ElementTypeError specifies that a method to obtain a BSON value an incorrect type was called on a bson.Value.
type ElementTypeError struct {
	Method string
	Type   Type
}

// Error implements the error interface.
func (ete ElementTypeError) Error() string {
	return "Call of " + ete.Method + " on " + ete.Type.String() + " type"
}

// Element represents a BSON element, i.e. key-value pair of a BSON document.
type Element struct {
	value *Value
}

func newElement(start uint32, offset uint32) *Element {
	return &Element{&Value{start: start, offset: offset}}
}

// Clone creates a shallow copy of the element/
func (e *Element) Clone() *Element {
	return &Element{
		value: &Value{
			start:  e.value.start,
			offset: e.value.offset,
			data:   e.value.data,
			d:      e.value.d,
		},
	}
}

// Value returns the value associated with the BSON element.
func (e *Element) Value() *Value {
	return e.value
}

// Validate validates the element and returns its total size.
func (e *Element) Validate() (uint32, error) {
	if e == nil {
		return 0, ErrNilElement
	}
	if e.value == nil {
		return 0, ErrUninitializedElement
	}

	var total uint32 = 1
	n, err := e.validateKey()
	total += n
	if err != nil {
		return total, err
	}
	n, err = e.value.validate(false)
	total += n
	if err != nil {
		return total, err
	}
	return total, nil
}

// validate is a common validation method for elements.
//
// TODO(skriptble): Fill out this method and ensure all validation routines
// pass through this method.
func (e *Element) validate(recursive bool, currentDepth, maxDepth uint32) (uint32, error) {
	return 0, nil
}

func (e *Element) validateKey() (uint32, error) {
	if e.value.data == nil {
		return 0, ErrUninitializedElement
	}

	pos, end := e.value.start+1, e.value.offset
	var total uint32
	if end > uint32(len(e.value.data)) {
		end = uint32(len(e.value.data))
	}
	for ; pos < end && e.value.data[pos] != '\x00'; pos++ {
		total++
	}
	if pos == end || e.value.data[pos] != '\x00' {
		return total, ErrInvalidKey
	}
	total++
	return total, nil
}

// Key returns the key for this element.
// It panics if e is uninitialized.
func (e *Element) Key() string {
	if e == nil || e.value == nil || e.value.offset == 0 || e.value.data == nil {
		panic(ErrUninitializedElement)
	}
	return string(e.value.data[e.value.start+1 : e.value.offset-1])
}

// WriteTo implements the io.WriterTo interface.
func (e *Element) WriteTo(w io.Writer) (int64, error) {
	return 0, nil
}

// WriteElement serializes this element to the provided writer starting at the
// provided start position.
func (e *Element) WriteElement(start uint, writer interface{}) (int64, error) {
	return e.writeElement(true, start, writer)
}

func (e *Element) writeElement(key bool, start uint, writer interface{}) (int64, error) {
	// TODO(skriptble): Figure out if we want to use uint or uint32 and
	// standardize across all packages.
	var total int64
	size, err := e.Validate()
	if err != nil {
		return 0, err
	}
	switch w := writer.(type) {
	case []byte:
		n, err := e.writeByteSlice(key, start, size, w)
		if err != nil {
			return 0, NewErrTooSmall()
		}
		total += int64(n)
	default:
		return 0, ErrInvalidWriter
	}
	return total, nil
}

// writeByteSlice handles writing this element to a slice of bytes.
func (e *Element) writeByteSlice(key bool, start uint, size uint32, b []byte) (int64, error) {
	var startToWrite uint
	needed := start + uint(size)

	if key {
		startToWrite = uint(e.value.start)
	} else {
		startToWrite = uint(e.value.offset)

		// Fewer bytes are needed if the key isn't being written.
		needed -= uint(e.value.offset) - uint(e.value.start) - 1
	}

	if uint(len(b)) < needed {
		return 0, NewErrTooSmall()
	}

	var n int
	switch e.value.data[e.value.start] {
	case '\x03':
		if e.value.d == nil {
			n = copy(b[start:], e.value.data[startToWrite:e.value.start+size])
			break
		}

		header := e.value.offset - e.value.start
		size -= header
		if key {
			n += copy(b[start:], e.value.data[startToWrite:e.value.offset])
			start += uint(n)
		}

		nn, err := e.value.d.writeByteSlice(start, size, b)
		n += int(nn)
		if err != nil {
			return int64(n), err
		}
	case '\x04':
		if e.value.d == nil {
			n = copy(b[start:], e.value.data[startToWrite:e.value.start+size])
			break
		}

		header := e.value.offset - e.value.start
		size -= header
		if key {
			n += copy(b[start:], e.value.data[startToWrite:e.value.offset])
			start += uint(n)
		}

		arr := &Array{doc: e.value.d}

		nn, err := arr.writeByteSlice(start, size, b)
		n += int(nn)
		if err != nil {
			return int64(n), err
		}
	case '\x0F':
		// Get length of code
		codeStart := e.value.offset + 4
		codeLength := readi32(e.value.data[codeStart : codeStart+4])

		if e.value.d != nil {
			lengthWithoutScope := 4 + 4 + codeLength

			scopeLength, err := e.value.d.Validate()
			if err != nil {
				return 0, err
			}

			codeWithScopeLength := lengthWithoutScope + int32(scopeLength)
			_, err = elements.Int32.Encode(uint(e.value.offset), e.value.data, codeWithScopeLength)
			if err != nil {
				return int64(n), err
			}

			typeAndKeyLength := e.value.offset - e.value.start
			n += copy(
				b[start:start+uint(typeAndKeyLength)+uint(lengthWithoutScope)],
				e.value.data[e.value.start:e.value.start+typeAndKeyLength+uint32(lengthWithoutScope)])
			start += uint(n)

			nn, err := e.value.d.writeByteSlice(start, scopeLength, b)
			n += int(nn)
			if err != nil {
				return int64(n), err
			}

			break
		}

		// Get length of scope
		scopeStart := codeStart + 4 + uint32(codeLength)
		scopeLength := readi32(e.value.data[scopeStart : scopeStart+4])

		// Calculate end of entire CodeWithScope value
		codeWithScopeEnd := int32(scopeStart) + scopeLength

		// Set the length of the value
		codeWithScopeLength := codeWithScopeEnd - int32(e.value.offset)
		_, err := elements.Int32.Encode(uint(e.value.offset), e.value.data, codeWithScopeLength)
		if err != nil {
			return 0, err
		}

		fallthrough
	default:
		n = copy(b[start:], e.value.data[startToWrite:e.value.start+size])
	}

	return int64(n), nil
}

// MarshalBSON implements the Marshaler interface.
func (e *Element) MarshalBSON() ([]byte, error) {
	size, err := e.Validate()
	if err != nil {
		return nil, err
	}
	b := make([]byte, size)
	_, err = e.writeByteSlice(true, 0, size, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// String implements the fmt.Stringer interface.
func (e *Element) String() string {
	val := e.Value().Interface()
	if s, ok := val.(string); ok && e.Value().Type() == TypeString {
		val = strconv.Quote(s)
	}
	return fmt.Sprintf(`bson.Element{[%s]"%s": %v}`, e.Value().Type(), e.Key(), val)
}

func (e *Element) equal(e2 *Element) bool {
	if e == nil && e2 == nil {
		return true
	}
	if e == nil || e2 == nil {
		return false
	}

	return e.value.equal(e2.value)
}

func elemsFromValues(values []*Value) []*Element {
	elems := make([]*Element, len(values))

	for i, v := range values {
		if v == nil {
			elems[i] = nil
		} else {
			elems[i] = &Element{v}
		}
	}

	return elems
}

func convertValueToElem(key string, v *Value) *Element {
	if v == nil || v.offset == 0 || v.data == nil {
		return nil
	}

	keyLen := len(key)
	// We add the length of the data so when we compare values
	// we don't have extra space at the end of the data property.
	d := make([]byte, 2+len(key)+len(v.data[v.offset:]))

	d[0] = v.data[v.start]
	copy(d[1:keyLen+1], key)
	d[keyLen+1] = 0x00
	copy(d[keyLen+2:], v.data[v.offset:])

	elem := newElement(0, uint32(keyLen+2))
	elem.value.data = d
	elem.value.d = v.d

	return elem
}
