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

// goVersions is the list of Go versions to test compilation against.
// To run tests for specific version(s), use the -run flag:
//
//	go test -v -run '^TestCompileCheck/go:1.19$'
//	go test -v -run '^TestCompileCheck/go:1\.(19|20)$'
var goVersions = []string{
	"1.19", // Minimum supported Go version for mongo-driver v2
	"1.20",
	"1.21",
	"1.22",
	"1.23",
	"1.24",
	"1.25", // Test suite Go Version
}

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

// goExecConfig contains optional configuration for execGo.
type goExecConfig struct {
	version string            // Optional: Go version to use with GOTOOLCHAIN. If empty, uses default.
	env     map[string]string // Optional: Additional environment variables.
}

// execContainer executes a shell command in the container and validates its output.
func execContainer(t *testing.T, c testcontainers.Container, cmd string) string {
	t.Helper()

	exit, out, err := c.Exec(context.Background(), []string{"bash", "-lc", cmd})
	require.NoError(t, err)

	b, err := io.ReadAll(out)
	require.NoError(t, err)
	require.Equal(t, 0, exit, "command failed: %s", b)

	s := string(b)
	// Strip leading non-printable bytes (some Docker/TTY combos emit these).
	for len(s) > 0 && s[0] < 0x20 {
		s = s[1:]
	}
	return s
}

// execGo runs a Go command, trying GOTOOLCHAIN=goX.Y.0 first, then goX.Y.
func execGo(t *testing.T, c testcontainers.Container, cfg *goExecConfig, args ...string) string {
	t.Helper()

	if cfg == nil {
		cfg = &goExecConfig{}
	}

	envParts := []string{"PATH=/usr/local/go/bin:$PATH"}
	for k, v := range cfg.env {
		envParts = append(envParts, fmt.Sprintf("%s=%s", k, v))
	}
	envStr := strings.Join(envParts, " ")
	goArgs := strings.Join(args, " ")

	var cmd string
	if cfg.version != "" {
		primaryCmd := fmt.Sprintf("%s GOTOOLCHAIN=go%s.0 go %s 2>&1", envStr, cfg.version, goArgs)
		fallbackCmd := fmt.Sprintf("%s GOTOOLCHAIN=go%s go %s 2>&1", envStr, cfg.version, goArgs)
		cmd = fmt.Sprintf("%s || %s", primaryCmd, fallbackCmd)
	} else {
		cmd = fmt.Sprintf("%s go %s 2>&1", envStr, goArgs)
	}

	return execContainer(t, c, cmd)
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
		Files: []testcontainers.ContainerFile{
			{
				Reader:            strings.NewReader(mainGo),
				ContainerFilePath: "/workspace/main.go",
				FileMode:          0o644,
			},
		},
		// Entrypoint is set to "tail -f /dev/null" so the container stays running and available to execute multiple shell commands as needed during tests.
		// This keeps the container alive and ready for exec calls, rather than immediately exiting.
		Entrypoint: []string{"tail", "-f", "/dev/null"},
		WorkingDir: "/workspace",
	}

	genReq := testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true}

	container, err := testcontainers.GenericContainer(context.Background(), genReq)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, container.Terminate(context.Background()))
	})

	testSuiteVersion := goVersions[len(goVersions)-1]

	// Initialize Go module and download dependencies using the test suite Go version.
	execGo(t, container, &goExecConfig{version: testSuiteVersion}, "mod", "init", "compilecheck")
	execGo(t, container, nil, "mod", "edit", "-replace=go.mongodb.org/mongo-driver/v2=/mongo-go-driver")
	execGo(t, container, &goExecConfig{version: testSuiteVersion}, "mod", "tidy")

	// Set minimum Go version to what the driver claims (first version in our test list).
	execGo(t, container, nil, "mod", "edit", "-go="+goVersions[0])

	for _, ver := range goVersions {
		ver := ver // capture
		t.Run("go:"+ver, func(t *testing.T) {
			t.Parallel()

			versionCfg := &goExecConfig{version: ver}

			// Verify the Go version is available.
			versionOutput := execGo(t, container, versionCfg, "version")
			require.Contains(t, versionOutput, "go"+ver, "unexpected go version: %s", versionOutput)

			execGo(t, container, versionCfg, "build", "-buildvcs=false", "-o", "/dev/null", "main.go")

			// Dynamic linking build.
			execGo(t, container, versionCfg, "build", "-buildvcs=false", "-buildmode=plugin", "-o", "/dev/null", "main.go")

			// Build with build tags.
			execGo(t, container, &goExecConfig{
				version: ver,
				env: map[string]string{
					"PKG_CONFIG_PATH": "/root/install/libmongocrypt/lib/pkgconfig",
					"CGO_CFLAGS":      "'-I/root/install/libmongocrypt/include'",
					"CGO_LDFLAGS":     "'-L/root/install/libmongocrypt/lib -Wl,-rpath,/root/install/libmongocrypt/lib'",
				},
			}, "build", "-buildvcs=false", "-tags=cse,gssapi,mongointernal", "-o", "/dev/null", "main.go")

			// Build for each architecture.
			for _, architecture := range architectures {
				architecture := architecture // capture
				t.Run("arch:"+architecture, func(t *testing.T) {
					t.Parallel()

					// Standard build.
					execGo(t, container, &goExecConfig{
						version: ver,
						env: map[string]string{
							"GOOS":   "linux",
							"GOARCH": architecture,
						},
					}, "build", "-buildvcs=false", "-o", "/dev/null", "main.go")

					t.Logf("compilation checks passed for go%s on %s", ver, architecture)
				})
			}
		})
	}
}
