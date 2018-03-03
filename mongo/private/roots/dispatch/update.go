package dispatch

import (
	"context"

	"github.com/mongodb/mongo-go-driver/mongo/private/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/command"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/result"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
	"github.com/mongodb/mongo-go-driver/mongo/writeconcern"
)

// Update handles the full cycle dispatch and execution of an update command against the provided
// topology.
func Update(
	ctx context.Context,
	cmd command.Update,
	topo *topology.Topology,
	selector topology.ServerSelector,
	wc *writeconcern.WriteConcern,
) (result.Update, error) {

	ss, err := topo.SelectServer(ctx, selector)
	if err != nil {
		return result.Update{}, err
	}

	if wc != nil {
		opt, err := writeConcernOption(wc)
		if err != nil {
			return result.Update{}, err
		}
		cmd.Opts = append(cmd.Opts, opt)
	}

	// NOTE: We iterate through the options because the user may have provided
	// an option explicitly and that needs to override the provided write concern.
	// We put this here because it would complicate the methods that call this to
	// parse out the option.
	acknowledged := true
	for _, opt := range cmd.Opts {
		wc, ok := opt.(options.OptWriteConcern)
		if !ok {
			continue
		}
		acknowledged = wc.Acknowledged
		break
	}
	desc := ss.Description()
	conn, err := ss.Connection(ctx)
	if err != nil {
		return result.Update{}, err
	}

	if !acknowledged {
		go func() {
			defer func() { _ = recover() }()
			defer conn.Close()
			_, _ = cmd.RoundTrip(ctx, desc, conn)
		}()
		return result.Update{}, ErrUnacknowledgedWrite
	}
	defer conn.Close()

	return cmd.RoundTrip(ctx, desc, conn)
}
