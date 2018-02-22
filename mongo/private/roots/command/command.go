// Package command contains abstractions for operations that can performed against a MongoDB
// deployment. The types in this package are meant to be used when the user does not care
// or know the specific version of MongoDB they are sending operations to. The types in this
// package are designed to be used both with the dispatch package and on their own directly
// with a connection.Connection. This is done so that users who have specific servers or
// connections they want to send operations to do not need to first create a fake topology.
// This also means the dispatch package can remain simpler since it only needs to handle the
// general use case of getting a topology.Topology and doing server selection on that.
package command

import (
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// Command represents a generic database command.
//
// This can be used to send arbitrary commands to the database, e.g. runCommand.
type Command struct{}

// NewCommand creates a generic command for the given database.
func NewCommand(db string, command interface{}) (Command, error) { return Command{}, nil }

// Encode will encode this command into a wire message for the given server description.
func (c Command) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) {
	return nil, nil
}

// Decode will decode the wire message using the provided server description into a result.
func (c Command) Decode(topology.ServerDescription, wiremessage.WireMessage) (bson.Reader, error) {
	return nil, nil
}
