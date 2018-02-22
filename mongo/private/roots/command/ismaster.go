package command

import (
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/result"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// IsMaster represents the ismaster command.
//
// The ismaster command is used for setting up a connection to MongoDB and
// for monitoring a MongoDB server.
type IsMaster struct{}

// NewIsMaster creates an ismaster command with the given client document.
func NewIsMaster(client *bson.Document) (IsMaster, error) { return IsMaster{}, nil }

// Encode will encode this command into a wire message for the given server description.
func (im IsMaster) Encode() (wiremessage.WireMessage, error) { return nil, nil }

// Decode will decode the wire message using the provided server description into a result.
func (im IsMaster) Decode(wiremessage.WireMessage) (result.IsMaster, error) {
	return result.IsMaster{}, nil
}
