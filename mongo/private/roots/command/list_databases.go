package command

import (
	"context"

	"github.com/mongodb/mongo-go-driver/mongo/private/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/description"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// ListDatabases represents the listDatabases command.
//
// The listDatabases command lists the databases in a MongoDB deployment.
type ListDatabases struct {
	Opts []options.ListDatabasesOptioner
}

// Encode will encode this command into a wire message for the given server description.
func (ld *ListDatabases) Encode(description.Server) (wiremessage.WireMessage, error) {
	return nil, nil
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (ld *ListDatabases) Decode(description.Server, wiremessage.WireMessage) *ListDatabases {
	return nil
}

// Result returns the result of a decoded wire message and server description.
func (ld *ListDatabases) Result() (Cursor, error) { return nil, nil }

// Err returns the error set on this command.
func (ld *ListDatabases) Err() error { return nil }

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter.
func (ld *ListDatabases) RoundTrip(context.Context, description.Server, wiremessage.ReadWriter) (Cursor, error) {
	return nil, nil
}
