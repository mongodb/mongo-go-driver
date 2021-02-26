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

	if awsAccessKeyID == "" {
		t.Fatal("AWS access key ID not set")
	}
	if awsSecretAccessKey == "" {
		t.Fatal("AWS secret access key not set")
	}
	if awsTempAccessKeyID == "" {
		t.Fatal("AWS temp access key ID not set")
	}
	if awsTempSecretAccessKey == "" {
		t.Fatal("AWS temp secret access key not set")
	}
	if awsTempSessionToken == "" {
		t.Fatal("AWS temp session token not set")
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
