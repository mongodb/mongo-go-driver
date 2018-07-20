// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import "bytes"

// Marshal converts a BSON type to bytes.
//
// The value can be any one of the following types:
//
//   - bson.Marshaler
//   - io.Reader
//   - []byte
//   - bson.Reader
//   - any map with string keys
//   - a struct (possibly with tags)
//
// In the case of a struct, the lowercased field name is used as the key for each exported
// field but this behavior may be changed using a struct tag. The tag may also contain flags to
// adjust the marshalling behavior for the field. The tag formats accepted are:
//
//     "[<key>][,<flag1>[,<flag2>]]"
//
//     `(...) bson:"[<key>][,<flag1>[,<flag2>]]" (...)`
//
// The following flags are currently supported:
//
//     omitempty  Only include the field if it's not set to the zero value for the type or to
//                empty slices or maps.
//
//     minsize    Marshal an integer of a type larger than 32 bits value as an int32, if that's
// 				  feasible while preserving the numeric value.
//
//     inline     Inline the field, which must be a struct or a map, causing all of its fields
//                or keys to be processed as if they were part of the outer struct. For maps,
//                keys must not conflict with the bson keys of other struct fields.
//
// An example:
//
//     type T struct {
//         A bool
//         B int    "myb"
//         C string "myc,omitempty"
//         D string `bson:",omitempty" json:"jsonkey"`
//         E int64  ",minsize"
//         F int64  "myf,omitempty,minsize"
//     }
func Marshal(value interface{}) ([]byte, error) {
	var out bytes.Buffer

	err := NewEncoder(&out).Encode(value)
	if err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

// Unmarshal converts bytes into a BSON type.
//
// The value can be any one of the following types:
//
//   - bson.Unmarshaler
//   - io.Writer
//   - []byte
//   - bson.Reader
//   - any map with string keys
//   - a struct (possibly with tags)
//
// In the case of struct values, only exported fields will be deserialized. The lowercased field
// name is used as the key for each exported field, but this behavior may be changed using a struct
// tag. The tag may also contain flags to adjust the unmarshaling behavior for the field. The tag
// formats accepted are:
//
//     "[<key>][,<flag1>[,<flag2>]]"
//
//     `(...) bson:"[<key>][,<flag1>[,<flag2>]]" (...)`
//
// The target field or element types of out may not necessarily match the BSON values of the
// provided data. The following conversions are made automatically:
//
//   - Numeric types are converted if at least the integer part of the value would be preserved
//     correctly
//
// If the value would not fit the type and cannot be converted, it is silently skipped.
//
// Pointer values are initialized when necessary.
func Unmarshal(in []byte, out interface{}) error {
	return NewDecoder(bytes.NewReader(in)).Decode(out)
}

// UnmarshalDocument converts bytes into a *bson.Document.
func UnmarshalDocument(bson []byte) (*Document, error) {
	return ReadDocument(bson)
}

// Marshal returns the BSON encoding of val.
//
// Marshal will use the default registry created by NewRegistry to recursively
// marshal val into a []byte. Marshal will inspect struct tags and alter the
// marshaling process accordingly.
func Marshalv2(val interface{}) ([]byte, error) {
	return MarshalWithRegistry(defaultRegistry, val)
}

// MarshalAppend will append the BSON encoding of val to dst. If dst is not
// large enough to hold the BSON encoding of val, dst will be grown.
func MarshalAppend(dst []byte, val interface{}) ([]byte, error) {
	return MarshalAppendWithRegistry(defaultRegistry, dst, val)
}

// MarshalWithRegistry returns the BSON encoding of val using Registry r.
func MarshalWithRegistry(r *Registry, val interface{}) ([]byte, error) {
	dst := make([]byte, 0, 1024) // TODO: make the default cap a constant
	return MarshalAppendWithRegistry(r, dst, val)
}

// MarshalAppendWithRegistry will append the BSON encoding of val to dst using
// Registry r. If dst is not large enough to hold the BSON encoding of val, dst
// will be grown.
func MarshalAppendWithRegistry(r *Registry, dst []byte, val interface{}) ([]byte, error) {
	w := writer(dst)
	vw := newValueWriter(&w)

	enc := encPool.Get().(*Encoderv2)
	defer encPool.Put(enc)

	enc.Reset(vw)
	enc.SetRegistry(r)

	err := enc.Encode(val)
	if err != nil {
		return nil, err
	}

	return []byte(w), nil
}

// MarshalDocument returns val encoded as a *Document.
//
// MarshalDocument will use the default registry created by NewRegistry to recursively
// marshal val into a *Document. MarshalDocument will inspect struct tags and alter the
// marshaling process accordingly.
func MarshalDocument(val interface{}) (*Document, error) { return nil, nil }

// MarshalDocumentAppend will append val encoded to dst. If dst is nil, a new *Document will be
// allocated and the encoding of val will be appended to that.
func MarshalDocumentAppend(dst *Document, val interface{}) (*Document, error) { return nil, nil }

// MarshalDocumentWithRegistry returns val encoded as a *Document using r.
func MarshalDocumentWithRegistry(r *Registry, val interface{}) (*Document, error) { return nil, nil }

// MarshalDocumentAppendWithRegistry will append val encoded to dst using r. If dst is nil, a new
// *Document will be allocated and the encoding of val will be appended to that.
func MarshalDocumentAppendWithRegistry(r *Registry, dst *Document, val interface{}) (*Document, error) {
	return nil, nil
}
