// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package testutil

import (
	"io"
	"testing"
)

func CloseOrError(t *testing.T, c io.Closer) {
	if err := c.Close(); err != nil {
		t.Errorf("unable to close file: %v", err)
	}
}

func CloseReadOnlyFile(c io.Closer) {
	_ = c.Close()
}
