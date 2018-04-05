package dispatch

import (
	"context"

	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/topology"
)

// ListCollections handles the full cycle dispatch and execution of a listCollections command against the provided
// topology.
func ListCollections(
	ctx context.Context,
	cmd command.ListCollections,
	topo *topology.Topology,
	selector description.ServerSelector,
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
