package logger

import (
	"os"
	"strconv"
	"strings"
	"syscall"
)

// DefaultMaxDocumentLength is the default maximum number of bytes that can be
// logged for a stringified BSON document.
const DefaultMaxDocumentLength = 1000

// TruncationSuffix are trailling ellipsis "..." appended to a message to
// indicate to the user that truncation occurred. This constant does not count
// toward the max document length.
const TruncationSuffix = "..."

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

// Logger represents the configuration for the internal logger.
type Logger struct {
	ComponentLevels   map[Component]Level // Log levels for each component.
	Sink              LogSink             // LogSink for log printing.
	MaxDocumentLength uint                // Command truncation width.
}

// New will construct a new logger. If any of the given options are the
// zero-value of the argument type, then the constructor will attempt to
// source the data from the environment. If the environment has not been set,
// then the constructor will the respective default values.
func New(sink LogSink, maxDocLen uint, compLevels map[Component]Level) *Logger {
	return &Logger{
		ComponentLevels:   selectComponentLevels(compLevels),
		MaxDocumentLength: selectMaxDocumentLength(maxDocLen),
		Sink:              selectLogSink(sink),
	}

}

// LevelComponentEnabled will return true if the given LogLevel is enabled for
// the given LogComponent.
func (logger *Logger) LevelComponentEnabled(level Level, component Component) bool {
	return logger.ComponentLevels[component] >= level
}

// Print will synchronously print the given message to the configured LogSink.
// If the LogSink is nil, then this method will do nothing. Future work could be done to make
// this method asynchronous, see buffer management in libraries such as log4j.
func (logger *Logger) Print(level Level, component Component, msg string, keysAndValues ...interface{}) {
	// If the level is not enabled for the component, then
	// skip the message.
	if !logger.LevelComponentEnabled(level, component) {
		return
	}

	// If the sink is nil, then skip the message.
	if logger.Sink == nil {
		return
	}

	logger.Sink.Info(int(level)-DiffToInfo, msg, keysAndValues...)
}

// Error logs an error, with the given message and key/value pairs.
// It functions similarly to Print, but may have unique behavior, and should be
// preferred for logging errors.
func (logger *Logger) Error(err error, msg string, keysAndValues ...interface{}) {
	if logger.Sink == nil {
		return
	}

	logger.Sink.Error(err, msg, keysAndValues...)
}

// selectMaxDocumentLength will return the integer value of the first non-zero
// function, with the user-defined function taking priority over the environment
// variables. For the environment, the function will attempt to get the value of
// "MONGODB_LOG_MAX_DOCUMENT_LENGTH" and parse it as an unsigned integer. If the
// environment variable is not set or is not an unsigned integer, then this
// function will return the default max document length.
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

const (
	logSinkPathStdout = "stdout"
	logSinkPathStderr = "stderr"
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

	if lowerPath == string(logSinkPathStderr) {
		return NewIOSink(os.Stderr)
	}

	if lowerPath == string(logSinkPathStdout) {
		return NewIOSink(os.Stdout)
	}

	if path != "" {
		return NewIOSink(os.NewFile(uintptr(syscall.Stdout), path))
	}

	return NewIOSink(os.Stderr)
}

// selectComponentLevels returns a new map of LogComponents to LogLevels that is
// the result of merging the user-defined data with the environment, with the
// user-defined data taking priority.
func selectComponentLevels(componentLevels map[Component]Level) map[Component]Level {
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

// truncate will truncate a string to the given width, appending "..." to the
// end of the string if it is truncated. This routine is safe for multi-byte
// characters.
func truncate(str string, width uint) string {
	if width == 0 {
		return ""
	}

	if len(str) <= int(width) {
		return str
	}

	// Truncate the byte slice of the string to the given width.
	newStr := str[:width]

	// Check if the last byte is at the beginning of a multi-byte character.
	// If it is, then remove the last byte.
	if newStr[len(newStr)-1]&0xC0 == 0xC0 {
		return newStr[:len(newStr)-1] + TruncationSuffix
	}

	// Check if the last byte is in the middle of a multi-byte character. If
	// it is, then step back until we find the beginning of the character.
	if newStr[len(newStr)-1]&0xC0 == 0x80 {
		for i := len(newStr) - 1; i >= 0; i-- {
			if newStr[i]&0xC0 == 0xC0 {
				return newStr[:i] + TruncationSuffix
			}
		}
	}

	return newStr + TruncationSuffix
}

// FormatMessage formats a BSON document for logging. The document is truncated
// to the given width.
func FormatMessage(msg string, width uint) string {
	if len(msg) == 0 {
		return "{}"
	}

	return truncate(msg, width)
}
