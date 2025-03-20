package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

const minSupportedVersion = "1.18"

func TestCompileCheck(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	internalDir := filepath.Dir(filepath.Dir(filepath.Dir(cwd)))

	versions, err := getDockerGolangImages()
	require.NoError(t, err)

	for _, version := range versions {
		version := version // Capture range variable.

		image := fmt.Sprintf("golang:%s", version)
		t.Run(image, func(t *testing.T) {
			t.Parallel()

			req := testcontainers.ContainerRequest{
				Image: image,
				Cmd:   []string{"tail", "-f", "/dev/null"},
				Mounts: []testcontainers.ContainerMount{
					testcontainers.BindMount(internalDir, "/workspace"),
				},
				WorkingDir: "/workspace",
				Env: map[string]string{
					"GO_VERSION": version,
				},
			}

			genReq := testcontainers.GenericContainerRequest{
				ContainerRequest: req,
				Started:          true,
			}

			container, err := testcontainers.GenericContainer(context.Background(), genReq)
			require.NoError(t, err)

			defer func() {
				err := container.Terminate(context.Background())
				require.NoError(t, err)
			}()

			exitCode, outputReader, err := container.Exec(context.Background(), []string{"bash", "etc/compile_check.sh"})
			require.NoError(t, err)

			output, err := io.ReadAll(outputReader)
			require.NoError(t, err)

			t.Logf("output: %s", output)
			assert.Equal(t, 0, exitCode)
		})
	}
}

func compareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	n := len(parts1)
	if len(parts2) > n {
		n = len(parts2)
	}

	for i := 0; i < n; i++ {
		var num1, num2 int
		if i < len(parts2) {
			num1, _ = strconv.Atoi(parts1[i])
		}

		if i < len(parts2) {
			num2, _ = strconv.Atoi(parts2[i])
		}

		if num1 < num2 {
			return -1
		} else if num1 > num2 {
			return 1
		}
	}

	return 0
}

type tagResponse struct {
	Results []struct {
		Name string `json:"name"`
	} `json:"results"`
	Next string `json:"next"`
}

func getDockerGolangImages() ([]string, error) {
	url := "https://hub.docker.com/v2/repositories/library/golang/tags?page_size=100"

	versionSet := map[string]bool{}
	for url != "" {
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()

		var data tagResponse
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, err
		}

		finished := false
		for _, tag := range data.Results {
			// Skip tags that don't start with a digit (e.g. alpine, buster, etc)
			if len(tag.Name) == 0 || tag.Name[0] < '0' || tag.Name[0] > '9' {
				continue
			}

			// Extract the base version (e.g. 1.18.1 from 1.18.1-alpine)
			base := strings.Split(tag.Name, "-")[0]

			// If its not at least three characters, do nothing.
			if len(base) < 3 {
				continue
			}

			// Skip release candidates.
			if strings.Contains(base, "rc") {
				continue
			}

			// Only take major versions.
			if strings.Count(base, ".") > 1 {
				continue
			}

			if compareVersions(base, minSupportedVersion) >= 0 {
				versionSet[base] = true

				continue
			} else {
				finished = true
			}
		}

		if finished {
			break
		}

		url = data.Next
	}

	versions := []string{}
	for v := range versionSet {
		versions = append(versions, v)
	}

	return versions, nil
}
