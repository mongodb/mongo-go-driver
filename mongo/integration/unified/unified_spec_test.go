// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"context"
	"fmt"
	"io/ioutil"
	"path"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	testhelpers "go.mongodb.org/mongo-driver/internal/testutil/helpers"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

var (
	directories = []string{
		"unified-test-format/valid-pass",
	}

	skippedTestDescriptions = map[string]struct{}{
		// GODRIVER-1773: This test runs a "find" with limit=4 and batchSize=3. It expects batchSize values of three for
		// the "find" and one for the "getMore", but we send three for both.
		"A successful find event with a getmore and the server kills the cursor": {},

		// This test expects the driver to raise a client-side error when inserting a document with a key that contains
		// a "." or "$". We don't do this validation and the server is moving towards supporting this, so we don't have
		// any plans to add it. This test will need to be changed once the server support lands anyway.
		"Client side error in command starting transaction": {},
	}
)

const (
	dataDirectory               = "../../../data"
	lowHeartbeatFrequency int32 = 50
)

type TestCase struct {
	Description       string             `bson:"description"`
	RunOnRequirements []mtest.RunOnBlock `bson:"runOnRequirements"`
	SkipReason        *string            `bson:"skipReason"`
	Operations        []*Operation       `bson:"operations"`
	ExpectedEvents    []*ExpectedEvents  `bson:"expectEvents"`
	Outcome           []*CollectionData  `bson:"outcome"`
}

func (t *TestCase) PerformsDistinct() bool {
	return t.performsOperation("distinct")
}

func (t *TestCase) SetsFailPoint() bool {
	return t.performsOperation("failPoint")
}

func (t *TestCase) StartsTransaction() bool {
	return t.performsOperation("startTransaction")
}

func (t *TestCase) performsOperation(name string) bool {
	for _, op := range t.Operations {
		if op.Name == name {
			return true
		}
	}
	return false
}

type TestFile struct {
	Description       string                      `bson:"description"`
	SchemaVersion     string                      `bson:"schemaVersion"`
	RunOnRequirements []mtest.RunOnBlock          `bson:"runOnRequirements"`
	CreateEntities    []map[string]*EntityOptions `bson:"createEntities"`
	InitialData       []*CollectionData           `bson:"initialData"`
	TestCases         []*TestCase                 `bson:"tests"`
}

func TestUnifiedSpec(t *testing.T) {
	// Ensure the cluster is in a clean state before test execution begins.
	if err := TerminateOpenSessions(mtest.Background); err != nil {
		t.Fatalf("error terminating open transactions: %v", err)
	}

	for _, testDir := range directories {
		t.Run(testDir, func(t *testing.T) {
			for _, filename := range testhelpers.FindJSONFilesInDir(t, path.Join(dataDirectory, testDir)) {
				t.Run(filename, func(t *testing.T) {
					runTestFile(t, path.Join(dataDirectory, testDir, filename))
				})
			}
		})
	}
}

func runTestFile(t *testing.T, filepath string) {
	content, err := ioutil.ReadFile(filepath)
	assert.Nil(t, err, "ReadFile error for file %q: %v", filepath, err)

	var testFile TestFile
	err = bson.UnmarshalExtJSON(content, false, &testFile)
	assert.Nil(t, err, "UnmarshalExtJSON error for file %q: %v", filepath, err)

	// Validate that we support the schema declared by the test file before attempting to use its contents.
	err = CheckSchemaVersion(testFile.SchemaVersion)
	assert.Nil(t, err, "schema version %q not supported: %v", testFile.SchemaVersion, err)

	// Create mtest wrapper, which will skip the test if needed.
	mtOpts := mtest.NewOptions().
		RunOn(testFile.RunOnRequirements...).
		CreateClient(false)
	mt := mtest.New(t, mtOpts)
	defer mt.Close()

	for _, testCase := range testFile.TestCases {
		mtOpts := mtest.NewOptions().
			RunOn(testCase.RunOnRequirements...).
			CreateClient(false)
		mt.RunOpts(testCase.Description, mtOpts, func(mt *mtest.T) {
			runTestCase(mt, testFile, testCase)
		})
	}
}

