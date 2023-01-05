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
	errLogClientNotFound    = fmt.Errorf("client not found")
	errTestCaseRequired     = fmt.Errorf("test case is required")
	errEntitiesRequired     = fmt.Errorf("entities is required")
	errLogContextCanceled   = fmt.Errorf("context cancelled before all log messages were verified")
)

// logMessage is a log message that is expected to be observed by the driver.
type logMessage struct {
	LevelLiteral      logger.LevelLiteral     `bson:"level"`
	ComponentLiteral  logger.ComponentLiteral `bson:"component"`
	Data              bson.Raw                `bson:"data"`
	FailureIsRedacted bool                    `bson:"failureIsRedacted"`
}

// newLogMessage will create a "logMessage" from the level and a slice of arguments.
func newLogMessage(level int, args ...interface{}) (*logMessage, error) {
	logMessage := new(logMessage)

	// Iterate over the literal levels until we get the highest "LevelLiteral" that matches the level of the
	// "LogMessage".
	for _, l := range logger.AllLevelLiterals() {
		if l.Level() == logger.Level(level) {
			logMessage.LevelLiteral = l
		}
	}

	if len(args) == 0 {
		return logMessage, nil
	}

	// The argument slice must have an even number of elements, otherwise it would not maintain the key-value
	// structure of the document.
	if len(args)%2 != 0 {
		return nil, fmt.Errorf("%w: %v", errLogStructureInvalid, args)
	}

	// Create a new document from the arguments.
	actualD := bson.D{}
	for i := 0; i < len(args); i += 2 {
		actualD = append(actualD, bson.E{Key: args[i].(string), Value: args[i+1]})
	}

	// Marshal the document into a raw value and assign it to the logMessage.
	bytes, err := bson.Marshal(actualD)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errLogMarshalingFailure, err)
	}

	logMessage.Data = bson.Raw(bytes)

	return logMessage, nil
}

