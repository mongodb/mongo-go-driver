// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"encoding/binary"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValue(t *testing.T) {
	t.Run("panic", func(t *testing.T) {
		handle := func() {
			if got := recover(); got != ErrUninitializedElement {
				want := ErrUninitializedElement
				t.Errorf("Incorrect value for panic. got %s; want %s", got, want)
			}
		}
		t.Run("key", func(t *testing.T) {
			defer handle()
			(*Element)(nil).Key()
		})
		t.Run("type", func(t *testing.T) {
			defer handle()
			(*Value)(nil).Type()
		})
		t.Run("double", func(t *testing.T) {
			defer handle()
			(*Value)(nil).Double()
		})
		t.Run("string", func(t *testing.T) {
			defer handle()
			(*Value)(nil).StringValue()
		})
		t.Run("document", func(t *testing.T) {
			defer handle()
			(*Value)(nil).ReaderDocument()
		})
	})
	t.Run("key", func(t *testing.T) {
		buf := []byte{
			'\x00', '\x00', '\x00', '\x00',
			'\x02', 'f', 'o', 'o', '\x00',
			'\x00', '\x00', '\x00', '\x00', '\x00',
			'\x00'}
		e := &Element{&Value{start: 4, offset: 9, data: buf}}
		want := "foo"
		got := e.Key()
		if got != want {
			t.Errorf("Unexpected result. got %s; want %s", got, want)
		}
	})
	t.Run("type", func(t *testing.T) {
		buf := []byte{
			'\x00', '\x00', '\x00', '\x00',
			'\x02', 'f', 'o', 'o', '\x00',
			'\x00', '\x00', '\x00', '\x00', '\x00',
			'\x00',
		}
		e := &Element{&Value{start: 4, offset: 9, data: buf}}
		want := TypeString
		got := e.value.Type()
		if got != want {
			t.Errorf("Unexpected result. got %v; want %v", got, want)
		}
	})
	t.Run("double", func(t *testing.T) {
		buf := []byte{
			'\x00', '\x00', '\x00', '\x00',
			'\x01', 'f', 'o', 'o', '\x00',
			'\x00', '\x00', '\x00', '\x00',
			'\x00', '\x00', '\x00', '\x00',
			'\x00',
		}
		e := &Element{&Value{start: 4, offset: 9, data: buf}}
		binary.LittleEndian.PutUint64(buf[9:17], math.Float64bits(3.14159))
		want := 3.14159
		got := e.value.Double()
		if got != want {
			t.Errorf("Unexpected result. got %f; want %f", got, want)
		}
	})
	t.Run("string", func(t *testing.T) {
		buf := []byte{
			'\x00', '\x00', '\x00', '\x00',
			'\x02', 'f', 'o', 'o', '\x00',
			'\x00', '\x00', '\x00', '\x00',
			'b', 'a', 'r', '\x00',
			'\x00',
		}
		e := &Element{&Value{start: 4, offset: 9, data: buf}}
		binary.LittleEndian.PutUint32(buf[9:13], 4)
		want := "bar"
		got := e.value.StringValue()
		if got != want {
			t.Errorf("Unexpected result. got %s; want %s", got, want)
		}
	})
	t.Run("document", func(t *testing.T) {})
}

func TestTimeRoundTrip(t *testing.T) {
	val := struct {
		Value time.Time
		ID    string
	}{
		ID: "time-rt-test",
	}

	assert.True(t, val.Value.IsZero())

	bsonOut, err := Marshal(val)
	assert.NoError(t, err)
	rtval := struct {
		Value time.Time
		ID    string
	}{}

	err = Unmarshal(bsonOut, &rtval)
	assert.NoError(t, err)
	assert.Equal(t, val, rtval)
	assert.True(t, rtval.Value.IsZero())

}

func TestUTCDateTimeRoundTrip(t *testing.T) {
	timeNow := time.Date(2018, 9, 20, 1, 1, 1, 0, time.UTC)
	int64Now := timeNow.UnixNano() / 1e6

	timeVal := struct {
		Value time.Time `bson:"Value"`
		ID    string    `bson:"ID"`
	}{
		Value: timeNow,
		ID:    "time-decode-test",
	}

	bsonOut, err := Marshal(timeVal)
	assert.NoError(t, err)

	timeRtval := struct {
		Value time.Time `bson:"Value"`
		ID    string    `bson:"ID"`
	}{}

	int64Rtval := struct {
		Value int64  `bson:"Value"`
		ID    string `bson:"ID"`
	}{}

	err = Unmarshal(bsonOut, &timeRtval)
	assert.NoError(t, err)
	err = Unmarshal(bsonOut, &int64Rtval)
	assert.NoError(t, err)

	timeRtval.Value = timeRtval.Value.UTC()

	assert.Equal(t, timeRtval, timeVal)
	assert.Equal(t, int64Rtval.Value, int64Now)

}

func TestBasicEncode(t *testing.T) {
	for _, tc := range marshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			got := make(writer, 0, 1024)
			vw := newValueWriter(&got)
			reg := NewRegistryBuilder().Build()
			codec, err := reg.Lookup(reflect.TypeOf(tc.val))
			noerr(t, err)
			err = codec.EncodeValue(EncodeContext{Registry: reg}, vw, tc.val)
			noerr(t, err)

			if !bytes.Equal(got, tc.want) {
				t.Errorf("Bytes are not equal. got %v; want %v", Reader(got), Reader(tc.want))
				t.Errorf("Bytes:\n%v\n%v", got, tc.want)
			}
		})
	}
}

func TestBasicDecode(t *testing.T) {
	for _, tc := range unmarshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			got := reflect.New(tc.sType).Interface()
			vr := newValueReader(tc.data)
			reg := NewRegistryBuilder().Build()
			codec, err := reg.Lookup(reflect.TypeOf(got))
			noerr(t, err)
			err = codec.DecodeValue(DecodeContext{Registry: reg}, vr, got)
			noerr(t, err)

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Results do not match. got %+v; want %+v", got, tc.want)
			}
		})
	}
}
