// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package prose

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

var (
	loadSecretsFlag = flag.Bool("load-secrets", false, "Use AWS SSO login to load secrets into os.TempDir()")
	cseFlag         = flag.Bool("cse", false, "Setup required for running CSE tests")
)

func secretsRequested() bool {
	return *loadSecretsFlag || *cseFlag
}

func TestMain(m *testing.M) {
	flag.Parse()

	if secretsRequested() {
		if err := loadSecrets(); err != nil {
			fmt.Fprintln(os.Stderr, "loading secrets:", err)
			os.Exit(1)
		}
	}

	os.Exit(m.Run())
}

func TestSecretsLoaded(t *testing.T) {
	if !secretsRequested() {
		t.Skip("pass -load-secrets or -cse to load secrets")
	}

	path, err := exportSecrets()
	if err != nil {
		t.Fatalf("failed to export secrets: %v", err)
	}

	secrets, err := godotenv.Read(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	if len(secrets) == 0 {
		t.Fatalf("%s exported no secrets", path)
	}

	for key, want := range secrets {
		if os.Getenv(key) != want {
			t.Errorf("expected %s to be loaded from %s", key, secretsFileName)
		}
	}
}
