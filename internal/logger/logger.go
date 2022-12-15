package logger

import (
	"io"
	"os"

	"go.mongodb.org/mongo-driver/bson"
)

const messageKey = "message"

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
func New(sink LogSink, componentLevels ...map[Component]Level) Logger {
	logger := Logger{
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
	logger.jobs = make(chan job)
	go logger.startPrinter(logger.jobs)

	return logger
}

// NewWithWriter will construct a new logger with the given writer. If the given writer is nil, then the logger will
// log using the standard library with output to os.Stderr.
func NewWithWriter(w io.Writer, componentLevels ...map[Component]Level) Logger {
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
	select {
	case logger.jobs <- job{level, msg}:
		// job sent
	default:
		// job dropped
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

		// Gather the keys and values from the BSON message as a variadic slice.
		elems, err := rawMsg.Elements()
		if err != nil {
			sink.Info(levelInt, "error getting elements from BSON message: %v", err)

			return
		}

		var keysAndValues []interface{}
		for _, elem := range elems {
			keysAndValues = append(keysAndValues, elem.Key(), elem.Value())
		}

		// Get the message string from the rawMsg.
		msgValue, err := rawMsg.LookupErr(messageKey)
		if err != nil {
			sink.Info(levelInt, "error getting message from BSON message: %v", err)

			return
		}

		sink.Info(int(level), msgValue.String(), keysAndValues...)
	}
}
