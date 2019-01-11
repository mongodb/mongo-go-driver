package driverx

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
	"go.mongodb.org/mongo-driver/x/network/description"
)

// Deployment is implemented by types that can select a server from a deployment.
type Deployment interface {
	SelectServer(context.Context, description.ServerSelector) (Server, error)
	Description() description.Topology
}

// Server represents a MongoDB server. Implementations should pool connections and handle the
// retrieving and returning of connections.
type Server interface {
	Connection(context.Context) (Connection, error)
}

// Connection represents a connection to a MongoDB server.
type Connection interface {
	WriteWireMessage(context.Context, []byte) error
	ReadWireMessage(ctx context.Context, dst []byte) ([]byte, error)
	Description() description.Server
	Close() error
	ID() string
}

// Namespace encapsulates a database and collection name, which together
// uniquely identifies a collection within a MongoDB cluster.
type Namespace struct {
	DB         string
	Collection string
}

type BatchCursor struct{}

// Collation allows users to specify language-specific rules for string comparison, such as
// rules for lettercase and accent marks.
type Collation struct {
	Locale          string `bson:",omitempty"` // The locale
	CaseLevel       bool   `bson:",omitempty"` // The case level
	CaseFirst       string `bson:",omitempty"` // The case ordering
	Strength        int    `bson:",omitempty"` // The number of comparision levels to use
	NumericOrdering bool   `bson:",omitempty"` // Whether to order numbers based on numerical order and not collation order
	Alternate       string `bson:",omitempty"` // Whether spaces and punctuation are considered base characters
	MaxVariable     string `bson:",omitempty"` // Which characters are affected by alternate: "shifted"
	Backwards       bool   `bson:",omitempty"` // Causes secondary differences to be considered in reverse order, as it is done in the French language
}

// Retryable writes are supported if the server supports sessions, the operation is not
// within a transaction, and the write is acknowledged
func retrySupported(
	tdesc description.Topology,
	desc description.Server,
	sess *session.Client,
	wc *writeconcern.WriteConcern,
) bool {
	return (tdesc.SessionTimeoutMinutes != 0 && tdesc.Kind != description.Single) &&
		description.SessionsSupported(desc.WireVersion) &&
		sess != nil &&
		!(sess.TransactionInProgress() || sess.TransactionStarting()) &&
		writeconcern.AckWrite(wc)
}
