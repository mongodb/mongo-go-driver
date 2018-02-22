// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import "bytes"

// Marshal converts a BSON type to bytes.
//
// TODO(GODRIVER-257): Document which types are valid for value.
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
// TODO(GODRIVER-257): Document which types are valid for value.
func Unmarshal(in []byte, out interface{}) error {
	return NewDecoder(bytes.NewReader(in)).Decode(out)
}

// UnmarshalDocument converts bytes into a *bson.Document.
func UnmarshalDocument(bson []byte) (*Document, error) {
	return ReadDocument(bson)
}
