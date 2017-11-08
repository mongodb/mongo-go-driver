// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package cluster_test

import (
	"encoding/json"
	"io/ioutil"
	"path"
	"testing"
	"time"

	"github.com/10gen/mongo-go-driver/mongo/internal/testutil/helpers"
	"github.com/10gen/mongo-go-driver/mongo/model"
	"github.com/10gen/mongo-go-driver/mongo/private/cluster"
	"github.com/10gen/mongo-go-driver/mongo/readpref"
	"github.com/stretchr/testify/require"
)

type testCase struct {
	TopologyDescription topDesc       `json:"topology_description"`
	Operation           string        `json:"operation"`
	ReadPreference      readPref      `json:"read_preference"`
	SuitableServers     []*serverDesc `json:"suitable_servers"`
	InLatencyWindow     []*serverDesc `json:"in_latency_window"`
}

type topDesc struct {
	Type    string        `json:"type"`
	Servers []*serverDesc `json:"servers"`
}

type serverDesc struct {
	Address      string            `json:"address"`
	AverageRTTMS int               `json:"avg_rtt_ms"`
	Type         string            `json:"type"`
	Tags         map[string]string `json:"tags"`
}

type readPref struct {
	Mode    string              `json:"mode"`
	TagSets []map[string]string `json:"tag_sets"`
}

const testsDir = "../../../data/server-selection/server_selection"

func clusterKindFromString(s string) model.ClusterKind {
	switch s {
	case "Single":
		return model.Single
	case "ReplicaSet":
		return model.ReplicaSet
	case "ReplicaSetNoPrimary":
		return model.ReplicaSetNoPrimary
	case "ReplicaSetWithPrimary":
		return model.ReplicaSetWithPrimary
	case "Sharded":
		return model.Sharded
	}

	return model.Unknown
}

func serverKindFromString(s string) model.ServerKind {
	switch s {
	case "Standalone":
		return model.Standalone
	case "RSOther":
		return model.RSMember
	case "RSPrimary":
		return model.RSPrimary
	case "RSSecondary":
		return model.RSSecondary
	case "RSArbiter":
		return model.RSArbiter
	case "RSGhost":
		return model.RSGhost
	case "Mongos":
		return model.Mongos
	}

	return model.Unknown
}

func findServerByAddress(servers []*model.Server, address string) *model.Server {
	for _, server := range servers {
		if server.Addr.String() == address {
			return server
		}
	}

	return nil
}

func anyTagsInSets(sets []model.TagSet) bool {
	for _, set := range sets {
		if len(set) > 0 {
			return true
		}
	}

	return false
}

func compareServers(t *testing.T, expected []*serverDesc, actual []*model.Server) {
	require.Equal(t, len(expected), len(actual))

	for _, expectedServer := range expected {
		actualServer := findServerByAddress(actual, expectedServer.Address)
		require.NotNil(t, actualServer)
		require.Equal(t, expectedServer.AverageRTTMS, int(actualServer.AverageRTT/time.Millisecond))
		require.Equal(t, expectedServer.Type, actualServer.Kind.String())

		require.Equal(t, len(expectedServer.Tags), len(actualServer.Tags))
		for _, actualTag := range actualServer.Tags {
			expectedTag, ok := expectedServer.Tags[actualTag.Name]
			require.True(t, ok)
			require.Equal(t, expectedTag, actualTag.Value)
		}
	}
}

func runTest(t *testing.T, directory string, filename string) {
	filepath := path.Join(testsDir, directory, filename)
	content, err := ioutil.ReadFile(filepath)
	require.NoError(t, err)

	// Remove ".json" from filename.
	filename = filename[:len(filename)-5]
	testName := directory + "/" + filename + ":"

	t.Run(testName, func(t *testing.T) {
		var test testCase
		require.NoError(t, json.Unmarshal(content, &test))

		servers := make([]*model.Server, 0, len(test.TopologyDescription.Servers))

		for _, serverDescription := range test.TopologyDescription.Servers {
			server := &model.Server{
				AverageRTT:    time.Duration(serverDescription.AverageRTTMS) * time.Millisecond,
				AverageRTTSet: true,
				Addr:          model.Addr(serverDescription.Address),
				Kind:          serverKindFromString(serverDescription.Type),
				Tags:          model.NewTagSetFromMap(serverDescription.Tags),
			}

			servers = append(servers, server)
		}

		c := &model.Cluster{
			Kind:    clusterKindFromString(test.TopologyDescription.Type),
			Servers: servers,
		}

		readprefMode, err := readpref.ModeFromString(test.ReadPreference.Mode)
		require.NoError(t, err)

		options := make([]readpref.Option, 0, 1)

		tagSets := model.NewTagSetsFromMaps(test.ReadPreference.TagSets)
		if anyTagsInSets(tagSets) {
			options = append(options, readpref.WithTagSets(tagSets...))
		}

		rp, err := readpref.New(readprefMode, options...)
		require.NoError(t, err)

		selector := readpref.Selector(rp)
		if test.Operation == "write" {
			selector = cluster.CompositeSelector(
				[]cluster.ServerSelector{cluster.WriteSelector(), selector},
			)
		}

		result, err := selector(c, c.Servers)
		require.NoError(t, err)

		compareServers(t, test.SuitableServers, result)

		latencySelector := cluster.LatencySelector(time.Duration(15) * time.Millisecond)
		result, err = cluster.CompositeSelector(
			[]cluster.ServerSelector{selector, latencySelector},
		)(c, c.Servers)
		require.NoError(t, err)

		compareServers(t, test.InLatencyWindow, result)
	})
}

// Test case for all SDAM spec tests.
func TestServerSelectionSpec(t *testing.T) {
	for _, topology := range [...]string{
		"ReplicaSetNoPrimary",
		"ReplicaSetWithPrimary",
		"Sharded",
		"Single",
		"Unknown",
	} {
		for _, subdir := range [...]string{"read", "write"} {
			subdirPath := path.Join(topology, subdir)

			for _, file := range testhelpers.FindJSONFilesInDir(t, path.Join(testsDir, subdirPath)) {
				runTest(t, subdirPath, file)
			}
		}
	}
}
