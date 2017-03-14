package ops

import (
	"context"
	"time"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/internal"
	"github.com/10gen/mongo-go-driver/msg"
)

// AggregationOptions are the options for the aggregate command.
type AggregationOptions struct {
	// Whether the server can use stable storage for sorting results.
	AllowDiskUse bool
	// The batch size for fetching results. A zero value indicates the server's default batch size.
	BatchSize int32
	// The maximum execution time.  A zero value indicates no maximum.
	MaxTime time.Duration
}

// Aggregate executes the aggregate command with the given pipeline and options.
//
// The pipeline must encode as a BSON array of pipeline stages.
func Aggregate(ctx context.Context, s *SelectedServer, ns Namespace, pipeline interface{}, options AggregationOptions) (Cursor, error) {
	if err := ns.validate(); err != nil {
		return nil, err
	}

	aggregateCommand := struct {
		Collection   string         `bson:"aggregate"`
		AllowDiskUse bool           `bson:"allowDiskUse,omitempty"`
		MaxTimeMS    int64          `bson:"maxTimeMS,omitempty"`
		Pipeline     interface{}    `bson:"pipeline"`
		Cursor       *cursorRequest `bson:"cursor"`
	}{
		Collection:   ns.Collection,
		AllowDiskUse: options.AllowDiskUse,
		MaxTimeMS:    int64(options.MaxTime / time.Millisecond),
		Pipeline:     pipeline,
		Cursor: &cursorRequest{
			BatchSize: options.BatchSize,
		},
	}

	request := msg.NewCommand(
		msg.NextRequestID(),
		ns.DB,
		slaveOk(s.ReadPref),
		aggregateCommand,
	)

	var result cursorReturningResult

	c, err := s.Connection(ctx)
	if err != nil {
		return nil, internal.WrapError(err, "unable to get a connection to execute aggregate")
	}
	defer c.Close()

	if rpMeta := readPrefMeta(s.ReadPref, c.Model().Kind); rpMeta != nil {
		msg.AddMeta(request, map[string]interface{}{
			"$readPreference": rpMeta,
		})
	}

	err = conn.ExecuteCommand(ctx, c, request, &result)
	if err != nil {
		return nil, internal.WrapError(err, "failed to execute aggregate")
	}

	return NewCursor(&result.Cursor, options.BatchSize, s)
}
