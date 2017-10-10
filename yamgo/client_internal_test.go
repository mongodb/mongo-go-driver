package yamgo

import (
	"testing"

	"github.com/10gen/mongo-go-driver/yamgo/internal/testutil"
	"github.com/10gen/mongo-go-driver/yamgo/readpref"
	"github.com/stretchr/testify/require"
)

func createTestClient(t *testing.T) *Client {
	return &Client{
		cluster:        testutil.Cluster(t),
		connString:     testutil.ConnString(t),
		readPreference: readpref.Primary(),
	}
}

func TestNewClient(t *testing.T) {
	t.Parallel()

	c := createTestClient(t)
	require.NotNil(t, c.cluster)
}

func TestClient_Database(t *testing.T) {
	t.Parallel()

	dbName := "foo"

	c := createTestClient(t)
	db := c.Database(dbName)
	require.Equal(t, db.Name(), dbName)
	require.Exactly(t, c, db.Client())
}
