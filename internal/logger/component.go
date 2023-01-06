package logger

import (
	"os"
)

// Component is an enumeration representing the "components" which can be logged against. A LogLevel can be
// configured on a per-component basis.
type Component int

const (
	// AllLogComponents enables logging for all components.
	AllComponent Component = iota

	// CommandComponent enables command monitor logging.
	CommandComponent

	// TopologyComponent enables topology logging.
	TopologyComponent

	// ServerSelectionComponent enables server selection logging.
	ServerSelectionComponent

	// ConnectionComponent enables connection services logging.
	ConnectionComponent
)

// ComponentLiteral is an enumeration representing the string literal "components" which can be logged against.
type ComponentLiteral string

const (
	AllComponentLiteral             ComponentLiteral = "all"
	CommandComponentLiteral         ComponentLiteral = "command"
	TopologyComponentLiteral        ComponentLiteral = "topology"
	ServerSelectionComponentLiteral ComponentLiteral = "serverSelection"
	ConnectionComponentLiteral      ComponentLiteral = "connection"
)

// Component returns the Component for the given ComponentLiteral.
func (componentl ComponentLiteral) Component() Component {
	switch componentl {
	case AllComponentLiteral:
		return AllComponent
	case CommandComponentLiteral:
		return CommandComponent
	case TopologyComponentLiteral:
		return TopologyComponent
	case ServerSelectionComponentLiteral:
		return ServerSelectionComponent
	case ConnectionComponentLiteral:
		return ConnectionComponent
	default:
		return AllComponent
	}
}

type ComponentMessage interface {
	Component() Component
	Message() string
	Serialize() []interface{}
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
func getEnvComponentLevels() map[Component]Level {
	clvls := make(map[Component]Level)
	if all := parseLevel(os.Getenv(string(allComponentEnv))); all != OffLevel {
		clvls[CommandComponent] = all
		clvls[TopologyComponent] = all
		clvls[ServerSelectionComponent] = all
		clvls[ConnectionComponent] = all
	} else {
		clvls[CommandComponent] = parseLevel(os.Getenv(string(commandComponentEnv)))
		clvls[TopologyComponent] = parseLevel(os.Getenv(string(topologyComponentEnv)))
		clvls[ServerSelectionComponent] = parseLevel(os.Getenv(string(serverSelectionComponentEnv)))
		clvls[ConnectionComponent] = parseLevel(os.Getenv(string(connectionComponentEnv)))
	}

	return clvls
}

// mergeComponentLevels returns a new map of LogComponents to LogLevels that is the result of merging the provided
// maps. The maps are merged in order, with the later maps taking precedence over the earlier maps.
func mergeComponentLevels(componentLevels ...map[Component]Level) map[Component]Level {
	merged := make(map[Component]Level)
	for _, clvls := range componentLevels {
		for component, level := range clvls {
			merged[component] = level
		}
	}

	return merged
}
