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

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/mongo/internal/testutil"
	. "github.com/10gen/mongo-go-driver/mongo/private/ops"
	"github.com/stretchr/testify/require"
)

func TestListDatabases(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)
	testutil.AutoDropCollection(t)
	testutil.AutoInsertDocs(t, bson.D{bson.NewDocElem("_id", 1)})

	s := getServer(t)
	cursor, err := ListDatabases(context.Background(), s, ListDatabasesOptions{})
	require.NoError(t, err)

	var next bson.M
	var found bool
	for cursor.Next(context.Background(), &next) {
		if next["name"] == testutil.DBName(t) {
			found = true
			break
		}
	}
	require.True(t, found, "Expected to have listed at least database named %v", testutil.DBName(t))
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

	_, err := ListDatabases(context.Background(), s, ListDatabasesOptions{
		MaxTime: time.Millisecond,
	})
	require.Error(t, err)
	// Hacky check for the error message.  Should we be returning a more structured error?
	require.Contains(t, err.Error(), "operation exceeded time limit")
}
