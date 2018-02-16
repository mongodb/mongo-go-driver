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

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/internal/testutil"
	. "github.com/mongodb/mongo-go-driver/mongo/private/ops"
	"github.com/stretchr/testify/require"
)

func TestAggregateWithInvalidNamespace(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)

	_, err := Aggregate(
		context.Background(),
		getServer(t),
		Namespace{},
		bson.NewArray(),
		false,
	)
	require.Error(t, err)
}

func TestAggregateWithMultipleBatches(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)
	documents := []*bson.Document{
		bson.NewDocument(bson.EC.Int32("_id", 1)),
		bson.NewDocument(bson.EC.Int32("_id", 2)),
		bson.NewDocument(bson.EC.Int32("_id", 3)),
		bson.NewDocument(bson.EC.Int32("_id", 4)),
		bson.NewDocument(bson.EC.Int32("_id", 5)),
	}
	testutil.AutoInsertDocs(t, nil, documents...)

	readers := make([]bson.Reader, 0, len(documents))
	for _, doc := range documents {
		r, err := doc.MarshalBSON()
		require.NoError(t, err)
		readers = append(readers, r)
	}

	server := getServer(t)
	namespace := Namespace{DB: testutil.DBName(t), Collection: testutil.ColName(t)}
	cursor, err := Aggregate(context.Background(), server, namespace,
		bson.NewArray(
			bson.VC.Document(
				bson.NewDocument(
					bson.EC.SubDocument("$match", bson.NewDocument(
						bson.EC.SubDocument("_id", bson.NewDocument(bson.EC.Int32("$gt", 2)))),
					)),
			),
			bson.VC.Document(bson.NewDocument(bson.EC.SubDocument("$sort", bson.NewDocument(bson.EC.Int32("_id", -1)))))),
		false,
		mongo.Opt.BatchSize(2),
	)
	require.NoError(t, err)

	var hasNext bool
	var next = make(bson.Reader, 1024)

	hasNext = cursor.Next(context.Background())
	require.True(t, hasNext)
	err = cursor.Decode(next)
	require.NoError(t, err)
	require.Equal(t, readers[4], next[:len(readers[4])])

	cursor.Next(context.Background())
	require.True(t, hasNext)
	err = cursor.Decode(next)
	require.NoError(t, err)
	require.Equal(t, readers[3], next[:len(readers[3])])

	cursor.Next(context.Background())
	require.True(t, hasNext)
	err = cursor.Decode(next)
	require.NoError(t, err)
	require.Equal(t, readers[2], next[:len(readers[2])])

	hasNext = cursor.Next(context.Background())
	require.False(t, hasNext)
}

// This is not a great test since there are no visible side effects of allowDiskUse, and there server does not currently
// check the validity of field names for the aggregate command
func TestAggregateWithAllowDiskUse(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)
	testutil.AutoInsertDocs(t,
		nil,
		bson.NewDocument(bson.EC.Int32("_id", 1)),
		bson.NewDocument(bson.EC.Int32("_id", 2)),
	)

	server := getServer(t)
	namespace := Namespace{DB: testutil.DBName(t), Collection: testutil.ColName(t)}
	_, err := Aggregate(context.Background(), server, namespace,
		bson.NewArray(),
		false,
		mongo.Opt.AllowDiskUse(true),
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
	_, err := Aggregate(context.Background(), s, namespace,
		bson.NewArray(),
		false,
		mongo.Opt.MaxTime(time.Millisecond),
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
		bson.NewArray(),
		AggregationOptions{})
	require.Error(t, err)
}

func TestLegacyAggregateWithMultipleBatches(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)
	documents := []*bson.Document{
		bson.NewDocument(bson.EC.Int32("_id", 1)),
		bson.NewDocument(bson.EC.Int32("_id", 2)),
		bson.NewDocument(bson.EC.Int32("_id", 3)),
		bson.NewDocument(bson.EC.Int32("_id", 4)),
		bson.NewDocument(bson.EC.Int32("_id", 5)),
	}
	testutil.AutoInsertDocs(t, nil, documents...)

	readers := make([]bson.Reader, 0, len(documents))
	for _, doc := range documents {
		r, err := doc.MarshalBSON()
		require.NoError(t, err)
		readers = append(readers, r)
	}

	server := getServer(t)
	namespace := Namespace{DB: testutil.DBName(t), Collection: testutil.ColName(t)}
	cursor, err := LegacyAggregate(context.Background(), server, namespace,
		bson.NewArray(
			bson.VC.DocumentFromElements(
				bson.EC.SubDocumentFromElements(
					"$match",
					bson.EC.SubDocumentFromElements(
						"_id",
						bson.EC.Int32("$gt", 2),
					),
				),
			),
			bson.VC.DocumentFromElements(
				bson.EC.SubDocumentFromElements(
					"$sort",
					bson.EC.Int32("_id", -1),
				),
			)),

		AggregationOptions{
			BatchSize: 2})
	require.NoError(t, err)

	var hasNext bool
	var next = make(bson.Reader, 1024)

	hasNext = cursor.Next(context.Background())
	require.True(t, hasNext)
	err = cursor.Decode(next)
	require.NoError(t, err)
	require.Equal(t, readers[4], next[:len(readers[4])])

	hasNext = cursor.Next(context.Background())
	require.True(t, hasNext)
	err = cursor.Decode(next)
	require.NoError(t, err)
	require.Equal(t, readers[3], next[:len(readers[3])])

	hasNext = cursor.Next(context.Background())
	require.True(t, hasNext)
	err = cursor.Decode(next)
	require.NoError(t, err)
	require.Equal(t, readers[2], next[:len(readers[2])])

	hasNext = cursor.Next(context.Background())
	require.False(t, hasNext)
}

// This is not a great test since there are no visible side effects of allowDiskUse, and there server does not currently
// check the validity of field names for the aggregate command
func TestLegacyAggregateWithAllowDiskUse(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)
	testutil.AutoInsertDocs(t,
		nil,
		bson.NewDocument(bson.EC.Int32("_id", 1)),
		bson.NewDocument(bson.EC.Int32("_id", 2)),
	)

	server := getServer(t)
	namespace := Namespace{DB: testutil.DBName(t), Collection: testutil.ColName(t)}
	_, err := LegacyAggregate(context.Background(), server, namespace,
		bson.NewArray(),
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
		bson.NewArray(),
		AggregationOptions{
			MaxTime: time.Millisecond})
	require.Error(t, err)

	// Hacky check for the error message.  Should we be returning a more structured error?
	require.Contains(t, err.Error(), "operation exceeded time limit")
}
