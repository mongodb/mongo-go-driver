// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

var versions = []string{
	"1.19",
	"1.20",
	"1.21",
	"1.22",
	"1.23",
	"1.24",
	"1.25",
}

func TestCompileCheck(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	rootDir := filepath.Dir(filepath.Dir(filepath.Dir(cwd)))

	for _, version := range versions {
		version := version // Capture range variable.

		image := fmt.Sprintf("golang:%s", version)
		t.Run(image, func(t *testing.T) {
			t.Parallel()

			req := testcontainers.ContainerRequest{
				Image: image,
				Cmd:   []string{"tail", "-f", "/dev/null"},
				Mounts: []testcontainers.ContainerMount{
					testcontainers.BindMount(rootDir, "/workspace"),
				},
				WorkingDir: "/workspace",
				Env: map[string]string{
					"GC": "go",
					// Compilation modules are not part of the workspace as testcontainers requires
					// a version of klauspost/compress not supported by the Go Driver / other modules
					// in the workspace.
					"GOWORK": "off",
				},
			}

			genReq := testcontainers.GenericContainerRequest{
				ContainerRequest: req,
				Started:          true,
			}

			container, err := testcontainers.GenericContainer(context.Background(), genReq)
			require.NoError(t, err)

			defer func() {
				err := container.Terminate(context.Background())
				require.NoError(t, err)
			}()

			exitCode, outputReader, err := container.Exec(context.Background(), []string{"bash", "etc/compile_check.sh"})
			require.NoError(t, err)

			output, err := io.ReadAll(outputReader)
			require.NoError(t, err)

			t.Logf("output: %s", output)
			assert.Equal(t, 0, exitCode)
		})
	}
}
