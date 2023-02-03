// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package logger

import (
	"io"
	"log"
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

func logCommandMessageStarted(log *log.Logger, kvMap map[string]interface{}) {
	format := "Command %q started on database %q using a connection with " +
		"driver-generated ID %q and server-generated ID %d to %s:%d " +
		"with service ID %q. The requestID is %d and the operation " +
		"ID is %d. Command: %s"

	log.Printf(format,
		kvMap["commandName"],
		kvMap["databaseName"],
		kvMap["driverConnectionId"],
		kvMap["serverConnectionId"],
		kvMap["serverHost"],
		kvMap["serverPort"],
		kvMap["serviceId"],
		kvMap["requestId"],
		kvMap["operationId"],
		kvMap["command"])

}

func logCommandMessageSucceeded(log *log.Logger, kvMap map[string]interface{}) {
	format := "Command %q succeeded in %d ms using a connection with " +
		"driver-generated ID %q and server-generated ID %d to %s:%d " +
		"with service ID %q. The requestID is %d and the operation " +
		"ID is %d. Command reply: %s"

	log.Printf(format,
		kvMap["commandName"],
		kvMap["durationMS"],
		kvMap["driverConnectionId"],
		kvMap["serverConnectionId"],
		kvMap["serverHost"],
		kvMap["serverPort"],
		kvMap["serviceId"],
		kvMap["requestId"],
		kvMap["operationId"],
		kvMap["reply"])
}

func logCommandMessageFailed(log *log.Logger, kvMap map[string]interface{}) {
	format := "Command %q failed in %d ms using a connection with " +
		"driver-generated ID %q and server-generated ID %d to %s:%d " +
		"with service ID %q. The requestID is %d and the operation " +
		"ID is %d. Error: %s"

	log.Printf(format,
		kvMap["commandName"],
		kvMap["durationMS"],
		kvMap["serverConnectionID"],
		kvMap["serverHost"],
		kvMap["serverPort"],
		kvMap["requestId"],
		kvMap["operationId"],
		kvMap["failure"])
}

func (osSink *IOSink) Info(_ int, msg string, keysAndValues ...interface{}) {
	kvMap := make(map[string]interface{})
	for i := 0; i < len(keysAndValues); i += 2 {
		kvMap[keysAndValues[i].(string)] = keysAndValues[i+1]
	}

	switch msg {
	case CommandStarted:
		logCommandMessageStarted(osSink.log, kvMap)
	case CommandSucceeded:
		logCommandMessageSucceeded(osSink.log, kvMap)
	case CommandFailed:
		logCommandMessageFailed(osSink.log, kvMap)
	}
}

func (osSink *IOSink) Error(err error, msg string, kv ...interface{}) {
	osSink.Info(0, msg, kv...)
}
