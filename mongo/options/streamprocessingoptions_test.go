// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options_test

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/mongoutil"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestCreateStreamProcessorOptions_Builder(t *testing.T) {
	dlq := bson.Raw([]byte{0x05, 0x00, 0x00, 0x00, 0x00}) // empty BSON doc
	b := options.CreateStreamProcessor().
		SetDLQ(dlq).
		SetStreamMetaFieldName("_meta").
		SetTier("SP10").
		SetFailoverEnabled(true)

	args, err := mongoutil.NewOptions[options.CreateStreamProcessorOptions](b)
	require.NoError(t, err)
	assert.Equal(t, dlq, args.DLQ)
	require.NotNil(t, args.StreamMetaFieldName)
	assert.Equal(t, "_meta", *args.StreamMetaFieldName)
	require.NotNil(t, args.Tier)
	assert.Equal(t, "SP10", *args.Tier)
	require.NotNil(t, args.FailoverEnabled)
	assert.True(t, *args.FailoverEnabled)
}

func TestStartStreamProcessorOptions_Builder(t *testing.T) {
	fo := options.StreamProcessorFailover().SetRegion("us-east-1").SetMode("FORCED").SetDryRun(true)
	foArgs, err := mongoutil.NewOptions[options.StreamProcessorFailoverOptions](fo)
	require.NoError(t, err)

	b := options.StartStreamProcessor().
		SetWorkers(3).
		SetClearCheckpoints(true).
		SetStartAtOperationTime(bson.Timestamp{T: 100, I: 1}).
		SetTier("SP30").
		SetEnableAutoScaling(true).
		SetFailover(foArgs)

	args, err := mongoutil.NewOptions[options.StartStreamProcessorOptions](b)
	require.NoError(t, err)
	require.NotNil(t, args.Workers)
	assert.Equal(t, int32(3), *args.Workers)
	require.NotNil(t, args.ClearCheckpoints)
	assert.True(t, *args.ClearCheckpoints)
	require.NotNil(t, args.StartAtOperationTime)
	assert.Equal(t, uint32(100), args.StartAtOperationTime.T)
	require.NotNil(t, args.Tier)
	assert.Equal(t, "SP30", *args.Tier)
	require.NotNil(t, args.EnableAutoScaling)
	assert.True(t, *args.EnableAutoScaling)
	require.NotNil(t, args.Failover)
	assert.Equal(t, "us-east-1", args.Failover.Region)
}

func TestGetStreamProcessorStatsOptions_Builder(t *testing.T) {
	b := options.GetStreamProcessorStats().SetVerbose(true)
	args, err := mongoutil.NewOptions[options.GetStreamProcessorStatsOptions](b)
	require.NoError(t, err)
	require.NotNil(t, args.Verbose)
	assert.True(t, *args.Verbose)
}

func TestGetStreamProcessorSamplesOptions_Builder(t *testing.T) {
	b := options.GetStreamProcessorSamples().SetCursorID(42).SetLimit(100).SetBatchSize(20)
	args, err := mongoutil.NewOptions[options.GetStreamProcessorSamplesOptions](b)
	require.NoError(t, err)
	require.NotNil(t, args.CursorID)
	assert.Equal(t, int64(42), *args.CursorID)
	require.NotNil(t, args.Limit)
	assert.Equal(t, int32(100), *args.Limit)
	require.NotNil(t, args.BatchSize)
	assert.Equal(t, int32(20), *args.BatchSize)
}
