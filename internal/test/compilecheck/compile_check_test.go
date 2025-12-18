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

const goMod = `module compilecheck

go 1.19

require go.mongodb.org/mongo-driver/v2 v2.1.0

replace go.mongodb.org/mongo-driver/v2 => /mongo-go-driver
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

// goBuild constructs a build command that tries GOTOOLCHAIN=goX.Y.0 first, then falls back to goX.Y.
func goBuild(ver, workdir string, env, extraFlags []string) string {
	baseEnv := "PATH=/usr/local/go/bin:$PATH"
	envStr := baseEnv
	if len(env) > 0 {
		envStr = strings.Join(env, " ") + " " + baseEnv
	}

	flags := "-buildvcs=false"
	if len(extraFlags) > 0 {
		flags += " " + strings.Join(extraFlags, " ")
	}

	return fmt.Sprintf("cd %s && %s GOTOOLCHAIN=go%s.0 go build %s -o /dev/null main.go 2>&1 || %s GOTOOLCHAIN=go%s go build %s -o /dev/null main.go 2>&1",
		workdir, envStr, ver, flags, envStr, ver, flags)
}

func TestCompileCheck(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

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

	t.Cleanup(func() {
		require.NoError(t, container.Terminate(context.Background()))
	})

	// Write main.go into the container.
	exitCode, outputReader, err := container.Exec(context.Background(), []string{"sh", "-c", fmt.Sprintf("cat > /workspace/main.go << 'GOFILE'\n%s\nGOFILE", mainGo)})
	require.NoError(t, err)

	output, err := io.ReadAll(outputReader)
	require.NoError(t, err)
	require.Equal(t, 0, exitCode, "failed to write main.go: %s", output)

	// Write go.mod into the container.
	exitCode, outputReader, err = container.Exec(context.Background(), []string{"sh", "-c", fmt.Sprintf("cat > /workspace/go.mod << 'GOMOD'\n%s\nGOMOD", goMod)})
	require.NoError(t, err)

	output, err = io.ReadAll(outputReader)
	require.NoError(t, err)
	require.Equal(t, 0, exitCode, "failed to write go.mod: %s", output)

	exitCode, outputReader, err = container.Exec(context.Background(), []string{"sh", "-c", "cd /workspace && PATH=/usr/local/go/bin:$PATH go mod tidy 2>&1"})
	require.NoError(t, err)

	output, err = io.ReadAll(outputReader)
	require.NoError(t, err)
	require.Equal(t, 0, exitCode, "failed to tidy dependencies: %s", output)

	for _, ver := range goVersions {
		ver := ver // capture
		t.Run("go:"+ver, func(t *testing.T) {
			t.Parallel()

			t.Cleanup(func() {
				t.Logf("compilation checks passed for Go ver %s", ver)
			})

			// Each version gets its own workspace to avoid conflicts when running in parallel.
			workspace := fmt.Sprintf("/workspace-%s", ver)

			setupCmd := fmt.Sprintf("mkdir -p %[1]s && cp /workspace/main.go /workspace/go.mod /workspace/go.sum %[1]s/", workspace)
			exitCode, outputReader, err := container.Exec(context.Background(), []string{"sh", "-c", setupCmd})
			require.NoError(t, err)

			output, err := io.ReadAll(outputReader)
			require.NoError(t, err)
			require.Equal(t, 0, exitCode, "failed to setup workspace: %s", output)

			cmd := fmt.Sprintf("PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=go%[1]s.0+auto go version || PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=go%[1]s go version", ver)

			exit, out, err := container.Exec(context.Background(), []string{"bash", "-lc", cmd})
			require.NoError(t, err)

			b, err := io.ReadAll(out)

			require.NoError(t, err)
			require.Equal(t, 0, exit, "go version failed: %s", b)
			require.Contains(t, string(b), "go"+ver, "unexpected go version: %s", b)

			// Standard build.
			exitCode, outputReader, err = container.Exec(context.Background(), []string{
				"sh", "-c", goBuild(ver, workspace, nil, nil),
			})
			require.NoError(t, err)

			output, err = io.ReadAll(outputReader)
			require.NoError(t, err)

			require.Equal(t, 0, exitCode, "standard build failed: %s", output)

			// Dynamic linking build.
			exitCode, outputReader, err = container.Exec(context.Background(), []string{
				"sh", "-c", goBuild(ver, workspace, nil, []string{"-buildmode=plugin"}),
			})
			require.NoError(t, err)

			output, err = io.ReadAll(outputReader)
			require.NoError(t, err)

			require.Equal(t, 0, exitCode, "dynamic linking build failed: %s", output)

			// Build with build tags.
			cgoEnv := []string{
				"PKG_CONFIG_PATH=/root/install/libmongocrypt/lib/pkgconfig",
				"CGO_CFLAGS='-I/root/install/libmongocrypt/include'",
				"CGO_LDFLAGS='-L/root/install/libmongocrypt/lib -Wl,-rpath,/root/install/libmongocrypt/lib'",
			}
			exitCode, outputReader, err = container.Exec(context.Background(), []string{
				"sh", "-c", goBuild(ver, workspace, cgoEnv, []string{"-tags=cse,gssapi,mongointernal"}),
			})
			require.NoError(t, err)

			output, err = io.ReadAll(outputReader)
			require.NoError(t, err)

			require.Equal(t, 0, exitCode, "build with build tags failed: %s", output)

			// Build for each architecture.
			for _, architecture := range architectures {
				architecture := architecture // capture
				t.Run("arch:"+architecture, func(t *testing.T) {
					t.Parallel()

					archEnv := []string{"GOOS=linux", "GOARCH=" + architecture}
					exitCode, outputReader, err := container.Exec(
						context.Background(),
						[]string{"sh", "-c", goBuild(ver, workspace, archEnv, nil)},
					)
					require.NoError(t, err)

					output, err := io.ReadAll(outputReader)
					require.NoError(t, err)

					require.Equal(t, 0, exitCode, "build failed for architecture %s: %s", architecture, output)
				})
			}
		})
	}
}
