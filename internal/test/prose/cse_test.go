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

func TestClientSideEncryptionProse_27(t *testing.T) {
	goTestCSE(t, "./internal/integration", "TestClientSideEncryptionProse_27")
}

func goTestCSE(t *testing.T, pkg, name string) {
	t.Helper()

	if !*cseFlag {
		t.Skip("pass -cse to run CSE tests")
	}

	run := "^" + name + "$"
	if f := flag.Lookup("test.run"); f != nil && f.Value.String() != "" {
		if _, sub, ok := strings.Cut(f.Value.String(), "/"); ok {
			run += "/" + sub
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
