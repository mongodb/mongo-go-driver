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
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/spectest"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var (
	logMessageValidatorTimeout = 10 * time.Millisecond
	lowHeartbeatFrequency      = 500 * time.Millisecond
)

// TestCase holds and runs a unified spec test case
type TestCase struct {
	Description       string               `bson:"description"`
	RunOnRequirements []mtest.RunOnBlock   `bson:"runOnRequirements"`
	SkipReason        *string              `bson:"skipReason"`
	Operations        []*operation         `bson:"operations"`
	ExpectedEvents    []*expectedEvents    `bson:"expectEvents"`
	ExpectLogMessages []*clientLogMessages `bson:"expectLogMessages"`
	Outcome           []*collectionData    `bson:"outcome"`

	initialData     []*collectionData
	createEntities  []map[string]*entityOptions
	killAllSessions bool
	schemaVersion   string

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

// runTestDirectory runs the files in the given directory, which must be in the unified spec format, with
// expectValidFail determining whether the tests should expect to pass or fail
func runTestDirectory(t *testing.T, directoryPath string, expectValidFail bool) {
	for _, filename := range spectest.FindJSONFilesInDir(t, directoryPath) {
		t.Run(filename, func(t *testing.T) {
			runTestFile(t, path.Join(directoryPath, filename), expectValidFail)
		})
	}
}

// runTestFile runs the tests in the given file, with expectValidFail determining whether the tests should expect to pass or fail
func runTestFile(t *testing.T, filepath string, expectValidFail bool, opts ...*Options) {
	content, err := ioutil.ReadFile(filepath)
	assert.Nil(t, err, "ReadFile error for file %q: %v", filepath, err)

	fileReqs, testCases := ParseTestFile(t, content, expectValidFail, opts...)

	mtOpts := mtest.NewOptions().
		RunOn(fileReqs...).
		CreateClient(false)
	mt := mtest.New(t, mtOpts)

	for _, testCase := range testCases {
		mtOpts := mtest.NewOptions().
			RunOn(testCase.RunOnRequirements...).
			CreateClient(false)

		mt.RunOpts(testCase.Description, mtOpts, func(mt *mtest.T) {
			spectest.CheckSkip(mt.T)

			// Skip CSOT spec tests when SKIP_CSOT_TESTS=true. In Evergreen, we
			// typically set that environment variable on Windows and macOS
			// because the CSOT spec tests are unreliable on those hosts.
			if os.Getenv("SKIP_CSOT_TESTS") == "true" && strings.Contains(filepath, "client-side-operations-timeout") {
				mt.Skip("Skipping CSOT spec test because SKIP_CSOT_TESTS=true")
			}

			defer func() {
				// catch panics from looking up elements and fail if it's unexpected
				if r := recover(); r != nil {
					if !expectValidFail {
						mt.Fatal(r)
					}
				}
			}()

			err := testCase.Run(mt)
			if expectValidFail {
				if err != nil {
					return
				}
				mt.Fatalf("expected test to error, got nil")
			}
			if err != nil {
				mt.Fatal(err)
			}
		})
	}
}

func parseTestFile(testJSON []byte, opts ...*Options) ([]mtest.RunOnBlock, []*TestCase, error) {
	var testFile TestFile
	if err := bson.UnmarshalExtJSON(testJSON, false, &testFile); err != nil {
		return nil, nil, err
	}

	op := NewOptions()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if opt.RunKillAllSessions != nil {
			op.RunKillAllSessions = opt.RunKillAllSessions
		}
	}

	for _, testCase := range testFile.TestCases {
		testCase.initialData = testFile.InitialData
		testCase.createEntities = testFile.CreateEntities
		testCase.schemaVersion = testFile.SchemaVersion
		testCase.entities = newEntityMap()
		testCase.loopDone = make(chan struct{})
		testCase.killAllSessions = *op.RunKillAllSessions
	}

	return testFile.RunOnRequirements, testFile.TestCases, nil
}

