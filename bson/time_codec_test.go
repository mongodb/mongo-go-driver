// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

func TestTimeCodec(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)

	t.Run("UseLocalTimeZone", func(t *testing.T) {
		reader := &valueReaderWriter{BSONType: TypeDateTime, Return: now.UnixNano() / int64(time.Millisecond)}
		testCases := []struct {
			name      string
			timeCodec *timeCodec
			utc       bool
		}{
			{"default", &timeCodec{}, true},
			{"false", &timeCodec{useLocalTimeZone: false}, true},
			{"true", &timeCodec{useLocalTimeZone: true}, false},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				actual := reflect.New(reflect.TypeOf(now)).Elem()
				err := tc.timeCodec.DecodeValue(DecodeContext{}, reader, actual)
				assert.Nil(t, err, "TimeCodec.DecodeValue error: %v", err)

				actualTime := actual.Interface().(time.Time)
				assert.Equal(t, actualTime.Location().String() == "UTC", tc.utc,
					"Expected UTC: %v, got %v", tc.utc, actualTime.Location())

				if tc.utc {
					nowUTC := now.UTC()
					assert.Equal(t, nowUTC, actualTime, "expected time %v, got %v", nowUTC, actualTime)
				} else {
					assert.Equal(t, now, actualTime, "expected time %v, got %v", now, actualTime)
				}
			})
		}
	})

	t.Run("DecodeFromBsontype", func(t *testing.T) {
		testCases := []struct {
			name   string
			reader *valueReaderWriter
		}{
			{"string", &valueReaderWriter{BSONType: TypeString, Return: now.Format(timeFormatString)}},
			{"int64", &valueReaderWriter{BSONType: TypeInt64, Return: now.Unix()*1000 + int64(now.Nanosecond()/1e6)}},
			{"timestamp", &valueReaderWriter{BSONType: TypeTimestamp,
				Return: bsoncore.Value{
					Type: bsoncore.TypeTimestamp,
					Data: bsoncore.AppendTimestamp(nil, uint32(now.Unix()), 0),
				}},
			},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				actual := reflect.New(reflect.TypeOf(now)).Elem()
				err := (&timeCodec{}).DecodeValue(DecodeContext{}, tc.reader, actual)
				assert.Nil(t, err, "DecodeValue error: %v", err)

				actualTime := actual.Interface().(time.Time)
				if tc.name == "timestamp" {
					now = time.Unix(now.Unix(), 0)
				}
				nowUTC := now.UTC()
				assert.Equal(t, nowUTC, actualTime, "expected time %v, got %v", nowUTC, actualTime)
			})
		}
	})
}
