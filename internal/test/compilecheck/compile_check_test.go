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
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

func getVersions(t *testing.T) []string {
	t.Helper()

	env := os.Getenv("GO_VERSIONS")
	if env == "" {
		t.Skip("GO_VERSIONS environment variable not set")
	}

	return strings.Split(env, ",")
}

func TestCompileCheck(t *testing.T) {
	versions := getVersions(t)
	cwd, err := os.Getwd()
	require.NoError(t, err)

	rootDir := filepath.Dir(filepath.Dir(filepath.Dir(cwd)))

	for _, version := range versions {
		version := version // Capture range variable.

		image := fmt.Sprintf("golang:%s", version)
		t.Run(image, func(t *testing.T) {
			t.Parallel()

			req := testcontainers.ContainerRequest{
				Image:      image,
				Cmd:        []string{"tail", "-f", "/dev/null"},
				WorkingDir: "/workspace",
				HostConfigModifier: func(hostConfig *container.HostConfig) {
					hostConfig.Binds = []string{fmt.Sprintf("%s:/workspace", rootDir)}
				},
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
