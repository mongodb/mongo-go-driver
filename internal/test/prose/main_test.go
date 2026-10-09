// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package prose

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/joho/godotenv"
	"github.com/moby/moby/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	tcexec "github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	loadSecretsFlag = flag.Bool("load-secrets", false, "Use AWS SSO login to load secrets into the repository root")
	cseFlag         = flag.Bool("cse", false, "Setup required for running CSE tests")
	ciFlag          = flag.Bool("ci", false, "Run in CI: use the existing secrets-export.sh and the host network")
)

func secretsRequested() bool {
	return *loadSecretsFlag || *cseFlag
}

var (
	// cseContainer is the long-lived CSE container, set by TestMain when -cse
	// is passed.
	cseContainer testcontainers.Container

	// cseEnv is the mongodb environment passed to every command run in cseContainer,
	// set by TestMain when -cse is passed.
	cseEnv []string

	// loadedSecretsPath is the secrets-export.sh file loaded by TestMain.
	loadedSecretsPath string
)

func TestMain(m *testing.M) {
	flag.Parse()

	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

	if secretsRequested() {
		path, err := exportSecrets()
		if err != nil {
			log.Panicf("error loading secrets: %v", err)
		}
		if err := godotenv.Overload(path); err != nil {
			log.Panicf("error loading secrets: %v", err)
		}
		loadedSecretsPath = path
	}

	if *cseFlag {
		env, err := buildCSEEnv(loadedSecretsPath)
		if err != nil {
			log.Panicf("error building CSE environment: %v", err)
		}
		cseEnv = env

		ctr, err := startCSE()
		if err != nil {
			log.Panicf("error starting CSE container: %v", err)
		}
		cseContainer = ctr

		if err := checkHosts(context.Background()); err != nil {
			log.Panicf("error checking hosts: %v", err)
		}

		if kmsMocksRunning() && !*ciFlag {
			if err := forwardKMSMocks(context.Background()); err != nil {
				log.Panicf("error forwarding KMS mock servers: %v", err)
			}
		}
	}

	os.Exit(m.Run())
}

// =============================================================================
// Test Runner Helpers
// =============================================================================

const (
	dockerRepo       = "10gen/go-driver-tools"
	dockerfileName   = "aws-sso-login.Dockerfile"
	entrypointName   = "aws-sso-login.sh"
	secretsFileName  = "secrets-export.sh"
	containerOutDir  = "/out"
	loginTimeout     = 10 * time.Minute
	secretsMaxAge    = 50 * time.Minute
	secretsDirPrefix = "mongo-go-driver-prose"

	cseDockerfileName   = "cse.Dockerfile"
	cseInstallScript    = "install-libmongocrypt.sh"
	cseMongoDL          = "mongodl.py"
	cseContainerName    = "mongo-go-driver-cse"
	cseContainerRepoDir = "/mongo-go-driver"

	// cseHostGateway is the hostname the CSE container uses to reach the host.
	defaultMongoURI = "mongodb://localhost:27017"
)

// repoRoot returns the absolute path of the Go Driver repository root.
func repoRoot() (string, error) {
	root, err := filepath.Abs("../../../")
	if err != nil {
		return "", fmt.Errorf("failed to resolve relative path: %w", err)
	}
	return root, nil
}

func keepExports(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", secretsFileName, err)
	}

	var exports strings.Builder
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "export ") {
			exports.WriteString(line + "\n")
		}
	}

	return os.WriteFile(path, []byte(exports.String()), 0o600)
}

// exportSecrets returns the path to secrets-export.sh, running the AWS SSO
// login container to create it if it is missing or stale. The secrets-export.sh
// file lives in the Go Driver repository root.
func exportSecrets() (string, error) {
	root, err := repoRoot()
	if err != nil {
		return "", err
	}
	secretsPath := filepath.Join(root, secretsFileName)

	// In CI, etc/setup-encryption.sh writes secrets-export.sh using the
	// task's assumed AWS role, so there is no interactive login.
	if *ciFlag {
		if _, err := os.Stat(secretsPath); err != nil {
			return "", fmt.Errorf("%s not found; run etc/setup-encryption.sh first: %w", secretsFileName, err)
		}
		return secretsPath, keepExports(secretsPath)
	}

	// The file holds temporary AWS, Azure and GCP tokens that expire about an
	// hour after the login, but it does not record when. Reuse it only while
	// it is younger than secretsMaxAge; after that, log in again.
	if info, err := os.Stat(secretsPath); err == nil && time.Since(info.ModTime()) < secretsMaxAge {
		return secretsPath, nil
	}

	buildDir, err := os.MkdirTemp("", secretsDirPrefix+"-build")
	if err != nil {
		return "", fmt.Errorf("failed to create build context: %w", err)
	}
	defer os.RemoveAll(buildDir)

	if err := downloadDockerfile(buildDir); err != nil {
		return "", err
	}

	if err := runLogin(buildDir, root); err != nil {
		return "", err
	}

	if _, err := os.Stat(secretsPath); err != nil {
		return "", fmt.Errorf("AWS SSO login container did not write %s: %w", secretsFileName, err)
	}

	if err := keepExports(secretsPath); err != nil {
		return "", err
	}

	return secretsPath, nil
}

