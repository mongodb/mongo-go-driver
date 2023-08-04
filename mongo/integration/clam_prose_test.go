// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/integtest"
	"go.mongodb.org/mongo-driver/internal/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrInvalidTruncation = fmt.Errorf("invalid truncation")

func clamTruncErr(mt *mtest.T, op string, want, got int) error {
	mt.Helper()

	return fmt.Errorf("%w: expected length %s %d, got %d", ErrInvalidTruncation, op, want, got)
}

func clamDefaultTruncLimitOp(ctx context.Context, mt *mtest.T, coll *mongo.Collection) {
	mt.Helper()

	const documentsSize = 100

	// Construct an array of docs containing the document {"x" : "y"}
	// repeated "documentSize" times.
	docs := []interface{}{}
	for i := 0; i < documentsSize; i++ {
		docs = append(docs, bson.D{{"x", "y"}})
	}

	// Insert docs to a the collection.
	_, err := coll.InsertMany(ctx, docs)
	assert.Nil(mt, err, "InsertMany error: %v", err)

	// Run find() on the collection.
	_, err = coll.Find(ctx, bson.D{})
	assert.Nil(mt, err, "Find error: %v", err)
}

func clamDefaultTruncLimitLogs(mt *mtest.T) []truncValidator {
	mt.Helper()

	const cmd = "command"
	const rpl = "reply"

	expTruncLen := len(logger.TruncationSuffix) + logger.DefaultMaxDocumentLength
	validators := make([]truncValidator, 4)

	// Insert started.
	validators[0] = newTruncValidator(mt, cmd, func(cmd string) error {
		if len(cmd) != expTruncLen {
			return clamTruncErr(mt, "=", expTruncLen, len(cmd))
		}

		return nil
	})

	// Insert succeeded.
	validators[1] = newTruncValidator(mt, rpl, func(cmd string) error {
		if len(cmd) > expTruncLen {
			return clamTruncErr(mt, "<=", expTruncLen, len(cmd))
		}

		return nil
	})

	// Find started, nothing to validate.
	validators[2] = nil

	// Find succeeded.
	validators[3] = newTruncValidator(mt, rpl, func(cmd string) error {
		if len(cmd) != expTruncLen {
			return clamTruncErr(mt, "=", expTruncLen, len(cmd))
		}

		return nil
	})

	return validators
}

func clamExplicitTruncLimitOp(ctx context.Context, mt *mtest.T, coll *mongo.Collection) {
	mt.Helper()

	result := coll.Database().RunCommand(ctx, bson.D{{"hello", true}})
	assert.Nil(mt, result.Err(), "RunCommand error: %v", result.Err())
}

func clamExplicitTruncLimitLogs(mt *mtest.T) []truncValidator {
	mt.Helper()

	const cmd = "command"
	const rpl = "reply"

	expTruncLen := len(logger.TruncationSuffix) + 5
	validators := make([]truncValidator, 2)

	// Hello started.
	validators[0] = newTruncValidator(mt, cmd, func(cmd string) error {
		if len(cmd) != expTruncLen {
			return clamTruncErr(mt, "=", expTruncLen, len(cmd))
		}

		return nil
	})

	// Hello succeeded.
	validators[1] = newTruncValidator(mt, rpl, func(cmd string) error {
		if len(cmd) != expTruncLen {
			return clamTruncErr(mt, "=", expTruncLen, len(cmd))
		}

		return nil
	})

	return validators
}

func clamExplicitTruncLimitFailOp(ctx context.Context, mt *mtest.T, coll *mongo.Collection) {
	mt.Helper()

	result := coll.Database().RunCommand(ctx, bson.D{{"notARealCommand", true}})
	assert.NotNil(mt, result.Err(), "expected RunCommand error, got: %v", result.Err())
}

func clamExplicitTruncLimitFailLogs(mt *mtest.T) []truncValidator {
	mt.Helper()

	const fail = "failure"

	expTruncLen := len(logger.TruncationSuffix) + 5
	validators := make([]truncValidator, 2)

	// Hello started, nothing to validate.
	validators[0] = nil

	// Hello failed.
	validators[1] = newTruncValidator(mt, fail, func(cmd string) error {
		if len(cmd) != expTruncLen {
			return clamTruncErr(mt, "=", expTruncLen, len(cmd))
		}

		return nil
	})

	return validators
}

