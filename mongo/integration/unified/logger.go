package unified

import (
	"fmt"

	"go.mongodb.org/mongo-driver/internal/logger"
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

func newLogger(logQueue chan orderedLogMessage) *Logger {
	return &Logger{
		lastOrder: 0,
		logQueue:  logQueue,
	}
}

func (log *Logger) close() {
	close(log.logQueue)
}

// Info ...
func (log *Logger) Info(level int, msg string, args ...interface{}) {
	if log.logQueue == nil {
		return
	}

	// Add the Diff back to the level, as there is no need to create a logging offset.
	level = level + logger.DiffToInfo

	logMessage, err := newLogMessage(level, args...)
	if err != nil {
		panic(err)
	}

	// Send the log message to the "orderedLogMessage" channel for validation.
	log.logQueue <- orderedLogMessage{
		order:      log.lastOrder + 1,
		logMessage: logMessage,
	}

	log.lastOrder++
}

// setLoggerClientOptions sets the logger options for the client entity using client options and the observeLogMessages
// configuration.
func setLoggerClientOptions(entity *clientEntity, clientOptions *options.ClientOptions, olm *observeLogMessages) error {
	if olm == nil {
		return fmt.Errorf("observeLogMessages is nil")
	}

	loggerOpts := options.Logger().SetSink(newLogger(entity.logQueue)).
		SetComponentLevel(options.CommandLogComponent, options.LogLevel(olm.Command.Level())).
		SetComponentLevel(options.TopologyLogComponent, options.LogLevel(olm.Topology.Level())).
		SetComponentLevel(options.ServerSelectionLogComponent, options.LogLevel(olm.ServerSelection.Level())).
		SetComponentLevel(options.ConnectionLogComponent, options.LogLevel(olm.Connection.Level()))

	clientOptions.SetLoggerOptions(loggerOpts)

	return nil
}
