package dispatch

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/command"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/result"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
)

func Command(context.Context, topology.Topology, command.Command) (bson.Reader, error) {
	return nil, nil
}

func Count(context.Context, topology.Topology, command.Count) (int64, error) { return 0, nil }

func Delete(context.Context, topology.Topology, command.Delete) (result.Delete, error) {
	return result.Delete{}, nil
}

func Distinct(context.Context, topology.Topology, command.Distinct) ([]interface{}, error) {
	return nil, nil
}

func Insert(context.Context, topology.Topology, command.Insert) error { return nil }

func Update(context.Context, topology.Topology, command.Update) (result.Update, error) {
	return result.Update{}, nil
}

func Aggregate(context.Context, topology.Topology, command.Aggregate) (command.Cursor, error) {
	return nil, nil
}

func Find(context.Context, topology.Topology, command.Find) (command.Cursor, error) {
	return nil, nil
}

func FindOneAndDelete(context.Context, topology.Topology, command.FindOneAndDelete) (command.Cursor, error) {
	return nil, nil
}

func FindOneAndReplace(context.Context, topology.Topology, command.FindOneAndReplace) (command.Cursor, error) {
	return nil, nil
}

func FindOneAndUpdate(context.Context, topology.Topology, command.FindOneAndUpdate) (command.Cursor, error) {
	return nil, nil
}

func ListCollections(context.Context, topology.Topology, command.ListCollections) (command.Cursor, error) {
	return nil, nil
}

func ListDatabases(context.Context, topology.Topology, command.ListDatabases) (command.Cursor, error) {
	return nil, nil
}

func ListIndexes(context.Context, topology.Topology, command.ListIndexes) (command.Cursor, error) {
	return nil, nil
}
