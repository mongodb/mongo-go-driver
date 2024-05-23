// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"reflect"
)

// ConfigurableEncoderRegistry refers a EncoderRegistry that is configurable with *RegistryOpt.
type ConfigurableEncoderRegistry interface {
	EncoderRegistry
	SetCodecOptions(opts ...*RegistryOpt)
}

// An Encoder writes a serialization format to an output stream. It writes to a ValueWriter
// as the destination of BSON data.
type Encoder struct {
	reg ConfigurableEncoderRegistry
	vw  ValueWriter
}

// NewEncoder returns a new encoder that uses the default registry to write to vw.
func NewEncoder(vw ValueWriter) *Encoder {
	return &Encoder{
		reg: NewRegistryBuilder().Build(),
		vw:  vw,
	}
}

// NewEncoderWithRegistry returns a new encoder that uses the given registry to write to vw.
func NewEncoderWithRegistry(r ConfigurableEncoderRegistry, vw ValueWriter) *Encoder {
	return &Encoder{
		reg: r,
		vw:  vw,
	}
}

// Encode writes the BSON encoding of val to the stream.
//
// See [Marshal] for details about BSON marshaling behavior.
func (e *Encoder) Encode(val interface{}) error {
	if marshaler, ok := val.(Marshaler); ok {
		// TODO(skriptble): Should we have a MarshalAppender interface so that we can have []byte reuse?
		buf, err := marshaler.MarshalBSON()
		if err != nil {
			return err
		}
		return copyDocumentFromBytes(e.vw, buf)
	}

	encoder, err := e.reg.LookupEncoder(reflect.TypeOf(val))
	if err != nil {
		return err
	}

	return encoder.EncodeValue(e.reg, e.vw, reflect.ValueOf(val))
}

// SetBehavior set the encoder behavior with *RegistryOpt.
func (e *Encoder) SetBehavior(opts ...*RegistryOpt) {
	e.reg.SetCodecOptions(opts...)
}
