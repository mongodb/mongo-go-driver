package ops_test

import (
	"context"
	"testing"

	. "github.com/10gen/mongo-go-driver/ops"
	"github.com/stretchr/testify/require"
	"gopkg.in/mgo.v2/bson"
)

func TestListIndexesWithInvalidDatabaseName(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	s := getServer()
	ns := Namespace{"space", "ex"}
	_, err := ListIndexes(context.Background(), s, ns, ListIndexesOptions{})
	require.NotNil(t, err)
}

func TestListIndexes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	s := getServer()

	collectionName := "TestListIndexes"
	ns := NewNamespace(databaseName, collectionName)

	indexNames := []string{"_id_", "a", "b", "c"}

	dropCollection(s, collectionName, t)

	createIndex(s, collectionName, []string{"a"}, t)
	createIndex(s, collectionName, []string{"b"}, t)
	createIndex(s, collectionName, []string{"c"}, t)

	cursor, err := ListIndexes(context.Background(), s, ns, ListIndexesOptions{})
	require.Nil(t, err)

	names := []string{}
	var next bson.M

	for cursor.Next(context.Background(), &next) {
		names = append(names, next["name"].(string))
	}

	require.Equal(t, 4, len(names))
	require.Contains(t, names, indexNames[0])
	require.Contains(t, names, indexNames[1])
	require.Contains(t, names, indexNames[2])
	require.Contains(t, names, indexNames[3])
}

func TestListIndexesMultipleBatches(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	s := getServer()

	collectionName := "TestListIndexes"

	indexNames := []string{"_id_", "a", "b", "c"}

	ns := NewNamespace(databaseName, collectionName)

	dropCollection(s, collectionName, t)

	createIndex(s, collectionName, []string{"a"}, t)
	createIndex(s, collectionName, []string{"b"}, t)
	createIndex(s, collectionName, []string{"c"}, t)

	options := ListIndexesOptions{BatchSize: 1}
	cursor, err := ListIndexes(context.Background(), s, ns, options)
	require.Nil(t, err)

	names := []string{}
	var next bson.M

	for cursor.Next(context.Background(), &next) {
		names = append(names, next["name"].(string))
	}

	require.Equal(t, 4, len(names))
	require.Contains(t, names, indexNames[0])
	require.Contains(t, names, indexNames[1])
	require.Contains(t, names, indexNames[2])
	require.Contains(t, names, indexNames[3])
}
