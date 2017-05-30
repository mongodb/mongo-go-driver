package ops

import (
	"context"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/model"
	"github.com/10gen/mongo-go-driver/msg"
)

// Run executes an arbitrary command against the given database.
func Run(ctx context.Context, s *SelectedServer, db string, command interface{}, result interface{}) error {
	return runMayUseSecondary(ctx, s, db, command, result)
}

func runMustUsePrimary(ctx context.Context, s *SelectedServer, db string, command interface{}, result interface{}) error {
	request := msg.NewCommand(
		msg.NextRequestID(),
		db,
		s.ClusterKind == model.Single, // slaveOk
		command,
	)

	c, err := s.Connection(ctx)
	if err != nil {
		return err
	}
	defer c.Close()

	err = conn.ExecuteCommand(ctx, c, request, result)
	if err != nil {
		return err
	}

	return nil
}

func runMayUseSecondary(ctx context.Context, s *SelectedServer, db string, command interface{}, result interface{}) error {
	request := msg.NewCommand(
		msg.NextRequestID(),
		db,
		slaveOk(s),
		command,
	)

	c, err := s.Connection(ctx)
	if err != nil {
		return err
	}
	defer c.Close()

	if rpMeta := readPrefMeta(s.ReadPref, c.Model().Kind); rpMeta != nil {
		msg.AddMeta(request, map[string]interface{}{
			"$readPreference": rpMeta,
		})
	}

	err = conn.ExecuteCommand(ctx, c, request, result)
	if err != nil {
		return err
	}

	return nil
}
