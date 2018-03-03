package command

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/connection"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/description"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// ListCollections represents the listCollections command.
//
// The listCollections command lists the collections in a database.
type ListCollections struct {
	DB     string
	Filter *bson.Document
	Opts   []options.ListCollectionsOptioner
}

// Encode will encode this command into a wire message for the given server description.
func (lc *ListCollections) Encode(description.Server) (wiremessage.WireMessage, error) {
	return nil, nil
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (lc *ListCollections) Decode(description.Server, wiremessage.WireMessage) *ListCollections {
	return nil
}

// Result returns the result of a decoded wire message and server description.
func (lc *ListCollections) Result() (Cursor, error) { return nil, nil }

// Err returns the error set on this command.
func (lc *ListCollections) Err() error { return nil }

// Dispatch handles the full cycle dispatch and execution of this command against the provided
// topology.
func (lc *ListCollections) Dispatch(context.Context, topology.Topology) (Cursor, error) {
	return nil, nil
}

// RoundTrip handles the execution of this command using the provided connection.
func (lc *ListCollections) RoundTrip(context.Context, description.Server, connection.Connection) (Cursor, error) {
	return nil, nil
}
