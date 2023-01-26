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

type Reason string

const (
	ReasonConnectionClosedStale      Reason = "Connection became stale because the pool was cleared"
	ReasonConnectionClosedIdle       Reason = "Connection has been available but unused for longer than the configured max idle time"
	ReasonConnectionClosedError      Reason = "An error occurred while using the connection"
	ReasonConnectionClosedPoolClosed Reason = "Connection pool was closed"
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

type Command struct {
	DriverConnectionID int32
	Name               string
	Message            string
	OperationID        int32
	RequestID          int64
	ServerConnectionID *int32
	ServerHost         string
	ServerPort         string
	ServiceID          *primitive.ObjectID
}

// SerializeCommand serializes a CommandMessage into a slice of keys and values
// that can be passed to a logger.
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

	// Add the optionsl keys and values
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

// ConnectionMessage contains data that all connection log messages MUST contain.
type Connection struct {
	// Message is the literal message to be logged defining the underlying
	// event.
	Message string

	// ServerHost is the hostname, IP address, or Unix domain socket path
	// for the endpoint the pool is for.
	ServerHost string

	// Port is the port for the endpoint the pool is for. If the user does
	// not specify a port and the default (27017) is used, the driver
	// SHOULD include it here.
	ServerPort string
}

// SerializeConnection serializes a ConnectionMessage into a slice of keys
// and values that can be passed to a logger.
func SerializeConnection(conn Connection, extraKeysAndValues ...interface{}) []interface{} {
	keysAndValues := append([]interface{}{
		"message", conn.Message,
		"serverHost", conn.ServerHost,
	}, extraKeysAndValues...)

	// Convert the ServerPort into an integer.
	port, err := strconv.ParseInt(conn.ServerPort, 0, 32)
	if err == nil {
		keysAndValues = append(keysAndValues, "serverPort", port)
	}

	return keysAndValues
}
