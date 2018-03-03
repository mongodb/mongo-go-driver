package command

import (
	"context"

	"github.com/mongodb/mongo-go-driver/mongo/private/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/connection"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/description"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// ListIndexes represents the listIndexes command.
//
// The listIndexes command lists the indexes for a namespace.
type ListIndexes struct {
	NS   Namespace
	Opts []options.ListIndexesOptioner
}

// Encode will encode this command into a wire message for the given server description.
func (li *ListIndexes) Encode(description.Server) (wiremessage.WireMessage, error) {
	return nil, nil
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (li *ListIndexes) Decode(description.Server, wiremessage.WireMessage) *ListIndexes {
	return nil
}

// Result returns the result of a decoded wire message and server description.
func (li *ListIndexes) Result() (Cursor, error) { return nil, nil }

// Err returns the error set on this command.
func (li *ListIndexes) Err() error { return nil }

// Dispatch handles the full cycle dispatch and execution of this command against the provided
// topology.
func (li *ListIndexes) Dispatch(context.Context, topology.Topology) (Cursor, error) {
	return nil, nil
}

// RoundTrip handles the execution of this command using the provided connection.
func (li *ListIndexes) RoundTrip(context.Context, description.Server, connection.Connection) (Cursor, error) {
	return nil, nil
}
