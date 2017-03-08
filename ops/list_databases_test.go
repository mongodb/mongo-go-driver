package ops_test

import (
	"context"
	"testing"
	"time"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/internal/testconfig"
	. "github.com/10gen/mongo-go-driver/ops"
	"github.com/stretchr/testify/require"
)

func TestListDatabases(t *testing.T) {
	t.Parallel()
	testconfig.Integration(t)
	testconfig.AutoDropCollection(t)
	testconfig.AutoInsertDocs(t, bson.D{{"_id", 1}})

	s := getServer(t)
	cursor, err := ListDatabases(context.Background(), s, ListDatabasesOptions{})
	require.NoError(t, err)

	var next bson.M
	var found bool
	for cursor.Next(context.Background(), &next) {
		if next["name"] == testconfig.DBName(t) {
			found = true
			break
		}
	}
	require.True(t, found, "Expected to have listed at least database named %v", testconfig.DBName(t))
	require.NoError(t, cursor.Err())
	require.NoError(t, cursor.Close(context.Background()))
}

func TestListDatabasesWithMaxTimeMS(t *testing.T) {
	t.Parallel()
	testconfig.Integration(t)

	s := getServer(t)

	if testconfig.EnableMaxTimeFailPoint(t, s) != nil {
		t.Skip("skipping maxTimeMS test when max time failpoint is disabled")
	}
	defer testconfig.DisableMaxTimeFailPoint(t, s)

	_, err := ListDatabases(context.Background(), s, ListDatabasesOptions{
		MaxTime: time.Millisecond,
	})
	require.Error(t, err)
	// Hacky check for the error message.  Should we be returning a more structured error?
	require.Contains(t, err.Error(), "operation exceeded time limit")
}
