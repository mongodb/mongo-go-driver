package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

// CompressJSON takes in a JSON string and removes whitespace from it.
func CompressJSON(js string) string {
	var jsonBytes = []byte(js)
	buffer := new(bytes.Buffer)
	if err := json.Compact(buffer, jsonBytes); err != nil {
		fmt.Println(err)
	}
	return buffer.String()
}

// FindJSONFilesInDir takes in a directory as a string and returns all JSON files from that directory.
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
