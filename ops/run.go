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
		s.ClusterKind == model.Single && s.Model().Kind != model.Mongos, // slaveOk
		command,
	)

	c, err := s.Connection(ctx)
	if err != nil {
		return err
	}

	errChan := make(chan error, 1)
	go func() {
		defer c.Close()
		errChan <- conn.ExecuteCommand(context.Background(), c, request, result)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err = <-errChan:
		return err
	}
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
	errChan := make(chan error, 1)
	go func() {
		defer c.Close()

		if rpMeta := readPrefMeta(s.ReadPref, c.Model().Kind); rpMeta != nil {
			msg.AddMeta(request, map[string]interface{}{
				"$readPreference": rpMeta,
			})
		}

		errChan <- conn.ExecuteCommand(context.Background(), c, request, result)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err = <-errChan:
		return err
	}
}
