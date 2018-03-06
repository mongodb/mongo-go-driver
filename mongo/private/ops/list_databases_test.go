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
	"github.com/mongodb/mongo-go-driver/mongo/internal/testutil"
	. "github.com/mongodb/mongo-go-driver/mongo/private/ops"
	"github.com/mongodb/mongo-go-driver/mongo/writeconcern"
	"github.com/stretchr/testify/require"
)

func TestListDatabases(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)
	testutil.AutoDropCollection(t)
	testutil.AutoInsertDocs(t, writeconcern.New(writeconcern.WMajority()), bson.NewDocument(bson.EC.Int32("_id", 1)))

	s := getServer(t)
	cursor, err := ListDatabases(context.Background(), s, nil, ListDatabasesOptions{})
	require.NoError(t, err)

	var next = bson.NewDocument()
	dbs := make([]string, 0)

	for cursor.Next(context.Background()) {
		err = cursor.Decode(next)
		require.NoError(t, err)

		elem, err := next.Lookup("name")
		require.NoError(t, err)
		require.Equal(t, elem.Value().Type(), bson.TypeString)
		dbs = append(dbs, elem.Value().StringValue())

		if elem.Value().StringValue() == testutil.DBName(t) {
			break
		}
	}

	require.NoError(t, cursor.Err())
	require.NoError(t, cursor.Close(context.Background()))
}

func TestListDatabasesWithMaxTimeMS(t *testing.T) {
	t.Skip("max time is flaky on the server")
	t.Parallel()
	testutil.Integration(t)

	s := getServer(t)

	if testutil.EnableMaxTimeFailPoint(t, s) != nil {
		t.Skip("skipping maxTimeMS test when max time failpoint is disabled")
	}
	defer testutil.DisableMaxTimeFailPoint(t, s)

	_, err := ListDatabases(context.Background(), s, nil, ListDatabasesOptions{
		MaxTime: time.Millisecond,
	})
	require.Error(t, err)
	// Hacky check for the error message.  Should we be returning a more structured error?
	require.Contains(t, err.Error(), "operation exceeded time limit")
}
