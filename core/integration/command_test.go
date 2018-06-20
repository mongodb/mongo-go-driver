// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/address"
	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/result"
	"github.com/mongodb/mongo-go-driver/core/topology"
	"github.com/mongodb/mongo-go-driver/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestCommand(t *testing.T) {
	noerr := func(t *testing.T, err error) {
		t.Helper()
		if err != nil {
			t.Errorf("Unepexted error: %v", err)
			t.FailNow()
		}
	}
	t.Parallel()

	server, err := topology.ConnectServer(context.Background(), address.Address(*host), serveropts(t)...)
	noerr(t, err)

	ctx := context.Background()
	var result *bson.Document

	cmd := &command.Read{
		DB:      "admin",
		Command: bson.NewDocument(bson.EC.Int32("getnonce", 1)),
	}
	rw, err := server.Connection(ctx)
	noerr(t, err)

	rdr, err := cmd.RoundTrip(ctx, server.SelectedDescription(), rw)
	noerr(t, err)

	result, err = bson.ReadDocument(rdr)
	noerr(t, err)

	val, err := result.LookupErr("ok")
	noerr(t, err)
	if got, want := val.Type(), bson.TypeDouble; got != want {
		t.Errorf("Did not get correct type for 'ok'. got %s; want %s", got, want)
	}
	if got, want := val.Double(), float64(1); got != want {
		t.Errorf("Did not get correct value for 'ok'. got %f; want %f", got, want)
	}

	val, err = result.LookupErr("nonce")
	require.NoError(t, err)
	require.Equal(t, val.Type(), bson.TypeString)
	require.NotEqual(t, "", val.StringValue(), "MongoDB returned empty nonce")

	result.Reset()
	cmd.Command = bson.NewDocument(bson.EC.Int32("ping", 1))

	rw, err = server.Connection(ctx)
	noerr(t, err)
	rdr, err = cmd.RoundTrip(ctx, server.SelectedDescription(), rw)
	noerr(t, err)

	result, err = bson.ReadDocument(rdr)
	require.NoError(t, err)

	val, err = result.LookupErr("ok")
	require.NoError(t, err)
	require.Equal(t, val.Type(), bson.TypeDouble)
	require.Equal(t, float64(1), val.Double(), "Unable to ping MongoDB")
}

func TestWriteCommands(t *testing.T) {
	t.Run("Insert", func(t *testing.T) {
		t.Run("Should return write error", func(t *testing.T) {
			ctx := context.TODO()
			server, err := testutil.Topology(t).SelectServer(context.Background(), description.WriteSelector())
			noerr(t, err)
			conn, err := server.Connection(context.Background())
			noerr(t, err)

			cmd := &command.Insert{
				WriteConcern: nil,
				NS:           command.Namespace{DB: dbName, Collection: testutil.ColName(t)},
				Docs:         []*bson.Document{bson.NewDocument(bson.EC.String("_id", "helloworld"))},
			}
			_, err = cmd.RoundTrip(ctx, server.SelectedDescription(), conn)
			noerr(t, err)

			conn, err = server.Connection(context.Background())
			noerr(t, err)
			res, err := cmd.RoundTrip(ctx, server.SelectedDescription(), conn)
			noerr(t, err)
			if len(res.WriteErrors) != 1 {
				t.Errorf("Expected to get a write error. got %d; want %d", len(res.WriteErrors), 1)
				t.FailNow()
			}
			want := result.WriteError{Code: 11000}
			if res.WriteErrors[0].Code != want.Code {
				t.Errorf("Expected error codes to match. want %d; got %d", res.WriteErrors[0].Code, want.Code)
			}
		})
	})
}
