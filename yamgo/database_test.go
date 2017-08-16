package yamgo

import (
	"context"
	"testing"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/stretchr/testify/require"
)

func createTestDatabase(t *testing.T, name string) *Database {
	client := createTestClient(t)
	return client.Database(name)
}

func TestDatabase_initialize(t *testing.T) {
	t.Parallel()

	name := "foo"

	db := createTestDatabase(t, name)
	require.Equal(t, db.name, name)
	require.NotNil(t, db.client)
}

func TestDatabase_simpleCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	name := "foo"

	var result bson.D
	db := createTestDatabase(t, name)
	err := db.RunCommand(context.Background(), nil, bson.D{{"isMaster", 1}}, &result)
	require.NoError(t, err)
}
