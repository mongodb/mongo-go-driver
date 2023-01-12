package options

import (
	"go.mongodb.org/mongo-driver/internal/logger"
)

// LogLevel is an enumeration representing the supported log severity levels.
type LogLevel int

const (
	// LogLevelInfo enables logging of informational messages. These logs
	// are High-level information about normal driver behavior.
	LogLevelInfo LogLevel = LogLevel(logger.LevelInfo)

	// LogLevelDebug enables logging of debug messages. These logs can be
	// voluminous and are intended for detailed information that may be
	// helpful when debugging an application.
	LogLevelDebug LogLevel = LogLevel(logger.LevelDebug)
)

// LogComponent is an enumeration representing the "components" which can be
// logged against. A LogLevel can be configured on a per-component basis.
type LogComponent int

const (
	// LogComponentAll enables logging for all components.
	LogComponentAll LogComponent = LogComponent(logger.ComponentAll)

	// LogComponentCommand enables command monitor logging.
	LogComponentCommand LogComponent = LogComponent(logger.ComponentCommand)

	// LogComponentTopology enables topology logging.
	LogComponentTopology LogComponent = LogComponent(logger.ComponentTopology)

	// LogComponentServerSelection enables server selection logging.
	LogComponentServerSelection LogComponent = LogComponent(logger.ComponentServerSelection)

	// LogComponentconnection enables connection services logging.
	LogComponentconnection LogComponent = LogComponent(logger.ComponentConnection)
)

// LogSink is an interface that can be implemented to provide a custom sink for
// the driver's logs.
type LogSink interface {
	Info(int, string, ...interface{})
	Error(error, string, ...interface{})
}

// ComponentLevels is a map of LogComponent to LogLevel.
type ComponentLevels map[LogComponent]LogLevel

// LoggerOptions represent options used to configure Logging in the Go Driver.
type LoggerOptions struct {
	// ComponentLevels is a map of LogComponent to LogLevel. The LogLevel
	// for a given LogComponent will be used to determine if a log message
	// should be logged.
	ComponentLevels ComponentLevels

	// Sink is the LogSink that will be used to log messages. If this is
	// nil, the driver will use the standard logging library.
	Sink LogSink

	// MaxDocumentLength is the maximum length of a document to be logged.
	// If the underlying document is larger than this value, it will be
	// truncated and appended with an ellipses "...".
	MaxDocumentLength uint
}

// Logger creates a new LoggerOptions instance.
func Logger() *LoggerOptions {
	return &LoggerOptions{
		ComponentLevels: ComponentLevels{},
	}
}

// SetComponentLevel sets the LogLevel value for a LogComponent.
func (opts *LoggerOptions) SetComponentLevel(component LogComponent, level LogLevel) *LoggerOptions {
	opts.ComponentLevels[component] = level

	return opts
}

// SetMaxDocumentLength sets the maximum length of a document to be logged.
func (opts *LoggerOptions) SetMaxDocumentLength(maxDocumentLength uint) *LoggerOptions {
	opts.MaxDocumentLength = maxDocumentLength

	return opts
}

// SetSink sets the LogSink to use for logging.
func (opts *LoggerOptions) SetSink(sink LogSink) *LoggerOptions {
	opts.Sink = sink

	return opts
}
