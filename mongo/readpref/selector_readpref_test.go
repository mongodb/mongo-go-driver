// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package readpref_test

import (
	"testing"
	"time"

	"github.com/mongodb/mongo-go-driver/mongo/model"
	. "github.com/mongodb/mongo-go-driver/mongo/readpref"
	"github.com/stretchr/testify/require"
)

var readPrefTestPrimary = &model.Server{
	Addr:              model.Addr("localhost:27017"),
	HeartbeatInterval: time.Duration(10) * time.Second,
	LastWriteTime:     time.Date(2017, 2, 11, 14, 0, 0, 0, time.UTC),
	LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
	Kind:              model.RSPrimary,
	Tags:              model.TagSet{model.Tag{Name: "a", Value: "1"}},
	Version:           model.Version{Parts: []uint8{3, 4, 0}},
}
var readPrefTestSecondary1 = &model.Server{
	Addr:              model.Addr("localhost:27018"),
	HeartbeatInterval: time.Duration(10) * time.Second,
	LastWriteTime:     time.Date(2017, 2, 11, 13, 58, 0, 0, time.UTC),
	LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
	Kind:              model.RSSecondary,
	Tags:              model.TagSet{model.Tag{Name: "a", Value: "1"}},
	Version:           model.Version{Parts: []uint8{3, 4, 0}},
}
var readPrefTestSecondary2 = &model.Server{
	Addr:              model.Addr("localhost:27018"),
	HeartbeatInterval: time.Duration(10) * time.Second,
	LastWriteTime:     time.Date(2017, 2, 11, 14, 0, 0, 0, time.UTC),
	LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
	Kind:              model.RSSecondary,
	Tags:              model.TagSet{model.Tag{Name: "a", Value: "2"}},
	Version:           model.Version{Parts: []uint8{3, 4, 0}},
}
var readPrefTestCluster = &model.Cluster{
	Kind:    model.ReplicaSetWithPrimary,
	Servers: []*model.Server{readPrefTestPrimary, readPrefTestSecondary1, readPrefTestSecondary2},
}

func TestModeFromString(t *testing.T) {
	t.Parallel()

	require := require.New(t)

	mode, err := ModeFromString("primary")
	require.NoError(err)
	require.Equal(mode, PrimaryMode)
	mode, err = ModeFromString("primaryPreferred")
	require.NoError(err)
	require.Equal(mode, PrimaryPreferredMode)
	mode, err = ModeFromString("secondary")
	require.NoError(err)
	require.Equal(mode, SecondaryMode)
	mode, err = ModeFromString("secondaryPreferred")
	require.NoError(err)
	require.Equal(mode, SecondaryPreferredMode)
	mode, err = ModeFromString("nearest")
	require.NoError(err)
	require.Equal(mode, NearestMode)
	_, err = ModeFromString("invalid")
	require.Error(err)
}

func TestSelector_Sharded(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := Primary()

	s := &model.Server{
		Addr:              model.Addr("localhost:27017"),
		HeartbeatInterval: time.Duration(10) * time.Second,
		LastWriteTime:     time.Date(2017, 2, 11, 14, 0, 0, 0, time.UTC),
		LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
		Kind:              model.Mongos,
		Version:           model.Version{Parts: []uint8{3, 4, 0}},
	}
	c := &model.Cluster{
		Kind:    model.Sharded,
		Servers: []*model.Server{s},
	}

	result, err := Selector(subject)(c, c.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*model.Server{s}, result)
}

func TestSelector_Single(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := Primary()

	s := &model.Server{
		Addr:              model.Addr("localhost:27017"),
		HeartbeatInterval: time.Duration(10) * time.Second,
		LastWriteTime:     time.Date(2017, 2, 11, 14, 0, 0, 0, time.UTC),
		LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
		Kind:              model.Mongos,
		Version:           model.Version{Parts: []uint8{3, 4, 0}},
	}
	c := &model.Cluster{
		Kind:    model.Single,
		Servers: []*model.Server{s},
	}

	result, err := Selector(subject)(c, c.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*model.Server{s}, result)
}

func TestSelector_Primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := Primary()

	result, err := Selector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*model.Server{readPrefTestPrimary}, result)
}

