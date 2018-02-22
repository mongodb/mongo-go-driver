package command

import (
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/result"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// Update represents the update command.
//
// The update command updates a set of documents with the database.
type Update struct{}

// NewUpdate creates an update command for the given namespace, with the given documents and options.
func NewUpdate(ns Namespace, docs []*bson.Document, opts ...options.UpdateOptioner) (Update, error) {
	return Update{}, nil
}

// Encode will encode this command into a wire message for the given server description.
func (u Update) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) { return nil, nil }

// Decode will decode the wire message using the provided server description into a result.
func (u Update) Decode(topology.ServerDescription, wiremessage.WireMessage) (result.Update, error) {
	return result.Update{}, nil
}
