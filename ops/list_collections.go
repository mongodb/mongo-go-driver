package ops

import (
	"github.com/10gen/mongo-go-driver/core"
	"github.com/10gen/mongo-go-driver/core/msg"
	"time"
)

// ListCollectionsOptions are the options for listing collections
type ListCollectionsOptions struct {
	// A query filter for the collections
	Filter    interface{}
	// The batch size for fetching results.  A zero value indicate the server's default batch size.
	BatchSize int32
	// The maximum execution time in milliseconds.  A zero value indicates no maximum.
	MaxTime   time.Duration
}

// ListCollections lists the collections in the given database with the given options.
func ListCollections(conn core.Connection, db string, options ListCollectionsOptions) (Cursor, error) {
	if err := validateDB(db); err != nil {
		return nil, err
	}

	listCollectionsCommand := struct {
		ListCollections int32          `bson:"listCollections"`
		Filter          interface{}    `bson:"filter,omitempty"`
		MaxTimeMS       int64          `bson:"maxTimeMS,omitempty"`
		Cursor          *cursorRequest `bson:"cursor"`
	}{
		ListCollections: 1,
		Filter:          options.Filter,
		MaxTimeMS:       int64(options.MaxTime / time.Millisecond),
		Cursor: &cursorRequest{
			BatchSize: options.BatchSize,
		},
	}
	request := msg.NewCommand(
		msg.NextRequestID(),
		db,
		false,
		listCollectionsCommand,
	)

	var result cursorReturningResult

	err := core.ExecuteCommand(conn, request, &result)
	if err != nil {
		return nil, err
	}
	return NewCursor(&result.Cursor, options.BatchSize, conn)
}
