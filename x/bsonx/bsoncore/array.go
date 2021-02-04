// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncore

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
)

// Array is a raw bytes representation of a BSON array.
type Array []byte

// NewArrayFromReader reads an array from r. This function will only validate the length is
// correct and that the array ends with a null byte.
func NewArrayFromReader(r io.Reader) (Array, error) {
	if r == nil {
		return nil, ErrNilReader
	}

	var lengthBytes [4]byte

	// ReadFull guarantees that we will have read at least len(lengthBytes) if err == nil
	_, err := io.ReadFull(r, lengthBytes[:])
	if err != nil {
		return nil, err
	}

	length, _, _ := readi32(lengthBytes[:]) // ignore ok since we always have enough bytes to read a length
	if length < 0 {
		return nil, ErrInvalidLength
	}
	array := make([]byte, length)

	copy(array, lengthBytes[:])

	_, err = io.ReadFull(r, array[4:])
	if err != nil {
		return nil, err
	}

	if array[length-1] != 0x00 {
		return nil, ErrMissingNull
	}

	return array, nil
}

// Index searches for and retrieves the element at the given index. This method will panic if
// the array is invalid or if the index is out of bounds.
func (a Array) Index(index uint) Element {
	elem, err := a.IndexErr(index)
	if err != nil {
		panic(err)
	}
	return elem
}

// IndexErr searches for and retrieves the element at the given index.
func (a Array) IndexErr(index uint) (Element, error) {
	length, rem, ok := ReadLength(a)
	if !ok {
		return nil, NewInsufficientBytesError(a, rem)
	}

	length -= 4

	var current uint
	var elem Element
	for length > 1 {
		elem, rem, ok = ReadElement(rem)
		length -= int32(len(elem))
		if !ok {
			return nil, NewInsufficientBytesError(a, rem)
		}
		if current != index {
			current++
			continue
		}
		return elem, nil
	}
	return nil, ErrOutOfBounds
}

// DebugString outputs a human readable version of Array. It will attempt to stringify the
// valid components of the array even if the entire array is not valid.
func (a Array) DebugString() string {
	if len(a) < 5 {
		return "<malformed>"
	}
	var buf bytes.Buffer
	buf.WriteString("Array")
	length, rem, _ := ReadLength(a) // We know we have enough bytes to read the length
	buf.WriteByte('[')
	buf.WriteString(strconv.Itoa(int(length)))
	length -= 4
	buf.WriteString(")[")
	var elem Element
	var ok bool
	for length > 1 {
		elem, rem, ok = ReadElement(rem)
		length -= int32(len(elem))
		if !ok {
			buf.WriteString(fmt.Sprintf("<malformed (%d)>", length))
			break
		}
		fmt.Fprintf(&buf, "%s ", elem.DebugString())
	}
	buf.WriteByte(']')

	return buf.String()
}

// String outputs an ExtendedJSON version of Array. If the Array is not valid, this method
// returns an empty string.
func (a Array) String() string {
	if len(a) < 5 {
		return ""
	}
	var buf bytes.Buffer
	buf.WriteByte('[')

	length, rem, _ := ReadLength(a) // We know we have enough bytes to read the length

	length -= 4

	var elem Element
	var ok bool
	first := true
	for length > 1 {
		if !first {
			buf.WriteByte(',')
		}
		elem, rem, ok = ReadElement(rem)
		length -= int32(len(elem))
		if !ok {
			return ""
		}
		fmt.Fprintf(&buf, "%s", elem.String())
		first = false
	}
	buf.WriteByte(']')

	return buf.String()
}

// Elements returns this array as a slice of elements. The returned slice will contain valid
// elements. If the array is not valid, the elements up to the invalid point will be returned
// along with an error.
func (a Array) Elements() ([]Element, error) {
	length, rem, ok := ReadLength(a)
	if !ok {
		return nil, NewInsufficientBytesError(a, rem)
	}

	length -= 4

	var elem Element
	var elems []Element
	for length > 1 {
		elem, rem, ok = ReadElement(rem)
		length -= int32(len(elem))
		if !ok {
			return elems, NewInsufficientBytesError(a, rem)
		}
		if err := elem.Validate(); err != nil {
			return elems, err
		}
		elems = append(elems, elem)
	}
	return elems, nil
}

// Values returns this array as a slice of values. The returned slice will contain valid values.
// If the array is not valid, the values up to the invalid point will be returned along with an
// error.
func (a Array) Values() ([]Value, error) {
	length, rem, ok := ReadLength(a)
	if !ok {
		return nil, NewInsufficientBytesError(a, rem)
	}

	length -= 4

	var elem Element
	var vals []Value
	for length > 1 {
		elem, rem, ok = ReadElement(rem)
		length -= int32(len(elem))
		if !ok {
			return vals, NewInsufficientBytesError(a, rem)
		}
		if err := elem.Value().Validate(); err != nil {
			return vals, err
		}
		vals = append(vals, elem.Value())
	}
	return vals, nil
}

// Validate validates the array and ensures the elements contained within are valid.
func (a Array) Validate() error {
	length, rem, ok := ReadLength(a)
	if !ok {
		return NewInsufficientBytesError(a, rem)
	}
	if int(length) > len(a) {
		return a.lengtherror(int(length), len(a))
	}
	if a[length-1] != 0x00 {
		return ErrMissingNull
	}

	length -= 4
	var elem Element

	for length > 1 {
		elem, rem, ok = ReadElement(rem)
		length -= int32(len(elem))
		if !ok {
			return NewInsufficientBytesError(a, rem)
		}
		err := elem.Validate()
		if err != nil {
			return err
		}
	}

	if len(rem) < 1 || rem[0] != 0x00 {
		return ErrMissingNull
	}
	return nil
}

func (Array) lengtherror(length, rem int) error {
	return DocumentValidationError(fmt.Sprintf("array length exceeds available bytes. length=%d remainingBytes=%d", length, rem))
}
