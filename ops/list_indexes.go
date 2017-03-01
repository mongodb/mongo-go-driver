package ops

import (
	"context"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/internal"
	"github.com/10gen/mongo-go-driver/msg"
)

// ListIndexesOptions are the options for listing indexes.
type ListIndexesOptions struct {
	// The batch size for fetching results. A zero value indicates the server's default batch size.
	BatchSize int32
}

// ListIndexes lists the indexes on the given namespace.
func ListIndexes(ctx context.Context, s *SelectedServer, ns Namespace, options ListIndexesOptions) (Cursor, error) {

	listIndexesCommand := struct {
		ListIndexes string `bson:"listIndexes"`
	}{
		ListIndexes: ns.Collection,
	}

	request := msg.NewCommand(
		msg.NextRequestID(),
		ns.DB,
		slaveOk(s.ReadPref),
		listIndexesCommand,
	)

	c, err := s.Connection(ctx)
	if err != nil {
		return nil, internal.WrapError(err, "unable to get a connection to execute listIndexes")
	}
	defer c.Close()

	var result cursorReturningResult
	err = conn.ExecuteCommand(ctx, c, request, &result)

	switch err {
	case nil:
		return NewCursor(&result.Cursor, options.BatchSize, s)
	default:
		if conn.IsNsNotFound(err) {
			return NewExhaustedCursor()
		}
		return nil, err
	}

}
