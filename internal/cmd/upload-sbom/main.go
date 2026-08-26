// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Command upload-sbom runs Silkbomb, via the Docker Engine API, in two
// steps: "update --select-licenses" categorizes sbom.json's license evidence
// per MongoDB's Inbound Open Source Policy, then "augment" enriches the
// result with Kondukto scan results, producing augmented.sbom.json.new. The
// augmented output is diffed against the previously committed
// augmented.sbom.json; if the two differ, a non-fatal failed status is
// reported to the running Evergreen task so the change is surfaced for
// review without blocking the build.
//
// Required environment: branch_name, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY,
// AWS_SESSION_TOKEN (credentials for the Silkbomb IAM role, passed through to
// the container). ECR pull authentication is read from the Docker CLI config
// (DOCKER_CONFIG, or ~/.docker) that an earlier CI step populated using a
// separate ECR-readonly role — the Silkbomb role's credentials do not
// necessarily have ECR pull permission, so the two must not be conflated.
package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
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

	inputSBOM    = "sbom.json"
	selectedSBOM = "sbom.selected.json" // sbom.json with license categorization applied

	augmentedOld = "augmented.sbom.json"
	augmentedNew = "augmented.sbom.json.new"
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

	if err := runSilkbombUpdate(ctx, cli); err != nil {
		return err
	}
	if _, err := diffAndPrint(inputSBOM, selectedSBOM, "license-old.json", "license-new.json", "license-diff.txt"); err != nil {
		return fmt.Errorf("diffing license selection: %w", err)
	}

	if err := runSilkbombAugment(ctx, cli, branch); err != nil {
		return err
	}

	return diffAndReport()
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

// runSilkbombUpdate runs Silkbomb's "update" subcommand with -select-licenses,
// which categorizes each component's license per MongoDB's Inbound Open
// Source Policy (go/inbound-oss) without merging in a purl list. Unlike
// augment/upload, update doesn't publish anywhere, so it needs no AWS
// credentials.
func runSilkbombUpdate(ctx context.Context, cli *client.Client) error {
	if err := runSilkbombContainer(ctx, cli, []string{
		"update",
		"--sbom-in", "/pwd/" + inputSBOM,
		"--sbom-out", "/pwd/" + selectedSBOM,
		"--select-licenses",
	}, nil); err != nil {
		return err
	}
	if _, err := os.Stat(selectedSBOM); err != nil {
		return fmt.Errorf("failed to produce license-selected SBOM: %w", err)
	}
	return nil
}

// runSilkbombAugment runs Silkbomb's "augment" subcommand against the
// license-selected SBOM, enriching it with Kondukto scan results.
func runSilkbombAugment(ctx context.Context, cli *client.Client, branch string) error {
	if err := runSilkbombContainer(ctx, cli, []string{
		"augment",
		"--repo", repo,
		"--branch", branch,
		"--sbom-in", "/pwd/" + selectedSBOM,
		"--sbom-out", "/pwd/" + augmentedNew,
		// Any notable updates to the Augmented SBOM version should be
		// done manually after careful inspection. Otherwise, it
		// should be equal to the existing SBOM version.
		"--no-update-sbom-version",
	}, []string{
		"AWS_ACCESS_KEY_ID=" + os.Getenv("AWS_ACCESS_KEY_ID"),
		"AWS_SECRET_ACCESS_KEY=" + os.Getenv("AWS_SECRET_ACCESS_KEY"),
		"AWS_SESSION_TOKEN=" + os.Getenv("AWS_SESSION_TOKEN"),
	}); err != nil {
		return err
	}
	if _, err := os.Stat(augmentedNew); err != nil {
		return fmt.Errorf("failed to produce augmented SBOM: %w", err)
	}
	return nil
}

// runSilkbombContainer runs the silkbomb image with the given command and
// environment, mounting the working directory at /pwd, and streams its logs
// to stdout/stderr. It returns an error if the container exits non-zero.
func runSilkbombContainer(ctx context.Context, cli *client.Client, cmd, env []string) error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	resp, err := cli.ContainerCreate(ctx,
		&container.Config{
			Image: silkbombImage,
			Cmd:   cmd,
			Env:   env,
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

// diffAndReport diffs the previous and newly generated Augmented SBOMs into
// old.json, new.json, and diff.txt — the exact names the calling Evergreen
// task uploads to S3 regardless of outcome. If the content differs, a
// non-fatal failed status is reported to the task so the change surfaces for
// review without blocking the build.
func diffAndReport() error {
	fmt.Println("Comparing Augmented SBOM...")

	changed, err := diffAndPrint(augmentedOld, augmentedNew, "old.json", "new.json", "diff.txt")
	if err != nil {
		return err
	}

	if changed {
		if err := postEvergreenStatus("detected significant changes in Augmented SBOM"); err != nil {
			fmt.Fprintln(os.Stderr, "upload-sbom: reporting task status:", err)
		}
	}

	fmt.Println("Comparing Augmented SBOM... done.")
	return nil
}

// diffAndPrint normalizes the SBOMs at oldPath and newPath (dropping the
// timestamp, which always changes), writes them to oldOut and newOut, diffs
// them side by side into diffOut, prints the diff, and reports whether the
// two differed.
func diffAndPrint(oldPath, newPath, oldOut, newOut, diffOut string) (changed bool, err error) {
	if err := normalizeToFile(oldPath, oldOut); err != nil {
		return false, fmt.Errorf("reading %s: %w", oldPath, err)
	}
	if err := normalizeToFile(newPath, newOut); err != nil {
		return false, fmt.Errorf("reading %s: %w", newPath, err)
	}

	changed, err = diffFiles(oldOut, newOut, diffOut)
	if err != nil {
		return false, err
	}

	diffContent, err := os.ReadFile(diffOut)
	if err != nil {
		return false, err
	}
	os.Stdout.Write(diffContent)

	return changed, nil
}

// normalizeToFile reads the SBOM at inPath, strips metadata.timestamp, and
// writes it pretty-printed (matching `jq -S`) to outPath for a stable,
// line-oriented diff.
func normalizeToFile(inPath, outPath string) error {
	data, err := os.ReadFile(inPath)
	if err != nil {
		return err
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("parsing %s: %w", inPath, err)
	}
	if md, ok := m["metadata"].(map[string]any); ok {
		delete(md, "timestamp")
	}
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, out, 0o644)
}

// diffFiles runs `diff -sty --left-column -W 200` between a and b, writing
// its output to outPath, and reports whether the files differ.
func diffFiles(a, b, outPath string) (changed bool, err error) {
	out, err := os.Create(outPath)
	if err != nil {
		return false, err
	}
	defer out.Close()

	cmd := exec.Command("diff", "-sty", "--left-column", "-W", "200", a, b)
	cmd.Stdout = out
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err == nil {
		return false, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return true, nil
	}
	return false, fmt.Errorf("diff: %w", err)
}

func postEvergreenStatus(desc string) error {
	body, err := json.Marshal(map[string]any{
		"status":          "failed",
		"type":            "test",
		"should_continue": true,
		"desc":            desc,
	})
	if err != nil {
		return err
	}

	resp, err := http.Post("http://localhost:2285/task_status", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil // non-fatal, mirror the shell script's `|| true`
	}
	defer resp.Body.Close()
	return nil
}
