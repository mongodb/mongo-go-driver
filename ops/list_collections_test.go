package ops_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/internal/testconfig"
	. "github.com/10gen/mongo-go-driver/ops"
	"github.com/stretchr/testify/require"
)

func TestListCollectionsWithInvalidDatabaseName(t *testing.T) {
	t.Parallel()
	testconfig.Integration(t)

	s := getServer(t)
	_, err := ListCollections(context.Background(), s, "", ListCollectionsOptions{})
	require.Error(t, err)
}

func TestListCollections(t *testing.T) {
	t.Parallel()
	testconfig.Integration(t)
	dbname := testconfig.DBName(t)
	collectionNameOne := testconfig.ColName(t)
	collectionNameTwo := collectionNameOne + "2"
	collectionNameThree := collectionNameOne + "3"
	testconfig.DropCollection(t, dbname, collectionNameOne)
	testconfig.DropCollection(t, dbname, collectionNameTwo)
	testconfig.DropCollection(t, dbname, collectionNameThree)
	testconfig.InsertDocs(t, dbname, collectionNameOne, bson.D{{"_id", 1}})
	testconfig.InsertDocs(t, dbname, collectionNameTwo, bson.D{{"_id", 1}})
	testconfig.InsertDocs(t, dbname, collectionNameThree, bson.D{{"_id", 1}})

	s := getServer(t)
	cursor, err := ListCollections(context.Background(), s, dbname, ListCollectionsOptions{})
	require.NoError(t, err)
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
	t.Parallel()
	testconfig.Integration(t)
	dbname := testconfig.DBName(t)
	collectionNameOne := testconfig.ColName(t)
	collectionNameTwo := collectionNameOne + "2"
	collectionNameThree := collectionNameOne + "3"
	testconfig.DropCollection(t, dbname, collectionNameOne)
	testconfig.DropCollection(t, dbname, collectionNameTwo)
	testconfig.DropCollection(t, dbname, collectionNameThree)
	testconfig.InsertDocs(t, dbname, collectionNameOne, bson.D{{"_id", 1}})
	testconfig.InsertDocs(t, dbname, collectionNameTwo, bson.D{{"_id", 1}})
	testconfig.InsertDocs(t, dbname, collectionNameThree, bson.D{{"_id", 1}})

	s := getServer(t)
	cursor, err := ListCollections(context.Background(), s, dbname, ListCollectionsOptions{
		Filter:    bson.D{{"name", bson.RegEx{Pattern: fmt.Sprintf("^%s.*", collectionNameOne)}}},
		BatchSize: 2})
	require.NoError(t, err)

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
	t.Skip("max time is flaky on the server")
	t.Parallel()
	testconfig.Integration(t)

	s := getServer(t)

	if testconfig.EnableMaxTimeFailPoint(t, s) != nil {
		t.Skip("skipping maxTimeMS test when max time failpoint is disabled")
	}
	defer testconfig.DisableMaxTimeFailPoint(t, s)

	_, err := ListCollections(context.Background(), s, testconfig.DBName(t), ListCollectionsOptions{MaxTime: time.Millisecond})
	require.Error(t, err)

	// Hacky check for the error message.  Should we be returning a more structured error?
	require.Contains(t, err.Error(), "operation exceeded time limit")
}
