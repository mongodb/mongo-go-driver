// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"io"

	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

// RawArray is a raw bytes representation of a BSON array.
type RawArray []byte

// ReadArray reads a BSON array from the io.Reader and returns it as a
// bson.RawArray.
func ReadArray(r io.Reader) (RawArray, error) {
	doc, err := bsoncore.NewArrayFromReader(r)

	return RawArray(doc), err
}

// Index searches for and retrieves the value at the given index. This method
// will panic if the array is invalid or if the index is out of bounds.
func (a RawArray) Index(index uint) RawValue {
	return convertFromCoreValue(bsoncore.Array(a).Index(index))
}

// IndexErr searches for and retrieves the value at the given index.
func (a RawArray) IndexErr(index uint) (RawValue, error) {
	elem, err := bsoncore.Array(a).IndexErr(index)

	return convertFromCoreValue(elem), err
}

// DebugString outputs a human readable version of Array. It will attempt to
// stringify the valid components of the array even if the entire array is not
// valid.
func (a RawArray) DebugString() string {
	return bsoncore.Array(a).DebugString()
}

// String outputs an ExtendedJSON version of Array. If the Array is not valid,
// this method returns an empty string.
func (a RawArray) String() string {
	return bsoncore.Array(a).String()
}

// Values returns this array as a slice of values. The returned slice will
// contain valid values. If the array is not valid, the values up to the invalid
// point will be returned along with an error.
func (a RawArray) Values() ([]RawValue, error) {
	vals, err := bsoncore.Array(a).Values()
	if err != nil {
		return nil, err
	}

	rvals := make([]RawValue, 0, len(vals))
	for _, val := range vals {
		rvals = append(rvals, convertFromCoreValue(val))
	}

	return rvals, err
}

// Validate validates the array and ensures the elements contained within are
// valid.
func (a RawArray) Validate() error {
	return bsoncore.Array(a).Validate()
}
