package cluster_test

import (
	"testing"

	. "github.com/10gen/mongo-go-driver/cluster"
	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/server"
	"github.com/stretchr/testify/require"
)

func TestWriteSelector_ReplicaSetWithPrimary(t *testing.T) {
	t.Parallel()

	require := require.New(t)

	var c = &Desc{
		Type: ReplicaSetWithPrimary,
		Servers: []*server.Desc{
			&server.Desc{
				Endpoint: conn.Endpoint("localhost:27017"),
				Type:     server.RSPrimary,
			},
			&server.Desc{
				Endpoint: conn.Endpoint("localhost:27018"),
				Type:     server.RSSecondary,
			},
			&server.Desc{
				Endpoint: conn.Endpoint("localhost:27018"),
				Type:     server.RSSecondary,
			},
		},
	}

	result, err := WriteSelector()(c, c.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*server.Desc{c.Servers[0]}, result)
}

func TestWriteSelector_ReplicaSetNoPrimary(t *testing.T) {
	t.Parallel()

	require := require.New(t)

	var c = &Desc{
		Type: ReplicaSetNoPrimary,
		Servers: []*server.Desc{
			&server.Desc{
				Endpoint: conn.Endpoint("localhost:27018"),
				Type:     server.RSSecondary,
			},
			&server.Desc{
				Endpoint: conn.Endpoint("localhost:27018"),
				Type:     server.RSSecondary,
			},
		},
	}

	result, err := WriteSelector()(c, c.Servers)

	require.NoError(err)
	require.Len(result, 0)
	require.Empty(result)
}

func TestWriteSelector_Sharded(t *testing.T) {
	t.Parallel()

	require := require.New(t)

	var c = &Desc{
		Type: Sharded,
		Servers: []*server.Desc{
			&server.Desc{
				Endpoint: conn.Endpoint("localhost:27018"),
				Type:     server.Mongos,
			},
			&server.Desc{
				Endpoint: conn.Endpoint("localhost:27018"),
				Type:     server.Mongos,
			},
		},
	}

	result, err := WriteSelector()(c, c.Servers)

	require.NoError(err)
	require.Len(result, 2)
	require.Equal([]*server.Desc{c.Servers[0], c.Servers[1]}, result)
}

func TestWriteSelector_Single(t *testing.T) {
	t.Parallel()

	require := require.New(t)

	var c = &Desc{
		Type: Single,
		Servers: []*server.Desc{
			&server.Desc{
				Endpoint: conn.Endpoint("localhost:27018"),
				Type:     server.Standalone,
			},
		},
	}

	result, err := WriteSelector()(c, c.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*server.Desc{c.Servers[0]}, result)
}
