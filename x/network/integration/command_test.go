// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
	"go.mongodb.org/mongo-driver/x/network/address"
	"go.mongodb.org/mongo-driver/x/network/command"
	"go.mongodb.org/mongo-driver/x/network/description"
	"go.mongodb.org/mongo-driver/x/network/result"
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
	var result bsonx.Doc

	cmd := &command.Read{
		DB:      "admin",
		Command: bsonx.Doc{{"getnonce", bsonx.Int32(1)}},
	}
	rw, err := server.Connection(ctx)
	noerr(t, err)

	rdr, err := cmd.RoundTrip(ctx, server.SelectedDescription(), rw)
	noerr(t, err)

	result, err = bsonx.ReadDoc(rdr)
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

	result = result[:0]
	cmd.Command = bsonx.Doc{{"ping", bsonx.Int32(1)}}

	rw, err = server.Connection(ctx)
	noerr(t, err)
	rdr, err = cmd.RoundTrip(ctx, server.SelectedDescription(), rw)
	noerr(t, err)

	result, err = bsonx.ReadDoc(rdr)
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
				Docs:         []bsonx.Doc{{{"_id", bsonx.String("helloworld")}}},
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
