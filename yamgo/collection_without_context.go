package yamgo

import (
	"context"

	"github.com/10gen/mongo-go-driver/yamgo/options"
)

// InsertOne inserts a single document into the collection.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) InsertOne(document interface{},
	options ...options.InsertOption) (*InsertOneResult, error) {

	return coll.InsertOneContext(context.Background(), document, options...)
}

// DeleteOne deletes a single document from the collection.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) DeleteOne(filter interface{},
	options ...options.DeleteOption) (*DeleteOneResult, error) {

	return coll.DeleteOneContext(context.Background(), filter, options...)
}

// UpdateOne updates a single document in the collection.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) UpdateOne(filter interface{}, update interface{},
	options ...options.UpdateOption) (*UpdateOneResult, error) {

	return coll.UpdateOneContext(context.Background(), filter, update, options...)
}

// ReplaceOne replaces a single document in the collection.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) ReplaceOne(filter interface{}, replacement interface{},
	options ...options.UpdateOption) (*UpdateOneResult, error) {

	return coll.ReplaceOneContext(context.Background(), filter, replacement, options...)
}
