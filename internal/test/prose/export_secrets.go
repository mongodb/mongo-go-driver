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

// defaultProfile is the AWS profile the container logs in with. It, and the
// SSO settings baked into docker/entrypoint.sh that go with it, are what make
// a login zero-setup: the profile is written inside the container, so nothing
// has to be configured on the host first.
const defaultProfile = "drivers-test-secrets-role-857654397073"

// defaultSSOCacheDir returns the host directory the AWS CLI caches SSO tokens
// in, or "" if the user's home directory cannot be determined.
func defaultSSOCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".aws", "sso", "cache")
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

// containerSSOCacheDir is where the AWS CLI caches SSO tokens. The aws-cli
// image runs as root, so HOME is /root. The CLI always reads the cache from
// there, even though entrypoint.sh points AWS_CONFIG_FILE elsewhere.
const containerSSOCacheDir = "/root/.aws/sso/cache"

// exportConfig is the resolved configuration for ExportSecrets.
type exportConfig struct {
	profile     string
	dockerfile  string
	ssoCacheDir string
	vaults      []string
	timeout     time.Duration
}

// Option configures ExportSecrets.
type Option func(*exportConfig)

// WithProfile sets the AWS profile to log in with. It defaults to the
// AWS_PROFILE environment variable, or to defaultProfile when that is unset.
func WithProfile(profile string) Option {
	return func(cfg *exportConfig) { cfg.profile = profile }
}

// WithDockerfile sets the Dockerfile to build, relative to the docker
// directory. It defaults to "aws-sso-login.Dockerfile" and exists so tests can
// substitute a fixture image.
func WithDockerfile(name string) Option {
	return func(cfg *exportConfig) { cfg.dockerfile = name }
}

// WithSSOCacheDir sets the host directory mounted at the container's SSO token
// cache. It defaults to $HOME/.aws/sso/cache, the same cache the host's AWS
// CLI uses, so a login done either place is reused by the other until the
// token expires. The container never reads or writes the host's AWS config.
func WithSSOCacheDir(dir string) Option {
	return func(cfg *exportConfig) { cfg.ssoCacheDir = dir }
}

// WithVaults names AWS Secrets Manager vaults to fetch once the login
// succeeds, e.g. "drivers/csfle". Each vault is a flat JSON object; its keys
// are upper-cased and merged into the returned Secrets, matching what
// drivers-evergreen-tools' setup_secrets.py produces. Fetching them here means
// the caller needs neither a Python environment nor a host AWS profile.
func WithVaults(vaults ...string) Option {
	return func(cfg *exportConfig) { cfg.vaults = vaults }
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
// No host setup is required. The profile and its SSO settings are written
// inside the container, so a machine that has never run "aws configure sso"
// only has to approve the login. Set AWS_PROFILE, or pass WithProfile, to log
// in as something other than defaultProfile.
func ExportSecrets(t *testing.T, opts ...Option) Secrets {
	t.Helper()

	cfg := &exportConfig{
		profile:     os.Getenv("AWS_PROFILE"),
		dockerfile:  "aws-sso-login.Dockerfile",
		ssoCacheDir: defaultSSOCacheDir(),
		timeout:     defaultLoginTimeout,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.profile == "" {
		cfg.profile = defaultProfile
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

	// Create the cache directory if it is missing: on a machine that has never
	// used the AWS CLI there is nothing to mount yet, and Docker would
	// otherwise create it root-owned.
	var absCacheDir string
	if cfg.ssoCacheDir != "" {
		absCacheDir, err = filepath.Abs(cfg.ssoCacheDir)
		if err != nil {
			t.Fatalf("failed to resolve SSO cache directory: %v", err)
		}

		if err := os.MkdirAll(absCacheDir, 0o700); err != nil {
			t.Fatalf("failed to create SSO cache directory %s: %v", absCacheDir, err)
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
			"AWS_PROFILE":   cfg.profile,
			"SECRETS_DIR":   "/secrets",
			"SECRET_VAULTS": strings.Join(cfg.vaults, " "),
		},
		HostConfigModifier: func(hc *container.HostConfig) {
			hc.Binds = append(hc.Binds, absSecretsDir+":/secrets")

			// Share the host's SSO token cache so a live session is reused
			// instead of prompting again. The host's AWS config is not
			// mounted: entrypoint.sh writes the profile itself.
			if absCacheDir != "" {
				hc.Binds = append(hc.Binds, absCacheDir+":"+containerSSOCacheDir)
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

// parseSecretsFile reads a drivers-evergreen-tools style secrets-export.sh, such as
// export AWS_ACCESS_KEY_ID=value
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
