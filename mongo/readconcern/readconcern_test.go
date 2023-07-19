// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package readconcern_test

import (
	"testing"

	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func TestReadConcern_MarshalBSONValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		rc        *readconcern.ReadConcern
		bytes     []byte
		wantError error
	}{
		{
			name:      "local",
			rc:        &readconcern.ReadConcern{Level: "local"},
			bytes:     bsoncore.BuildDocument(nil, bsoncore.AppendStringElement(nil, "level", "local")),
			wantError: nil,
		},
		{
			name:      "empty",
			rc:        &readconcern.ReadConcern{},
			bytes:     bsoncore.BuildDocument(nil, nil),
			wantError: nil,
		},
		{
			name:      "nil",
			rc:        nil,
			bytes:     nil,
			wantError: readconcern.ErrEmptyReadConcern,
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable.

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, b, err := tc.rc.MarshalBSONValue()
			assert.Equal(t, tc.bytes, b, "expected and actual outputs do not match")
			assert.Equal(t, tc.wantError, err, "expected and actual errors do not match")
		})
	}
}
