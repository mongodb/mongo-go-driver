// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"context"
	"fmt"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/logger"
)

// errLoggerVerification is use to wrap errors associated with validating the
// correctness of logs while testing operations.
var errLoggerVerification = fmt.Errorf("logger verification failed")

// logMessage is a log message that is expected to be observed by the driver.
type logMessage struct {
	LevelLiteral      string   `bson:"level"`
	ComponentLiteral  string   `bson:"component"`
	Data              bson.Raw `bson:"data"`
	FailureIsRedacted bool     `bson:"failureIsRedacted"`
}

// newLogMessage will create a "logMessage" from the level and a slice of
// arguments.
func newLogMessage(level int, msg string, args ...interface{}) (*logMessage, error) {
	logMessage := new(logMessage)

	// Iterate over the literal levels until we get the first
	// "LevelLiteral" that matches the level of the "LogMessage". It doesn't
	// matter which literal is chose so long as the mapping results in the
	// correct level.
	for literal, logLevel := range logger.LevelLiteralMap {
		if level == int(logLevel) {
			logMessage.LevelLiteral = literal

			break
		}
	}

	// The argument slice must have an even number of elements, otherwise it
	// would not maintain the key-value structure of the document.
	if len(args)%2 != 0 {
		return nil, fmt.Errorf("%w: invalid arguments: %v", errLoggerVerification, args)
	}

	// Create a new document from the arguments.
	actualD := bson.D{{"message", msg}}
	for i := 0; i < len(args); i += 2 {
		actualD = append(actualD, bson.E{
			Key:   args[i].(string),
			Value: args[i+1],
		})
	}

	// Marshal the document into a raw value and assign it to the
	// logMessage.
	bytes, err := bson.Marshal(actualD)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to marshal: %v", errLoggerVerification, err)
	}

	logMessage.Data = bson.Raw(bytes)

	return logMessage, nil
}

// clientLogMessages is a struct representing the expected "LogMessages" for a
// client.
type clientLogMessages struct {
	Client         string        `bson:"client"`
	IgnoreMessages []*logMessage `bson:"ignoreMessages"`
	LogMessages    []*logMessage `bson:"messages"`
}

// logMessageValidator defines the expectation for log messages across all
// clients.
type logMessageValidator struct {
	testCase   *TestCase
	clientErrs map[string]chan error
}

// newLogMessageValidator will create a new logMessageValidator.
func newLogMessageValidator(testCase *TestCase) *logMessageValidator {
	validator := &logMessageValidator{testCase: testCase}
	validator.clientErrs = make(map[string]chan error)

	// Make the error channels for the clients.
	for _, exp := range testCase.ExpectLogMessages {
		validator.clientErrs[exp.Client] = make(chan error)
	}

	return validator
}

func logQueue(ctx context.Context, exp *clientLogMessages) <-chan orderedLogMessage {
	clients := entities(ctx).clients()

	clientEntity, ok := clients[exp.Client]
	if !ok {
		return nil
	}

	return clientEntity.logQueue
}

// verifyLogMatch will verify that the actual log match the expected log.
func verifyLogMatch(ctx context.Context, exp, act *logMessage) error {
	if act == nil && exp == nil {
		return nil
	}

	if act == nil || exp == nil {
		return fmt.Errorf("%w: document mismatch", errLoggerVerification)
	}

	levelExp := logger.ParseLevel(exp.LevelLiteral)
	levelAct := logger.ParseLevel(act.LevelLiteral)

	// The levels of the expected log message and the actual log message
	// must match, upto logger.Level.
	if levelExp != levelAct {
		return fmt.Errorf("%w: level mismatch: want %v, got %v",
			errLoggerVerification, levelExp, levelAct)
	}

	rawExp := documentToRawValue(exp.Data)
	rawAct := documentToRawValue(act.Data)

	// Top level data does not have to be 1-1 with the expectation, there
	// are a number of unrequired fields that may not be present on the
	// expected document.
	if err := verifyValuesMatch(ctx, rawExp, rawAct, true); err != nil {
		return fmt.Errorf("%w: document length mismatch: %v", errLoggerVerification, err)
	}

	return nil
}

// isUnorderedLog will return true if the log is/should be unordered in the Go
// Driver.
func isUnorderedLog(log *logMessage) bool {
	msg, err := log.Data.LookupErr(logger.KeyMessage)
	if err != nil {
		return false
	}

	msgStr := msg.StringValue()

	// There is a race condition in the connection pool's workflow where it
	// is non-deterministic whether the connection pool will fail a checkout
	// or close a connection first. Because of this, either log may be
	// received in any order. To account for this behavior, we considered
	// both logs to be "unordered".
	//
	// The connection pool must clear before the connection is closed.
	// However, either of these conditions are valid:
	//
	//   1. connection checkout failed > connection pool cleared
	//   2. connection pool cleared > connection checkout failed
	//
	// Therefore, the ConnectionPoolCleared literal is added to the
	// unordered list. The check for cleared > closed is made in the
	// matching logic.
	return msgStr == logger.ConnectionCheckoutFailed ||
		msgStr == logger.ConnectionClosed ||
		msgStr == logger.ConnectionPoolCleared
}

