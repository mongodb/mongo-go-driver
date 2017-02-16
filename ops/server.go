package ops

import (
	"context"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/readpref"
	"github.com/10gen/mongo-go-driver/server"
)

// Server represents a server.
type Server interface {
	// Connection gets a connection to use.
	Connection(context.Context) (conn.Connection, error)
	// Desc gets the description of the server.
	Desc() *server.Desc
}

// SelectedServer represents a binding to a server. It contains a
// read preference in the case that needs to be passed on to the
// server during communication.
type SelectedServer struct {
	Server
	// ReadPref indicates the read preference that should
	// be passed to MongoS. This can be nil.
	ReadPref *readpref.ReadPref
}
