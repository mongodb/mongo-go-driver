// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driver

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

func TestExtractErrorFromServerResponse_BaseBackoffMS(t *testing.T) {
	t.Parallel()

	marshal := func(t *testing.T, doc bson.D) bsoncore.Document {
		t.Helper()

		raw, err := bson.Marshal(doc)
		require.NoError(t, err)

		return bsoncore.Document(raw)
	}

	t.Run("command error", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			doc  bson.D
			want int64
		}{
			{
				name: "absent",
				doc:  bson.D{{"ok", 0}, {"code", 462}, {"errmsg", "overloaded"}},
				want: 0,
			},
			{
				name: "int32",
				doc:  bson.D{{"ok", 0}, {"code", 462}, {"errmsg", "overloaded"}, {"baseBackoffMS", int32(50)}},
				want: 50,
			},
			{
				name: "int64",
				doc:  bson.D{{"ok", 0}, {"code", 462}, {"errmsg", "overloaded"}, {"baseBackoffMS", int64(50)}},
				want: 50,
			},
			{
				name: "double",
				doc:  bson.D{{"ok", 0}, {"code", 462}, {"errmsg", "overloaded"}, {"baseBackoffMS", 50.0}},
				want: 50,
			},
		}

		for _, test := range tests {
			test := test

			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				err := ExtractErrorFromServerResponse(marshal(t, test.doc))

				cerr, ok := err.(Error)
				require.Truef(t, ok, "expected an Error, got %T: %v", err, err)
				require.Equal(t, test.want, cerr.BaseBackoffMS)
			})
		}
	})

	t.Run("write command error", func(t *testing.T) {
		t.Parallel()

		doc := bson.D{
			{"ok", 1},
			{"baseBackoffMS", int32(50)},
			{"writeErrors", bson.A{bson.D{{"index", 0}, {"code", 462}, {"errmsg", "overloaded"}}}},
		}

		err := ExtractErrorFromServerResponse(marshal(t, doc))

		wce, ok := err.(WriteCommandError)
		require.Truef(t, ok, "expected a WriteCommandError, got %T: %v", err, err)
		require.Equal(t, int64(50), wce.BaseBackoffMS)
	})

	t.Run("write concern error", func(t *testing.T) {
		t.Parallel()

		doc := bson.D{
			{"ok", 1},
			{"writeConcernError", bson.D{
				{"code", 462},
				{"errmsg", "overloaded"},
				{"baseBackoffMS", int32(50)},
			}},
		}

		err := ExtractErrorFromServerResponse(marshal(t, doc))

		wce, ok := err.(WriteCommandError)
		require.Truef(t, ok, "expected a WriteCommandError, got %T: %v", err, err)
		require.Equal(t, int64(50), wce.BaseBackoffMS)
	})
}
