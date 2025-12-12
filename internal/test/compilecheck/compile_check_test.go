// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

const mainGo = `package main

import (
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func main() {
	_, _ = mongo.Connect(options.Client())
	fmt.Println(bson.D{{Key: "key", Value: "value"}})
}
`

// goVersions is the list of Go versions to test compilation against.
// To run tests for specific version(s), use the -run flag:
//
//	go test -v -run '^TestCompileCheck/golang:1.19$'
//	go test -v -run '^TestCompileCheck/golang:1\.(19|20)$'
var goVersions = []string{"1.19", "1.20", "1.21", "1.22", "1.23", "1.24", "1.25"}
var architectures = []string{
	"386",
	"amd64",
	"arm",
	"arm64",
	"mips",
	"mips64",
	"mips64le",
	"mipsle",
	"ppc64",
	"ppc64le",
	"riscv64",
	"s390x",
}

func TestCompileCheck(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	rootDir := filepath.Dir(filepath.Dir(filepath.Dir(cwd)))

	for _, version := range goVersions {
		version := version // Capture range variable.

		image := fmt.Sprintf("golang:%s", version)
		t.Run(image, func(t *testing.T) {
			t.Parallel()

			req := testcontainers.ContainerRequest{
				Image: image,
				// Keep container running so we can Exec commands into it.
				Cmd:        []string{"tail", "-f", "/dev/null"},
				WorkingDir: "/app",
				HostConfigModifier: func(hostConfig *container.HostConfig) {
					hostConfig.Binds = []string{fmt.Sprintf("%s:/driver", rootDir)}
				},
				Files: []testcontainers.ContainerFile{
					{
						ContainerFilePath: "/app/main.go",
						Reader:            bytes.NewReader([]byte(mainGo)),
					},
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

			// Initialize go module and set up replace directive to use local driver.
			setupCmds := [][]string{
				{"go", "mod", "init", "app"},
				{"go", "mod", "edit", "-replace", "go.mongodb.org/mongo-driver/v2=/driver"},
				{"go", "mod", "tidy"},
			}

			for _, cmd := range setupCmds {
				exitCode, outputReader, err := container.Exec(context.Background(), cmd)
				require.NoError(t, err)

				output, err := io.ReadAll(outputReader)
				require.NoError(t, err)

				require.Equal(t, 0, exitCode, "command %v failed: %s", cmd, output)
			}

			// Standard build.
			exitCode, outputReader, err := container.Exec(context.Background(), []string{"go", "build", "-buildvcs=false", "./..."})
			require.NoError(t, err)

			output, err := io.ReadAll(outputReader)
			require.NoError(t, err)

			require.Equal(t, 0, exitCode, "standard build failed: %s", output)

			exitCode, outputReader, err = container.Exec(context.Background(), []string{"go", "build", "-buildvcs=false", "-buildmode=plugin", "./..."})
			require.NoError(t, err)

			output, err = io.ReadAll(outputReader)
			require.NoError(t, err)

			require.Equal(t, 0, exitCode, "dynamic linking build failed: %s", output)

			// Build with tags (install libmongocrypt and gssapi headers).
			// Use the driver's install-libmongocrypt.sh script which is mounted at /driver.
			installCmds := [][]string{
				{"apt-get", "update"},
				{"apt-get", "install", "-y", "libkrb5-dev", "cmake", "libssl-dev", "git", "pkg-config"},
				{"bash", "/driver/etc/install-libmongocrypt.sh"},
			}

			for _, cmd := range installCmds {
				exitCode, outputReader, err = container.Exec(context.Background(), cmd)
				require.NoError(t, err)

				output, err = io.ReadAll(outputReader)
				require.NoError(t, err)

				require.Equal(t, 0, exitCode, "install command %v failed: %s", cmd, output)
			}

			// The install script creates an "install" directory in the current working directory (/app).
			exitCode, outputReader, err = container.Exec(context.Background(), []string{
				"sh", "-c", "PKG_CONFIG_PATH=/app/install/lib/pkgconfig " +
					"LD_LIBRARY_PATH=/app/install/lib " +
					"go build -buildvcs=false -tags=cse,gssapi,mongointernal ./...",
			})
			require.NoError(t, err)

			output, err = io.ReadAll(outputReader)
			require.NoError(t, err)

			require.Equal(t, 0, exitCode, "build with build tags failed: %s", output)

			for _, architecture := range architectures {
				exitCode, outputReader, err := container.Exec(
					context.Background(),
					[]string{"sh", "-c", fmt.Sprintf("GOOS=linux GOARCH=%s go build -buildvcs=false ./...", architecture)},
				)
				require.NoError(t, err)

				output, err := io.ReadAll(outputReader)
				require.NoError(t, err)

				require.Equal(t, 0, exitCode, "build failed for architecture %s: %s", architecture, output)
			}

			t.Logf("compilation checks passed for %s", image)
		})
	}
}
