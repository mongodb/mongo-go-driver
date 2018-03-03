package command

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/description"
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
func (c *Count) Encode(description.Server) (wiremessage.WireMessage, error) { return nil, nil }

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (c *Count) Decode(description.Server, wiremessage.WireMessage) *Count { return nil }

// Result returns the result of a decoded wire message and server description.
func (c *Count) Result() (int64, error) { return 0, nil }

// Err returns the error set on this command.
func (c *Count) Err() error { return nil }

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter.
func (c *Count) RoundTrip(context.Context, description.Server, wiremessage.ReadWriter) (int64, error) {
	return 0, nil
}
