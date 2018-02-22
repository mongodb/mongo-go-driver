package command

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/connection"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// Command represents a generic database command.
//
// This can be used to send arbitrary commands to the database, e.g. runCommand.
type Command struct {
	DB      string
	Command interface{}
}

// Encode will encode this command into a wire message for the given server description.
func (c *Command) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) {
	return nil, nil
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (c *Command) Decode(topology.ServerDescription, wiremessage.WireMessage) *Command {
	return nil
}

// Result returns the result of a decoded wire message and server description.
func (c *Command) Result() (bson.Reader, error) { return nil, nil }

// Err returns the error set on this command.
func (c *Command) Err() error { return nil }

// Dispatch handles the full cycle dispatch and execution of this command against the provided
// topology.
func (c *Command) Dispatch(context.Context, topology.Topology) (bson.Reader, error) {
	return nil, nil
}

// RoundTrip handles the execution of this command using the provided connection.
func (c *Command) RoundTrip(context.Context, topology.ServerDescription, connection.Connection) (bson.Reader, error) {
	return nil, nil
}
