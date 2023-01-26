// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/logger"
)

// ErrLoggerVerification is use to wrap errors associated with validating the
// correctness of logs while testing operations.
var ErrLoggerVerification = fmt.Errorf("logger verification failed")

// logMessage is a log message that is expected to be observed by the driver.
type logMessage struct {
	LevelLiteral      string   `bson:"level"`
	ComponentLiteral  string   `bson:"component"`
	Data              bson.Raw `bson:"data"`
	FailureIsRedacted bool     `bson:"failureIsRedacted"`
}

// newLogMessage will create a "logMessage" from the level and a slice of
// arguments.
func newLogMessage(level int, args ...interface{}) (*logMessage, error) {
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

	if len(args) == 0 {
		return logMessage, nil
	}

	// The argument slice must have an even number of elements, otherwise it
	// would not maintain the key-value structure of the document.
	if len(args)%2 != 0 {
		return nil, fmt.Errorf("%w: invalid arguments: %v", ErrLoggerVerification, args)
	}

	// Create a new document from the arguments.
	actualD := bson.D{}
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
		return nil, fmt.Errorf("%w: failed to marshal: %v", ErrLoggerVerification, err)
	}

	logMessage.Data = bson.Raw(bytes)

	return logMessage, nil
}

// validate will validate the expectedLogMessage and return an error if it is
// invalid.
func validateLogMessage(message *logMessage) error {
	if message.LevelLiteral == "" {
		return fmt.Errorf("%w: level is required", ErrLoggerVerification)
	}

	if message.ComponentLiteral == "" {
		return fmt.Errorf("%w: component is required", ErrLoggerVerification)
	}

	if message.Data == nil {
		return fmt.Errorf("%w: data is required", ErrLoggerVerification)
	}

	return nil
}

// clientLogMessages is a struct representing the expected "LogMessages" for a
// client.
type clientLogMessages struct {
	Client      string        `bson:"client"`
	LogMessages []*logMessage `bson:"messages"`
}

// validateClientLogMessages will validate a single "clientLogMessages" object
// and return an error if it is invalid, i.e. not testable.
func validateClientLogMessages(log *clientLogMessages) error {
	if log.Client == "" {
		return fmt.Errorf("%w: client is required", ErrLoggerVerification)
	}

	if len(log.LogMessages) == 0 {
		return fmt.Errorf("%w: log messages are required", ErrLoggerVerification)
	}

	for _, message := range log.LogMessages {
		if err := validateLogMessage(message); err != nil {
			return fmt.Errorf("%w: message is invalid: %v", ErrLoggerVerification, err)
		}
	}

	return nil
}

// validateExpectLogMessages will validate a slice of "clientLogMessages"
// objects and return the first error encountered.
func validateExpectLogMessages(logs []*clientLogMessages) error {
	seenClientNames := make(map[string]struct{}) // Check for client duplication

	for _, log := range logs {
		if err := validateClientLogMessages(log); err != nil {
			return fmt.Errorf("%w: client is invalid: %v", ErrLoggerVerification, err)
		}

		if _, ok := seenClientNames[log.Client]; ok {
			return fmt.Errorf("%w: duplicate client: %v", ErrLoggerVerification, log.Client)
		}

		seenClientNames[log.Client] = struct{}{}
	}

	return nil
}

// logMessageValidator defines the expectation for log messages across all
// clients.
type logMessageValidator struct {
	testCase    *TestCase
	err         error
	done        chan struct{}
	cardinality int
}

// newLogMessageValidator will create a new "logMessageValidator" from a test
// case.
func newLogMessageValidator(testCase *TestCase) (*logMessageValidator, error) {
	if testCase == nil {
		return nil, fmt.Errorf("%w: test case is required", ErrLoggerVerification)
	}

	if testCase.entities == nil {
		return nil, fmt.Errorf("%w: entities are required", ErrLoggerVerification)
	}

	validator := &logMessageValidator{testCase: testCase}

	// Count the number of LogMessage objects on each ExpectedLogMessages.
	// This will give us the minimal number of log messages we expect to
	// receive from each client. That is, we want Σ (1 + len(messages))
	// over all clients.
	for _, clientLogMessages := range testCase.ExpectLogMessages {
		validator.cardinality += len(clientLogMessages.LogMessages)
	}

	validator.done = make(chan struct{}, validator.cardinality)

	return validator, nil
}

type actualLogQueues map[string]chan orderedLogMessage

