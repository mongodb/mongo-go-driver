// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops_test

import (
	"context"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/internal/testutil"
	. "github.com/mongodb/mongo-go-driver/mongo/private/ops"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)

	server := getServer(t)

	ctx := context.Background()
	var result *bson.Document

	rdr, err := Run(
		ctx,
		server,
		"admin",
		bson.NewDocument(bson.EC.Int32("getnonce", 1)),
	)
	require.NoError(t, err)

	result, err = bson.ReadDocument(rdr)
	require.NoError(t, err)

	elem, err := result.Lookup("ok")
	require.NoError(t, err)
	require.Equal(t, elem.Value().Type(), bson.TypeDouble)
	require.Equal(t, float64(1), elem.Value().Double())

	elem, err = result.Lookup("nonce")
	require.NoError(t, err)
	require.Equal(t, elem.Value().Type(), bson.TypeString)
	require.NotEqual(t, "", elem.Value().StringValue(), "MongoDB returned empty nonce")

	result.Reset()
	rdr, err = Run(
		ctx,
		server,
		"admin",
		bson.NewDocument(bson.EC.Int32("ping", 1)),
	)
	require.NoError(t, err)

	result, err = bson.ReadDocument(rdr)
	require.NoError(t, err)

	elem, err = result.Lookup("ok")
	require.NoError(t, err)
	require.Equal(t, elem.Value().Type(), bson.TypeDouble)
	require.Equal(t, float64(1), elem.Value().Double(), "Unable to ping MongoDB")

}
