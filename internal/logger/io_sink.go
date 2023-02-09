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

func logCommandStartedMessage(log *log.Logger, kvMap map[string]interface{}) {
	format := "Command %q started on database %q using a connection with " +
		"driver-generated ID %q and server-generated ID %d to %s:%d " +
		"with service ID %q. The requestID is %d and the operation " +
		"ID is %d. Command: %s"

	var serviceID string
	if id, ok := kvMap[KeyServiceID].(string); ok {
		serviceID = id
	}

	log.Printf(format,
		kvMap[KeyCommandName],
		kvMap[KeyDatabaseName],
		kvMap[KeyDriverConnectionID],
		kvMap[KeyServerConnectionID],
		kvMap[KeyServerHost],
		kvMap[KeyServerPort],
		serviceID,
		kvMap[KeyRequestID],
		kvMap[KeyOperationID],
		kvMap[KeyCommand])

}

func logCommandSucceededMessage(log *log.Logger, kvMap map[string]interface{}) {
	format := "Command %q succeeded in %d ms using a connection with " +
		"driver-generated ID %q and server-generated ID %d to %s:%d " +
		"with service ID %q. The requestID is %d and the operation " +
		"ID is %d. Command reply: %s"

	var serviceID string
	if id, ok := kvMap[KeyServiceID].(string); ok {
		serviceID = id
	}

	log.Printf(format,
		kvMap[KeyCommandName],
		kvMap[KeyDurationMS],
		kvMap[KeyDriverConnectionID],
		kvMap[KeyServerConnectionID],
		kvMap[KeyServerHost],
		kvMap[KeyServerPort],
		serviceID,
		kvMap[KeyRequestID],
		kvMap[KeyOperationID],
		kvMap[KeyReply])
}

func logCommandFailedMessage(log *log.Logger, kvMap map[string]interface{}) {
	format := "Command %q failed in %d ms using a connection with " +
		"driver-generated ID %q and server-generated ID %d to %s:%d " +
		"with service ID %q. The requestID is %d and the operation " +
		"ID is %d. Error: %s"

	var serviceID string
	if id, ok := kvMap[KeyServiceID].(string); ok {
		serviceID = id
	}

	log.Printf(format,
		kvMap[KeyCommandName],
		kvMap[KeyDurationMS],
		kvMap[KeyDriverConnectionID],
		kvMap[KeyServerConnectionID],
		kvMap[KeyServerHost],
		kvMap[KeyServerPort],
		serviceID,
		kvMap[KeyRequestID],
		kvMap[KeyOperationID],
		kvMap[KeyFailure])
}

func logPoolCreatedMessage(log *log.Logger, kvMap map[string]interface{}) {
	format := "Connection pool created for %s:%d using options " +
		"maxIdleTimeMS=%d, minPoolSize=%d, maxPoolSize=%d, " +
		"maxConnecting=%d"

	log.Printf(format,
		kvMap[KeyServerHost],
		kvMap[KeyServerPort],
		kvMap[KeyMaxIdleTimeMS],
		kvMap[KeyMinPoolSize],
		kvMap[KeyMaxPoolSize],
		kvMap[KeyMaxConnecting])
}

func logPoolReadyMessage(log *log.Logger, kvMap map[string]interface{}) {
	format := "Connection pool ready for %s:%d"

	log.Printf(format,
		kvMap[KeyServerHost],
		kvMap[KeyServerPort])
}

func logPoolClearedMessage(log *log.Logger, kvMap map[string]interface{}) {
	format := "Connection pool for %s:%d cleared for serviceId %q"

	log.Printf(format,
		kvMap[KeyServerHost],
		kvMap[KeyServerPort],
		kvMap[KeyServiceID])
}

func logPoolClosedMessage(log *log.Logger, kvMap map[string]interface{}) {
	format := "Connection pool closed for %s:%d"

	log.Printf(format,
		kvMap[KeyServerHost],
		kvMap[KeyServerPort])
}

func logConnectionCreatedMessage(log *log.Logger, kvMap map[string]interface{}) {
	format := "Connection created: address=%s:%d, driver-generated ID=%q"

	log.Printf(format,
		kvMap[KeyServerHost],
		kvMap[KeyServerPort],
		kvMap[KeyDriverConnectionID])
}

func logConnectionReadyMessage(log *log.Logger, kvMap map[string]interface{}) {
	format := "Connection ready: address=%s:%d, driver-generated ID=%q"

	log.Printf(format,
		kvMap[KeyServerHost],
		kvMap[KeyServerPort],
		kvMap[KeyDriverConnectionID])
}

func logConnectionClosedMessage(log *log.Logger, kvMap map[string]interface{}) {
	format := "Connection closed: address=%s:%d, driver-generated ID=%q. " +
		"Reason: %s. Error: %s"

	log.Printf(format,
		kvMap[KeyServerHost],
		kvMap[KeyServerPort],
		kvMap[KeyDriverConnectionID],
		kvMap[KeyReason],
		kvMap[KeyError])
}

func logConnectionCheckoutStartedMessage(log *log.Logger, kvMap map[string]interface{}) {
	format := "Checkout started for connection to %s:%d"

	log.Printf(format,
		kvMap[KeyServerHost],
		kvMap[KeyServerPort])
}

func logConnectionCheckoutFailed(log *log.Logger, kvMap map[string]interface{}) {
	format := "Checkout failed for connection to %s:%d. Reason: %s. " +
		"Error: %s"

	log.Printf(format,
		kvMap[KeyServerHost],
		kvMap[KeyServerPort],
		kvMap[KeyReason],
		kvMap[KeyError])
}

func logConnectionCheckedOut(log *log.Logger, kvMap map[string]interface{}) {
	format := "Connection checked out: address=%s:%d, driver-generated ID=%q"

	log.Printf(format,
		kvMap[KeyServerHost],
		kvMap[KeyServerPort],
		kvMap[KeyDriverConnectionID])
}

func logConnectionCheckedIn(log *log.Logger, kvMap map[string]interface{}) {
	format := "Connection checked in: address=%s:%d, driver-generated ID=%q"

	log.Printf(format,
		kvMap[KeyServerHost],
		kvMap[KeyServerPort],
		kvMap[KeyDriverConnectionID])
}

func (osSink *IOSink) Info(_ int, msg string, keysAndValues ...interface{}) {
	kvMap := make(map[string]interface{})
	for i := 0; i < len(keysAndValues); i += 2 {
		kvMap[keysAndValues[i].(string)] = keysAndValues[i+1]
	}

	map[string]func(*log.Logger, map[string]interface{}){
		CommandStarted:            logCommandStartedMessage,
		CommandSucceeded:          logCommandSucceededMessage,
		CommandFailed:             logCommandFailedMessage,
		ConnectionPoolCreated:     logPoolCreatedMessage,
		ConnectionPoolReady:       logPoolReadyMessage,
		ConnectionPoolCleared:     logPoolClearedMessage,
		ConnectionPoolClosed:      logPoolClosedMessage,
		ConnectionCreated:         logConnectionCreatedMessage,
		ConnectionReady:           logConnectionReadyMessage,
		ConnectionClosed:          logConnectionClosedMessage,
		ConnectionCheckoutStarted: logConnectionCheckoutStartedMessage,
		ConnectionCheckoutFailed:  logConnectionCheckoutFailed,
		ConnectionCheckedOut:      logConnectionCheckedOut,
		ConnectionCheckedIn:       logConnectionCheckedIn,
	}[msg](osSink.log, kvMap)
}

func (osSink *IOSink) Error(err error, msg string, kv ...interface{}) {
	osSink.Info(0, msg, kv...)
}
