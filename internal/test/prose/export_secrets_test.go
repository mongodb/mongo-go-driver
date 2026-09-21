// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package prose

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixtureDockerfile builds the real entrypoint against a stub AWS CLI, so the
// test needs neither real credentials nor an interactive SSO prompt.
const fixtureDockerfile = "aws-sso-login-fixture.Dockerfile"

func TestExportSecrets(t *testing.T) {
	// Point the AWS directory at an empty temp dir: the stub CLI never reads
	// it, and this keeps the test from mounting the developer's real
	// credentials into a container.
	secrets := ExportSecrets(t,
		WithProfile("fixture-profile"),
		WithDockerfile(fixtureDockerfile),
		WithAWSDir(t.TempDir()),
		WithTimeout(2*time.Minute))

	assert.Equal(t, Secrets{
		"AWS_ACCESS_KEY_ID":     "fixture-access-key-id",
		"AWS_SECRET_ACCESS_KEY": "fixture-secret-access-key",
		"AWS_SESSION_TOKEN":     "fixture-session-token",
	}, secrets)
}

func TestExportSecrets_skipsWithoutProfile(t *testing.T) {
	t.Setenv("AWS_PROFILE", "")

	// ExportSecrets calls t.Skip, which only takes effect on the goroutine's
	// own *testing.T, so run it in a subtest and assert that it skipped.
	res := t.Run("no profile", func(t *testing.T) {
		// Pin the AWS directory to an empty one: with no profile configured
		// there is nothing to fall back to, and the developer's real profile
		// is never consulted.
		ExportSecrets(t,
			WithDockerfile(fixtureDockerfile),
			WithAWSDir(t.TempDir()))

		t.Error("expected ExportSecrets to skip when AWS_PROFILE is unset")
	})

	assert.True(t, res, "expected the subtest to skip rather than fail")
}

func TestConfiguredProfiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "named profiles",
			content: "[profile one]\nregion = us-east-1\n\n[profile two]\nregion = us-west-2\n",
			want:    []string{"one", "two"},
		},
		{
			name:    "default profile",
			content: "[default]\nregion = us-east-1\n",
			want:    []string{"default"},
		},
		{
			name:    "sso-session is not a profile",
			content: "[sso-session corp]\nsso_region = us-east-1\n\n[profile one]\n",
			want:    []string{"one"},
		},
		{
			name:    "no profiles",
			content: "# nothing here\n",
			want:    nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "config")
			require.NoError(t, os.WriteFile(path, []byte(test.content), 0o600))

			assert.Equal(t, test.want, configuredProfiles(path))
		})
	}
}

func TestConfiguredProfiles_missing(t *testing.T) {
	t.Parallel()

	assert.Nil(t, configuredProfiles(filepath.Join(t.TempDir(), "config")))
}

func TestSecrets_Env(t *testing.T) {
	t.Parallel()

	env := Secrets{"AWS_ACCESS_KEY_ID": "id", "AWS_SESSION_TOKEN": "token"}.Env()

	assert.ElementsMatch(t, []string{"AWS_ACCESS_KEY_ID=id", "AWS_SESSION_TOKEN=token"}, env)
}

func TestParseSecretsFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    Secrets
		wantErr bool
	}{
		{
			name:    "export prefix",
			content: "export AWS_ACCESS_KEY_ID=id\nexport AWS_SESSION_TOKEN=token\n",
			want:    Secrets{"AWS_ACCESS_KEY_ID": "id", "AWS_SESSION_TOKEN": "token"},
		},
		{
			name:    "no export prefix",
			content: "AWS_ACCESS_KEY_ID=id\n",
			want:    Secrets{"AWS_ACCESS_KEY_ID": "id"},
		},
		{
			name:    "quoted values",
			content: "export AWS_ACCESS_KEY_ID=\"id\"\nexport AWS_SESSION_TOKEN='token'\n",
			want:    Secrets{"AWS_ACCESS_KEY_ID": "id", "AWS_SESSION_TOKEN": "token"},
		},
		{
			name:    "comments and blank lines",
			content: "# a comment\n\nexport AWS_ACCESS_KEY_ID=id\n\n",
			want:    Secrets{"AWS_ACCESS_KEY_ID": "id"},
		},
		{
			name:    "value containing equals",
			content: "export AWS_SESSION_TOKEN=abc==\n",
			want:    Secrets{"AWS_SESSION_TOKEN": "abc=="},
		},
		{
			name:    "empty file",
			content: "",
			want:    Secrets{},
		},
		{
			name:    "malformed line",
			content: "export AWS_ACCESS_KEY_ID\n",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), secretsFileName)
			require.NoError(t, os.WriteFile(path, []byte(test.content), 0o600))

			got, err := parseSecretsFile(path)
			if test.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestParseSecretsFile_missing(t *testing.T) {
	t.Parallel()

	_, err := parseSecretsFile(filepath.Join(t.TempDir(), secretsFileName))
	assert.ErrorIs(t, err, os.ErrNotExist)
}
