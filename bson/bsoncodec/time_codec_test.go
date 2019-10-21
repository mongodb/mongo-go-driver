// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncodec

import (
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
	t.Run("UseLocalTimeZone", func(t *testing.T) {
		timeCodec, err := NewTimeCodec(bsonoptions.TimeCodec().SetUseLocalTimeZone(true))
		assert.Nil(t, err, "NewTimeCodec error: %v", err)

		actual := reflect.New(reflect.TypeOf(now)).Elem()
		reader := &bsonrwtest.ValueReaderWriter{BSONType: bsontype.DateTime, Return: int64(now.UnixNano() / int64(time.Millisecond))}
		err = timeCodec.DecodeValue(DecodeContext{}, reader, actual)
		assert.Nil(t, err, "TimeCodec.DecodeValue error: %v", err)

		actualTime := actual.Interface().(time.Time)
		assert.True(t, actualTime.Location() != time.UTC, "Should decode to local time instead of UTC")
	})
	t.Run("DecodeFromString", func(t *testing.T) {
		timeCodec, err := NewTimeCodec(bsonoptions.TimeCodec().SetDecodeFromString(true))
		assert.Nil(t, err, "NewTimeCodec error: %v", err)

		actual := reflect.New(reflect.TypeOf(now)).Elem()
		reader := &bsonrwtest.ValueReaderWriter{BSONType: bsontype.String, Return: now.Format(timeFormatString)}
		err = timeCodec.DecodeValue(DecodeContext{}, reader, actual)
		assert.Nil(t, err, "TimeCodec.DecodeValue error: %v", err)

		actualTime := actual.Interface().(time.Time)
		assert.True(t, actualTime.Equal(now), "Should be equal to the original time")
	})
}
