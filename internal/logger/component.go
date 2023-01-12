package logger

// Component is an enumeration representing the "components" which can be
// logged against. A LogLevel can be configured on a per-component basis.
type Component int

const mongoDBLogAllEnvVar = "MONGODB_LOG_ALL"

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

var componentEnvVarMap = map[string]Component{
	mongoDBLogAllEnvVar:            ComponentAll,
	"MONGODB_LOG_COMMAND":          ComponentCommand,
	"MONGODB_LOG_TOPOLOGY":         ComponentTopology,
	"MONGODB_LOG_SERVER_SELECTION": ComponentServerSelection,
	"MONGODB_LOG_CONNECTION":       ComponentConnection,
}

type ComponentMessage interface {
	Component() Component
	Message() string
	Serialize(maxDocumentLength uint) ([]interface{}, error)
}
