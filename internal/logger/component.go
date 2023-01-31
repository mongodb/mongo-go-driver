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
	CommandFailed    = "Command failed"
	CommandStarted   = "Command started"
	CommandSucceeded = "Command succeeded"
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
		"commandName", cmd.Name,
		"driverConnectionId", cmd.DriverConnectionID,
		"message", cmd.Message,
		"operationId", cmd.OperationID,
		"requestId", cmd.RequestID,
		"serverHost", cmd.ServerHost,
	}, extraKeysAndValues...)

	// Add the optional keys and values.
	port, err := strconv.ParseInt(cmd.ServerPort, 0, 32)
	if err == nil {
		keysAndValues = append(keysAndValues, "serverPort", port)
	}

	// Add the "serverConnectionId" if it is not nil.
	if cmd.ServerConnectionID != nil {
		keysAndValues = append(keysAndValues,
			"serverConnectionId", *cmd.ServerConnectionID)
	}

	// Add the "serviceId" if it is not nil.
	if cmd.ServiceID != nil {
		keysAndValues = append(keysAndValues,
			"serviceId", cmd.ServiceID.Hex())
	}

	return keysAndValues
}
