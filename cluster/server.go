package cluster

import (
	"context"

	"github.com/10gen/mongo-go-driver/conn"
)

// Server represents a logical connection to a server.
type Server interface {
	// Connection gets a connection to the server.
	Connection(context.Context) (conn.Connection, error)
}
