// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package writeconcern_test

import (
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

func TestWriteConcernWithOptions(t *testing.T) {
	t.Parallel()

	t.Run("on nil WriteConcern", func(t *testing.T) {
		t.Parallel()

		var wc *writeconcern.WriteConcern

		wc = wc.WithOptions(writeconcern.WMajority())
		assert.Equal(t, "majority", wc.GetW().(string))
		assert.False(t, wc.GetJ())
	})
	t.Run("on existing WriteConcern", func(t *testing.T) {
		t.Parallel()

		wc := writeconcern.New(writeconcern.W(1), writeconcern.J(true))
		assert.Equal(t, 1, wc.GetW().(int))
		assert.True(t, wc.GetJ())

		wc = wc.WithOptions(writeconcern.WMajority())
		assert.Equal(t, "majority", wc.GetW().(string))
		assert.True(t, wc.GetJ())
	})
	t.Run("with multiple options", func(t *testing.T) {
		t.Parallel()

		wc := writeconcern.New(writeconcern.W(1), writeconcern.J(true))
		assert.Equal(t, 1, wc.GetW().(int))
		assert.True(t, wc.GetJ())
		assert.Equal(t, time.Duration(0), wc.GetWTimeout())

		wc = wc.WithOptions(writeconcern.WMajority(), writeconcern.WTimeout(time.Second))
		assert.Equal(t, "majority", wc.GetW().(string))
		assert.True(t, wc.GetJ())
		assert.Equal(t, time.Second, wc.GetWTimeout())
	})
}

func TestWriteConcern_MarshalBSONValue(t *testing.T) {
	t.Parallel()

	boolPtr := func(b bool) *bool { return &b }

	testCases := []struct {
		description string
		input       *writeconcern.WriteConcern
		wantType    bsontype.Type
		wantValue   bson.D
		wantError   error
	}{
		{
			description: "all fields",
			input: &writeconcern.WriteConcern{
				W:        "majority",
				Journal:  boolPtr(false),
				WTimeout: 1 * time.Minute,
			},
			wantType: bson.TypeEmbeddedDocument,
			wantValue: bson.D{
				{Key: "w", Value: "majority"},
				{Key: "j", Value: false},
				{Key: "wtimeout", Value: int64(60_000)},
			},
		},
		{
			description: "string W",
			input:       &writeconcern.WriteConcern{W: "majority"},
			wantType:    bson.TypeEmbeddedDocument,
			wantValue:   bson.D{{Key: "w", Value: "majority"}},
		},
		{
			description: "int W",
			input:       &writeconcern.WriteConcern{W: 1},
			wantType:    bson.TypeEmbeddedDocument,
			wantValue:   bson.D{{Key: "w", Value: int32(1)}},
		},
		{
			description: "int32 W",
			input:       &writeconcern.WriteConcern{W: int32(1)},
			wantError:   errors.New("WriteConcern.W must be a string or int, but is a int32"),
		},
		{
			description: "bool W",
			input:       &writeconcern.WriteConcern{W: false},
			wantError:   errors.New("WriteConcern.W must be a string or int, but is a bool"),
		},
		{
			description: "W=0 and J=true",
			input:       &writeconcern.WriteConcern{W: 0, Journal: boolPtr(true)},
			wantError:   writeconcern.ErrInconsistent,
		},
		{
			description: "negative W",
			input:       &writeconcern.WriteConcern{W: -1},
			wantError:   writeconcern.ErrNegativeW,
		},
		{
			description: "negative WTimeout",
			input:       &writeconcern.WriteConcern{W: 1, WTimeout: -1},
			wantError:   writeconcern.ErrNegativeWTimeout,
		},
		{
			description: "empty",
			input:       &writeconcern.WriteConcern{},
			wantError:   writeconcern.ErrEmptyWriteConcern,
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable.

		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			typ, b, err := tc.input.MarshalBSONValue()
			if tc.wantError != nil {
				assert.Equal(t, tc.wantError, err, "expected and actual errors do not match")
				return
			}
			require.NoError(t, err, "bson.MarshalValue error")

			assert.Equal(t, tc.wantType, typ, "expected and actual BSON types do not match")

			rv := bson.RawValue{
				Type:  typ,
				Value: b,
			}
			var gotValue bson.D
			err = rv.Unmarshal(&gotValue)
			require.NoError(t, err, "error unmarshaling RawValue")
			assert.Equal(t, tc.wantValue, gotValue, "expected and actual BSON values do not match")
		})
	}
}

func TestWriteConcern_Acknowledged(t *testing.T) {
	boolPtr := func(b bool) *bool { return &b }

	testCases := []struct {
		description  string
		writeConcern *writeconcern.WriteConcern
		want         bool
	}{
		{
			description:  "Unacknowledged",
			writeConcern: writeconcern.Unacknowledged(),
			want:         false,
		},
		{
			description:  "W1",
			writeConcern: writeconcern.W1(),
			want:         true,
		},
		{
			description:  "Journaled",
			writeConcern: writeconcern.Journaled(),
			want:         true,
		},
		{
			description:  "Majority",
			writeConcern: writeconcern.Majority(),
			want:         true,
		},
		{
			description: "{w: 0, j: true}",
			writeConcern: &writeconcern.WriteConcern{
				W:       0,
				Journal: boolPtr(true),
			},
			want: true,
		},
		{
			description:  "{w: custom}",
			writeConcern: &writeconcern.WriteConcern{W: "custom"},
			want:         true,
		},
		{
			description:  "nil",
			writeConcern: nil,
			want:         true,
		},
		{
			description: "invalid",
			writeConcern: &writeconcern.WriteConcern{
				W: struct{ Field string }{},
			},
			want: true,
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable.

		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			got := tc.writeConcern.Acknowledged()
			assert.Equal(t, tc.want, got, "expected and actual Acknowledged value are different")
		})
	}
}
