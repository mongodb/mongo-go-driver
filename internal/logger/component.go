package logger

import (
	"os"
	"strconv"

	"go.mongodb.org/mongo-driver/bson/primitive"
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
