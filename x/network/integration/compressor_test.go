// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"testing"

	"context"
	"os"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"go.mongodb.org/mongo-driver/x/network/command"
	"go.mongodb.org/mongo-driver/x/network/description"
)

func TestCompression(t *testing.T) {
	comp := os.Getenv("MONGO_GO_DRIVER_COMPRESSOR")
	if len(comp) == 0 {
		t.Skip("Skipping because no compressor specified")
	}

	server, err := testutil.Topology(t).SelectServer(context.Background(), description.WriteSelector())
	noerr(t, err)

	wc := writeconcern.New(writeconcern.WMajority())
	collOne := testutil.ColName(t)

	testutil.DropCollection(t, testutil.DBName(t), collOne)
	testutil.InsertDocs(t, testutil.DBName(t), collOne, wc, bsonx.Doc{{"name", bsonx.String("compression_test")}})

	cmd := &command.Read{
		DB:      testutil.DBName(t),
		Command: bsonx.Doc{{"serverStatus", bsonx.Int32(1)}},
	}

	ctx := context.Background()
	rw, err := server.Connection(ctx)
	noerr(t, err)

	rdr, err := cmd.RoundTrip(ctx, server.SelectedDescription(), rw)
	noerr(t, err)

	result, err := bsonx.ReadDoc(rdr)
	noerr(t, err)

	serverVersion, err := result.LookupErr("version")
	noerr(t, err)

	if testutil.CompareVersions(t, serverVersion.StringValue(), "3.4") < 0 {
		t.Skip("skipping compression test for version < 3.4")
	}

	networkVal, err := result.LookupErr("network")
	noerr(t, err)

	require.Equal(t, networkVal.Type(), bson.TypeEmbeddedDocument)

	compressionVal, err := networkVal.Document().LookupErr("compression")
	noerr(t, err)

	snappy, err := compressionVal.Document().LookupErr("snappy")
	noerr(t, err)

	compressorKey := "compressor"
	compareTo36 := testutil.CompareVersions(t, serverVersion.StringValue(), "3.6")
	if compareTo36 < 0 {
		compressorKey = "compressed"
	}
	compressor, err := snappy.Document().LookupErr(compressorKey)
	noerr(t, err)

	bytesIn, err := compressor.Document().LookupErr("bytesIn")
	noerr(t, err)

	require.True(t, bytesIn.IsNumber())
	require.True(t, bytesIn.Int64() > 0)
}
