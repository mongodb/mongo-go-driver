package ops

import (
	"github.com/10gen/mongo-go-driver/core"
	"github.com/10gen/mongo-go-driver/core/msg"
	"time"
)

// AggregationOptions are the options for the aggregate command.
type AggregationOptions struct {
	// Whether the server can use stable storage for sorting results.
	AllowDiskUse bool
	// The batch size for fetching results.  A zero value indicate the server's default batch size.
	BatchSize    int32
	// The maximum execution time.  A zero value indicates no maximum.
	MaxTime      time.Duration
}

// Aggregate executes the aggregate command with the given pipeline and options.
//
// The pipeline must encode as a BSON array of pipeline stages.
func Aggregate(conn core.Connection, ns Namespace, pipeline interface{}, options AggregationOptions) (Cursor, error) {
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
		false,
		aggregateCommand,
	)

	var result cursorReturningResult

	err := core.ExecuteCommand(conn, request, &result)
	if err != nil {
		return nil, err
	}

	return NewCursor(&result.Cursor, options.BatchSize, conn)
}
