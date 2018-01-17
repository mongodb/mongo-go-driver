// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops_test

import (
	"context"
	"testing"
	"time"

	oldbson "github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/mongo"
	"github.com/10gen/mongo-go-driver/mongo/internal/testutil"
	. "github.com/10gen/mongo-go-driver/mongo/private/ops"
	"github.com/skriptble/wilson/bson"
	"github.com/stretchr/testify/require"
)

func TestAggregateWithInvalidNamespace(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)

	_, err := Aggregate(
		context.Background(),
		getServer(t),
		Namespace{},
		nil,
		bson.NewArray(0),
	)
	require.Error(t, err)
}

func TestAggregateWithMultipleBatches(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)
	documents := []oldbson.D{
		{oldbson.NewDocElem("_id", 1)},
		{oldbson.NewDocElem("_id", 2)},
		{oldbson.NewDocElem("_id", 3)},
		{oldbson.NewDocElem("_id", 4)},
		{oldbson.NewDocElem("_id", 5)},
	}
	testutil.AutoInsertDocs(t, documents...)

	server := getServer(t)
	namespace := Namespace{DB: testutil.DBName(t), Collection: testutil.ColName(t)}
	cursor, err := Aggregate(context.Background(), server, namespace, nil,
		bson.NewArray(2).Append(
			bson.AC.Document(
				bson.NewDocument(1).
					Append(bson.C.SubDocument("$match",
						bson.NewDocument(1).
							Append(bson.C.SubDocument("_id", bson.NewDocument(1).Append(bson.C.Int32("$gt", 2)))),
					)),
			),
			bson.AC.Document(bson.NewDocument(1).Append(bson.C.SubDocument("$sort", bson.NewDocument(1).Append(bson.C.Int32("_id", -1)))))),
		mongo.BatchSize(2),
	)
	require.NoError(t, err)

	var next oldbson.D

	cursor.Next(context.Background(), &next)
	require.Equal(t, documents[4], next)

	cursor.Next(context.Background(), &next)
	require.Equal(t, documents[3], next)

	cursor.Next(context.Background(), &next)
	require.Equal(t, documents[2], next)

	hasNext := cursor.Next(context.Background(), &next)
	require.False(t, hasNext)
}

// This is not a great test since there are no visible side effects of allowDiskUse, and there server does not currently
// check the validity of field names for the aggregate command
func TestAggregateWithAllowDiskUse(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)
	testutil.AutoInsertDocs(t,
		oldbson.D{oldbson.NewDocElem("_id", 1)},
		oldbson.D{oldbson.NewDocElem("_id", 2)},
	)

	server := getServer(t)
	namespace := Namespace{DB: testutil.DBName(t), Collection: testutil.ColName(t)}
	_, err := Aggregate(context.Background(), server, namespace, nil,
		bson.NewArray(0),
		mongo.AllowDiskUse(true),
	)
	require.NoError(t, err)
}

func TestAggregateWithMaxTimeMS(t *testing.T) {
	t.Skip("max time is flaky on the server")
	t.Parallel()
	testutil.Integration(t)

	s := getServer(t)

	if testutil.EnableMaxTimeFailPoint(t, s) != nil {
		t.Skip("skipping maxTimeMS test when max time failpoint is disabled")
	}
	defer testutil.DisableMaxTimeFailPoint(t, s)

	namespace := Namespace{DB: testutil.DBName(t), Collection: testutil.ColName(t)}
	_, err := Aggregate(context.Background(), s, namespace, nil,
		bson.NewArray(0),
		mongo.MaxTime(time.Millisecond),
	)
	require.Error(t, err)

	// Hacky check for the error message.  Should we be returning a more structured error?
	require.Contains(t, err.Error(), "operation exceeded time limit")
}

func TestLegacyAggregateWithInvalidNamespace(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)

	_, err := LegacyAggregate(
		context.Background(),
		getServer(t),
		Namespace{},
		[]oldbson.D{},
		AggregationOptions{})
	require.Error(t, err)
}

func TestLegacyAggregateWithMultipleBatches(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)
	documents := []oldbson.D{
		{oldbson.NewDocElem("_id", 1)},
		{oldbson.NewDocElem("_id", 2)},
		{oldbson.NewDocElem("_id", 3)},
		{oldbson.NewDocElem("_id", 4)},
		{oldbson.NewDocElem("_id", 5)},
	}
	testutil.AutoInsertDocs(t, documents...)

	server := getServer(t)
	namespace := Namespace{DB: testutil.DBName(t), Collection: testutil.ColName(t)}
	cursor, err := LegacyAggregate(context.Background(), server, namespace,
		[]oldbson.D{
			{
				oldbson.NewDocElem("$match", oldbson.D{
					oldbson.NewDocElem("_id", oldbson.D{oldbson.NewDocElem("$gt", 2)}),
				}),
			},
			{
				oldbson.NewDocElem("$sort", oldbson.D{oldbson.NewDocElem("_id", -1)}),
			},
		},
		AggregationOptions{
			BatchSize: 2})
	require.NoError(t, err)

	var next oldbson.D

	cursor.Next(context.Background(), &next)
	require.Equal(t, documents[4], next)

	cursor.Next(context.Background(), &next)
	require.Equal(t, documents[3], next)

	cursor.Next(context.Background(), &next)
	require.Equal(t, documents[2], next)

	hasNext := cursor.Next(context.Background(), &next)
	require.False(t, hasNext)
}

// This is not a great test since there are no visible side effects of allowDiskUse, and there server does not currently
// check the validity of field names for the aggregate command
func TestLegacyAggregateWithAllowDiskUse(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)
	testutil.AutoInsertDocs(t,
		oldbson.D{oldbson.NewDocElem("_id", 1)},
		oldbson.D{oldbson.NewDocElem("_id", 2)},
	)

	server := getServer(t)
	namespace := Namespace{DB: testutil.DBName(t), Collection: testutil.ColName(t)}
	_, err := LegacyAggregate(context.Background(), server, namespace,
		[]oldbson.D{},
		AggregationOptions{
			AllowDiskUse: true})
	require.NoError(t, err)
}

func TestLegacyAggregateWithMaxTimeMS(t *testing.T) {
	t.Skip("max time is flaky on the server")
	t.Parallel()
	testutil.Integration(t)

	s := getServer(t)

	if testutil.EnableMaxTimeFailPoint(t, s) != nil {
		t.Skip("skipping maxTimeMS test when max time failpoint is disabled")
	}
	defer testutil.DisableMaxTimeFailPoint(t, s)

	namespace := Namespace{DB: testutil.DBName(t), Collection: testutil.ColName(t)}
	_, err := LegacyAggregate(context.Background(), s, namespace,
		[]oldbson.D{},
		AggregationOptions{
			MaxTime: time.Millisecond})
	require.Error(t, err)

	// Hacky check for the error message.  Should we be returning a more structured error?
	require.Contains(t, err.Error(), "operation exceeded time limit")
}
