// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncore

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// NewArrayLengthError creates and returns an error for when the length of an array exceeds the
// bytes available.
func NewArrayLengthError(length, rem int) error {
	return lengthError("array", length, rem)
}

// Array is a raw bytes representation of a BSON array.
type Array []byte

// NewArrayFromReader reads an array from r. This function will only validate the length is
// correct and that the array ends with a null byte.
func NewArrayFromReader(r io.Reader) (Array, error) {
	return newBufferFromReader(r)
}

// Index searches for and retrieves the value at the given index. This method will panic if
// the array is invalid or if the index is out of bounds.
func (a Array) Index(index uint) Value {
	value, err := a.IndexErr(index)
	if err != nil {
		panic(err)
	}
	return value
}

// IndexErr searches for and retrieves the value at the given index.
func (a Array) IndexErr(index uint) (Value, error) {
	elem, err := indexErr(a, index)
	if err != nil {
		return Value{}, err
	}
	return elem.Value(), err
}

// DebugString outputs a human readable version of Array. It will attempt to stringify the
// valid components of the array even if the entire array is not valid.
func (a Array) DebugString() string {
	if len(a) < 5 {
		return "<malformed>"
	}
	var buf strings.Builder
	buf.WriteString("Array")
	length, rem, _ := ReadLength(a) // We know we have enough bytes to read the length
	buf.WriteByte('(')
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
		buf.WriteString(elem.Value().DebugString())
		if length != 1 {
			buf.WriteByte(',')
		}
	}
	buf.WriteByte(']')

	return buf.String()
}

// String outputs an ExtendedJSON version of Array. If the Array is not valid, this method
// returns an empty string.
func (a Array) String() string {
	str, _ := a.StringN(-1)
	return str
}

// StringN stringifies an array. If N is non-negative, it will truncate the string to N bytes.
// Otherwise, it will return the full string representation. The second return value indicates
// whether the string was truncated or not.
func (a Array) StringN(n int) (string, bool) {
	length, rem, ok := ReadLength(a)
	if !ok || length < 5 {
		return "", false
	}
	length -= 4 // length bytes
	length--    // final null byte

	if n == 0 {
		return "", true
	}

	var buf strings.Builder
	buf.WriteByte('[')

	var truncated bool
	var elem Element
	var str string
	first := true
	for length > 0 && !truncated {
		needStrLen := -1
		// Set needStrLen if n is positive, meaning we want to limit the string length.
		if n > 0 {
			// Stop stringifying if we reach the limit, that also ensures needStrLen is
			// greater than 0 if we need to limit the length.
			if buf.Len() >= n {
				truncated = true
				break
			}
			needStrLen = n - buf.Len()
		}

		// Append a comma if this is not the first element.
		if !first {
			buf.WriteByte(',')
			// If we are truncating, we need to account for the comma in the length.
			if needStrLen > 0 {
				needStrLen--
				if needStrLen == 0 {
					truncated = true
					break
				}
			}
		}

		elem, rem, ok = ReadElement(rem)
		length -= int32(len(elem))
		// Exit on malformed element.
		if !ok || length < 0 {
			return "", false
		}

		// Delegate to StringN() on the element.
		str, truncated = elem.Value().StringN(needStrLen)
		buf.WriteString(str)

		first = false
	}

	if n <= 0 || (buf.Len() < n && !truncated) {
		buf.WriteByte(']')
	} else {
		truncated = true
	}

	return buf.String(), truncated
}

// Values returns this array as a slice of values. The returned slice will contain valid values.
// If the array is not valid, the values up to the invalid point will be returned along with an
// error.
func (a Array) Values() ([]Value, error) {
	return values(a)
}

// Validate validates the array and ensures the elements contained within are valid.
func (a Array) Validate() error {
	length, rem, ok := ReadLength(a)
	if !ok {
		return NewInsufficientBytesError(a, rem)
	}
	if int(length) > len(a) {
		return NewArrayLengthError(int(length), len(a))
	}
	if a[length-1] != 0x00 {
		return ErrMissingNull
	}

	length -= 4
	var elem Element

	var keyNum int64
	for length > 1 {
		elem, rem, ok = ReadElement(rem)
		length -= int32(len(elem))
		if !ok {
			return NewInsufficientBytesError(a, rem)
		}

		// validate element
		err := elem.Validate()
		if err != nil {
			return err
		}

		// validate keys increase numerically
		if fmt.Sprint(keyNum) != elem.Key() {
			return fmt.Errorf("array key %q is out of order or invalid", elem.Key())
		}
		keyNum++
	}

	if len(rem) < 1 || rem[0] != 0x00 {
		return ErrMissingNull
	}
	return nil
}
