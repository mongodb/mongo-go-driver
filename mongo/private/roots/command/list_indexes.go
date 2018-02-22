package command

import (
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// ListIndexes represents the listIndexes command.
//
// The listIndexes command lists the indexes for a namespace.
type ListIndexes struct{}

// NewListIndexes creates a listIndexes command for the given namespace with the given options.
func NewListIndexes(ns Namespace, opts ...options.ListIndexesOptioner) (ListIndexes, error) {
	return ListIndexes{}, nil
}

// Encode will encode this command into a wire message for the given server description.
func (li ListIndexes) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) {
	return nil, nil
}

// Decode will decode the wire message using the provided server description into a result.
func (li ListIndexes) Decode(topology.ServerDescription, wiremessage.WireMessage) (Cursor, error) {
	return nil, nil
}
