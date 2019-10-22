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

	t.Run("UseLocalTimeZone", func(t *testing.T) {
		reader := &bsonrwtest.ValueReaderWriter{BSONType: bsontype.DateTime, Return: int64(now.UnixNano() / int64(time.Millisecond))}
		testCases := []struct {
			name string
			opts *bsonoptions.TimeCodecOptions
			utc  bool
		}{
			{"default", bsonoptions.TimeCodec(), true},
			{"false", bsonoptions.TimeCodec().SetUseLocalTimeZone(false), true},
			{"true", bsonoptions.TimeCodec().SetUseLocalTimeZone(true), false},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				timeCodec, err := NewTimeCodec(tc.opts)
				assert.Nil(t, err, "NewTimeCodec error: %v", err)

				actual := reflect.New(reflect.TypeOf(now)).Elem()
				err = timeCodec.DecodeValue(DecodeContext{}, reader, actual)
				assert.Nil(t, err, "TimeCodec.DecodeValue error: %v", err)

				actualTime := actual.Interface().(time.Time)
				assert.Equal(t, actualTime.Location().String() == "UTC", tc.utc,
					"Expected UTC: %v, got %v", tc.utc, actualTime.Location())
				assert.Equal(t, now, actualTime, "expected time %v, got %v", now, actualTime)
			})
		}
	})

	t.Run("DecodeFromString", func(t *testing.T) {
		reader := &bsonrwtest.ValueReaderWriter{BSONType: bsontype.String, Return: now.Format(timeFormatString)}
		testCases := []struct {
			name string
			opts *bsonoptions.TimeCodecOptions
			err  error
		}{
			{"default", bsonoptions.TimeCodec(), fmt.Errorf("cannot decode string into a time.Time")},
			{"false", bsonoptions.TimeCodec().SetDecodeFromString(false), fmt.Errorf("cannot decode string into a time.Time")},
			{"true", bsonoptions.TimeCodec().SetDecodeFromString(true), nil},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				timeCodec, err := NewTimeCodec(tc.opts)
				assert.Nil(t, err, "NewTimeCodec error: %v", err)

				actual := reflect.New(reflect.TypeOf(now)).Elem()
				err = timeCodec.DecodeValue(DecodeContext{}, reader, actual)
				if !compareErrors(err, tc.err) {
					t.Errorf("Errors do not match. got %v; want %v", err, tc.err)
				}

				if tc.err == nil {
					actualTime := actual.Interface().(time.Time)
					assert.Equal(t, now, actualTime, "expected time %v, got %v", now, actualTime)
				}
			})
		}
	})
}
