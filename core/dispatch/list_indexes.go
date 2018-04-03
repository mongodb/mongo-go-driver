package dispatch

import (
	"context"

	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/topology"
)

// ListIndexes handles the full cycle dispatch and execution of a listIndexes command against the provided
// topology.
func ListIndexes(
	ctx context.Context,
	cmd command.ListIndexes,
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
