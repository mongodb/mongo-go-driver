package command

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/description"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// FindOneAndDelete represents the findOneAndDelete operation.
//
// The findOneAndDelete command deletes a single document that matches a query and returns it.
type FindOneAndDelete struct {
	NS    Namespace
	Query *bson.Document
	Opts  []options.FindOneAndDeleteOptioner
}

// Encode will encode this command into a wire message for the given server description.
func (f *FindOneAndDelete) Encode(description.Server) (wiremessage.WireMessage, error) {
	return nil, nil
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (f *FindOneAndDelete) Decode(description.Server, wiremessage.WireMessage) *FindOneAndDelete {
	return nil
}

// Result returns the result of a decoded wire message and server description.
func (f *FindOneAndDelete) Result() (Cursor, error) { return nil, nil }

// Err returns the error set on this command.
func (f *FindOneAndDelete) Err() error { return nil }

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter.
func (f *FindOneAndDelete) RoundTrip(context.Context, description.Server, wiremessage.ReadWriter) (Cursor, error) {
	return nil, nil
}