// downloads the Dockerfile and entrypoint from the private repo.
func downloadDockerfile(dir string) error {
	client, err := api.DefaultRESTClient()
	if err != nil {
		return fmt.Errorf("failed to create GitHub client (is `gh auth login` set up?): %w", err)
	}

	for _, name := range []string{dockerfileName, entrypointName} {
		var response struct {
			Content string `json:"content"`
		}

		if err := client.Get("repos/"+dockerRepo+"/contents/docker/"+name, &response); err != nil {
			return fmt.Errorf("failed to fetch %s from %s: %w", name, dockerRepo, err)
		}

		content, err := base64.StdEncoding.DecodeString(response.Content)
		if err != nil {
			return fmt.Errorf("failed to decode %s: %w", name, err)
		}

		if err := os.WriteFile(filepath.Join(dir, name), content, 0o755); err != nil {
			return fmt.Errorf("failed to write %s: %w", name, err)
		}
	}

	return nil
}

// runLogin builds the image in buildDir and runs it with secretsDir mounted
// at the container's output directory. The login is interactive: container
// output is streamed to stderr so the verification URL and code are visible.
func runLogin(buildDir, secretsDir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), loginTimeout)
	defer cancel()

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    buildDir,
			Dockerfile: dockerfileName,
		},
		HostConfigModifier: func(hc *container.HostConfig) {
			hc.Binds = append(hc.Binds, secretsDir+":"+containerOutDir)
		},
		LogConsumerCfg: &testcontainers.LogConsumerConfig{
			Consumers: []testcontainers.LogConsumer{stderrLogConsumer{}},
		},
		// The entrypoint exits once the secrets are written.
		WaitingFor: wait.ForExit().WithExitTimeout(loginTimeout),
	}

	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	defer func() {
		if ctr != nil {
			_ = ctr.Terminate(context.Background())
		}
	}()
	if err != nil {
		return fmt.Errorf("failed to run AWS SSO login container: %w", err)
	}

	state, err := ctr.State(ctx)
	if err != nil {
		return fmt.Errorf("failed to get AWS SSO login container state: %w", err)
	}
	if state.ExitCode != 0 {
		return fmt.Errorf("AWS SSO login container exited with code %d", state.ExitCode)
	}

	return nil
}

type stderrLogConsumer struct{}

func (stderrLogConsumer) Accept(log testcontainers.Log) {
	fmt.Fprintln(os.Stderr, "aws-sso-login:", strings.TrimRight(string(log.Content), "\n"))
}

// startCSE builds the CSE image and starts a container named cseContainerName,
// or reuses it if it already exists. The container is not terminated when the
// tests finish so later runs skip the libmongocrypt build. The repository root
// is bind-mounted at cseContainerRepoDir so driver changes are picked up
// without a rebuild. To force a rebuild, remove the container:
//
//	docker rm -f mongo-go-driver-cse
func startCSE() (testcontainers.Container, error) {
	root, err := repoRoot()
	if err != nil {
		return nil, err
	}

	buildDir, err := os.MkdirTemp("", secretsDirPrefix+"-cse-build")
	if err != nil {
		return nil, fmt.Errorf("failed to create build context: %w", err)
	}
	defer os.RemoveAll(buildDir)

	for src, dst := range map[string]string{
		cseDockerfileName: cseDockerfileName,
		filepath.Join(root, "etc", cseInstallScript):                                           cseInstallScript,
		filepath.Join(root, ".evergreen", "drivers-evergreen-tools", ".evergreen", cseMongoDL): cseMongoDL,
	} {
		if err := copyFile(src, filepath.Join(buildDir, dst)); err != nil {
			return nil, err
		}
	}

	req := testcontainers.ContainerRequest{
		Name: cseContainerName,
		FromDockerfile: testcontainers.FromDockerfile{
			Context:       buildDir,
			Dockerfile:    cseDockerfileName,
			PrintBuildLog: true,
		},
		HostConfigModifier: func(hc *container.HostConfig) {
			hc.Binds = append(hc.Binds, root+":"+cseContainerRepoDir)

			// On Linux CI hosts, mongod and the KMS mock servers listen on the
			// host's 127.0.0.1, which the bridge network cannot reach.
			if *ciFlag {
				hc.NetworkMode = "host"
				return
			}
			hc.ExtraHosts = append(hc.ExtraHosts, "localhost:host-gateway")
		},
		// Block on "tail -f /dev/null" so the container stays alive and ready
		// for exec calls, rather than immediately exiting.
		Entrypoint: []string{"tail", "-f", "/dev/null"},
		WorkingDir: cseContainerRepoDir,
	}

	ctr, err := testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
		Reuse:            true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start CSE container: %w", err)
	}

	return ctr, nil
}

