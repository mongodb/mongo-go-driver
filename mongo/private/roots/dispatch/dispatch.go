package dispatch

import (
	"context"
	"errors"

	"github.com/mongodb/mongo-go-driver/mongo/private/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/command"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
	"github.com/mongodb/mongo-go-driver/mongo/readconcern"
	"github.com/mongodb/mongo-go-driver/mongo/writeconcern"
)

// ErrUnacknowledgedWrite is returned from functions that have an unacknowledged
// write concern.
var ErrUnacknowledgedWrite = errors.New("unacknowledged write")

func writeConcernOption(wc *writeconcern.WriteConcern) (options.OptWriteConcern, error) {
	elem, err := wc.MarshalBSONElement()
	if err != nil {
		return options.OptWriteConcern{}, err
	}
	return options.OptWriteConcern{WriteConcern: elem, Acknowledged: wc.Acknowledged()}, nil
}

func readConcernOption(rc *readconcern.ReadConcern) (options.OptReadConcern, error) {
	elem, err := rc.MarshalBSONElement()
	if err != nil {
		return options.OptReadConcern{}, err
	}
	return options.OptReadConcern{ReadConcern: elem}, nil
}

// ListCollections handles the full cycle dispatch and execution of a listCollections command against the provided
// topology.
func ListCollections(context.Context, *topology.Topology, command.ListCollections) (command.Cursor, error) {
	return nil, nil
}

// ListDatabases handles the full cycle dispatch and execution of a listDatabases command against the provided
// topology.
func ListDatabases(context.Context, *topology.Topology, command.ListDatabases) (command.Cursor, error) {
	return nil, nil
}

// ListIndexes handles the full cycle dispatch and execution of a listIndexes command against the provided
// topology.
func ListIndexes(context.Context, *topology.Topology, command.ListIndexes) (command.Cursor, error) {
	return nil, nil
}
