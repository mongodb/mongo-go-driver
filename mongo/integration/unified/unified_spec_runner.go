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
	lowHeartbeatFrequency int32 = 50
)

type testCase struct {
	Description       string             `bson:"description"`
	RunOnRequirements []mtest.RunOnBlock `bson:"runOnRequirements"`
	SkipReason        *string            `bson:"skipReason"`
	Operations        []*operation       `bson:"operations"`
	expectedEvents    []*expectedEvents  `bson:"expectEvents"`
	Outcome           []*collectionData  `bson:"outcome"`
}

func (t *testCase) performsDistinct() bool {
	return t.performsOperation("distinct")
}

func (t *testCase) setsFailPoint() bool {
	return t.performsOperation("failPoint")
}

func (t *testCase) startsTransaction() bool {
	return t.performsOperation("startTransaction")
}

func (t *testCase) performsOperation(name string) bool {
	for _, op := range t.Operations {
		if op.Name == name {
			return true
		}
	}
	return false
}

// TestFile holds the contents of a unified spec test file
type TestFile struct {
	Description       string                      `bson:"description"`
	SchemaVersion     string                      `bson:"schemaVersion"`
	RunOnRequirements []mtest.RunOnBlock          `bson:"runOnRequirements"`
	CreateEntities    []map[string]*entityOptions `bson:"createEntities"`
	InitialData       []*collectionData           `bson:"initialData"`
	TestCases         []*testCase                 `bson:"tests"`
}

// RunTestDirectory runs the files in the given directory, which must be in the unifed spec format
func RunTestDirectory(t *testing.T, directoryPath string) {
	for _, filename := range testhelpers.FindJSONFilesInDir(t, directoryPath) {
		t.Run(filename, func(t *testing.T) {
			RunTestFile(t, path.Join(directoryPath, filename))
		})
	}
}

// RunTestFile runs the tests in the given file, which must be in the unifed spec format
func RunTestFile(t *testing.T, filepath string) {
	content, err := ioutil.ReadFile(filepath)
	assert.Nil(t, err, "ReadFile error for file %q: %v", filepath, err)

	var testFile TestFile
	err = bson.UnmarshalExtJSON(content, false, &testFile)
	assert.Nil(t, err, "UnmarshalExtJSON error for file %q: %v", filepath, err)

	// Validate that we support the schema declared by the test file before attempting to use its contents.
	err = checkSchemaVersion(testFile.SchemaVersion)
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

func runTestCase(mt *mtest.T, testFile TestFile, testCase *testCase) {
	if testCase.SkipReason != nil {
		mt.Skipf("skipping for reason: %q", *testCase.SkipReason)
	}
	if _, ok := skippedTestDescriptions[testCase.Description]; ok {
		mt.Skip("skipping due to known failure")
	}

	testCtx := newTestContext(mtest.Background)

	defer func() {
		// If anything fails while doing test cleanup, we only log the error because the actual test may have already
		// failed and that failure should be preserved.

		for _, err := range DisableUntargetedFailPoints(testCtx) {
			mt.Log(err)
		}
		for _, err := range DisableTargetedFailPoints(testCtx) {
			mt.Log(err)
		}
		for _, err := range entities(testCtx).close(testCtx) {
			mt.Log(err)
		}
		// Tests that started a transaction should terminate any sessions left open on the server. This is required even
		// if the test attempted to commit/abort the transaction because an abortTransaction command can fail if it's
		// sent to a mongos that isn't aware of the transaction.
		if testCase.startsTransaction() {
			if err := terminateOpenSessions(mtest.Background); err != nil {
				mt.Logf("error terminating open transactions after failed test: %v", err)
			}
		}
	}()

	// Set up collections based on the file-level initialData field.
	for _, collData := range testFile.InitialData {
		if err := collData.createCollection(testCtx); err != nil {
			mt.Fatalf("error setting up collection %q: %v", collData.namespace(), err)
		}
	}

	// Set up entities based on the file-level createEntities field. For client entities, if the test will configure
	// a fail point, set a low heartbeatFrequencyMS value into the URI options map if one is not already present.
	// This speeds up recovery time for the client if the fail point forces the server to return a state change
	// error.
	shouldSetHeartbeatFrequency := testCase.setsFailPoint()
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

			if err := entities(testCtx).addEntity(testCtx, entityType, entityOptions); err != nil {
				mt.Fatalf("error creating entity at index %d: %v", idx, err)
			}
		}
	}

	// Work around SERVER-39704.
	if mtest.ClusterTopologyKind() == mtest.Sharded && testCase.performsDistinct() {
		if err := performDistinctWorkaround(testCtx); err != nil {
			mt.Fatalf("error performing \"distinct\" workaround: %v", err)
		}
	}

	for idx, operation := range testCase.Operations {
		err := operation.execute(testCtx)
		assert.Nil(mt, err, "error running operation %q at index %d: %v", operation.Name, idx, err)
	}

	for _, client := range entities(testCtx).clients() {
		client.StopListeningForEvents()
	}

	for idx, expectedEvents := range testCase.expectedEvents {
		err := verifyEvents(testCtx, expectedEvents)
		assert.Nil(mt, err, "events verification failed at index %d: %v", idx, err)
	}

	for idx, collData := range testCase.Outcome {
		err := collData.verifyContents(testCtx)
		assert.Nil(mt, err, "error verifying outcome for collection %q at index %d: %v",
			collData.namespace(), idx, err)
	}
}

// DisableUntargetedFailPoints turns off untargeted fail points
func DisableUntargetedFailPoints(ctx context.Context) []error {
	var errs []error
	for fpName, client := range failPoints(ctx) {
		if err := disableFailPointWithClient(ctx, fpName, client); err != nil {
			errs = append(errs, fmt.Errorf("error disabling fail point %q: %v", fpName, err))
		}
	}
	return errs
}

// DisableTargetedFailPoints turns off Targeted fail points
func DisableTargetedFailPoints(ctx context.Context) []error {
	var errs []error
	for fpName, host := range targetedFailPoints(ctx) {
		commandFn := func(ctx context.Context, client *mongo.Client) error {
			return disableFailPointWithClient(ctx, fpName, client)
		}
		if err := runCommandOnHost(ctx, host, commandFn); err != nil {
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
