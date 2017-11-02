// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package testhelpers

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
