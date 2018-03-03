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

// FindOneAndReplace represents the findOneAndReplace operation.
//
// The findOneAndReplace command modifies and returns a single document.
type FindOneAndReplace struct {
	NS          Namespace
	Query       *bson.Document
	Replacement *bson.Document
	Opts        []options.FindOneAndReplaceOptioner
}

// Encode will encode this command into a wire message for the given server description.
func (f *FindOneAndReplace) Encode(description.Server) (wiremessage.WireMessage, error) {
	return nil, nil
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (f *FindOneAndReplace) Decode(description.Server, wiremessage.WireMessage) *FindOneAndReplace {
	return nil
}

// Result returns the result of a decoded wire message and server description.
func (f *FindOneAndReplace) Result() (Cursor, error) { return nil, nil }

// Err returns the error set on this command.
func (f *FindOneAndReplace) Err() error { return nil }

// Dispatch handles the full cycle dispatch and execution of this command against the provided
// topology.
func (f *FindOneAndReplace) Dispatch(context.Context, topology.Topology) (Cursor, error) {
	return nil, nil
}

// RoundTrip handles the execution of this command using the provided connection.
func (f *FindOneAndReplace) RoundTrip(context.Context, description.Server, connection.Connection) (Cursor, error) {
	return nil, nil
}
