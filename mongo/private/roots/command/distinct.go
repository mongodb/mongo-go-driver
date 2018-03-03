package command

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/description"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// Distinct represents the disctinct command.
//
// The distinct command returns the distinct values for a specified field
// across a single collection.
type Distinct struct {
	NS    Namespace
	Field string
	Query *bson.Document
	Opts  []options.DistinctOptioner
}

// Encode will encode this command into a wire message for the given server description.
func (d *Distinct) Encode(description.Server) (wiremessage.WireMessage, error) {
	return nil, nil
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (d *Distinct) Decode(description.Server, wiremessage.WireMessage) *Distinct {
	return nil
}

// Result returns the result of a decoded wire message and server description.
func (d *Distinct) Result() ([]interface{}, error) { return nil, nil }

// Err returns the error set on this command.
func (d *Distinct) Err() error { return nil }

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter.
func (d *Distinct) RoundTrip(context.Context, description.Server, wiremessage.ReadWriter) ([]interface{}, error) {
	return nil, nil
}
