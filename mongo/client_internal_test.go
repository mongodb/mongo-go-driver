// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"os"
	"path"
	"testing"

	"fmt"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/internal/testutil"
	"github.com/mongodb/mongo-go-driver/mongo/readpref"
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

	result, err := db.RunCommand(context.Background(), bson.NewDocument(bson.EC.Int32("serverStatus", 1)))
	require.NoError(t, err)

	security, err := result.Lookup("security")
	require.Nil(t, err)

	require.Equal(t, security.Value().Type(), bson.TypeEmbeddedDocument)

	_, found := security.Value().ReaderDocument().Lookup("SSLServerSubjectName")
	require.Nil(t, found)

	_, found = security.Value().ReaderDocument().Lookup("SSLServerHasCertificateAuthority")
	require.Nil(t, found)

}

func TestClient_X509Auth(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip()
	}

	caFile := os.Getenv("MONGO_GO_DRIVER_CA_FILE")

	if len(caFile) == 0 || os.Getenv("AUTH") == "auth" {
		t.Skip()
	}

	const user = "C=US,ST=New York,L=New York City,O=MongoDB,OU=other,CN=external"

	c := createTestClient(t)
	db := c.Database("$external")

	// We don't care if the user doesn't already exist.
	_, _ = db.RunCommand(
		context.Background(),
		bson.NewDocument(
			bson.EC.String("dropUser", user),
		),
	)

	_, err := db.RunCommand(
		context.Background(),
		bson.NewDocument(
			bson.EC.String("createUser", user),
			bson.EC.ArrayFromElements("roles",
				bson.VC.DocumentFromElements(
					bson.EC.String("role", "readWrite"),
					bson.EC.String("db", "test"),
				),
			),
		),
	)
	require.NoError(t, err)

	basePath := path.Join("..", "data", "certificates")
	baseConnString := testutil.ConnString(t)
	cs := fmt.Sprintf(
		"%s&sslClientCertificateKeyFile=%s&authMechanism=MONGODB-X509",
		baseConnString.String(),
		path.Join(basePath, "client.pem"),
	)

	authClient, err := NewClient(cs)
	require.NoError(t, err)

	db = authClient.Database("test")
	rdr, err := db.RunCommand(
		context.Background(),
		bson.NewDocument(
			bson.EC.Int32("connectionStatus", 1),
		),
	)
	require.NoError(t, err)

	users, err := rdr.Lookup("authInfo", "authenticatedUsers")
	require.NoError(t, err)

	array := users.Value().MutableArray()

	for i := uint(0); i < uint(array.Len()); i++ {
		v, err := array.Lookup(i)
		require.NoError(t, err)

		rdr := v.ReaderDocument()
		var u struct {
			User string
			DB   string
		}

		if err := bson.Unmarshal(rdr, &u); err != nil {
			continue
		}

		if u.User == user && u.DB == "$external" {
			return
		}
	}

	t.Error("unable to find authenticated user")
}
