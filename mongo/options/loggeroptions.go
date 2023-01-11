package options

import (
	"io"

	"go.mongodb.org/mongo-driver/internal/logger"
)

// LogLevel is an enumeration representing the supported log severity levels.
type LogLevel int

const (
	// InfoLogLevel enables logging of informational messages. These logs are High-level information about normal
	// driver behavior. Example: MongoClient creation or close.
	InfoLogLevel LogLevel = LogLevel(logger.InfoLevel)

	// DebugLogLevel enables logging of debug messages. These logs can be voluminous and are intended for detailed
	// information that may be helpful when debugging an application. Example: A command starting.
	DebugLogLevel LogLevel = LogLevel(logger.DebugLevel)
)

// LogComponent is an enumeration representing the "components" which can be logged against. A LogLevel can be
// configured on a per-component basis.
type LogComponent int

const (
	// AllLogComponents enables logging for all components.
	AllLogComponent LogComponent = LogComponent(logger.ComponentAll)

	// CommandLogComponent enables command monitor logging.
	CommandLogComponent LogComponent = LogComponent(logger.ComponentCommand)

	// TopologyLogComponent enables topology logging.
	TopologyLogComponent LogComponent = LogComponent(logger.ComponentTopology)

	// ServerSelectionLogComponent enables server selection logging.
	ServerSelectionLogComponent LogComponent = LogComponent(logger.ComponentServerSelection)

	// ConnectionLogComponent enables connection services logging.
	ConnectionLogComponent LogComponent = LogComponent(logger.ComponentConnection)
)

// LogSink is an interface that can be implemented to provide a custom sink for the driver's logs.
type LogSink interface {
	// Print(LogLevel, LogComponent, []byte, ...interface{})
	Info(int, string, ...interface{})
}

type ComponentLevels map[LogComponent]LogLevel

// LoggerOptions represent options used to configure Logging in the Go Driver.
type LoggerOptions struct {
	ComponentLevels ComponentLevels

	// Sink is the LogSink that will be used to log messages. If this is nil, the driver will use the standard
	// logging library.
	Sink LogSink

	// Output is the writer to write logs to. If nil, the default is os.Stderr. Output is ignored if Sink is set.
	Output io.Writer

	MaxDocumentLength uint
}

// Logger creates a new LoggerOptions instance.
func Logger() *LoggerOptions {
	return &LoggerOptions{
		ComponentLevels: ComponentLevels{},
	}
}

// SetComponentLevels sets the LogLevel value for a LogComponent.
func (opts *LoggerOptions) SetComponentLevel(component LogComponent, level LogLevel) *LoggerOptions {
	opts.ComponentLevels[component] = level

	return opts
}

func (opts *LoggerOptions) SetMaxDocumentLength(maxDocumentLength uint) *LoggerOptions {
	opts.MaxDocumentLength = maxDocumentLength

	return opts
}

func (opts *LoggerOptions) SetSink(sink LogSink) *LoggerOptions {
	opts.Sink = sink

	return opts
}
