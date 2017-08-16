package yamgo

import (
	"testing"

	"github.com/10gen/mongo-go-driver/internal/testconfig"
	"github.com/stretchr/testify/require"
)

func createTestClient(t *testing.T) *Client {
	return &Client{
		cluster:    testconfig.Cluster(t),
		connString: testconfig.ConnString(t),
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
