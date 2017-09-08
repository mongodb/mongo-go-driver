package ops

import (
	"context"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/yamgo/internal"
	"github.com/10gen/mongo-go-driver/yamgo/options"
)

// Insert executes an insert command for the given set of  documents.
func Insert(ctx context.Context, s *SelectedServer, ns Namespace, docs []interface{},
	result interface{}, options ...options.InsertOption) error {

	if err := ns.validate(); err != nil {
		return err
	}

	command := bson.D{
		{Name: "insert", Value: ns.Collection},
		{Name: "documents", Value: docs},
	}

	for _, option := range options {
		command.AppendElem(option.InsertName(), option.InsertValue())
	}

	// TODO GODRIVER-27: write concern

	err := runMustUsePrimary(ctx, s, ns.DB, command, result)
	if err != nil {
		return internal.WrapError(err, "failed to execute insert")
	}

	return nil
}
