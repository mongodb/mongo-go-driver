package logger

import (
	"fmt"
	"io"
	"os"
)

// LogSink is an interface that can be implemented to provide a custom sink for the driver's logs.
type LogSink interface {
	Info(int, string, ...interface{})
}

type job struct {
	level LogLevel
	msg   ComponentMessage
}

// Logger is the driver's logger. It is used to log messages from the driver either to OS or to a custom LogSink.
type Logger struct {
	componentLevels map[LogComponent]LogLevel
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
func New(sink LogSink, componentLevels ...map[LogComponent]LogLevel) Logger {
	logger := Logger{
		componentLevels: mergeComponentLevels([]map[LogComponent]LogLevel{
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
func NewWithWriter(w io.Writer, componentLevels ...map[LogComponent]LogLevel) Logger {
	return New(newOSSink(w), componentLevels...)
}

// Close will close the logger and stop the printer goroutine.
func (logger Logger) Close() {
	close(logger.jobs)
}

// Is will return true if the given LogLevel is enabled for the given LogComponent.
func (logger Logger) Is(level LogLevel, component LogComponent) bool {
	return logger.componentLevels[component] >= level
}

func (logger Logger) Print(level LogLevel, msg ComponentMessage) {
	select {
	case logger.jobs <- job{level, msg}:
		// job sent
	default:
		// job dropped
	}
}

func (logger *Logger) startPrinter(jobs <-chan job) {
	for job := range jobs {
		fmt.Printf("printer job: %v\n", job)

		level := job.level
		msg := job.msg

		if !logger.Is(level, msg.Component()) {
			fmt.Println("printer job dropped", level, msg.Component())
			return
		}

		bytes, err := msg.ExtJSONBytes()
		if err != nil {
			panic(err)
		}

		if sink := logger.sink; sink != nil {
			fmt.Println("printer job sent to sink", level, msg.Component())
			// TODO: the -2 offset is to align the printer with the logr API. We probably shouldn't bake
			// TODO: this into the code. How should we handle this?
			sink.Info(int(level)-2, string(bytes), msg.KeysAndValues()...)
		}

		fmt.Println("printer job done")
	}
}
