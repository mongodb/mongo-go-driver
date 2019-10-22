// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncodec

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/bsonoptions"
	"go.mongodb.org/mongo-driver/bson/bsonrw/bsonrwtest"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
)

func TestTimeCodec(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)

	type timeSubtest struct {
		name string
		opts *bsonoptions.TimeCodecOptions
		utc  bool
		err  error
	}

	testCases := []struct {
		name     string
		reader   *bsonrwtest.ValueReaderWriter
		subtests []timeSubtest
	}{
		{
			"UseLocalTimeZone",
			&bsonrwtest.ValueReaderWriter{BSONType: bsontype.DateTime, Return: int64(now.UnixNano() / int64(time.Millisecond))},
			[]timeSubtest{
				{
					"default",
					bsonoptions.TimeCodec(),
					true,
					nil,
				},
				{
					"false",
					bsonoptions.TimeCodec().SetUseLocalTimeZone(false),
					true,
					nil,
				},
				{
					"true",
					bsonoptions.TimeCodec().SetUseLocalTimeZone(true),
					false,
					nil,
				},
			},
		},
		{
			"DecodeFromString",
			&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String, Return: now.Format(timeFormatString)},
			[]timeSubtest{
				{
					"default",
					bsonoptions.TimeCodec(),
					true,
					fmt.Errorf("cannot decode string into a time.Time"),
				},
				{
					"false",
					bsonoptions.TimeCodec().SetDecodeFromString(false),
					true,
					fmt.Errorf("cannot decode string into a time.Time"),
				},
				{
					"true",
					bsonoptions.TimeCodec().SetDecodeFromString(true),
					true,
					nil,
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, rc := range tc.subtests {
				t.Run(rc.name, func(t *testing.T) {
					timeCodec, err := NewTimeCodec(rc.opts)
					assert.Nil(t, err, "NewTimeCodec error: %v", err)

					actual := reflect.New(reflect.TypeOf(now)).Elem()
					err = timeCodec.DecodeValue(DecodeContext{}, tc.reader, actual)
					if !compareErrors(err, rc.err) {
						t.Errorf("Errors do not match. got %v; want %v", err, rc.err)
					}

					if rc.err == nil {
						actualTime := actual.Interface().(time.Time)
						assert.Equal(t, actualTime.Location().String() == "UTC", rc.utc,
							"Expected UTC: %v, got %v", rc.utc, actualTime.Location())
						assert.Equal(t, now, actualTime, "expected time %v, got %v", now, actualTime)
					}
				})
			}
		})
	}
}
