package dispatch

import (
	"context"

	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/topology"
)

// Count handles the full cycle dispatch and execution of a count command against the provided
// topology.
func Count(
	ctx context.Context,
	cmd command.Count,
	topo *topology.Topology,
	selector topology.ServerSelector,
	rc *readconcern.ReadConcern,
) (int64, error) {

	ss, err := topo.SelectServer(ctx, selector)
	if err != nil {
		return 0, err
	}

	if rc != nil {
		opt, err := readConcernOption(rc)
		if err != nil {
			return 0, err
		}
		cmd.Opts = append(cmd.Opts, opt)
	}

	desc := ss.Description()
	conn, err := ss.Connection(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	return cmd.RoundTrip(ctx, desc, conn)
}
