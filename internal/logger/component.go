package logger

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

type ComponentMessage interface {
	Component() Component
	Message() string
	Serialize(maxDocumentLength uint) ([]interface{}, error)
}