type logQueues struct {
	expected  *clientLogMessages
	ordered   <-chan *logMessage
	unordered <-chan *logMessage
}

// partitionLogQueue will partition the expected logs into "unordered" and
// "ordered" log channels.
func partitionLogQueue(ctx context.Context, exp *clientLogMessages) logQueues {
	orderedLogCh := make(chan *logMessage, len(exp.LogMessages))
	unorderedLogCh := make(chan *logMessage, len(exp.LogMessages))

	// Get the unordered indices from the expected log messages.
	unorderedIndices := make(map[int]struct{})
	for i, log := range exp.LogMessages {
		if isUnorderedLog(log) {
			unorderedIndices[i] = struct{}{}
		}
	}

	go func() {
		defer close(orderedLogCh)
		defer close(unorderedLogCh)

		for actual := range logQueue(ctx, exp) {
			msg := actual.logMessage
			if _, ok := unorderedIndices[actual.order-2]; ok {
				unorderedLogCh <- msg
			} else {
				orderedLogCh <- msg
			}
		}
	}()

	return logQueues{
		expected:  exp,
		ordered:   orderedLogCh,
		unordered: unorderedLogCh,
	}
}

func matchOrderedLogs(ctx context.Context, logs logQueues) <-chan error {
	// Remove all of the unordered log messages from the expected.
	expLogMessages := make([]*logMessage, 0, len(logs.expected.LogMessages))
	for _, log := range logs.expected.LogMessages {
		if !isUnorderedLog(log) {
			expLogMessages = append(expLogMessages, log)
		}
	}

	errs := make(chan error, 1)

	go func() {
		defer close(errs)

		for actual := range logs.ordered {
			expected := expLogMessages[0]
			if expected == nil {
				continue
			}

			err := verifyLogMatch(ctx, expected, actual)
			if err != nil {
				errs <- err
			}

			// Remove the first element from the expected log.
			expLogMessages = expLogMessages[1:]
		}
	}()

	return errs
}

func matchUnorderedLogs(ctx context.Context, logs logQueues) <-chan error {
	unordered := make(map[*logMessage]struct{}, len(logs.expected.LogMessages))

	for _, log := range logs.expected.LogMessages {
		if isUnorderedLog(log) {
			unordered[log] = struct{}{}
		}
	}

	errs := make(chan error, 1)

	go func() {
		defer close(errs)

		// Record the message literals as they occur.
		actualMessageSet := map[string]bool{}

		for actual := range logs.unordered {
			msg, err := actual.Data.LookupErr(logger.KeyMessage)
			if err != nil {
				errs <- fmt.Errorf("could not lookup message from unordered log: %w", err)

				break
			}

			msgStr := msg.StringValue()
			if msgStr == logger.ConnectionPoolCleared && actualMessageSet[logger.ConnectionClosed] {
				errs <- fmt.Errorf("connection has been closed before the pool could clear")
			}

			// Iterate over the unordered log messages and verify
			// that at least one of them matches the actual log
			// message.
			for expected := range unordered {
				err = verifyLogMatch(ctx, expected, actual)
				if err == nil {
					// Remove the matched unordered log
					// message from the unordered map.
					delete(unordered, expected)

					break
				}
			}

			// If there was no match, return an error.
			if err != nil {
				errs <- err
			}

			actualMessageSet[msgStr] = true
		}
	}()

	return errs
}

// startLogValidators will start a goroutine for each client's expected log
// messages, listening to the channel of actual log messages and comparing them
// to the expected log messages.
func startLogValidators(ctx context.Context, validator *logMessageValidator) {
	for _, expected := range validator.testCase.ExpectLogMessages {
		logs := partitionLogQueue(ctx, expected)

		wg := &sync.WaitGroup{}
		wg.Add(2)

		go func(expected *clientLogMessages) {
			defer wg.Done()

			errCh := matchOrderedLogs(ctx, logs)
			if errCh == nil {
				return
			}

			if errs := <-errCh; errs != nil {
				validator.clientErrs[expected.Client] <- errs
			}
		}(expected)

		go func(expected *clientLogMessages) {
			defer wg.Done()

			errCh := matchUnorderedLogs(ctx, logs)
			if errCh == nil {
				return
			}

			if errs := <-errCh; errs != nil {
				validator.clientErrs[expected.Client] <- errs
			}
		}(expected)

		go func(expected *clientLogMessages) {
			wg.Wait()

			close(validator.clientErrs[expected.Client])
		}(expected)
	}
}

func stopLogValidatorsErr(clientName string, err error) error {
	return fmt.Errorf("%w: %s: %v", errLoggerVerification, clientName, err)
}

// stopLogValidators will gracefully validate all log messages received by all
// clients and return the first error encountered.
func stopLogValidators(ctx context.Context, validator *logMessageValidator) error {
	for clientName, errChan := range validator.clientErrs {
		select {
		case err := <-errChan:
			if err != nil {
				return stopLogValidatorsErr(clientName, err)
			}
		case <-ctx.Done():
			return stopLogValidatorsErr(clientName, ctx.Err())
		}
	}

	return nil
}
