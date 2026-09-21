// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package prose provides helpers for prose tests that need credentials from an
// AWS SSO login. The login runs inside a container so it does not depend on a
// particular AWS CLI version being installed on the host.
package prose

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// secretsFileName is the name entrypoint.sh writes inside the bind-mounted
// directory. It matches the file name used by drivers-evergreen-tools.
const secretsFileName = "secrets-export.sh"

// defaultLoginTimeout bounds the whole login. An SSO login is interactive, so
// the timeout has to accommodate a human completing the device flow.
const defaultLoginTimeout = 5 * time.Minute

// defaultAWSDir returns the host AWS config directory, or "" if the user's
// home directory cannot be determined.
func defaultAWSDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".aws")
}

// configuredProfiles returns the profile names defined in an AWS config file,
// which are section headers of the form "[profile name]" or "[default]". Only
// section names are read; the file's contents are otherwise ignored. An
// unreadable file yields no profiles, leaving the caller to skip.
func configuredProfiles(configPath string) []string {
	f, err := os.Open(configPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	var profiles []string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "[") || !strings.HasSuffix(line, "]") {
			continue
		}

		name := strings.TrimSpace(line[1 : len(line)-1])
		switch {
		case name == "default":
			profiles = append(profiles, name)
		case strings.HasPrefix(name, "profile "):
			profiles = append(profiles, strings.TrimSpace(strings.TrimPrefix(name, "profile ")))
		}
		// "[sso-session name]" sections are not profiles and are skipped.
	}

	return profiles
}

// Secrets holds the environment variables exported by an AWS SSO login, keyed
// by variable name (e.g. "AWS_ACCESS_KEY_ID").
type Secrets map[string]string

// Env returns the secrets as "KEY=VALUE" strings, suitable for exec.Cmd.Env.
func (s Secrets) Env() []string {
	env := make([]string, 0, len(s))
	for k, v := range s {
		env = append(env, k+"="+v)
	}
	return env
}

// containerAWSDir is where the AWS CLI looks for config and its SSO token
// cache. The aws-cli image runs as root, so HOME is /root.
const containerAWSDir = "/root/.aws"

// exportConfig is the resolved configuration for ExportSecrets.
type exportConfig struct {
	profile    string
	dockerfile string
	awsDir     string
	timeout    time.Duration
}

// Option configures ExportSecrets.
type Option func(*exportConfig)

// WithProfile sets the AWS profile to log in with. It defaults to the
// AWS_PROFILE environment variable.
func WithProfile(profile string) Option {
	return func(cfg *exportConfig) { cfg.profile = profile }
}

// WithDockerfile sets the Dockerfile to build, relative to the docker
// directory. It defaults to "aws-sso-login.Dockerfile" and exists so tests can
// substitute a fixture image.
func WithDockerfile(name string) Option {
	return func(cfg *exportConfig) { cfg.dockerfile = name }
}

// WithAWSDir sets the host directory mounted at /root/.aws in the container.
// It defaults to $HOME/.aws, which is where the profile named by AWS_PROFILE
// is defined. The mount is read-write so the SSO token cache the CLI writes
// persists on the host, letting later runs reuse a live session instead of
// prompting again.
func WithAWSDir(dir string) Option {
	return func(cfg *exportConfig) { cfg.awsDir = dir }
}

// WithTimeout bounds how long the login may take.
func WithTimeout(d time.Duration) Option {
	return func(cfg *exportConfig) { cfg.timeout = d }
}

