package command

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/description"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/result"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// Delete represents the delete command.
//
// The delete command executes a delete with a given set of delete documents
// and options.
type Delete struct {
	NS    Namespace
	Query *bson.Document
	Opts  []options.DeleteOptioner
}

// Encode will encode this command into a wire message for the given server description.
func (d *Delete) Encode(description.Server) (wiremessage.WireMessage, error) { return nil, nil }

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (d *Delete) Decode(description.Server, wiremessage.WireMessage) *Delete {
	return nil
}

// Result returns the result of a decoded wire message and server description.
func (d *Delete) Result() (result.Delete, error) { return result.Delete{}, nil }

// Err returns the error set on this command.
func (d *Delete) Err() error { return nil }

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter.
func (d *Delete) RoundTrip(context.Context, description.Server, wiremessage.ReadWriter) (result.Delete, error) {
	return result.Delete{}, nil
}
