package testutil

import (
	"io/ioutil"
	"path"

	"testing"

	"io"

	"github.com/stretchr/testify/require"
)

func FindJSONFilesInDir(t *testing.T, dir string) []string {
	files := make([]string, 0)

	entries, err := ioutil.ReadDir(dir)
	require.NoError(t, err)

	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".json" {
			continue
		}

		files = append(files, entry.Name())
	}

	return files
}

func RequireNoErrorOnClose(t *testing.T, c io.Closer) {
	require.NoError(t, c.Close())
}
