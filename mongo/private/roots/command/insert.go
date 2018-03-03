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

// Insert represents the insert command.
//
// The insert command inserts a set of documents into the database.
//
// Since the Insert command does not return any value other than ok or
// an error, this type has no Err method.
type Insert struct {
	NS   Namespace
	Docs []*bson.Document
	Opts []options.InsertOptioner
}

// Encode will encode this command into a wire message for the given server description.
func (i *Insert) Encode(description.Server) (wiremessage.WireMessage, error) { return nil, nil }

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (i *Insert) Decode(description.Server, wiremessage.WireMessage) *Insert { return nil }

// Result returns the result of a decoded wire message and server description.
func (i *Insert) Result() error { return nil }

// Dispatch handles the full cycle dispatch and execution of this command against the provided
// topology.
func (i *Insert) Dispatch(context.Context, topology.Topology) error { return nil }

// RoundTrip handles the execution of this command using the provided connection.
func (i *Insert) RoundTrip(context.Context, description.Server, connection.Connection) error {
	return nil
}
