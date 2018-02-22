package command

import (
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// Aggregate represents the aggregate command.
//
// The aggregate command performs an aggregation.
type Aggregate struct{}

// NewAggregate creates an aggregate command for the given namespace, with the given pipeline and options.
func NewAggregate(ns Namespace, pipeline *bson.Array, opts ...options.AggregateOptioner) (Aggregate, error) {
	return Aggregate{}, nil
}

// Encode will encode this command into a wire message for the given server description.
func (a Aggregate) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) {
	return nil, nil
}

// Decode will decode the wire message using the provided server description into a result.
func (a Aggregate) Decode(topology.ServerDescription, wiremessage.WireMessage) (Cursor, error) {
	return nil, nil
}
