package unified

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/logger"
)

var (
	errLogLevelRequired     = fmt.Errorf("level is required")
	errLogComponentRequired = fmt.Errorf("component is required")
	errLogDataRequired      = fmt.Errorf("data is required")
	errLogClientRequired    = fmt.Errorf("client is required")
	errLogMessagesRequired  = fmt.Errorf(" messages is required")
	errLogDocumentMismatch  = fmt.Errorf("document mismatch")
	errLogLevelMismatch     = fmt.Errorf("level mismatch")
	errLogMarshalingFailure = fmt.Errorf("marshaling failure")
	errLogMessageInvalid    = fmt.Errorf("message is invalid")
	errLogClientInvalid     = fmt.Errorf("client is invalid")
	errLogStructureInvalid  = fmt.Errorf("arguments are invalid")
	errLogClientDuplicate   = fmt.Errorf("lient already exists")
	errLogNotFound          = fmt.Errorf("not found")
)

// logMessage is a log message that is expected to be observed by the driver.
type logMessage struct {
	// LevelLiteral is the literal logging level of the expected log message. Note that this is not the same as the
	// LogLevel type in the driver's options package, which are the levels that can be configured for the driver's
	// logger. This is a required field.
	LevelLiteral logger.LevelLiteral `bson:"level"`

	// ComponentLiteral is the literal logging component of the expected log message. Note that this is not the
	// same as the Component type in the driver's logger package, which are the components that can be configured
	// for the driver's logger. This is a required field.
	ComponentLiteral logger.ComponentLiteral `bson:"component"`

	// Data is the expected data of the log message. This is a required field.
	Data bson.Raw `bson:"data"`

	// FailureIsRedacted is a boolean indicating whether or not the expected log message should be redacted. If
	// true, the expected log message should be redacted. If false, the expected log message should not be
	// redacted. This is a required field.
	FailureIsRedacted bool `bson:"failureIsRedacted"`
}

// newLogMessage will create a "logMessage" from the level and a slice of arguments.
func newLogMessage(level int, args ...interface{}) (*logMessage, error) {
	logMessage := new(logMessage)

	if len(args) > 0 {
		// actualD is the bson.D analogue of the got.args empty interface slice. For example, if got.args is
		// []interface{}{"foo", 1}, then actualD will be bson.D{{"foo", 1}}.
		actualD := bson.D{}
		for i := 0; i < len(args); i += 2 {
			// If args exceeds the length of the slice, then we have an invalid log message.
			if i+1 >= len(args) {
				return nil, fmt.Errorf("%w: %s", errLogStructureInvalid, "uneven number of arguments")
			}

			actualD = append(actualD, bson.E{Key: args[i].(string), Value: args[i+1]})
		}

		// Marshal the actualD bson.D into a bson.Raw so that we can compare it to the expectedDoc
		// bson.RawValue.
		bytes, err := bson.Marshal(actualD)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", errLogMarshalingFailure, err)
		}

		logMessage.Data = bson.Raw(bytes)
	}

	// Iterate over the literal levels until we get the highest level literal that matches the level of the log
	// message.
	for _, l := range logger.AllLevelLiterals() {
		if l.Level() == logger.Level(level) {
			logMessage.LevelLiteral = l
		}
	}

	return logMessage, nil
}

// validate will validate the expectedLogMessage and return an error if it is invalid.
func (message *logMessage) validate() error {
	if message.LevelLiteral == "" {
		return errLogLevelRequired
	}

	if message.ComponentLiteral == "" {
		return errLogComponentRequired
	}

	if message.Data == nil {
		return errLogDataRequired
	}

	return nil
}

// is will check if the "got" logActual argument matches the expectedLogMessage. Note that we do not need to
// compare the component literals, as that can be validated through the messages and arguments.
func (message logMessage) is(target *logMessage) error {
	if target == nil {
		return errLogNotFound
	}

	// The levels of the expected log message and the actual log message must match, upto logger.Level.
	if message.LevelLiteral.Level() != target.LevelLiteral.Level() {
		return fmt.Errorf("%w %v, got %v", errLogLevelMismatch, message.LevelLiteral,
			target.LevelLiteral)
	}

	// expectedDoc is the expected document that should be logged. This is the document that we will compare to the
	// document associated with logActual.
	expectedDoc := documentToRawValue(message.Data)

	// targetDoc is the actual document that was logged. This is the document that we will compare to the expected
	// document.
	targetDoc := documentToRawValue(target.Data)

	if err := verifyValuesMatch(context.Background(), expectedDoc, targetDoc, true); err != nil {
		return fmt.Errorf("%w: %v", errLogDocumentMismatch, err)
	}

	return nil
}

// clientLog is a struct representing the expected log messages for a client. This is used
// for the "expectEvents" assertion in the unified test format.
type clientLog struct {
	// Client is the name of the client to check for expected log messages. This is a required field.
	Client string `bson:"client"`

	// Messages is a slice of expected log messages. This is a required field.
	Messages []*logMessage `bson:"messages"`
}

