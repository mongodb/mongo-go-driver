// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package cluster_test

import (
	"testing"

	"github.com/mongodb/mongo-go-driver/mongo/model"
	. "github.com/mongodb/mongo-go-driver/mongo/private/cluster"
	"github.com/stretchr/testify/require"
)

func TestWriteSelector_ReplicaSetWithPrimary(t *testing.T) {
	t.Parallel()

	require := require.New(t)

	c := &model.Cluster{
		Kind: model.ReplicaSetWithPrimary,
		Servers: []*model.Server{
			{
				Addr: model.Addr("localhost:27017"),
				Kind: model.RSPrimary,
			},
			{
				Addr: model.Addr("localhost:27018"),
				Kind: model.RSSecondary,
			},
			{
				Addr: model.Addr("localhost:27018"),
				Kind: model.RSSecondary,
			},
		},
	}

	result, err := WriteSelector()(c, c.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*model.Server{c.Servers[0]}, result)
}

func TestWriteSelector_ReplicaSetNoPrimary(t *testing.T) {
	t.Parallel()

	require := require.New(t)

	c := &model.Cluster{
		Kind: model.ReplicaSetNoPrimary,
		Servers: []*model.Server{
			{
				Addr: model.Addr("localhost:27018"),
				Kind: model.RSSecondary,
			},
			{
				Addr: model.Addr("localhost:27018"),
				Kind: model.RSSecondary,
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

	c := &model.Cluster{
		Kind: model.Sharded,
		Servers: []*model.Server{
			{
				Addr: model.Addr("localhost:27018"),
				Kind: model.Mongos,
			},
			{
				Addr: model.Addr("localhost:27018"),
				Kind: model.Mongos,
			},
		},
	}

	result, err := WriteSelector()(c, c.Servers)

	require.NoError(err)
	require.Len(result, 2)
	require.Equal([]*model.Server{c.Servers[0], c.Servers[1]}, result)
}

func TestWriteSelector_Single(t *testing.T) {
	t.Parallel()

	require := require.New(t)

	c := &model.Cluster{
		Kind: model.Single,
		Servers: []*model.Server{
			{
				Addr: model.Addr("localhost:27018"),
				Kind: model.Standalone,
			},
		},
	}

	result, err := WriteSelector()(c, c.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*model.Server{c.Servers[0]}, result)
}
