// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// +build cse

package integration

import (
	"path"
	"testing"
)

const (
	encryptionSpecName = "client-side-encryption"
)

func verifyClientSideEncryptionVarsSet(t *testing.T) {
	t.Helper()

	// Existence of temporary AWS credentials (awsTempAccessKeyID, awsTempSessionToken and
	// awsTempSessionToken) is verified when the variables are used in json_helpers_test
	// because temporary AWS credentials are not always set.

	if awsAccessKeyID == "" {
		t.Fatal("AWS access key ID not set")
	}
	if awsSecretAccessKey == "" {
		t.Fatal("AWS secret access key not set")
	}
	if azureTenantID == "" {
		t.Fatal("azure tenant ID not set")
	}
	if azureClientID == "" {
		t.Fatal("azure client ID not set")
	}
	if azureClientSecret == "" {
		t.Fatal("azure client secret not set")
	}
	if gcpEmail == "" {
		t.Fatal("GCP email not set")
	}
	if gcpPrivateKey == "" {
		t.Fatal("GCP private key not set")
	}
}

func TestClientSideEncryptionSpec(t *testing.T) {
	verifyClientSideEncryptionVarsSet(t)

	for _, fileName := range jsonFilesInDir(t, path.Join(dataPath, encryptionSpecName)) {
		t.Run(fileName, func(t *testing.T) {
			runSpecTestFile(t, encryptionSpecName, fileName)
		})
	}
}
