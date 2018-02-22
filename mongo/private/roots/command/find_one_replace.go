package command

import (
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// FindOneAndReplace represents the findOneAndReplace command.
//
// The findOneAndReplace command modifies and returns a single document.
type FindOneAndReplace struct{}

// NewFindOneAndReplace creates a findOneAndReplace command for the given namespace,
// with the given query, replacement, and options.
func NewFindOneAndReplace(ns Namespace, query, replacement *bson.Document, opts ...options.FindOneAndReplaceOptioner) (FindOneAndReplace, error) {
	return FindOneAndReplace{}, nil
}

// Encode will encode this command into a wire message for the given server description.
func (f FindOneAndReplace) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) {
	return nil, nil
}

// Decode will decode the wire message using the provided server description into a result.
func (f FindOneAndReplace) Decode(topology.ServerDescription, wiremessage.WireMessage) (Cursor, error) {
	return nil, nil
}