func (validator *logMessageValidator) expected(ctx context.Context) ([]*clientLogMessages, actualLogQueues) {
	clients := entities(ctx).clients()

	expected := make([]*clientLogMessages, 0, len(validator.testCase.ExpectLogMessages))
	actual := make(actualLogQueues, len(clients))

	for _, clientLogMessages := range validator.testCase.ExpectLogMessages {
		clientName := clientLogMessages.Client

		clientEntity, ok := clients[clientName]
		if !ok {
			continue // If there is no entity for the client, skip it.
		}

		expected = append(expected, clientLogMessages)
		actual[clientName] = clientEntity.logQueue
	}

	return expected, actual
}

// stopLogMessageVerificationWorkers will gracefully validate all log messages
// received by all clients and return the first error encountered.
//
// Unfortunately, there is currently no way to communicate to a client entity
// constructor how many messages are expected to be received. Because of this,
// the LogSink assigned to each client has no way of knowing when to close the
// log queue. Therefore, it is the responsbility of this function to ensure that
// all log messages are received and validated: N errors for N log messages.
func stopLogMessageVerificationWorkers(ctx context.Context, validator *logMessageValidator) error {
	var ctxDeadlineExceededError error
	for i := 0; i < validator.cardinality; i++ {
		select {
		case <-validator.done:
		case <-ctx.Done():
			ctxDeadlineExceededError = fmt.Errorf("%w: context error: %v",
				ErrLoggerVerification, ctx.Err())
		}
	}

	// First check to see if we have any errors from validating log
	// messages.
	if validator.err != nil {
		return fmt.Errorf("%w:  %v", ErrLoggerVerification, validator.err)
	}

	// If we have a context deadline exceeded error, return it.
	if ctxDeadlineExceededError != nil {
		return fmt.Errorf("%w:  %v", ErrLoggerVerification, ctxDeadlineExceededError)
	}

	return nil
}

// verifyLogMessagesMatch will verify that the actual log message match the
// expected log messages.
func verifyLogMessagesMatch(ctx context.Context, exp, act *logMessage) error {
	if act == nil && exp == nil {
		return nil
	}

	if act == nil || exp == nil {
		return fmt.Errorf("document mismatch")
	}

	levelExp := logger.ParseLevel(exp.LevelLiteral)
	levelAct := logger.ParseLevel(act.LevelLiteral)

	// The levels of the expected log message and the actual log message
	// must match, upto logger.Level.
	if levelExp != levelAct {
		return fmt.Errorf("level mismatch: want %v, got %v", levelExp, levelAct)
	}

	rawExp := documentToRawValue(exp.Data)
	rawAct := documentToRawValue(act.Data)

	// Top level data does not have to be 1-1 with the expectation, there
	// are a number of unrequired fields that may not be present on the
	// expected document.
	if err := verifyValuesMatch(ctx, rawExp, rawAct, true); err != nil {
		return fmt.Errorf("data mismatch: %v", err)
	}

	return nil
}

// startLogMessageVerificationWorkers will start a goroutine for each client's
// expected log messages, listening to the channel of actual log messages and
// comparing them to the expected log messages.
//
// When validating the logs, it could be the case that there are more "actual"
// logs being queued than "expected" logs. For example, the Go Driver closes
// sessions when the client is disconnected, which triggers three extra checkout
// logs. In this case, a unified test will result in |actual| = |expected| + 3.
// This verification function will ignore the extra logs and assume that the
// expected messages are a "ordered subset" of the actual messages.
func startLogMessageVerificationWorkers(ctx context.Context, validator *logMessageValidator) {
	expected, actual := validator.expected(ctx)

	for _, expected := range expected {
		if expected == nil {
			continue
		}

		go func(expected *clientLogMessages) {
			// In good faith, if the message is not valid then we increment the offset in
			// case the next message is the one that we expect.
			offset := 1

			for act := range actual[expected.Client] {
				position := act.order - offset
				exp := expected.LogMessages[position]

				err := verifyLogMessagesMatch(ctx, exp, act.logMessage)
				if err != nil {
					// Only return the first error unless a more accurate error
					// occurs.
					if validator.err == nil {
						validator.err = err
					}

					// Attempt to capture a more accurate error message by
					// comparing the underlying messages and log levels.
					expectedMsg := exp.Data.Lookup("message").StringValue()
					actualMsg := act.logMessage.Data.Lookup("message").StringValue()

					expectedLevel := exp.LevelLiteral
					actualLevel := act.logMessage.LevelLiteral

					if expectedLevel == actualLevel && expectedMsg == actualMsg {
						validator.err = fmt.Errorf("error for message %q with level %q: %w",
							expectedMsg, expectedLevel, err)
					}

					offset++

					continue
				}

				// If the message is valid, we reset the err.
				validator.err = nil
				validator.done <- struct{}{}
			}

		}(expected)
	}
}
