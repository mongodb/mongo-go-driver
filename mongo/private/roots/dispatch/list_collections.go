package dispatch

import (
	"context"

	"github.com/mongodb/mongo-go-driver/mongo/private/roots/command"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
)

// ListCollections handles the full cycle dispatch and execution of a listCollections command against the provided
// topology.
func ListCollections(
	ctx context.Context,
	cmd command.ListCollections,
	topo *topology.Topology,
	selector topology.ServerSelector,
) (command.Cursor, error) {

	ss, err := topo.SelectServer(ctx, selector)
	if err != nil {
		return nil, err
	}

	conn, err := ss.Connection(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	return cmd.RoundTrip(ctx, ss.Description(), ss, conn)
}
