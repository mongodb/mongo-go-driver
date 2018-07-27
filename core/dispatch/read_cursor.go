package dispatch

import (
	"context"

	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/core/topology"
	"github.com/mongodb/mongo-go-driver/core/uuid"
)

// ReadCursor handles the full dispatch cycle and execution of a read command against the provided topology and returns
// a Cursor over the resulting BSON reader.
func ReadCursor(
	ctx context.Context,
	cmd command.Read,
	topo *topology.Topology,
	selecctor description.ServerSelector,
	clientID uuid.UUID,
	pool *session.Pool,
) (command.Cursor, error) {

	ss, err := topo.SelectServer(ctx, selecctor)
	if err != nil {
		return nil, err
	}

	desc := ss.Description()
	conn, err := ss.Connection(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if cmd.Session == nil && topo.SupportsSessions() {
		cmd.Session, err = session.NewClientSession(pool, clientID, session.Implicit)
		if err != nil {
			return nil, err
		}
	}

	rdr, err := cmd.RoundTrip(ctx, desc, conn)
	if err != nil {
		cmd.Session.EndSession()
		return nil, err
	}

	cursor, err := ss.BuildCursor(rdr, cmd.Session, cmd.Clock)
	if err != nil {
		cmd.Session.EndSession()
		return nil, err
	}

	return cursor, nil
}
