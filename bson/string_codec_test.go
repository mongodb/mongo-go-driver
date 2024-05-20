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
			name   string
			codec  *stringCodec
			err    error
			result string
		}{
			{"default", &stringCodec{}, errors.New("cannot decode ObjectID as string if DecodeObjectIDAsHex is not set"), ""},
			{"true", &stringCodec{decodeObjectIDAsHex: true}, nil, oid.Hex()},
			{"false", &stringCodec{decodeObjectIDAsHex: false}, errors.New("cannot decode ObjectID as string if DecodeObjectIDAsHex is not set"), ""},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				actual := reflect.New(reflect.TypeOf("")).Elem()
				err := tc.codec.DecodeValue(nil, reader, actual)
				if tc.err == nil {
					assert.NoError(t, err)
				} else {
					assert.EqualError(t, err, tc.err.Error())
				}

				actualString := actual.Interface().(string)
				assert.Equal(t, tc.result, actualString, "Expected string %v, got %v", tc.result, actualString)
			})
		}
	})
}
