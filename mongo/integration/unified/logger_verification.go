package unified

import (
	"context"
	"fmt"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/logger"
)

var (
	errLogMessageDocumentMismatch  = fmt.Errorf("log message document mismatch")
	errLogMessageMarshalingFailure = fmt.Errorf("log message marshaling failure")
	errLogMessageLevelMismatch     = fmt.Errorf("log message level mismatch")
)

// expectedLogMessage is a log message that is expected to be observed by the driver.
type expectedLogMessage struct {
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

// validate will validate the expectedLogMessage and return an error if it is invalid.
func (elm *expectedLogMessage) validate() error {
	if elm.LevelLiteral == "" {
		return fmt.Errorf("level is required")
	}

	if elm.ComponentLiteral == "" {
		return fmt.Errorf("component is required")
	}

	if elm.Data == nil {
		return fmt.Errorf("data is required")
	}

	return nil
}

// isLogActual will check if the "got" logActual argument matches the expectedLogMessage. Note that we do not need to
// compare the component literals, as that can be validated through the messages and arguments.
func (elm *expectedLogMessage) isLogActual(got logActual) error {
	// The levels of the expected log message and the actual log message must match, upto logger.Level.
	if int(elm.LevelLiteral.Level()) != got.level {
		return fmt.Errorf("%w %v, got %v", errLogMessageLevelMismatch, elm.LevelLiteral, got.level)
	}

	// expectedDoc is the expected document that should be logged. This is the document that we will compare to the
	// document associated with logActual.
	expectedDoc := documentToRawValue(elm.Data)

	// actualD is the bson.D analogue of the got.args empty interface slice. For example, if got.args is
	// []interface{}{"foo", 1}, then actualD will be bson.D{{"foo", 1}}.
	actualD := bson.D{}
	for i := 0; i < len(got.args); i += 2 {
		actualD = append(actualD, bson.E{Key: got.args[i].(string), Value: got.args[i+1]})
	}

	// Marshal the actualD bson.D into a bson.Raw so that we can compare it to the expectedDoc bson.RawValue.
	actualRaw, err := bson.Marshal(actualD)
	if err != nil {
		return fmt.Errorf("%w: %v", errLogMessageMarshalingFailure, err)
	}

	// actualDoc is the actual document that was logged. This is the document that we will compare to the expected
	// document.
	actualDoc := documentToRawValue(actualRaw)

	if err := verifyValuesMatch(context.Background(), expectedDoc, actualDoc, true); err != nil {
		return fmt.Errorf("%w: %v", errLogMessageDocumentMismatch, err)
	}

	return nil
}

// expectedLogMessagesForClient is a struct representing the expected log messages for a client. This is used
// for the "expectEvents" assertion in the unified test format.
type expectedLogMessagesForClient struct {
	// Client is the name of the client to check for expected log messages. This is a required field.
	Client string `bson:"client"`

	// Messages is a slice of expected log messages. This is a required field.
	Messages []*expectedLogMessage `bson:"messages"`
}

// validate will validate the expectedLogMessasagesForClient and return an error if it is invalid.
func (elmc *expectedLogMessagesForClient) validate() error {
	if elmc.Client == "" {
		return fmt.Errorf("client is required")
	}

	if elmc.Messages == nil {
		return fmt.Errorf("messages is required")
	}

	for _, msg := range elmc.Messages {
		if err := msg.validate(); err != nil {
			return fmt.Errorf("message is invalid: %v", err)
		}
	}

	return nil
}

type expectedLogMessagesForClients []*expectedLogMessagesForClient

// validate will validate the expectedLogMessagesForClients and return an error if it is invalid.
func (elmc expectedLogMessagesForClients) validate() error {
	// We need to keep track of the client names that we have already seen so that we can ensure that there are
	// not multiple expectedLogMessagesForClient objects for a single client entity.
	seenClientNames := make(map[string]struct{})

	for _, client := range elmc {
		if err := client.validate(); err != nil {
			return fmt.Errorf("client is invalid: %v", err)
		}

		if _, ok := seenClientNames[client.Client]; ok {
			return fmt.Errorf("client %q already exists", client.Client)
		}

		seenClientNames[client.Client] = struct{}{}
	}

	return nil
}

// forClient will return the expectedLogMessagesForClient for the given client name. If no expectedLogMessagesForClient
// exists for the given client name, this will return nil. Note that it should not technically be possible for multible
// expectedLogMessagesForClient objects to exist for a single client entity, but we will return the first one that we
// find.
func (elmc expectedLogMessagesForClients) forClient(clientName string) *expectedLogMessagesForClient {
	for _, client := range elmc {
		if client.Client == clientName {
			return client
		}
	}

	return nil
}

// logMessageResult represents the verification result of a log message.
type logMessageResult struct {
	// err is the error that occurred while verifying the log message. If no error occurred, this will be nil.
	err error
}

// logMesageClientValidator defines the expectation for log messages.
type logMessageClientValidator struct {
	// want are the expected log messages for a given client.
	want *expectedLogMessagesForClient

	// invalid are the message pointers to the log result.
	invalid sync.Map
}

// err will return the first error found for the expected log messages.
func (clientValidator *logMessageClientValidator) validate() error {
	if clientValidator.want == nil {
		return nil
	}

	for _, msg := range clientValidator.want.Messages {
		result, ok := clientValidator.invalid.Load(msg)
		if !ok {
			// If the log message is not found, that means the worker deleted it.
			continue
		}

		if err := result.(*logMessageResult).err; err != nil {
			return err
		}
	}

	return nil
}

// logMessageVAlidator defines the expectation for log messages accross all clients.
type logMessageValidator struct {
	clientValidators map[string]*logMessageClientValidator
}

func (validator *logMessageValidator) close() {}

// addClient wil add a new client to the "logMessageValidator". By default all messages are considered "invalid" and
// "missing" until they are verified.
func (validator *logMessageValidator) addClient(clientName string, all expectedLogMessagesForClients) {
	want := all.forClient(clientName)
	if want == nil {
		return
	}

	if validator.clientValidators == nil {
		validator.clientValidators = make(map[string]*logMessageClientValidator)
	}

	validator.clientValidators[clientName] = &logMessageClientValidator{
		want:    want,
		invalid: sync.Map{},
	}

	// Iterate through all of the "want" messages and create a logMessageResult for each one with a default error
	// message of "message expected, but not logged".
	for _, msg := range want.Messages {
		// Check to see if the "Data" field on the message has a "message" value.
		var err error

		msgStr, ok := msg.Data.Lookup("message").StringValueOK()
		if ok {
			err = fmt.Errorf("message %q for client %q expected, but not logged", msgStr, clientName)
		} else {
			err = fmt.Errorf("message for client %q expected, but not logged", clientName)
		}

		validator.clientValidators[clientName].invalid.Store(msg, &logMessageResult{err: err})
	}
}

// getClient will return the "logMessageClientValidator" for the given client name. If no client exists for the given
// client name, this will return nil.
func (validator *logMessageValidator) getClient(clientName string) *logMessageClientValidator {
	if validator.clientValidators == nil {
		return nil
	}

	return validator.clientValidators[clientName]
}

// validate will validate all log messages receiced by all clients and return the first error encountered.
func (validator *logMessageValidator) validate() error {
	for _, clientValidator := range validator.clientValidators {
		if err := clientValidator.validate(); err != nil {
			return err
		}
	}

	return nil
}

// startLogMessageClientValidator will listen to the "logActual" channel for a given client entity, updating the
// "invalid" map to either (1) delete the "missing message" if the message was found and is valid, or (2) update the
// map to express the error that occurred while validating the message.
func startLogMessageClientValidator(entity *clientEntity, validator *logMessageClientValidator) {
	if validator == nil || validator.want == nil {
		return
	}

	for actual := range entity.loggerActual {
		message := validator.want.Messages[actual.position-1]

		// Lookup the logMessageResult for the message.
		result, ok := validator.invalid.Load(message)
		if !ok {
			continue
		}

		if err := message.isLogActual(actual); err != nil {
			// If the log message is not valid, update the logMessageResult with the error as to why.
			result.(*logMessageResult).err = err

			continue
		}

		// If the message is valid, we can delete the logMessageResult from the map.
		validator.invalid.Delete(message)
	}
}

// startLogMessageValidate will start one worker per client entity that will validate the log messages for that client.
func startLogMessageValidator(tcase *TestCase) *logMessageValidator {
	validator := new(logMessageValidator)
	for clientName, entity := range tcase.entities.clients() {
		validator.addClient(clientName, tcase.ExpectLogMessages)

		go startLogMessageClientValidator(entity, validator.getClient(clientName))
	}

	return validator
}