// The environment is passed on each exec rather than when the container is
// created, because the container is reused across runs and the secrets expire.
func buildCSEEnv(secretsPath string) ([]string, error) {
	secrets, err := godotenv.Read(secretsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", secretsFileName, err)
	}

	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = defaultMongoURI
	}
	delete(secrets, "MONGODB_URI")

	// Match etc/setup-encryption.sh: the KMS mock servers use the EC certs in
	// testdata/kmip-certs.
	certDir := cseContainerRepoDir + "/testdata/kmip-certs/"
	secrets["CSFLE_TLS_CA_FILE"] = certDir + "ca-ec.pem"
	secrets["CSFLE_TLS_CLIENT_CERT_FILE"] = certDir + "client-ec.pem"

	// The failpoint server uses drivers-evergreen-tools' x509gen certs, which
	// the bind-mounted submodule provides.
	secrets["KMS_FAILPOINT_CA_FILE"] = cseContainerRepoDir + "/.evergreen/drivers-evergreen-tools/.evergreen/x509gen/ca.pem"

	env := make([]string, 0, len(secrets)+1)
	for key, val := range secrets {
		env = append(env, key+"="+val)
	}
	env = append(env, "MONGODB_URI="+uri)
	if kmsMocksRunning() {
		env = append(env, "KMS_MOCK_SERVERS_RUNNING=true", "KMS_FAILPOINT_SERVER_RUNNING=true")
	}
	return env, nil
}

// kmsMockPorts are the KMS mock servers the CSE tests require.
var kmsMockPorts = []string{"9000", "9001", "9002", "9003", "5698", "8080"}

// optionalKMSMockPorts are forwarded when they are running, but are not
// required: drivers-evergreen-tools' HTTP proxies.
var optionalKMSMockPorts = []string{"9004", "9005"}

func kmsMocksRunning() bool {
	for _, port := range kmsMockPorts {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort("localhost", port), time.Second)
		if err != nil {
			return false
		}
		_ = conn.Close()
	}
	return true
}

func forwardKMSMocks(ctx context.Context) error {
	for _, port := range append(kmsMockPorts, optionalKMSMockPorts...) {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort("localhost", port), time.Second)
		if err != nil {
			continue
		}
		_ = conn.Close()

		cmd := fmt.Sprintf("(echo > /dev/tcp/127.0.0.1/%[1]s) 2>/dev/null || "+
			"(nohup socat TCP-LISTEN:%[1]s,bind=127.0.0.1,fork,reuseaddr TCP:host.docker.internal:%[1]s </dev/null >/dev/null 2>&1 &)", port)
		exit, out, err := execCSE(ctx, cmd)
		if err != nil {
			return err
		}
		if exit != 0 {
			return fmt.Errorf("failed to forward port %s (exit code %d): %s", port, exit, out)
		}
	}
	return nil
}

func checkHosts(ctx context.Context) error {
	// If the container is already good to go, do nothing
	if ping(ctx) == nil {
		return nil
	}

	// If there is an error reaching the container through localhost, then
	// terminate and try to restart.
	if err := cseContainer.Terminate(ctx); err != nil {
		return fmt.Errorf("failed to terminate CSE container: %w", err)
	}

	ctr, err := startCSE()
	if err != nil {
		return err
	}

	cseContainer = ctr

	// Try to ping again, if this one fails return error to user.
	if err := ping(ctx); err != nil {
		return fmt.Errorf("CSE container is not reachable after restart (is a local mongod or mongos running?): %w", err)
	}

	return nil
}

func ping(ctx context.Context) error {
	exit, _, err := execCSE(ctx, "go run ./internal/cmd/testping")
	if err != nil {
		return err
	}

	if exit != 0 {
		return fmt.Errorf("exit code: %v", exit)
	}

	return nil
}

// execCSE runs cmd with bash in the CSE container and returns its exit code
// and combined output.
func execCSE(ctx context.Context, cmd string) (int, string, error) {
	if cseContainer == nil {
		return 0, "", fmt.Errorf("CSE container is not running; pass -cse")
	}

	exit, out, err := cseContainer.Exec(ctx, []string{"bash", "-c", cmd + " 2>&1"}, tcexec.Multiplexed(), tcexec.WithEnv(cseEnv))
	if err != nil {
		return 0, "", fmt.Errorf("failed to exec %q: %w", cmd, err)
	}

	b, err := io.ReadAll(out)
	if err != nil {
		return 0, "", fmt.Errorf("failed to read output of %q: %w", cmd, err)
	}

	return exit, string(b), nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", src, err)
	}
	if err := os.WriteFile(dst, data, 0o755); err != nil {
		return fmt.Errorf("failed to write %s: %w", dst, err)
	}
	return nil
}
