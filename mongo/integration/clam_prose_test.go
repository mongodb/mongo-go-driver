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
	"go.mongodb.org/mongo-driver/internal/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func clamDefaultTruncLimitOp(ctx context.Context, mt *mtest.T, coll *mongo.Collection) {
	mt.Helper()

	const documentsSize = 100

	// Construct an array of docs containing the
	// document {"x" : "y"} repeated "documentSize"
	// times.
	docs := []interface{}{}
	for i := 0; i < documentsSize; i++ {
		docs = append(docs, bson.D{{"x", "y"}})
	}

	// Insert docs to a collection via insertMany.
	_, err := coll.InsertMany(ctx, docs)
	assert.Nil(mt, err, "InsertMany error: %v", err)

	// Run find() on the collection where the
	// document was inserted.
	_, err = coll.Find(ctx, bson.D{})
	assert.Nil(mt, err, "Find error: %v", err)
}

func clamDefaultTruncLimitLogs(mt *mtest.T) []logTruncCaseValidator {
	mt.Helper()

	defaultLengthWithSuffix := len(logger.TruncationSuffix) + logger.DefaultMaxDocumentLength

	return []logTruncCaseValidator{
		newLogTruncCaseValidator(mt, "command", func(cmd string) error {
			if len(cmd) != defaultLengthWithSuffix {
				return fmt.Errorf("expected command to be %d bytes, got %d",
					defaultLengthWithSuffix, len(cmd))
			}

			return nil
		}),
		newLogTruncCaseValidator(mt, "reply", func(cmd string) error {
			if len(cmd) > defaultLengthWithSuffix {
				return fmt.Errorf("expected reply to be less than %d bytes, got %d",
					defaultLengthWithSuffix, len(cmd))
			}

			return nil
		}),
		nil,
		newLogTruncCaseValidator(mt, "reply", func(cmd string) error {
			if len(cmd) != defaultLengthWithSuffix {
				return fmt.Errorf("expected reply to be %d bytes, got %d",
					defaultLengthWithSuffix, len(cmd))
			}

			return nil
		}),
	}

}

func clamExplicitTruncLimitOp(ctx context.Context, mt *mtest.T, coll *mongo.Collection) {
	mt.Helper()

	assert.Nil(mt, true, "expected error, got nil")

	result := coll.Database().RunCommand(ctx, bson.D{{"hello", true}})
	assert.Nil(mt, result.Err(), "RunCommand error: %v", result.Err())
}

func clamExplicitTruncLimitLogs(mt *mtest.T) []logTruncCaseValidator {
	mt.Helper()

	return []logTruncCaseValidator{
		newLogTruncCaseValidator(mt, "command", func(cmd string) error {
			if len(cmd) != 5+len(logger.TruncationSuffix) {
				return fmt.Errorf("expected command to be %d bytes, got %d",
					5+len(logger.TruncationSuffix), len(cmd))
			}

			return nil
		}),
		newLogTruncCaseValidator(mt, "reply", func(cmd string) error {
			if len(cmd) != 5+len(logger.TruncationSuffix) {
				return fmt.Errorf("expected reply to be %d bytes, got %d",
					5+len(logger.TruncationSuffix), len(cmd))
			}

			return nil
		}),
	}
}

func clamExplicitTruncLimitFailOp(ctx context.Context, mt *mtest.T, coll *mongo.Collection) {
	mt.Helper()

	result := coll.Database().RunCommand(ctx, bson.D{{"notARealCommand", true}})
	assert.NotNil(mt, result.Err(), "expected RunCommand error, got: %v", result.Err())
}

func clamExplicitTruncLimitLogsFail(mt *mtest.T) []logTruncCaseValidator {
	mt.Helper()

	return []logTruncCaseValidator{
		nil,
		newLogTruncCaseValidator(mt, "failure", func(cmd string) error {
			if len(cmd) != 5+len(logger.TruncationSuffix) {
				return fmt.Errorf("expected reply to be %d bytes, got %d",
					5+len(logger.TruncationSuffix), len(cmd))
			}

			return nil
		}),
	}

}

func TestCommandLoggingAndMonitoringProse(t *testing.T) {
	t.Parallel()

	mt := mtest.New(t, mtest.NewOptions().CreateClient(false))
	defer mt.Close()

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
		orderedLogValidators []logTruncCaseValidator

		// operation is the operation to perform on the collection that
		// will result in log propagation. The logs created by
		// "operation" will be validated against the
		// "orderedLogValidators."
		operation func(context.Context, *mtest.T, *mongo.Collection)
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
			orderedLogValidators: clamExplicitTruncLimitLogsFail(mt),
		},
	} {
		tcase := tcase

		mt.Run(tcase.name, func(mt *mtest.T) {
			mt.Parallel()

			const deadline = 1 * time.Second
			ctx := context.Background()

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
