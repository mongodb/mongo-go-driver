package dispatch

import (
	"context"

	"github.com/mongodb/mongo-go-driver/mongo/private/roots/command"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
)

// Dispatch handles the full cycle dispatch and execution of an aggregate command against the provided
// topology.
func Aggregate(context.Context, command.Aggregate, topology.Topology) (command.Cursor, error) {
	return nil, nil
}
