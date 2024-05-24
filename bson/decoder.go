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
)

// ErrDecodeToNil is the error returned when trying to decode to a nil value
var ErrDecodeToNil = errors.New("cannot Decode to nil value")

// ConfigurableDecoderRegistry refers a DecoderRegistry that is configurable with *RegistryOpt.
type ConfigurableDecoderRegistry interface {
	DecoderRegistry
	SetCodecOption(opt *RegistryOpt) error
}

// A Decoder reads and decodes BSON documents from a stream. It reads from a ValueReader as
// the source of BSON data.
type Decoder struct {
	reg ConfigurableDecoderRegistry
	vr  ValueReader
}

// NewDecoder returns a new decoder that uses the default registry to read from vr.
func NewDecoder(vr ValueReader) *Decoder {
	r := NewRegistryBuilder().Build()
	return &Decoder{
		reg: r,
		vr:  vr,
	}
}

// NewDecoderWithRegistry returns a new decoder that uses the given registry to read from vr.
func NewDecoderWithRegistry(r *Registry, vr ValueReader) *Decoder {
	return &Decoder{
		reg: r,
		vr:  vr,
	}
}

// Decode reads the next BSON document from the stream and decodes it into the
// value pointed to by val.
//
// See [Unmarshal] for details about BSON unmarshaling behavior.
func (d *Decoder) Decode(val interface{}) error {
	if unmarshaler, ok := val.(Unmarshaler); ok {
		// TODO(skriptble): Reuse a []byte here and use the AppendDocumentBytes method.
		buf, err := copyDocumentToBytes(d.vr)
		if err != nil {
			return err
		}
		return unmarshaler.UnmarshalBSON(buf)
	}

	rval := reflect.ValueOf(val)
	switch rval.Kind() {
	case reflect.Ptr:
		if rval.IsNil() {
			return ErrDecodeToNil
		}
		rval = rval.Elem()
	case reflect.Map:
		if rval.IsNil() {
			return ErrDecodeToNil
		}
	default:
		return fmt.Errorf("argument to Decode must be a pointer or a map, but got %v", rval)
	}
	decoder, err := d.reg.LookupDecoder(rval.Type())
	if err != nil {
		return err
	}

	return decoder.DecodeValue(d.reg, d.vr, rval)
}

// SetBehavior set the decoder behavior with *RegistryOpt.
func (d *Decoder) SetBehavior(opt *RegistryOpt) error {
	return d.reg.SetCodecOption(opt)
}
