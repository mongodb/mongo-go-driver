// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerOptions(t *testing.T) {
	t.Run("serverConfig validation", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name string
			opts []ServerOption
			err  error
		}{
			{
				"minPoolSize < default maxPoolSize",
				[]ServerOption{
					WithMinConnections(func(uint64) uint64 { return uint64(64) }),
				},
				nil,
			},
			{
				"minPoolSize > default maxPoolSize",
				[]ServerOption{
					WithMinConnections(func(uint64) uint64 { return uint64(128) }),
				},
				errors.New("options should be: MaxConnections >= MinConnections; got: 128 MinConnections, 100 MaxConnections"),
			},
			{
				"minPoolSize < maxPoolSize",
				[]ServerOption{
					WithMinConnections(func(uint64) uint64 { return uint64(128) }),
					WithMaxConnections(func(uint64) uint64 { return uint64(256) }),
				},
				nil,
			},
			{
				"minPoolSize == maxPoolSize",
				[]ServerOption{
					WithMinConnections(func(uint64) uint64 { return uint64(128) }),
					WithMaxConnections(func(uint64) uint64 { return uint64(128) }),
				},
				nil,
			},
			{
				"minPoolSize > maxPoolSize",
				[]ServerOption{
					WithMinConnections(func(uint64) uint64 { return uint64(64) }),
					WithMaxConnections(func(uint64) uint64 { return uint64(32) }),
				},
				errors.New("options should be: MaxConnections >= MinConnections; got: 64 MinConnections, 32 MaxConnections"),
			},
			{
				"maxPoolSize == 0",
				[]ServerOption{
					WithMaxConnections(func(uint64) uint64 { return uint64(0) }),
				},
				nil,
			},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := newServerConfig(tc.opts...).validate()
				assert.Equal(t, tc.err, err, "expected error %v, got %v", tc.err, err)
			})
		}
	})
}
