package ops_test

import (
	"context"
	"testing"

	. "github.com/10gen/mongo-go-driver/ops"
	"github.com/stretchr/testify/require"
	"gopkg.in/mgo.v2/bson"
)

func TestCursorWithInvalidNamespace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	s := getServer()

	_, err := NewCursor(&firstBatchCursorResult{
		NS: "foo",
	}, 0, s)
	require.NotNil(t, err)
}

func TestCursorEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	s := getServer()

	collectionName := "TestCursorEmpty"
	dropCollection(s, collectionName, t)

	cursorResult := find(s, collectionName, 0, t)

	subject, _ := NewCursor(cursorResult, 0, s)
	hasNext := subject.Next(context.Background(), &bson.D{})
	require.False(t, hasNext, "Empty cursor should not have next")
}

func TestCursorSingleBatch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	s := getServer()

	collectionName := "TestCursorSingleBatch"
	dropCollection(s, collectionName, t)
	documents := []bson.D{{{"_id", 1}}, {{"_id", 2}}}
	insertDocuments(s, collectionName, documents, t)

	cursorResult := find(s, collectionName, 0, t)

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
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	s := getServer()

	collectionName := "TestCursorMultipleBatches"
	dropCollection(s, collectionName, t)
	documents := []bson.D{{{"_id", 1}}, {{"_id", 2}}, {{"_id", 3}}, {{"_id", 4}}, {{"_id", 5}}}
	insertDocuments(s, collectionName, documents, t)

	cursorResult := find(s, collectionName, 2, t)

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
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	s := getServer()

	collectionName := "TestCursorClose"
	dropCollection(s, collectionName, t)
	documents := []bson.D{{{"_id", 1}}, {{"_id", 2}}, {{"_id", 3}}, {{"_id", 4}}, {{"_id", 5}}}
	insertDocuments(s, collectionName, documents, t)

	cursorResult := find(s, collectionName, 2, t)

	subject, _ := NewCursor(cursorResult, 2, s)
	err := subject.Close(context.Background())
	require.Nil(t, err, "Unexpected error")

	// call it again
	err = subject.Close(context.Background())
	require.Nil(t, err, "Unexpected error")
}

func TestCursorError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	s := getServer()

	collectionName := "TestCursorError"
	dropCollection(s, collectionName, t)
	documents := []bson.D{{{"_id", 1}}, {{"_id", 2}}, {{"_id", 3}}, {{"_id", 4}}, {{"_id", 5}}}
	insertDocuments(s, collectionName, documents, t)

	cursorResult := find(s, collectionName, 2, t)

	subject, _ := NewCursor(cursorResult, 2, s)
	var next bson.D
	var hasNext bool

	// unmarshalling into a non-pointer struct should fail
	hasNext = subject.Next(context.Background(), next)
	require.NotNil(t, subject.Err(), "Unexpected error")
	require.False(t, hasNext, "Should not have result")
}
