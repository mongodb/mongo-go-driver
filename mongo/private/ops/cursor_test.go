// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops_test

import (
	"context"
	"testing"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/mongo/internal/testutil"
	. "github.com/10gen/mongo-go-driver/mongo/private/ops"
	"github.com/stretchr/testify/require"
)

func TestCursorWithInvalidNamespace(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)

	s := getServer(t)
	_, err := NewCursor(&firstBatchCursorResult{
		NS: "foo",
	}, 0, s)
	require.NotNil(t, err)
}

func TestCursorEmpty(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)
	testutil.AutoDropCollection(t)

	s := getServer(t)
	cursorResult := find(t, s, 0)

	subject, _ := NewCursor(cursorResult, 0, s)
	hasNext := subject.Next(context.Background(), &bson.D{})
	require.False(t, hasNext, "Empty cursor should not have next")
}

func TestCursorSingleBatch(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)
	testutil.AutoDropCollection(t)
	documents := []bson.D{
		{bson.NewDocElem("_id", 1)},
		{bson.NewDocElem("_id", 2)},
	}
	testutil.AutoInsertDocs(t, documents...)

	s := getServer(t)
	cursorResult := find(t, s, 0)
	subject, _ := NewCursor(cursorResult, 0, s)
	var next bson.D
	var hasNext bool

	hasNext = subject.Next(context.Background(), &next)
	require.True(t, hasNext, "Should have result")
	require.Equal(t, documents[0], next, "Documents should be equal")

	hasNext = subject.Next(context.Background(), &next)
	require.True(t, hasNext, "Should have result")
	require.Equal(t, documents[1], next, "Documents should be equal")

	hasNext = subject.Next(context.Background(), &next)
	require.False(t, hasNext, "Should be exhausted")
}

func TestCursorMultipleBatches(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)
	testutil.AutoDropCollection(t)
	documents := []bson.D{
		{bson.NewDocElem("_id", 1)},
		{bson.NewDocElem("_id", 2)},
		{bson.NewDocElem("_id", 3)},
		{bson.NewDocElem("_id", 4)},
		{bson.NewDocElem("_id", 5)},
	}
	testutil.AutoInsertDocs(t, documents...)

	s := getServer(t)
	cursorResult := find(t, s, 2)
	subject, _ := NewCursor(cursorResult, 2, s)
	var next bson.D
	var hasNext bool

	hasNext = subject.Next(context.Background(), &next)
	require.True(t, hasNext, "Should have result")
	require.Equal(t, documents[0], next, "Documents should be equal")

	hasNext = subject.Next(context.Background(), &next)
	require.True(t, hasNext, "Should have result")
	require.Equal(t, documents[1], next, "Documents should be equal")

	hasNext = subject.Next(context.Background(), &next)
	require.True(t, hasNext, "Should have result")
	require.Equal(t, documents[2], next, "Documents should be equal")

	hasNext = subject.Next(context.Background(), &next)
	require.True(t, hasNext, "Should have result")
	require.Equal(t, documents[3], next, "Documents should be equal")

	hasNext = subject.Next(context.Background(), &next)
	require.True(t, hasNext, "Should have result")
	require.Equal(t, documents[4], next, "Documents should be equal")

	hasNext = subject.Next(context.Background(), &next)
	require.False(t, hasNext, "Should be exhausted")
}

func TestCursorClose(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)
	testutil.AutoDropCollection(t)
	documents := []bson.D{
		{bson.NewDocElem("_id", 1)},
		{bson.NewDocElem("_id", 2)},
		{bson.NewDocElem("_id", 3)},
		{bson.NewDocElem("_id", 4)},
		{bson.NewDocElem("_id", 5)},
	}
	testutil.AutoInsertDocs(t, documents...)

	s := getServer(t)
	cursorResult := find(t, s, 2)
	subject, _ := NewCursor(cursorResult, 2, s)
	err := subject.Close(context.Background())
	require.NoError(t, err)

	// call it again
	err = subject.Close(context.Background())
	require.NoError(t, err)
}

func TestCursorError(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)
	testutil.AutoDropCollection(t)
	testutil.AutoInsertDocs(t,
		bson.D{bson.NewDocElem("_id", 1)},
		bson.D{bson.NewDocElem("_id", 2)},
		bson.D{bson.NewDocElem("_id", 3)},
		bson.D{bson.NewDocElem("_id", 4)},
		bson.D{bson.NewDocElem("_id", 5)},
	)

	s := getServer(t)
	cursorResult := find(t, s, 2)
	subject, _ := NewCursor(cursorResult, 2, s)
	var next bson.D
	var hasNext bool

	// unmarshalling into a non-pointer struct should fail
	hasNext = subject.Next(context.Background(), next)
	require.Error(t, subject.Err())
	require.False(t, hasNext, "Should not have result")
}
