// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driver

import (
	"errors"
	"testing"
	"time"

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
			want time.Duration
		}{
			{
				name: "absent",
				doc:  bson.D{{"ok", 0}, {"code", 462}, {"errmsg", "overloaded"}},
				want: 0,
			},
			{
				name: "int32",
				doc:  bson.D{{"ok", 0}, {"code", 462}, {"errmsg", "overloaded"}, {"baseBackoffMS", int32(50)}},
				want: 50 * time.Millisecond,
			},
			{
				name: "int64",
				doc:  bson.D{{"ok", 0}, {"code", 462}, {"errmsg", "overloaded"}, {"baseBackoffMS", int64(50)}},
				want: 50 * time.Millisecond,
			},
			{
				name: "double",
				doc:  bson.D{{"ok", 0}, {"code", 462}, {"errmsg", "overloaded"}, {"baseBackoffMS", 50.0}},
				want: 50 * time.Millisecond,
			},
		}

		for _, test := range tests {
			test := test

			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				err := ExtractErrorFromServerResponse(marshal(t, test.doc))

				cerr, ok := err.(Error)
				require.Truef(t, ok, "expected an Error, got %T: %v", err, err)
				require.Equal(t, test.want, cerr.BaseBackoff)
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
		require.Equal(t, 50*time.Millisecond, wce.BaseBackoff)
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
		require.Equal(t, 50*time.Millisecond, wce.BaseBackoff)
	})
}

func TestError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  Error
		want string
	}{
		{
			name: "message only",
			err:  Error{Message: "msg"},
			want: "msg",
		},
		{
			name: "name and message",
			err:  Error{Name: "n", Message: "msg"},
			want: "(n) msg",
		},
		{
			name: "message and wrapped",
			err:  Error{Message: "msg", Wrapped: errors.New("w")},
			want: "msg: w",
		},
		{
			name: "name, message and wrapped",
			err:  Error{Name: "n", Message: "msg", Wrapped: errors.New("w")},
			want: "(n) msg: w",
		},
		{
			name: "name and wrapped without message",
			err:  Error{Name: "n", Wrapped: errors.New("w")},
			want: "(n) w",
		},
		{
			name: "wrapped only",
			err:  Error{Wrapped: errors.New("w")},
			want: "w",
		},
		{
			name: "name only",
			err:  Error{Name: "n"},
			want: "(n)",
		},
		{
			name: "wrapped identical to message",
			err:  Error{Message: "msg", Wrapped: errors.New("msg")},
			want: "msg",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, test.err.Error())
		})
	}
}

// TestWriteCommandErrorMessages asserts that the WriteCommandError error
// message reports every cause it has, including the write error details, and
// omits the causes it does not have. mongo.WriteException defers its error
// message to this one, so anything missing here is missing from the error
// message returned to users. See GODRIVER-4172.
func TestWriteCommandErrorMessages(t *testing.T) {
	t.Parallel()

	details, err := bson.Marshal(bson.D{{"details", bson.D{{"operatorName", "$jsonSchema"}}}})
	require.NoError(t, err, "unexpected error marshaling BSON")

	manyWriteErrors := make(WriteErrors, 0, 500)
	for i := 0; i < 500; i++ {
		manyWriteErrors = append(manyWriteErrors, WriteError{
			Index:   int64(i),
			Code:    11000,
			Message: "duplicate key error collection: db.coll index: _id_",
		})
	}

	tests := []struct {
		name string
		err  WriteCommandError
		want string
	}{
		{
			name: "no causes",
			err:  WriteCommandError{},
			want: "write command error: no causes",
		},
		{
			name: "write errors only",
			err: WriteCommandError{
				WriteErrors: WriteErrors{{Code: 11000, Message: "duplicate key"}},
			},
			want: "write command error: write errors: [duplicate key]",
		},
		{
			name: "write concern error only",
			err: WriteCommandError{
				WriteConcernError: &WriteConcernError{Name: "wce", Message: "write concern failed"},
			},
			want: "write command error: write concern error: (wce) write concern failed",
		},
		{
			name: "write errors and write concern error",
			err: WriteCommandError{
				WriteConcernError: &WriteConcernError{Message: "write concern failed"},
				WriteErrors:       WriteErrors{{Code: 11000, Message: "duplicate key"}},
			},
			want: "write command error: write concern error: write concern failed, write errors: [duplicate key]",
		},
		{
			name: "write error details",
			err: WriteCommandError{
				WriteErrors: WriteErrors{{
					Code:    121,
					Message: "Document failed validation",
					Details: bsoncore.Document(details),
				}},
			},
			want: `write command error: write errors: [Document failed validation: {"details": {"operatorName": "$jsonSchema"}}]`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, test.want, test.err.Error())
		})
	}

	t.Run("truncates large batches", func(t *testing.T) {
		t.Parallel()

		got := WriteCommandError{WriteErrors: manyWriteErrors}.Error()

		require.Less(t, len(got), 3000, "expected a truncated error message")
		require.Contains(t, got, "more errors...", "expected the truncation message")
	})
}
