package command

import (
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// Count represents the count command.
//
// The count command counts how many documents in a collection match the given query.
type Count struct{}

// NewCount creates a count command for the given namespace, with the given query and options.
func NewCount(ns Namespace, query *bson.Document, opts ...options.CountOptioner) (Count, error) {
	return Count{}, nil
}

// Encode will encode this command into a wire message for the given server description.
func (c Count) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) { return nil, nil }

// Decode will decode the wire message using the provided server description into a result.
func (c Count) Decode(topology.ServerDescription, wiremessage.WireMessage) (int64, error) {
	return 0, nil
}
