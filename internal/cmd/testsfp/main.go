// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// To source environment locally, run the following command in your terminal:
//
//		aws sso login --profile $AWS_PROFILE
//		bash $DRIVERS_TOOLS/.evergreen/secrets_handling/setup-secrets.sh drivers/sfp && source ./secrets-export.sh
//
// See here for more information:
// https://github.com/mongodb-labs/drivers-evergreen-tools/blob/master/.evergreen/secrets_handling/README.md

package main

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	log.SetFlags(0)

	driversTools := os.Getenv("DRIVERS_TOOLS")
	if driversTools == "" {
		log.Fatal("DRIVERS_TOOLS environment variable is not set")
	}

	secrets, err := fetchSecrets(driversTools)
	if err != nil {
		log.Fatalf("Failed to fetch secrets: %v", err)
	}

	if err := runTests(secrets); err != nil {
		log.Fatalf("Tests failed: %v", err)
	}
}

func fetchSecrets(driversTools string) ([]string, error) {
	sfpSecretVarNAmes := []string{
		"SFP_ATLAS_URI",
		"SFP_ATLAS_USER",
		"SFP_ATLAS_PASSWORD",
		"SFP_ATLAS_X509_URI",
		"SFP_ATLAS_X509_BASE64",
	}

	// If the secrets are already set, do nothing.
	var secrets []string
	for _, secretName := range sfpSecretVarNAmes {
		secret := os.Getenv(secretName)
		if secret == "" {
			break
		}
		secrets = append(secrets, secretName+"="+secret)
	}

	if len(secrets) == len(sfpSecretVarNAmes) {
		log.Println("WARNING: All SFP secrets are already set in the environment. Skipping secret fetch. " +
			"This may cause the tests to fail if the secrets are stale or invalid.")
		return secrets, nil
	}

	// If any secrets are missing, do a complete refresh of all secrets.
	setup := filepath.Join(driversTools, ".evergreen", "secrets_handling", "setup-secrets.sh")
	if _, err := os.Stat(setup); os.IsNotExist(err) {
		return nil, err
	}

	const script = `
source "$SFP_SETUP" drivers/sfp >/dev/null
env -0
`

	args := append([]string{"-c", script, "testsfp"}, sfpSecretVarNAmes...)
	cmd := exec.Command("bash", args...)
	cmd.Env = append(os.Environ(), "SFP_SETUP="+setup)
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	secrets = nil
	for _, kv := range bytes.Split(output, []byte{0}) {
		if k, v, ok := bytes.Cut(kv, []byte{'='}); ok && len(v) > 0 {
			secrets = append(secrets, string(k)+"="+string(v))
		}
	}

	return secrets, nil
}

func runTests(secrets []string) error {
	cmd := exec.Command("go", "test", "./internal/test/sfp", "-v")
	cmd.Env = append(os.Environ(), secrets...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
