// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/require"
)

func TestErrorMessages(t *testing.T) {
	details, err := bson.Marshal(bson.D{{"details", bson.D{{"operatorName", "$jsonSchema"}}}})
	require.Nil(t, err, "unexpected error marshaling BSON")

	cases := []struct {
		desc     string
		err      error
		expected string
	}{
		{
			desc: "WriteException error message should contain the WriteError Message and Details",
			err: WriteException{
				WriteErrors: WriteErrors{
					{
						Message: "test message 1",
						Details: details,
					},
					{
						Message: "test message 2",
						Details: details,
					},
				},
			},
			expected: `write exception: write errors: [test message 1: {"details": {"operatorName": "$jsonSchema"}}, test message 2: {"details": {"operatorName": "$jsonSchema"}}]`,
		},
		{
			desc: "BulkWriteException error message should contain the WriteError Message and Details",
			err: BulkWriteException{
				WriteErrors: []BulkWriteError{
					{
						WriteError: WriteError{
							Message: "test message 1",
							Details: details,
						},
					},
					{
						WriteError: WriteError{
							Message: "test message 2",
							Details: details,
						},
					},
				},
			},
			expected: `bulk write exception: write errors: [test message 1: {"details": {"operatorName": "$jsonSchema"}}, test message 2: {"details": {"operatorName": "$jsonSchema"}}]`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, tc.err.Error())
		})
	}
}
