// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

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
	output, err := script.Exec("gofumpt -l .").String()
	if err != nil {
		fmt.Fprintln(os.Stderr, "gofumpt execution failed:", err)
		os.Exit(1)
	}
	if strings.TrimSpace(output) != "" {
		failWithOutput("gofumpt check failed for:", output)
	}
}

// checkLll runs the lll tool on all *_examples_test.go files and exits if any lines are too long.
// Use the "github.com/walle/lll" tool to check that all lines in *_examples_test.go files are
// wrapped at 80 characters to keep them readable when rendered on https://pkg.go.dev.
// Ignore long lines that are comments containing URI-like strings and testable example output
// comments like "// Output: ...".
// E.g ignored lines:
//
//	// "mongodb://ldap-user:ldap-pwd@localhost:27017/?authMechanism=PLAIN"
//	// (https://www.mongodb.com/docs/manual/core/authentication-mechanisms-enterprise/#security-auth-ldap).
//	// Output: {"myint": {"$numberLong":"1"},"int32": {"$numberLong":"1"},"int64": {"$numberLong":"1"}}
func checkLll() {
	output, err := script.Exec(`find . -type f -name "*_examples_test.go"`).
		Exec(`lll -w 4 -l 80 -e '^\s*\/\/(.+:\/\/| Output:)' --files`).
		String()
	if err != nil {
		fmt.Fprintln(os.Stderr, "lll execution failed:", err)
		os.Exit(1)
	}
	if strings.TrimSpace(output) != "" {
		failWithOutput("lll check failed for:", output)
	}
}

func failWithOutput(header, output string) {
	fmt.Println(header)
	script.Echo(output).FilterLine(func(line string) string {
		return " - " + line
	}).Stdout()
	os.Exit(1)
}

func main() {
	requireTool("gofumpt", "go install mvdan.cc/gofumpt@latest")
	requireTool("lll", "go install github.com/walle/lll@latest")

	checkGofumpt()
	checkLll()
}
