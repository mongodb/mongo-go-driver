package dispatch

import (
	"context"

	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/result"
	"github.com/mongodb/mongo-go-driver/core/topology"
)

// KillCursors handles the full cycle dispatch and execution of an aggregate command against the provided
// topology.
func KillCursors(
	ctx context.Context,
	cmd command.KillCursors,
	topo *topology.Topology,
	selector description.ServerSelector,
) (result.KillCursors, error) {

	ss, err := topo.SelectServer(ctx, selector)
	if err != nil {
		return result.KillCursors{}, err
	}

	desc := ss.Description()
	conn, err := ss.Connection(ctx)
	if err != nil {
		return result.KillCursors{}, err
	}

	return cmd.RoundTrip(ctx, desc, conn)
}
