package command

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/connection"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// Count represents the count command.
//
// The count command counts how many documents in a collection match the given query.
type Count struct {
	NS    Namespace
	Query *bson.Document
	Opts  []options.CountOptioner

	err    error
	result int64
}

// Encode will encode this command into a wire message for the given server description.
func (c *Count) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) { return nil, nil }

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (c *Count) Decode(topology.ServerDescription, wiremessage.WireMessage) *Count { return nil }

// Result returns the result of a decoded wire message and server description.
func (c *Count) Result() (int64, error) { return 0, nil }

// Err returns the error set on this command.
func (c *Count) Err() error { return nil }

// Dispatch handles the full cycle dispatch and execution of this command against the provided
// topology.
func (c *Count) Dispatch(context.Context, topology.Topology) (int64, error) { return 0, nil }

// RoundTrip handles the execution of this command using the provided connection.
func (c *Count) RoundTrip(context.Context, topology.ServerDescription, connection.Connection) (int64, error) {
	return 0, nil
}