// ExportSecrets builds and runs the AWS SSO login image and returns the
// credentials it exports. The container writes them to a directory
// bind-mounted from the host, which ExportSecrets then parses.
//
// The login is interactive: container output is streamed to the test log so
// the verification URL and code are visible to whoever is running the test.
//
// The profile comes from AWS_PROFILE, or from WithProfile. If neither is set
// and the host config defines exactly one profile, that one is used. Otherwise
// the test is skipped, since there is nothing to log in as.
func ExportSecrets(t *testing.T, opts ...Option) Secrets {
	t.Helper()

	cfg := &exportConfig{
		profile:    os.Getenv("AWS_PROFILE"),
		dockerfile: "aws-sso-login.Dockerfile",
		awsDir:     defaultAWSDir(),
		timeout:    defaultLoginTimeout,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Engineers commonly have exactly one profile configured, so fall back to
	// it rather than making AWS_PROFILE mandatory. Anything ambiguous is left
	// to the caller: guessing between profiles would log in as the wrong
	// account.
	if cfg.profile == "" {
		profiles := configuredProfiles(filepath.Join(cfg.awsDir, "config"))
		switch len(profiles) {
		case 1:
			cfg.profile = profiles[0]

			t.Logf("AWS_PROFILE is not set; using the only configured profile %q", cfg.profile)
		case 0:
			t.Skip("skipping because AWS_PROFILE is not set and no profile is configured")
		default:
			t.Skipf("skipping because AWS_PROFILE is not set and several profiles are configured: %s",
				strings.Join(profiles, ", "))
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()

	// The container writes the secrets file here, and the host reads it back
	// out. t.TempDir is removed when the test finishes, so the credentials do
	// not outlive the test.
	secretsDir := t.TempDir()

	// Docker Desktop only bind-mounts paths it has been granted access to, and
	// a relative path would be resolved inside the daemon rather than here.
	absSecretsDir, err := filepath.Abs(secretsDir)
	if err != nil {
		t.Fatalf("failed to resolve secrets directory: %v", err)
	}

	// A missing AWS directory is only fatal for a real login: the fixture
	// image stubs out the CLI and never reads config, so leave the mount off
	// rather than requiring tests to fabricate one.
	var absAWSDir string
	if cfg.awsDir != "" {
		absAWSDir, err = filepath.Abs(cfg.awsDir)
		if err != nil {
			t.Fatalf("failed to resolve AWS directory: %v", err)
		}

		if _, err := os.Stat(absAWSDir); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("failed to stat AWS directory %s: %v", absAWSDir, err)
			}

			t.Logf("AWS directory %s does not exist; not mounting it", absAWSDir)
			absAWSDir = ""
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:       filepath.Join(cwd, "docker"),
			Dockerfile:    cfg.dockerfile,
			PrintBuildLog: true,
		},
		Env: map[string]string{
			"AWS_PROFILE": cfg.profile,
			"SECRETS_DIR": "/secrets",
			// "aws configure sso" is prompt-driven and this container has no
			// stdin attached, so bootstrapping a profile from a test would
			// hang. The entrypoint exits with a pointer to the interactive
			// "docker run -it" command instead.
			"ALLOW_CONFIGURE_SSO": "0",
		},
		HostConfigModifier: func(hc *container.HostConfig) {
			hc.Binds = append(hc.Binds, absSecretsDir+":/secrets")

			// The container needs the host's AWS config to resolve the
			// profile, and writes its SSO token cache back so a live session
			// is reused on the next run.
			if absAWSDir != "" {
				hc.Binds = append(hc.Binds, absAWSDir+":"+containerAWSDir)
			}
		},
		LogConsumerCfg: &testcontainers.LogConsumerConfig{
			Consumers: []testcontainers.LogConsumer{&testLogConsumer{t: t}},
		},
		// The entrypoint exits once the secrets are written, so a successful
		// run is a clean exit rather than a listening port.
		WaitingFor: wait.ForExit().WithExitTimeout(cfg.timeout),
	}

	genReq := testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true}

	ctr, err := testcontainers.GenericContainer(ctx, genReq)
	if err != nil {
		t.Fatalf("failed to run AWS SSO login container: %v", err)
	}

	t.Cleanup(func() {
		if err := ctr.Terminate(context.Background()); err != nil {
			t.Logf("failed to terminate AWS SSO login container: %v", err)
		}
	})

	// The entrypoint runs under "set -e", so a failed login or a failed
	// credential verification shows up as a non-zero exit. Check it before
	// reading the file: a stale or partial secrets-export.sh would otherwise
	// parse cleanly and hide the failure.
	state, err := ctr.State(ctx)
	if err != nil {
		t.Fatalf("failed to get AWS SSO login container state: %v", err)
	}
	if state.ExitCode != 0 {
		t.Fatalf("AWS SSO login container exited with code %d; see the streamed logs above",
			state.ExitCode)
	}

	secrets, err := parseSecretsFile(filepath.Join(secretsDir, secretsFileName))
	if err != nil {
		t.Fatalf("failed to read exported secrets: %v", err)
	}
	if len(secrets) == 0 {
		t.Fatal("AWS SSO login exported no secrets")
	}

	return secrets
}

// testLogConsumer forwards container output to the test log so the
// interactive SSO prompt is visible while the test is running.
type testLogConsumer struct {
	t *testing.T
}

func (c *testLogConsumer) Accept(log testcontainers.Log) {
	c.t.Logf("aws-sso-login: %s", strings.TrimRight(string(log.Content), "\n"))
}

// parseSecretsFile reads a drivers-evergreen-tools style secrets-export.sh,
// whose lines look like:
//
//	export AWS_ACCESS_KEY_ID=value
//
// Blank lines and comments are ignored. Values may be single- or
// double-quoted.
func parseSecretsFile(path string) (Secrets, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", filepath.Base(path), err)
	}
	defer f.Close()

	secrets := Secrets{}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		line = strings.TrimPrefix(line, "export ")

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("malformed line in %s: %q", filepath.Base(path), line)
		}

		secrets[strings.TrimSpace(key)] = unquote(strings.TrimSpace(value))
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", filepath.Base(path), err)
	}

	return secrets, nil
}

// unquote strips one layer of matching single or double quotes.
func unquote(s string) string {
	if len(s) < 2 {
		return s
	}
	if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
		return s[1 : len(s)-1]
	}
	return s
}
