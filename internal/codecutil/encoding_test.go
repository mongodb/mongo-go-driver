//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package codecutil

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/internal/assert"
)

func TestMarshalValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		val      interface{}
		registry *bsoncodec.Registry
		encFn    EncoderFn
		want     string
		wantErr  error
	}{
		{
			name:    "empty",
			val:     nil,
			want:    "",
			wantErr: ErrNilValue{},
		},
		{
			name: "bson.D",
			val:  bson.D{{"foo", "bar"}},
			want: `{"foo": "bar"}`,
		},
		{
			name: "map",
			val:  map[string]interface{}{"foo": "bar"},
			want: `{"foo": "bar"}`,
		},
		{
			name: "struct",
			val:  struct{ Foo string }{Foo: "bar"},
			want: `{"foo": "bar"}`,
		},
		{
			name: "non-document type",
			val:  "foo: bar",
			want: `"foo: bar"`,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			value, err := MarshalValue(test.val, test.registry, test.encFn)

			assert.Equal(t, test.wantErr, err, "expected and actual error do not match")
			assert.Equal(t, test.want, value.String(), "expected and actual comments are different")
		})
	}
}
