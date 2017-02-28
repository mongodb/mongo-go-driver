package ops

import (
	"context"
	"time"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/internal"
	"github.com/10gen/mongo-go-driver/msg"
)

// ListCollectionsOptions are the options for listing collections.
type ListCollectionsOptions struct {
	// A query filter for the collections
	Filter interface{}
	// The batch size for fetching results. A zero value indicate the server's default batch size.
	BatchSize int32
	// The maximum execution time in milliseconds. A zero value indicates no maximum.
	MaxTime time.Duration
}

// ListCollections lists the collections in the given database with the given options.
func ListCollections(ctx context.Context, s *SelectedServer, db string, options ListCollectionsOptions) (Cursor, error) {
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
		slaveOk(s.ReadPref),
		listCollectionsCommand,
	)

	c, err := s.Connection(ctx)
	if err != nil {
		return nil, internal.WrapError(err, "unable to get a connection to execute listCollections")
	}
	defer c.Close()

	var result cursorReturningResult
	err = conn.ExecuteCommand(ctx, c, request, &result)
	if err != nil {
		return nil, err
	}

	return NewCursor(&result.Cursor, options.BatchSize, s)
}
