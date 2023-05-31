// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package internal

import (
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/assert"
)

func TestCommentToBSONCoreValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		comment interface{}
		want    string
		err     error
	}{
		{
			name:    "empty",
			comment: nil,
			want:    "",
			err:     ErrNilValue{},
		},
		{
			name:    "bson.D",
			comment: bson.D{{"foo", "bar"}},
			want:    `{"foo": "bar"}`,
		},
		{
			name:    "map",
			comment: map[string]interface{}{"foo": "bar"},
			want:    `{"foo": "bar"}`,
		},
		{
			name:    "struct",
			comment: struct{ Foo string }{Foo: "bar"},
			want:    `{"foo": "bar"}`,
		},
		{
			name:    "non-document type",
			comment: "foo: bar",
			want:    `"foo: bar"`,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			value, err := NewBSONValue(nil, test.comment, true, "comment")
			if !errors.Is(err, test.err) {
				t.Fatalf("failed to convert comment to bsoncore.Value: %v", err)
			}

			got := value.String()
			assert.Equal(t, test.want, got, "expected and actual comments are different")
		})
	}
}
