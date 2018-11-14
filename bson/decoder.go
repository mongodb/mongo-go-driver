// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"github.com/mongodb/mongo-go-driver/bson/bsonrw"
)

// This pool is used to keep the allocations of Decoders down. This is only used for the Marshal*
// methods and is not consumable from outside of this package. The Decoders retrieved from this pool
// must have both Reset and SetRegistry called on them.
var decPool = sync.Pool{
	New: func() interface{} {
		return new(Decoder)
	},
}

// A Decoder reads and decodes BSON documents from a stream. It reads from a bsonrw.ValueReader as
// the source of BSON data.
type Decoder struct {
	r  *bsoncodec.Registry
	vr bsonrw.ValueReader
}

// NewDecoder returns a new decoder that uses Registry reg to read from r.
func NewDecoder(r *bsoncodec.Registry, vr bsonrw.ValueReader) (*Decoder, error) {
	if r == nil {
		return nil, errors.New("cannot create a new Decoder with a nil Registry")
	}
	if vr == nil {
		return nil, errors.New("cannot create a new Decoder with a nil ValueReader")
	}

	return &Decoder{
		r:  r,
		vr: vr,
	}, nil
}

// Decode reads the next BSON document from the stream and decodes it into the
// value pointed to by val.
//
// The documentation for Unmarshal contains details about of BSON into a Go
// value.
func (d *Decoder) Decode(val interface{}) error {
	if unmarshaler, ok := val.(Unmarshaler); ok {
		// TODO(skriptble): Reuse a []byte here and use the AppendDocumentBytes method.
		buf, err := bsonrw.Copier{}.CopyDocumentToBytes(d.vr)
		if err != nil {
			return err
		}
		return unmarshaler.UnmarshalBSON(buf)
	}

	rval := reflect.TypeOf(val)
	if rval.Kind() != reflect.Ptr {
		return fmt.Errorf("argument to Decode must be a pointer to a type, but got %v", rval)
	}
	decoder, err := d.r.LookupDecoder(rval.Elem())
	if err != nil {
		return err
	}
	return decoder.DecodeValue(bsoncodec.DecodeContext{Registry: d.r}, d.vr, val)
}

// Reset will reset the state of the decoder, using the same *Registry used in
// the original construction but using r for reading.
func (d *Decoder) Reset(vr bsonrw.ValueReader) error {
	d.vr = vr
	return nil
}

// SetRegistry replaces the current registry of the decoder with r.
func (d *Decoder) SetRegistry(r *bsoncodec.Registry) error {
	d.r = r
	return nil
}
