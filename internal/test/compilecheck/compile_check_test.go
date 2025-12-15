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
//	go test -v -run '^TestCompileCheck/go:1.19$'
//	go test -v -run '^TestCompileCheck/go:1\.(19|20)$'
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

	// Navigate up from internal/test/compilecheck to the project root.
	rootDir := filepath.Dir(filepath.Dir(filepath.Dir(cwd)))

	// Build the image and start one container we can reuse for all subtests.
	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:       rootDir,
			Dockerfile:    "Dockerfile",
			PrintBuildLog: true,
		},
		Entrypoint: []string{"tail", "-f", "/dev/null"},
		WorkingDir: "/workspace",
	}

	genReq := testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true}

	container, err := testcontainers.GenericContainer(context.Background(), genReq)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, container.Terminate(context.Background()))
	}()

	for _, ver := range goVersions {
		ver := ver // capture
		t.Run("go:"+ver, func(t *testing.T) {
			cmd := fmt.Sprintf("PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=go%[1]s.0+auto go version || PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=go%[1]s go version", ver)

			exit, out, err := container.Exec(context.Background(), []string{"bash", "-lc", cmd})
			require.NoError(t, err)

			b, err := io.ReadAll(out)

			require.NoError(t, err)
			require.Equal(t, 0, exit, "go version failed: %s", b)
			require.Contains(t, string(b), "go"+ver, "unexpected go version: %s", b)

			// Standard build.
			exitCode, outputReader, err := container.Exec(context.Background(), []string{
				"sh", "-c", fmt.Sprintf("PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=go%s go build -buildvcs=false ./...", ver),
			})
			require.NoError(t, err)

			output, err := io.ReadAll(outputReader)
			require.NoError(t, err)

			require.Equal(t, 0, exitCode, "standard build failed: %s", output)

			// Dynamic linking build.
			exitCode, outputReader, err = container.Exec(context.Background(), []string{
				"sh", "-c", fmt.Sprintf("PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=go%s go build -buildvcs=false -buildmode=plugin ./...", ver),
			})
			require.NoError(t, err)

			output, err = io.ReadAll(outputReader)
			require.NoError(t, err)

			require.Equal(t, 0, exitCode, "dynamic linking build failed: %s", output)

			// Build with build tags.
			exitCode, outputReader, err = container.Exec(context.Background(), []string{
				"sh", "-c", fmt.Sprintf("PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=go%s go build -buildvcs=false -tags=cse,gssapi,mongointernal ./...", ver),
			})
			require.NoError(t, err)

			output, err = io.ReadAll(outputReader)
			require.NoError(t, err)

			require.Equal(t, 0, exitCode, "build with build tags failed: %s", output)

			// Build for each architecture.
			for _, architecture := range architectures {
				exitCode, outputReader, err := container.Exec(
					context.Background(),
					[]string{"sh", "-c", fmt.Sprintf("PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=go%s GOOS=linux GOARCH=%s go build -buildvcs=false ./...", ver, architecture)},
				)
				require.NoError(t, err)

				output, err := io.ReadAll(outputReader)
				require.NoError(t, err)

				require.Equal(t, 0, exitCode, "build failed for architecture %s: %s", architecture, output)
			}

			t.Logf("compilation checks passed for Go ver %s", ver)
		})
	}
}
