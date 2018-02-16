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
	"github.com/mongodb/mongo-go-driver/mongo/internal/testutil/helpers"
	"github.com/mongodb/mongo-go-driver/mongo/private/cluster"
	"github.com/mongodb/mongo-go-driver/mongo/private/conn"
	"github.com/mongodb/mongo-go-driver/mongo/private/msg"
	. "github.com/mongodb/mongo-go-driver/mongo/private/ops"
	"github.com/mongodb/mongo-go-driver/mongo/readpref"
	"github.com/stretchr/testify/require"
)

func getServer(t *testing.T) *SelectedServer {

	c := testutil.Cluster(t)

	server, err := c.SelectServer(context.Background(), cluster.WriteSelector(), readpref.Primary())
	require.NoError(t, err)

	return &SelectedServer{
		Server:   server,
		ReadPref: readpref.Primary(),
	}
}

func find(t *testing.T, s Server, batchSize int32) bson.Reader {
	findCommand := bson.NewDocument(
		bson.EC.String("find", testutil.ColName(t)))

	if batchSize != 0 {
		findCommand.Append(bson.EC.Int32("batchSize", batchSize))
	}
	request := msg.NewCommand(
		msg.NextRequestID(),
		testutil.DBName(t),
		false,
		findCommand,
	)

	c, err := s.Connection(context.Background())
	require.NoError(t, err)
	defer testhelpers.RequireNoErrorOnClose(t, c)

	rdr, err := conn.ExecuteCommand(context.Background(), c, request)
	require.NoError(t, err)

	return rdr
}