func runTestCase(mt *mtest.T, testFile TestFile, testCase *TestCase) {
	if testCase.SkipReason != nil {
		mt.Skipf("skipping for reason: %q", *testCase.SkipReason)
	}
	if _, ok := skippedTestDescriptions[testCase.Description]; ok {
		mt.Skip("skipping due to known failure")
	}

	testCtx := NewTestContext(mtest.Background)

	defer func() {
		// If anything fails while doing test cleanup, we only log the error because the actual test may have already
		// failed and that failure should be preserved.

		for _, err := range DisableUntargetedFailPoints(testCtx) {
			mt.Log(err)
		}
		for _, err := range DisableTargetedFailPoints(testCtx) {
			mt.Log(err)
		}
		for _, err := range Entities(testCtx).Close(testCtx) {
			mt.Log(err)
		}
		// Tests that started a transaction should terminate any sessions left open on the server. This is required even
		// if the test attempted to commit/abort the transaction because an abortTransaction command can fail if it's
		// sent to a mongos that isn't aware of the transaction.
		if testCase.StartsTransaction() {
			if err := TerminateOpenSessions(mtest.Background); err != nil {
				mt.Logf("error terminating open transactions after failed test: %v", err)
			}
		}
	}()

	// Set up collections based on the file-level initialData field.
	for _, collData := range testFile.InitialData {
		if err := collData.CreateCollection(testCtx); err != nil {
			mt.Fatalf("error setting up collection %q: %v", collData.Namespace(), err)
		}
	}

	// Set up entities based on the file-level createEntities field. For client entities, if the test will configure
	// a fail point, set a low heartbeatFrequencyMS value into the URI options map if one is not already present.
	// This speeds up recovery time for the client if the fail point forces the server to return a state change
	// error.
	shouldSetHeartbeatFrequency := testCase.SetsFailPoint()
	for idx, entity := range testFile.CreateEntities {
		for entityType, entityOptions := range entity {
			if shouldSetHeartbeatFrequency && entityType == "client" {
				if entityOptions.URIOptions == nil {
					entityOptions.URIOptions = make(bson.M)
				}
				if _, ok := entityOptions.URIOptions["heartbeatFrequencyMS"]; !ok {
					entityOptions.URIOptions["heartbeatFrequencyMS"] = lowHeartbeatFrequency
				}
			}

			if err := Entities(testCtx).AddEntity(testCtx, entityType, entityOptions); err != nil {
				mt.Fatalf("error creating entity at index %d: %v", idx, err)
			}
		}
	}

	// Work around SERVER-39704.
	if mtest.ClusterTopologyKind() == mtest.Sharded && testCase.PerformsDistinct() {
		if err := PerformDistinctWorkaround(testCtx); err != nil {
			mt.Fatalf("error performing \"distinct\" workaround: %v", err)
		}
	}

	for idx, operation := range testCase.Operations {
		err := operation.Execute(testCtx)
		assert.Nil(mt, err, "error running operation %q at index %d: %v", operation.Name, idx, err)
	}

	for _, client := range Entities(testCtx).Clients() {
		client.StopListeningForEvents()
	}

	for idx, expectedEvents := range testCase.ExpectedEvents {
		err := VerifyEvents(testCtx, expectedEvents)
		assert.Nil(mt, err, "events verification failed at index %d: %v", idx, err)
	}

	for idx, collData := range testCase.Outcome {
		err := collData.VerifyContents(testCtx)
		assert.Nil(mt, err, "error verifying outcome for collection %q at index %d: %v",
			collData.Namespace(), idx, err)
	}
}

func DisableUntargetedFailPoints(ctx context.Context) []error {
	var errs []error
	for fpName, client := range FailPoints(ctx) {
		if err := disableFailPointWithClient(ctx, fpName, client); err != nil {
			errs = append(errs, fmt.Errorf("error disabling fail point %q: %v", fpName, err))
		}
	}
	return errs
}

func DisableTargetedFailPoints(ctx context.Context) []error {
	var errs []error
	for fpName, host := range TargetedFailPoints(ctx) {
		commandFn := func(ctx context.Context, client *mongo.Client) error {
			return disableFailPointWithClient(ctx, fpName, client)
		}
		if err := RunCommandOnHost(ctx, host, commandFn); err != nil {
			errs = append(errs, fmt.Errorf("error disabling targeted fail point %q on host %q: %v", fpName, host, err))
		}
	}
	return errs
}

func disableFailPointWithClient(ctx context.Context, fpName string, client *mongo.Client) error {
	cmd := bson.D{
		{"configureFailPoint", fpName},
		{"mode", "off"},
	}
	if err := client.Database("admin").RunCommand(ctx, cmd).Err(); err != nil {
		return err
	}
	return nil
}
