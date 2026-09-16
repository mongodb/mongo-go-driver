// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Command upload-sbom runs Silkbomb's "upload" subcommand, via the Docker
// Engine API, to publish sbom.json to Dependency-Track and Kondukto.
//
// Required environment: branch_name, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY,
// AWS_SESSION_TOKEN (credentials for the Silkbomb IAM role, passed through to
// the container). ECR pull authentication is read from the Docker CLI config
// (DOCKER_CONFIG, or ~/.docker) that an earlier CI step populated using a
// separate ECR-readonly role — the Silkbomb role's credentials do not
// necessarily have ECR pull permission, so the two must not be conflated.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

const (
	silkbombImage = "901841024863.dkr.ecr.us-east-1.amazonaws.com/release-infrastructure/silkbomb:2.0"
	ecrRegistry   = "901841024863.dkr.ecr.us-east-1.amazonaws.com"
	repo          = "mongodb/mongo-go-driver"

	inputSBOM = "sbom.json"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "upload-sbom:", err)
		os.Exit(1)
	}
}

func run() error {
	branch, err := requireEnv("branch_name")
	if err != nil {
		return err
	}
	for _, k := range []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN"} {
		if _, err := requireEnv(k); err != nil {
			return err
		}
	}

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	defer cli.Close()

	authStr, err := ecrAuthFromDockerConfig()
	if err != nil {
		return err
	}

	if err := pullImage(ctx, cli, authStr); err != nil {
		return err
	}

	return runSilkbombUpload(ctx, cli, branch)
}

func requireEnv(k string) (string, error) {
	v := os.Getenv(k)
	if v == "" {
		return "", fmt.Errorf("%s must be set", k)
	}
	return v, nil
}

// ecrAuthFromDockerConfig reads the ECR credentials that an earlier CI step
// (using the separate ECR-readonly role) logged in with via `docker login`,
// and encodes them as a Docker registry auth string. We deliberately don't
// derive ECR credentials ourselves here — by the time this runs, the process
// has already assumed the Silkbomb IAM role, which is not guaranteed to have
// ECR pull permission.
func ecrAuthFromDockerConfig() (string, error) {
	dir := os.Getenv("DOCKER_CONFIG")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".docker")
	}

	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		return "", fmt.Errorf("reading docker config: %w", err)
	}

	var cfg struct {
		Auths map[string]struct {
			Auth string `json:"auth"`
		} `json:"auths"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("parsing docker config: %w", err)
	}

	entry, ok := cfg.Auths[ecrRegistry]
	if !ok {
		return "", fmt.Errorf("no docker login found for %s; run docker login first", ecrRegistry)
	}

	decoded, err := base64.StdEncoding.DecodeString(entry.Auth)
	if err != nil {
		return "", fmt.Errorf("decoding docker config auth: %w", err)
	}
	username, password, ok := strings.Cut(string(decoded), ":")
	if !ok {
		return "", fmt.Errorf("malformed docker config auth for %s", ecrRegistry)
	}

	authJSON, err := json.Marshal(registry.AuthConfig{
		Username:      username,
		Password:      password,
		ServerAddress: ecrRegistry,
	})
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(authJSON), nil
}

func pullImage(ctx context.Context, cli *client.Client, authStr string) error {
	rc, err := cli.ImagePull(ctx, silkbombImage, image.PullOptions{RegistryAuth: authStr})
	if err != nil {
		return fmt.Errorf("pull silkbomb: %w", err)
	}
	defer rc.Close()
	_, err = io.Copy(os.Stdout, rc)
	return err
}

// runSilkbombUpload runs Silkbomb's "upload" subcommand, which publishes
// sbom.json directly to Dependency-Track and Kondukto in one call.
func runSilkbombUpload(ctx context.Context, cli *client.Client, branch string) error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	resp, err := cli.ContainerCreate(ctx,
		&container.Config{
			Image: silkbombImage,
			Cmd: []string{
				"upload",
				"--repo", repo,
				"--branch", branch,
				"--sbom-in", "/pwd/" + inputSBOM,
			},
			Env: []string{
				"AWS_ACCESS_KEY_ID=" + os.Getenv("AWS_ACCESS_KEY_ID"),
				"AWS_SECRET_ACCESS_KEY=" + os.Getenv("AWS_SECRET_ACCESS_KEY"),
				"AWS_SESSION_TOKEN=" + os.Getenv("AWS_SESSION_TOKEN"),
			},
		},
		&container.HostConfig{Binds: []string{pwd + ":/pwd"}},
		nil, nil, "")
	if err != nil {
		return fmt.Errorf("create silkbomb container: %w", err)
	}
	defer cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("start silkbomb container: %w", err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	var exitCode int64
	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("waiting for silkbomb: %w", err)
		}
	case status := <-statusCh:
		exitCode = status.StatusCode
	}

	logs, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err == nil {
		defer logs.Close()
		_, _ = stdcopy.StdCopy(os.Stdout, os.Stderr, logs)
	}

	if exitCode != 0 {
		return fmt.Errorf("silkbomb exited with code %d", exitCode)
	}
	return nil
}
