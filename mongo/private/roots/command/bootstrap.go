package command

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/description"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// Boostrap represents a command used for bootstrapping such as isMaster,
// although this can be used by other commands as well. This command will
// always encode to an OP_QUERY and decode an OP_REPLY. Bootstrapping
// commands are also always run with the SlaveOK bit set.
type Bootstrap struct {
	DB string
	// Command to run. Only bson.Reader and *bson.Document are supported.
	Command interface{}
}

// Encode will encode this command into a wire message for the given server description.
func (b *Bootstrap) Encode(description.Server) (wiremessage.WireMessage, error) {
	return nil, nil
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (b *Bootstrap) Decode(description.Server, wiremessage.WireMessage) *Bootstrap {
	return nil
}

// Result returns the result of a decoded wire message and server description.
func (b *Bootstrap) Result() (bson.Reader, error) { return nil, nil }

// Err returns the error set on this command.
func (b *Bootstrap) Err() error { return nil }

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter.
func (b *Bootstrap) RoundTrip(context.Context, description.Server, wiremessage.ReadWriter) (bson.Reader, error) {
	return nil, nil
}
