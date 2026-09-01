// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package xoptions

import (
	"fmt"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/optionsutil"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/drivertest"
)

func TestSetInternalClientOptions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		key   string
		value any
	}{
		{
			key:   "authenticateToAnything",
			value: true,
		},
	}
	for _, tc := range cases {
		tc := tc

		t.Run(fmt.Sprintf("set %s", tc.key), func(t *testing.T) {
			t.Parallel()

			opts := options.Client()
			err := SetInternalClientOptions(opts, tc.key, tc.value)
			require.NoError(t, err, "error setting %s: %v", tc.key, err)
			v := optionsutil.Value(opts.Custom, tc.key)
			require.Equal(t, tc.value, v, "expected %v, got %v", tc.value, v)
		})
	}

	t.Run("set crypt", func(t *testing.T) {
		t.Parallel()

		c := driver.NewCrypt(&driver.CryptOptions{})
		opts := options.Client()
		err := SetInternalClientOptions(opts, "crypt", c)
		require.NoError(t, err, "error setting crypt: %v", err)
		require.Equal(t, c, opts.Crypt, "expected %v, got %v", c, opts.Crypt)
	})

	t.Run("set crypt - wrong type", func(t *testing.T) {
		t.Parallel()

		opts := options.Client()
		err := SetInternalClientOptions(opts, "crypt", &drivertest.MockDeployment{})
		require.EqualError(t, err, "unexpected type for \"crypt\": *drivertest.MockDeployment is not driver.Crypt")
	})

	t.Run("set deployment", func(t *testing.T) {
		t.Parallel()

		d := &drivertest.MockDeployment{}
		opts := options.Client()
		err := SetInternalClientOptions(opts, "deployment", d)
		require.NoError(t, err, "error setting deployment: %v", err)
		require.Equal(t, d, opts.Deployment, "expected %v, got %v", d, opts.Deployment)
	})

	t.Run("set deployment - wrong type", func(t *testing.T) {
		t.Parallel()

		opts := options.Client()
		err := SetInternalClientOptions(opts, "deployment", driver.NewCrypt(&driver.CryptOptions{}))
		require.EqualError(t, err, "unexpected type for \"deployment\": *driver.crypt is not driver.Deployment")
	})

	t.Run("set unsupported option", func(t *testing.T) {
		t.Parallel()

		opts := options.Client()
		err := SetInternalClientOptions(opts, "unsupported", "unsupported")
		require.EqualError(t, err, "unsupported option: \"unsupported\"")
	})
}

func TestSetInternalClientBulkWriteEntry(t *testing.T) {
	t.Parallel()

	t.Run("set collectionUUID", func(t *testing.T) {
		t.Parallel()

		uuid := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

		var internal optionsutil.Options
		err := SetInternalClientBulkWriteEntry(&internal, "collectionUUID", uuid)
		require.NoError(t, err, "SetInternalClientBulkWriteEntry error: %v", err)

		v := optionsutil.Value(internal, "collectionUUID")
		require.Equal(t, uuid, v)
	})

	t.Run("set collectionUUID - wrong type", func(t *testing.T) {
		t.Parallel()

		var internal optionsutil.Options
		err := SetInternalClientBulkWriteEntry(&internal, "collectionUUID", "not-a-slice")
		require.EqualError(t, err, "unexpected type for \"collectionUUID\": string is not []byte")
	})

	t.Run("set unsupported option", func(t *testing.T) {
		t.Parallel()

		var internal optionsutil.Options
		err := SetInternalClientBulkWriteEntry(&internal, "unsupported", "value")
		require.EqualError(t, err, "unsupported option: \"unsupported\"")
	})
}
