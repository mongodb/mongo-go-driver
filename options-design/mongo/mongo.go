package mongo

import (
	"context"

	"github.com/mongodb/mongo-go-driver/options-design/mongo/aggregateopt"
	"github.com/mongodb/mongo-go-driver/options-design/mongo/clientopt"
	"github.com/mongodb/mongo-go-driver/options-design/mongo/countopt"
	"github.com/mongodb/mongo-go-driver/options-design/mongo/dbopt"
	"github.com/mongodb/mongo-go-driver/options-design/mongo/deleteopt"
	"github.com/mongodb/mongo-go-driver/options-design/mongo/distinctopt"
	"github.com/mongodb/mongo-go-driver/options-design/mongo/findopt"
	"github.com/mongodb/mongo-go-driver/options-design/mongo/insertopt"
	"github.com/mongodb/mongo-go-driver/options-design/mongo/replaceopt"
	"github.com/mongodb/mongo-go-driver/options-design/option"
)

// Client is the main type used to access MongoDB. A client is created from a
// connection string and can be used to retrieve a Database. To perform
// individual queries a collection object is retrieved from a Database. For
// commands that are not supported directly by the porcelain API the RunCommand
// method on the Database type can be used to execute arbitrary commands
// against a database.
type Client struct{}

// NewClient creates a client from the given uri string.
func NewClient(opts ...clientopt.Option) (*Client, error) { return nil, nil }

// Database returns a handle to a MongoDB database.
func (c *Client) Database(name string) *Database { return nil }

// Database is a handle to a specific database within MongoDB. It can be used to retrieve
// handles to collections within the database and to perform arbitrary commands
// on the database via the RunCommand method.
type Database struct{}

// Name returns the name of this database.
func (db *Database) Name() string { return "" }

// Collection creates a handle to the given collection within this database.
func (db *Database) Collection(name string) *Collection { return nil }

// RunCommand allows the execution of arbitrary commands against the database.
// RunCommand always returns a non-nil value. Errors are deferred until
// RunCommandResult's Decode method is called. This allows easy chaining of the
// methods.
func (db *Database) RunCommand(ctx context.Context, cmd interface{}, opts ...dbopt.RunCommand) *DocumentResult {
	return nil
}

// Collection is a handle to a specific collection within a MongoDB database.
type Collection struct{}

// InsertOne inserts the provided document. If the provided document does not
// have an "_id" field, one will be generated.
func (c *Collection) InsertOne(ctx context.Context, doc interface{}, opts ...insertopt.One) (*InsertOneResult, error) {
	return nil, nil
}

// InsertMany inserts the provided documents. If any of the documents are
// missing an "_id" field, one will be generated.
func (c *Collection) InsertMany(ctx context.Context, docs []interface{}, opts ...insertopt.Many) (*InsertManyResult, error) {
	return nil, nil
}

// DeleteOne deletes a single document.
func (c *Collection) DeleteOne(ctx context.Context, filter interface{}, opts ...deleteopt.Delete) (*DeleteResult, error) {
	return nil, nil
}

// DeleteMany deletes multiple documents.
func (c *Collection) DeleteMany(ctx context.Context, filter interface{}, opts ...deleteopt.Delete) (*DeleteResult, error) {
	return nil, nil
}

// UpdateOne updates a single document.
func (c *Collection) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...option.UpdateOptioner) (*UpdateResult, error) {
	return nil, nil
}

// UpdateMany updates multiple documents.
func (c *Collection) UpdateMany(ctx context.Context, filter interface{}, update interface{}, opts ...option.UpdateOptioner) (*UpdateResult, error) {
	return nil, nil
}

// ReplaceOne replaces a single document.
func (c *Collection) ReplaceOne(ctx context.Context, filter interface{}, replacement interface{}, opts ...replaceopt.Replace) (*UpdateResult, error) {
	return nil, nil
}

// Aggregate runs an aggregation framework pipeline.
func (c *Collection) Aggregate(ctx context.Context, pipeline interface{}, opts ...aggregateopt.Aggregate) (Cursor, error) {
	return nil, nil
}

// Count returns the number of documents matching the filter.
func (c *Collection) Count(ctx context.Context, filter interface{}, opts ...countopt.Count) (int64, error) {
	return 0, nil
}

// Distinct finds the distinct values for a specified field across this
// collection.
func (c *Collection) Distinct(ctx context.Context, fieldName string, filter interface{}, opts ...distinctopt.Distinct) (Cursor, error) {
	return nil, nil
}

// Finds the documents, if any, matching the model.
func (c *Collection) Find(ctx context.Context, filter interface{}, opts ...findopt.Find) (Cursor, error) {
	return nil, nil
}

// FindOne finds a single document that matches the given filter.
func (c *Collection) FindOne(ctx context.Context, filter interface{}, opts ...findopt.One) *DocumentResult {
	return nil
}

// FindOneAndDelete finds a single document and deletes it, returning the
// original.
func (c *Collection) FindOneAndDelete(ctx context.Context, filter interface{}, opts ...findopt.DeleteOne) *DocumentResult {
	return nil
}

// FindOneAndReplace finds a single document and replaces it, returning either
//the original or the replaced document.
func (c *Collection) FindOneAndReplace(ctx context.Context, filter interface{}, replacement interface{}, opts ...findopt.ReplaceOne) *DocumentResult {
	return nil
}

// FindOneAndUpdate finds a single document and updates it, returning either
// the original or the updated document.
func (c *Collection) FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...findopt.UpdateOne) *DocumentResult {
	return nil
}
