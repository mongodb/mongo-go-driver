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
	client         *Client
	db             *Database
	name           string
	readPreference *readpref.ReadPref
	readSelector   cluster.ServerSelector
	writeSelector  cluster.ServerSelector
}

func newCollection(db *Database, name string, options ...CollectionOption) *Collection {
	coll := &Collection{client: db.client, db: db, name: name, readPreference: db.readPreference}

	for _, option := range options {
		option.setCollectionOption(coll)
	}

	latencySelector := cluster.LatencySelector(db.client.localThreshold)

	coll.readSelector = cluster.CompositeSelector([]cluster.ServerSelector{
		readpref.Selector(coll.readPreference),
		latencySelector,
	})

	coll.writeSelector = readpref.Selector(readpref.Primary())

	return coll
}

// namespace returns the namespace of the collection.
func (coll *Collection) namespace() ops.Namespace {
	return ops.NewNamespace(coll.db.name, coll.name)
}

func (coll *Collection) getWriteableServer(ctx context.Context) (*ops.SelectedServer, error) {
	return coll.client.selectServer(ctx, coll.writeSelector, readpref.Primary())
}

func (coll *Collection) getReadableServer(ctx context.Context) (*ops.SelectedServer, error) {
	return coll.client.selectServer(ctx, coll.readSelector, coll.readPreference)
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
func (coll *Collection) ReplaceOneContext(ctx context.Context, filter interface{},
	replacement interface{}, options ...options.UpdateOption) (*UpdateOneResult, error) {

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

// AggregateContext runs an aggregation framework pipeline. A user can supply a custom context to
// this method.
//
// See https://docs.mongodb.com/manual/aggregation/.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) AggregateContext(ctx context.Context, pipeline interface{},
	options ...options.AggregateOption) (Cursor, error) {

	// TODO GODRIVER-95: Check for $out and use readable server/read preference if not found
	s, err := coll.getWriteableServer(ctx)
	if err != nil {
		return nil, err
	}

	cursor, err := ops.Aggregate(ctx, s, coll.namespace(), pipeline, options...)
	if err != nil {
		return nil, err
	}

	return cursor, nil
}

// CountContext gets the number of documents matching the filter. A user can supply a custom
// context to this method.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) CountContext(ctx context.Context, filter interface{},
	options ...options.CountOption) (int64, error) {

	s, err := coll.getReadableServer(ctx)
	if err != nil {
		return 0, err
	}

	count, err := ops.Count(ctx, s, coll.namespace(), filter, options...)
	if err != nil {
		return 0, err
	}

	return int64(count), nil
}

// DistinctContext finds the distinct values for a specified field across a single collection. A user
// can supply a custom context to this method.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) DistinctContext(ctx context.Context, fieldName string, filter interface{},
	options ...options.DistinctOption) ([]interface{}, error) {

	s, err := coll.getReadableServer(ctx)
	if err != nil {
		return nil, err
	}

	values, err := ops.Distinct(ctx, s, coll.namespace(), fieldName, filter, options...)
	if err != nil {
		return nil, err
	}

	return values, nil
}

// FindContext finds the documents matching the model.
// A user can supply a custom context to this method.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) FindContext(ctx context.Context, filter interface{},
	options ...options.FindOption) (Cursor, error) {

	s, err := coll.getReadableServer(ctx)
	if err != nil {
		return nil, err
	}

	cursor, err := ops.Find(ctx, s, coll.namespace(), filter, options...)
	if err != nil {
		return nil, err
	}

	return cursor, nil
}

// FindOneContext returns up to one document that matches the model. A user can supply a custom
// context to this method.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func (coll *Collection) FindOneContext(ctx context.Context, filter interface{}, result interface{},
	options ...options.FindOption) (bool, error) {

	options = append(options, Limit(1))

	cursor, err := coll.FindContext(ctx, filter, options...)
	if err != nil {
		return false, err
	}

	found := cursor.Next(ctx, result)
	if err = cursor.Err(); err != nil {
		return false, err
	}

	return found, nil
}
