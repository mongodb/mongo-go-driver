// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongolog

import (
	"fmt"
	"io"
	"strconv"
)

// defaultLogger is the default logger for the mongo package. By default, it sends logs to stdErr,
// but can be configured to log to files with SetOutputFile Mongologger option.
// Logs are of the format {level:<level>,msg:<message>,<fieldName>:<fieldValue>,...}
type defaultLogger struct {
	maxLogLineLength int
	outputFile       string

	writer io.Writer
}

var _ Logger = defaultLogger{}

func (dl defaultLogger) log(level Level, message string, args ...Field) {
	log := fmt.Sprintf("{level:%v,msg:%v", level, message)
	if len(args) != 0 {
		for _, field := range args {
			log += fmt.Sprintf(",%v:%v", field.Key, getValueString(field))
		}
	}
	log += "}\n"
	_, _ = io.WriteString(dl.writer, log)
}

// getValueString returns the value stored in field f as a string
func getValueString(f Field) string {
	switch f.Type {
	case Int64Type:
		return strconv.FormatInt(f.Integer, 10)
	case StringType:
		return f.String
	case StringerType:
		return f.Interface.(fmt.Stringer).String()
	default:
		panic(fmt.Sprintf("unknown field type: %v", f))
	}
}

// Trace logs a message at trace level
func (dl defaultLogger) Trace(message string, args ...Field) {
	dl.log(Trace, message, args...)
}

// Debug logs a message at debug level
func (dl defaultLogger) Debug(message string, args ...Field) {
	dl.log(Debug, message, args...)
}

// Info logs a message at info level
func (dl defaultLogger) Info(message string, args ...Field) {
	dl.log(Info, message, args...)
}

// Notice logs a message at notice level
func (dl defaultLogger) Notice(message string, args ...Field) {
	dl.log(Notice, message, args...)
}

// Warning logs a message at warning level
func (dl defaultLogger) Warning(message string, args ...Field) {
	dl.log(Warning, message, args...)
}

// Error logs a message at error level
func (dl defaultLogger) Error(message string, args ...Field) {
	dl.log(Error, message, args...)
}
