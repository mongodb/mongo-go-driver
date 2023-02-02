// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package logger

import (
	"os"
	"strconv"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	CommandFailed             = "Command failed"
	CommandStarted            = "Command started"
	CommandSucceeded          = "Command succeeded"
	ConnectionPoolCreated     = "Connection pool created"
	ConnectionPoolReady       = "Connection pool ready"
	ConnectionPoolCleared     = "Connection pool cleared"
	ConnectionPoolClosed      = "Connection pool closed"
	ConnectionCreated         = "Connection created"
	ConnectionReady           = "Connection ready"
	ConnectionClosed          = "Connection closed"
	ConnectionCheckoutStarted = "Connection checkout started"
	ConnectionCheckoutFailed  = "Connection checkout failed"
	ConnectionCheckedOut      = "Connection checked out"
	ConnectionCheckedIn       = "Connection checked in"
)

const (
	KeyCommand            = "command"
	KeyCommandName        = "commandName"
	KeyDatabaseName       = "databaseName"
	KeyDriverConnectionID = "driverConnectionId"
	KeyDurationMS         = "durationMS"
	KeyError              = "error"
	KeyFailure            = "failure"
	KeyMaxConnecting      = "maxConnecting"
	KeyMaxIdleTimeMS      = "maxIdleTimeMS"
	KeyMaxPoolSize        = "maxPoolSize"
	KeyMinPoolSize        = "minPoolSize"
	KeyOperationID        = "operationId"
	KeyReason             = "reason"
	KeyReply              = "reply"
	KeyRequestID          = "requestId"
	KeyServerConnectionID = "serverConnectionId"
	KeyServerHost         = "serverHost"
	KeyServerPort         = "serverPort"
	KeyServiceID          = "serviceId"
)

// Reason represents why a connection was closed.
type Reason string

const (
	ReasonConnClosedStale              Reason = "Connection became stale because the pool was cleared"
	ReasonConnClosedIdle               Reason = "Connection has been available but unused for longer than the configured max idle time"
	ReasonConnClosedError              Reason = "An error occurred while using the connection"
	ReasonConnClosedPoolClosed         Reason = "Connection pool was closed"
	ReasonConnCheckoutFailedTimout     Reason = "Wait queue timeout elapsed without a connection becoming available"
	ReasonConnCheckoutFailedError      Reason = "An error occurred while trying to establish a new connection"
	ReasonConnCheckoutFailedPoolClosed Reason = "Connection pool was closed"
)

// Component is an enumeration representing the "components" which can be
// logged against. A LogLevel can be configured on a per-component basis.
type Component int

const (
	// ComponentAll enables logging for all components.
	ComponentAll Component = iota

	// ComponentCommand enables command monitor logging.
	ComponentCommand

	// ComponentTopology enables topology logging.
	ComponentTopology

	// ComponentServerSelection enables server selection logging.
	ComponentServerSelection

	// ComponentConnection enables connection services logging.
	ComponentConnection
)

const (
	mongoDBLogAllEnvVar             = "MONGODB_LOG_ALL"
	mongoDBLogCommandEnvVar         = "MONGODB_LOG_COMMAND"
	mongoDBLogTopologyEnvVar        = "MONGODB_LOG_TOPOLOGY"
	mongoDBLogServerSelectionEnvVar = "MONGODB_LOG_SERVER_SELECTION"
	mongoDBLogConnectionEnvVar      = "MONGODB_LOG_CONNECTION"
)

var componentEnvVarMap = map[string]Component{
	mongoDBLogAllEnvVar:             ComponentAll,
	mongoDBLogCommandEnvVar:         ComponentCommand,
	mongoDBLogTopologyEnvVar:        ComponentTopology,
	mongoDBLogServerSelectionEnvVar: ComponentServerSelection,
	mongoDBLogConnectionEnvVar:      ComponentConnection,
}

// EnvHasComponentVariables returns true if the environment contains any of the
// component environment variables.
func EnvHasComponentVariables() bool {
	for envVar := range componentEnvVarMap {
		if os.Getenv(envVar) != "" {
			return true
		}
	}

	return false
}

// Command is a struct defining common fields that must be included in all
// commands.
type Command struct {
	DriverConnectionID int32               // Driver's ID for the connection
	Name               string              // Command name
	Message            string              // Message associated with the command
	OperationID        int32               // Driver-generated operation ID
	RequestID          int64               // Driver-generated request ID
	ServerConnectionID *int32              // Server's ID for the connection used for the command
	ServerHost         string              // Hostname or IP address for the server
	ServerPort         string              // Port for the server
	ServiceID          *primitive.ObjectID // ID for the command  in load balancer mode
}

// SerializeCommand takes a command and a variable number of key-value pairs and
// returns a slice of interface{} that can be passed to the logger for
// structured logging.
func SerializeCommand(cmd Command, extraKeysAndValues ...interface{}) []interface{} {
	// Initialize the boilerplate keys and values.
	keysAndValues := append([]interface{}{
		KeyCommandName, cmd.Name,
		KeyDriverConnectionID, cmd.DriverConnectionID,
		"message", cmd.Message,
		KeyOperationID, cmd.OperationID,
		KeyRequestID, cmd.RequestID,
		KeyServerHost, cmd.ServerHost,
	}, extraKeysAndValues...)

	// Add the optional keys and values.
	port, err := strconv.ParseInt(cmd.ServerPort, 0, 32)
	if err == nil {
		keysAndValues = append(keysAndValues, KeyServerPort, port)
	}

	// Add the "serverConnectionId" if it is not nil.
	if cmd.ServerConnectionID != nil {
		keysAndValues = append(keysAndValues,
			KeyServerConnectionID, *cmd.ServerConnectionID)
	}

	// Add the "serviceId" if it is not nil.
	if cmd.ServiceID != nil {
		keysAndValues = append(keysAndValues,
			KeyServiceID, cmd.ServiceID.Hex())
	}

	return keysAndValues
}

// Connection contains data that all connection log messages MUST contain.
type Connection struct {
	Message    string // Message associated with the connection
	ServerHost string // Hostname or IP address for the server
	ServerPort string // Port for the server
}

// SerializeConnection serializes a ConnectionMessage into a slice of keys
// and values that can be passed to a logger.
func SerializeConnection(conn Connection, extraKeysAndValues ...interface{}) []interface{} {
	keysAndValues := append([]interface{}{
		"message", conn.Message,
		KeyServerHost, conn.ServerHost,
	}, extraKeysAndValues...)

	// Convert the ServerPort into an integer.
	port, err := strconv.ParseInt(conn.ServerPort, 0, 32)
	if err == nil {
		keysAndValues = append(keysAndValues, KeyServerPort, port)
	}

	return keysAndValues
}
