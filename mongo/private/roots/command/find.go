package command

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/connection"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// Find represents the find command.
//
// The find command finds documents within a collection that match a filter.
type Find struct {
	NS     Namespace
	Filter *bson.Document
	Opts   []options.FindOptioner
}

// Encode will encode this command into a wire message for the given server description.
func (f *Find) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) { return nil, nil }

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (f *Find) Decode(topology.ServerDescription, wiremessage.WireMessage) *Find {
	return nil
}

// Result returns the result of a decoded wire message and server description.
func (f *Find) Result() (Cursor, error) { return nil, nil }

// Err returns the error set on this command.
func (f *Find) Err() error { return nil }

// Dispatch handles the full cycle dispatch and execution of this command against the provided
// topology.
func (f *Find) Dispatch(context.Context, topology.Topology) (Cursor, error) {
	return nil, nil
}

// RoundTrip handles the execution of this command using the provided connection.
func (f *Find) RoundTrip(context.Context, topology.ServerDescription, connection.Connection) (Cursor, error) {
	return nil, nil
}
