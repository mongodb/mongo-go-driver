package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/mod/semver"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const minSupportedVersion = "v1.18"

func TestCompileCheck(t *testing.T) {
	ctx := context.Background()

	versions, err := getAllGoVersions()
	require.NoError(t, err)

	fmt.Println(versions)

	req := testcontainers.ContainerRequest{
		Image:      "alpine",
		Cmd:        []string{"echo", "hello world"},
		WaitingFor: wait.ForLog("hello world"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	require.NoError(t, err)

	defer func() {
		err := container.Terminate(ctx)
		require.NoError(t, err)
	}()
}

func getAllGoVersions() ([]string, error) {
	resp, err := http.Get("https://golang.org/dl/?mode=json&include=all")
	if err != nil {
		return nil, fmt.Errorf("failed to get response from golang.org: %v", err)
	}

	defer resp.Body.Close()

	var releases []struct {
		Version string `json:"version"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %v", err)
	}

	versions := []string{}
	for _, r := range releases {
		if len(r.Version) < 3 || r.Version[:2] != "go" {
			continue
		}

		v := "v" + r.Version[2:]
		if semver.Compare(v, minSupportedVersion) >= 0 {
			versions = append(versions, "go"+v[1:])
		}
	}

	return versions, nil
}
