// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"errors"
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/internal/assert"
)

func TestStringCodec(t *testing.T) {
	t.Run("ObjectIDAsHex", func(t *testing.T) {
		oid := NewObjectID()
		reader := &valueReaderWriter{BSONType: TypeObjectID, Return: oid}
		testCases := []struct {
			name        string
			stringCodec *stringCodec
			result      string
			err         error
		}{
			{"true", &stringCodec{decodeObjectIDAsHex: true}, oid.Hex(), nil},
			{"false", &stringCodec{decodeObjectIDAsHex: false}, "", errors.New("decoding an object ID to a hexadecimal string is disabled by default")},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				actual := reflect.New(reflect.TypeOf("")).Elem()
				err := tc.stringCodec.DecodeValue(DecodeContext{}, reader, actual)
				if tc.err == nil {
					assert.NoErrorf(t, err, "StringCodec.DecodeValue error: %q", err)
				} else {
					assert.EqualErrorf(t, err, tc.err.Error(), "Expected error %q, got %q", tc.err, err)
				}

				actualString := actual.Interface().(string)
				assert.Equal(t, tc.result, actualString, "Expected string %v, got %v", tc.result, actualString)
			})
		}
	})
}
