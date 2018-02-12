// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package cluster_test

import (
	"testing"
	"time"

	"github.com/mongodb/mongo-go-driver/mongo/model"
	. "github.com/mongodb/mongo-go-driver/mongo/private/cluster"
	"github.com/stretchr/testify/require"
)

func TestLatencySelector_NoRTTSet(t *testing.T) {
	t.Parallel()

	require := require.New(t)

	var c = &model.Cluster{
		Servers: []*model.Server{
			{
				Addr: model.Addr("localhost:27017"),
			},
			{
				Addr: model.Addr("localhost:27018"),
			},
			{
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
			{
				Addr:          model.Addr("localhost:27017"),
				AverageRTT:    time.Duration(5) * time.Second,
				AverageRTTSet: true,
			},
			{
				Addr: model.Addr("localhost:27018"),
			},
			{
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
			{
				Addr:          model.Addr("localhost:27017"),
				AverageRTT:    time.Duration(5) * time.Second,
				AverageRTTSet: true,
			},
			{
				Addr:          model.Addr("localhost:27018"),
				AverageRTT:    time.Duration(26) * time.Second,
				AverageRTTSet: true,
			},
			{
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
			{
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
