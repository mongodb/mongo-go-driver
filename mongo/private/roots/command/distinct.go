package command

import (
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// Distinct represents the disctinct command.
//
// The distinct command returns the distinct values for a specified field
// across a single collection.
type Distinct struct{}

// NewDistinct creates a distinct command for the given namespace, with the given field, query, and options.
func NewDistinct(ns Namespace, field string, query *bson.Document, opts ...options.DistinctOptioner) (Distinct, error) {
	return Distinct{}, nil
}

// Encode will encode this command into a wire message for the given server description.
func (d Distinct) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) {
	return nil, nil
}

// Decode will decode the wire message using the provided server description into a result.
func (d Distinct) Decode(topology.ServerDescription, wiremessage.WireMessage) ([]interface{}, error) {
	return nil, nil
}
