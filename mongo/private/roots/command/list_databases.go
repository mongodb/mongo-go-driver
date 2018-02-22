package command

import (
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// ListDatabases represents the listDatabases command.
//
// The listDatabases command lists the databases in a MongoDB deployment.
type ListDatabases struct{}

// NewListDatabases creates a listDatabases commands with the given options.
func NewListDatabases(opts ...options.ListDatabasesOptioner) (ListDatabases, error) {
	return ListDatabases{}, nil
}

// Encode will encode this command into a wire message for the given server description.
func (ld ListDatabases) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) {
	return nil, nil
}

// Decode will decode the wire message using the provided server description into a result.
func (ld ListDatabases) Decode(topology.ServerDescription, wiremessage.WireMessage) (Cursor, error) {
	return nil, nil
}