// ParseTestFile create an array of TestCases from the testJSON json blob
func ParseTestFile(t *testing.T, testJSON []byte, expectValidFail bool, opts ...*Options) ([]mtest.RunOnBlock, []*TestCase) {
	t.Helper()

	runOnRequirements, testCases, err := parseTestFile(testJSON, opts...)

	if !expectValidFail {
		assert.NoError(t, err, "error parsing test file")
	}

	return runOnRequirements, testCases
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

// skipTestError indicates that a test must be skipped because the runner cannot execute it (e.g. the test requires
// an operation or option that the driver does not support).
type skipTestError struct {
	reason string
}

// Error implements the error interface.
func (s skipTestError) Error() string {
	return fmt.Sprintf("test must be skipped: %q", s.reason)
}

func newSkipTestError(reason string) error {
	return &skipTestError{reason}
}

func isSkipTestError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "test must be skipped")
}

// Run runs the TestCase and returns an error if it fails
func (tc *TestCase) Run(ls LoggerSkipper) error {
	if tc.SkipReason != nil {
		ls.Skipf("skipping for reason: %q", *tc.SkipReason)
	}

	// Validate that we support the schema declared by the test file before attempting to use its contents.
	if err := checkSchemaVersion(tc.schemaVersion); err != nil {
		return fmt.Errorf("schema version %q not supported: %v", tc.schemaVersion, err)
	}

	testCtx := newTestContext(context.Background(), tc.entities, tc.ExpectLogMessages, tc.setsFailPoint())

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
			if err := terminateOpenSessions(context.Background()); err != nil {
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
	for idx, entity := range tc.createEntities {
		for entityType, entityOptions := range entity {
			if entityType == "client" && hasOperationalFailpoint(testCtx) {
				entityOptions.setHeartbeatFrequencyMS(lowHeartbeatFrequency)
			}

			if err := tc.entities.addEntity(testCtx, entityType, entityOptions); err != nil {
				if isSkipTestError(err) {
					ls.Skip(err)
				}

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
			if isSkipTestError(err) {
				ls.Skip(err)
			}

			return fmt.Errorf("error running operation %q at index %d: %v", operation.Name, idx, err)
		}
	}

	// Create a validator for log messages and start the workers that will
	// observe log messages as they occur operationally.
	logMessageValidator := newLogMessageValidator(tc)
	go startLogValidators(testCtx, logMessageValidator)

	for _, client := range tc.entities.clients() {
		client.stopListeningForEvents()
	}

	// One of the bulkWrite spec tests expects update and updateMany to be grouped together into a single batch,
	// but this isn't the case because of GODRIVER-1157. To work around this, we skip event verification for this test.
	// This guard should be removed when GODRIVER-1157 is done.
	if tc.Description != "BulkWrite on server that doesn't support arrayFilters with arrayFilters on second op" {
		for idx, expectedEvents := range tc.ExpectedEvents {
			if err := verifyEvents(testCtx, expectedEvents); err != nil {
				return fmt.Errorf("events verification failed at index %d: %v", idx, err)
			}
		}
	}

	for idx, collData := range tc.Outcome {
		if err := collData.verifyContents(testCtx); err != nil {
			return fmt.Errorf("error verifying outcome for collection %q at index %d: %v",
				collData.namespace(), idx, err)
		}
	}

	{
		// Create a context with a deadline to use for log message
		// validation. This will prevent any blocking from test cases
		// with N messages where only N - K (0 < K < N) messages are
		// observed.
		ctx, cancel := context.WithTimeout(testCtx, logMessageValidatorTimeout)
		defer cancel()

		// For each client, verify that all expected log messages were
		// received.
		if err := stopLogValidators(ctx, logMessageValidator); err != nil {
			return fmt.Errorf("error verifying log messages: %w", err)
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
	return client.Database("admin").RunCommand(ctx, cmd).Err()
}
