// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package primitive

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
)

// The same interface as bsoncodec.Zeroer implemented for tests.
type zeroer interface {
	IsZero() bool
}

func TestTimestampCompare(t *testing.T) {
	testcases := []struct {
		name     string
		tp       Timestamp
		tp2      Timestamp
		expected int
	}{
		{"equal", Timestamp{T: 12345, I: 67890}, Timestamp{T: 12345, I: 67890}, 0},
		{"T greater than", Timestamp{T: 12345, I: 67890}, Timestamp{T: 2345, I: 67890}, 1},
		{"I greater than", Timestamp{T: 12345, I: 67890}, Timestamp{T: 12345, I: 7890}, 1},
		{"T less than", Timestamp{T: 12345, I: 67890}, Timestamp{T: 112345, I: 67890}, -1},
		{"I less than", Timestamp{T: 12345, I: 67890}, Timestamp{T: 12345, I: 167890}, -1},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			result := CompareTimestamp(tc.tp, tc.tp2)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestPrimitiveIsZero(t *testing.T) {
	testcases := []struct {
		name    string
		zero    zeroer
		nonzero zeroer
	}{
		{"binary", Binary{}, Binary{Data: []byte{0x01, 0x02, 0x03}, Subtype: 0xFF}},
		{"regex", Regex{}, Regex{Pattern: "foo", Options: "bar"}},
		{"dbPointer", DBPointer{}, DBPointer{DB: "foobar", Pointer: ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}}},
		{"timestamp", Timestamp{}, Timestamp{T: 12345, I: 67890}},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			require.True(t, tc.zero.IsZero())
			require.False(t, tc.nonzero.IsZero())
		})
	}
}

func TestRegexCompare(t *testing.T) {
	testcases := []struct {
		name string
		r1   Regex
		r2   Regex
		eq   bool
	}{
		{"equal", Regex{Pattern: "foo1", Options: "bar1"}, Regex{Pattern: "foo1", Options: "bar1"}, true},
		{"not equal", Regex{Pattern: "foo1", Options: "bar1"}, Regex{Pattern: "foo2", Options: "bar2"}, false},
		{"not equal", Regex{Pattern: "foo1", Options: "bar1"}, Regex{Pattern: "foo1", Options: "bar2"}, false},
		{"not equal", Regex{Pattern: "foo1", Options: "bar1"}, Regex{Pattern: "foo2", Options: "bar1"}, false},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			require.True(t, tc.r1.Equal(tc.r2) == tc.eq)
		})
	}
}

func TestDateTime(t *testing.T) {
	t.Run("json", func(t *testing.T) {
		t.Run("round trip", func(t *testing.T) {
			original := DateTime(1000)
			jsonBytes, err := json.Marshal(original)
			assert.Nil(t, err, "Marshal error: %v", err)

			var unmarshalled DateTime
			err = json.Unmarshal(jsonBytes, &unmarshalled)
			assert.Nil(t, err, "Unmarshal error: %v", err)

			assert.Equal(t, original, unmarshalled, "expected DateTime %v, got %v", original, unmarshalled)
		})
		t.Run("decode null", func(t *testing.T) {
			jsonBytes := []byte("null")
			var dt DateTime
			err := json.Unmarshal(jsonBytes, &dt)
			assert.Nil(t, err, "Unmarshal error: %v", err)
			assert.Equal(t, DateTime(0), dt, "expected DateTime value to be 0, got %v", dt)
		})
	})
	t.Run("NewDateTimeFromTime", func(t *testing.T) {
		t.Run("range is not limited", func(t *testing.T) {
			// If the implementation internally calls time.Time.UnixNano(), the constructor cannot handle times after
			// the year 2262.

			timeFormat := "2006-01-02T15:04:05.999Z07:00"
			timeString := "3001-01-01T00:00:00Z"
			tt, err := time.Parse(timeFormat, timeString)
			assert.Nil(t, err, "Parse error: %v", err)

			dt := NewDateTimeFromTime(tt)
			assert.True(t, dt > 0, "expected a valid DateTime greater than 0, got %v", dt)
		})
	})
}
