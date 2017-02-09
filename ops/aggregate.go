package ops

import (
	. "github.com/10gen/mongo-go-driver/core"
	. "github.com/10gen/mongo-go-driver/core/msg"
)

// The options for the aggregate command
type AggregationOptions struct {
	// Whether the server can use stable storage for sorting results.
	AllowDiskUse bool
	// The batch size for fetching results.  A zero value indicate the server's default batch size.
	BatchSize    int32
	// The maximum execution time in milliseconds.  A zero value indicates no maximum.
	MaxTimeMS    int64
}

// Execute the aggregate command with the given pipeline and options
// The pipeline must encode as a BSON array of pipeline stages
func Aggregate(conn Connection, namespace *Namespace, pipeline interface{}, options *AggregationOptions) (Cursor, error) {

	aggregateCommand := struct {
		Collection   string         `bson:"aggregate"`
		AllowDiskUse bool           `bson:"allowDiskUse,omitempty"`
		MaxTimeMS    int64          `bson:"maxTimeMS,omitempty"`
		Pipeline     interface{}    `bson:"pipeline"`
		Cursor       *cursorRequest `bson:"cursor"`
	}{
		Collection:   namespace.CollectionName,
		AllowDiskUse: options.AllowDiskUse,
		MaxTimeMS:    options.MaxTimeMS,
		Pipeline:     pipeline,
		Cursor: &cursorRequest{
			BatchSize: options.BatchSize,
		},
	}
	request := NewCommand(
		NextRequestID(),
		namespace.DatabaseName,
		false,
		aggregateCommand,
	)

	var result cursorReturningResult

	err := ExecuteCommand(conn, request, &result)
	if err != nil {
		return nil, err
	}

	return NewCursor(&result.Cursor, options.BatchSize, conn), nil
}
