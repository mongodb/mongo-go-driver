// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"fmt"
	"io"
	"strconv"

	"github.com/mongodb/mongo-go-driver/bson/elements"
)

// Array represents an array in BSON. The methods of this type are more
// expensive than those on Document because they require potentially updating
// multiple keys to ensure the array stays valid at all times.
type Array struct {
	doc *Document
}

// NewArray creates a new array with the specified value.
func NewArray(values ...*Value) *Array {
	elems := make([]*Element, 0, len(values))
	for _, v := range values {
		elems = append(elems, &Element{value: v})
	}

	return &Array{doc: NewDocument(elems...)}
}

// ArrayFromDocument creates an array from a *Document. The returned array
// does not make a copy of the *Document, so any changes made to either will
// be present in both.
func ArrayFromDocument(doc *Document) *Array {
	return &Array{doc: doc}
}

// Len returns the number of elements in the array.
func (a *Array) Len() int {
	return len(a.doc.elems)
}

// Reset clears all elements from the array.
func (a *Array) Reset() {
	a.doc.Reset()
}

// Validate ensures that the array's underlying BSON is valid. It returns the the number of bytes
// in the underlying BSON if it is valid or an error if it isn't.
func (a *Array) Validate() (uint32, error) {
	var size uint32 = 4 + 1
	for i, elem := range a.doc.elems {
		n, err := elem.value.validate(false)
		if err != nil {
			return 0, err
		}

		// type
		size++
		// key
		size += uint32(len(strconv.Itoa(i))) + 1
		// value
		size += n
	}

	return size, nil
}

// Lookup returns the value in the array at the given index or an error if it cannot be found.
func (a *Array) Lookup(index uint) (*Value, error) {
	v, ok := a.doc.ElementAtOK(index)
	if !ok {
		return nil, ErrOutOfBounds
	}

	return v.value, nil
}

func (a *Array) lookupTraverse(index uint, keys ...string) (*Value, error) {
	value, err := a.Lookup(index)
	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return value, nil
	}

	switch value.Type() {
	case TypeEmbeddedDocument:
		element, err := value.MutableDocument().LookupElementErr(keys...)
		if err != nil {
			return nil, err
		}

		return element.Value(), nil
	case TypeArray:
		index, err := strconv.ParseUint(keys[0], 10, 0)
		if err != nil {
			return nil, ErrInvalidArrayKey
		}

		val, err := value.MutableArray().lookupTraverse(uint(index), keys[1:]...)
		if err != nil {
			return nil, err
		}

		return val, nil
	default:
		return nil, ErrInvalidDepthTraversal
	}
}

// Append adds the given values to the end of the array. It returns a reference to itself.
func (a *Array) Append(values ...*Value) *Array {
	a.doc.Append(elemsFromValues(values)...)

	return a
}

// Prepend adds the given values to the beginning of the array. It returns a reference to itself.
func (a *Array) Prepend(values ...*Value) *Array {
	a.doc.Prepend(elemsFromValues(values)...)

	return a
}

// Set replaces the value at the given index with the parameter value. It panics if the index is
// out of bounds.
func (a *Array) Set(index uint, value *Value) *Array {
	if index >= uint(len(a.doc.elems)) {
		panic(ErrOutOfBounds)
	}

	a.doc.elems[index] = &Element{value}

	return a
}

// Concat will append all the values from each of the arguments onto the array.
//
// Each argument must be one of the following:
//
//   - *Array
//   - *Document
//   - []byte
//   - bson.Reader
//
// Note that in the case of *Document, []byte, and bson.Reader, the keys will be ignored and only
// the values will be appended.
func (a *Array) Concat(docs ...interface{}) error {
	for _, arr := range docs {
		if arr == nil {
			if a.doc.IgnoreNilInsert {
				continue
			}

			return ErrNilDocument
		}

		switch val := arr.(type) {
		case *Array:
			if val == nil {
				if a.doc.IgnoreNilInsert {
					continue
				}

				return ErrNilDocument
			}

			for _, e := range val.doc.elems {
				a.Append(e.value)
			}
		case *Document:
			if val == nil {
				if a.doc.IgnoreNilInsert {
					continue
				}

				return ErrNilDocument
			}

			for _, e := range val.elems {
				a.Append(e.value)
			}
		case []byte:
			if err := a.concatReader(Reader(val)); err != nil {
				return err
			}
		case Reader:
			if err := a.concatReader(val); err != nil {
				return err
			}
		default:
			return ErrInvalidDocumentType
		}
	}

	return nil
}

func (a *Array) concatReader(r Reader) error {
	_, err := r.readElements(func(e *Element) error {
		a.Append(e.value)

		return nil
	})

	return err
}

// Delete removes the value at the given index from the array.
func (a *Array) Delete(index uint) *Value {
	if index >= uint(len(a.doc.elems)) {
		return nil
	}

	elem := a.doc.elems[index]
	a.doc.elems = append(a.doc.elems[:index], a.doc.elems[index+1:]...)

	return elem.value
}

// WriteTo implements the io.WriterTo interface.
func (a *Array) WriteTo(w io.Writer) (int64, error) {
	b, err := a.MarshalBSON()
	if err != nil {
		return 0, err
	}
	n, err := w.Write(b)
	return int64(n), err
}

// String implements the fmt.Stringer interface.
func (a *Array) String() string {
	var buf bytes.Buffer
	buf.Write([]byte("bson.Array["))
	for idx, elem := range a.doc.elems {
		if idx > 0 {
			buf.Write([]byte(", "))
		}
		fmt.Fprintf(&buf, "%s", elem)
	}
	buf.WriteByte(']')

	return buf.String()
}

// WriteArray will serialize this array to the provided writer beginning
// at the provided start position.
func (a *Array) WriteArray(start uint, writer []byte) (int64, error) {
	var total int64
	var pos = start

	size, err := a.Validate()
	if err != nil {
		return total, err
	}

	n, err := a.writeByteSlice(pos, size, writer)
	total += n
	pos += uint(n)
	if err != nil {
		return total, err
	}

	return total, nil
}

// writeByteSlice handles serializing this array to a slice of bytes starting
// at the given start position.
func (a *Array) writeByteSlice(start uint, size uint32, b []byte) (int64, error) {
	var total int64
	var pos = start

	if len(b) < int(start)+int(size) {
		return 0, NewErrTooSmall()
	}
	n, err := elements.Int32.Encode(start, b, int32(size))
	total += int64(n)
	pos += uint(n)
	if err != nil {
		return total, err
	}

	for i, elem := range a.doc.elems {
		b[pos] = elem.value.data[elem.value.start]
		total++
		pos++

		key := []byte(strconv.Itoa(i))
		key = append(key, 0)
		copy(b[pos:], key)
		total += int64(len(key))
		pos += uint(len(key))

		n, err := elem.writeElement(false, pos, b)
		total += int64(n)
		pos += uint(n)
		if err != nil {
			return total, err
		}
	}

	n, err = elements.Byte.Encode(pos, b, '\x00')
	total += int64(n)
	pos += uint(n)
	if err != nil {
		return total, err
	}
	return total, nil
}

// MarshalBSON implements the Marshaler interface.
func (a *Array) MarshalBSON() ([]byte, error) {
	size, err := a.Validate()
	if err != nil {
		return nil, err
	}
	b := make([]byte, size)
	_, err = a.writeByteSlice(0, size, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Iterator returns a ArrayIterator that can be used to iterate through the
// elements of this Array.
func (a *Array) Iterator() (*ArrayIterator, error) {
	return NewArrayIterator(a)
}
