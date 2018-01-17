// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"os"
	"testing"

	"github.com/10gen/mongo-go-driver/mongo/internal/testutil"
	"github.com/10gen/mongo-go-driver/mongo/readpref"
	"github.com/skriptble/wilson/bson"
	"github.com/stretchr/testify/require"
)

func createTestClient(t *testing.T) *Client {
	return &Client{
		cluster:        testutil.Cluster(t),
		connString:     testutil.ConnString(t),
		readPreference: readpref.Primary(),
	}
}

func TestNewClient(t *testing.T) {
	t.Parallel()

	c := createTestClient(t)
	require.NotNil(t, c.cluster)
}

func TestClient_Database(t *testing.T) {
	t.Parallel()

	dbName := "foo"

	c := createTestClient(t)
	db := c.Database(dbName)
	require.Equal(t, db.Name(), dbName)
	require.Exactly(t, c, db.Client())
}

func TestClient_TLSConnection(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip()
	}

	caFile := os.Getenv("MONGO_GO_DRIVER_CA_FILE")

	if len(caFile) == 0 {
		t.Skip()
	}

	c := createTestClient(t)
	db := c.Database("test")

	result, err := db.RunCommand(nil, bson.NewDocument(1).Append(bson.C.Int32("serverStatus", 1)), nil)
	require.NoError(t, err)

	security, err := result.Lookup("security")
	require.Nil(t, err)

	require.Equal(t, security.Value().Type(), bson.TypeEmbeddedDocument)

	_, found := security.Value().ReaderDocument().Lookup("SSLServerSubjectName")
	require.Nil(t, found)

	_, found = security.Value().ReaderDocument().Lookup("SSLServerHasCertificateAuthority")
	require.Nil(t, found)

}
