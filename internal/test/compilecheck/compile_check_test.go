// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"golang.org/x/mod/semver"
)

const minSupportedVersion = "1.18"

func TestCompileCheck(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	internalDir := filepath.Dir(filepath.Dir(filepath.Dir(cwd)))

	versions, err := getDockerGolangImages()
	require.NoError(t, err)

	for _, version := range versions {
		version := version // Capture range variable.

		image := fmt.Sprintf("golang:%s", version)
		t.Run(image, func(t *testing.T) {
			t.Parallel()

			req := testcontainers.ContainerRequest{
				Image: image,
				Cmd:   []string{"tail", "-f", "/dev/null"},
				Mounts: []testcontainers.ContainerMount{
					testcontainers.BindMount(internalDir, "/workspace"),
				},
				WorkingDir: "/workspace",
				Env: map[string]string{
					"GO_VERSION": version,
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

func getDockerGolangImages() ([]string, error) {
	url := "https://hub.docker.com/v2/repositories/library/golang/tags?page_size=100"

	versionSet := map[string]bool{}
	for url != "" {
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()

		var data struct {
			Results []struct {
				Name string `json:"name"`
			} `json:"results"`
			Next string `json:"next"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, err
		}

		finished := false
		for _, tag := range data.Results {
			// Skip tags that don't start with a digit (e.g. alpine, buster, etc)
			if len(tag.Name) == 0 || tag.Name[0] < '0' || tag.Name[0] > '9' {
				continue
			}

			// Extract the base version (e.g. 1.18.1 from 1.18.1-alpine)
			base := strings.Split(tag.Name, "-")[0]

			// If its not at least three characters, do nothing.
			if len(base) < 3 {
				continue
			}

			// Skip release candidates.
			if strings.Contains(base, "rc") {
				continue
			}

			// Only take major versions.
			if strings.Count(base, ".") > 1 {
				continue
			}

			//if compareVersions(base, minSupportedVersion) >= 0 {
			if semver.Compare("v"+base, "v"+minSupportedVersion) >= 0 {
				versionSet[base] = true

				continue
			} else {
				finished = true
			}
		}

		if finished {
			break
		}

		url = data.Next
	}

	versions := []string{}
	for v := range versionSet {
		versions = append(versions, v)
	}

	return versions, nil
}
