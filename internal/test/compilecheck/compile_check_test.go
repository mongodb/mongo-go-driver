// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
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
//
// To run tests for specific version(s), use the -run flag:
//
//	go test -v -run '^TestCompileCheck/go:1\.(25|26)$'
//
// To test only the minimum supported version, set COMPILE_CHECK_MIN_ONLY=true:
//
//	COMPILE_CHECK_MIN_ONLY=true go test -v
var goVersions = []string{
	"1.25", // Minimum supported Go version for mongo-driver v2
	"1.26", // Test suite Go Version
}

// parseVersion splits an "X.Y" version into its numeric components so
// versions compare numerically rather than lexically ("1.9" < "1.25").
//
// Versions must have exactly two numeric components: a patch-level entry such
// as "1.27.0" is rejected.
func parseVersion(t *testing.T, ver string) (int, int) {
	t.Helper()

	major, minor, ok := strings.Cut(ver, ".")
	require.True(t, ok, "malformed Go version in goVersions: %q", ver)

	majorNum, err := strconv.Atoi(major)
	require.NoError(t, err, "malformed Go version in goVersions: %q", ver)

	minorNum, err := strconv.Atoi(minor)
	require.NoError(t, err, "malformed Go version in goVersions: %q", ver)

	return majorNum, minorNum
}

// sortVersions returns versions sorted in ascending order.
func sortVersions(t *testing.T, versions []string) []string {
	t.Helper()

	v := slices.Clone(versions)
	slices.SortFunc(v, func(a, b string) int {
		aMajor, aMinor := parseVersion(t, a)
		bMajor, bMinor := parseVersion(t, b)

		if c := cmp.Compare(aMajor, bMajor); c != 0 {
			return c
		}

		return cmp.Compare(aMinor, bMinor)
	})

	return v
}

// testGoVersions returns the Go versions to compile-check, in ascending order.
// When COMPILE_CHECK_MIN_ONLY is set to a truthy value, only the minimum
// supported version is returned.
func testGoVersions(t *testing.T) []string {
	t.Helper()

	versions := sortVersions(t, goVersions)

	v := os.Getenv("COMPILE_CHECK_MIN_ONLY")
	if minOnly, err := strconv.ParseBool(v); err != nil && v != "" {
		require.NoError(t, err, "invalid COMPILE_CHECK_MIN_ONLY value: %q", v)
	} else if minOnly {
		return versions[:1]
	}

	return versions
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

// execGo runs a Go command. When cfg.version is set, it pins the toolchain
// with GOTOOLCHAIN=goX.Y.0 (the canonical first release, valid for Go >= 1.21).
// When empty, it uses the container's installed Go toolchain.
func execGo(t *testing.T, c testcontainers.Container, cfg *goExecConfig, args ...string) string {
	t.Helper()

	if cfg == nil {
		cfg = &goExecConfig{}
	}

	envParts := []string{"PATH=/usr/local/go/bin:$PATH"}
	for k, v := range cfg.env {
		envParts = append(envParts, fmt.Sprintf("%s=%s", k, v))
	}
	if cfg.version != "" {
		envParts = append(envParts, fmt.Sprintf("GOTOOLCHAIN=go%s.0", cfg.version))
	}
	envStr := strings.Join(envParts, " ")
	goArgs := strings.Join(args, " ")

	cmd := fmt.Sprintf("%s go %s 2>&1", envStr, goArgs)

	return execContainer(t, c, cmd)
}

func TestCompileCheck(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	rootDir := filepath.Dir(filepath.Dir(filepath.Dir(cwd)))

	// Resolve the versions under test before any container work, so invalid input
	// fails immediately rather than after the image build.
	testVersions := testGoVersions(t)

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

	// Initialize the Go module and resolve dependencies using the container's
	// installed Go toolchain, then pin the go directive to the driver's minimum
	// supported version (the lowest entry in goVersions). The directive is set to a
	// full X.Y.0 version to match what "go mod tidy" writes: a bare "X.Y" leaves
	// the module inconsistent.
	_ = execGo(t, container, nil, "mod", "init", "compilecheck")
	_ = execGo(t, container, nil, "mod", "edit", "-replace=go.mongodb.org/mongo-driver/v2=/mongo-go-driver")
	_ = execGo(t, container, nil, "mod", "tidy")
	_ = execGo(t, container, nil, "mod", "edit", "-go="+testVersions[0]+".0")

	for _, ver := range testVersions {
		ver := ver // capture
		t.Run("go:"+ver, func(t *testing.T) {
			t.Parallel()

			versionCfg := &goExecConfig{version: ver}

			// Verify the Go version is available.
			versionOutput := execGo(t, container, versionCfg, "version")
			require.Contains(t, versionOutput, "go"+ver, "unexpected go version: %s", versionOutput)

			_ = execGo(t, container, versionCfg, "build", "-buildvcs=false", "-o", "/dev/null", "main.go")

			// Dynamic linking build.
			_ = execGo(t, container, versionCfg, "build", "-buildvcs=false", "-buildmode=plugin", "-o", "/dev/null", "main.go")

			// Build with build tags.
			_ = execGo(t, container, &goExecConfig{
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
					_ = execGo(t, container, &goExecConfig{
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
