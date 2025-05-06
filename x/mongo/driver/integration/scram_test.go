// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/driverutil"
	"go.mongodb.org/mongo-driver/v2/internal/integtest"
	"go.mongodb.org/mongo-driver/v2/internal/serverselector"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
)

type scramTestCase struct {
	username    string
	password    string
	mechanisms  []string
	altPassword string
}

func TestSCRAM(t *testing.T) {
	if os.Getenv("AUTH") != "auth" {
		t.Skip("Skipping because authentication is required")
	}

	server, err := integtest.Topology(t).SelectServer(context.Background(), &serverselector.Write{})
	noerr(t, err)
	serverConnection, err := server.Connection(context.Background())
	noerr(t, err)
	defer serverConnection.Close()

	if !driverutil.VersionRangeIncludes(*serverConnection.Description().WireVersion, 7) {
		t.Skip("Skipping because MongoDB 4.0 is needed for SCRAM-SHA-256")
	}

	// Unicode constants for testing
	var romanFour = "\u2163" // ROMAN NUMERAL FOUR -> SASL prepped is "IV"
	var romanNine = "\u2168" // ROMAN NUMERAL NINE -> SASL prepped is "IX"

	testUsers := []scramTestCase{
		// SCRAM spec test steps 1-3
		{username: "sha1", password: "sha1", mechanisms: []string{"SCRAM-SHA-1"}},
		{username: "sha256", password: "sha256", mechanisms: []string{"SCRAM-SHA-256"}},
		{username: "both", password: "both", mechanisms: []string{"SCRAM-SHA-1", "SCRAM-SHA-256"}},
		// SCRAM spec test step 4
		{username: "IX", password: "IX", mechanisms: []string{"SCRAM-SHA-256"}, altPassword: "I\u00ADX"},
		{username: romanNine, password: romanFour, mechanisms: []string{"SCRAM-SHA-256"}, altPassword: "I\u00ADV"},
	}

	// Verify that test (root) user is authenticated.  If this fails, the
	// rest of the test can't succeed.
	wc := writeconcern.Majority()
	collOne := integtest.ColName(t)
	dropCollection(t, integtest.DBName(t), collOne)
	insertDocs(t, integtest.DBName(t),
		collOne, wc, bsoncore.BuildDocument(nil, bsoncore.AppendStringElement(nil, "name", "scram_test")),
	)

	// Test step 1: Create users for test cases
	err = createScramUsers(t, server, testUsers)
	if err != nil {
		t.Fatal(err)
	}

	// Step 2 and 3a: For each auth mechanism, "SCRAM-SHA-1", "SCRAM-SHA-256"
	// and "negotiate" (a fake, placeholder mechanism), iterate over each user
	// and ensure that each mechanism that should succeed does so and each
	// that should fail does so.
	for _, m := range []string{"SCRAM-SHA-1", "SCRAM-SHA-256", "negotiate"} {
		for _, c := range testUsers {
			t.Run(
				fmt.Sprintf("%s %s", c.username, m),
				func(t *testing.T) {
					err := testScramUserAuthWithMech(t, c, m)
					if m == "negotiate" || hasAuthMech(c.mechanisms, m) {
						noerr(t, err)
					} else {
						autherr(t, err)
					}
				},
			)
		}
	}

	// Step 3b: test non-existing user with negotiation fails with
	// an auth.Error type.
	bogus := scramTestCase{username: "eliot", password: "trustno1"}
	err = testScramUserAuthWithMech(t, bogus, "negotiate")
	autherr(t, err)

	// XXX Step 4: test alternate password forms
	for _, c := range testUsers {
		if c.altPassword == "" {
			continue
		}
		c.password = c.altPassword
		t.Run(
			fmt.Sprintf("%s alternate password", c.username),
			func(t *testing.T) {
				err := testScramUserAuthWithMech(t, c, "SCRAM-SHA-256")
				noerr(t, err)
			},
		)
	}

}

func hasAuthMech(mechs []string, m string) bool {
	for _, v := range mechs {
		if v == m {
			return true
		}
	}
	return false
}

func testScramUserAuthWithMech(t *testing.T, c scramTestCase, mech string) error {
	t.Helper()
	credential := options.Credential{
		Username:   c.username,
		Password:   c.password,
		AuthSource: integtest.DBName(t),
	}
	switch mech {
	case "negotiate":
		credential.AuthMechanism = ""
	default:
		credential.AuthMechanism = mech
	}
	return runScramAuthTest(t, credential)
}

func runScramAuthTest(t *testing.T, credential options.Credential) error {
	t.Helper()
	topology := integtest.TopologyWithCredential(t, credential)
	server, err := topology.SelectServer(context.Background(), &serverselector.Write{})
	noerr(t, err)

	cmd := bsoncore.BuildDocument(nil, bsoncore.AppendInt32Element(nil, "dbstats", 1))
	return runCommand(server, integtest.DBName(t), cmd)
}

func createScramUsers(t *testing.T, s driver.Server, cases []scramTestCase) error {
	db := integtest.DBName(t)
	for _, c := range cases {
		var values []bsoncore.Value
		for _, v := range c.mechanisms {
			values = append(values, bsoncore.Value{Type: bsoncore.TypeString, Data: bsoncore.AppendString(nil, v)})
		}
		newUserCmd := bsoncore.BuildDocumentFromElements(nil,
			bsoncore.AppendStringElement(nil, "createUser", c.username),
			bsoncore.AppendStringElement(nil, "pwd", c.password),
			bsoncore.AppendArrayElement(nil, "roles", bsoncore.BuildArray(nil,
				bsoncore.Value{Type: bsoncore.TypeEmbeddedDocument, Data: bsoncore.BuildDocumentFromElements(nil,
					bsoncore.AppendStringElement(nil, "role", "readWrite"),
					bsoncore.AppendStringElement(nil, "db", db),
				)},
			)),
			bsoncore.AppendArrayElement(nil, "mechanisms", bsoncore.BuildArray(nil, values...)),
		)
		err := runCommand(s, db, newUserCmd)
		if err != nil {
			return fmt.Errorf("Couldn't create user '%s' on db '%s': %w", c.username, integtest.DBName(t), err)
		}
	}
	return nil
}
