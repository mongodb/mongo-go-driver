// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/stretchr/testify/require"
)

// cloneDriverRepo clones the mongo-go-driver repo this test is running from
// into a temp directory, so generation runs against the real dependency
// graph instead of a synthetic fixture.
func cloneDriverRepo(t *testing.T) string {
	t.Helper()

	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	require.NoError(t, err)
	repoRoot := strings.TrimSpace(string(out))

	dir := t.TempDir()
	cmd := exec.Command("git", "clone", "--local", "--depth", "1", "--no-tags", repoRoot, dir)
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "-C", dir, "-c", "user.email=test@example.com", "-c", "user.name=test", "config", "commit.gpgsign", "false")
	require.NoError(t, cmd.Run())

	return dir
}

func readBOM(t *testing.T, moduleDir string) *cdx.BOM {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(moduleDir, "sbom.json"))
	require.NoError(t, err)

	var bom cdx.BOM
	require.NoError(t, json.Unmarshal(data, &bom))
	return &bom
}

func findComponent(bom *cdx.BOM, name string) *cdx.Component {
	if bom.Components == nil {
		return nil
	}
	for i := range *bom.Components {
		if (*bom.Components)[i].Name == name {
			return &(*bom.Components)[i]
		}
	}
	return nil
}

func TestGenerate(t *testing.T) {
	dir := cloneDriverRepo(t)

	libmongocryptVersionWant, err := libmongocryptVersion(filepath.Join(dir, "etc", "install-libmongocrypt.sh"))
	require.NoError(t, err)
	goVersionWant, err := goDirectiveVersion(filepath.Join(dir, "go.mod"))
	require.NoError(t, err)

	os.Args = []string{"generate-sbom", dir}
	require.NoError(t, run())

	bom := readBOM(t, dir)

	t.Run("driver dependency lands", func(t *testing.T) {
		// A stable, direct dependency of the driver module.
		dep := findComponent(bom, "github.com/youmark/pkcs8")
		require.NotNil(t, dep, "expected github.com/youmark/pkcs8 in generated SBOM")
		require.NotEmpty(t, dep.Version)
	})

	t.Run("stdlib lands", func(t *testing.T) {
		std := findComponent(bom, "std")
		require.NotNil(t, std, "expected a std component in generated SBOM")
		require.Equal(t, "go"+goVersionWant, std.Version)
	})

	t.Run("libmongocrypt lands", func(t *testing.T) {
		lmc := findComponent(bom, "libmongocrypt")
		require.NotNil(t, lmc, "expected a libmongocrypt component in generated SBOM")
		require.Equal(t, libmongocryptVersionWant, lmc.Version)
	})

	t.Run("committing sbom.json does not immediately invalidate it", func(t *testing.T) {
		cmd := exec.Command("git", "-C", dir, "add", "sbom.json")
		cmd.Stderr = os.Stderr
		require.NoError(t, cmd.Run())

		cmd = exec.Command("git", "-C", dir, "-c", "user.email=test@example.com", "-c", "user.name=test",
			"commit", "--allow-empty", "-m", "commit generated sbom.json")
		cmd.Stderr = os.Stderr
		require.NoError(t, cmd.Run())

		t.Setenv("EXPECT_ERROR", "1")
		os.Args = []string{"generate-sbom", dir}
		require.NoError(t, run(), "regenerating right after committing sbom.json should not report it as stale")
	})
}
