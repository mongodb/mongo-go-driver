package drivergen

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"golang.org/x/tools/imports"
)

func TestParseFile(t *testing.T) {
	op, err := ParseFile("example.operation.toml", "operation")
	spew.Dump(err, op)
	var b bytes.Buffer
	err = op.Generate(&b)
	if err != nil {
		t.Fatalf("Unexpected error while generating operation: %v", err)
	}
	fmt.Println(b.String())
	buf, err := imports.Process("~/src/x/operation/operation.go", b.Bytes(), nil)
	spew.Dump(err)
	fmt.Println(string(buf))
}
