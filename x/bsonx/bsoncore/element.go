// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncore

import (
	"bytes"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/internal/bsoncoreutil"
)

// MalformedElementError represents a class of errors that RawElement methods return.
type MalformedElementError string

func (mee MalformedElementError) Error() string { return string(mee) }

// ErrElementMissingKey is returned when a RawElement is missing a key.
const ErrElementMissingKey MalformedElementError = "element is missing key"

// ErrElementMissingType is returned when a RawElement is missing a type.
const ErrElementMissingType MalformedElementError = "element is missing type"

// Element is a raw bytes representation of a BSON element.
type Element []byte

// Key returns the key for this element. If the element is not valid, this method returns an empty
// string. If knowing if the element is valid is important, use KeyErr.
func (e Element) Key() string {
	key, _ := e.KeyErr()
	return key
}

// KeyBytes returns the key for this element as a []byte. If the element is not valid, this method
// returns an empty string. If knowing if the element is valid is important, use KeyErr. This method
// will not include the null byte at the end of the key in the slice of bytes.
func (e Element) KeyBytes() []byte {
	key, _ := e.KeyBytesErr()
	return key
}

// KeyErr returns the key for this element, returning an error if the element is not valid.
func (e Element) KeyErr() (string, error) {
	key, err := e.KeyBytesErr()
	return string(key), err
}

// KeyBytesErr returns the key for this element as a []byte, returning an error if the element is
// not valid.
func (e Element) KeyBytesErr() ([]byte, error) {
	if len(e) == 0 {
		return nil, ErrElementMissingType
	}
	idx := bytes.IndexByte(e[1:], 0x00)
	if idx == -1 {
		return nil, ErrElementMissingKey
	}
	return e[1 : idx+1], nil
}

// Validate ensures the element is a valid BSON element.
func (e Element) Validate() error {
	if len(e) < 1 {
		return ErrElementMissingType
	}
	idx := bytes.IndexByte(e[1:], 0x00)
	if idx == -1 {
		return ErrElementMissingKey
	}
	return Value{Type: Type(e[0]), Data: e[idx+2:]}.Validate()
}

// CompareKey will compare this element's key to key. This method makes it easy to compare keys
// without needing to allocate a string. The key may be null terminated. If a valid key cannot be
// read this method will return false.
func (e Element) CompareKey(key []byte) bool {
	if len(e) < 2 {
		return false
	}
	idx := bytes.IndexByte(e[1:], 0x00)
	if idx == -1 {
		return false
	}
	if index := bytes.IndexByte(key, 0x00); index > -1 {
		key = key[:index]
	}
	return bytes.Equal(e[1:idx+1], key)
}

// Value returns the value of this element. If the element is not valid, this method returns an
// empty Value. If knowing if the element is valid is important, use ValueErr.
func (e Element) Value() Value {
	val, _ := e.ValueErr()
	return val
}

// ValueErr returns the value for this element, returning an error if the element is not valid.
func (e Element) ValueErr() (Value, error) {
	if len(e) == 0 {
		return Value{}, ErrElementMissingType
	}
	idx := bytes.IndexByte(e[1:], 0x00)
	if idx == -1 {
		return Value{}, ErrElementMissingKey
	}

	val, rem, exists := ReadValue(e[idx+2:], Type(e[0]))
	if !exists {
		return Value{}, NewInsufficientBytesError(e, rem)
	}
	return val, nil
}

// String implements the fmt.String interface. The output will be in extended JSON format.
func (e Element) String() string {
	str, _ := e.StringN(-1)
	return str
}

// StringN will return values in extended JSON format that will stringify an element upto N bytes.
// If N is non-negative, it will truncate the string to N bytes. Otherwise, it will return the full
// string representation. The second return value indicates whether the string was truncated or not.
// If the element is not valid, this returns an empty string
func (e Element) StringN(n int) (string, bool) {
	if len(e) == 0 {
		return "", false
	}
	if n == 0 {
		return "", true
	}
	if n == 1 {
		return `"`, true
	}

	t := Type(e[0])
	idx := bytes.IndexByte(e[1:], 0x00)
	if idx <= 0 {
		return "", false
	}
	key := e[1 : idx+1]

	var buf strings.Builder
	buf.WriteByte('"')
	const suffix = `": `
	switch {
	case n < 0 || idx <= n-buf.Len()-len(suffix):
		buf.Write(key)
		buf.WriteString(suffix)
	case idx < n:
		buf.Write(key)
		buf.WriteString(suffix[:n-idx-1])
		return buf.String(), true
	default:
		buf.WriteString(bsoncoreutil.Truncate(string(key), n-1))
		return buf.String(), true
	}

	needStrLen := -1
	// Set needStrLen if n is positive, meaning we want to limit the string length.
	if n > 0 {
		// Stop stringifying if we reach the limit, that also ensures needStrLen is
		// greater than 0 if we need to limit the length.
		if buf.Len() >= n {
			return buf.String(), true
		}
		needStrLen = n - buf.Len()
	}

	val, _, valid := ReadValue(e[idx+2:], t)
	if !valid {
		return "", false
	}

	var str string
	var truncated bool
	if _, ok := val.StringValueOK(); ok {
		str, truncated = val.StringN(needStrLen)
	} else if arr, ok := val.ArrayOK(); ok {
		str, truncated = arr.StringN(needStrLen)
	} else {
		str = val.String()
		if needStrLen > 0 && len(str) > needStrLen {
			truncated = true
			str = bsoncoreutil.Truncate(str, needStrLen)
		}
	}

	buf.WriteString(str)
	return buf.String(), truncated
}

// DebugString outputs a human readable version of RawElement. It will attempt to stringify the
// valid components of the element even if the entire element is not valid.
func (e Element) DebugString() string {
	if len(e) == 0 {
		return "<malformed>"
	}
	t := Type(e[0])
	idx := bytes.IndexByte(e[1:], 0x00)
	if idx == -1 {
		return fmt.Sprintf(`bson.Element{[%s]<malformed>}`, t)
	}
	key, valBytes := []byte(e[1:idx+1]), []byte(e[idx+2:])
	val, _, valid := ReadValue(valBytes, t)
	if !valid {
		return fmt.Sprintf(`bson.Element{[%s]"%s": <malformed>}`, t, key)
	}
	return fmt.Sprintf(`bson.Element{[%s]"%s": %v}`, t, key, val)
}
