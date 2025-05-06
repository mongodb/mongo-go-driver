// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package codecutil

import (
	"io"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
)

func testEncFn(t *testing.T) EncoderFn {
	t.Helper()

	return func(w io.Writer) *bson.Encoder {
		rw := bson.NewDocumentWriter(w)
		return bson.NewEncoder(rw)
	}
}

func TestMarshalValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		val      interface{}
		registry *bson.Registry
		encFn    EncoderFn
		want     string
		wantErr  error
	}{
		{
			name:    "empty",
			val:     nil,
			want:    "",
			wantErr: ErrNilValue,
			encFn:   testEncFn(t),
		},
		{
			name:  "bson.D",
			val:   bson.D{{"foo", "bar"}},
			want:  `{"foo": "bar"}`,
			encFn: testEncFn(t),
		},
		{
			name:  "map",
			val:   map[string]interface{}{"foo": "bar"},
			want:  `{"foo": "bar"}`,
			encFn: testEncFn(t),
		},
		{
			name:  "struct",
			val:   struct{ Foo string }{Foo: "bar"},
			want:  `{"foo": "bar"}`,
			encFn: testEncFn(t),
		},
		{
			name:  "non-document type",
			val:   "foo: bar",
			want:  `"foo: bar"`,
			encFn: testEncFn(t),
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			value, err := MarshalValue(test.val, test.encFn)

			assert.Equal(t, test.wantErr, err, "expected and actual error do not match")
			assert.Equal(t, test.want, value.String(), "expected and actual comments are different")
		})
	}
}
