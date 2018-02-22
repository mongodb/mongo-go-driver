package command

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/connection"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/result"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// IsMaster represents the isMaster command.
//
// The isMaster command is used for setting up a connection to MongoDB and
// for monitoring a MongoDB server.
//
// Since IsMaster can only be run on a connection, there is no Dispatch method.
type IsMaster struct {
	Client *bson.Document
}

// Encode will encode this command into a wire message for the given server description.
func (im *IsMaster) Encode() (wiremessage.WireMessage, error) { return nil, nil }

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (im *IsMaster) Decode(wiremessage.WireMessage) *IsMaster {
	return nil
}

// Result returns the result of a decoded wire message and server description.
func (im *IsMaster) Result() (result.IsMaster, error) { return result.IsMaster{}, nil }

// Err returns the error set on this command.
func (im *IsMaster) Err() error { return nil }

// RoundTrip handles the execution of this command using the provided connection.
func (im *IsMaster) RoundTrip(context.Context, connection.Connection) (result.IsMaster, error) {
	return result.IsMaster{}, nil
}
