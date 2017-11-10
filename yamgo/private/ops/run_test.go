// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops_test

import (
	"context"
	"testing"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/yamgo/internal/testutil"
	. "github.com/10gen/mongo-go-driver/yamgo/private/ops"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Parallel()
	testutil.Integration(t)

	server := getServer(t)

	ctx := context.Background()
	result := bson.M{}

	err := Run(
		ctx,
		server,
		"admin",
		bson.D{bson.NewDocElem("getnonce", 1)},
		result,
	)
	require.NoError(t, err)
	require.Equal(t, float64(1), result["ok"])
	require.NotEqual(t, "", result["nonce"], "MongoDB returned empty nonce")

	result = bson.M{}
	err = Run(
		ctx,
		server,
		"admin",
		bson.D{bson.NewDocElem("ping", 1)},
		result,
	)

	require.NoError(t, err)
	require.Equal(t, float64(1), result["ok"], "Unable to ping MongoDB")

}
