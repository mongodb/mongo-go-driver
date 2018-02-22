package command

import (
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/result"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// Delete represents the delete command.
//
// The delete command executes a delete with a given set of delete documents
// and options.
type Delete struct{}

// NewDelete creates a delete command for the given namespace, with the given query and options.
func NewDelete(ns Namespace, query *bson.Document, opts ...options.DeleteOptioner) (Delete, error) {
	return Delete{}, nil
}

// Encode will encode this command into a wire message for the given server description.
func (d Delete) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) { return nil, nil }

// Decode will decode the wire message using the provided server description into a result.
func (d Delete) Decode(topology.ServerDescription, wiremessage.WireMessage) (result.Delete, error) {
	return result.Delete{}, nil
}
