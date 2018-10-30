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
	"strconv"

	"github.com/mongodb/mongo-go-driver/bson/bsoncore"
	"github.com/mongodb/mongo-go-driver/bson/bsontype"
)

// ErrNilArray indicates that an operation was attempted on a nil *Array.
var ErrNilArray = errors.New("array is nil")

// Arr represents an array in BSON.
type Arr []Val

// lookupTraverse searches this array for a key and possibly recursively searches the value.
//
// NOTE: This method should only return KeyNotFound or nil as the error. If this changes, update the
// errors handled by Document.LookupElementErr.
func (a Arr) lookupTraverse(key ...string) (Elem, error) {
	if len(key) == 0 {
		return Elem{}, KeyNotFound{Key: key}
	}

	index, err := strconv.ParseUint(key[1], 10, 0)
	if err != nil {
		return Elem{}, KeyNotFound{Key: key}
	}

	if index >= uint64(len(a)) {
		return Elem{}, KeyNotFound{}
	}
	val := a[index]

	if len(key) == 1 {
		return Elem{Key: key[0], Value: val}, nil
	}

	var elem Elem
	switch val.Type() {
	case bsontype.EmbeddedDocument:
		elem, err = val.Document().LookupElementErr(key[1:]...)
	case bsontype.Array:
		elem, err = val.Array().lookupTraverse(key[1:]...)
	default:
		err = KeyNotFound{Type: elem.Value.Type()}
	}

	switch tt := err.(type) {
	case KeyNotFound:
		tt.Depth++
		return Elem{}, tt
	case nil:
		return elem, nil
	default:
		return Elem{}, err // This should never happen.
	}
}

// String implements the fmt.Stringer interface.
func (a Arr) String() string {
	var buf bytes.Buffer
	buf.Write([]byte("bson.Array["))
	for idx, val := range a {
		if idx > 0 {
			buf.Write([]byte(", "))
		}
		fmt.Fprintf(&buf, "%s", val)
	}
	buf.WriteByte(']')

	return buf.String()
}

// MarshalBSONValue implements the bsoncodec.ValueMarshaler interface.
func (a Arr) MarshalBSONValue() (bsontype.Type, []byte, error) {
	if a == nil {
		// TODO: Should we do this?
		return bsontype.Null, nil, nil
	}

	idx, dst := bsoncore.ReserveLength(nil)
	for idx, value := range a {
		t, data, _ := value.MarshalBSONValue() // marshalBSONValue never returns an error.
		dst = append(dst, byte(t))
		dst = append(dst, strconv.Itoa(idx)...)
		dst = append(dst, 0x00)
		dst = append(dst, data...)
	}
	dst = append(dst, 0x00)
	dst = bsoncore.UpdateLength(dst, idx, int32(len(dst[idx:])))
	return bsontype.Array, dst, nil
}

// UnmarshalBSONValue implements the bsoncodec.ValueUnmarshaler interface.
func (a *Arr) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
	if a == nil {
		return ErrNilArray
	}
	*a = (*a)[:0]

	elements, err := Raw(data).Elements()
	if err != nil {
		return err
	}

	for _, elem := range elements {
		var val Val
		rawval := elem.Value()
		err = val.UnmarshalBSONValue(rawval.Type, rawval.Value)
		if err != nil {
			return err
		}
		*a = append(*a, val)
	}
	return nil
}

// Equal compares this document to another, returning true if they are equal.
func (a Arr) Equal(a2 Arr) bool {
	if len(a) != len(a2) {
		return false
	}
	for idx := range a {
		if !a[idx].Equal(a2[idx]) {
			return false
		}
	}
	return true
}

func (Arr) idoc() {}
