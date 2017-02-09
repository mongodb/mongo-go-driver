package ops_test

import (
	. "github.com/10gen/mongo-go-driver/ops"
	"github.com/stretchr/testify/require"
	"gopkg.in/mgo.v2/bson"
	"testing"
)

func TestListDatabases(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	conn := getConnection()

	collectionName := "TestListDatabases"
	dropCollection(conn, collectionName, t)
	insertDocuments(conn, collectionName, []bson.D{{{"_id", 1}}}, t)

	cursor, err := ListDatabases(conn, &ListDatabasesOptions{})
	require.Nil(t, err)

	var next bson.M
	var found bool
	for cursor.Next(&next) {
		if next["name"] == databaseName {
			found = true
		}
	}
	require.True(t, found, "Expected to have listed at least database named %v", databaseName)
	require.Nil(t, cursor.Err())
	require.Nil(t, cursor.Close())
}

func TestListDatabasesWithMaxTimeMS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	conn := getConnection()

	enableMaxTimeFailPoint(conn, t)
	defer disableMaxTimeFailPoint(conn, t)

	_, err := ListDatabases(conn, &ListDatabasesOptions{
		MaxTimeMS: 1,
	})
	require.NotNil(t, err)
	// Hacky check for the error message.  Should we be returning a more structured error?
	require.Contains(t, err.Error(), "operation exceeded time limit")
}
