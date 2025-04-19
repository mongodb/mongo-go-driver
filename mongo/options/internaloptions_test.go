// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/drivertest"
)

func TestSetClientOptions(t *testing.T) {
	t.Parallel()

	t.Run("set Crypt with driver.Crypt", func(t *testing.T) {
		t.Parallel()

		opts := &ClientOptions{}
		c := driver.NewCrypt(&driver.CryptOptions{})
		opts, err := SetInternalClientOptions(opts, map[string]any{
			"crypt": c,
		})
		require.NoError(t, err)
		require.Equal(t, c, opts.Crypt)
	})

	t.Run("set Crypt with driver.Deployment", func(t *testing.T) {
		t.Parallel()

		opts := &ClientOptions{}
		_, err := SetInternalClientOptions(opts, map[string]any{
			"crypt": &drivertest.MockDeployment{},
		})
		require.EqualError(t, err, "unexpected type for crypt")
	})

	t.Run("set Deployment with driver.Deployment", func(t *testing.T) {
		t.Parallel()

		opts := &ClientOptions{}
		d := &drivertest.MockDeployment{}
		opts, err := SetInternalClientOptions(opts, map[string]any{
			"deployment": d,
		})
		require.NoError(t, err)
		require.Equal(t, d, opts.Deployment)
	})

	t.Run("set Deployment with driver.Crypt", func(t *testing.T) {
		t.Parallel()

		opts := &ClientOptions{}
		_, err := SetInternalClientOptions(opts, map[string]any{
			"deployment": driver.NewCrypt(&driver.CryptOptions{}),
		})
		require.EqualError(t, err, "unexpected type for deployment")
	})

	t.Run("set unsupported option", func(t *testing.T) {
		t.Parallel()

		opts := &ClientOptions{}
		_, err := SetInternalClientOptions(opts, map[string]any{
			"unsupported": "unsupported",
		})
		require.EqualError(t, err, "unsupported option: unsupported")
	})
}
