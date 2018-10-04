// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"github.com/mongodb/mongo-go-driver/bson/bsonrw"
	"github.com/mongodb/mongo-go-driver/bson/bsontype"
)

var bvwPool = bsonrw.NewBSONValueWriterPool()

// Marshaler is an interface implemented by types that can marshal themselves
// into a BSON document represented as bytes. The bytes returned must be a valid
// BSON document if the error is nil.
type Marshaler interface {
	MarshalBSON() ([]byte, error)
}

// ValueMarshaler is an interface implemented by types that can marshal
// themselves into a BSON value as bytes. The type must be the valid type for
// the bytes returned. The bytes and byte type together must be valid if the
// error is nil.
type ValueMarshaler interface {
	MarshalBSONValue() (bsontype.Type, []byte, error)
}

// Marshal returns the BSON encoding of val.
//
// Marshal will use the default registry created by NewRegistry to recursively
// marshal val into a []byte. Marshal will inspect struct tags and alter the
// marshaling process accordingly.
func Marshal(val interface{}) ([]byte, error) {
	return MarshalWithRegistry(DefaultRegistry, val)
}

// MarshalAppend will append the BSON encoding of val to dst. If dst is not
// large enough to hold the BSON encoding of val, dst will be grown.
func MarshalAppend(dst []byte, val interface{}) ([]byte, error) {
	return MarshalAppendWithRegistry(DefaultRegistry, dst, val)
}

// MarshalWithRegistry returns the BSON encoding of val using Registry r.
func MarshalWithRegistry(r *bsoncodec.Registry, val interface{}) ([]byte, error) {
	dst := make([]byte, 0, 256) // TODO: make the default cap a constant
	return MarshalAppendWithRegistry(r, dst, val)
}

// MarshalAppendWithRegistry will append the BSON encoding of val to dst using
// Registry r. If dst is not large enough to hold the BSON encoding of val, dst
// will be grown.
func MarshalAppendWithRegistry(r *bsoncodec.Registry, dst []byte, val interface{}) ([]byte, error) {
	// w := writer(dst)
	// vw := newValueWriter(&w)

	sw := new(bsonrw.SliceWriter)
	*sw = dst
	vw := bvwPool.Get(sw)
	defer bvwPool.Put(vw)
	// vw := vwPool.Get().(*valueWriter)
	// defer vwPool.Put(vw)
	//
	// vw.reset(dst)

	enc := encPool.Get().(*Encoder)
	defer encPool.Put(enc)

	err := enc.Reset(vw)
	if err != nil {
		return nil, err
	}
	err = enc.SetRegistry(r)
	if err != nil {
		return nil, err
	}

	err = enc.Encode(val)
	if err != nil {
		return nil, err
	}

	return *sw, nil
}

// func marshalElement(r *Registry, dst []byte, key string, val interface{}) ([]byte, error) {
// 	vw := vwPool.Get().(*valueWriter)
// 	defer vwPool.Put(vw)
//
// 	vw.reset(dst)
// 	_, err := vw.WriteDocumentElement(key)
// 	if err != nil {
// 		return dst, err
// 	}
// 	t := reflect.TypeOf(val)
// 	enc, err := r.LookupEncoder(t)
// 	if err != nil {
// 		return dst, err
// 	}
// 	err = enc.EncodeValue(EncodeContext{Registry: r}, vw, val)
// 	return dst, err
// }
