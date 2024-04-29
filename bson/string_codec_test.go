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
			dctx   DecodeContext
			err    error
			result string
		}{
			{"default", DecodeContext{}, errors.New("cannot decode ObjectID as string if DecodeObjectIDAsHex is not set"), ""},
			{"true", DecodeContext{decodeObjectIDAsHex: true}, nil, oid.Hex()},
			{"false", DecodeContext{decodeObjectIDAsHex: false}, errors.New("cannot decode ObjectID as string if DecodeObjectIDAsHex is not set"), ""},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				stringCodec := &stringCodec{}

				actual := reflect.New(reflect.TypeOf("")).Elem()
				err := stringCodec.DecodeValue(tc.dctx, reader, actual)
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
