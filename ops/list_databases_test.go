package ops_test

import (
	"context"
	"testing"
	"time"

	. "github.com/10gen/mongo-go-driver/ops"
	"github.com/stretchr/testify/require"
	"gopkg.in/mgo.v2/bson"
)

func TestListDatabases(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	s := getServer()

	collectionName := "TestListDatabases"
	dropCollection(s, collectionName, t)
	insertDocuments(s, collectionName, []bson.D{{{"_id", 1}}}, t)

	cursor, err := ListDatabases(context.Background(), s, ListDatabasesOptions{})
	require.Nil(t, err)

	var next bson.M
	var found bool
	for cursor.Next(context.Background(), &next) {
		if next["name"] == databaseName {
			found = true
		}
	}
	require.True(t, found, "Expected to have listed at least database named %v", databaseName)
	require.Nil(t, cursor.Err())
	require.Nil(t, cursor.Close(context.Background()))
}

func TestListDatabasesWithMaxTimeMS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	s := getServer()

	if enableMaxTimeFailPoint(s, t) != nil {
		t.Skip("skipping maxTimeMS test when max time failpoint is disabled")
	}
	defer disableMaxTimeFailPoint(s, t)

	_, err := ListDatabases(context.Background(), s, ListDatabasesOptions{
		MaxTime: time.Millisecond,
	})
	require.NotNil(t, err)
	// Hacky check for the error message.  Should we be returning a more structured error?
	require.Contains(t, err.Error(), "operation exceeded time limit")
}
