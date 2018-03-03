package command

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/description"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// Find represents the find command.
//
// The find command finds documents within a collection that match a filter.
type Find struct {
	NS     Namespace
	Filter *bson.Document
	Opts   []options.FindOptioner
}

// Encode will encode this command into a wire message for the given server description.
func (f *Find) Encode(description.Server) (wiremessage.WireMessage, error) { return nil, nil }

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (f *Find) Decode(description.Server, wiremessage.WireMessage) *Find {
	return nil
}

// Result returns the result of a decoded wire message and server description.
func (f *Find) Result() (Cursor, error) { return nil, nil }

// Err returns the error set on this command.
func (f *Find) Err() error { return nil }

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter.
func (f *Find) RoundTrip(context.Context, description.Server, wiremessage.ReadWriter) (Cursor, error) {
	return nil, nil
}