// clamMultiByteTrunc runs an operation to insert a very large document with the
// multi-byte character "界" repeated a large number of times. This repetition
// is done to categorically ensure that the truncation point is made somewhere
// within the multi-byte character. For example a typical insertion reply may
// look something like this:
//
// {"insert": "setuptest","ordered": true,"lsid": {"id": ...
//
// We have no control over how the "header" portion of this reply is formatted.
// Over time the server might support newer fields or change the formatting of
// existing fields. This means that the truncation point could be anywhere in
// the "header" portion of the reply. A large document lowers the likelihood of
// the truncation point being in the "header" portion of the reply.
func clamMultiByteTrunc(ctx context.Context, mt *mtest.T, coll *mongo.Collection) {
	mt.Helper()

	const multiByteCharStrLen = 50_000
	const strToRepeat = "界"

	// Repeat the string "strToRepeat" "multiByteCharStrLen" times.
	multiByteCharStr := ""
	for i := 0; i < multiByteCharStrLen; i++ {
		multiByteCharStr += strToRepeat
	}

	_, err := coll.InsertOne(ctx, bson.D{{"x", multiByteCharStr}})
	assert.Nil(mt, err, "InsertOne error: %v", err)
}

func clamMultiByteTruncLogs(mt *mtest.T) []truncValidator {
	mt.Helper()

	const cmd = "command"
	const strToRepeat = "界"

	validators := make([]truncValidator, 2)

	// Insert started.
	validators[0] = newTruncValidator(mt, cmd, func(cmd string) error {

		// Remove the suffix from the command string.
		cmd = cmd[:len(cmd)-len(logger.TruncationSuffix)]

		// Get the last 3 bytes of the command string.
		last3Bytes := cmd[len(cmd)-3:]

		// Make sure the last 3 bytes are the multi-byte character.
		if last3Bytes != strToRepeat {
			return fmt.Errorf("expected last 3 bytes to be %q, got %q", strToRepeat, last3Bytes)
		}

		return nil
	})

	return validators
}

