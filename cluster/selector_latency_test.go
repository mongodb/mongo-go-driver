package cluster_test

import (
	"testing"
	"time"

	. "github.com/10gen/mongo-go-driver/cluster"
	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/server"
	"github.com/stretchr/testify/require"
)

func TestLatencySelector_NoRTTSet(t *testing.T) {
	t.Parallel()

	require := require.New(t)

	var c = &Desc{
		Servers: []*server.Desc{
			&server.Desc{
				Endpoint: conn.Endpoint("localhost:27017"),
			},
			&server.Desc{
				Endpoint: conn.Endpoint("localhost:27018"),
			},
			&server.Desc{
				Endpoint: conn.Endpoint("localhost:27018"),
			},
		},
	}

	result, err := LatencySelector(time.Duration(20)*time.Second)(c, c.Servers)

	require.NoError(err)
	require.Len(result, 3)
}

func TestLatencySelector_MultipleServers_PartialNoRTTSet(t *testing.T) {
	t.Parallel()

	require := require.New(t)

	var c = &Desc{
		Servers: []*server.Desc{
			&server.Desc{
				Endpoint:      conn.Endpoint("localhost:27017"),
				AverageRTT:    time.Duration(5) * time.Second,
				AverageRTTSet: true,
			},
			&server.Desc{
				Endpoint: conn.Endpoint("localhost:27018"),
			},
			&server.Desc{
				Endpoint:      conn.Endpoint("localhost:27018"),
				AverageRTT:    time.Duration(10) * time.Second,
				AverageRTTSet: true,
			},
		},
	}

	result, err := LatencySelector(time.Duration(20)*time.Second)(c, c.Servers)

	require.NoError(err)
	require.Len(result, 2)
	require.Equal([]*server.Desc{c.Servers[0], c.Servers[2]}, result)
}

func TestLatencySelector_MultipleServers(t *testing.T) {
	t.Parallel()

	require := require.New(t)

	var c = &Desc{
		Servers: []*server.Desc{
			&server.Desc{
				Endpoint:      conn.Endpoint("localhost:27017"),
				AverageRTT:    time.Duration(5) * time.Second,
				AverageRTTSet: true,
			},
			&server.Desc{
				Endpoint:      conn.Endpoint("localhost:27018"),
				AverageRTT:    time.Duration(26) * time.Second,
				AverageRTTSet: true,
			},
			&server.Desc{
				Endpoint:      conn.Endpoint("localhost:27018"),
				AverageRTT:    time.Duration(10) * time.Second,
				AverageRTTSet: true,
			},
		},
	}

	result, err := LatencySelector(time.Duration(20)*time.Second)(c, c.Servers)

	require.NoError(err)
	require.Len(result, 2)
	require.Equal([]*server.Desc{c.Servers[0], c.Servers[2]}, result)
}

func TestLatencySelector_No_servers(t *testing.T) {
	t.Parallel()

	require := require.New(t)

	var c = &Desc{}

	result, err := LatencySelector(time.Duration(20)*time.Second)(c, c.Servers)

	require.NoError(err)
	require.Len(result, 0)
}

func TestLatencySelector_1_server(t *testing.T) {
	t.Parallel()

	require := require.New(t)

	var c = &Desc{
		Servers: []*server.Desc{
			&server.Desc{
				Endpoint:      conn.Endpoint("localhost:27018"),
				AverageRTT:    time.Duration(26) * time.Second,
				AverageRTTSet: true,
			},
		},
	}

	result, err := LatencySelector(time.Duration(20)*time.Second)(c, c.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*server.Desc{c.Servers[0]}, result)
}
