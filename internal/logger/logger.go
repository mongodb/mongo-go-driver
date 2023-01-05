package logger

import (
	"io"
	"os"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/internal"
)

const messageKey = "message"
const jobBufferSize = 100

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
	componentLevels map[Component]Level
	sink            LogSink
	jobs            chan job
}

// New will construct a new logger with the given LogSink. If the given LogSink is nil, then the logger will log using
// the standard library.
//
// If the given LogSink is nil, then the logger will log using the standard library with output to os.Stderr.
//
// The "componentLevels" parameter is variadic with the latest value taking precedence. If no component has a LogLevel
// set, then the constructor will attempt to source the LogLevel from the environment.
func New(sink LogSink, componentLevels ...map[Component]Level) *Logger {
	logger := &Logger{
		componentLevels: mergeComponentLevels([]map[Component]Level{
			getEnvComponentLevels(),
			mergeComponentLevels(componentLevels...),
		}...),
	}

	if sink != nil {
		logger.sink = sink
	} else {
		logger.sink = newOSSink(os.Stderr)
	}

	// Initialize the jobs channel and start the printer goroutine.
	logger.jobs = make(chan job, jobBufferSize)
	go logger.startPrinter(logger.jobs)

	return logger
}

// NewWithWriter will construct a new logger with the given writer. If the given writer is nil, then the logger will
// log using the standard library with output to os.Stderr.
func NewWithWriter(w io.Writer, componentLevels ...map[Component]Level) *Logger {
	return New(newOSSink(w), componentLevels...)
}

// Close will close the logger and stop the printer goroutine.
func (logger Logger) Close() {
	close(logger.jobs)
}

// Is will return true if the given LogLevel is enabled for the given LogComponent.
func (logger Logger) Is(level Level, component Component) bool {
	return logger.componentLevels[component] >= level
}

func (logger Logger) Print(level Level, msg ComponentMessage) {
	// TODO: (GODRIVER-2570) We should buffer the "jobs" channel and then accept some level of drop rate with a message to the user.
	// TODO: after the buffer limit has been reached.
	select {
	case logger.jobs <- job{level, msg}:
	default:
		logger.jobs <- job{level, &CommandMessageDropped{
			Message: CommandMessageDroppedDefault,
		}}
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

		sink := logger.sink

		// If the sink is nil, then skip the message.
		if sink == nil {
			return
		}

		// leveInt is the integer representation of the level.
		levelInt := int(level)

		// convert the component message into raw BSON.
		msgBytes, err := bson.Marshal(msg)
		if err != nil {
			sink.Info(levelInt, "error marshalling message to BSON: %v", err)

			return
		}

		rawMsg := bson.Raw(msgBytes)

		// Get the message string from the rawMsg.
		msgValue, err := rawMsg.LookupErr(messageKey)
		if err != nil {
			sink.Info(levelInt, "error getting message from BSON message: %v", err)

			return
		}

		keysAndValues, err := parseKeysAndValues(rawMsg)
		if err != nil {
			sink.Info(levelInt, "error parsing keys and values from BSON message: %v", err)
		}

		sink.Info(int(level), msgValue.StringValue(), keysAndValues...)
	}
}

func commandFinder(keyName string, values []string) func(bson.RawElement) bool {
	valueSet := make(map[string]struct{}, len(values))
	for _, commandName := range values {
		valueSet[commandName] = struct{}{}
	}

	return func(elem bson.RawElement) bool {
		if elem.Key() != keyName {
			return false
		}

		val := elem.Value().StringValue()
		_, ok := valueSet[val]
		if !ok {
			return false
		}

		return true
	}
}

// TODO: (GODRIVER-2570) figure out how to remove the magic strings from this function.
func redactHello(msg bson.Raw, elem bson.RawElement) bool {
	if elem.Key() != "commandName" {
		return false
	}

	val := elem.Value().StringValue()
	if strings.ToLower(val) != internal.LegacyHelloLowercase && val != "hello" {
		return false
	}

	command, err := msg.LookupErr("command")
	if err != nil {
		// If there is no command, then we can't redact anything.
		return false
	}

	// If "command" is a string and it contains "speculativeAuthenticate", then we must redact the command.
	// TODO: (GODRIVER-2570) is this safe? An injection could be possible. Alternative would be to convert the string into
	// TODO: a document.
	if command.Type == bsontype.String {
		return strings.Contains(command.StringValue(), "\"speculativeAuthenticate\":")
	}

	return false
}

// TODO: (GODRIVER-2570) remove magic strings from this function. These strings could probably go into internal/const.go
func parseKeysAndValues(msg bson.Raw) ([]interface{}, error) {
	isRedactableCommand := commandFinder("commandName", []string{
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

	elems, err := msg.Elements()
	if err != nil {
		return nil, err
	}

	var redactCommand bool

	keysAndValues := make([]interface{}, 0, len(elems)*2)
	for _, elem := range elems {
		if isRedactableCommand(elem) || redactHello(msg, elem) {
			redactCommand = true
		}

		var value interface{} = elem.Value()
		switch elem.Key() {
		case "command":
			if redactCommand {
				value = bson.RawValue{
					Type:  bsontype.EmbeddedDocument,
					Value: []byte{0x05, 0x00, 0x00, 0x00, 0x00},
				}.String()
			}
		}

		keysAndValues = append(keysAndValues, elem.Key(), value)
	}

	return keysAndValues, nil
}
