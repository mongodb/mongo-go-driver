package logger

import (
	"fmt"
	"io"
	"os"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/internal"
)

const messageKey = "message"
const jobBufferSize = 100

// TODO: (GODRIVER-2570) add comment
const DefaultMaxDocumentLength = 1000

// TODO: (GODRIVER-2570) add comment
const TruncationSuffix = "..."

// LogSink is an interface that can be implemented to provide a custom sink for the driver's logs.
type LogSink interface {
	Info(int, string, ...interface{})
}

type job struct {
	level Level
	msg   ComponentMessage
}

// Logger is the driver's logger. It is used to log messages from the driver either to OS or to a custom LogSink.
type Logger struct {
	ComponentLevels   map[Component]Level
	Sink              LogSink
	MaxDocumentLength uint

	jobs chan job
}

// New will construct a new logger with the given LogSink. If the given LogSink is nil, then the logger will log using
// the standard library.
//
// If the given LogSink is nil, then the logger will log using the standard library with output to os.Stderr.
//
// The "componentLevels" parameter is variadic with the latest value taking precedence. If no component has a LogLevel
// set, then the constructor will attempt to source the LogLevel from the environment.
// TODO: (GODRIVER-2570) Does this need a constructor? Can we just use a struct?
func New(sink LogSink, maxDocumentLength uint, componentLevels ...map[Component]Level) *Logger {
	logger := &Logger{
		ComponentLevels: mergeComponentLevels([]map[Component]Level{
			getEnvComponentLevels(),
			mergeComponentLevels(componentLevels...),
		}...),
	}

	if sink != nil {
		logger.Sink = sink
	} else {
		logger.Sink = newOSSink(os.Stderr)
	}

	if maxDocumentLength > 0 {
		logger.MaxDocumentLength = maxDocumentLength
	} else {
		logger.MaxDocumentLength = DefaultMaxDocumentLength
	}

	// Initialize the jobs channel and start the printer goroutine.
	logger.jobs = make(chan job, jobBufferSize)
	go logger.startPrinter(logger.jobs)

	return logger
}

// NewWithWriter will construct a new logger with the given writer. If the given writer is nil, then the logger will
// log using the standard library with output to os.Stderr.
func NewWithWriter(w io.Writer, maxDocumentLength uint, componentLevels ...map[Component]Level) *Logger {
	return New(newOSSink(w), maxDocumentLength, componentLevels...)
}

// Close will close the logger and stop the printer goroutine.
func (logger Logger) Close() {
	close(logger.jobs)
}

// Is will return true if the given LogLevel is enabled for the given LogComponent.
func (logger Logger) Is(level Level, component Component) bool {
	return logger.ComponentLevels[component] >= level
}

// TODO: (GODRIVER-2570) add an explanation
func (logger Logger) Print(level Level, msg ComponentMessage) {
	select {
	case logger.jobs <- job{level, msg}:
	default:
		logger.jobs <- job{level, &CommandMessageDropped{}}
	}
}

func (logger *Logger) startPrinter(jobs <-chan job) {
	for job := range jobs {
		level := job.level
		msg := job.msg

		// If the level is not enabled for the component, then skip the message.
		if !logger.Is(level, msg.Component()) {
			return
		}

		sink := logger.Sink

		// If the sink is nil, then skip the message.
		if sink == nil {
			return
		}

		levelInt := int(level)

		keysAndValues, err := formatMessage(msg.Serialize(), logger.MaxDocumentLength)
		if err != nil {
			sink.Info(levelInt, "error parsing keys and values from BSON message: %v", err)

		}

		sink.Info(levelInt-DiffToInfo, msg.Message(), keysAndValues...)
	}
}

func commandFinder(keyName string, values []string) func(string, interface{}) bool {
	valueSet := make(map[string]struct{}, len(values))
	for _, commandName := range values {
		valueSet[commandName] = struct{}{}
	}

	return func(key string, value interface{}) bool {
		valueStr, ok := value.(string)
		if !ok {
			return false
		}

		if key != keyName {
			return false
		}

		_, ok = valueSet[valueStr]
		if !ok {
			return false
		}

		return true
	}
}

// TODO: (GODRIVER-2570) figure out how to remove the magic strings from this function.
func shouldRedactHello(key, val string) bool {
	if key != "commandName" {
		return false
	}

	if strings.ToLower(val) != internal.LegacyHelloLowercase && val != "hello" {
		return false
	}

	return strings.Contains(val, "\"speculativeAuthenticate\":")
}

func truncate(str string, width uint) string {
	if len(str) <= int(width) {
		return str
	}

	// Truncate the byte slice of the string to the given width.
	newStr := str[:width]

	// Check if the last byte is at the beginning of a multi-byte character.
	// If it is, then remove the last byte.
	if newStr[len(newStr)-1]&0xC0 == 0xC0 {
		return newStr[:len(newStr)-1]
	}

	// Check if the last byte is in the middle of a multi-byte character. If it is, then step back until we
	// find the beginning of the character.
	if newStr[len(newStr)-1]&0xC0 == 0x80 {
		for i := len(newStr) - 1; i >= 0; i-- {
			if newStr[i]&0xC0 == 0xC0 {
				return newStr[:i]
			}
		}
	}

	return newStr + TruncationSuffix
}

// TODO: (GODRIVER-2570) remove magic strings from this function. These strings could probably go into internal/const.go
func formatMessage(keysAndValues []interface{}, commandWidth uint) ([]interface{}, error) {
	shouldRedactCommand := commandFinder("commandName", []string{
		"authenticate",
		"saslStart",
		"saslContinue",
		"getnonce",
		"createUser",
		"updateUser",
		"copydbgetnonce",
		"copydbsaslstart",
		"copydb",
	})

	formattedKeysAndValues := make([]interface{}, len(keysAndValues))
	for i := 0; i < len(keysAndValues); i += 2 {
		key := keysAndValues[i].(string)
		val := keysAndValues[i+1]

		switch key {
		case "command", "reply":
			// Command should be a bson.Raw value.
			raw, ok := val.(bson.Raw)
			if !ok {
				return nil, fmt.Errorf("expected value for key %q to be a bson.Raw, but got %T",
					key, val)
			}

			str := raw.String()
			val = truncate(str, commandWidth)

			if shouldRedactCommand(key, str) || shouldRedactHello(key, str) || len(str) == 0 {
				val = bson.RawValue{
					Type:  bsontype.EmbeddedDocument,
					Value: []byte{0x05, 0x00, 0x00, 0x00, 0x00},
				}.String()
			}

		}

		formattedKeysAndValues[i] = key
		formattedKeysAndValues[i+1] = val
	}

	return formattedKeysAndValues, nil
}
