// Package command contains abstractions for operations that can performed against a MongoDB
// deployment. The types in this package are meant to be used when the user does not care
// or know the specific version of MongoDB they are sending operations to. The types in this
// package are designed to be used both with the dispatch package and on their own directly
// with a connection.Connection. This is done so that users who have specific servers or
// connections they want to send operations to do not need to first create a fake topology.
// This also means the dispatch package can remain simpler since it only needs to handle the
// general use case of getting a topology.Topology and doing server selection on that.
package command

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/result"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
	"github.com/skriptble/giant/options"
)

// Cursor instances iterate a stream of documents. Each document can then be
// decoded into a result type or returned directly as a bson.Reader.
//
// A typical usage of the Cursor interface would be:
//
//      cursor := ... // get a cursor from some operation
//      ctx := ...    // create a context for this operation
//      defer cursor.Close(ctx)
//      for cursor.Next(ctx) {
//           ...
//       }
//       err := cursor.Err()
//       if err != nil {
//           ... // handle error
//       }
//
type Cursor interface {
	Next(context.Context) bool
	Decode(interface{}) error
	DecodeBytes() (bson.Reader, error)
	Err() error
	Close(context.Context) error
}

type Optioner interface {
	Option(*bson.Document)
}

type Namespace struct {
	DB         string
	Collection string
}

func NewNamspace(db, collection string) Namespace { return Namespace{} }

type Command struct{}
type Count struct{}
type Delete struct{}
type Distinct struct{}
type Insert struct{}
type Update struct{}
type Aggregate struct{}
type Find struct{}
type FindOneAndDelete struct{}
type FindOneAndReplace struct{}
type FindOneAndUpdate struct{}
type ListCollections struct{}
type ListDatabases struct{}
type ListIndexes struct{}

func NewCommand(db string, command interface{}) (Command, error) { return Command{}, nil }
func (c Command) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) {
	return nil, nil
}
func (c Command) Decode(topology.ServerDescription, wiremessage.WireMessage) (bson.Reader, error) {
	return nil, nil
}

func NewCount(ns Namespace, query *bson.Document, opts ...optionx.CountOptioner) (Count, error) {
	return Count{}, nil
}

func (c Count) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) { return nil, nil }
func (c Count) Decode(topology.ServerDescription, wiremessage.WireMessage) (int64, error) {
	return 0, nil
}

func NewDelete(ns Namespace, query *bson.Document, opts ...optionx.DeleteOptioner) (Delete, error) {
	return Delete{}, nil
}

func (d Delete) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) { return nil, nil }
func (d Delete) Decode(topology.ServerDescription, wiremessage.WireMessage) (result.Delete, error) {
	return result.Delete{}, nil
}

func NewDistinct(ns Namespace, field string, query *bson.Document, opts ...optionx.DistinctOptioner) (Distinct, error) {
	return Distinct{}, nil
}

func (d Distinct) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) {
	return nil, nil
}
func (d Distinct) Decode(topology.ServerDescription, wiremessage.WireMessage) ([]interface{}, error) {
	return nil, nil
}

func NewInsert(ns Namespace, docs []*bson.Document, opts ...optionx.InsertOptioner) (Insert, error) {
	return Insert{}, nil
}

func (i Insert) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) { return nil, nil }
func (i Insert) Decode(topology.ServerDescription, wiremessage.WireMessage) error   { return nil }

func NewUpdate(ns Namespace, docs []*bson.Document, opts ...optionx.UpdateOptioner) (Update, error) {
	return Update{}, nil
}

func (u Update) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) { return nil, nil }
func (u Update) Decode(topology.ServerDescription, wiremessage.WireMessage) (result.Update, error) {
	return result.Update{}, nil
}

func NewAggregate(ns Namespace, pipeline *bson.Array, opts ...optionx.AggregateOptioner) (Aggregate, error) {
	return Aggregate{}, nil
}

func (a Aggregate) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) {
	return nil, nil
}
func (a Aggregate) Decode(topology.ServerDescription, wiremessage.WireMessage) (Cursor, error) {
	return nil, nil
}

func NewFind(ns Namespace, filter *bson.Document, opts ...optionx.FindOptioner) (Find, error) {
	return Find{}, nil
}

func (f Find) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) { return nil, nil }
func (f Find) Decode(topology.ServerDescription, wiremessage.WireMessage) (Cursor, error) {
	return nil, nil
}

func NewFindOneAndDelete(ns Namespace, query *bson.Document, opts ...optionx.FindOneAndDeleteOptioner) (FindOneAndDelete, error) {
	return FindOneAndDelete{}, nil
}

func (f FindOneAndDelete) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) {
	return nil, nil
}
func (f FindOneAndDelete) Decode(topology.ServerDescription, wiremessage.WireMessage) (Cursor, error) {
	return nil, nil
}

func NewFindOneAndReplace(ns Namespace, query, replacement *bson.Document, opts ...optionx.FindOneAndReplaceOptioner) (FindOneAndReplace, error) {
	return FindOneAndReplace{}, nil
}

func (f FindOneAndReplace) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) {
	return nil, nil
}
func (f FindOneAndReplace) Decode(topology.ServerDescription, wiremessage.WireMessage) (Cursor, error) {
	return nil, nil
}

func NewFindOneAndUpdate(ns Namespace, query, update *bson.Document, opts ...optionx.FindOneAndUpdateOptioner) (FindOneAndUpdate, error) {
	return FindOneAndUpdate{}, nil
}

func (f FindOneAndUpdate) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) {
	return nil, nil
}
func (f FindOneAndUpdate) Decode(topology.ServerDescription, wiremessage.WireMessage) (Cursor, error) {
	return nil, nil
}

func NewListCollections(db string, filter *bson.Document, opts ...optionx.ListCollectionsOptioner) (ListCollections, error) {
	return ListCollections{}, nil
}

func (lc ListCollections) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) {
	return nil, nil
}
func (lc ListCollections) Decode(topology.ServerDescription, wiremessage.WireMessage) (Cursor, error) {
	return nil, nil
}

func NewListDatabases(opts ...optionx.ListDatabasesOptioner) (ListDatabases, error) {
	return ListDatabases{}, nil
}
func (ld ListDatabases) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) {
	return nil, nil
}
func (ld ListDatabases) Decode(topology.ServerDescription, wiremessage.WireMessage) (Cursor, error) {
	return nil, nil
}

func NewListIndexes(ns Namespace, opts ...optionx.ListIndexesOptioner) (ListIndexes, error) {
	return ListIndexes{}, nil
}
func (li ListIndexes) Encode(topology.ServerDescription) (wiremessage.WireMessage, error) {
	return nil, nil
}
func (li ListIndexes) Decode(topology.ServerDescription, wiremessage.WireMessage) (Cursor, error) {
	return nil, nil
}
