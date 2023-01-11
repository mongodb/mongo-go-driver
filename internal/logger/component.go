package logger

import (
	"os"
	"strings"
)

// Component is an enumeration representing the "components" which can be logged against. A LogLevel can be
// configured on a per-component basis.
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

// ComponentLiteral is an enumeration representing the string literal "components" which can be logged against.
type ComponentLiteral string

const (
	ComponentLiteralAll             ComponentLiteral = "all"
	ComponentLiterallCommand        ComponentLiteral = "command"
	ComponentLiteralTopology        ComponentLiteral = "topology"
	ComponentLiteralServerSelection ComponentLiteral = "serverSelection"
	ComponentLiteralConnection      ComponentLiteral = "connection"
)

// Component returns the Component for the given ComponentLiteral.
func (componentLiteral ComponentLiteral) Component() Component {
	switch componentLiteral {
	case ComponentLiteralAll:
		return ComponentAll
	case ComponentLiterallCommand:
		return ComponentCommand
	case ComponentLiteralTopology:
		return ComponentTopology
	case ComponentLiteralServerSelection:
		return ComponentServerSelection
	case ComponentLiteralConnection:
		return ComponentConnection
	default:
		return ComponentAll
	}
}

// componentEnvVar is an enumeration representing the environment variables which can be used to configure
// a component's log level.
type componentEnvVar string

const (
	componentEnvVarAll             componentEnvVar = "MONGODB_LOG_ALL"
	componentEnvVarCommand         componentEnvVar = "MONGODB_LOG_COMMAND"
	componentEnvVarTopology        componentEnvVar = "MONGODB_LOG_TOPOLOGY"
	componentEnvVarServerSelection componentEnvVar = "MONGODB_LOG_SERVER_SELECTION"
	componentEnvVarConnection      componentEnvVar = "MONGODB_LOG_CONNECTION"
)

var allComponentEnvVars = []componentEnvVar{
	componentEnvVarAll,
	componentEnvVarCommand,
	componentEnvVarTopology,
	componentEnvVarServerSelection,
	componentEnvVarConnection,
}

func (env componentEnvVar) component() Component {
	return ComponentLiteral(strings.ToLower(os.Getenv(string(env)))).Component()
}

type ComponentMessage interface {
	Component() Component
	Message() string
	Serialize() []interface{}
}
