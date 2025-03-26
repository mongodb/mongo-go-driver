// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/internal/spectest"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type credential struct {
	Username  string
	Password  *string
	Source    string
	Mechanism string
	MechProps map[string]interface{} `json:"mechanism_properties"`
}

type testCase struct {
	Description string
	URI         string
	Valid       bool
	Credential  *credential
}

type testContainer struct {
	Tests []testCase
}

// Note a test supporting the deprecated gssapiServiceName property was removed from data/auth/auth_tests.json
var authTestsDir = spectest.TestPath(4, "auth", "legacy")

func runTestsInFile(t *testing.T, dirname string, filename string) {
	filepath := path.Join(dirname, filename)
	content, err := ioutil.ReadFile(filepath)
	require.NoError(t, err)

	var container testContainer
	require.NoError(t, json.Unmarshal(content, &container))

	t.Run(filename, func(t *testing.T) {
		for _, testCase := range container.Tests {
			testCase := testCase // Capture range variable.

			t.Run(testCase.Description, func(t *testing.T) {
				spectest.CheckSkip(t)

				runTest(t, testCase)
			})
		}
	})
}

func runTest(t *testing.T, test testCase) {
	opts := options.Client().ApplyURI(test.URI)

	if test.Valid {
		require.NoError(t, opts.Validate())
	} else {
		require.Error(t, opts.Validate())

		return
	}

	if test.Credential == nil {
		require.Nil(t, opts.Auth)
		return
	}
	require.NotNil(t, opts.Auth)
	require.Equal(t, test.Credential.Username, opts.Auth.Username)

	if test.Credential.Password == nil {
		require.False(t, opts.Auth.PasswordSet)
	} else {
		require.True(t, opts.Auth.PasswordSet)
		require.Equal(t, *test.Credential.Password, opts.Auth.Password)
	}

	require.Equal(t, test.Credential.Source, opts.Auth.AuthSource)

	require.Equal(t, test.Credential.Mechanism, opts.Auth.AuthMechanism)

	if len(test.Credential.MechProps) > 0 {
		require.Equal(t, mapInterfaceToString(test.Credential.MechProps), opts.Auth.AuthMechanismProperties)
	} else {
		require.Equal(t, 0, len(opts.Auth.AuthMechanismProperties))
	}
}

// Convert each interface{} value in the map to a string.
func mapInterfaceToString(m map[string]interface{}) map[string]string {
	out := make(map[string]string)

	for key, value := range m {
		out[key] = fmt.Sprint(value)
	}

	return out
}

// Test case for all connection string spec tests.
func TestAuthSpec(t *testing.T) {
	for _, file := range spectest.FindJSONFilesInDir(t, authTestsDir) {
		fmt.Println("file", file)
		runTestsInFile(t, authTestsDir, file)
	}
}
