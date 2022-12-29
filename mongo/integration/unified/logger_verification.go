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
	// LevelLiteral is the literal logging level of the expected log message.
	LevelLiteral logger.LevelLiteral `bson:"level"`

	// ComponentLiteral is the literal logging component of the expected log message.
	ComponentLiteral logger.ComponentLiteral `bson:"component"`

	// Data is the expected data of the log message. This is a required field.
	Data bson.Raw `bson:"data"`

	// FailureIsRedacted is a boolean indicating whether or not the expected log message should be redacted.
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

// clientLogMessages is a struct representing the expected log messages for a client. This is used
// for the "expectEvents" assertion in the unified test format.
type clientLogMessages struct {
	// Client is the name of the client to check for expected log messages. This is a required field.
	Client string `bson:"client"`

	// LogMessages is a slice of expected log messages. This is a required field.
	LogMessages []*logMessage `bson:"messages"`
}

// validate will validate the expectedLogMessasagesForClient and return an error if it is invalid.
func (log *clientLogMessages) validate() error {
	if log.Client == "" {
		return errLogClientRequired
	}

	if log.LogMessages == nil || len(log.LogMessages) == 0 {
		return errLogMessagesRequired
	}

	for _, msg := range log.LogMessages {
		if err := msg.validate(); err != nil {
			return fmt.Errorf("%w: %v", errLogMessageInvalid, err)
		}
	}

	return nil
}

type clientLogs []*clientLogMessages

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

func (logs clientLogs) client(clientName string) *clientLogMessages {
	for _, client := range logs {
		if client.Client == clientName {
			return client
		}
	}

	return nil
}

func (logs clientLogs) clientVolume(clientName string) int {
	client := logs.client(clientName)
	if client == nil {
		return 0
	}

	return len(client.LogMessages)
}

// logMessageValidator defines the expectation for log messages accross all clients.
type logMessageValidator struct {
	testCase *TestCase

	clientLogs map[string]*clientLogMessages
	done       chan struct{}
	err        chan error
}

// startLogMessageValidate will start one worker per client entity that will validate the log messages for that client.
func newLogMessageValidator(testCase *TestCase) *logMessageValidator {
	const messageKey = "message"

	validator := &logMessageValidator{
		testCase:   testCase,
		clientLogs: make(map[string]*clientLogMessages),
		done:       make(chan struct{}, len(testCase.entities.clients())),
		err:        make(chan error, 1),
	}

	for _, clientLogMessages := range validator.testCase.ExpectLogMessages {
		validator.clientLogs[clientLogMessages.Client] = clientLogMessages
	}

	return validator
}

// validate will validate all log messages receiced by all clients and return the first error encountered.
func (validator *logMessageValidator) validate(ctx context.Context) error {
	// Wait until all of the workers have finished or the context has been cancelled.
	for i := 0; i < len(validator.testCase.entities.clients()); i++ {
		select {
		case err := <-validator.err:
			return err
		case <-validator.done:
		case <-ctx.Done():
			// Get the client and log message "message" field for the logs that have not been processed
			// yet.
			var clientNames []string

			for clientName, clientLogMessages := range validator.clientLogs {
				for _, logMessage := range clientLogMessages.LogMessages {
					if logMessage == nil {
						continue
					}

					message, err := logMessage.Data.LookupErr("message")
					if err != nil {
						panic(fmt.Sprintf("expected log message to have a %q field", "message"))
					}

					clientNames = append(clientNames, fmt.Sprintf("%s: %s", clientName, message))
				}
			}

			// This error will likely only happen if the expected logs specified in the "clientNames" have
			// not been implemented.
			return fmt.Errorf("context cancelled before all log messages were processed: %v", clientNames)
		}
	}

	return nil
}

func (validator *logMessageValidator) startClientWorker(clientName string, clientEntity *clientEntity) {
	clientLogs := validator.clientLogs
	if clientLogs == nil {
		return
	}

	clientLog, ok := clientLogs[clientName]
	if !ok {
		return
	}

	for actual := range clientEntity.logQueue {
		expectedMessage := clientLog.LogMessages[actual.order-1]
		if expectedMessage == nil {
			continue
		}

		if err := expectedMessage.is(actual.logMessage); err != nil {
			validator.err <- err

			continue
		}

		// Remove the expected message from the slice so that we can ensure that all expected messages are
		// received.
		clientLog.LogMessages[actual.order-1] = nil
	}

	validator.done <- struct{}{}
}

func (validator *logMessageValidator) startWorkers() {
	for clientName, clientEntity := range validator.testCase.entities.clients() {
		go validator.startClientWorker(clientName, clientEntity)
	}
}

func (validator *logMessageValidator) close() {}
