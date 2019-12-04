// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsonoptions

import (
	"reflect"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var defaultDecodeType = reflect.TypeOf(primitive.D{})

// EmptyInterfaceCodecOptions represents all possible options for interface{} encoding and decoding.
type EmptyInterfaceCodecOptions struct {
	DecodeDefaultType  *reflect.Type // Specifies if the default type for decoding. Defaults to bson.D.
	DecodeUnpackBinary *bool         // Specifies if Old and Generic type binarys should default to []slice instead of primitive.Binary. Defaults to false.
}

// EmptyInterfaceCodec creates a new *EmptyInterfaceCodecOptions
func EmptyInterfaceCodec() *EmptyInterfaceCodecOptions {
	return &EmptyInterfaceCodecOptions{}
}

// SetDecodeDefaultType specifies if the default type for decoding. Defaults to bson.D.
func (e *EmptyInterfaceCodecOptions) SetDecodeDefaultType(t reflect.Type) *EmptyInterfaceCodecOptions {
	e.DecodeDefaultType = &t
	return e
}

// SetDecodeUnpackBinary specifies if Old and Generic type binarys should default to []slice instead of primitive.Binary. Defaults to false.
func (e *EmptyInterfaceCodecOptions) SetDecodeUnpackBinary(b bool) *EmptyInterfaceCodecOptions {
	e.DecodeUnpackBinary = &b
	return e
}

// MergeEmptyInterfaceCodecOptions combines the given *EmptyInterfaceCodecOptions into a single *EmptyInterfaceCodecOptions in a last one wins fashion.
func MergeEmptyInterfaceCodecOptions(opts ...*EmptyInterfaceCodecOptions) *EmptyInterfaceCodecOptions {
	e := &EmptyInterfaceCodecOptions{
		DecodeDefaultType: &defaultDecodeType,
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if opt.DecodeDefaultType != nil {
			e.DecodeDefaultType = opt.DecodeDefaultType
		}
		if opt.DecodeUnpackBinary != nil {
			e.DecodeUnpackBinary = opt.DecodeUnpackBinary
		}
	}

	return e
}
