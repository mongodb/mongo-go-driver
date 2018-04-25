// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/internal/testutil"
	"github.com/stretchr/testify/require"
)

func createTestDatabase(t *testing.T, name *string) *Database {
	if name == nil {
		db := testutil.DBName(t)
		name = &db
	}

	client := createTestClient(t)
	return client.Database(*name)
}

func TestDatabase_initialize(t *testing.T) {
	t.Parallel()

	name := "foo"

	db := createTestDatabase(t, &name)
	require.Equal(t, db.name, name)
	require.NotNil(t, db.client)
}

func TestDatabase_RunCommand(t *testing.T) {
	t.Parallel()

	db := createTestDatabase(t, nil)

	result, err := db.RunCommand(context.Background(), bson.NewDocument(bson.EC.Int32("ismaster", 1)))
	require.NoError(t, err)

	isMaster, err := result.Lookup("ismaster")
	require.NoError(t, err)
	require.Equal(t, isMaster.Value().Type(), bson.TypeBoolean)
	require.Equal(t, isMaster.Value().Boolean(), true)

	ok, err := result.Lookup("ok")
	require.NoError(t, err)
	require.Equal(t, ok.Value().Type(), bson.TypeDouble)
	require.Equal(t, ok.Value().Double(), 1.0)
}

func TestDatabase_Drop(t *testing.T) {
	t.Parallel()

	name := "TestDatabase_Drop"

	db := createTestDatabase(t, &name)

	client := createTestClient(t)
	err := db.Drop(context.Background())
	require.NoError(t, err)
	list, err := client.ListDatabaseNames(context.Background(), nil)

	require.NoError(t, err)
	require.NotContains(t, list, name)

}
