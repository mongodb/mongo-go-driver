// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncodec

import (
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/bson/bsonoptions"
	"go.mongodb.org/mongo-driver/bson/bsonrw/bsonrwtest"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
)

func TestStringCodec(t *testing.T) {
	t.Run("ObjectIDAsHex", func(t *testing.T) {
		oid := primitive.NewObjectID()
		reader := &bsonrwtest.ValueReaderWriter{BSONType: bsontype.ObjectID, Return: oid}
		testCases := []struct {
			name string
			opts *bsonoptions.StringCodecOptions
			hex  bool
		}{
			{"default", bsonoptions.StringCodec(), true},
			{"true", bsonoptions.StringCodec().SetObjectIDAsHex(true), true},
			{"false", bsonoptions.StringCodec().SetObjectIDAsHex(false), false},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				stringCodec, err := NewStringCodec(tc.opts)
				assert.Nil(t, err, "NewStringCodec error: %v", err)

				actual := reflect.New(reflect.TypeOf("")).Elem()
				err = stringCodec.DecodeValue(DecodeContext{}, reader, actual)
				assert.Nil(t, err, "TimeCodec.DecodeValue error: %v", err)

				actualString := actual.Interface().(string)
				if tc.hex {
					assert.Equal(t, oid.Hex(), actualString,
						"Expected string %v, got %v", oid.Hex(), actualString)
				} else {
					byteArray := [12]byte(oid)
					assert.Equal(t, string(byteArray[:]), actualString,
						"Expected string %v, got %v", string(byteArray[:]), actualString)
				}
			})
		}
	})
}
