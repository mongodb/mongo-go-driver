package command

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/description"
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
func (a *Aggregate) Encode(description.Server) (wiremessage.WireMessage, error) {
	return nil, nil
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (a *Aggregate) Decode(description.Server, wiremessage.WireMessage) *Aggregate {
	return nil
}

// Result returns the result of a decoded wire message and server description.
func (a *Aggregate) Result() (Cursor, error) { return nil, nil }

// Err returns the error set on this command.
func (a *Aggregate) Err() error { return nil }

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter.
func (a *Aggregate) RoundTrip(context.Context, description.Server, wiremessage.ReadWriter) (Cursor, error) {
	return nil, nil
}
