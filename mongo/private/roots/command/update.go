package command

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/connection"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/result"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// Update represents the update command.
//
// The update command updates a set of documents with the database.
type Update struct {
	NS   Namespace
	Docs []*bson.Document
	Opts []options.UpdateOptioner
}

// Encode will encode this command into a wire message for the given server description.
func (u *Update) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) { return nil, nil }

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (u *Update) Decode(topology.ServerDescription, wiremessage.WireMessage) *Update {
	return nil
}

// Result returns the result of a decoded wire message and server description.
func (u *Update) Result() (result.Update, error) { return result.Update{}, nil }

// Err returns the error set on this command.
func (u *Update) Err() error { return nil }

// Dispatch handles the full cycle dispatch and execution of this command against the provided
// topology.
func (u *Update) Dispatch(context.Context, topology.Topology) (result.Update, error) {
	return result.Update{}, nil
}

// RoundTrip handles the execution of this command using the provided connection.
func (u *Update) RoundTrip(context.Context, topology.ServerDescription, connection.Connection) (result.Update, error) {
	return result.Update{}, nil
}
