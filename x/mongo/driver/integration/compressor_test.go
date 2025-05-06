// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"os"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/integtest"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/operation"
)

func TestCompression(t *testing.T) {
	comp := os.Getenv("MONGO_GO_DRIVER_COMPRESSOR")
	if len(comp) == 0 {
		t.Skip("Skipping because no compressor specified")
	}

	wc := writeconcern.Majority()
	collOne := integtest.ColName(t)

	dropCollection(t, integtest.DBName(t), collOne)
	insertDocs(t, integtest.DBName(t), collOne, wc,
		bsoncore.BuildDocument(nil, bsoncore.AppendStringElement(nil, "name", "compression_test")),
	)

	cmd := operation.NewCommand(bsoncore.BuildDocument(nil, bsoncore.AppendInt32Element(nil, "serverStatus", 1))).
		Deployment(integtest.Topology(t)).
		Database(integtest.DBName(t))

	ctx := context.Background()
	err := cmd.Execute(ctx)
	noerr(t, err)
	result := cmd.Result()

	serverVersion, err := result.LookupErr("version")
	noerr(t, err)

	if integtest.CompareVersions(t, serverVersion.StringValue(), "3.4") < 0 {
		t.Skip("skipping compression test for version < 3.4")
	}

	networkVal, err := result.LookupErr("network")
	noerr(t, err)

	require.Equal(t, bsoncore.TypeEmbeddedDocument, networkVal.Type)

	compressionVal, err := networkVal.Document().LookupErr("compression")
	noerr(t, err)

	compressorDoc, err := compressionVal.Document().LookupErr(comp)
	noerr(t, err)

	compressorKey := "compressor"
	compareTo36 := integtest.CompareVersions(t, serverVersion.StringValue(), "3.6")
	if compareTo36 < 0 {
		compressorKey = "compressed"
	}
	compressor, err := compressorDoc.Document().LookupErr(compressorKey)
	noerr(t, err)

	bytesIn, err := compressor.Document().LookupErr("bytesIn")
	noerr(t, err)

	require.True(t, bytesIn.IsNumber())
	require.True(t, bytesIn.Int64() > 0)
}