// validate will validate the expectedLogMessasagesForClient and return an error if it is invalid.
func (log *clientLog) validate() error {
	if log.Client == "" {
		return errLogClientRequired
	}

	if log.Messages == nil || len(log.Messages) == 0 {
		return errLogMessagesRequired
	}

	for _, msg := range log.Messages {
		if err := msg.validate(); err != nil {
			return fmt.Errorf("%w: %v", errLogMessageInvalid, err)
		}
	}

	return nil
}

type clientLogs []*clientLog

// validate will validate the expectedLogMessagesForClients and return an error if it is invalid.
func (logs clientLogs) validate() error {
	// We need to keep track of the client names that we have already seen so that we can ensure that there are
	// not multiple expectedLogMessagesForClient objects for a single client entity.
	seenClientNames := make(map[string]struct{})

	for _, client := range logs {
		if err := client.validate(); err != nil {
			return fmt.Errorf("%w: %v", errLogClientInvalid, err)
		}

		if _, ok := seenClientNames[client.Client]; ok {
			return fmt.Errorf("%w: %v", errLogClientDuplicate, client.Client)
		}

		seenClientNames[client.Client] = struct{}{}
	}

	return nil
}

// logMessageWithError is the logMessage given by the TestFile with an associated error.
type logMessageWithError struct {
	*logMessage
	err error
}

// newLogMessageWithError will create a logMessageWithError from a logMessage and an error.
func newLogMessageWithError(message *logMessage, err error) *logMessageWithError {
	return &logMessageWithError{
		logMessage: message,
		err:        err,
	}
}

// clientLogWithError is the clientLog given by the TestFile where each logMessage has an associated error encountered
// by the test runner.
type clientLogWithError struct {
	client   string
	messages []*logMessageWithError
}

// newClientLogWithError will create a clientLogWithError from a clientLog. Each logMessage in the clientLog will be
// converted to a logMessageWithError with a default error indicating that the log message was not found. The error for
// a message will be updated when the test runner encounters the "actual" analogue to the "expected" log message. When
// the analogue encountered, one of two things will happen:
//
//  1. If the "actual" log matches the "expected" log, then the error will be updated to nil.
//  2. If the "actual" log does not match the "expected" log, then the error will be updated to indicate that the
//     "actual" log did not match the "expected" log and why.
//
// This is done in the event that test runner expects a log but never encounters it, propagating the "not found but
// expected" error to the user.
func newClientLogWithError(log *clientLog) *clientLogWithError {
	const messageKey = "message"

	clwe := &clientLogWithError{
		client:   log.Client,
		messages: make([]*logMessageWithError, len(log.Messages)),
	}

	for i, msg := range log.Messages {
		clwe.messages[i] = newLogMessageWithError(msg, fmt.Errorf("%w: client=%q, message=%q",
			errLogNotFound, log.Client, msg.Data.Lookup(messageKey).StringValue()))

	}

	return clwe
}

// err will return the first error found for the expected log messages.
func (clwe *clientLogWithError) validate() error {
	for _, msg := range clwe.messages {
		if msg.err != nil {
			return msg.err
		}
	}

	return nil
}

// logMessageVAlidator defines the expectation for log messages accross all clients.
type logMessageValidator struct {
	clientLogs map[string]*clientLogWithError
}

func (validator *logMessageValidator) close() {}

// addClient wil add a new client to the "logMessageValidator". By default all messages are considered "invalid" and
// "missing" until they are verified.
func (validator *logMessageValidator) addClients(clients clientLogs) {
	const messageKey = "message"

	if validator.clientLogs == nil {
		validator.clientLogs = make(map[string]*clientLogWithError)
	}

	for _, clientMessages := range clients {
		validator.clientLogs[clientMessages.Client] = newClientLogWithError(clientMessages)
	}
}

// getClient will return the "logMessageClientValidator" for the given client name. If no client exists for the given
// client name, this will return nil.
func (validator *logMessageValidator) getClient(clientName string) *clientLogWithError {
	if validator.clientLogs == nil {
		return nil
	}

	return validator.clientLogs[clientName]
}

// validate will validate all log messages receiced by all clients and return the first error encountered.
func (validator *logMessageValidator) validate() error {
	for _, client := range validator.clientLogs {
		if err := client.validate(); err != nil {
			return err
		}
	}

	return nil
}

// startLogMessageClientValidator will listen to the "logActual" channel for a given client entity, updating the
// "invalid" map to either (1) delete the "missing message" if the message was found and is valid, or (2) update the
// map to express the error that occurred while validating the message.
func startLogMessageClientValidator(entity *clientEntity, clientLogs *clientLogWithError) {
	for actual := range entity.loggerActual {
		message := clientLogs.messages[actual.position-1]
		if message == nil {
			continue
		}

		message.err = message.is(actual.message)
	}
}

// startLogMessageValidate will start one worker per client entity that will validate the log messages for that client.
func startLogMessageValidator(tcase *TestCase) *logMessageValidator {
	validator := new(logMessageValidator)
	for clientName, entity := range tcase.entities.clients() {
		validator.addClients(tcase.ExpectLogMessages)

		go startLogMessageClientValidator(entity, validator.getClient(clientName))
	}

	return validator
}