func TestSelector_Primary_with_no_primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := Primary()

	result, err := Selector(subject)(readPrefTestCluster, []*model.Server{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(err)
	require.Empty(result, 0)
}

func TestSelector_PrimaryPreferred(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := PrimaryPreferred()

	result, err := Selector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*model.Server{readPrefTestPrimary}, result)
}

func TestSelector_PrimaryPreferred_ignores_tags(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := PrimaryPreferred(
		WithTags("a", "2"),
	)

	result, err := Selector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*model.Server{readPrefTestPrimary}, result)
}

func TestSelector_PrimaryPreferred_with_no_primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := PrimaryPreferred()

	result, err := Selector(subject)(readPrefTestCluster, []*model.Server{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(err)
	require.Len(result, 2)
	require.Equal([]*model.Server{readPrefTestSecondary1, readPrefTestSecondary2}, result)
}

func TestSelector_PrimaryPreferred_with_no_primary_and_tags(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := PrimaryPreferred(
		WithTags("a", "2"),
	)

	result, err := Selector(subject)(readPrefTestCluster, []*model.Server{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*model.Server{readPrefTestSecondary2}, result)
}

func TestSelector_PrimaryPreferred_with_maxStaleness(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := PrimaryPreferred(
		WithMaxStaleness(time.Duration(90) * time.Second),
	)

	result, err := Selector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*model.Server{readPrefTestPrimary}, result)
}

func TestSelector_PrimaryPreferred_with_maxStaleness_and_no_primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := PrimaryPreferred(
		WithMaxStaleness(time.Duration(90) * time.Second),
	)

	result, err := Selector(subject)(readPrefTestCluster, []*model.Server{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*model.Server{readPrefTestSecondary2}, result)
}

func TestSelector_SecondaryPreferred(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := SecondaryPreferred()

	result, err := Selector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 2)
	require.Equal([]*model.Server{readPrefTestSecondary1, readPrefTestSecondary2}, result)
}

func TestSelector_SecondaryPreferred_with_tags(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := SecondaryPreferred(
		WithTags("a", "2"),
	)

	result, err := Selector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*model.Server{readPrefTestSecondary2}, result)
}

func TestSelector_SecondaryPreferred_with_tags_that_do_not_match(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := SecondaryPreferred(
		WithTags("a", "3"),
	)

	result, err := Selector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*model.Server{readPrefTestPrimary}, result)
}

func TestSelector_SecondaryPreferred_with_tags_that_do_not_match_and_no_primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := SecondaryPreferred(
		WithTags("a", "3"),
	)

	result, err := Selector(subject)(readPrefTestCluster, []*model.Server{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(err)
	require.Len(result, 0)
}

func TestSelector_SecondaryPreferred_with_no_secondaries(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := SecondaryPreferred()

	result, err := Selector(subject)(readPrefTestCluster, []*model.Server{readPrefTestPrimary})

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*model.Server{readPrefTestPrimary}, result)
}

func TestSelector_SecondaryPreferred_with_no_secondaries_or_primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := SecondaryPreferred()

	result, err := Selector(subject)(readPrefTestCluster, []*model.Server{})

	require.NoError(err)
	require.Len(result, 0)
}

func TestSelector_SecondaryPreferred_with_maxStaleness(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := SecondaryPreferred(
		WithMaxStaleness(time.Duration(90) * time.Second),
	)

	result, err := Selector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*model.Server{readPrefTestSecondary2}, result)
}

func TestSelector_SecondaryPreferred_with_maxStaleness_and_no_primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := SecondaryPreferred(
		WithMaxStaleness(time.Duration(90) * time.Second),
	)

	result, err := Selector(subject)(readPrefTestCluster, []*model.Server{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*model.Server{readPrefTestSecondary2}, result)
}

func TestSelector_Secondary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := Secondary()

	result, err := Selector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 2)
	require.Equal([]*model.Server{readPrefTestSecondary1, readPrefTestSecondary2}, result)
}

func TestSelector_Secondary_with_tags(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := Secondary(
		WithTags("a", "2"),
	)

	result, err := Selector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*model.Server{readPrefTestSecondary2}, result)
}

