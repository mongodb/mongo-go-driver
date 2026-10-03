// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
)

// commandKeys builds the aggregate command for op and returns its top-level
// field names in wire order.
func commandKeys(t *testing.T, op aggregateOp) []string {
	t.Helper()

	dst, err := op.command(nil, description.SelectedServer{})
	require.NoError(t, err, "command error: %v", err)

	elems, err := bson.Raw(bsoncore.BuildDocument(nil, dst)).Elements()
	require.NoError(t, err, "Elements error: %v", err)

	keys := make([]string, 0, len(elems))
	for _, elem := range elems {
		keys = append(keys, elem.Key())
	}

	return keys
}

// TestAggregateOpAdditionalCmd asserts that additionalCmd elements are spliced
// into the aggregate command verbatim: in the caller's order, preserving
// duplicate keys, and positioned after the operation's own fields but before
// the "cursor" subdocument.
func TestAggregateOpAdditionalCmd(t *testing.T) {
	t.Run("preserves order", func(t *testing.T) {
		keys := commandKeys(t, aggregateOp{
			collection: "coll",
			additionalCmd: bson.D{
				{Key: "b", Value: int32(1)},
				{Key: "a", Value: int32(2)},
				{Key: "d", Value: int32(3)},
				{Key: "c", Value: int32(4)},
			},
		})

		want := []string{"aggregate", "b", "a", "d", "c", "cursor"}
		require.Equal(t, want, keys, "expected command fields %v, got %v", want, keys)
	})

	t.Run("preserves duplicate keys", func(t *testing.T) {
		keys := commandKeys(t, aggregateOp{
			collection: "coll",
			additionalCmd: bson.D{
				{Key: "a", Value: int32(1)},
				{Key: "a", Value: int32(2)},
			},
		})

		want := []string{"aggregate", "a", "a", "cursor"}
		require.Equal(t, want, keys, "expected command fields %v, got %v", want, keys)
	})
}
