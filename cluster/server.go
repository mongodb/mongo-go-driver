package cluster

import (
	"context"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/server"
)

// Server represents a logical connection to a server.
type Server interface {
	// Connection gets a connection to the server.
	Connection(context.Context) (conn.Connection, error)
	// Desc gets a description of the server.
	Desc() *server.Desc
}
