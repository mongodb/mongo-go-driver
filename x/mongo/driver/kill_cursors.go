package driver

import (
	"context"

	"github.com/mongodb/mongo-go-driver/x/mongo/driver/topology"
	"github.com/mongodb/mongo-go-driver/x/network/command"
	"github.com/mongodb/mongo-go-driver/x/network/description"
	"github.com/mongodb/mongo-go-driver/x/network/result"
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
