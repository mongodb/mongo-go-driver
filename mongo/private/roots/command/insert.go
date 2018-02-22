package command

import (
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// Insert represents the insert command.
//
// The insert command inserts a set of documents into the database.
type Insert struct{}

// NewInsert creates an insert command for the given namespace, with the given documents and options.
func NewInsert(ns Namespace, docs []*bson.Document, opts ...options.InsertOptioner) (Insert, error) {
	return Insert{}, nil
}

// Encode will encode this command into a wire message for the given server description.
func (i Insert) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) { return nil, nil }

// Decode will decode the wire message using the provided server description into a result.
func (i Insert) Decode(topology.ServerDescription, wiremessage.WireMessage) error { return nil }
