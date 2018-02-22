package command

import (
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// ListCollections represents the listCollections command.
//
// The listCollections command lists the collections in a database.
type ListCollections struct{}

// NewListCollections creates a listCollections command for the given database, with the given filter
// and options.
func NewListCollections(db string, filter *bson.Document, opts ...options.ListCollectionsOptioner) (ListCollections, error) {
	return ListCollections{}, nil
}

// Encode will encode this command into a wire message for the given server description.
func (lc ListCollections) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) {
	return nil, nil
}

// Decode will decode the wire message using the provided server description into a result.
func (lc ListCollections) Decode(topology.ServerDescription, wiremessage.WireMessage) (Cursor, error) {
	return nil, nil
}
