package command

import (
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// FindOneAndDelete represents the findOneAndDelete command.
//
// The findOneAndDelete command deletes a single document that matches a query and returns it.
type FindOneAndDelete struct{}

// NewFindOneAndDelete creates a findOneAndDelete command for the given namespace, with the given
// query and options.
func NewFindOneAndDelete(ns Namespace, query *bson.Document, opts ...options.FindOneAndDeleteOptioner) (FindOneAndDelete, error) {
	return FindOneAndDelete{}, nil
}

// Encode will encode this command into a wire message for the given server description.
func (f FindOneAndDelete) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) {
	return nil, nil
}

// Decode will decode the wire message using the provided server description into a result.
func (f FindOneAndDelete) Decode(topology.ServerDescription, wiremessage.WireMessage) (Cursor, error) {
	return nil, nil
}
