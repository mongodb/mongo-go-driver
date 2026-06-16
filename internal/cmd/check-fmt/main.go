// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/bitfield/script"
)

// requireTool checks that the given tool is installed and exits with an error
// message if it is not.
func requireTool(toolName, installCmd string) {
	if _, err := exec.LookPath(toolName); err != nil {
		fmt.Fprintf(os.Stderr, "%s is not installed. Install it with: %s\n", toolName, installCmd)
		os.Exit(1)
	}
}

// checkGofumpt runs gofumpt on all packages in the repo and exits if any
// files are not formatted.
func checkGofumpt() {
	requireTool("gofumpt", "go install mvdan.cc/gofumpt@latest")
	pipe := script.Exec("gofumpt -l .")
	outputPipe(pipe, "gofumpt check failed for:")
}

// checkLll uses the "github.com/walle/lll" tool to check that all lines in *_examples_test.go
// and *_example_test.go files are wrapped at 80 characters to keep them readable when rendered
// on https://pkg.go.dev.
// Ignore long lines that are comments containing URI-like strings and testable example output
// comments like "// Output: ...".
// E.g ignored lines:
//
//	// "mongodb://ldap-user:ldap-pwd@localhost:27017/?authMechanism=PLAIN"
//	// (https://www.mongodb.com/docs/manual/core/authentication-mechanisms-enterprise/#security-auth-ldap).
//	// Output: {"myint": {"$numberLong":"1"},"int32": {"$numberLong":"1"},"int64": {"$numberLong":"1"}}
func checkLll() {
	requireTool("lll", "go install github.com/walle/lll@latest")
	pipe := script.Exec(`find . -type f -regex ".*_example.*_test\.go"`).
		Exec(`lll -w 4 -l 80 -e '^\s*\/\/(.+:\/\/| Output:)' --files`)
	outputPipe(pipe, "lll check failed for:")
}

type writerWithHeader struct {
	header string
	w      io.Writer
}

func (wh *writerWithHeader) Write(p []byte) (n int, err error) {
	if len(wh.header) > 0 {
		n, err := wh.w.Write([]byte(wh.header + "\n"))
		if err != nil {
			return n, err
		}
		wh.header = ""
	}
	return wh.w.Write(p)
}

func outputPipe(pipe *script.Pipe, header string) {
	out := &writerWithHeader{header: header, w: os.Stdout}
	pipe.WithStdout(out)
	n, err := pipe.FilterLine(func(line string) string {
		return " - " + line
	}).WithStdout(out).Stdout()
	if err != nil {
		fmt.Fprintln(os.Stderr, "execution failed:", err)
		os.Exit(1)
	}
	if n > 0 {
		os.Exit(1)
	}
}

func main() {
	checkGofumpt()
	checkLll()
}
