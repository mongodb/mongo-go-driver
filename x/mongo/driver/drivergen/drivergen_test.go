package drivergen

import (
	"bytes"
	"testing"

	"golang.org/x/tools/imports"
)

func TestParseFile(t *testing.T) {
	op, err := ParseFile("example.operation.toml", "operation")
	if err != nil {
		t.Fatalf("Unexepcted error while parsing the operation file: %v", err)
	}
	var b bytes.Buffer
	err = op.Generate(&b)
	if err != nil {
		t.Fatalf("Unexpected error while generating operation: %v", err)
	}
	_, err = imports.Process("~/src/x/operation/operation.go", b.Bytes(), nil)
	if err != nil {
		t.Fatalf("Unexpected error while running imports: %v", err)
	}
}
