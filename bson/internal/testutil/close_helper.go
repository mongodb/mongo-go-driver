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
