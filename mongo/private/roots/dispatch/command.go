package dispatch

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/command"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
)

// Command handles the full cycle dispatch and execution of a command against the provided
// topology.
func Command(context.Context, command.Command, *topology.Topology) (bson.Reader, error) {
	return nil, nil
}
