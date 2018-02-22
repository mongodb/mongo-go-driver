package command

import (
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// Find represents the find command.
//
// The find command finds documents within a collection that match a filter.
type Find struct{}

// NewFind creates a find command for the given namespace, with the given filter and options.
func NewFind(ns Namespace, filter *bson.Document, opts ...options.FindOptioner) (Find, error) {
	return Find{}, nil
}

// Encode will encode this command into a wire message for the given server description.
func (f Find) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) { return nil, nil }

// Decode will decode the wire message using the provided server description into a result.
func (f Find) Decode(topology.ServerDescription, wiremessage.WireMessage) (Cursor, error) {
	return nil, nil
}
