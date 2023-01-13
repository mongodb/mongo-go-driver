package logger

import (
	"os"
	"strconv"
	"strings"
	"syscall"
)

const jobBufferSize = 100
const logSinkPathEnvVar = "MONGODB_LOG_PATH"
const maxDocumentLengthEnvVar = "MONGODB_LOG_MAX_DOCUMENT_LENGTH"

// LogSink represents a logging implementation, this interface should be 1-1
// with the exported "LogSink" interface in the mongo/options package.
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
		ComponentLevels:   selectedComponentLevels(compLevels),
		MaxDocumentLength: selectMaxDocumentLength(maxDocLen),
		Sink:              selectLogSink(sink),

		jobs: make(chan job, jobBufferSize),
	}

}

// Close will close the logger and stop the printer goroutine.
func (logger Logger) Close() {
	// TODO: this is causing test failures
	//close(logger.jobs)
}

// LevelComponentEnabled will return true if the given LogLevel is enabled for
// the given LogComponent.
func (logger Logger) LevelComponentEnabled(level Level, component Component) bool {
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
			if !logger.LevelComponentEnabled(level, msg.Component()) {
				return
			}

			sink := logger.Sink

			// If the sink is nil, then skip the message.
			if sink == nil {
				return
			}

			kv, err := msg.Serialize(logger.MaxDocumentLength)
			if err != nil {
				sink.Error(err, "error serializing message")

				return
			}

			sink.Info(int(level)-DiffToInfo, msg.Message(), kv...)
		}
	}()
}

// selectMaxDocumentLength will return the integer value of the first non-zero
// function, with the user-defined function taking priority over the environment
// variables. For the environment, the function will attempt to get the value of
// "MONGODB_LOG_MAX_DOCUMENT_LENGTH" and parse it as an unsigned integer. If the
// environment variable is not set, then this function will return 0.
func selectMaxDocumentLength(maxDocLen uint) uint {
	if maxDocLen != 0 {
		return maxDocLen
	}

	maxDocLenEnv := os.Getenv(maxDocumentLengthEnvVar)
	if maxDocLenEnv != "" {
		maxDocLenEnvInt, err := strconv.ParseUint(maxDocLenEnv, 10, 32)
		if err == nil {
			return uint(maxDocLenEnvInt)
		}
	}

	return DefaultMaxDocumentLength
}

type logSinkPath string

const (
	logSinkPathStdOut logSinkPath = "stdout"
	logSinkPathStdErr logSinkPath = "stderr"
)

// selectLogSink will return the first non-nil LogSink, with the user-defined
// LogSink taking precedence over the environment-defined LogSink. If no LogSink
// is defined, then this function will return a LogSink that writes to stderr.
func selectLogSink(sink LogSink) LogSink {
	if sink != nil {
		return sink
	}

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

	return newOSSink(os.Stderr)
}

// selectComponentLevels returns a new map of LogComponents to LogLevels that is
// the result of merging the user-defined data with the environment, with the
// user-defined data taking priority.
func selectedComponentLevels(componentLevels map[Component]Level) map[Component]Level {
	selected := make(map[Component]Level)

	// Determine if the "MONGODB_LOG_ALL" environment variable is set.
	var globalEnvLevel *Level
	if all := os.Getenv(mongoDBLogAllEnvVar); all != "" {
		level := ParseLevel(all)
		globalEnvLevel = &level
	}

	for envVar, component := range componentEnvVarMap {
		// If the component already has a level, then skip it.
		if _, ok := componentLevels[component]; ok {
			selected[component] = componentLevels[component]

			continue
		}

		// If the "MONGODB_LOG_ALL" environment variable is set, then
		// set the level for the component to the value of the
		// environment variable.
		if globalEnvLevel != nil {
			selected[component] = *globalEnvLevel

			continue
		}

		// Otherwise, set the level for the component to the value of
		// the environment variable.
		selected[component] = ParseLevel(os.Getenv(envVar))
	}

	return selected
}
