// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"go.mongodb.org/mongo-driver/v2/internal/logger"
)

// LogLevel is an enumeration representing the supported log severity levels.
type LogLevel int

const (
	// LogLevelInfo enables logging of informational messages. These logs
	// are high-level information about normal driver behavior.
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

	// LogComponentConnection enables connection services logging.
	LogComponentConnection LogComponent = LogComponent(logger.ComponentConnection)
)

// LogSink is an interface that can be implemented to provide a custom sink for
// the driver's logs.
type LogSink interface {
	// Info logs a non-error message with the given key/value pairs. This
	// method will only be called if the provided level has been defined
	// for a component in the LoggerOptions.
	//
	// Here are the following level mappings for V = "Verbosity":
	//
	//  - V(0): off
	//  - V(1): informational
	//  - V(2): debugging
	//
	// This level mapping is taken from the go-logr/logr library
	// specifications, specifically:
	//
	// "Level V(0) is the default, and logger.V(0).Info() has the same
	// meaning as logger.Info()."
	Info(level int, message string, keysAndValues ...interface{})

	// Error logs an error message with the given key/value pairs
	Error(err error, message string, keysAndValues ...interface{})
}

// LoggerOptions represent arguments used to configure Logging in the Go Driver.
type LoggerOptions struct {
	// ComponentLevels is a map of LogComponent to LogLevel. The LogLevel
	// for a given LogComponent will be used to determine if a log message
	// should be logged.
	ComponentLevels map[LogComponent]LogLevel

	// Sink is the LogSink that will be used to log messages. If this is
	// nil, the driver will use the standard logging library.
	Sink LogSink

	// MaxDocumentLength is the maximum length of a document to be logged.
	// If the underlying document is larger than this value, it will be
	// truncated and appended with an ellipses "...".
	MaxDocumentLength uint
}

// LoggerOptionsBuilder contains options to configure a logger. Each option can
// be set through setter functions. See documentation for each setter function
// for an explanation of the option.
type LoggerOptionsBuilder struct {
	Opts []func(*LoggerOptions) error
}

// Logger creates a new LoggerOptions instance.
func Logger() *LoggerOptionsBuilder {
	return &LoggerOptionsBuilder{}
}

// List returns a list of LoggerOptions setter functions.
func (opts *LoggerOptionsBuilder) List() []func(*LoggerOptions) error {
	return opts.Opts
}

// SetComponentLevel sets the LogLevel value for a LogComponent.
func (opts *LoggerOptionsBuilder) SetComponentLevel(component LogComponent, level LogLevel) *LoggerOptionsBuilder {
	opts.Opts = append(opts.Opts, func(opts *LoggerOptions) error {
		if opts.ComponentLevels == nil {
			opts.ComponentLevels = map[LogComponent]LogLevel{}
		}

		opts.ComponentLevels[component] = level

		return nil
	})

	return opts
}

// SetMaxDocumentLength sets the maximum length of a document to be logged.
func (opts *LoggerOptionsBuilder) SetMaxDocumentLength(maxDocumentLength uint) *LoggerOptionsBuilder {
	opts.Opts = append(opts.Opts, func(opts *LoggerOptions) error {
		opts.MaxDocumentLength = maxDocumentLength

		return nil
	})

	return opts
}

// SetSink sets the LogSink to use for logging.
func (opts *LoggerOptionsBuilder) SetSink(sink LogSink) *LoggerOptionsBuilder {
	opts.Opts = append(opts.Opts, func(opts *LoggerOptions) error {
		opts.Sink = sink

		return nil
	})

	return opts
}
