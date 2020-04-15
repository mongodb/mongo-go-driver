// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncodec

import (
	"reflect"

	"go.mongodb.org/mongo-driver/bson/bsonrw"
)

// CondAddrEncoder is the encoder used when a pointer to the encoding value has an encoder.
type CondAddrEncoder struct {
	canAddrEnc ValueEncoder
	elseEnc    ValueEncoder
}

var _ ValueEncoder = &CondAddrEncoder{}

// NewCondAddrEncoder returns an CondAddrEncoder.
func NewCondAddrEncoder(canAddrEnc, elseEnc ValueEncoder) *CondAddrEncoder {
	encoder := CondAddrEncoder{canAddrEnc: canAddrEnc, elseEnc: elseEnc}
	return &encoder
}

// EncodeValue is the ValueEncoderFunc for a value that may be addressable.
func (cae *CondAddrEncoder) EncodeValue(ec EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
	if val.CanAddr() {
		return cae.canAddrEnc.EncodeValue(ec, vw, val)
	}
	if cae.elseEnc != nil {
		return cae.elseEnc.EncodeValue(ec, vw, val)
	}
	return ErrNoEncoder{Type: val.Type()}
}

// CondAddrDecoder is the decoder used when a pointer to the value has a decoder.
type CondAddrDecoder struct {
	canAddrDec ValueDecoder
	elseDec    ValueDecoder
}

var _ ValueDecoder = &CondAddrDecoder{}

// NewCondAddrDecoder returns an CondAddrDecoder.
func NewCondAddrDecoder(canAddrDec, elseDec ValueDecoder) *CondAddrDecoder {
	decoder := CondAddrDecoder{canAddrDec: canAddrDec, elseDec: elseDec}
	return &decoder
}

// DecodeValue is the ValueDecoderFunc for a value that may be addressable.
func (cad *CondAddrDecoder) DecodeValue(dc DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if val.CanAddr() {
		return cad.canAddrDec.DecodeValue(dc, vr, val)
	}
	if cad.elseDec != nil {
		return cad.elseDec.DecodeValue(dc, vr, val)
	}
	return ErrNoDecoder{Type: val.Type()}
}
