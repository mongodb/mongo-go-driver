// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops_test

import (
	"context"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/internal/testutil"
	. "github.com/mongodb/mongo-go-driver/mongo/private/ops"
	"github.com/stretchr/testify/require"
)

func TestCursorWithInvalidNamespace(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)

	s := getServer(t)
	var rdr bson.Reader
	var err error
	rdr, err = bson.NewDocument(
		bson.EC.SubDocumentFromElements(
			"cursor",
			bson.EC.String("ns", "foo"),
		)).
		MarshalBSON()
	require.NoError(t, err)

	_, err = NewCursor(rdr, 0, s)
	require.NotNil(t, err)
}

func TestCursorEmpty(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)
	testutil.AutoDropCollection(t)

	s := getServer(t)
	cursorResult := find(t, s, 0)

	subject, _ := NewCursor(cursorResult, 0, s)
	hasNext := subject.Next(context.Background())
	require.False(t, hasNext, "Empty cursor should not have next")
}

func TestCursorSingleBatch(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)
	testutil.AutoDropCollection(t)
	documents := []*bson.Document{
		bson.NewDocument(bson.EC.Int32("_id", 1)),
		bson.NewDocument(bson.EC.Int32("_id", 2)),
	}
	testutil.AutoInsertDocs(t, nil, documents...)

	readers := make([]bson.Reader, 0, len(documents))
	for _, doc := range documents {
		r, err := doc.MarshalBSON()
		require.NoError(t, err)
		readers = append(readers, r)
	}
	s := getServer(t)
	cursorResult := find(t, s, 0)
	subject, _ := NewCursor(cursorResult, 0, s)
	var hasNext bool
	var err error
	var next = make(bson.Reader, 1024)

	hasNext = subject.Next(context.Background())
	require.True(t, hasNext, "Should have result")
	err = subject.Decode(next)
	require.NoError(t, err)
	require.Equal(t, readers[0], next[:len(readers[0])], "Documents should be equal")

	hasNext = subject.Next(context.Background())
	require.True(t, hasNext, "Should have result")
	err = subject.Decode(next)
	require.NoError(t, err)
	require.Equal(t, readers[1], next[:len(readers[1])], "Documents should be equal")

	hasNext = subject.Next(context.Background())
	require.False(t, hasNext, "Should be exhausted")
}

func TestCursorMultipleBatches(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)
	testutil.AutoDropCollection(t)
	documents := []*bson.Document{
		bson.NewDocument(bson.EC.Int32("_id", 1)),
		bson.NewDocument(bson.EC.Int32("_id", 2)),
		bson.NewDocument(bson.EC.Int32("_id", 3)),
		bson.NewDocument(bson.EC.Int32("_id", 4)),
		bson.NewDocument(bson.EC.Int32("_id", 5)),
	}
	testutil.AutoInsertDocs(t, nil, documents...)

	readers := make([]bson.Reader, 0, len(documents))
	for _, doc := range documents {
		r, err := doc.MarshalBSON()
		require.NoError(t, err)
		readers = append(readers, r)
	}

	s := getServer(t)
	cursorResult := find(t, s, 2)
	subject, _ := NewCursor(cursorResult, 2, s)
	var hasNext bool
	var err error
	var next = make(bson.Reader, 1024)

	hasNext = subject.Next(context.Background())
	require.True(t, hasNext, "Should have result")
	err = subject.Decode(next)
	require.NoError(t, err)
	require.Equal(t, readers[0], next[:len(readers[0])], "Documents should be equal")

	hasNext = subject.Next(context.Background())
	require.True(t, hasNext, "Should have result")
	err = subject.Decode(next)
	require.NoError(t, err)
	require.Equal(t, readers[1], next[:len(readers[1])], "Documents should be equal")

	hasNext = subject.Next(context.Background())
	require.True(t, hasNext, "Should have result")
	err = subject.Decode(next)
	require.NoError(t, err)
	require.Equal(t, readers[2], next[:len(readers[2])], "Documents should be equal")

	hasNext = subject.Next(context.Background())
	require.True(t, hasNext, "Should have result")
	err = subject.Decode(next)
	require.NoError(t, err)
	require.Equal(t, readers[3], next[:len(readers[3])], "Documents should be equal")

	hasNext = subject.Next(context.Background())
	require.True(t, hasNext, "Should have result")
	err = subject.Decode(next)
	require.NoError(t, err)
	require.Equal(t, readers[4], next[:len(readers[4])], "Documents should be equal")

	hasNext = subject.Next(context.Background())
	require.False(t, hasNext, "Should be exhausted")
}

func TestCursorClose(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)
	testutil.AutoDropCollection(t)
	documents := []*bson.Document{
		bson.NewDocument(bson.EC.Int32("_id", 1)),
		bson.NewDocument(bson.EC.Int32("_id", 2)),
		bson.NewDocument(bson.EC.Int32("_id", 3)),
		bson.NewDocument(bson.EC.Int32("_id", 4)),
		bson.NewDocument(bson.EC.Int32("_id", 5)),
	}
	testutil.AutoInsertDocs(t, nil, documents...)

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
		nil,
		bson.NewDocument(bson.EC.Int32("_id", 1)),
		bson.NewDocument(bson.EC.Int32("_id", 2)),
		bson.NewDocument(bson.EC.Int32("_id", 3)),
		bson.NewDocument(bson.EC.Int32("_id", 4)),
		bson.NewDocument(bson.EC.Int32("_id", 5)),
	)

	s := getServer(t)
	cursorResult := find(t, s, 2)
	subject, _ := NewCursor(cursorResult, 2, s)
	var hasNext bool
	var err error
	var next = make(bson.Reader, 2)

	// unmarshalling into a non-pointer struct should fail
	hasNext = subject.Next(context.Background())
	require.True(t, hasNext)
	err = subject.Decode(next)
	require.Error(t, err)
}
