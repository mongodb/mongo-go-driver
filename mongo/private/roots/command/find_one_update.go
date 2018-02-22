package command

import (
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// FindOneAndUpdate represents the findOneAndUpdate command.
//
// The findOneAndUpdate command modifies and returns a single document.
type FindOneAndUpdate struct{}

// NewFindOneAndUpdate creates a findOneAndUpdate command for the given namespace,
// with the given query, update, and options.
func NewFindOneAndUpdate(ns Namespace, query, update *bson.Document, opts ...options.FindOneAndUpdateOptioner) (FindOneAndUpdate, error) {
	return FindOneAndUpdate{}, nil
}

// Encode will encode this command into a wire message for the given server description.
func (f FindOneAndUpdate) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) {
	return nil, nil
}

// Decode will decode the wire message using the provided server description into a result.
func (f FindOneAndUpdate) Decode(topology.ServerDescription, wiremessage.WireMessage) (Cursor, error) {
	return nil, nil
}
