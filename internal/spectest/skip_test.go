// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package spectest

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/require"
)

// fakeT records whether Skip was called and the message.
type fakeT struct {
	skipped bool
	skipMsg string
}

func (f *fakeT) Name() string         { return f.skipMsg }
func (f *fakeT) Skipf(string, ...any) { f.skipped = true }

func TestCheckSkip(t *testing.T) {
	const (
		unconditionalTest = "TestFoo/unconditional"
		topoTest          = "TestFoo/topo_gated"
		versionTest       = "TestFoo/version_gated"
		unknownTest       = "TestFoo/unknown"
	)

	// Temporarily override skipTests with controlled cases.
	orig := skipTests
	t.Cleanup(func() { skipTests = orig })

	skipTests = map[string][]skipCase{
		"unconditional skip": {
			{
				tests: []string{unconditionalTest},
			},
		},
		"topology-gated skip": {
			{
				tests:      []string{topoTest},
				topologies: []string{"sharded"},
			},
		},
		"version-gated skip": {
			{
				tests:            []string{versionTest},
				minServerVersion: "8.0",
			},
		},
	}

	tests := []struct {
		name        string
		testName    string
		opts        []Option
		wantSkipped bool
	}{
		{
			name:        "unconditional skip fires",
			testName:    unconditionalTest,
			wantSkipped: true,
		},
		{
			name:        "unknown test is not skipped",
			testName:    unknownTest,
			wantSkipped: false,
		},
		{
			name:        "topology match skips",
			testName:    topoTest,
			opts:        []Option{WithTopology("sharded")},
			wantSkipped: true,
		},
		{
			name:        "topology mismatch does not skip",
			testName:    topoTest,
			opts:        []Option{WithTopology("replicaset")},
			wantSkipped: false,
		},
		{
			name:        "server version at minimum skips",
			testName:    versionTest,
			opts:        []Option{WithServerVersion("8.0")},
			wantSkipped: true,
		},
		{
			name:        "server version above minimum skips",
			testName:    versionTest,
			opts:        []Option{WithServerVersion("9.0")},
			wantSkipped: true,
		},
		{
			name:        "server version below minimum does not skip",
			testName:    versionTest,
			opts:        []Option{WithServerVersion("7.0")},
			wantSkipped: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ft := &fakeT{skipMsg: tc.testName}
			CheckSkip(ft, tc.opts...)
			require.Equal(t, tc.wantSkipped, ft.skipped)
		})
	}
}
