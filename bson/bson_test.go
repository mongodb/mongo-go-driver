// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func noerr(t *testing.T, err error) {
	if err != nil {
		t.Helper()
		t.Errorf("Unexpected error: (%T)%v", err, err)
		t.FailNow()
	}
}

func TestCompareTimestamp(t *testing.T) {
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

func TestTimestamp(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		description     string
		tp              Timestamp
		tp2             Timestamp
		expectedAfter   bool
		expectedBefore  bool
		expectedEqual   bool
		expectedCompare int
	}{
		{
			description:     "equal",
			tp:              Timestamp{T: 12345, I: 67890},
			tp2:             Timestamp{T: 12345, I: 67890},
			expectedBefore:  false,
			expectedAfter:   false,
			expectedEqual:   true,
			expectedCompare: 0,
		},
		{
			description:     "T greater than",
			tp:              Timestamp{T: 12345, I: 67890},
			tp2:             Timestamp{T: 2345, I: 67890},
			expectedBefore:  false,
			expectedAfter:   true,
			expectedEqual:   false,
			expectedCompare: 1,
		},
		{
			description:     "I greater than",
			tp:              Timestamp{T: 12345, I: 67890},
			tp2:             Timestamp{T: 12345, I: 7890},
			expectedBefore:  false,
			expectedAfter:   true,
			expectedEqual:   false,
			expectedCompare: 1,
		},
		{
			description:     "T less than",
			tp:              Timestamp{T: 12345, I: 67890},
			tp2:             Timestamp{T: 112345, I: 67890},
			expectedBefore:  true,
			expectedAfter:   false,
			expectedEqual:   false,
			expectedCompare: -1,
		},
		{
			description:     "I less than",
			tp:              Timestamp{T: 12345, I: 67890},
			tp2:             Timestamp{T: 12345, I: 167890},
			expectedBefore:  true,
			expectedAfter:   false,
			expectedEqual:   false,
			expectedCompare: -1,
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable.

		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expectedAfter, tc.tp.After(tc.tp2), "expected After results to be the same")
			assert.Equal(t, tc.expectedBefore, tc.tp.Before(tc.tp2), "expected Before results to be the same")
			assert.Equal(t, tc.expectedEqual, tc.tp.Equal(tc.tp2), "expected Equal results to be the same")
			assert.Equal(t, tc.expectedCompare, tc.tp.Compare(tc.tp2), "expected Compare result to be the same")
		})
	}
}

