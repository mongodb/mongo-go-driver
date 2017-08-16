package yamgo

import (
	"testing"
	"github.com/stretchr/testify/require"
	"fmt"
)

func createTestCollection(t *testing.T, dbName string, collName string) *Collection {
	db := createTestDatabase(t, dbName)

	return db.Collection(collName)
}

func TestCollection_initialize(t *testing.T) {
	t.Parallel()

	dbName := "foo"
	collName := "bar"

	coll := createTestCollection(t, dbName, collName)
	require.Equal(t, coll.name, collName)
	require.NotNil(t, coll.db)
}

func TestCollection_getNamespace(t *testing.T) {
	t.Parallel()

	dbName := "foo"
	collName := "bar"

	coll := createTestCollection(t, dbName, collName)
	require.Equal(t, coll.namespace(), fmt.Sprintf("%s.%s", dbName, collName))
}