// validate will validate the expectedLogMessage and return an error if it is invalid.
func validateLogMessage(_ context.Context, message *logMessage) error {
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

// verifyLogMessagesMatch will verify that the actual log messages match the expected log messages.
func verifyLogMessagesMatch(ctx context.Context, expected, actual *logMessage) error {
	const commandKey = "command"

	if actual == nil && expected == nil {
		return nil
	}

	if actual == nil || expected == nil {
		return errLogDocumentMismatch
	}

	// The levels of the expected log message and the actual log message must match, upto logger.Level.
	if expected.LevelLiteral.Level() != actual.LevelLiteral.Level() {
		return fmt.Errorf("%w: want %v, got %v", errLogLevelMismatch, expected.LevelLiteral,
			actual.LevelLiteral)
	}

	rawExp := documentToRawValue(expected.Data)
	rawAct := documentToRawValue(actual.Data)

	// Top level data does not have to be 1-1 with the expectation, there are a number of unrequired fields that
	// may not be present on the expected document.
	if err := verifyValuesMatch(ctx, rawExp, rawAct, true); err != nil {
		return fmt.Errorf("%w: %v", errLogDocumentMismatch, err)
	}

	//rawCommandExp := expected.Data.Lookup(commandKey)
	//rawCommandAct := actual.Data.Lookup(commandKey)

	// The command field in the data must be 1-1 with the expectation.
	// TODO: Is there a better way to handle this?
	//if err := verifyValuesMatch(ctx, rawCommandExp, rawCommandAct, true); err != nil {
	//	return fmt.Errorf("%w: %v", errLogDocumentMismatch, err)
	//}

	return nil
}

// clientLogMessages is a struct representing the expected "LogMessages" for a client.
type clientLogMessages struct {
	Client      string        `bson:"client"`
	LogMessages []*logMessage `bson:"messages"`
}

// validateClientLogMessages will validate a single "clientLogMessages" object and return an error if it is invalid,
// i.e. not testable.
func validateClientLogMessages(ctx context.Context, log *clientLogMessages) error {
	if log.Client == "" {
		return errLogClientRequired
	}

	if len(log.LogMessages) == 0 {
		return errLogMessagesRequired
	}

	for _, message := range log.LogMessages {
		if err := validateLogMessage(ctx, message); err != nil {
			return fmt.Errorf("%w: %v", errLogMessageInvalid, err)
		}
	}

	return nil
}

// validateExpectLogMessages will validate a slice of "clientLogMessages" objects and return the first error
// encountered.
func validateExpectLogMessages(ctx context.Context, logs []*clientLogMessages) error {
	seenClientNames := make(map[string]struct{}) // Check for client duplication

	for _, log := range logs {
		if err := validateClientLogMessages(ctx, log); err != nil {
			return fmt.Errorf("%w: %v", errLogClientInvalid, err)
		}

		if _, ok := seenClientNames[log.Client]; ok {
			return fmt.Errorf("%w: %v", errLogClientDuplicate, log.Client)
		}

		seenClientNames[log.Client] = struct{}{}
	}

	return nil
}

// findClientLogMessages will return the first "clientLogMessages" object from a slice of "clientLogMessages" objects
// that matches the client name.
func findClientLogMessages(clientName string, logs []*clientLogMessages) *clientLogMessages {
	for _, client := range logs {
		if client.Client == clientName {
			return client
		}
	}

	return nil
}

// finedClientLogMessagesVolume will return the number of "logMessages" for the first "clientLogMessages" object that
// matches the client name.
func findClientLogMessagesVolume(clientName string, logs []*clientLogMessages) int {
	clm := findClientLogMessages(clientName, logs)
	if clm == nil {
		return 0
	}

	return len(clm.LogMessages)
}

// logMessageValidator defines the expectation for log messages accross all clients.
type logMessageValidator struct {
	testCase *TestCase
	//done     chan struct{} // Channel to signal that the validator is done
	err chan error // Channel to signal that an error has occurred
}

// newLogMessageValidator will create a new "logMessageValidator" from a test case.
func newLogMessageValidator(testCase *TestCase) (*logMessageValidator, error) {
	if testCase == nil {
		return nil, errTestCaseRequired
	}

	if testCase.entities == nil {
		return nil, errEntitiesRequired
	}

	validator := &logMessageValidator{
		testCase: testCase,
		err:      make(chan error, len(testCase.entities.clients())),
	}

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

// stopLogMessageVerificationWorkers will gracefully validate all log messages receiced by all clients and return the
// first error encountered.
func stopLogMessageVerificationWorkers(ctx context.Context, validator *logMessageValidator) error {
	for i := 0; i < len(validator.testCase.ExpectLogMessages); i++ {
		select {
		//case <-validator.done:
		case err := <-validator.err:
			if err != nil {
				return err
			}
		case <-ctx.Done():
			// This error will likely only happen if the expected log workflow have not been implemented
			// for a compontent.
			return fmt.Errorf("%w: %v", errLogContextCanceled, ctx.Err())
		}
	}

	return nil
}

// startLogMessageVerificationWorkers will start a goroutine for each client's expected log messages, listingin on the
// the channel of actual log messages and comparing them to the expected log messages.
func startLogMessageVerificationWorkers(ctx context.Context, validator *logMessageValidator) {
	expected, actual := validator.expected(ctx)
	for _, expected := range expected {
		if expected == nil {
			continue
		}

		go func(expected *clientLogMessages) {
			for actual := range actual[expected.Client] {
				expectedmessage := expected.LogMessages[actual.order-1]
				if expectedmessage == nil {
					validator.err <- nil

					continue
				}

				err := verifyLogMessagesMatch(ctx, expectedmessage, actual.logMessage)
				if err != nil {
					validator.err <- err

					continue
				}

				validator.err <- nil
			}

		}(expected)
	}
}

func (validator *logMessageValidator) close() {}
