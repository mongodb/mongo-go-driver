// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/bitfield/script"
)

func main() {
	// Check if gofumpt is installed
	if _, err := exec.LookPath("gofumpt"); err != nil {
		fmt.Fprintln(os.Stderr, "gofumpt is not installed. Install it with: go install mvdan.cc/gofumpt@latest")
		os.Exit(1)
	}

	gofumptOut, _ := script.Exec("gofumpt -l .").String()

	if strings.TrimSpace(gofumptOut) != "" {
		fmt.Println("gofumpt check failed for:")
		script.Echo(gofumptOut).FilterLine(func(line string) string {
			return " - " + line
		}).Stdout()
		os.Exit(1)
	}

	// Use the "github.com/walle/lll" tool to check that all lines in *_example_test.go files are
	// wrapped at 80 characters to keep them readable when rendered on https://pkg.go.dev.
	// Ignore long lines that are comments containing URI-like strings and testable example output
	// comments like "// Output: ...".
	// E.g ignored lines:
	//     // "mongodb://ldap-user:ldap-pwd@localhost:27017/?authMechanism=PLAIN"
	//     // (https://www.mongodb.com/docs/manual/core/authentication-mechanisms-enterprise/#security-auth-ldap).
	//     // Output: {"myint": {"$numberLong":"1"},"int32": {"$numberLong":"1"},"int64": {"$numberLong":"1"}}
	filesPipe := script.FindFiles(".").Match("_examples_test.go")
	cmd := exec.Command("lll", "-w", "4", "-l", "80", "-e", `^\s*\/\/(.+:\/\/| Output:)`, "--files")
	cmd.Stdin = filesPipe // pipe the file list directly as stdin to lll
	var lllBuf bytes.Buffer
	cmd.Stdout = &lllBuf
	cmd.Run()

	lllOut := lllBuf.String()
	if strings.TrimSpace(lllOut) != "" {
		fmt.Println("lll check failed for:")
		script.Echo(lllOut).FilterLine(func(line string) string {
			return " - " + line
		}).Stdout()
		os.Exit(1)
	}
}