func TestSelector_Secondary_with_tags_that_do_not_match(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := Secondary(
		WithTags("a", "3"),
	)

	result, err := Selector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 0)
}

func TestSelector_Secondary_with_no_secondaries(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := Secondary()

	result, err := Selector(subject)(readPrefTestCluster, []*model.Server{readPrefTestPrimary})

	require.NoError(err)
	require.Len(result, 0)
}

func TestSelector_Secondary_with_maxStaleness(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := Secondary(
		WithMaxStaleness(time.Duration(90) * time.Second),
	)

	result, err := Selector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*model.Server{readPrefTestSecondary2}, result)
}

func TestSelector_Secondary_with_maxStaleness_and_no_primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := Secondary(
		WithMaxStaleness(time.Duration(90) * time.Second),
	)

	result, err := Selector(subject)(readPrefTestCluster, []*model.Server{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*model.Server{readPrefTestSecondary2}, result)
}

func TestSelector_Nearest(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := Nearest()

	result, err := Selector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 3)
	require.Equal([]*model.Server{readPrefTestPrimary, readPrefTestSecondary1, readPrefTestSecondary2}, result)
}

func TestSelector_Nearest_with_tags(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := Nearest(
		WithTags("a", "1"),
	)

	result, err := Selector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 2)
	require.Equal([]*model.Server{readPrefTestPrimary, readPrefTestSecondary1}, result)
}

func TestSelector_Nearest_with_tags_that_do_not_match(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := Nearest(
		WithTags("a", "3"),
	)

	result, err := Selector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 0)
}

func TestSelector_Nearest_with_no_primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := Nearest()

	result, err := Selector(subject)(readPrefTestCluster, []*model.Server{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(err)
	require.Len(result, 2)
	require.Equal([]*model.Server{readPrefTestSecondary1, readPrefTestSecondary2}, result)
}

func TestSelector_Nearest_with_no_secondaries(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := Nearest()

	result, err := Selector(subject)(readPrefTestCluster, []*model.Server{readPrefTestPrimary})

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*model.Server{readPrefTestPrimary}, result)
}

func TestSelector_Nearest_with_maxStaleness(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := Nearest(
		WithMaxStaleness(time.Duration(90) * time.Second),
	)

	result, err := Selector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 2)
	require.Equal([]*model.Server{readPrefTestPrimary, readPrefTestSecondary2}, result)
}

func TestSelector_Nearest_with_maxStaleness_and_no_primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := Nearest(
		WithMaxStaleness(time.Duration(90) * time.Second),
	)

	result, err := Selector(subject)(readPrefTestCluster, []*model.Server{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*model.Server{readPrefTestSecondary2}, result)
}

func TestSelector_Max_staleness_is_less_than_90_seconds(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := Nearest(
		WithMaxStaleness(time.Duration(50) * time.Second),
	)

	s := &model.Server{
		Addr:              model.Addr("localhost:27017"),
		HeartbeatInterval: time.Duration(10) * time.Second,
		LastWriteTime:     time.Date(2017, 2, 11, 14, 0, 0, 0, time.UTC),
		LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
		Kind:              model.RSPrimary,
		Version:           model.Version{Parts: []uint8{3, 4, 0}},
	}
	c := &model.Cluster{
		Kind:    model.ReplicaSetWithPrimary,
		Servers: []*model.Server{s},
	}

	_, err := Selector(subject)(c, c.Servers)

	require.Error(err)
}

func TestSelector_Max_staleness_is_too_low(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := Nearest(
		WithMaxStaleness(time.Duration(100) * time.Second),
	)

	s := &model.Server{
		Addr:              model.Addr("localhost:27017"),
		HeartbeatInterval: time.Duration(100) * time.Second,
		LastWriteTime:     time.Date(2017, 2, 11, 14, 0, 0, 0, time.UTC),
		LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
		Kind:              model.RSPrimary,
		Version:           model.Version{Parts: []uint8{3, 4, 0}},
	}
	c := &model.Cluster{
		Kind:    model.ReplicaSetWithPrimary,
		Servers: []*model.Server{s},
	}

	_, err := Selector(subject)(c, c.Servers)

	require.Error(err)
}