func TestCommandLoggingAndMonitoringProse(t *testing.T) {
	t.Parallel()

	mt := mtest.New(t, mtest.NewOptions().CreateClient(false))

	for _, tcase := range []struct {
		// name is the name of the test case
		name string

		// collectionName is the name to assign the collection for
		// processing the operations. This should be unique across test
		// cases.
		collectionName string

		// maxDocumentLength is the maximum document length for a
		// command message.
		maxDocumentLength uint

		// orderedLogValidators is a slice of log validators that should
		// be 1-1 with the actual logs that are propagated by the
		// LogSink. The order here matters, the first log will be
		// validated by the 0th validator, the second log will be
		// validated by the 1st validator, etc.
		orderedLogValidators []truncValidator

		// operation is the operation to perform on the collection that
		// will result in log propagation. The logs created by
		// "operation" will be validated against the
		// "orderedLogValidators."
		operation func(context.Context, *mtest.T, *mongo.Collection)

		// Setup is a function that will be run before the test case.
		// Operations performed in this function will not be logged.
		setup func(context.Context, *mtest.T, *mongo.Collection)
	}{
		{
			name:                 "1 Default truncation limit",
			collectionName:       "46a624c57c72463d90f88a733e7b28b4",
			operation:            clamDefaultTruncLimitOp,
			orderedLogValidators: clamDefaultTruncLimitLogs(mt),
		},
		{
			name:                 "2 Explicitly configured truncation limit",
			collectionName:       "540baa64dc854ca2a639627e2f0918df",
			maxDocumentLength:    5,
			operation:            clamExplicitTruncLimitOp,
			orderedLogValidators: clamExplicitTruncLimitLogs(mt),
		},
		{
			name:                 "2 Explicitly configured truncation limit for failures",
			collectionName:       "aff43dfcaa1a4014b58aaa9606f5bd44",
			maxDocumentLength:    5,
			operation:            clamExplicitTruncLimitFailOp,
			orderedLogValidators: clamExplicitTruncLimitFailLogs(mt),
		},

		// The third test case is to ensure that a truncation point made
		// within a multi-byte character is handled correctly. The
		// chosen multi-byte character for this test is "界" (U+754C).
		// This character is repeated a large number of times (50,000).
		// We need to run this test 3 times to ensure that the
		// truncation occurs at a middle point within the multi-byte
		// character at least once (at most twice).
		{
			name:                 "3.1 Truncation with multi-byte codepoints",
			collectionName:       "5ed6d1b7-e358-438a-b067-e1d1dd10fee1",
			maxDocumentLength:    20_000,
			operation:            clamMultiByteTrunc,
			orderedLogValidators: clamMultiByteTruncLogs(mt),
		},
		{
			name:                 "3.2 Truncation with multi-byte codepoints",
			collectionName:       "5ed6d1b7-e358-438a-b067-e1d1dd10fee1",
			maxDocumentLength:    20_001,
			operation:            clamMultiByteTrunc,
			orderedLogValidators: clamMultiByteTruncLogs(mt),
		},
		{
			name:                 "3.3 Truncation with multi-byte codepoints",
			collectionName:       "5ed6d1b7-e358-438a-b067-e1d1dd10fee1",
			maxDocumentLength:    20_002,
			operation:            clamMultiByteTrunc,
			orderedLogValidators: clamMultiByteTruncLogs(mt),
		},
	} {
		tcase := tcase

		mt.Run(tcase.name, func(mt *mtest.T) {
			mt.Parallel()

			const deadline = 10 * time.Second
			ctx := context.Background()

			// Before the test case, we need to see if there is a
			// setup function to run.
			if tcase.setup != nil {
				clientOpts := options.Client().ApplyURI(mtest.ClusterURI())

				// Create a context with a deadline so that the
				// test setup doesn't hang forever.
				ctx, cancel := context.WithTimeout(ctx, deadline)
				defer cancel()

				integtest.AddTestServerAPIVersion(clientOpts)

				client, err := mongo.Connect(ctx, clientOpts)
				assert.Nil(mt, err, "Connect error in setup: %v", err)

				coll := mt.CreateCollection(mtest.Collection{
					Name:   tcase.collectionName,
					Client: client,
				}, false)

				tcase.setup(ctx, mt, coll)
			}

			// If there is no operation, then we don't need to run
			// the test case.
			if tcase.operation == nil {
				return
			}

			// If there are no log validators, then we should error.
			if len(tcase.orderedLogValidators) == 0 {
				mt.Fatalf("no log validators provided")
			}

			sinkCtx, sinkCancel := context.WithDeadline(ctx, time.Now().Add(deadline))
			defer sinkCancel()

			validator := func(order int, level int, msg string, keysAndValues ...interface{}) error {
				// If the order exceeds the length of the
				// "orderedCaseValidators," then throw an error.
				if order >= len(tcase.orderedLogValidators) {
					return fmt.Errorf("not enough expected cases to validate")
				}

				caseValidator := tcase.orderedLogValidators[order]
				if caseValidator == nil {
					return nil
				}

				return tcase.orderedLogValidators[order](keysAndValues...)
			}

			sink := newTestLogSink(sinkCtx, mt, len(tcase.orderedLogValidators), validator)

			// Configure logging with a minimum severity level of
			// "debug" for the "command" component without
			// explicitly configuring the max document length.
			loggerOpts := options.Logger().SetSink(sink).
				SetComponentLevel(options.LogComponentCommand, options.LogLevelDebug)

			// If the test case requires a maximum document length,
			// then configure it.
			if mdl := tcase.maxDocumentLength; mdl != 0 {
				loggerOpts.SetMaxDocumentLength(mdl)
			}

			clientOpts := options.Client().SetLoggerOptions(loggerOpts).ApplyURI(mtest.ClusterURI())

			integtest.AddTestServerAPIVersion(clientOpts)

			client, err := mongo.Connect(context.Background(), clientOpts)
			assert.Nil(mt, err, "Connect error: %v", err)

			coll := mt.CreateCollection(mtest.Collection{
				Name:   tcase.collectionName,
				Client: client,
			}, false)

			tcase.operation(ctx, mt, coll)

			// Verify the logs.
			if err := <-sink.errs(); err != nil {
				mt.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
