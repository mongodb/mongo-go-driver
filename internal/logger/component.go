package logger

import (
	"os"
)

// LogComponent is an enumeration representing the "components" which can be logged against. A LogLevel can be
// configured on a per-component basis.
type LogComponent int

const (
	// AllLogComponents enables logging for all components.
	AllLogComponent LogComponent = iota

	// CommandLogComponent enables command monitor logging.
	CommandLogComponent

	// TopologyLogComponent enables topology logging.
	TopologyLogComponent

	// ServerSelectionLogComponent enables server selection logging.
	ServerSelectionLogComponent

	// ConnectionLogComponent enables connection services logging.
	ConnectionLogComponent
)

type ComponentMessage interface {
	Component() LogComponent
	ExtJSONBytes() ([]byte, error)

	// KeysAndValues returns a slice of alternating keys and values. The keys are strings and the values are
	// arbitrary types. The keys are used to identify the values in the output. This method is used by the log
	// sink for structured logging.
	KeysAndValues() []interface{}
}

type componentEnv string

const (
	allComponentEnv             componentEnv = "MONGODB_LOG_ALL"
	commandComponentEnv         componentEnv = "MONGODB_LOG_COMMAND"
	topologyComponentEnv        componentEnv = "MONGODB_LOG_TOPOLOGY"
	serverSelectionComponentEnv componentEnv = "MONGODB_LOG_SERVER_SELECTION"
	connectionComponentEnv      componentEnv = "MONGODB_LOG_CONNECTION"
)

// getEnvComponentLevels returns a map of LogComponents to LogLevels based on the environment variables set. The
// "MONGODB_LOG_ALL" environment variable takes precedence over all other environment variables. Setting a value for
// "MONGODB_LOG_ALL" is equivalent to setting that value for all of the per-component variables.
func getEnvComponentLevels() map[LogComponent]LogLevel {
	clvls := make(map[LogComponent]LogLevel)
	if all := parseLevel(os.Getenv(string(allComponentEnv))); all != OffLogLevel {
		clvls[CommandLogComponent] = all
		clvls[TopologyLogComponent] = all
		clvls[ServerSelectionLogComponent] = all
		clvls[ConnectionLogComponent] = all
	} else {
		clvls[CommandLogComponent] = parseLevel(os.Getenv(string(commandComponentEnv)))
		clvls[TopologyLogComponent] = parseLevel(os.Getenv(string(topologyComponentEnv)))
		clvls[ServerSelectionLogComponent] = parseLevel(os.Getenv(string(serverSelectionComponentEnv)))
		clvls[ConnectionLogComponent] = parseLevel(os.Getenv(string(connectionComponentEnv)))
	}

	return clvls
}

// mergeComponentLevels returns a new map of LogComponents to LogLevels that is the result of merging the provided
// maps. The maps are merged in order, with the later maps taking precedence over the earlier maps.
func mergeComponentLevels(componentLevels ...map[LogComponent]LogLevel) map[LogComponent]LogLevel {
	merged := make(map[LogComponent]LogLevel)
	for _, clvls := range componentLevels {
		for component, level := range clvls {
			merged[component] = level
		}
	}

	return merged
}
