package yamgo

import (
	"context"
	"errors"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/yamgo/options"
	"github.com/10gen/mongo-go-driver/yamgo/private/cluster"
	"github.com/10gen/mongo-go-driver/yamgo/private/ops"
	"github.com/10gen/mongo-go-driver/yamgo/readpref"
)

// Collection performs operations on a given collection.
type Collection struct {
	db   *Database
	name string
}

// namespace returns the namespace of the collection.
func (coll *Collection) namespace() ops.Namespace {
	return ops.NewNamespace(coll.db.name, coll.name)
}

func (coll *Collection) getWriteableServer(ctx context.Context) (*ops.SelectedServer, error) {
	return coll.db.selectServer(ctx, cluster.WriteSelector(), readpref.Primary())
}

// InsertOneContext inserts a single document into the collection. A user can supply a custom
// context to this method.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) InsertOneContext(ctx context.Context, document interface{},
	options ...options.InsertOption) (*InsertOneResult, error) {

	doc, insertedID, err := getOrInsertID(document)
	if err != nil {
		return nil, err
	}

	// TODO GODRIVER-27: write concern

	s, err := coll.getWriteableServer(ctx)
	if err != nil {
		return nil, err
	}

	var d bson.D
	err = ops.Insert(ctx, s, coll.namespace(), []interface{}{doc}, &d, options...)
	if err != nil {
		return nil, err
	}

	return &InsertOneResult{InsertedID: insertedID}, nil
}

// DeleteOneContext deletes a single document from the collection. A user can supply a custom
// context to this method.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) DeleteOneContext(ctx context.Context, filter interface{},
	options ...options.DeleteOption) (*DeleteOneResult, error) {

	deleteDocs := []bson.D{{
		{Name: "q", Value: filter},
		{Name: "limit", Value: 1},
	}}

	// TODO GODRIVER-27: write concern

	s, err := coll.getWriteableServer(ctx)
	if err != nil {
		return nil, err
	}

	var result DeleteOneResult
	err = ops.Delete(ctx, s, coll.namespace(), deleteDocs, &result, options...)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) updateOrReplaceOne(ctx context.Context, filter interface{},
	update interface{}, options ...options.UpdateOption) (*UpdateOneResult, error) {

	updateDocs := []bson.D{{
		{Name: "q", Value: filter},
		{Name: "u", Value: update},
		{Name: "multi", Value: false},
	}}

	// TODO GODRIVER-27: write concern

	s, err := coll.getWriteableServer(ctx)
	if err != nil {
		return nil, err
	}

	var result UpdateOneResult
	err = ops.Update(ctx, s, coll.namespace(), updateDocs, &result, options...)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateOneContext updates a single document in the collection. A user can supply a custom
// context to this method.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) UpdateOneContext(ctx context.Context, filter interface{}, update interface{},
	options ...options.UpdateOption) (*UpdateOneResult, error) {

	bytes, err := bson.Marshal(update)
	if err != nil {
		return nil, err
	}

	// XXX: Roundtrip is inefficient.
	var doc bson.D
	err = bson.Unmarshal(bytes, &doc)
	if err != nil {
		return nil, err
	}

	if len(doc) > 0 && doc[0].Name[0] != '$' {
		return nil, errors.New("update document must contain key beginning with '$")
	}

	return coll.updateOrReplaceOne(ctx, filter, update, options...)
}

// ReplaceOneContext replaces a single document in the collection. A user can supply a custom
// context to this method.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) ReplaceOneContext(ctx context.Context, filter interface{}, replacement interface{},
	options ...options.UpdateOption) (*UpdateOneResult, error) {

	bytes, err := bson.Marshal(replacement)
	if err != nil {
		return nil, err
	}

	// XXX: Roundtrip is inefficient.
	var doc bson.D
	err = bson.Unmarshal(bytes, &doc)
	if err != nil {
		return nil, err
	}

	if len(doc) > 0 && doc[0].Name[0] == '$' {
		return nil, errors.New("replacement document cannot contains keys beginning with '$")
	}

	return coll.updateOrReplaceOne(ctx, filter, replacement, options...)
}
