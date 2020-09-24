// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongolog

import (
	"fmt"
	"os"
)

// Logger is an interface for loggers to implement
type Logger interface {
	Trace(message string, args ...Field)
	Debug(message string, args ...Field)
	Info(message string, args ...Field)
	Notice(message string, args ...Field)
	Warning(message string, args ...Field)
	Error(message string, args ...Field)
}

// MongoLogger manages the mongo log messages sent to the logger
type MongoLogger struct {
	logger Logger

	logFullCommands      bool
	commandLevel         Level
	connectionLevel      Level
	sdamLevel            Level
	serverSelectionLevel Level
}

// NewMongoLogger creates a new Mongologger to use with a mongo.Client. Log level defaults to Warning
// for all components.
func NewMongoLogger(opts ...*Options) (*MongoLogger, error) {
	mlo := MergeOptions(opts...)

	if mlo.Logger == nil {
		dl := &defaultLogger{}

		dl.writer = os.Stderr
		var err error
		if mlo.OutputFile != nil {
			dl.outputFile = *mlo.OutputFile
			dl.writer, err = os.Create(dl.outputFile)
			if err != nil {
				return nil, fmt.Errorf("error opening logging output file: %v", err)
			}
		}
		mlo.Logger = dl
	}

	ml := MongoLogger{
		logger: mlo.Logger,

		commandLevel:         Warning,
		connectionLevel:      Warning,
		sdamLevel:            Warning,
		serverSelectionLevel: Warning,
	}

	if mlo.LogFullCommands != nil {
		ml.logFullCommands = *mlo.LogFullCommands
	}
	if mlo.CommandLevel != nil {
		ml.commandLevel = *mlo.CommandLevel
	}
	if mlo.ConnectionLevel != nil {
		ml.connectionLevel = *mlo.ConnectionLevel
	}
	if mlo.SDAMLevel != nil {
		ml.sdamLevel = *mlo.SDAMLevel
	}
	if mlo.ServerSelectionLevel != nil {
		ml.serverSelectionLevel = *mlo.ServerSelectionLevel
	}

	return &ml, nil
}

// Log logs a message on logger for component if it passes the log level. Ignores invalid components
// and log levels.
func (ml *MongoLogger) Log(comp Component, level Level, message string, args ...Field) {
	// check component level vs level
	var loggerLevel Level
	switch comp {
	case Command:
		loggerLevel = ml.commandLevel
	case Connection:
		loggerLevel = ml.connectionLevel
	case SDAM:
		loggerLevel = ml.sdamLevel
	case ServerSelection:
		loggerLevel = ml.serverSelectionLevel
	}
	if !loggerLevel.Includes(level) {
		return
	}

	// call appropriate logger function
	switch level {
	case Trace:
		ml.logger.Trace(message, args...)
	case Debug:
		ml.logger.Debug(message, args...)
	case Info:
		ml.logger.Info(message, args...)
	case Notice:
		ml.logger.Notice(message, args...)
	case Warning:
		ml.logger.Warning(message, args...)
	case Error:
		ml.logger.Error(message, args...)
	}
}

// Component indicates the component being logged on.
type Component uint8

// Component constants
const (
	_ Component = iota
	Command
	Connection
	SDAM
	ServerSelection
)

// String returns the string representation of the component.
func (comp Component) String() string {
	switch comp {
	case Command:
		return "Command"
	case Connection:
		return "Connection"
	case SDAM:
		return "SDAM"
	case ServerSelection:
		return "ServerSelection"
	default:
		return "unknown"
	}
}
