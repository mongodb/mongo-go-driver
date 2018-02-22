package command

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/connection"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// Aggregate represents the aggregate command.
//
// The aggregate command performs an aggregation.
type Aggregate struct {
	NS       Namespace
	Pipeline *bson.Array
	Opts     []options.AggregateOptioner
}

// Encode will encode this command into a wire message for the given server description.
func (a *Aggregate) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) {
	return nil, nil
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (a *Aggregate) Decode(topology.ServerDescription, wiremessage.WireMessage) *Aggregate {
	return nil
}

// Result returns the result of a decoded wire message and server description.
func (a *Aggregate) Result() (Cursor, error) { return nil, nil }

// Err returns the error set on this command.
func (a *Aggregate) Err() error { return nil }

// Dispatch handles the full cycle dispatch and execution of this command against the provided
// topology.
func (a *Aggregate) Dispatch(context.Context, topology.Topology) (Cursor, error) {
	return nil, nil
}

// RoundTrip handles the execution of this command using the provided connection.
func (a *Aggregate) RoundTrip(context.Context, topology.ServerDescription, connection.Connection) (Cursor, error) {
	return nil, nil
}
