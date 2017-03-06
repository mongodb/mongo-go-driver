package ops_test

import (
	"context"
	"testing"

	"github.com/10gen/mongo-go-driver/bson"
	. "github.com/10gen/mongo-go-driver/ops"
	"github.com/stretchr/testify/require"
)

func TestListIndexesWithInvalidDatabaseName(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	s := getServer()
	ns := Namespace{"space", "ex"}
	cursor, err := ListIndexes(context.Background(), s, ns, ListIndexesOptions{})
	require.Nil(t, err)

	indexes := []string{}
	var next bson.M

	for cursor.Next(context.Background(), &next) {
		indexes = append(indexes, next["name"].(string))
	}

	require.Equal(t, 0, len(indexes))
}

func TestListIndexesWithInvalidCollectionName(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	s := getServer()
	ns := Namespace{databaseName, "ex"}
	cursor, err := ListIndexes(context.Background(), s, ns, ListIndexesOptions{})
	require.Nil(t, err)

	indexes := []string{}
	var next bson.M

	for cursor.Next(context.Background(), &next) {
		indexes = append(indexes, next["name"].(string))
	}

	require.Equal(t, 0, len(indexes))
}

func TestListIndexes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	s := getServer()

	collectionName := "TestListIndexes"
	ns := NewNamespace(databaseName, collectionName)

	indexNames := []string{"_id_", "a", "b", "c", "d_e"}

	dropCollection(s, collectionName, t)

	createIndex(s, collectionName, []string{"a"}, t)
	createIndex(s, collectionName, []string{"b"}, t)
	createIndex(s, collectionName, []string{"c"}, t)
	createIndex(s, collectionName, []string{"d", "e"}, t)

	cursor, err := ListIndexes(context.Background(), s, ns, ListIndexesOptions{})
	require.Nil(t, err)

	indexes := []string{}
	var next bson.M

	for cursor.Next(context.Background(), &next) {
		indexes = append(indexes, next["name"].(string))
	}

	require.Equal(t, 5, len(indexes))
	require.Contains(t, indexes, indexNames[0])
	require.Contains(t, indexes, indexNames[1])
	require.Contains(t, indexes, indexNames[2])
	require.Contains(t, indexes, indexNames[3])
	require.Contains(t, indexes, indexNames[4])
}

func TestListIndexesMultipleBatches(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	s := getServer()

	collectionName := "TestListIndexesMultipleBatches"

	indexNames := []string{"_id_", "a", "b", "c"}

	ns := NewNamespace(databaseName, collectionName)

	dropCollection(s, collectionName, t)

	createIndex(s, collectionName, []string{"a"}, t)
	createIndex(s, collectionName, []string{"b"}, t)
	createIndex(s, collectionName, []string{"c"}, t)

	options := ListIndexesOptions{BatchSize: 1}
	cursor, err := ListIndexes(context.Background(), s, ns, options)
	require.Nil(t, err)

	indexes := []string{}
	var next bson.M

	for cursor.Next(context.Background(), &next) {
		indexes = append(indexes, next["name"].(string))
	}

	require.Equal(t, 4, len(indexes))
	require.Contains(t, indexes, indexNames[0])
	require.Contains(t, indexes, indexNames[1])
	require.Contains(t, indexes, indexNames[2])
	require.Contains(t, indexes, indexNames[3])
}
