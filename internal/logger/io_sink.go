// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package logger

import (
	"io"
	"log"

	"go.mongodb.org/mongo-driver/bson"
)

// IOSink writes to an io.Writer using the standard library logging solution and
// is the default sink for the logger, with the default IO being os.Stderr.
type IOSink struct {
	log *log.Logger
}

// Compile-time check to ensure osSink implements the LogSink interface.
var _ LogSink = &IOSink{}

// NewIOSink will create a new IOSink that writes to the provided io.Writer.
func NewIOSink(out io.Writer) *IOSink {
	return &IOSink{
		log: log.New(out, "", log.LstdFlags),
	}
}

// Info will write the provided message and key-value pairs to the io.Writer
// as extended JSON.
func (osSink *IOSink) Info(_ int, msg string, keysAndValues ...interface{}) {
	kvMap := make(map[string]interface{})
	kvMap[KeyMessage] = msg

	for i := 0; i < len(keysAndValues); i += 2 {
		kvMap[keysAndValues[i].(string)] = keysAndValues[i+1]
	}

	kvBytes, err := bson.MarshalExtJSON(kvMap, false, false)
	if err != nil {
		panic(err)
	}

	osSink.log.Println(string(kvBytes))
}

// Error will write the provided error and key-value pairs to the io.Writer
// as extended JSON.
func (osSink *IOSink) Error(err error, msg string, kv ...interface{}) {
	osSink.Info(0, msg, kv...)
}
