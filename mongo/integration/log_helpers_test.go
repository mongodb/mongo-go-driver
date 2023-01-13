// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/internal/logger"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

type testLogSink struct {
	logs       chan func() (int, string, []interface{})
	bufferSize int
	logsCount  int
	errsCh     chan error
}

type logValidator func(order int, lvl int, msg string, kv ...interface{}) error

func newTestLogSink(ctx context.Context, mt *mtest.T, bufferSize int, validator logValidator) *testLogSink {
	mt.Helper()

	sink := &testLogSink{
		logs:       make(chan func() (int, string, []interface{}), bufferSize),
		errsCh:     make(chan error, bufferSize),
		bufferSize: bufferSize,
	}

	go func() {
		order := 0
		for log := range sink.logs {
			select {
			case <-ctx.Done():
				sink.errsCh <- ctx.Err()

				return
			default:
			}

			level, msg, args := log()
			if err := validator(order, level, msg, args...); err != nil {
				sink.errsCh <- fmt.Errorf("invalid log at position %d, level %d, and msg %q: %v", order,
					level, msg, err)
			}

			order++
		}

		close(sink.errsCh)
	}()

	return sink
}

func (sink *testLogSink) Info(level int, msg string, keysAndValues ...interface{}) {
	sink.logs <- func() (int, string, []interface{}) {
		return level, msg, keysAndValues
	}

	if sink.logsCount++; sink.logsCount == sink.bufferSize {
		close(sink.logs)
	}
}

func (sink *testLogSink) Error(err error, msg string, keysAndValues ...interface{}) {
	sink.Info(int(logger.LevelInfo), msg, keysAndValues)
}

func (sink *testLogSink) errs() <-chan error {
	return sink.errsCh
}

func findLogValue(mt *mtest.T, key string, values ...interface{}) interface{} {
	mt.Helper()

	for i := 0; i < len(values); i += 2 {
		if values[i] == key {
			return values[i+1]
		}
	}

	return nil
}

type logTruncCaseValidator func(values ...interface{}) error

func newLogTruncCaseValidator(mt *mtest.T, commandName string, cond func(string) error) logTruncCaseValidator {
	mt.Helper()

	return func(values ...interface{}) error {
		cmd := findLogValue(mt, commandName, values...)
		if cmd == nil {
			return fmt.Errorf("%q not found in keys and values", commandName)
		}

		cmdStr, ok := cmd.(string)

		if !ok {
			return fmt.Errorf("command is not a string")
		}

		if err := cond(cmdStr); err != nil {
			return err
		}

		return nil
	}
}
