package unified

import (
	"fmt"

	"go.mongodb.org/mongo-driver/mongo/options"
)

// orderedLogMessage is logMessage with a "order" field representing the order in which the log message was observed.
type orderedLogMessage struct {
	*logMessage
	order int
}

// Logger is the Sink used to captured log messages for logger verification in the unified spec tests.
type Logger struct {
	left      int
	lastOrder int
	logQueue  chan orderedLogMessage
}

func newLogger(logQueue chan orderedLogMessage, expectedCount int) *Logger {
	return &Logger{
		left:      expectedCount,
		lastOrder: 0,
		logQueue:  logQueue,
	}
}

func (logger *Logger) close() {
	close(logger.logQueue)
}

// Info ...
func (logger *Logger) Info(level int, msg string, args ...interface{}) {
	if logger.logQueue == nil {
		return
	}

	logMessage, err := newLogMessage(level, args...)
	if err != nil {
		panic(err)
	}

	// Send the log message to the "orderedLogMessage" channel for validation.
	logger.logQueue <- orderedLogMessage{
		order:      logger.lastOrder + 1,
		logMessage: logMessage,
	}

	logger.left--
	logger.lastOrder++

	if logger.left == 0 {
		close(logger.logQueue)
	}
}

// setLoggerClientOptions sets the logger options for the client entity using client options and the observeLogMessages
// configuration.
func setLoggerClientOptions(entity *clientEntity, clientOptions *options.ClientOptions, olm *observeLogMessages) error {
	if olm == nil {
		return fmt.Errorf("observeLogMessages is nil")
	}

	loggerOpts := options.Logger().SetSink(newLogger(entity.logQueue, olm.volume)).
		SetComponentLevels(map[options.LogComponent]options.LogLevel{
			options.CommandLogComponent:         options.LogLevel(olm.Command.Level()),
			options.TopologyLogComponent:        options.LogLevel(olm.Topology.Level()),
			options.ServerSelectionLogComponent: options.LogLevel(olm.ServerSelection.Level()),
			options.ConnectionLogComponent:      options.LogLevel(olm.Connection.Level()),
		})

	clientOptions.SetLoggerOptions(loggerOpts)

	return nil
}
