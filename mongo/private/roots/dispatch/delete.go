package dispatch

import (
	"context"

	"github.com/mongodb/mongo-go-driver/mongo/private/roots/command"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/result"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
)

// Delete handles the full cycle dispatch and execution of a delete command against the provided
// topology.
func Delete(context.Context, command.Delete, *topology.Topology) (result.Delete, error) {
	return result.Delete{}, nil
}
