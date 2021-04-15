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

// TestCase holds and runs a unified spec test case
type TestCase struct {
	Description       string             `bson:"description"`
	RunOnRequirements []mtest.RunOnBlock `bson:"runOnRequirements"`
	SkipReason        *string            `bson:"skipReason"`
	Operations        []*operation       `bson:"operations"`
	ExpectedEvents    []*expectedEvents  `bson:"expectEvents"`
	Outcome           []*collectionData  `bson:"outcome"`

	initialData     []*collectionData
	createEntities  []map[string]*entityOptions
	fileReqs        []mtest.RunOnBlock
	killAllSessions bool

	entities *EntityMap
	loopDone chan struct{}
}

func (tc *TestCase) performsDistinct() bool {
	return tc.performsOperation("distinct")
}

func (tc *TestCase) setsFailPoint() bool {
	return tc.performsOperation("failPoint")
}

func (tc *TestCase) startsTransaction() bool {
	return tc.performsOperation("startTransaction")
}

func (tc *TestCase) performsOperation(name string) bool {
	for _, op := range tc.Operations {
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
	TestCases         []*TestCase                 `bson:"tests"`
}

// runTestDirectory runs the files in the given directory, which must be in the unifed spec format
func runTestDirectory(t *testing.T, directoryPath string) {
	for _, filename := range testhelpers.FindJSONFilesInDir(t, directoryPath) {
		t.Run(filename, func(t *testing.T) {
			RunTestFile(t, path.Join(directoryPath, filename))
		})
	}
}

// RunTestFile runs the tests in the given file, which must be in the unifed spec format
func RunTestFile(t *testing.T, filepath string, opts ...*Options) {
	content, err := ioutil.ReadFile(filepath)
	assert.Nil(t, err, "ReadFile error for file %q: %v", filepath, err)

	fileReqs, testCases := ParseTestFile(t, content, opts...)
	mtOpts := mtest.NewOptions().
		RunOn(fileReqs...).
		CreateClient(false)
	mt := mtest.New(t, mtOpts)
	defer mt.Close()

	for _, testCase := range testCases {
		mtOpts := mtest.NewOptions().
			RunOn(testCase.RunOnRequirements...).
			CreateClient(false)

		mt.RunOpts(testCase.Description, mtOpts, func(mt *mtest.T) {
			if err := testCase.Run(mt); err != nil {
				mt.Fatal(err)
			}
		})
	}
}

// ParseTestFile create an array of TestCases from the testJSON json blob
func ParseTestFile(t *testing.T, testJSON []byte, opts ...*Options) ([]mtest.RunOnBlock, []*TestCase) {
	var testFile TestFile
	err := bson.UnmarshalExtJSON(testJSON, false, &testFile)
	assert.Nil(t, err, "UnmarshalExtJSON error: %v", err)

	// Validate that we support the schema declared by the test file before attempting to use its contents.
	err = checkSchemaVersion(testFile.SchemaVersion)
	assert.Nil(t, err, "schema version %q not supported: %v", testFile.SchemaVersion, err)

	op := MergeOptions(opts...)
	for _, testCase := range testFile.TestCases {
		testCase.initialData = testFile.InitialData
		testCase.createEntities = testFile.CreateEntities
		testCase.entities = newEntityMap()
		testCase.loopDone = make(chan struct{})
		testCase.killAllSessions = *op.RunKillAllSessions
	}
	return testFile.RunOnRequirements, testFile.TestCases
}

// GetEntities returns a pointer to the EntityMap for the TestCase. This should not be called until after
// the test is run
func (tc *TestCase) GetEntities() *EntityMap {
	return tc.entities
}

// EndLoop will cause the runner to stop a loop operation if one is included in the test. If the test has finished
// running, this will panic
func (tc *TestCase) EndLoop() {
	tc.loopDone <- struct{}{}
}

// LoggerSkipper is passed to TestCase.Run to allow it to perform logging and skipping operations
type LoggerSkipper interface {
	Log(args ...interface{})
	Logf(format string, args ...interface{})
	Skip(args ...interface{})
	Skipf(format string, args ...interface{})
}

// Run runs the TestCase and returns an error if it fails
func (tc *TestCase) Run(ls LoggerSkipper) error {
	if tc.SkipReason != nil {
		ls.Skipf("skipping for reason: %q", *tc.SkipReason)
	}
	if _, ok := skippedTestDescriptions[tc.Description]; ok {
		ls.Skip("skipping due to known failure")
	}

	testCtx := newTestContext(mtest.Background, tc.entities)

	defer func() {
		// If anything fails while doing test cleanup, we only log the error because the actual test may have already
		// failed and that failure should be preserved.

		for _, err := range disableUntargetedFailPoints(testCtx) {
			ls.Log(err)
		}
		for _, err := range disableTargetedFailPoints(testCtx) {
			ls.Log(err)
		}
		for _, err := range entities(testCtx).close(testCtx) {
			ls.Log(err)
		}
		// Tests that started a transaction should terminate any sessions left open on the server. This is required even
		// if the test attempted to commit/abort the transaction because an abortTransaction command can fail if it's
		// sent to a mongos that isn't aware of the transaction.
		if tc.startsTransaction() && tc.killAllSessions {
			if err := terminateOpenSessions(mtest.Background); err != nil {
				ls.Logf("error terminating open transactions after failed test: %v", err)
			}
		}

		close(tc.loopDone)
	}()

	// Set up collections based on the file-level initialData field.
	for _, collData := range tc.initialData {
		if err := collData.createCollection(testCtx); err != nil {
			return fmt.Errorf("error setting up collection %q: %v", collData.namespace(), err)
		}
	}

	// Set up entities based on the file-level createEntities field. For client entities, if the test will configure
	// a fail point, set a low heartbeatFrequencyMS value into the URI options map if one is not already present.
	// This speeds up recovery time for the client if the fail point forces the server to return a state change
	// error.
	shouldSetHeartbeatFrequency := tc.setsFailPoint()
	for idx, entity := range tc.createEntities {
		for entityType, entityOptions := range entity {
			if shouldSetHeartbeatFrequency && entityType == "client" {
				if entityOptions.URIOptions == nil {
					entityOptions.URIOptions = make(bson.M)
				}
				if _, ok := entityOptions.URIOptions["heartbeatFrequencyMS"]; !ok {
					entityOptions.URIOptions["heartbeatFrequencyMS"] = lowHeartbeatFrequency
				}
			}

			if err := tc.entities.addEntity(testCtx, entityType, entityOptions); err != nil {
				return fmt.Errorf("error creating entity at index %d: %v", idx, err)
			}
		}
	}

	// Work around SERVER-39704.
	if mtest.ClusterTopologyKind() == mtest.Sharded && tc.performsDistinct() {
		if err := performDistinctWorkaround(testCtx); err != nil {
			return fmt.Errorf("error performing \"distinct\" workaround: %v", err)
		}
	}

	for idx, operation := range tc.Operations {
		if err := operation.execute(testCtx, tc.loopDone); err != nil {
			return fmt.Errorf("error running operation %q at index %d: %v", operation.Name, idx, err)
		}
	}

	for _, client := range tc.entities.clients() {
		client.stopListeningForEvents()
	}

	for idx, expectedEvents := range tc.ExpectedEvents {
		if err := verifyEvents(testCtx, expectedEvents); err != nil {
			return fmt.Errorf("events verification failed at index %d: %v", idx, err)
		}
	}

	for idx, collData := range tc.Outcome {
		if err := collData.verifyContents(testCtx); err != nil {
			return fmt.Errorf("error verifying outcome for collection %q at index %d: %v",
				collData.namespace(), idx, err)
		}
	}
	return nil
}

func disableUntargetedFailPoints(ctx context.Context) []error {
	var errs []error
	for fpName, client := range failPoints(ctx) {
		if err := disableFailPointWithClient(ctx, fpName, client); err != nil {
			errs = append(errs, fmt.Errorf("error disabling fail point %q: %v", fpName, err))
		}
	}
	return errs
}

func disableTargetedFailPoints(ctx context.Context) []error {
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
