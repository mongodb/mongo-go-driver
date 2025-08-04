// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"fmt"
	"reflect"
	"time"
)

const (
	timeFormatString = "2006-01-02T15:04:05.999Z07:00"
)

// timeCodec is the Codec used for time.Time values.
type timeCodec struct {
	// useLocalTimeZone specifies if we should decode into the local time zone. Defaults to false.
	useLocalTimeZone bool
}

// Assert that timeCodec satisfies the typeDecoder interface, which allows it to be used
// by collection type decoders (e.g. map, slice, etc) to set individual values in a collection.
var _ typeDecoder = &timeCodec{}

func (tc *timeCodec) decodeType(dc DecodeContext, vr ValueReader, t reflect.Type) (reflect.Value, error) {
	if t != tTime {
		return emptyValue, ValueDecoderError{
			Name:     "TimeDecodeValue",
			Types:    []reflect.Type{tTime},
			Received: reflect.Zero(t),
		}
	}

	var timeVal time.Time
	switch vrType := vr.Type(); vrType {
	case TypeDateTime:
		dt, err := vr.ReadDateTime()
		if err != nil {
			return emptyValue, err
		}
		timeVal = time.Unix(dt/1000, dt%1000*1000000)
	case TypeString:
		// assume strings are in the isoTimeFormat
		timeStr, err := vr.ReadString()
		if err != nil {
			return emptyValue, err
		}
		timeVal, err = time.Parse(timeFormatString, timeStr)
		if err != nil {
			return emptyValue, err
		}
	case TypeInt64:
		i64, err := vr.ReadInt64()
		if err != nil {
			return emptyValue, err
		}
		timeVal = time.Unix(i64/1000, i64%1000*1000000)
	case TypeTimestamp:
		t, _, err := vr.ReadTimestamp()
		if err != nil {
			return emptyValue, err
		}
		timeVal = time.Unix(int64(t), 0)
	case TypeNull:
		if err := vr.ReadNull(); err != nil {
			return emptyValue, err
		}
	case TypeUndefined:
		if err := vr.ReadUndefined(); err != nil {
			return emptyValue, err
		}
	default:
		return emptyValue, fmt.Errorf("cannot decode %v into a time.Time", vrType)
	}

	if !tc.useLocalTimeZone && !dc.useLocalTimeZone {
		timeVal = timeVal.UTC()
	}
	return reflect.ValueOf(timeVal), nil
}

// DecodeValue is the ValueDecoderFunc for time.Time.
func (tc *timeCodec) DecodeValue(dc DecodeContext, vr ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tTime {
		return ValueDecoderError{Name: "TimeDecodeValue", Types: []reflect.Type{tTime}, Received: val}
	}

	elem, err := tc.decodeType(dc, vr, tTime)
	if err != nil {
		return err
	}

	val.Set(elem)
	return nil
}

// EncodeValue is the ValueEncoderFunc for time.TIme.
func (tc *timeCodec) EncodeValue(_ EncodeContext, vw ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tTime {
		return ValueEncoderError{Name: "TimeEncodeValue", Types: []reflect.Type{tTime}, Received: val}
	}
	tt := val.Interface().(time.Time)
	dt := NewDateTimeFromTime(tt)
	return vw.WriteDateTime(int64(dt))
}
