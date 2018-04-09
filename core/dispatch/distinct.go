package dispatch

import (
	"context"

	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/result"
	"github.com/mongodb/mongo-go-driver/core/topology"
)

// Distinct handles the full cycle dispatch and execution of a distinct command against the provided
// topology.
func Distinct(
	ctx context.Context,
	cmd command.Distinct,
	topo *topology.Topology,
	selector description.ServerSelector,
	rc *readconcern.ReadConcern,
) (result.Distinct, error) {

	ss, err := topo.SelectServer(ctx, selector)
	if err != nil {
		return result.Distinct{}, err
	}

	if rc != nil {
		opt, err := readConcernOption(rc)
		if err != nil {
			return result.Distinct{}, err
		}
		cmd.Opts = append(cmd.Opts, opt)
	}

	desc := ss.Description()
	conn, err := ss.Connection(ctx)
	if err != nil {
		return result.Distinct{}, err
	}

	return cmd.RoundTrip(ctx, desc, conn)
}
