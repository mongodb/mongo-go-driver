// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"testing"

	"github.com/10gen/mongo-go-driver/mongo/internal/testutil"
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
