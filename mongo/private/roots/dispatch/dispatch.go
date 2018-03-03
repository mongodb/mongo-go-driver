package dispatch

import (
	"context"

	"github.com/mongodb/mongo-go-driver/mongo/private/roots/command"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/result"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
)

// Dispatch handles the full cycle dispatch and execution of a count command against the provided
// topology.
func Count(context.Context, topology.Topology, command.Count) (int64, error) { return 0, nil }

// Dispatch handles the full cycle dispatch and execution of a distinct command against the provided
// topology.
func Distinct(context.Context, topology.Topology, command.Distinct) ([]interface{}, error) {
	return nil, nil
}

// Dispatch handles the full cycle dispatch and execution of an insert command against the provided
// topology.
func Insert(context.Context, topology.Topology, command.Insert) error { return nil }

// Dispatch handles the full cycle dispatch and execution of an update command against the provided
// topology.
func Update(context.Context, topology.Topology, command.Update) (result.Update, error) {
	return result.Update{}, nil
}

// Dispatch handles the full cycle dispatch and execution of a find command against the provided
// topology.
func Find(context.Context, topology.Topology, command.Find) (command.Cursor, error) {
	return nil, nil
}

// Dispatch handles the full cycle dispatch and execution of a FindOneAndDelete command against the provided
// topology.
func FindOneAndDelete(context.Context, topology.Topology, command.FindOneAndDelete) (command.Cursor, error) {
	return nil, nil
}

// Dispatch handles the full cycle dispatch and execution of a FindOneAndReplace command against the provided
// topology.
func FindOneAndReplace(context.Context, topology.Topology, command.FindOneAndReplace) (command.Cursor, error) {
	return nil, nil
}

// Dispatch handles the full cycle dispatch and execution of a FindOneAndUpdate command against the provided
// topology.
func FindOneAndUpdate(context.Context, topology.Topology, command.FindOneAndUpdate) (command.Cursor, error) {
	return nil, nil
}

// Dispatch handles the full cycle dispatch and execution of a listCollections command against the provided
// topology.
func ListCollections(context.Context, topology.Topology, command.ListCollections) (command.Cursor, error) {
	return nil, nil
}

// Dispatch handles the full cycle dispatch and execution of a listDatabases command against the provided
// topology.
func ListDatabases(context.Context, topology.Topology, command.ListDatabases) (command.Cursor, error) {
	return nil, nil
}

// Dispatch handles the full cycle dispatch and execution of a listIndexes command against the provided
// topology.
func ListIndexes(context.Context, topology.Topology, command.ListIndexes) (command.Cursor, error) {
	return nil, nil
}
