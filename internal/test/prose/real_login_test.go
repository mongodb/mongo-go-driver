// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package prose

import (
	"path/filepath"
	"testing"
)

func TestRealLogin(t *testing.T) {
	dir := ExportSecrets(t)

	secrets, err := parseSecretsFile(filepath.Join(dir, secretsFileName))
	if err != nil {
		t.Fatalf("failed to read exported secrets: %v", err)
	}

	for _, key := range []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN"} {
		if secrets[key] == "" {
			t.Errorf("expected %s to be set", key)
		}
	}

	t.Logf("exported %d credentials to %s", len(secrets), dir)
}
