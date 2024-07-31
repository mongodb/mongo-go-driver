// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
)

func TestCondAddrCodec(t *testing.T) {
	var inner int
	canAddrVal := reflect.ValueOf(&inner)
	addressable := canAddrVal.Elem()
	unaddressable := reflect.ValueOf(inner)
	rw := &valueReaderWriter{}

	t.Run("addressEncode", func(t *testing.T) {
		invoked := 0
		encode1 := ValueEncoderFunc(func(EncodeContext, ValueWriter, reflect.Value) error {
			invoked = 1
			return nil
		})
		encode2 := ValueEncoderFunc(func(EncodeContext, ValueWriter, reflect.Value) error {
			invoked = 2
			return nil
		})
		condEncoder := newCondAddrEncoder(encode1, encode2)

		testCases := []struct {
			name    string
			val     reflect.Value
			invoked int
		}{
			{"canAddr", addressable, 1},
			{"else", unaddressable, 2},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := condEncoder.EncodeValue(EncodeContext{}, rw, tc.val)
				assert.Nil(t, err, "CondAddrEncoder error: %v", err)

				assert.Equal(t, invoked, tc.invoked, "Expected function %v to be called, called %v", tc.invoked, invoked)
			})
		}

		t.Run("error", func(t *testing.T) {
			errEncoder := newCondAddrEncoder(encode1, nil)
			err := errEncoder.EncodeValue(EncodeContext{}, rw, unaddressable)
			want := errNoEncoder{Type: unaddressable.Type()}
			assert.Equal(t, err, want, "expected error %v, got %v", want, err)
		})
	})
	t.Run("addressDecode", func(t *testing.T) {
		invoked := 0
		decode1 := ValueDecoderFunc(func(DecodeContext, ValueReader, reflect.Value) error {
			invoked = 1
			return nil
		})
		decode2 := ValueDecoderFunc(func(DecodeContext, ValueReader, reflect.Value) error {
			invoked = 2
			return nil
		})
		condDecoder := newCondAddrDecoder(decode1, decode2)

		testCases := []struct {
			name    string
			val     reflect.Value
			invoked int
		}{
			{"canAddr", addressable, 1},
			{"else", unaddressable, 2},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := condDecoder.DecodeValue(DecodeContext{}, rw, tc.val)
				assert.Nil(t, err, "CondAddrDecoder error: %v", err)

				assert.Equal(t, invoked, tc.invoked, "Expected function %v to be called, called %v", tc.invoked, invoked)
			})
		}

		t.Run("error", func(t *testing.T) {
			errDecoder := newCondAddrDecoder(decode1, nil)
			err := errDecoder.DecodeValue(DecodeContext{}, rw, unaddressable)
			want := errNoDecoder{Type: unaddressable.Type()}
			assert.Equal(t, err, want, "expected error %v, got %v", want, err)
		})
	})
}
