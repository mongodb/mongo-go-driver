// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongolog

import (
	"fmt"
	"os"
	"strings"
)

// MongoLogger manages the mongo log messages sent to the logger
type MongoLogger struct {
	logger            Logger
	maxDocumentLength interface{}

	commandLevel         Level
	connectionLevel      Level
	sdamLevel            Level
	serverSelectionLevel Level
}

// NewMongoLogger creates a new Mongologger to use with a mongo.Client. Log level defaults to Off
// for all components.
func NewMongoLogger(opts ...*Options) (*MongoLogger, error) {
	mlo := MergeOptions(opts...)

	if mlo.Logger == nil {
		dl := &defaultLogger{}

		dl.writer = os.Stderr
		var err error
		if mlo.OutputFile != nil {
			dl.outputFile = *mlo.OutputFile
			switch strings.ToLower(dl.outputFile) {
			case "stderr":
			case "stdout":
				dl.writer = os.Stdout
			default:
				dl.writer, err = os.OpenFile(dl.outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					return nil, fmt.Errorf("error opening logging output file: %v", err)
				}
			}
		}
		mlo.Logger = dl
	}

	ml := MongoLogger{
		logger:            mlo.Logger,
		maxDocumentLength: 1000,

		commandLevel:         Off,
		connectionLevel:      Off,
		sdamLevel:            Off,
		serverSelectionLevel: Off,
	}

	if mlo.MaxDocumentLength != nil {
		ml.maxDocumentLength = mlo.MaxDocumentLength
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

// TruncateDocument shortens the given string to ml.maxDocumentLength
func (ml MongoLogger) TruncateDocument(doc string) string {
	maxLen, ok := ml.maxDocumentLength.(int)
	if ok && len(doc) > maxLen {
		return doc[:maxLen]
	}
	return doc
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
