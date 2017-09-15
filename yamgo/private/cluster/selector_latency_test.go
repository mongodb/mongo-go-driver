package cluster_test

import (
	"testing"
	"time"

	. "github.com/10gen/mongo-go-driver/yamgo/private/cluster"
	"github.com/10gen/mongo-go-driver/yamgo/model"
	"github.com/stretchr/testify/require"
)

func TestLatencySelector_NoRTTSet(t *testing.T) {
	t.Parallel()

	require := require.New(t)

	var c = &model.Cluster{
		Servers: []*model.Server{
			&model.Server{
				Addr: model.Addr("localhost:27017"),
			},
			&model.Server{
				Addr: model.Addr("localhost:27018"),
			},
			&model.Server{
				Addr: model.Addr("localhost:27018"),
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

	var c = &model.Cluster{
		Servers: []*model.Server{
			&model.Server{
				Addr:          model.Addr("localhost:27017"),
				AverageRTT:    time.Duration(5) * time.Second,
				AverageRTTSet: true,
			},
			&model.Server{
				Addr: model.Addr("localhost:27018"),
			},
			&model.Server{
				Addr:          model.Addr("localhost:27018"),
				AverageRTT:    time.Duration(10) * time.Second,
				AverageRTTSet: true,
			},
		},
	}

	result, err := LatencySelector(time.Duration(20)*time.Second)(c, c.Servers)

	require.NoError(err)
	require.Len(result, 2)
	require.Equal([]*model.Server{c.Servers[0], c.Servers[2]}, result)
}

func TestLatencySelector_MultipleServers(t *testing.T) {
	t.Parallel()

	require := require.New(t)

	var c = &model.Cluster{
		Servers: []*model.Server{
			&model.Server{
				Addr:          model.Addr("localhost:27017"),
				AverageRTT:    time.Duration(5) * time.Second,
				AverageRTTSet: true,
			},
			&model.Server{
				Addr:          model.Addr("localhost:27018"),
				AverageRTT:    time.Duration(26) * time.Second,
				AverageRTTSet: true,
			},
			&model.Server{
				Addr:          model.Addr("localhost:27018"),
				AverageRTT:    time.Duration(10) * time.Second,
				AverageRTTSet: true,
			},
		},
	}

	result, err := LatencySelector(time.Duration(20)*time.Second)(c, c.Servers)

	require.NoError(err)
	require.Len(result, 2)
	require.Equal([]*model.Server{c.Servers[0], c.Servers[2]}, result)
}

func TestLatencySelector_No_servers(t *testing.T) {
	t.Parallel()

	require := require.New(t)

	var c = &model.Cluster{}

	result, err := LatencySelector(time.Duration(20)*time.Second)(c, c.Servers)

	require.NoError(err)
	require.Len(result, 0)
}

func TestLatencySelector_1_server(t *testing.T) {
	t.Parallel()

	require := require.New(t)

	var c = &model.Cluster{
		Servers: []*model.Server{
			&model.Server{
				Addr:          model.Addr("localhost:27018"),
				AverageRTT:    time.Duration(26) * time.Second,
				AverageRTTSet: true,
			},
		},
	}

	result, err := LatencySelector(time.Duration(20)*time.Second)(c, c.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*model.Server{c.Servers[0]}, result)
}
