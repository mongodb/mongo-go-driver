package dispatch

import (
	"context"

	"github.com/mongodb/mongo-go-driver/mongo/private/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/command"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
	"github.com/mongodb/mongo-go-driver/mongo/writeconcern"
)

// Aggregate handles the full cycle dispatch and execution of an aggregate command against the provided
// topology.
func Aggregate(
	ctx context.Context,
	cmd command.Aggregate,
	topo *topology.Topology,
	readSelector, writeSelector topology.ServerSelector,
	wc *writeconcern.WriteConcern,
) (command.Cursor, error) {

	dollarOut := cmd.HasDollarOut()

	var ss *topology.SelectedServer
	var err error
	acknowledged := true
	switch dollarOut {
	case true:
		ss, err = topo.SelectServer(ctx, writeSelector)
		if err != nil {
			return nil, err
		}
		if wc != nil {
			elem, err := wc.MarshalBSONElement()
			if err != nil {
				return nil, err
			}

			opt := options.OptWriteConcern{WriteConcern: elem, Acknowledged: wc.Acknowledged()}
			cmd.Opts = append(cmd.Opts, opt)
		}

		for _, opt := range cmd.Opts {
			wc, ok := opt.(options.OptWriteConcern)
			if !ok {
				continue
			}
			acknowledged = wc.Acknowledged
			break
		}

	case false:
		ss, err = topo.SelectServer(ctx, readSelector)
		if err != nil {
			return nil, err
		}
	}

	desc := ss.Description()
	conn, err := ss.Connection(ctx)
	if err != nil {
		return nil, err
	}

	if !acknowledged {
		go func() {
			defer func() { _ = recover() }()
			defer conn.Close()
			_, _ = cmd.RoundTrip(ctx, desc, ss, conn)
		}()
		return nil, ErrUnacknowledgedWrite
	}
	defer conn.Close()

	return cmd.RoundTrip(ctx, desc, ss, conn)
}
