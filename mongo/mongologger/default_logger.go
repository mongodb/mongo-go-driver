// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongologger

import (
	"fmt"
	"io"
	"math"
	"strconv"
	"time"
)

// DefaultLogger is the default logger used for the mongo package
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
			log += fmt.Sprintf(",%v:%v", field.Key, getValue(field))
		}
	}
	log += "}\n"
	_, _ = io.WriteString(dl.writer, log)
}

func getValue(f Field) string {
	switch f.Type {
	case BoolType:
		return strconv.FormatBool(f.Integer == 1)
	case DurationType:
		return time.Duration(f.Integer).String()
	case Float64Type:
		return fmt.Sprintf("%v", math.Float64frombits(uint64(f.Integer)))
	case Float32Type:
		return fmt.Sprintf("%v", math.Float32frombits(uint32(f.Integer)))
	case Int64Type,
		Int32Type:
		return strconv.FormatInt(f.Integer, 10)
	case StringType:
		return f.String
	case TimeType:
		if f.Interface != nil {
			return time.Unix(0, f.Integer).In(f.Interface.(*time.Location)).String()
		}
		// Fall back to UTC if location is nil.
		return time.Unix(0, f.Integer).String()
	case TimeFullType:
		return f.Interface.(time.Time).String()
	case Uint64Type,
		Uint32Type:
		return strconv.FormatUint(uint64(f.Integer), 10)
	case UintptrType:
		return fmt.Sprintf("%v", uintptr(f.Integer))
	case ReflectType:
		return fmt.Sprintf("%v", f.Interface)
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
