package unified

import (
	"fmt"

	"go.mongodb.org/mongo-driver/mongo/options"
)

// logActual is a struct representing an actual log message that was observed by the driver.
type logActual struct {
	position int
	message  *logMessage
	//level    int
	//message  string
	//args     []interface{}
}

// Logger is the Sink used to captured log messages for logger verification in the unified spec tests.
type Logger struct {
	// nextPosition represents the line number of the next log message that will be captured. The first log message
	// will have a position of 1, the second will have a position of 2, and so on. This is used to ensure that the
	// log messages are captured in the order that they are observed, per the specification.
	nextPosition int

	actualCh chan logActual
}

func newLogger(actualCh chan logActual) *Logger {
	return &Logger{
		nextPosition: 1,
		actualCh:     actualCh,
	}
}

// Info ...
func (logger *Logger) Info(level int, msg string, args ...interface{}) {
	if logger.actualCh != nil {
		logMessage, err := newLogMessage(level, args...)
		if err != nil {
			panic(err)
		}

		logger.actualCh <- logActual{
			position: logger.nextPosition,
			message:  logMessage,
		}

		// Increment the nextPosition so that the next log message will have the correct position.
		logger.nextPosition++
	}
}

// setLoggerClientOptions sets the logger options for the client entity using client options and the observeLogMessages
// configuration.
func setLoggerClientOptions(ch chan logActual, clientOptions *options.ClientOptions, olm *observeLogMessages) error {
	if olm == nil {
		return fmt.Errorf("observeLogMessages is nil")
	}

	loggerOpts := options.Logger().SetSink(newLogger(ch)).
		SetComponentLevels(map[options.LogComponent]options.LogLevel{
			options.CommandLogComponent:         options.LogLevel(olm.Command.Level()),
			options.TopologyLogComponent:        options.LogLevel(olm.Topology.Level()),
			options.ServerSelectionLogComponent: options.LogLevel(olm.ServerSelection.Level()),
			options.ConnectionLogComponent:      options.LogLevel(olm.Connection.Level()),
		})

	clientOptions.SetLoggerOptions(loggerOpts)

	return nil
}
