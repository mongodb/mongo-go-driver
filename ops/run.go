package ops

import (
	"context"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/internal"
	"github.com/10gen/mongo-go-driver/msg"
)

// Run executes an arbitrary command against the admin database.
func Run(ctx context.Context, s *SelectedServer, db string, command interface{}, result interface{}) error {
	request := msg.NewCommand(
		msg.NextRequestID(),
		db,
		slaveOk(s.ReadPref),
		command,
	)

	c, err := s.Connection(ctx)
	if err != nil {
		return internal.WrapError(err, "unable to get a connection to execute run command")
	}
	defer c.Close()

	err = conn.ExecuteCommand(ctx, c, request, result)
	if err != nil {
		return internal.WrapError(err, "failed to execute run command")
	}

	return nil
}
