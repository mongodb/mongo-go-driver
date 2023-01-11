package logger

import (
	"os"
	"strconv"
	"strings"
	"syscall"
)

const messageKey = "message"
const jobBufferSize = 100
const logSinkPathEnvVar = "MONGODB_LOG_PATH"
const maxDocumentLengthEnvVar = "MONGODB_LOG_MAX_DOCUMENT_LENGTH"

// LogSink represents a logging implementation, this interface should be 1-1
// with the exported "LogSink" interface in the mongo/options pacakge.
type LogSink interface {
	// Info logs a non-error message with the given key/value pairs. The
	// level argument is provided for optional logging.
	Info(level int, msg string, keysAndValues ...interface{})

	// Error logs an error, with the given message and key/value pairs.
	Error(err error, msg string, keysAndValues ...interface{})
}

type job struct {
	level Level
	msg   ComponentMessage
}

// Logger represents the configuration for the internal logger.
type Logger struct {
	ComponentLevels   map[Component]Level // Log levels for each component.
	Sink              LogSink             // LogSink for log printing.
	MaxDocumentLength uint                // Command truncation width.
	jobs              chan job            // Channel of logs to print.
}

// New will construct a new logger. If any of the given options are the
// zero-value of the argument type, then the constructor will attempt to
// source the data from the environment. If the environment has not been set,
// then the constructor will the respective default values.
func New(sink LogSink, maxDocLen uint, compLevels map[Component]Level) *Logger {
	return &Logger{
		ComponentLevels: selectComponentLevels(
			func() map[Component]Level { return compLevels },
			getEnvComponentLevels,
		),

		MaxDocumentLength: selectMaxDocumentLength(
			func() uint { return maxDocLen },
			getEnvMaxDocumentLength,
		),

		Sink: selectLogSink(
			func() LogSink { return sink },
			getEnvLogSink,
		),

		jobs: make(chan job, jobBufferSize),
	}

}

// Close will close the logger and stop the printer goroutine.
func (logger Logger) Close() {
	// TODO: this is causing test failures
	//close(logger.jobs)
}

// Is will return true if the given LogLevel is enabled for the given
// LogComponent.
func (logger Logger) Is(level Level, component Component) bool {
	return logger.ComponentLevels[component] >= level
}

// Print will print the given message to the configured LogSink. Once the buffer
// is full, conflicting messages will be dropped.
func (logger *Logger) Print(level Level, msg ComponentMessage) {
	select {
	case logger.jobs <- job{level, msg}:
	default:
	}
}

// StartPrintListener will start a goroutine that will listen for log messages
// and attempt to print them to the configured LogSink.
func StartPrintListener(logger *Logger) {
	go func() {
		for job := range logger.jobs {
			level := job.level
			msg := job.msg

			// If the level is not enabled for the component, then
			// skip the message.
			if !logger.Is(level, msg.Component()) {
				return
			}

			sink := logger.Sink

			// If the sink is nil, then skip the message.
			if sink == nil {
				return
			}

			sink.Info(int(level)-DiffToInfo, msg.Message(),
				msg.Serialize(logger.MaxDocumentLength)...)
		}
	}()
}

// getEnvMaxDocumentLength will attempt to get the value of
// "MONGODB_LOG_MAX_DOCUMENT_LENGTH" from the environment, and then parse it as
// an unsigned integer. If the environment variable is not set, then this
// function will return 0.
func getEnvMaxDocumentLength() uint {
	max := os.Getenv(maxDocumentLengthEnvVar)
	if max == "" {
		return 0
	}

	maxUint, err := strconv.ParseUint(max, 10, 32)
	if err != nil {
		return 0
	}

	return uint(maxUint)
}

// selectMaxDocumentLength will return the first non-zero result of the getter
// functions.
func selectMaxDocumentLength(getLen ...func() uint) uint {
	for _, get := range getLen {
		if len := get(); len != 0 {
			return len
		}
	}

	return DefaultMaxDocumentLength
}

type logSinkPath string

const (
	logSinkPathStdOut logSinkPath = "stdout"
	logSinkPathStdErr logSinkPath = "stderr"
)

// getEnvLogsink will check the environment for LogSink specifications. If none
// are found, then a LogSink with an stderr writer will be returned.
func getEnvLogSink() LogSink {
	path := os.Getenv(logSinkPathEnvVar)
	lowerPath := strings.ToLower(path)

	if lowerPath == string(logSinkPathStdErr) {
		return newOSSink(os.Stderr)
	}

	if lowerPath == string(logSinkPathStdOut) {
		return newOSSink(os.Stdout)
	}

	if path != "" {
		return newOSSink(os.NewFile(uintptr(syscall.Stdout), path))
	}

	return nil
}

// selectLogSink will select the first non-nil LogSink from the given LogSinks.
func selectLogSink(getSink ...func() LogSink) LogSink {
	for _, getSink := range getSink {
		if sink := getSink(); sink != nil {
			return sink
		}
	}

	return newOSSink(os.Stderr)
}

// getEnvComponentLevels returns a component-to-level mapping defined by the
// environment variables, with "MONGODB_LOG_ALL" taking priority.
func getEnvComponentLevels() map[Component]Level {
	componentLevels := make(map[Component]Level)

	// If the "MONGODB_LOG_ALL" environment variable is set, then set the
	// level for all components to the value of the environment variable.
	if all := os.Getenv(mongoDBLogAllEnvVar); all != "" {
		level := ParseLevel(all)
		for _, component := range componentEnvVarMap {
			componentLevels[component] = level
		}

		return componentLevels
	}

	// Otherwise, set the level for each component to the value of the
	// environment variable.
	for envVar, component := range componentEnvVarMap {
		componentLevels[component] = ParseLevel(os.Getenv(envVar))
	}

	return componentLevels
}

// selectComponentLevels returns a new map of LogComponents to LogLevels that is
// the result of merging the provided maps. The maps are merged in order, with
// the earlier maps taking priority.
func selectComponentLevels(getters ...func() map[Component]Level) map[Component]Level {
	selected := make(map[Component]Level)
	set := make(map[Component]struct{})

	for _, getComponentLevels := range getters {
		for component, level := range getComponentLevels() {
			if _, ok := set[component]; !ok {
				selected[component] = level
			}

			set[component] = struct{}{}
		}
	}

	return selected
}
