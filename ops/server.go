package ops

import (
	"context"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/model"
	"github.com/10gen/mongo-go-driver/readpref"
)

// Server represents a server.
type Server interface {
	// Connection gets a connection to use.
	Connection(context.Context) (conn.Connection, error)
	// Model gets the description of the server.
	Model() *model.Server
}

// SelectedServer represents a binding to a server. It contains a
// read preference in the case that needs to be passed on to the
// server during communication.
type SelectedServer struct {
	Server
	// ClusterKind indicates the kind of the cluster the
	// server was selected from.
	ClusterKind model.ClusterKind
	// ReadPref indicates the read preference that should
	// be passed to MongoS. This can be nil.
	ReadPref *readpref.ReadPref
}
