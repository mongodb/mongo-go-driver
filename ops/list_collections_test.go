package ops_test

import (
	"context"
	"testing"
	"time"

	. "github.com/10gen/mongo-go-driver/ops"
	"github.com/stretchr/testify/require"
	"gopkg.in/mgo.v2/bson"
)

func TestListCollectionsWithInvalidDatabaseName(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	s := getServer()
	_, err := ListCollections(context.Background(), s, "", ListCollectionsOptions{})
	require.NotNil(t, err)
}

func TestListCollections(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	s := getServer()

	collectionNameOne := "TestListCollections1"
	collectionNameTwo := "TestListCollections2"
	collectionNameThree := "TestListCollections3"

	dropCollection(s, collectionNameOne, t)
	dropCollection(s, collectionNameTwo, t)
	dropCollection(s, collectionNameThree, t)

	insertDocuments(s, collectionNameOne, []bson.D{{{"_id", 1}}}, t)
	insertDocuments(s, collectionNameTwo, []bson.D{{{"_id", 1}}}, t)
	insertDocuments(s, collectionNameThree, []bson.D{{{"_id", 1}}}, t)

	cursor, err := ListCollections(context.Background(), s, databaseName, ListCollectionsOptions{})
	require.Nil(t, err)

	names := []string{}
	var next bson.M

	for cursor.Next(context.Background(), &next) {
		names = append(names, next["name"].(string))
	}

	require.Contains(t, names, collectionNameOne)
	require.Contains(t, names, collectionNameTwo)
	require.Contains(t, names, collectionNameThree)
}

func TestListCollectionsMultipleBatches(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	s := getServer()

	collectionNameOne := "TestListCollectionsMultipleBatches1"
	collectionNameTwo := "TestListCollectionsMultipleBatches2"
	collectionNameThree := "TestListCollectionsMultipleBatches3"

	dropCollection(s, collectionNameOne, t)
	dropCollection(s, collectionNameTwo, t)
	dropCollection(s, collectionNameThree, t)

	insertDocuments(s, collectionNameOne, []bson.D{{{"_id", 1}}}, t)
	insertDocuments(s, collectionNameTwo, []bson.D{{{"_id", 1}}}, t)
	insertDocuments(s, collectionNameThree, []bson.D{{{"_id", 1}}}, t)

	cursor, err := ListCollections(context.Background(), s, databaseName, ListCollectionsOptions{
		Filter:    bson.D{{"name", bson.RegEx{Pattern: "^TestListCollectionsMultipleBatches.*"}}},
		BatchSize: 2})
	require.Nil(t, err)

	names := []string{}
	var next bson.M

	for cursor.Next(context.Background(), &next) {
		names = append(names, next["name"].(string))
	}

	require.Equal(t, 3, len(names))
	require.Contains(t, names, collectionNameOne)
	require.Contains(t, names, collectionNameTwo)
	require.Contains(t, names, collectionNameThree)
}

func TestListCollectionsWithMaxTimeMS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	s := getServer()

	if enableMaxTimeFailPoint(s, t) != nil {
		t.Skip("skipping maxTimeMS test when max time failpoint is disabled")
	}
	defer disableMaxTimeFailPoint(s, t)

	_, err := ListCollections(context.Background(), s, databaseName, ListCollectionsOptions{MaxTime: time.Millisecond})
	require.NotNil(t, err)

	// Hacky check for the error message.  Should we be returning a more structured error?
	require.Contains(t, err.Error(), "operation exceeded time limit")
}
