// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"testing"

	"path"

	"go.mongodb.org/mongo-driver/internal/testutil/helpers"
)

const convenientTransactionTestsDir = "../data/convenient-transactions"

type runOn struct {
	MinServerVersion string   `json:"minServerVersion"`
	MaxServerVersion string   `json:"maxServerVersion"`
	Topology         []string `json:"topology"`
}

type withTransactionArgs struct {
	Callback *struct {
		Operations []*transOperation `json:"operations"`
	} `json:"callback"`
	Options map[string]interface{} `json:"options"`
}

// test case for all TransactionSpec tests
func TestConvTransactionSpec(t *testing.T) {
	for _, file := range testhelpers.FindJSONFilesInDir(t, convenientTransactionTestsDir) {
		runTransactionTestFile(t, path.Join(convenientTransactionTestsDir, file))
	}
}
