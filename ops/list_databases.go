package ops

import (
	"context"
	"time"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/internal"
	"github.com/10gen/mongo-go-driver/msg"
	"gopkg.in/mgo.v2/bson"
)

// ListDatabasesOptions are the options for listing databases.
type ListDatabasesOptions struct {
	// The maximum execution time in milliseconds.  A zero value indicates no maximum.
	MaxTime time.Duration
}

// ListDatabases lists the databases with the given options
func ListDatabases(ctx context.Context, s *SelectedServer, options ListDatabasesOptions) (Cursor, error) {

	listDatabasesCommand := struct {
		ListDatabases int32 `bson:"listDatabases"`
		MaxTimeMS     int64 `bson:"maxTimeMS,omitempty"`
	}{
		ListDatabases: 1,
		MaxTimeMS:     int64(options.MaxTime / time.Millisecond),
	}
	request := msg.NewCommand(
		msg.NextRequestID(),
		"admin",
		false,
		listDatabasesCommand,
	)

	c, err := s.Connection(ctx)
	if err != nil {
		return nil, internal.WrapError(err, "unable to get a connection to execute listCollections")
	}
	defer c.Close()

	var result struct {
		Databases []bson.Raw `bson:"databases"`
	}
	err = conn.ExecuteCommand(ctx, c, request, &result)
	if err != nil {
		return nil, err
	}

	return &listDatabasesCursor{
		databases: result.Databases,
		current:   0,
	}, nil
}

type listDatabasesCursor struct {
	databases []bson.Raw
	current   int
}

func (cursor *listDatabasesCursor) Next(_ context.Context, result interface{}) bool {
	if cursor.current < len(cursor.databases) {
		bson.Unmarshal(cursor.databases[cursor.current].Data, result)
		cursor.current++
		return true
	}
	return false
}

// Returns the error status of the cursor
func (cursor *listDatabasesCursor) Err() error {
	return nil
}

// Close the cursor.  Ordinarily this is a no-op as the server closes the cursor when it is exhausted.
// Returns the error status of this cursor so that clients do not have to call Err() separately
func (cursor *listDatabasesCursor) Close(_ context.Context) error {
	return nil
}
