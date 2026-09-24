// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package prose

import (
	"fmt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if err := loadSecrets(); err != nil {
		fmt.Fprintln(os.Stderr, "loading secrets:", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestSecretsLoaded(t *testing.T) {
	path, err := exportSecrets()
	if err != nil {
		t.Fatalf("failed to export secrets: %v", err)
	}

	secrets, err := readSecrets(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(secrets) == 0 {
		t.Fatalf("%s exported no secrets", path)
	}

	// Values already set in the environment take precedence over the file,
	// so only check that every exported key is present.
	for key := range secrets {
		if os.Getenv(key) == "" {
			t.Errorf("expected %s to be set in the environment", key)
		}
	}
}
