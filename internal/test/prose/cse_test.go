// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package prose

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"testing"

	"github.com/joho/godotenv"
)

func TestCSE(t *testing.T) {
	if !*cseFlag {
		t.Skip("pass -cse to run CSE tests")
	}

	t.Run("mongocrypt", func(t *testing.T) {
		exit, out, err := execCSE(context.Background(), "go test -tags cse ./x/mongo/driver/mongocrypt")
		if err != nil {
			t.Fatalf("failed to run CSE tests: %v", err)
		}
		if exit != 0 {
			t.Fatalf("go test failed with exit code %d:\n%s", exit, out)
		}
	})

	t.Run("secrets", func(t *testing.T) {
		secrets, err := godotenv.Read(loadedSecretsPath)
		if err != nil {
			t.Fatalf("failed to read %s: %v", loadedSecretsPath, err)
		}

		exit, out, err := execCSE(context.Background(), "env | cut -d= -f1")
		if err != nil {
			t.Fatalf("failed to list container environment: %v", err)
		}
		if exit != 0 {
			t.Fatalf("env failed with exit code %d:\n%s", exit, out)
		}

		set := make(map[string]bool)
		for _, name := range strings.Fields(out) {
			set[name] = true
		}
		for key := range secrets {
			if !set[key] {
				t.Errorf("expected %s from %s to be set in the CSE container", key, secretsFileName)
			}
		}
	})

	t.Run("ping", func(t *testing.T) {
		exit, out, err := execCSE(context.Background(), "go run ./internal/cmd/testping")
		if err != nil {
			t.Fatalf("failed to run ping: %v", err)
		}
		if exit != 0 {
			t.Fatalf("ping failed with exit code %d:\n%s", exit, out)
		}
	})
}

func TestCSEIntegration(t *testing.T) {
	if !*cseFlag {
		t.Skip("pass -cse to run CSE tests")
	}

	names := listTestsCSE(t, "./internal/integration")
	if len(names) == 0 {
		t.Fatal("found no cse-tagged tests in ./internal/integration")
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			goTestCSE(t, "./internal/integration", name)
		})
	}
}

// listTestsCSE returns the tests in pkg that only exist with the cse build tag.
func listTestsCSE(t *testing.T, pkg string) []string {
	t.Helper()

	with := listTests(t, pkg, "-tags", "cse")
	without := make(map[string]bool)
	for _, name := range listTests(t, pkg) {
		without[name] = true
	}

	var names []string
	for _, name := range with {
		if !without[name] {
			names = append(names, name)
		}
	}
	return names
}

func listTests(t *testing.T, pkg string, flags ...string) []string {
	t.Helper()

	cmd := fmt.Sprintf("go test -list . %s %s", strings.Join(flags, " "), pkg)
	exit, out, err := execCSE(context.Background(), cmd)
	if err != nil {
		t.Fatalf("failed to list tests: %v", err)
	}
	if exit != 0 {
		t.Fatalf("%q failed with exit code %d:\n%s", cmd, exit, out)
	}

	// The output is one test name per line, followed by a line like
	// "ok  <pkg>  <time>".
	var names []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Test") && !strings.ContainsAny(line, " \t") {
			names = append(names, line)
		}
	}
	return names
}

func goTestCSE(t *testing.T, pkg, name string) {
	t.Helper()

	if !*cseFlag {
		t.Skip("pass -cse to run CSE tests")
	}

	run := "^" + name + "$"
	if f := flag.Lookup("test.run"); f != nil && f.Value.String() != "" {
		// The pattern is TestCSEIntegration/<test>/<subtest>...; only the part
		// after <test> belongs to the test running in the container.
		if parts := strings.SplitN(f.Value.String(), "/", 3); len(parts) == 3 {
			run += "/" + parts[2]
		}
	}

	exit, out, err := execCSE(context.Background(), fmt.Sprintf("go test -tags cse -v -run '%s' %s", run, pkg))
	if err != nil {
		t.Fatalf("failed to run %s: %v", name, err)
	}
	t.Log(out)
	if exit != 0 {
		t.Fatalf("%s failed with exit code %d", name, exit)
	}
}