func TestPrimitiveIsZero(t *testing.T) {
	testcases := []struct {
		name    string
		zero    Zeroer
		nonzero Zeroer
	}{
		{"binary", Binary{}, Binary{Data: []byte{0x01, 0x02, 0x03}, Subtype: 0xFF}},
		{"decimal128", Decimal128{}, NewDecimal128(1, 2)},
		{"objectID", ObjectID{}, NewObjectID()},
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
		t.Run("UTC", func(t *testing.T) {
			dt := DateTime(1681145535123)
			jsonBytes, err := json.Marshal(dt)
			assert.Nil(t, err, "Marshal error: %v", err)
			assert.Equal(t, `"2023-04-10T16:52:15.123Z"`, string(jsonBytes))
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

func TestTimeRoundTrip(t *testing.T) {
	val := struct {
		Value time.Time
		ID    string
	}{
		ID: "time-rt-test",
	}

	if !val.Value.IsZero() {
		t.Errorf("Did not get zero time as expected.")
	}

	bsonOut, err := Marshal(val)
	noerr(t, err)
	rtval := struct {
		Value time.Time
		ID    string
	}{}

	err = Unmarshal(bsonOut, &rtval)
	noerr(t, err)
	if !cmp.Equal(val, rtval) {
		t.Errorf("Did not round trip properly. got %v; want %v", val, rtval)
	}
	if !rtval.Value.IsZero() {
		t.Errorf("Did not get zero time as expected.")
	}
}

func TestNonNullTimeRoundTrip(t *testing.T) {
	now := time.Now()
	now = time.Unix(now.Unix(), 0)
	val := struct {
		Value time.Time
		ID    string
	}{
		ID:    "time-rt-test",
		Value: now,
	}

	bsonOut, err := Marshal(val)
	noerr(t, err)
	rtval := struct {
		Value time.Time
		ID    string
	}{}

	err = Unmarshal(bsonOut, &rtval)
	noerr(t, err)
	if !cmp.Equal(val, rtval) {
		t.Errorf("Did not round trip properly. got %v; want %v", val, rtval)
	}
}

func TestD(t *testing.T) {
	t.Run("can marshal", func(t *testing.T) {
		d := D{{"foo", "bar"}, {"hello", "world"}, {"pi", 3.14159}}
		idx, want := bsoncore.AppendDocumentStart(nil)
		want = bsoncore.AppendStringElement(want, "foo", "bar")
		want = bsoncore.AppendStringElement(want, "hello", "world")
		want = bsoncore.AppendDoubleElement(want, "pi", 3.14159)
		want, err := bsoncore.AppendDocumentEnd(want, idx)
		noerr(t, err)
		got, err := Marshal(d)
		noerr(t, err)
		if !bytes.Equal(got, want) {
			t.Errorf("Marshaled documents do not match. got %v; want %v", Raw(got), Raw(want))
		}
	})
	t.Run("can unmarshal", func(t *testing.T) {
		want := D{{"foo", "bar"}, {"hello", "world"}, {"pi", 3.14159}}
		idx, doc := bsoncore.AppendDocumentStart(nil)
		doc = bsoncore.AppendStringElement(doc, "foo", "bar")
		doc = bsoncore.AppendStringElement(doc, "hello", "world")
		doc = bsoncore.AppendDoubleElement(doc, "pi", 3.14159)
		doc, err := bsoncore.AppendDocumentEnd(doc, idx)
		noerr(t, err)
		var got D
		err = Unmarshal(doc, &got)
		noerr(t, err)
		if !cmp.Equal(got, want) {
			t.Errorf("Unmarshaled documents do not match. got %v; want %v", got, want)
		}
	})
}

type stringerString string

func (ss stringerString) String() string {
	return "bar"
}

type keyBool bool

func (kb keyBool) MarshalKey() (string, error) {
	return fmt.Sprintf("%v", kb), nil
}

func (kb *keyBool) UnmarshalKey(key string) error {
	switch key {
	case "true":
		*kb = true
	case "false":
		*kb = false
	default:
		return fmt.Errorf("invalid bool value %v", key)
	}
	return nil
}

type keyStruct struct {
	val int64
}

func (k keyStruct) MarshalText() (text []byte, err error) {
	str := strconv.FormatInt(k.val, 10)

	return []byte(str), nil
}

func (k *keyStruct) UnmarshalText(text []byte) error {
	val, err := strconv.ParseInt(string(text), 10, 64)
	if err != nil {
		return err
	}

	*k = keyStruct{
		val: val,
	}

	return nil
}

func TestMapCodec(t *testing.T) {
	t.Run("EncodeKeysWithStringer", func(t *testing.T) {
		strstr := stringerString("foo")
		mapObj := map[stringerString]int{strstr: 1}
		testCases := []struct {
			name  string
			codec *mapCodec
			key   string
		}{
			{"default", &mapCodec{}, "foo"},
			{"true", &mapCodec{encodeKeysWithStringer: true}, "bar"},
			{"false", &mapCodec{encodeKeysWithStringer: false}, "foo"},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				mapRegistry := NewRegistryBuilder()
				mapRegistry.RegisterKindEncoder(reflect.Map, func() ValueEncoder { return tc.codec })
				buf := new(bytes.Buffer)
				vw := NewValueWriter(buf)
				enc := NewEncoder(vw)
				enc.SetRegistry(mapRegistry.Build())
				err := enc.Encode(mapObj)
				assert.Nil(t, err, "Encode error: %v", err)
				str := buf.String()
				assert.True(t, strings.Contains(str, tc.key), "expected result to contain %v, got: %v", tc.key, str)
			})
		}
	})

	t.Run("keys implements keyMarshaler and keyUnmarshaler", func(t *testing.T) {
		mapObj := map[keyBool]int{keyBool(true): 1}

		doc, err := Marshal(mapObj)
		assert.Nil(t, err, "Marshal error: %v", err)
		idx, want := bsoncore.AppendDocumentStart(nil)
		want = bsoncore.AppendInt32Element(want, "true", 1)
		want, _ = bsoncore.AppendDocumentEnd(want, idx)
		assert.Equal(t, want, doc, "expected result %v, got %v", string(want), string(doc))

		var got map[keyBool]int
		err = Unmarshal(doc, &got)
		assert.Nil(t, err, "Unmarshal error: %v", err)
		assert.Equal(t, mapObj, got, "expected result %v, got %v", mapObj, got)

	})

	t.Run("keys implements encoding.TextMarshaler and encoding.TextUnmarshaler", func(t *testing.T) {
		mapObj := map[keyStruct]int{
			{val: 10}: 100,
		}

		doc, err := Marshal(mapObj)
		assert.Nil(t, err, "Marshal error: %v", err)
		idx, want := bsoncore.AppendDocumentStart(nil)
		want = bsoncore.AppendInt32Element(want, "10", 100)
		want, _ = bsoncore.AppendDocumentEnd(want, idx)
		assert.Equal(t, want, doc, "expected result %v, got %v", string(want), string(doc))

		var got map[keyStruct]int
		err = Unmarshal(doc, &got)
		assert.Nil(t, err, "Unmarshal error: %v", err)
		assert.Equal(t, mapObj, got, "expected result %v, got %v", mapObj, got)

	})
}

func TestExtJSONEscapeKey(t *testing.T) {
	doc := D{{Key: "\\usb#", Value: int32(1)}}
	b, err := MarshalExtJSON(&doc, false, false)
	noerr(t, err)

	want := "{\"\\\\usb#\":1}"
	if diff := cmp.Diff(want, string(b)); diff != "" {
		t.Errorf("Marshaled documents do not match. got %v, want %v", string(b), want)
	}

	var got D
	err = UnmarshalExtJSON(b, false, &got)
	noerr(t, err)
	if !cmp.Equal(got, doc) {
		t.Errorf("Unmarshaled documents do not match. got %v; want %v", got, doc)
	}
}

func TestBsoncoreArray(t *testing.T) {
	type BSONDocumentArray struct {
		Array []D `bson:"array"`
	}

	type BSONArray struct {
		Array bsoncore.Array `bson:"array"`
	}

	bda := BSONDocumentArray{
		Array: []D{
			{{"x", 1}},
			{{"x", 2}},
			{{"x", 3}},
		},
	}

	expectedBSON, err := Marshal(bda)
	assert.Nil(t, err, "Marshal bsoncore.Document array error: %v", err)

	var ba BSONArray
	err = Unmarshal(expectedBSON, &ba)
	assert.Nil(t, err, "Unmarshal error: %v", err)

	actualBSON, err := Marshal(ba)
	assert.Nil(t, err, "Marshal bsoncore.Array error: %v", err)

	assert.Equal(t, expectedBSON, actualBSON,
		"expected BSON to be %v after Marshalling again; got %v", expectedBSON, actualBSON)

	doc := bsoncore.Document(actualBSON)
	v := doc.Lookup("array")
	assert.Equal(t, bsoncore.TypeArray, v.Type, "expected type array, got %v", v.Type)
}
