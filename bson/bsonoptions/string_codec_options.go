// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsonoptions

// StringCodecOptions represents all possible options for time.Time encoding and decoding.
type StringCodecOptions struct {
	ObjectIDAsHex *bool // Specifies if we should decode ObjectID as the hex value. Defaults to true.
}

// StringCodec creates a new *StringCodecOptions
func StringCodec() *StringCodecOptions {
	return &StringCodecOptions{}
}

// SetObjectIDAsHex specifies if we should decode ObjectID as the hex value. Defaults to true.
func (t *StringCodecOptions) SetObjectIDAsHex(b bool) *StringCodecOptions {
	t.ObjectIDAsHex = &b
	return t
}

// MergeStringCodecOptions combines the given *StringCodecOptions into a single *StringCodecOptions in a last one wins fashion.
func MergeStringCodecOptions(opts ...*StringCodecOptions) *StringCodecOptions {
	s := StringCodec()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if opt.ObjectIDAsHex != nil {
			s.ObjectIDAsHex = opt.ObjectIDAsHex
		}
	}

	return s
}
