// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
)

func newTestGridFSDownloadStream(t *testing.T, chunk bson.D) *GridFSDownloadStream {
	t.Helper()

	cursor, err := NewCursorFromDocuments([]any{chunk}, nil, nil)
	require.NoError(t, err, "NewCursorFromDocuments error")

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	return newGridFSDownloadStream(ctx, cancel, cursor, 4, &GridFSFile{Length: 4})
}

func TestGridFSDownloadStream_malformedChunk(t *testing.T) {
	data := bson.Binary{Data: []byte("abcd")}

	testCases := []struct {
		name    string
		chunk   bson.D
		wantErr string
	}{
		{
			name:    "n is not an integer",
			chunk:   bson.D{{"n", 0.0}, {"data", data}},
			wantErr: "chunk 'n' field has unexpected BSON type double",
		},
		{
			name:    "data is not binary",
			chunk:   bson.D{{"n", int32(0)}, {"data", "abcd"}},
			wantErr: "chunk 'data' field has unexpected BSON type string",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name+": Read", func(t *testing.T) {
			ds := newTestGridFSDownloadStream(t, tc.chunk)

			_, err := ds.Read(make([]byte, 4))
			assert.ErrorContains(t, err, tc.wantErr)
		})

		t.Run(tc.name+": Skip", func(t *testing.T) {
			ds := newTestGridFSDownloadStream(t, tc.chunk)

			_, err := ds.Skip(4)
			assert.ErrorContains(t, err, tc.wantErr)
		})
	}

	t.Run("well-formed chunk", func(t *testing.T) {
		ds := newTestGridFSDownloadStream(t, bson.D{{"n", int32(0)}, {"data", data}})

		buf := make([]byte, 4)
		n, err := ds.Read(buf)
		require.NoError(t, err, "Read error")
		assert.Equal(t, "abcd", string(buf[:n]))
	})
}
