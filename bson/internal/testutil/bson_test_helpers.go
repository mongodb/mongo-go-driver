package testutil

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path"
	"testing"

	"github.com/10gen/mongo-go-driver/bson/internal/json"
	"github.com/stretchr/testify/require"
)

func CompressJSON(js string) string {
	var json_bytes = []byte(js)
	buffer := new(bytes.Buffer)
	if err := json.Compact(buffer, json_bytes); err != nil {
		fmt.Println(err)
	}
	return buffer.String()
}

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
